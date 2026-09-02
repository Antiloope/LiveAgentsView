// Package pilotrunner implements "lav pilot-runner", the detached shim that
// gives a piloted session's child process (Claude Code or Cursor's CLI)
// restart continuity. The daemon spawns one of these per live process,
// fully detached (its own session via setsid, no pipes back to the daemon),
// then talks to it only over a Unix domain socket at a path derived from the
// session id — see internal/pilotwire. The shim itself is intentionally
// dumb: it owns the real child process, durably logs its stdout to disk,
// relays stdin/kill from whichever daemon is currently attached (there may
// be none, if the daemon is down or between restarts), and relays each
// permission decision between the daemon and this session's own pilot-mcp
// helper (internal/pilotmcp, Claude Code's --permission-prompt-tool target)
// — all protocol parsing (turning provider stdout lines and permission asks
// into transcript events and session state) stays in internal/pilot, the one
// place that already knew how.
package pilotrunner

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotwire"
)

// Run is cmd/lav's entry point for `lav pilot-runner <flags>`.
func Run(args []string) error {
	fs := flag.NewFlagSet("pilot-runner", flag.ExitOnError)
	session := fs.String("session", "", "session id (also the socket/transcript file name)")
	provider := fs.String("provider", "", "claude-code or cursor")
	cwd := fs.String("cwd", "", "working directory for the child process")
	lavHome := fs.String("lav-home", "", "LiveAgentsView data directory")
	resume := fs.Bool("resume", false, "re-attach an existing provider session instead of starting a fresh one")
	prompt := fs.String("prompt", "", "initial message (cursor: passed as a CLI argument; claude: sent over stdin after attach)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *session == "" || *provider == "" || *cwd == "" || *lavHome == "" {
		return fmt.Errorf("pilot-runner: --session, --provider, --cwd and --lav-home are required")
	}

	r := &runner{
		id:       *session,
		provider: *provider,
		cwd:      *cwd,
		lavHome:  *lavHome,
		pending:  make(map[string]chan bool),
	}
	return r.run(*resume, *prompt)
}

type runner struct {
	id       string
	provider string
	cwd      string
	lavHome  string

	mu   sync.Mutex
	seq  int64
	log  []string // in-memory replay buffer: log[i] is seq i+1
	conn net.Conn // the currently attached daemon, or nil

	stdinMu sync.Mutex
	stdin   io.WriteCloser // nil for a provider that never receives stdin (cursor)

	child *exec.Cmd // set once Start succeeds, read by the "kill" op handler

	permMu  sync.Mutex
	pending map[string]chan bool // request_id -> channel awaiting the daemon's decision, for a permission_request currently in flight
}

func (r *runner) run(resume bool, prompt string) error {
	if err := pilotwire.EnsureDir(r.lavHome); err != nil {
		return fmt.Errorf("create pilot dir: %w", err)
	}
	sockPath := pilotwire.SocketPath(r.lavHome, r.id)
	if conn, err := net.DialTimeout("unix", sockPath, 200*time.Millisecond); err == nil {
		// A live runner is still listening here — its child process is
		// still alive and owns the real stdio. Removing and rebinding the
		// socket out from under it would silently orphan that process
		// (permanently, since nothing else holds a reference to it). Refuse
		// instead; the daemon dialing this same path reconnects to the
		// runner that's actually there.
		conn.Close()
		return fmt.Errorf("pilot-runner: a live runner already holds the control socket for session %s, refusing to start a second one", r.id)
	}
	os.Remove(sockPath) // clear a stale socket left by a runner that crashed without cleaning up

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", sockPath, err)
	}

	cmd, stdout, err := r.buildCommand(resume, prompt)
	if err != nil {
		ln.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		ln.Close()
		return fmt.Errorf("start %s: %w", r.provider, err)
	}
	r.child = cmd

	transcript, err := os.OpenFile(pilotwire.TranscriptPath(r.lavHome, r.id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		ln.Close()
		return fmt.Errorf("open transcript file: %w", err)
	}
	defer transcript.Close()

	go r.acceptLoop(ln)

	sc := pilotwire.NewScanner(stdout)
	for sc.Scan() {
		r.appendLine(sc.Text(), transcript)
	}

	waitErr := cmd.Wait()
	code := 0
	errText := ""
	if waitErr != nil {
		errText = waitErr.Error()
		code = -1
		if ee, ok := waitErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}

	r.mu.Lock()
	if r.conn != nil {
		pilotwire.Encode(r.conn, pilotwire.ServerMsg{Exited: true, Code: code, Err: errText})
		r.conn.Close()
		r.conn = nil
	}
	r.mu.Unlock()

	ln.Close()
	os.Remove(sockPath)
	return nil
}

// buildCommand mirrors internal/pilot's own claude/cursor argv exactly —
// the runner is the one spawning the real process now, but the shape of
// that invocation is unchanged.
func (r *runner) buildCommand(resume bool, prompt string) (*exec.Cmd, io.Reader, error) {
	var cmd *exec.Cmd
	switch r.provider {
	case "claude-code":
		exe, err := os.Executable()
		if err != nil {
			return nil, nil, fmt.Errorf("resolve own binary path: %w", err)
		}
		// Live-confirmed against the real CLI: headless stream-json print
		// mode never asks for tool permission over its main channel, no
		// matter the --permission-mode value — the only way it asks at all
		// is via an MCP tool named by --permission-prompt-tool. "lav" here
		// registers this same binary, re-invoked as "lav pilot-mcp", as that
		// tool (see internal/pilotmcp) — it relays each call to whichever
		// daemon is attached to this session's own control socket.
		mcpConfig, err := json.Marshal(map[string]any{
			"mcpServers": map[string]any{
				"lav": map[string]any{
					"command": exe,
					"args":    []string{"pilot-mcp", "--sock", pilotwire.SocketPath(r.lavHome, r.id)},
				},
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("build mcp config: %w", err)
		}
		args := []string{
			"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
			"--permission-mode", "default",
			"--mcp-config", string(mcpConfig),
			"--permission-prompt-tool", "mcp__lav__approval_prompt",
		}
		if resume {
			args = append(args, "--resume", r.id)
		} else {
			args = append(args, "--session-id", r.id)
		}
		cmd = exec.Command("claude", args...)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, fmt.Errorf("claude stdin pipe: %w", err)
		}
		r.stdin = stdin
	case "cursor":
		args := []string{"-p", "--output-format", "stream-json", "--force", "--trust", "--workspace", r.cwd, "--resume", r.id}
		if prompt != "" {
			args = append(args, prompt)
		}
		cmd = exec.Command("agent", args...)
	default:
		return nil, nil, fmt.Errorf("unsupported provider: %s", r.provider)
	}
	cmd.Dir = r.cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("%s stdout pipe: %w", r.provider, err)
	}
	if errFile, err := os.OpenFile(pilotwire.StderrPath(r.lavHome, r.id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		cmd.Stderr = errFile
	}
	return cmd, stdout, nil
}

// appendLine durably logs one child stdout line and forwards it live to
// whichever daemon is currently attached, if any — see the package doc for
// why writes to conn happen inside the same lock that updates the replay
// buffer (it's what keeps a fresh attach's replay and newly arriving live
// lines from interleaving out of order on the wire).
func (r *runner) appendLine(line string, transcript *os.File) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	r.log = append(r.log, line)
	fmt.Fprintln(transcript, line)
	if r.conn != nil {
		if err := pilotwire.Encode(r.conn, pilotwire.ServerMsg{Seq: r.seq, Line: line}); err != nil {
			r.conn.Close()
			r.conn = nil
		}
	}
}

func (r *runner) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed — the child has exited, run() is winding down
		}
		go r.handleConn(conn)
	}
}

func (r *runner) handleConn(conn net.Conn) {
	sc := pilotwire.NewScanner(conn)
	for sc.Scan() {
		var msg pilotwire.ClientMsg
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		switch msg.Op {
		case "attach":
			r.handleAttach(conn, msg.Since)
		case "stdin":
			r.stdinMu.Lock()
			if r.stdin != nil {
				io.WriteString(r.stdin, msg.Data)
			}
			r.stdinMu.Unlock()
		case "kill":
			if r.child != nil && r.child.Process != nil {
				r.child.Process.Kill()
			}
		case "permission_request":
			r.handlePermissionRequest(conn, msg.RequestID, msg.ToolName, msg.Input)
		case "permission_response":
			r.resolvePermission(msg.RequestID, msg.Approve)
		}
	}

	r.mu.Lock()
	if r.conn == conn {
		r.conn = nil
	}
	r.mu.Unlock()
}

// handleAttach replays everything after since, then makes conn the live
// sink for future lines — both under the same lock the append path uses,
// so nothing arrives out of order or twice.
func (r *runner) handleAttach(conn net.Conn, since int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conn != nil && r.conn != conn {
		r.conn.Close()
	}
	start := since
	if start < 0 || start > int64(len(r.log)) {
		start = 0
	}
	for i := start; i < int64(len(r.log)); i++ {
		if err := pilotwire.Encode(conn, pilotwire.ServerMsg{Seq: i + 1, Line: r.log[i]}); err != nil {
			return
		}
	}
	r.conn = conn
}

// handlePermissionRequest relays one pilot-mcp permission ask to whichever
// daemon connection is (or shortly becomes) attached, then blocks this
// connection's own goroutine until that daemon answers. The decision is
// written back on conn, the same connection pilot-mcp is itself blocked
// reading — see internal/pilotmcp, which dials fresh for exactly this one
// round trip.
func (r *runner) handlePermissionRequest(conn net.Conn, requestID, toolName string, input json.RawMessage) {
	ch := make(chan bool, 1)
	r.permMu.Lock()
	r.pending[requestID] = ch
	r.permMu.Unlock()
	defer func() {
		r.permMu.Lock()
		delete(r.pending, requestID)
		r.permMu.Unlock()
	}()

	approve := false
	// A brief wait, not an immediate fail-closed, absorbs the ordinary gap
	// of a daemon mid-restart (see dialWithRetry in internal/pilot for the
	// same allowance on the daemon's own side of a fresh spawn) — but with
	// truly nobody to ask, deny rather than hang the child's turn forever or
	// silently allow a tool call nobody actually approved.
	if daemon := r.waitForDaemon(5 * time.Second); daemon != nil {
		if err := pilotwire.Encode(daemon, pilotwire.ServerMsg{
			Permission: &pilotwire.PermissionRequest{RequestID: requestID, ToolName: toolName, Input: input},
		}); err == nil {
			approve = <-ch
		}
	}
	pilotwire.Encode(conn, pilotwire.ServerMsg{RequestID: requestID, Approve: approve})
}

// resolvePermission delivers the daemon's decision for a still-pending
// permission_request to the pilot-mcp connection blocked waiting on it. A
// request_id with nothing pending (already resolved, or a stale/duplicate
// response) is silently ignored.
func (r *runner) resolvePermission(requestID string, approve bool) {
	r.permMu.Lock()
	ch := r.pending[requestID]
	r.permMu.Unlock()
	if ch != nil {
		ch <- approve
	}
}

// waitForDaemon polls for a currently-attached daemon connection, absorbing
// the brief window during a restart where nothing is attached yet.
func (r *runner) waitForDaemon(timeout time.Duration) net.Conn {
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		conn := r.conn
		r.mu.Unlock()
		if conn != nil {
			return conn
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

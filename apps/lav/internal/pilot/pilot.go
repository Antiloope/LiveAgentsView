// Package pilot launches Claude Code and Cursor as child processes on behalf
// of the dashboard, streams their transcript live, and routes free-form
// messages, permission decisions, interrupts and cancellation back to them.
//
// The actual provider process is not a direct child of this daemon: Manager
// spawns "lav pilot-runner" (internal/pilotrunner) fully detached — its own
// session, no shared pipes — which in turn owns the real `claude`/`agent`
// process, durably logs its stdout to disk, and exposes a Unix domain
// socket for control. Manager talks to a session's runner only over that
// socket (see internal/pilotwire), so a daemon restart never has to kill
// the child to clean up after itself, and reconnecting after one is just
// dialing the same socket again — see ReconcileOnStartup.
package pilot

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotwire"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
)

// Event is one transcript entry, persisted as JSON and broadcast as-is to a
// session's SSE subscribers so the frontend parses the exact same shape
// whether it came from history or live.
type Event struct {
	Kind      string          `json:"kind"`
	Text      string          `json:"text,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Approved  *bool           `json:"approved,omitempty"`
	At        time.Time       `json:"at"`
}

const (
	EventUser               = "user"
	EventAssistant          = "assistant"
	EventToolCall           = "tool_call"
	EventPermissionRequest  = "permission_request"
	EventPermissionResolved = "permission_resolved"
	EventSystem             = "system"
	EventError              = "error"
)

// LaunchSpec is what the dashboard's "new piloted session" form submits.
type LaunchSpec struct {
	Provider model.Provider
	Cwd      string
	Branch   string
	Prompt   string
}

// ErrNotFound is returned when an action targets a session Manager does not
// know about (never launched here, or the daemon restarted since).
var ErrNotFound = fmt.Errorf("piloted session not found")

// ErrNotRunning is returned when an action needs a live process and none is
// attached — the session exists but its process died or was never resumed.
var ErrNotRunning = fmt.Errorf("piloted session has no running process")

// ErrTurnInProgress is Cursor-specific: each message is its own one-shot
// process, so a second one cannot start until the current one exits or is
// interrupted.
var ErrTurnInProgress = fmt.Errorf("a turn is already in progress for this session")

// Manager owns every piloted session's control connection to its detached
// pilot-runner. It never keeps a session's process itself alive — a
// restart's in-memory state is always rebuilt via ReconcileOnStartup, which
// reconnects to whatever is still actually running.
type Manager struct {
	store      *store.Store
	classifier classifier.Classifier
	onSession  func(model.Session)
	lavHome    string
	selfExe    string

	mu       sync.Mutex
	sessions map[string]*pilotSession
}

// pilotSession is the in-memory state for one piloted session's live
// process, guarded by mu for the fields a concurrent stdin write or a
// runner-line read can race on.
type pilotSession struct {
	id       string
	provider model.Provider
	cwd      string
	branch   string
	hub      *sse.Hub

	mu            sync.Mutex
	conn          net.Conn       // control socket to this session's pilot-runner, nil if not attached
	stdin         io.WriteCloser // relays to the child's real stdin via conn; nil for cursor's bootstrap turn (see cursor.go)
	bootstrapCmd  *exec.Cmd      // set only for cursor's first, not-yet-detached turn — see launchCursorBootstrap
	running       bool
	stoppedByUser bool // set before killing so exit handling reports IDLE, not FAILED
	lastText      string
	pending       map[string]bool // request_id -> true while awaiting a permission decision
}

// NewManager builds a Manager. lavHome is where pilot-runner sockets,
// transcripts and offset files live; selfExe is this running binary's own
// path, re-exec'd as "lav pilot-runner" for every detached process.
func NewManager(st *store.Store, cls classifier.Classifier, lavHome, selfExe string, onSession func(model.Session)) *Manager {
	return &Manager{
		store:      st,
		classifier: cls,
		lavHome:    lavHome,
		selfExe:    selfExe,
		onSession:  onSession,
		sessions:   make(map[string]*pilotSession),
	}
}

func newUUID() string {
	var b [16]byte
	// crypto/rand.Read never returns a short read or a non-nil error on any
	// platform this project targets — nothing meaningful to do with an
	// error here beyond a worse-than-random ID.
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Hub returns a session's transcript SSE hub, creating it on first access so
// a subscriber connecting before Launch's goroutine has fully started still
// gets one to wait on.
func (m *Manager) Hub(id string) *sse.Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	if !ok {
		ps = &pilotSession{id: id, hub: sse.NewHub(), pending: map[string]bool{}}
		m.sessions[id] = ps
	}
	return ps.hub
}

func (m *Manager) get(id string) (*pilotSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	return ps, ok
}

func (m *Manager) getOrCreate(id string) *pilotSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	if !ok {
		ps = &pilotSession{id: id, hub: sse.NewHub(), pending: map[string]bool{}}
		m.sessions[id] = ps
	}
	return ps
}

// Events returns a session's persisted transcript, oldest first.
func (m *Manager) Events(ctx context.Context, id string) ([]json.RawMessage, error) {
	raws, err := m.store.ListEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(raws))
	for i, r := range raws {
		out[i] = json.RawMessage(r)
	}
	return out, nil
}

// emit persists one transcript event and broadcasts it to the session's live
// subscribers.
func (m *Manager) emit(ctx context.Context, ps *pilotSession, ev Event) {
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if err := m.store.AppendEvent(ctx, ps.id, ps.provider, "pilot:"+ev.Kind, "", string(b)); err != nil {
		// Persistence failing does not stop the live stream — the user
		// still sees it happen even if history replay would miss it.
		_ = err
	}
	ps.hub.Broadcast(ev)
}

// upsert persists the session's canonical state and tells the daemon to
// broadcast it on its global session hub, so piloted sessions land in the
// one dashboard and attention queue.
func (m *Manager) upsert(ctx context.Context, ps *pilotSession, state model.State, lastMessage string) {
	now := time.Now().UTC()
	createdAt := now
	if existing, found, _ := m.store.GetSession(ctx, ps.id); found {
		createdAt = existing.CreatedAt
	}
	sess := model.Session{
		ID:          ps.id,
		Provider:    ps.provider,
		Fidelity:    model.FidelityDriver,
		Cwd:         ps.cwd,
		Repo:        filepath.Base(ps.cwd),
		Branch:      ps.branch,
		State:       state,
		LastMessage: lastMessage,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	if err := m.store.UpsertSession(ctx, sess); err != nil {
		return
	}
	if m.onSession != nil {
		m.onSession(sess)
	}
}

// classify resolves an ambiguous end-of-turn signal: empty means still
// ambiguous after a non-error exit, so run the shared classifier over the
// last assistant text.
func (m *Manager) classify(lastText string) model.State {
	switch m.classifier.Classify(lastText) {
	case classifier.VerdictWaiting:
		return model.StateWaiting
	default:
		return model.StateDone
	}
}

// finalizeProcess runs once a piloted process has actually exited, for
// either provider. A user-requested stop (interrupt or cancel, both set
// stoppedByUser before killing) always reads as IDLE, distinct from an
// unrequested exit, which is a real crash.
func (m *Manager) finalizeProcess(ctx context.Context, ps *pilotSession, waitErr error) {
	ps.mu.Lock()
	stopped := ps.stoppedByUser
	ps.stoppedByUser = false
	lastText := ps.lastText
	ps.mu.Unlock()

	if stopped {
		m.upsert(ctx, ps, model.StateIdle, lastText)
		return
	}
	if waitErr == nil {
		return // a clean exit already went through the "result" line handler
	}
	m.emit(ctx, ps, Event{Kind: EventError, Text: waitErr.Error()})
	m.upsert(ctx, ps, model.StateFailed, lastText)
}

// resolve returns the in-memory session state for id, reconstructing it from
// SQLite if this Manager has no live record (e.g. a session whose process
// genuinely exited, resolved via Resume rather than reconnect).
func (m *Manager) resolve(ctx context.Context, id string) (*pilotSession, error) {
	m.mu.Lock()
	ps, ok := m.sessions[id]
	m.mu.Unlock()
	if ok {
		return ps, nil
	}

	sess, found, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found || sess.Fidelity != model.FidelityDriver {
		return nil, ErrNotFound
	}

	ps = m.getOrCreate(id)
	ps.provider = sess.Provider
	ps.cwd = sess.Cwd
	ps.branch = sess.Branch
	return ps, nil
}

// ReconcileOnStartup runs once at daemon startup. For every piloted session,
// it dials that session's pilot-runner socket: if the detached process is
// still actually running, it reconnects (replaying any transcript lines
// produced while this daemon was down, with no duplicates and no drops —
// see reconnect) and leaves its state as-is, immediately sendable/
// interruptible/cancelable again with no user action needed. This is tried
// regardless of the session's last recorded state, not just
// Working/Waiting/Blocked: Claude Code's process stays resident after a
// turn finishes, so a DONE session can very much still have a live process
// worth reconnecting to. Only once the socket is confirmed gone — the
// process genuinely exited while this daemon was down — does a session
// still showing a live-looking state (Working/Waiting/Blocked) fall back to
// the pre-restart-continuity behavior: marked IDLE, Resume offered. A
// session already reading as non-live (Done/Failed/Idle) is left as-is
// either way; there is nothing to correct.
func (m *Manager) ReconcileOnStartup(ctx context.Context) error {
	sessions, err := m.store.ListSessions(ctx)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.Fidelity != model.FidelityDriver {
			continue
		}
		if m.reconnect(ctx, sess) {
			continue
		}
		switch sess.State {
		case model.StateWorking, model.StateWaiting, model.StateBlocked:
		default:
			continue
		}
		sess.State = model.StateIdle
		sess.UpdatedAt = time.Now().UTC()
		if err := m.store.UpsertSession(ctx, sess); err != nil {
			continue
		}
		if m.onSession != nil {
			m.onSession(sess)
		}
	}
	return nil
}

func (m *Manager) currentSession(ctx context.Context, ps *pilotSession) (model.Session, error) {
	sess, found, err := m.store.GetSession(ctx, ps.id)
	if err != nil {
		return model.Session{}, err
	}
	if !found {
		return model.Session{}, ErrNotFound
	}
	return sess, nil
}

// Launch starts a brand-new piloted session for spec.Provider.
func (m *Manager) Launch(ctx context.Context, spec LaunchSpec) (model.Session, error) {
	switch spec.Provider {
	case model.ProviderClaudeCode:
		ps := &pilotSession{hub: sse.NewHub(), pending: map[string]bool{}}
		if err := m.launchClaude(ctx, ps, spec, ""); err != nil {
			return model.Session{}, err
		}
		return m.currentSession(ctx, ps)
	case model.ProviderCursor:
		ps := &pilotSession{hub: sse.NewHub(), pending: map[string]bool{}}
		if err := m.launchCursor(ctx, ps, spec, ""); err != nil {
			return model.Session{}, err
		}
		return m.currentSession(ctx, ps)
	default:
		return model.Session{}, fmt.Errorf("unsupported piloted provider: %s", spec.Provider)
	}
}

// SendMessage delivers free-form text to a piloted session: over stdin for
// Claude Code's persistent process, or as a new --resume'd one-shot
// invocation for Cursor.
func (m *Manager) SendMessage(ctx context.Context, id, text string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode:
		return m.sendClaudeMessage(ps, text)
	case model.ProviderCursor:
		return m.sendCursorMessage(ctx, ps, text)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// ApprovePermission answers a pending permission request. Claude Code only —
// Cursor piloted sessions auto-approve and never raise one.
func (m *Manager) ApprovePermission(ctx context.Context, id, requestID string, approve bool) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	if ps.provider != model.ProviderClaudeCode {
		return fmt.Errorf("permission approval is not available for %s piloted sessions", ps.provider)
	}
	return m.approveClaudePermission(ps, requestID, approve)
}

// Interrupt stops the current turn. For Claude Code the process stays
// attached, ready for the next message; for Cursor there is no separate
// "current turn" to stop without ending the one-shot process that is running
// it, so this and Cancel do the same thing.
func (m *Manager) Interrupt(ctx context.Context, id string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode:
		return m.interruptClaude(ps)
	case model.ProviderCursor:
		return m.killPilotProcess(ps)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// Cancel ends the piloted session's current process.
func (m *Manager) Cancel(ctx context.Context, id string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode, model.ProviderCursor:
		return m.killPilotProcess(ps)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// Resume re-attaches a piloted session with no live process — after its
// process really did exit (crash, or a user cancel), not after a daemon
// restart alone (ReconcileOnStartup already reconnects those). For Claude
// Code this starts a new `--resume`'d process. Cursor has no standing
// process to re-attach (each message already is its own `--resume`'d
// invocation — see sendCursorMessage), so this just confirms the session is
// known and ready.
func (m *Manager) Resume(ctx context.Context, id string) (model.Session, error) {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return model.Session{}, err
	}
	ps.mu.Lock()
	running := ps.running
	ps.mu.Unlock()
	if running {
		return model.Session{}, fmt.Errorf("piloted session %s is already running", id)
	}

	switch ps.provider {
	case model.ProviderClaudeCode:
		spec := LaunchSpec{Provider: ps.provider, Cwd: ps.cwd, Branch: ps.branch}
		if err := m.launchClaude(ctx, ps, spec, ps.id); err != nil {
			return model.Session{}, err
		}
	case model.ProviderCursor:
		// Nothing to start.
	default:
		return model.Session{}, fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
	return m.currentSession(ctx, ps)
}

// --- pilot-runner process management ---------------------------------------

// spawnRunner starts "lav pilot-runner" for ps.id fully detached from this
// daemon process — its own session (SysProcAttr.Setsid), stdio pointed at
// /dev/null rather than shared pipes — so nothing about the runner's
// lifetime depends on this process staying alive, then dials its control
// socket and attaches from the beginning (since=0: a freshly spawned runner
// has nothing to replay). Used for every Claude Code launch and resume, and
// for every Cursor message after the session's first — see cursor.go.
func (m *Manager) spawnRunner(ps *pilotSession, extraArgs []string) error {
	args := append([]string{
		"pilot-runner",
		"--session", ps.id,
		"--provider", string(ps.provider),
		"--cwd", ps.cwd,
		"--lav-home", m.lavHome,
	}, extraArgs...)

	cmd := exec.Command(m.selfExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pilot-runner: %w", err)
	}
	go cmd.Wait() // reap whenever it exits; never blocks this daemon

	conn, err := dialWithRetry(pilotwire.SocketPath(m.lavHome, ps.id), 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to pilot-runner: %w", err)
	}
	if err := pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "attach", Since: 0}); err != nil {
		conn.Close()
		return fmt.Errorf("attach to pilot-runner: %w", err)
	}

	ps.mu.Lock()
	ps.conn = conn
	ps.stdin = &socketStdin{conn: conn}
	ps.running = true
	ps.stoppedByUser = false
	ps.mu.Unlock()

	go m.readFromRunner(ps, conn)
	return nil
}

// reconnect dials an already-running session's pilot-runner socket — used
// only by ReconcileOnStartup. Returns false when the socket is gone (the
// process genuinely exited while this daemon was down), which the caller
// treats as today's pre-restart-continuity fallback.
func (m *Manager) reconnect(ctx context.Context, sess model.Session) bool {
	conn, err := net.DialTimeout("unix", pilotwire.SocketPath(m.lavHome, sess.ID), 500*time.Millisecond)
	if err != nil {
		return false
	}
	since := pilotwire.ReadOffset(m.lavHome, sess.ID)
	if err := pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "attach", Since: since}); err != nil {
		conn.Close()
		return false
	}

	ps := m.getOrCreate(sess.ID)
	ps.provider = sess.Provider
	ps.cwd = sess.Cwd
	ps.branch = sess.Branch
	ps.mu.Lock()
	ps.conn = conn
	ps.stdin = &socketStdin{conn: conn}
	ps.running = true
	ps.mu.Unlock()

	go m.readFromRunner(ps, conn)
	return true
}

// readFromRunner decodes transcript lines forwarded over a pilot-runner's
// control socket and feeds them through the same per-provider line handler
// live stdout used before this session had restart continuity — the wire
// format changed, the parsing didn't. An explicit "exited" frame means the
// child process itself is gone, handled exactly like today's process exit.
// A plain disconnect with no such frame means only the *connection* died —
// almost always this daemon process shutting down for a restart — and must
// not be mistaken for the piloted process exiting: the session's persisted
// state is left untouched, for ReconcileOnStartup to resolve for real on
// the next boot via a fresh dial.
func (m *Manager) readFromRunner(ps *pilotSession, conn net.Conn) {
	ctx := context.Background()
	sc := pilotwire.NewScanner(conn)
	for sc.Scan() {
		var msg pilotwire.ServerMsg
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Exited {
			ps.mu.Lock()
			ps.running = false
			ps.conn = nil
			ps.mu.Unlock()
			var waitErr error
			if msg.Err != "" {
				waitErr = fmt.Errorf("%s", msg.Err)
			}
			m.finalizeProcess(ctx, ps, waitErr)
			return
		}
		if msg.Line != "" {
			switch ps.provider {
			case model.ProviderClaudeCode:
				m.handleClaudeLine(ctx, ps, []byte(msg.Line))
			case model.ProviderCursor:
				m.handleCursorLine(ctx, ps, []byte(msg.Line))
			}
		}
		if msg.Seq > 0 {
			_ = pilotwire.WriteOffset(m.lavHome, ps.id, msg.Seq)
		}
	}

	ps.mu.Lock()
	ps.running = false
	ps.conn = nil
	ps.mu.Unlock()
}

// killPilotProcess ends a piloted session's process regardless of which
// path launched it: over the control socket for anything running through a
// pilot-runner (every Claude Code session, every Cursor turn after the
// first), or directly for Cursor's not-yet-detached bootstrap turn.
func (m *Manager) killPilotProcess(ps *pilotSession) error {
	ps.mu.Lock()
	conn := ps.conn
	bootstrap := ps.bootstrapCmd
	ps.stoppedByUser = true
	ps.mu.Unlock()

	if conn != nil {
		return pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "kill"})
	}
	if bootstrap != nil && bootstrap.Process != nil {
		return bootstrap.Process.Kill()
	}
	return ErrNotRunning
}

// socketStdin relays writes meant for a piloted child's real stdin over the
// control socket to the pilot-runner actually holding that pipe — the
// io.WriteCloser every send/interrupt/permission-response code path already
// wrote to before restart continuity existed.
type socketStdin struct{ conn net.Conn }

func (w *socketStdin) Write(p []byte) (int, error) {
	if err := pilotwire.Encode(w.conn, pilotwire.ClientMsg{Op: "stdin", Data: string(p)}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *socketStdin) Close() error { return w.conn.Close() }

// dialWithRetry absorbs the brief window between starting pilot-runner and
// its control socket existing — process creation and socket bind are not
// synchronous from this side.
func dialWithRetry(path string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(50 * time.Millisecond)
	}
}

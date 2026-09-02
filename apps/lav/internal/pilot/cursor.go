package pilot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// launchCursor starts one cursor-agent turn. cursor-agent assigns its own
// session id on a fresh launch — unlike Claude Code's --session-id, there is
// no flag to choose one upfront — so the very first turn (resumeChatID
// empty) has no id to name a pilot-runner's control socket after until that
// id is known, and stays a direct, not-yet-detached child of this daemon
// (launchCursorBootstrap). Every later message already knows the chat id
// upfront (resumeChatID, from ps.id) and gets full restart continuity via
// spawnRunner, same as Claude Code. This narrow gap for the bootstrap turn
// is an accepted, low-stakes one: cursor-agent's own --resume semantics
// pick a mid-turn loss back up on the very next message regardless of what
// happened to the process that was running it.
func (m *Manager) launchCursor(ctx context.Context, ps *pilotSession, spec LaunchSpec, resumeChatID string) error {
	ps.provider = model.ProviderCursor
	ps.cwd = spec.Cwd
	ps.branch = spec.Branch

	if resumeChatID == "" {
		return m.launchCursorBootstrap(ctx, ps, spec)
	}

	ps.id = resumeChatID
	if err := m.spawnRunner(ps, []string{"--prompt", spec.Prompt}); err != nil {
		return err
	}
	if spec.Prompt != "" {
		m.emit(ctx, ps, Event{Kind: EventUser, Text: spec.Prompt})
	}
	m.upsert(ctx, ps, model.StateWorking, "")
	return nil
}

// launchCursorBootstrap runs cursor-agent directly (real pipes, a normal
// child of this daemon process) just long enough to learn the session id it
// assigns itself — confirmed live against the installed CLI: --force/--yolo
// is the only way a Cursor piloted session can get work done without a live
// permission gate, and every message is its own process chained by
// --resume/--continue rather than a persistent stdin channel.
func (m *Manager) launchCursorBootstrap(ctx context.Context, ps *pilotSession, spec LaunchSpec) error {
	args := []string{"-p", "--output-format", "stream-json", "--force", "--trust", "--workspace", spec.Cwd}
	if spec.Prompt != "" {
		args = append(args, spec.Prompt)
	}
	cmd := exec.Command("agent", args...)
	cmd.Dir = spec.Cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cursor agent stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start cursor agent: %w", err)
	}

	ps.mu.Lock()
	ps.bootstrapCmd = cmd
	ps.running = true
	ps.stoppedByUser = false
	ps.mu.Unlock()

	reader := bufio.NewReader(stdout)

	firstLine, err := readLineWithTimeout(reader, 15*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("read cursor agent init line: %w", err)
	}
	var sys struct {
		SessionID string `json:"session_id"`
	}
	_ = json.Unmarshal(firstLine, &sys)
	if sys.SessionID == "" {
		cmd.Process.Kill()
		return fmt.Errorf("cursor agent did not report a session_id on startup")
	}
	ps.id = sys.SessionID
	m.mu.Lock()
	m.sessions[ps.id] = ps
	m.mu.Unlock()
	if len(firstLine) > 0 {
		m.handleCursorLine(ctx, ps, firstLine)
	}

	if spec.Prompt != "" {
		m.emit(ctx, ps, Event{Kind: EventUser, Text: spec.Prompt})
	}
	m.upsert(ctx, ps, model.StateWorking, "")

	go m.readCursorBootstrapStdout(ps, reader)
	return nil
}

// sendCursorMessage starts a new turn chained to the existing chat via
// --resume — see launchCursor's doc comment for why this can't just write
// to a running process's stdin the way Claude Code does.
func (m *Manager) sendCursorMessage(ctx context.Context, ps *pilotSession, text string) error {
	ps.mu.Lock()
	running := ps.running
	ps.mu.Unlock()
	if running {
		return ErrTurnInProgress
	}
	spec := LaunchSpec{Provider: model.ProviderCursor, Cwd: ps.cwd, Branch: ps.branch, Prompt: text}
	return m.launchCursor(ctx, ps, spec, ps.id)
}

// readLineWithTimeout reads one line without blocking Launch forever if the
// CLI never produces output (bad workspace, hung auth, ...). The read
// goroutine is left running on timeout — it unblocks once the caller kills
// the process, which is what every timeout path here does.
func readLineWithTimeout(r *bufio.Reader, timeout time.Duration) ([]byte, error) {
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadBytes('\n')
		ch <- result{line, err}
	}()
	select {
	case res := <-ch:
		return bytes.TrimSpace(res.line), res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for output")
	}
}

type cursorAssistantLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []claudeContentBlock `json:"content"`
	} `json:"message"`
}

// cursorToolName picks a human-readable label out of tool_call's
// discriminated-union shape ({"shellToolCall":{...}} or
// {"editToolCall":{...}}) without needing to model every tool's own field
// layout — the raw payload is kept in full as ToolInput for anyone who wants
// the detail.
func cursorToolName(raw json.RawMessage) string {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "tool"
	}
	for _, known := range []string{"shellToolCall", "editToolCall", "readToolCall", "searchToolCall"} {
		if _, ok := probe[known]; ok {
			return known
		}
	}
	for k := range probe {
		return k
	}
	return "tool"
}

// handleCursorLine turns one raw stream-json line from cursor-agent's stdout
// (relayed live by pilot-runner, or replayed from its durable transcript on
// reconnect, or read directly during the bootstrap turn) into transcript
// events and session state transitions.
func (m *Manager) handleCursorLine(ctx context.Context, ps *pilotSession, line []byte) {
	if len(line) == 0 {
		return
	}
	var probe struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		m.emit(ctx, ps, Event{Kind: EventError, Text: string(line)})
		return
	}

	switch probe.Type {
	case "user":
		return // our own prompt, already emitted proactively

	case "system":
		if probe.Subtype == "init" {
			m.emit(ctx, ps, Event{Kind: EventSystem, Text: "session started"})
		}

	case "assistant":
		var a cursorAssistantLine
		if err := json.Unmarshal(line, &a); err != nil {
			return
		}
		for _, block := range a.Message.Content {
			if block.Type != "text" || block.Text == "" {
				continue
			}
			ps.mu.Lock()
			ps.lastText = block.Text
			ps.mu.Unlock()
			m.emit(ctx, ps, Event{Kind: EventAssistant, Text: block.Text})
		}

	case "tool_call":
		if probe.Subtype != "started" {
			return // "completed" is skipped — the assistant's own follow-up text already narrates the outcome
		}
		var t struct {
			CallID   string          `json:"call_id"`
			ToolCall json.RawMessage `json:"tool_call"`
		}
		if err := json.Unmarshal(line, &t); err != nil {
			return
		}
		m.emit(ctx, ps, Event{Kind: EventToolCall, ToolName: cursorToolName(t.ToolCall), ToolInput: t.ToolCall, RequestID: t.CallID})

	case "thinking":
		// Streamed reasoning deltas — too noisy for a chat transcript, same
		// call as Claude Code's hook_started/hook_response/api_retry noise.

	case "result":
		var r struct {
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
		}
		_ = json.Unmarshal(line, &r)
		ps.mu.Lock()
		lastText := ps.lastText
		ps.mu.Unlock()
		if r.Result != "" {
			lastText = r.Result
		}
		if r.IsError {
			m.upsert(ctx, ps, model.StateFailed, lastText)
			return
		}
		m.upsert(ctx, ps, m.classify(lastText), lastText)

	default:
		m.emit(ctx, ps, Event{Kind: EventSystem, Text: string(line)})
	}
}

// readCursorBootstrapStdout drains the first, not-yet-detached cursor-agent
// turn's stdout directly — see launchCursorBootstrap. Every later turn is
// read via readFromRunner instead.
func (m *Manager) readCursorBootstrapStdout(ps *pilotSession, reader *bufio.Reader) {
	ctx := context.Background()
	for {
		line, err := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			m.handleCursorLine(ctx, ps, line)
		}
		if err != nil {
			break
		}
	}

	ps.mu.Lock()
	cmd := ps.bootstrapCmd
	ps.bootstrapCmd = nil
	ps.running = false
	ps.mu.Unlock()

	var waitErr error
	if cmd != nil {
		waitErr = cmd.Wait()
	}
	m.finalizeProcess(ctx, ps, waitErr)
}

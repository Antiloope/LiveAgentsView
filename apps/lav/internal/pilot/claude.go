package pilot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotwire"
)

// launchClaude starts a Claude Code piloted session by spawning its
// pilot-runner (see spawnRunner) with --input-format stream-json, which
// keeps it reading stdin for further turns instead of exiting after the
// first one — what lets SendMessage keep talking to the same process across
// a whole session, and what lets ReconcileOnStartup reconnect to it later.
// resumeSessionID is empty for a fresh launch (a new --session-id is
// generated) and set to re-attach a session whose process previously
// exited (--resume).
func (m *Manager) launchClaude(ctx context.Context, ps *pilotSession, spec LaunchSpec, resumeSessionID string) error {
	id := resumeSessionID
	if id == "" {
		id = newUUID()
	}
	ps.id = id
	ps.provider = model.ProviderClaudeCode
	ps.cwd = spec.Cwd
	ps.branch = spec.Branch
	m.mu.Lock()
	m.sessions[id] = ps
	m.mu.Unlock()

	var extra []string
	if resumeSessionID != "" {
		extra = append(extra, "--resume")
	}
	if err := m.spawnRunner(ps, extra); err != nil {
		return err
	}

	m.upsert(ctx, ps, model.StateWorking, "")

	if spec.Prompt != "" {
		m.emit(ctx, ps, Event{Kind: EventUser, Text: spec.Prompt})
		if _, err := ps.stdin.Write(claudeUserMessage(spec.Prompt)); err != nil {
			return fmt.Errorf("send initial prompt: %w", err)
		}
	}
	return nil
}

func (m *Manager) sendClaudeMessage(ps *pilotSession, text string) error {
	ps.mu.Lock()
	running, stdin := ps.running, ps.stdin
	ps.mu.Unlock()
	if !running || stdin == nil {
		return ErrNotRunning
	}
	m.emit(context.Background(), ps, Event{Kind: EventUser, Text: text})
	_, err := stdin.Write(claudeUserMessage(text))
	return err
}

// approveClaudePermission answers a pending approval_prompt call — relayed by
// pilot-runner from its pilot-mcp helper, see handleClaudePermissionRequest —
// by sending the decision back over the same control socket the runner
// already dials this session's stdin/kill through. Not written to the
// child's own stdin: the child never asked there in the first place (see
// package doc), the runner's blocked pilot-mcp connection is what's actually
// waiting for this.
func (m *Manager) approveClaudePermission(ps *pilotSession, requestID string, approve bool) error {
	ps.mu.Lock()
	running, conn := ps.running, ps.conn
	known := ps.pending[requestID]
	if known {
		delete(ps.pending, requestID)
	}
	ps.mu.Unlock()
	if !running || conn == nil {
		return ErrNotRunning
	}
	if !known {
		return fmt.Errorf("no pending permission request %s", requestID)
	}
	approved := approve
	m.emit(context.Background(), ps, Event{Kind: EventPermissionResolved, RequestID: requestID, Approved: &approved})
	return pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "permission_response", RequestID: requestID, Approve: approve})
}

// handleClaudePermissionRequest turns one permission ask relayed by
// pilot-runner (originally an approval_prompt tools/call it received from
// its pilot-mcp helper — see internal/pilotrunner and internal/pilotmcp)
// into the same EventPermissionRequest/StateBlocked the dashboard already
// renders an Approve/Deny card for. approveClaudePermission answers it.
func (m *Manager) handleClaudePermissionRequest(ctx context.Context, ps *pilotSession, req *pilotwire.PermissionRequest) {
	ps.mu.Lock()
	ps.pending[req.RequestID] = true
	ps.mu.Unlock()
	m.emit(ctx, ps, Event{Kind: EventPermissionRequest, ToolName: req.ToolName, ToolInput: req.Input, RequestID: req.RequestID})
	m.upsert(ctx, ps, model.StateBlocked, "")
}

// interruptClaude asks the live process to stop its current turn without
// exiting. Live-confirmed shape: the CLI acknowledges the control_request
// with control_response{subtype:"success"} and the process survives, but the
// turn's own "result" line comes back as is_error:true,
// subtype:"error_during_execution" — indistinguishable on its own from a
// genuine failure. interrupted marks that the next such result line was
// asked for, not a crash — see handleClaudeLine's "result" case.
func (m *Manager) interruptClaude(ps *pilotSession) error {
	ps.mu.Lock()
	running, stdin := ps.running, ps.stdin
	if running {
		ps.interrupted = true
	}
	ps.mu.Unlock()
	if !running || stdin == nil {
		return ErrNotRunning
	}
	_, err := stdin.Write(buildInterruptRequest())
	return err
}

// claudeUserMessage wraps free text in the role/content envelope Claude
// Code's stream-json input format expects for a new user turn.
func claudeUserMessage(text string) []byte {
	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []map[string]any{{"type": "text", "text": text}},
		},
	}
	b, _ := json.Marshal(msg)
	return append(b, '\n')
}

func buildInterruptRequest() []byte {
	msg := map[string]any{
		"type":       "control_request",
		"request_id": newUUID(),
		"request":    map[string]any{"subtype": "interrupt"},
	}
	b, _ := json.Marshal(msg)
	return append(b, '\n')
}

type claudeContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type claudeAssistantLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []claudeContentBlock `json:"content"`
	} `json:"message"`
}

type claudeResultLine struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

type claudeSystemLine struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
}

// handleClaudeLine turns one raw stream-json line from Claude Code's stdout
// (relayed live by pilot-runner, or replayed from its durable transcript on
// reconnect) into transcript events and session state transitions.
// Permission requests never arrive on this channel at all — see
// handleClaudePermissionRequest. Unrecognized top-level types (including the
// control_response that acknowledges an interrupt) are still shown, as a raw
// passthrough event, rather than silently dropped.
func (m *Manager) handleClaudeLine(ctx context.Context, ps *pilotSession, line []byte) {
	if len(line) == 0 {
		return
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		m.emit(ctx, ps, Event{Kind: EventError, Text: string(line)})
		return
	}

	switch probe.Type {
	case "user":
		// The CLI replays our own message back for acknowledgment — already
		// emitted proactively when we sent it, so this is a duplicate.
		return

	case "assistant":
		var a claudeAssistantLine
		if err := json.Unmarshal(line, &a); err != nil {
			return
		}
		for _, block := range a.Message.Content {
			switch block.Type {
			case "text":
				if block.Text == "" {
					continue
				}
				ps.mu.Lock()
				ps.lastText = block.Text
				ps.mu.Unlock()
				m.emit(ctx, ps, Event{Kind: EventAssistant, Text: block.Text})
			case "tool_use":
				m.emit(ctx, ps, Event{Kind: EventToolCall, ToolName: block.Name, ToolInput: block.Input, RequestID: block.ID})
			}
		}

	case "result":
		var r claudeResultLine
		if err := json.Unmarshal(line, &r); err != nil {
			return
		}
		ps.mu.Lock()
		lastText := ps.lastText
		interrupted := ps.interrupted
		ps.interrupted = false
		ps.mu.Unlock()
		if r.Result != "" {
			lastText = r.Result
		}
		if r.IsError {
			if interrupted && r.Subtype == "error_during_execution" {
				// A requested interrupt, not a crash: the process is still
				// alive and its stdin still open, so this reads as DONE
				// (ready for the next message) rather than FAILED (which the
				// dashboard reads as "process gone, offer Resume").
				m.emit(ctx, ps, Event{Kind: EventSystem, Text: "turn interrupted"})
				m.upsert(ctx, ps, model.StateDone, lastText)
				return
			}
			m.upsert(ctx, ps, model.StateFailed, lastText)
			return
		}
		m.upsert(ctx, ps, m.classify(lastText), lastText)

	case "system":
		var s claudeSystemLine
		if err := json.Unmarshal(line, &s); err == nil && s.Subtype == "init" {
			m.emit(ctx, ps, Event{Kind: EventSystem, Text: "session started"})
		}
		// hook_started/hook_response/api_retry and other system noise are
		// not shown — the chat stays readable as a conversation, not a log.

	default:
		m.emit(ctx, ps, Event{Kind: EventSystem, Text: string(line)})
	}
}

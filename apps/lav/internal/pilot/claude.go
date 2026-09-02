package pilot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// launchClaude starts Claude Code as a long-lived driver process: with
// --input-format stream-json it keeps reading stdin for further turns
// instead of exiting after the first one, which is what lets SendMessage
// keep talking to the same process across a whole piloted session.
// resumeSessionID is empty for a fresh launch (a new --session-id is
// generated) and set to re-attach to a session whose process previously
// died (--resume), per Acceptance's resume item.
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

	args := []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--permission-mode", "default"}
	if resumeSessionID != "" {
		args = append(args, "--resume", resumeSessionID)
	} else {
		args = append(args, "--session-id", id)
	}
	cmd := exec.Command("claude", args...)
	cmd.Dir = spec.Cwd

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("claude stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("claude stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	ps.mu.Lock()
	ps.cmd = cmd
	ps.stdin = stdin
	ps.running = true
	ps.mu.Unlock()

	m.upsert(ctx, ps, model.StateWorking, "")

	if spec.Prompt != "" {
		m.emit(ctx, ps, Event{Kind: EventUser, Text: spec.Prompt})
		if _, err := stdin.Write(claudeUserMessage(spec.Prompt)); err != nil {
			return fmt.Errorf("send initial prompt: %w", err)
		}
	}

	go m.readClaudeStdout(ps, stdout)
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

func (m *Manager) approveClaudePermission(ps *pilotSession, requestID string, approve bool) error {
	ps.mu.Lock()
	running, stdin := ps.running, ps.stdin
	known := ps.pending[requestID]
	if known {
		delete(ps.pending, requestID)
	}
	ps.mu.Unlock()
	if !running || stdin == nil {
		return ErrNotRunning
	}
	if !known {
		return fmt.Errorf("no pending permission request %s", requestID)
	}
	approved := approve
	m.emit(context.Background(), ps, Event{Kind: EventPermissionResolved, RequestID: requestID, Approved: &approved})
	_, err := stdin.Write(claudePermissionResponse(requestID, approve))
	return err
}

// interruptClaude asks the live process to stop its current turn without
// exiting. The exact control-protocol shape is Claude Code's documented
// stream-json control channel, not something confirmed against a real
// authenticated run in this environment (the sandbox this was built in
// cannot log in to the CLI it is driving) — if a real run shows a different
// shape, buildInterruptRequest is the one place to fix.
func (m *Manager) interruptClaude(ps *pilotSession) error {
	ps.mu.Lock()
	running, stdin := ps.running, ps.stdin
	ps.mu.Unlock()
	if !running || stdin == nil {
		return ErrNotRunning
	}
	_, err := stdin.Write(buildInterruptRequest())
	return err
}

func (m *Manager) cancelClaude(ps *pilotSession) error {
	ps.mu.Lock()
	cmd := ps.cmd
	ps.stoppedByUser = true
	ps.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return ErrNotRunning
	}
	return cmd.Process.Kill()
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

// claudePermissionResponse answers a control_request{subtype:"can_use_tool"}
// the CLI sent asking whether a tool call may proceed. Shape follows Claude
// Code's documented Agent SDK control protocol; not live-confirmed here for
// the same auth reason as buildInterruptRequest.
func claudePermissionResponse(requestID string, approve bool) []byte {
	body := map[string]any{"behavior": "deny", "message": "denied from LiveAgentsView"}
	if approve {
		body = map[string]any{"behavior": "allow"}
	}
	msg := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "can_use_tool",
			"request_id": requestID,
			"response":   body,
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

// claudeControlRequest is the CLI-initiated shape asking the driver for a
// permission decision.
type claudeControlRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Request   struct {
		Subtype  string          `json:"subtype"`
		ToolName string          `json:"tool_name"`
		Input    json.RawMessage `json:"input"`
	} `json:"request"`
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

// readClaudeStdout parses one JSON line at a time and turns it into
// transcript events / state transitions. Unrecognized top-level types are
// still shown, as a raw passthrough event, rather than silently dropped —
// this driver's model of the protocol is best-effort (see the interrupt/
// permission functions above), so anything it doesn't specifically know
// about should still reach the dashboard.
func (m *Manager) readClaudeStdout(ps *pilotSession, stdout io.Reader) {
	ctx := context.Background()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		m.handleClaudeLine(ctx, ps, line)
	}

	ps.mu.Lock()
	cmd := ps.cmd
	ps.running = false
	ps.mu.Unlock()

	var waitErr error
	if cmd != nil {
		waitErr = cmd.Wait()
	}
	m.finalizeProcess(ctx, ps, waitErr)
}

func (m *Manager) handleClaudeLine(ctx context.Context, ps *pilotSession, line []byte) {
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

	case "control_request":
		var cr claudeControlRequest
		if err := json.Unmarshal(line, &cr); err != nil {
			return
		}
		if cr.Request.Subtype != "can_use_tool" {
			return
		}
		ps.mu.Lock()
		ps.pending[cr.RequestID] = true
		ps.mu.Unlock()
		m.emit(ctx, ps, Event{Kind: EventPermissionRequest, ToolName: cr.Request.ToolName, ToolInput: cr.Request.Input, RequestID: cr.RequestID})
		m.upsert(ctx, ps, model.StateBlocked, "")

	case "result":
		var r claudeResultLine
		if err := json.Unmarshal(line, &r); err != nil {
			return
		}
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


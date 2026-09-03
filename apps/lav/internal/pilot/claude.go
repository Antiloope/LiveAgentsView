package pilot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// createClaude starts a brand-new Claude Code character's process
// immediately — its process just sits attached, waiting on stdin, once
// there is nothing left to do, which is why a Claude Code character created
// with no prompt is ready and awake rather than asleep like Cursor's.
func (m *Manager) createClaude(ctx context.Context, pc *pilotChar, prompt string) error {
	if err := m.spawnRunner(pc, nil); err != nil {
		return err
	}
	m.upsert(ctx, pc, model.ActivityReady, "")
	if prompt == "" {
		return nil
	}
	m.emit(ctx, pc, Event{Kind: EventUser, Text: prompt})
	if _, err := pc.stdin.Write(claudeUserMessage(prompt)); err != nil {
		return fmt.Errorf("send initial prompt: %w", err)
	}
	m.upsert(ctx, pc, model.ActivityWorking, "")
	return nil
}

// deliverClaude writes directly to an already-attached process's stdin, or
// wakes one first if none is attached — a character that has run before
// (it has a recorded provider session id) resumes it, one that never has
// starts a brand-new one.
func (m *Manager) deliverClaude(ctx context.Context, pc *pilotChar, text string) error {
	pc.mu.Lock()
	running, stdin := pc.running, pc.stdin
	pc.mu.Unlock()

	if !running || stdin == nil {
		ch, found, err := m.store.GetCharacter(ctx, pc.id)
		if err != nil {
			return err
		}
		var extra []string
		if found && ch.SessionID != "" {
			extra = []string{"--resume"}
		}
		if err := m.spawnRunner(pc, extra); err != nil {
			return err
		}
		stdin = pc.stdin
	}

	m.emit(ctx, pc, Event{Kind: EventUser, Text: text})
	if _, err := stdin.Write(claudeUserMessage(text)); err != nil {
		return err
	}
	m.upsert(ctx, pc, model.ActivityWorking, "")
	return nil
}

// interruptClaude asks the live process to stop its current turn without
// exiting. Live-confirmed shape: the CLI acknowledges the control_request
// with control_response{subtype:"success"} and the process survives, but the
// turn's own "result" line comes back as is_error:true,
// subtype:"error_during_execution" — indistinguishable on its own from a
// genuine failure. interrupted marks that the next such result line was
// asked for, not a crash — see handleClaudeLine's "result" case.
func (m *Manager) interruptClaude(pc *pilotChar) error {
	pc.mu.Lock()
	running, stdin := pc.running, pc.stdin
	if running {
		pc.interrupted = true
	}
	pc.mu.Unlock()
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
// reconnect) into transcript events and activity transitions. Unrecognized
// top-level types (including the control_response that acknowledges an
// interrupt) are still shown, as a raw passthrough event, rather than
// silently dropped.
func (m *Manager) handleClaudeLine(ctx context.Context, pc *pilotChar, line []byte) {
	if len(line) == 0 {
		return
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		m.emit(ctx, pc, Event{Kind: EventError, Text: string(line)})
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
				pc.mu.Lock()
				pc.lastText = block.Text
				pc.mu.Unlock()
				m.emit(ctx, pc, Event{Kind: EventAssistant, Text: block.Text})
			case "tool_use":
				m.emit(ctx, pc, Event{Kind: EventToolCall, ToolName: block.Name, ToolInput: block.Input, RequestID: block.ID})
			}
		}

	case "result":
		var r claudeResultLine
		if err := json.Unmarshal(line, &r); err != nil {
			return
		}
		pc.mu.Lock()
		lastText := pc.lastText
		interrupted := pc.interrupted
		pc.interrupted = false
		pc.mu.Unlock()
		if r.Result != "" {
			lastText = r.Result
		}
		if r.IsError {
			if interrupted && r.Subtype == "error_during_execution" {
				// A requested interrupt, not a crash: the process is still
				// alive and its stdin still open, so this reads as READY
				// (nothing to do, can be talked to again) rather than
				// FAILED.
				m.emit(ctx, pc, Event{Kind: EventSystem, Text: "turn interrupted"})
				m.afterTurn(ctx, pc, func() { m.upsert(ctx, pc, model.ActivityReady, lastText) })
				return
			}
			m.afterTurn(ctx, pc, func() { m.upsert(ctx, pc, model.ActivityFailed, lastText) })
			return
		}
		activity, unread := m.classify(lastText)
		m.afterTurn(ctx, pc, func() { m.finishQuest(ctx, pc, activity, lastText, unread) })

	case "system":
		var s claudeSystemLine
		if err := json.Unmarshal(line, &s); err == nil && s.Subtype == "init" {
			m.emit(ctx, pc, Event{Kind: EventSystem, Text: "session started"})
			if s.SessionID != "" {
				m.recordSessionID(ctx, pc, s.SessionID)
			}
		}
		// hook_started/hook_response/api_retry and other system noise are
		// not shown — the chat stays readable as a conversation, not a log.

	default:
		m.emit(ctx, pc, Event{Kind: EventSystem, Text: string(line)})
	}
}

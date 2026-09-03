package pilot

import (
	"context"
	"encoding/json"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// createCursor leaves a brand-new Cursor character asleep with nothing
// spawned — cursor-agent has no idle-process concept, every turn is its own
// one-shot invocation, so there is nothing to sit attached with nothing to
// do. A prompt, if given, wakes it immediately via deliverCursor.
func (m *Manager) createCursor(ctx context.Context, pc *pilotChar, prompt string) error {
	m.upsert(ctx, pc, model.ActivityReady, "")
	if prompt == "" {
		return nil
	}
	return m.deliverCursor(ctx, pc, prompt)
}

// deliverCursor starts one cursor-agent turn via a fresh pilot-runner —
// cursor-agent assigns its own chat id on the very first turn (there is
// nothing yet to --resume), which pilot-runner reports back on its init
// line and recordSessionID captures; every later turn passes that id along
// so the CLI's own --resume continues the same chat.
func (m *Manager) deliverCursor(ctx context.Context, pc *pilotChar, text string) error {
	ch, _, err := m.store.GetCharacter(ctx, pc.id)
	if err != nil {
		return err
	}
	extra := []string{"--prompt", text}
	if ch.SessionID != "" {
		extra = append(extra, "--provider-session", ch.SessionID)
	}
	if err := m.spawnRunner(pc, extra); err != nil {
		return err
	}
	m.emit(ctx, pc, Event{Kind: EventUser, Text: text})
	m.upsert(ctx, pc, model.ActivityWorking, "")
	return nil
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
// reconnect) into transcript events and activity transitions.
func (m *Manager) handleCursorLine(ctx context.Context, pc *pilotChar, line []byte) {
	if len(line) == 0 {
		return
	}
	var probe struct {
		Type      string `json:"type"`
		Subtype   string `json:"subtype"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		m.emit(ctx, pc, Event{Kind: EventError, Text: string(line)})
		return
	}
	if probe.SessionID != "" {
		m.recordSessionID(ctx, pc, probe.SessionID)
	}

	switch probe.Type {
	case "user":
		return // our own prompt, already emitted proactively

	case "system":
		if probe.Subtype == "init" {
			m.emit(ctx, pc, Event{Kind: EventSystem, Text: "session started"})
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
			pc.mu.Lock()
			pc.lastText = block.Text
			pc.mu.Unlock()
			m.emit(ctx, pc, Event{Kind: EventAssistant, Text: block.Text})
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
		m.emit(ctx, pc, Event{Kind: EventToolCall, ToolName: cursorToolName(t.ToolCall), ToolInput: t.ToolCall, RequestID: t.CallID})

	case "thinking":
		// Streamed reasoning deltas — too noisy for a chat transcript, same
		// call as Claude Code's hook_started/hook_response/api_retry noise.

	case "result":
		var r struct {
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
		}
		_ = json.Unmarshal(line, &r)
		pc.mu.Lock()
		lastText := pc.lastText
		pc.mu.Unlock()
		if r.Result != "" {
			lastText = r.Result
		}
		if r.IsError {
			m.afterTurn(ctx, pc, func() { m.upsert(ctx, pc, model.ActivityFailed, lastText) })
			return
		}
		activity, unread := m.classify(lastText)
		m.afterTurn(ctx, pc, func() { m.finishQuest(ctx, pc, activity, lastText, unread) })

	default:
		m.emit(ctx, pc, Event{Kind: EventSystem, Text: string(line)})
	}
}

package ingest

import (
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// claudeCodePayload covers the fields Claude Code's hooks are documented to
// send that this adapter cares about. Unknown fields are ignored by
// encoding/json, so payloads with more fields than this still parse fine.
type claudeCodePayload struct {
	SessionID            string `json:"session_id"`
	Cwd                  string `json:"cwd"`
	HookEventName        string `json:"hook_event_name"`
	NotificationType     string `json:"notification_type"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

// ParseClaudeCode maps one Claude Code hook payload to a Signal.
// event is the hook name (SessionStart, Notification, Stop, ...), taken from
// which hook entry invoked the forwarder script rather than trusted purely
// from the payload body, since that is known statically at "lav init" time.
func ParseClaudeCode(event string, body []byte) (model.Signal, error) {
	var p claudeCodePayload
	if len(body) > 0 {
		if err := json.Unmarshal(body, &p); err != nil {
			return model.Signal{}, fmt.Errorf("decode claude code payload: %w", err)
		}
	}

	sig := model.Signal{
		Provider:    model.ProviderClaudeCode,
		SessionID:   p.SessionID,
		Cwd:         p.Cwd,
		HookEvent:   event,
		LastMessage: p.LastAssistantMessage,
		Raw:         string(body),
	}

	switch event {
	case "SessionStart", "UserPromptSubmit":
		sig.State = model.StateWorking
	case "SessionEnd":
		sig.State = model.StateIdle
	case "Notification":
		switch p.NotificationType {
		case "permission_prompt":
			sig.State = model.StateBlocked
		case "agent_needs_input", "idle_prompt":
			sig.State = model.StateWaiting
		default:
			// Notification fires for more than permissions (auth, quota,
			// elicitation...) — treat anything we don't specifically
			// recognize as "still working" rather than guessing wrong.
			sig.State = model.StateWorking
		}
	case "StopFailure":
		sig.State = model.StateFailed
	case "Stop", "SubagentStop":
		// Ambiguous by design: Claude Code's Stop payload carries no reason
		// field. Leave State empty — the caller runs the classifier on
		// LastMessage to resolve WAITING vs DONE.
	default:
		sig.State = model.StateWorking
	}

	if sig.SessionID == "" {
		return sig, fmt.Errorf("claude code %s payload missing session_id", event)
	}
	return sig, nil
}

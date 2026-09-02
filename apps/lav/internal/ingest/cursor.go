package ingest

import (
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// cursorHookPayload's field names are best-effort from documentation
// research, not verified against a live payload — cursor-agent is not
// installed on the machine this was built on. Verify against a real payload
// before relying on this in production.
type cursorHookPayload struct {
	SessionID   string `json:"sessionId"`
	Cwd         string `json:"cwd"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	LastMessage string `json:"lastMessage"`
}

// ParseCursor maps a cursor-agent hook payload to a Signal. Cursor has no
// dedicated BLOCKED signal, so this adapter never emits that state.
func ParseCursor(event string, body []byte) (model.Signal, error) {
	var p cursorHookPayload
	if len(body) > 0 {
		if err := json.Unmarshal(body, &p); err != nil {
			return model.Signal{}, fmt.Errorf("decode cursor payload: %w", err)
		}
	}

	sig := model.Signal{
		Provider:    model.ProviderCursor,
		SessionID:   p.SessionID,
		Cwd:         p.Cwd,
		HookEvent:   event,
		LastMessage: p.LastMessage,
		Raw:         string(body),
	}

	switch event {
	case "sessionStart", "beforeShellExecution", "afterShellExecution", "afterFileEdit", "postToolUse":
		sig.State = model.StateWorking
	case "postToolUseFailure":
		// A single tool call failing is not the same as the session
		// failing — do not mark the whole session FAILED over one flaky
		// command.
		sig.State = model.StateWorking
	case "stop":
		switch p.Status {
		case "error", "aborted":
			sig.State = model.StateFailed
		default: // "completed", or unset
			// Ambiguous: classifier resolves WAITING vs DONE.
		}
	case "sessionEnd":
		if p.Reason == "error" {
			sig.State = model.StateFailed
		} else {
			sig.State = model.StateIdle
		}
	default:
		sig.State = model.StateWorking
	}

	if sig.SessionID == "" {
		return sig, fmt.Errorf("cursor %s payload missing sessionId", event)
	}
	return sig, nil
}

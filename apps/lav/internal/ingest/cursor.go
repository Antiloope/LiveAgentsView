package ingest

import (
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// cursorHookPayload's field names are confirmed for sessionStart/sessionEnd
// against a real cursor-agent hook payload. stop and postToolUseFailure
// never fired during that verification (a read-only run never reaches a
// tool-using turn), so FinalStatus/Status/LastMessage for those two events
// are still guesses — checked in snake_case, matching every field that is
// confirmed, rather than the previous camelCase guess.
type cursorHookPayload struct {
	SessionID      string   `json:"session_id"`
	WorkspaceRoots []string `json:"workspace_roots"`
	FinalStatus    string   `json:"final_status"`
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	LastMessage    string   `json:"last_message"`
}

// outcome returns whichever of final_status/status the payload actually
// carried — see the confirmed-vs-guessed note on cursorHookPayload above.
func (p cursorHookPayload) outcome() string {
	if p.FinalStatus != "" {
		return p.FinalStatus
	}
	return p.Status
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

	cwd := ""
	if len(p.WorkspaceRoots) > 0 {
		cwd = p.WorkspaceRoots[0]
	}

	sig := model.Signal{
		Provider:    model.ProviderCursor,
		SessionID:   p.SessionID,
		Cwd:         cwd,
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
		switch p.outcome() {
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
		return sig, fmt.Errorf("cursor %s payload missing session_id", event)
	}
	return sig, nil
}

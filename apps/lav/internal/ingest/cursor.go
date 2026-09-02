package ingest

import (
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// cursorHookPayload's field names are confirmed for sessionStart/sessionEnd
// against a real `agent` (a symlink to `cursor-agent`, confirmed via `ls -la
// $(which agent)`) install, version 2026.08.31-4057e58, on 2026-09-02 — see
// docs/sdd/specs/native-host-runtime.md Validation. `stop` and
// `postToolUseFailure` did not fire in that session (a read-only `--mode
// ask` run never reaches a tool-using turn) and stay best-effort: Status is
// a documentation-era guess, FinalStatus is the sibling field confirmed on
// sessionEnd, checked first since Cursor's real payloads are snake_case
// throughout and a shared field name across event types is plausible but
// unconfirmed for `stop` specifically.
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

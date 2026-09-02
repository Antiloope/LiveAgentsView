package ingest

import (
	"encoding/json"
	"fmt"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// codexNotifyPayload matches the one currently-documented Codex `notify`
// event type, agent-turn-complete. Field names use hyphens per Codex's own
// docs. This is the ONLY signal Codex gives us at Hooks fidelity — no
// BLOCKED, no FAILED (see docs/03-decisions.md 2026-09-01 "Canonical
// event/state model" and the spec's Event model table). Every call here is
// ambiguous by nature: State is always left empty for the classifier.
type codexNotifyPayload struct {
	Type                  string `json:"type"`
	ThreadID              string `json:"thread-id"`
	TurnID                string `json:"turn-id"`
	Cwd                   string `json:"cwd"`
	LastAssistantMessage string `json:"last-assistant-message"`
}

// ParseCodex maps a Codex `notify` payload to a Signal. event/hook-type
// distinction does not apply here — Codex's notify only fires one shape.
func ParseCodex(body []byte) (model.Signal, error) {
	var p codexNotifyPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return model.Signal{}, fmt.Errorf("decode codex payload: %w", err)
	}
	if p.ThreadID == "" {
		return model.Signal{}, fmt.Errorf("codex payload missing thread-id")
	}

	return model.Signal{
		Provider:    model.ProviderCodex,
		SessionID:   p.ThreadID,
		Cwd:         p.Cwd,
		HookEvent:   p.Type,
		LastMessage: p.LastAssistantMessage,
		Raw:         string(body),
		// State intentionally left empty: caller runs the classifier.
	}, nil
}

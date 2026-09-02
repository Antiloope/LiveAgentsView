// Package model holds the canonical types every provider adapter and the
// daemon agree on.
package model

import "time"

type Provider string

const (
	ProviderClaudeCode Provider = "claude-code"
	ProviderCodex      Provider = "codex"
	ProviderCursor     Provider = "cursor"
)

// Fidelity has exactly one value now that adopted/hooks sessions are gone —
// kept as a field (not collapsed away) so the Session shape and its stored
// rows don't change; every session LiveAgentsView creates is Driver.
type Fidelity string

const FidelityDriver Fidelity = "driver"

type State string

const (
	StateWorking State = "working"
	StateWaiting State = "waiting"
	StateBlocked State = "blocked"
	StateDone    State = "done"
	StateFailed  State = "failed"
	StateIdle    State = "idle"
)

// Session is one tracked agent session, as persisted and served to the UI.
type Session struct {
	ID          string    `json:"id"`
	Provider    Provider  `json:"provider"`
	Fidelity    Fidelity  `json:"fidelity"`
	Cwd         string    `json:"cwd"`
	Repo        string    `json:"repo"`
	Branch      string    `json:"branch"`
	Worktree    string    `json:"worktree"`
	State       State     `json:"state"`
	LastMessage string    `json:"last_message"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

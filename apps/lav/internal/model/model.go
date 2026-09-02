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

type Fidelity string

const (
	FidelityDriver  Fidelity = "driver"
	FidelityHooks   Fidelity = "hooks"
	FidelityTailing Fidelity = "tailing"
)

type State string

const (
	StateWorking State = "working"
	StateWaiting State = "waiting"
	StateBlocked State = "blocked"
	StateDone    State = "done"
	StateFailed  State = "failed"
	StateIdle    State = "idle"
)

// Signal is what a provider adapter derives from one raw hook payload.
// State is left empty when the raw event is ambiguous (every provider's
// "turn ended" signal looks the same whether the agent finished or asked a
// question) — the caller then runs the end-of-turn classifier on
// LastMessage to resolve it to StateWaiting or StateDone.
type Signal struct {
	Provider    Provider
	SessionID   string
	Cwd         string
	Repo        string
	Branch      string
	HookEvent   string
	State       State
	LastMessage string
	Raw         string
}

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

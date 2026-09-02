// Package pilot launches Claude Code and Cursor as child processes on behalf
// of the dashboard, streams their transcript live, and routes free-form
// messages, permission decisions, interrupts and cancellation back to them.
// Unlike internal/ingest, which only ever receives a hook POST, this package
// is the one part of the daemon that spawns processes and writes to their
// stdin.
package pilot

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
)

// Event is one transcript entry, persisted as JSON and broadcast as-is to a
// session's SSE subscribers so the frontend parses the exact same shape
// whether it came from history or live.
type Event struct {
	Kind      string          `json:"kind"`
	Text      string          `json:"text,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Approved  *bool           `json:"approved,omitempty"`
	At        time.Time       `json:"at"`
}

const (
	EventUser               = "user"
	EventAssistant          = "assistant"
	EventToolCall           = "tool_call"
	EventPermissionRequest  = "permission_request"
	EventPermissionResolved = "permission_resolved"
	EventSystem             = "system"
	EventError              = "error"
)

// LaunchSpec is what the dashboard's "new piloted session" form submits.
type LaunchSpec struct {
	Provider model.Provider
	Cwd      string
	Branch   string
	Prompt   string
}

// ErrNotFound is returned when an action targets a session Manager does not
// know about (never launched here, or the daemon restarted since).
var ErrNotFound = fmt.Errorf("piloted session not found")

// ErrNotRunning is returned when an action needs a live process and none is
// attached — the session exists but its process died or was never resumed.
var ErrNotRunning = fmt.Errorf("piloted session has no running process")

// ErrTurnInProgress is Cursor-specific: each message is its own one-shot
// process, so a second one cannot start until the current one exits or is
// interrupted.
var ErrTurnInProgress = fmt.Errorf("a turn is already in progress for this session")

// Manager owns every piloted session's child process for the life of this
// daemon process. It never persists process handles — a restart always
// leaves every session with no running process, resolved via Resume.
type Manager struct {
	store      *store.Store
	classifier classifier.Classifier
	onSession  func(model.Session)

	mu       sync.Mutex
	sessions map[string]*pilotSession
}

// pilotSession is the in-memory state for one piloted session's live
// process, guarded by mu for the fields a concurrent stdin write or stdout
// read can race on.
type pilotSession struct {
	id       string
	provider model.Provider
	cwd      string
	branch   string
	hub      *sse.Hub

	mu            sync.Mutex
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	running       bool
	stoppedByUser bool // set before Kill() so exit handling reports IDLE, not FAILED
	lastText      string
	pending       map[string]bool // request_id -> true while awaiting a permission decision
}

func NewManager(st *store.Store, cls classifier.Classifier, onSession func(model.Session)) *Manager {
	return &Manager{
		store:      st,
		classifier: cls,
		onSession:  onSession,
		sessions:   make(map[string]*pilotSession),
	}
}

func newUUID() string {
	var b [16]byte
	// crypto/rand.Read never returns a short read or a non-nil error on any
	// platform this project targets — nothing meaningful to do with an
	// error here beyond a worse-than-random ID.
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Hub returns a session's transcript SSE hub, creating it on first access so
// a subscriber connecting before Launch's goroutine has fully started still
// gets one to wait on.
func (m *Manager) Hub(id string) *sse.Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	if !ok {
		ps = &pilotSession{id: id, hub: sse.NewHub(), pending: map[string]bool{}}
		m.sessions[id] = ps
	}
	return ps.hub
}

func (m *Manager) get(id string) (*pilotSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	return ps, ok
}

func (m *Manager) getOrCreate(id string) *pilotSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	if !ok {
		ps = &pilotSession{id: id, hub: sse.NewHub(), pending: map[string]bool{}}
		m.sessions[id] = ps
	}
	return ps
}

// Events returns a session's persisted transcript, oldest first.
func (m *Manager) Events(ctx context.Context, id string) ([]json.RawMessage, error) {
	raws, err := m.store.ListEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(raws))
	for i, r := range raws {
		out[i] = json.RawMessage(r)
	}
	return out, nil
}

// emit persists one transcript event and broadcasts it to the session's live
// subscribers.
func (m *Manager) emit(ctx context.Context, ps *pilotSession, ev Event) {
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if err := m.store.AppendEvent(ctx, ps.id, ps.provider, "pilot:"+ev.Kind, "", string(b)); err != nil {
		// Persistence failing does not stop the live stream — the user
		// still sees it happen even if history replay would miss it.
		_ = err
	}
	ps.hub.Broadcast(ev)
}

// upsert persists the session's canonical state and tells the daemon to
// broadcast it on the same global hub adopted sessions use, so piloted
// sessions land in the one dashboard and attention queue.
func (m *Manager) upsert(ctx context.Context, ps *pilotSession, state model.State, lastMessage string) {
	now := time.Now().UTC()
	createdAt := now
	if existing, found, _ := m.store.GetSession(ctx, ps.id); found {
		createdAt = existing.CreatedAt
	}
	sess := model.Session{
		ID:          ps.id,
		Provider:    ps.provider,
		Fidelity:    model.FidelityDriver,
		Cwd:         ps.cwd,
		Repo:        filepath.Base(ps.cwd),
		Branch:      ps.branch,
		State:       state,
		LastMessage: lastMessage,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	if err := m.store.UpsertSession(ctx, sess); err != nil {
		return
	}
	if m.onSession != nil {
		m.onSession(sess)
	}
}

// classify resolves an ambiguous end-of-turn signal exactly like the hooks
// path does: empty means still ambiguous after a non-error exit, so run the
// shared classifier over the last assistant text.
func (m *Manager) classify(lastText string) model.State {
	switch m.classifier.Classify(lastText) {
	case classifier.VerdictWaiting:
		return model.StateWaiting
	default:
		return model.StateDone
	}
}

// finalizeProcess runs once a piloted process has actually exited (stdout
// closed and Wait returned) for either provider. A user-requested stop
// (interrupt or cancel, both set stoppedByUser before killing) always reads
// as IDLE, distinct from an unrequested exit, which is a real crash.
func (m *Manager) finalizeProcess(ctx context.Context, ps *pilotSession, waitErr error) {
	ps.mu.Lock()
	stopped := ps.stoppedByUser
	ps.stoppedByUser = false
	lastText := ps.lastText
	ps.mu.Unlock()

	if stopped {
		m.upsert(ctx, ps, model.StateIdle, lastText)
		return
	}
	if waitErr == nil {
		return // a clean exit already went through the "result" line handler
	}
	m.emit(ctx, ps, Event{Kind: EventError, Text: waitErr.Error()})
	m.upsert(ctx, ps, model.StateFailed, lastText)
}

// resolve returns the in-memory session state for id, reconstructing it from
// SQLite if this Manager has no live record (a fresh daemon process — every
// piloted session loses its in-memory state on restart, which is exactly
// what Resume exists to recover from).
func (m *Manager) resolve(ctx context.Context, id string) (*pilotSession, error) {
	m.mu.Lock()
	ps, ok := m.sessions[id]
	m.mu.Unlock()
	if ok {
		return ps, nil
	}

	sess, found, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found || sess.Fidelity != model.FidelityDriver {
		return nil, ErrNotFound
	}

	ps = m.getOrCreate(id)
	ps.provider = sess.Provider
	ps.cwd = sess.Cwd
	ps.branch = sess.Branch
	return ps, nil
}

// ReconcileOnStartup corrects every piloted session a previous process left
// in a "live" state. A fresh daemon start has no process attached to any of
// them regardless of what was last persisted, so leaving them as WORKING/
// WAITING/BLOCKED would silently lie to the dashboard — see Resume for how
// the user gets a live process back.
func (m *Manager) ReconcileOnStartup(ctx context.Context) error {
	sessions, err := m.store.ListSessions(ctx)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.Fidelity != model.FidelityDriver {
			continue
		}
		switch sess.State {
		case model.StateWorking, model.StateWaiting, model.StateBlocked:
		default:
			continue
		}
		sess.State = model.StateIdle
		sess.UpdatedAt = time.Now().UTC()
		if err := m.store.UpsertSession(ctx, sess); err != nil {
			continue
		}
		if m.onSession != nil {
			m.onSession(sess)
		}
	}
	return nil
}

func (m *Manager) currentSession(ctx context.Context, ps *pilotSession) (model.Session, error) {
	sess, found, err := m.store.GetSession(ctx, ps.id)
	if err != nil {
		return model.Session{}, err
	}
	if !found {
		return model.Session{}, ErrNotFound
	}
	return sess, nil
}

// Launch starts a brand-new piloted session for spec.Provider.
func (m *Manager) Launch(ctx context.Context, spec LaunchSpec) (model.Session, error) {
	switch spec.Provider {
	case model.ProviderClaudeCode:
		ps := &pilotSession{hub: sse.NewHub(), pending: map[string]bool{}}
		if err := m.launchClaude(ctx, ps, spec, ""); err != nil {
			return model.Session{}, err
		}
		return m.currentSession(ctx, ps)
	case model.ProviderCursor:
		ps := &pilotSession{hub: sse.NewHub(), pending: map[string]bool{}}
		if err := m.launchCursor(ctx, ps, spec, ""); err != nil {
			return model.Session{}, err
		}
		return m.currentSession(ctx, ps)
	default:
		return model.Session{}, fmt.Errorf("unsupported piloted provider: %s", spec.Provider)
	}
}

// SendMessage delivers free-form text to a piloted session: over stdin for
// Claude Code's persistent process, or as a new --resume'd one-shot
// invocation for Cursor.
func (m *Manager) SendMessage(ctx context.Context, id, text string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode:
		return m.sendClaudeMessage(ps, text)
	case model.ProviderCursor:
		return m.sendCursorMessage(ctx, ps, text)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// ApprovePermission answers a pending permission request. Claude Code only —
// Cursor piloted sessions auto-approve and never raise one.
func (m *Manager) ApprovePermission(ctx context.Context, id, requestID string, approve bool) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	if ps.provider != model.ProviderClaudeCode {
		return fmt.Errorf("permission approval is not available for %s piloted sessions", ps.provider)
	}
	return m.approveClaudePermission(ps, requestID, approve)
}

// Interrupt stops the current turn. For Claude Code the process stays
// attached, ready for the next message; for Cursor there is no separate
// "current turn" to stop without ending the one-shot process that is running
// it, so this and Cancel do the same thing.
func (m *Manager) Interrupt(ctx context.Context, id string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode:
		return m.interruptClaude(ps)
	case model.ProviderCursor:
		return m.killCursor(ps)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// Cancel ends the piloted session's current process.
func (m *Manager) Cancel(ctx context.Context, id string) error {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch ps.provider {
	case model.ProviderClaudeCode:
		return m.cancelClaude(ps)
	case model.ProviderCursor:
		return m.killCursor(ps)
	default:
		return fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
}

// Resume re-attaches a piloted session with no live process — after a daemon
// restart, a crash, or an interrupt/cancel. For Claude Code this starts a
// new `--resume`'d process. Cursor has no standing process to re-attach
// (each message already is its own `--resume`'d invocation — see
// sendCursorMessage), so this just confirms the session is known and ready.
func (m *Manager) Resume(ctx context.Context, id string) (model.Session, error) {
	ps, err := m.resolve(ctx, id)
	if err != nil {
		return model.Session{}, err
	}
	ps.mu.Lock()
	running := ps.running
	ps.mu.Unlock()
	if running {
		return model.Session{}, fmt.Errorf("piloted session %s is already running", id)
	}

	switch ps.provider {
	case model.ProviderClaudeCode:
		spec := LaunchSpec{Provider: ps.provider, Cwd: ps.cwd, Branch: ps.branch}
		if err := m.launchClaude(ctx, ps, spec, ps.id); err != nil {
			return model.Session{}, err
		}
	case model.ProviderCursor:
		// Nothing to start.
	default:
		return model.Session{}, fmt.Errorf("unsupported piloted provider: %s", ps.provider)
	}
	return m.currentSession(ctx, ps)
}

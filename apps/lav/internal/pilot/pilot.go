// Package pilot launches Claude Code and Cursor as child processes on
// behalf of the dashboard, streams their transcript live, and routes
// free-form messages, interrupts and stops back to them.
//
// The actual provider process is not a direct child of this daemon: Manager
// spawns "lav pilot-runner" (internal/pilotrunner) fully detached — its own
// session, no shared pipes — which in turn owns the real `claude`/`agent`
// process, durably logs its stdout to disk, and exposes a Unix domain
// socket for control. Manager talks to a character's runner only over that
// socket (see internal/pilotwire), so a daemon restart never has to kill
// the child to clean up after itself, and reconnecting after one is just
// dialing the same socket again — see ReconcileOnStartup. That same socket
// is also the one signal a character's presence is ever computed from
// (IsAwake): SQLite never stores whether a process is alive.
package pilot

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotwire"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/territory"
)

// Event is one transcript entry, persisted as JSON and broadcast as-is to a
// character's SSE subscribers so the frontend parses the exact same shape
// whether it came from history or live.
type Event struct {
	Kind      string          `json:"kind"`
	Text      string          `json:"text,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	At        time.Time       `json:"at"`
}

const (
	EventUser      = "user"
	EventAssistant = "assistant"
	EventToolCall  = "tool_call"
	EventThinking  = "thinking"
	EventSystem    = "system"
	EventError     = "error"
)

// CreateSpec is what the recruit panel submits to bring a new character
// into camp.
type CreateSpec struct {
	Race          model.Race
	TerritoryMode model.TerritoryMode
	Cwd           string // source repo (own territory) or the directory itself (shared)
	Branch        string
	Class         model.Class
	Prompt        string
}

// modelArgs returns the CLI args for a --model flag, or nil when class is
// empty — omitting the flag entirely lets the provider's own CLI default
// apply rather than passing an empty value it would have to reject.
func modelArgs(class string) []string {
	if class == "" {
		return nil
	}
	return []string{"--model", class}
}

// ErrNotFound is returned when an action targets a character Manager does
// not know about (never created here, or the daemon restarted since and the
// row is gone).
var ErrNotFound = fmt.Errorf("character not found")

// ErrNotRunning is returned when an action needs a live process and none is
// attached.
var ErrNotRunning = fmt.Errorf("character has no running process")

// Manager owns every character's control connection to its detached
// pilot-runner. It never keeps a character's process itself alive — a
// restart's in-memory state is always rebuilt via ReconcileOnStartup, which
// reconnects to whatever is still actually running.
type Manager struct {
	store       *store.Store
	classifier  classifier.Classifier
	onCharacter func(model.Character)
	lavHome     string
	selfExe     string

	mu         sync.Mutex
	characters map[string]*pilotChar
}

// pilotChar is the in-memory state for one character's live process,
// guarded by mu for the fields a concurrent stdin write or a runner-line
// read can race on.
type pilotChar struct {
	id     string
	race   model.Race
	mode   model.TerritoryMode
	path   string // territory.Path: where the process actually runs
	source string // territory.Source
	branch string
	class  model.Class
	hub    *sse.Hub

	mu            sync.Mutex
	sessionID     string         // the provider-side conversation id, once known
	conn          net.Conn       // control socket to this character's pilot-runner, nil if not attached
	stdin         io.WriteCloser // relays to the child's real stdin via conn
	running       bool
	stoppedByUser bool // set before killing so exit handling reports READY, not FAILED
	interrupted   bool // set before sending an interrupt request so the result line reports READY, not FAILED
	lastText      string
	thinking      []string // cursor-agent reasoning deltas, accumulated until the run they belong to closes
	queue         []string // messages sent while working, delivered in order as each turn ends
	dismissed     bool     // set by Dismiss before killing; blocks any further persistence for this character
}

func (pc *pilotChar) isDismissed() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.dismissed
}

// NewManager builds a Manager. lavHome is where pilot-runner sockets,
// transcripts and offset files live; selfExe is this running binary's own
// path, re-exec'd as "lav pilot-runner" for every detached process.
func NewManager(st *store.Store, cls classifier.Classifier, lavHome, selfExe string, onCharacter func(model.Character)) *Manager {
	return &Manager{
		store:       st,
		classifier:  cls,
		lavHome:     lavHome,
		selfExe:     selfExe,
		onCharacter: onCharacter,
		characters:  make(map[string]*pilotChar),
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

// Hub returns a character's transcript SSE hub, creating it on first access
// so a subscriber connecting before Create's goroutine has fully started
// still gets one to wait on.
func (m *Manager) Hub(id string) *sse.Hub {
	return m.getOrCreate(id).hub
}

func (m *Manager) get(id string) (*pilotChar, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pc, ok := m.characters[id]
	return pc, ok
}

func (m *Manager) getOrCreate(id string) *pilotChar {
	m.mu.Lock()
	defer m.mu.Unlock()
	pc, ok := m.characters[id]
	if !ok {
		pc = &pilotChar{id: id, hub: sse.NewHub()}
		m.characters[id] = pc
	}
	return pc
}

// Events returns a character's persisted transcript, oldest first.
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

// IsAwake reports whether id's pilot-runner control socket is currently
// reachable — the one signal a character's presence is ever computed from.
func (m *Manager) IsAwake(id string) bool {
	conn, err := net.DialTimeout("unix", pilotwire.SocketPath(m.lavHome, id), 300*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// emit persists one transcript event and broadcasts it to the character's
// live subscribers.
func (m *Manager) emit(ctx context.Context, pc *pilotChar, ev Event) {
	if pc.isDismissed() {
		return
	}
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	if err := m.store.AppendEvent(ctx, pc.id, pc.race, "pilot:"+ev.Kind, "", string(b)); err != nil {
		// Persistence failing does not stop the live stream — the user
		// still sees it happen even if history replay would miss it.
		_ = err
	}
	pc.hub.Broadcast(ev)
}

// characterFromMemory builds the model.Character this manager currently
// believes pc to be, for persisting and broadcasting.
func (m *Manager) characterFromMemory(ctx context.Context, pc *pilotChar, activity model.Activity, lastMessage string) model.Character {
	now := time.Now().UTC()
	createdAt := now
	if existing, found, _ := m.store.GetCharacter(ctx, pc.id); found {
		createdAt = existing.CreatedAt
	}
	pc.mu.Lock()
	sessionID := pc.sessionID
	pc.mu.Unlock()
	return model.Character{
		ID:        pc.id,
		SessionID: sessionID,
		Race:      pc.race,
		Class:     pc.class,
		Activity:  activity,
		Territory: model.Territory{Mode: pc.mode, Path: pc.path, Source: pc.source, Branch: pc.branch},
		// Source, not Path: for own territory Path is the worktree's own
		// directory, named after the character id, not the project.
		Repo:        filepath.Base(pc.source),
		LastMessage: lastMessage,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
}

// upsert persists the character's activity and tells the daemon to
// broadcast it on its global stream. It never touches Unread or Archived —
// see store.UpsertCharacter.
func (m *Manager) upsert(ctx context.Context, pc *pilotChar, activity model.Activity, lastMessage string) {
	if pc.isDismissed() {
		return
	}
	ch := m.characterFromMemory(ctx, pc, activity, lastMessage)
	if err := m.store.UpsertCharacter(ctx, ch); err != nil {
		return
	}
	if m.onCharacter != nil {
		m.onCharacter(ch)
	}
}

// finishQuest persists the outcome of a turn that ended on its own —
// activity plus, when the quest ended without a question, the unread mark —
// in one write and one broadcast.
func (m *Manager) finishQuest(ctx context.Context, pc *pilotChar, activity model.Activity, lastMessage string, unread bool) {
	if pc.isDismissed() {
		return
	}
	ch := m.characterFromMemory(ctx, pc, activity, lastMessage)
	if err := m.store.UpsertCharacter(ctx, ch); err != nil {
		return
	}
	if unread {
		if _, _, err := m.store.SetUnread(ctx, pc.id, true); err != nil {
			return
		}
	}
	final, found, err := m.store.GetCharacter(ctx, pc.id)
	if err != nil || !found {
		return
	}
	if m.onCharacter != nil {
		m.onCharacter(final)
	}
}

// recordSessionID captures the provider's own conversation id the first
// time it is reported — Claude Code's --session-id echoed back, or
// cursor-agent's own assigned session id. Never overwritten once set.
func (m *Manager) recordSessionID(ctx context.Context, pc *pilotChar, sessionID string) {
	pc.mu.Lock()
	known := pc.sessionID != ""
	if !known {
		pc.sessionID = sessionID
	}
	pc.mu.Unlock()
	if known {
		return
	}
	_ = m.store.SetSessionID(ctx, pc.id, sessionID)
}

// classify resolves an ambiguous end-of-turn signal into the character's
// next activity, and whether it should carry the unread mark — set only
// when the quest ended without a question.
func (m *Manager) classify(lastText string) (model.Activity, bool) {
	if m.classifier.Classify(lastText) == classifier.VerdictWaiting {
		return model.ActivityWaiting, false
	}
	return model.ActivityReady, true
}

// afterTurn is called once a turn has genuinely ended on its own (not by a
// kill) to decide what happens next: if a message arrived while this
// character was working, it is delivered now as the next turn instead of
// finalizing — see SendMessage's queueing while working.
func (m *Manager) afterTurn(ctx context.Context, pc *pilotChar, finalize func()) {
	pc.mu.Lock()
	var next string
	if len(pc.queue) > 0 {
		next = pc.queue[0]
		pc.queue = pc.queue[1:]
	}
	pc.mu.Unlock()

	if next == "" {
		finalize()
		return
	}
	if err := m.deliverNow(ctx, pc, next); err != nil {
		finalize()
	}
}

// finalizeProcess runs once a character's process has actually exited, for
// either race. A user-requested stop (Interrupt-via-kill or Stop, both set
// stoppedByUser before killing) always reads as READY, distinct from an
// unrequested exit, which is a real failure. Either way, anything still
// queued is dropped — an explicit stop or a crash both cancel intent to
// keep going, rather than silently starting a new turn behind the user's
// back.
func (m *Manager) finalizeProcess(ctx context.Context, pc *pilotChar, waitErr error) {
	if pc.isDismissed() {
		return
	}
	pc.mu.Lock()
	stopped := pc.stoppedByUser
	pc.stoppedByUser = false
	lastText := pc.lastText
	pc.queue = nil
	pc.mu.Unlock()

	if stopped {
		m.upsert(ctx, pc, model.ActivityReady, lastText)
		return
	}
	if waitErr == nil {
		return // a clean exit already went through the "result" line handler
	}
	m.emit(ctx, pc, Event{Kind: EventError, Text: waitErr.Error()})
	m.upsert(ctx, pc, model.ActivityFailed, lastText)
}

// resolve returns the in-memory character state for id, reconstructing it
// from SQLite if this Manager has no live record for it yet (e.g. after a
// daemon restart, before ReconcileOnStartup's reconnect for it runs, or for
// a character that has never had a live process at all).
func (m *Manager) resolve(ctx context.Context, id string) (*pilotChar, error) {
	m.mu.Lock()
	pc, ok := m.characters[id]
	m.mu.Unlock()
	if ok {
		return pc, nil
	}

	ch, found, err := m.store.GetCharacter(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}

	pc = m.getOrCreate(id)
	pc.race = ch.Race
	pc.mode = ch.Territory.Mode
	pc.path = ch.Territory.Path
	pc.source = ch.Territory.Source
	pc.branch = ch.Territory.Branch
	pc.class = ch.Class
	pc.mu.Lock()
	pc.sessionID = ch.SessionID
	pc.mu.Unlock()
	return pc, nil
}

// ReconcileOnStartup runs once at daemon startup. For every known
// character, it dials that character's pilot-runner socket: if the
// detached process is still actually running, it reconnects (replaying any
// transcript lines produced while this daemon was down, with no
// duplicates and no drops — see reconnect) and leaves its activity
// untouched — a daemon restart never changes activity by itself. Only a
// character recorded as working whose process is confirmed gone is moved
// to failed, since nothing is actually running its quest any more. It then
// sweeps orphans: pilot files and own-territory worktrees with no
// character behind them any more.
func (m *Manager) ReconcileOnStartup(ctx context.Context) error {
	characters, err := m.store.ListCharacters(ctx)
	if err != nil {
		return err
	}

	known := make(map[string]bool, len(characters))
	for _, ch := range characters {
		known[ch.ID] = true
		if m.reconnect(ctx, ch) {
			continue
		}
		if ch.Activity == model.ActivityWorking {
			m.markDisappeared(ctx, ch, nil)
		}
	}

	if files, err := pilotwire.KnownFiles(m.lavHome); err == nil {
		for id, paths := range files {
			if known[id] {
				continue
			}
			for _, p := range paths {
				os.Remove(p)
			}
		}
	}
	territory.SweepOrphans(m.lavHome, known)
	return nil
}

// StartReconciler runs for as long as ctx is alive, checking every interval
// whether each character recorded as working still has a reachable runner
// — the same continuous correction ReconcileOnStartup does once, kept
// running so a runner that disappears mid-quest (kill -9, machine sleep,
// OOM) is caught without a daemon restart.
func (m *Manager) StartReconciler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.reconcileOnce(ctx)
		}
	}
}

func (m *Manager) reconcileOnce(ctx context.Context) {
	characters, err := m.store.ListCharacters(ctx)
	if err != nil {
		return
	}
	for _, ch := range characters {
		if ch.Activity != model.ActivityWorking || m.IsAwake(ch.ID) {
			continue
		}
		pc, _ := m.get(ch.ID)
		m.markDisappeared(ctx, ch, pc)
		if pc != nil {
			pc.mu.Lock()
			pc.running = false
			pc.conn = nil
			pc.mu.Unlock()
		}
	}
}

// markDisappeared moves a character whose runner has vanished while it was
// still recorded as working to failed, with a transcript entry saying so.
// pc is nil at startup reconciliation (nothing has resolved it into memory
// yet); when present, the event is also broadcast live on its hub.
func (m *Manager) markDisappeared(ctx context.Context, ch model.Character, pc *pilotChar) {
	ev := Event{Kind: EventSystem, Text: "the process disappeared", At: time.Now().UTC()}
	b, err := json.Marshal(ev)
	if err == nil {
		_ = m.store.AppendEvent(ctx, ch.ID, ch.Race, "reconcile", model.ActivityFailed, string(b))
	}
	if pc != nil {
		pc.hub.Broadcast(ev)
	}

	ch.Activity = model.ActivityFailed
	ch.UpdatedAt = time.Now().UTC()
	if err := m.store.UpsertCharacter(ctx, ch); err != nil {
		return
	}
	if m.onCharacter != nil {
		m.onCharacter(ch)
	}
}

// Create brings a brand-new character into camp: prepares its territory,
// then starts its process for races that sit attached with nothing to do
// (Claude Code) or leaves it asleep until its first message wakes it
// (Cursor, which has no idle-process concept) — see createClaude/
// createCursor. A prompt, if given, is delivered either way. The territory
// is torn down again if anything after it fails, so a failed create leaves
// nothing behind.
func (m *Manager) Create(ctx context.Context, spec CreateSpec) (_ model.Character, err error) {
	id := newUUID()
	terr, terrErr := territory.Prepare(m.lavHome, territory.Spec{
		Mode: spec.TerritoryMode, SourceRepo: spec.Cwd, Branch: spec.Branch, CharacterID: id,
	})
	if terrErr != nil {
		return model.Character{}, terrErr
	}
	defer func() {
		if err != nil {
			territory.Remove(terr)
			m.mu.Lock()
			delete(m.characters, id)
			m.mu.Unlock()
		}
	}()

	pc := m.getOrCreate(id)
	pc.race = spec.Race
	pc.mode = terr.Mode
	pc.path = terr.Path
	pc.source = terr.Source
	pc.branch = terr.Branch
	pc.class = spec.Class

	switch spec.Race {
	case model.RaceClaudeCode:
		err = m.createClaude(ctx, pc, spec.Prompt)
	case model.RaceCursor:
		err = m.createCursor(ctx, pc, spec.Prompt)
	default:
		err = fmt.Errorf("unsupported race: %s", spec.Race)
	}
	if err != nil {
		return model.Character{}, err
	}
	return m.currentCharacter(ctx, pc)
}

func (m *Manager) currentCharacter(ctx context.Context, pc *pilotChar) (model.Character, error) {
	ch, found, err := m.store.GetCharacter(ctx, pc.id)
	if err != nil {
		return model.Character{}, err
	}
	if !found {
		return model.Character{}, ErrNotFound
	}
	return ch, nil
}

// SendMessage delivers free-form text to a character: queued if it is
// currently working (flushed once that quest ends — see afterTurn), woken
// if asleep, written directly otherwise. Never rejected with "a turn is
// already in progress".
func (m *Manager) SendMessage(ctx context.Context, id, text string) error {
	pc, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	ch, found, err := m.store.GetCharacter(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	if ch.Activity == model.ActivityWorking {
		pc.mu.Lock()
		pc.queue = append(pc.queue, text)
		pc.mu.Unlock()
		return nil
	}
	return m.deliverNow(ctx, pc, text)
}

// deliverNow starts pc's next turn with text, race-specific.
func (m *Manager) deliverNow(ctx context.Context, pc *pilotChar, text string) error {
	switch pc.race {
	case model.RaceClaudeCode:
		return m.deliverClaude(ctx, pc, text)
	case model.RaceCursor:
		return m.deliverCursor(ctx, pc, text)
	default:
		return fmt.Errorf("unsupported race: %s", pc.race)
	}
}

// Interrupt stops the current turn. For Claude Code the process stays
// attached, ready for the next message; for Cursor there is no separate
// "current turn" to stop without ending the one-shot process running it,
// so this and Stop do the same thing.
func (m *Manager) Interrupt(ctx context.Context, id string) error {
	pc, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	switch pc.race {
	case model.RaceClaudeCode:
		return m.interruptClaude(pc)
	case model.RaceCursor:
		return m.killProcess(pc)
	default:
		return fmt.Errorf("unsupported race: %s", pc.race)
	}
}

// Stop ends the character's current process outright.
func (m *Manager) Stop(ctx context.Context, id string) error {
	pc, err := m.resolve(ctx, id)
	if err != nil {
		return err
	}
	return m.killProcess(pc)
}

// Archive stops a character's process (in any activity, including
// working), keeps its transcript and territory, and takes it out of camp.
// Already being asleep does not fail it — ErrNotRunning is expected then.
func (m *Manager) Archive(ctx context.Context, id string) (model.Character, error) {
	pc, err := m.resolve(ctx, id)
	if err != nil {
		return model.Character{}, err
	}
	if err := m.killProcess(pc); err != nil && err != ErrNotRunning {
		return model.Character{}, err
	}
	ch, found, err := m.store.SetArchived(ctx, id, true)
	if err != nil {
		return model.Character{}, err
	}
	if !found {
		return model.Character{}, ErrNotFound
	}
	if m.onCharacter != nil {
		m.onCharacter(ch)
	}
	return ch, nil
}

// Unarchive returns a character to camp. Talking to it is what actually
// wakes it — SendMessage's own wake path handles that, nothing extra here.
func (m *Manager) Unarchive(ctx context.Context, id string) (model.Character, error) {
	ch, found, err := m.store.SetArchived(ctx, id, false)
	if err != nil {
		return model.Character{}, err
	}
	if !found {
		return model.Character{}, ErrNotFound
	}
	if m.onCharacter != nil {
		m.onCharacter(ch)
	}
	return ch, nil
}

// MarkRead clears a character's unread mark — the interface's explicit
// signal that the user has read its transcript. Nothing else clears it.
func (m *Manager) MarkRead(ctx context.Context, id string) error {
	ch, found, err := m.store.SetUnread(ctx, id, false)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	if m.onCharacter != nil {
		m.onCharacter(ch)
	}
	return nil
}

// Dismiss removes a character for good: stops its process, deletes its row,
// events and pilot files, and removes its territory if it is an own
// worktree with no uncommitted changes. leftAt is non-empty when an own
// worktree was left in place (dirty), so the caller can tell the user where
// it is.
func (m *Manager) Dismiss(ctx context.Context, id string) (leftAt string, err error) {
	pc, err := m.resolve(ctx, id)
	if err != nil {
		return "", err
	}
	pc.mu.Lock()
	pc.dismissed = true
	pc.mu.Unlock()
	if kerr := m.killProcess(pc); kerr != nil && kerr != ErrNotRunning {
		return "", kerr
	}

	ch, found, err := m.store.GetCharacter(ctx, id)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrNotFound
	}

	removed, err := territory.Remove(ch.Territory)
	if err != nil {
		return "", err
	}

	if _, err := m.store.DeleteCharacter(ctx, id); err != nil {
		return "", err
	}
	for _, p := range pilotFiles(m.lavHome, id) {
		os.Remove(p)
	}

	m.mu.Lock()
	delete(m.characters, id)
	m.mu.Unlock()

	if !removed && ch.Territory.Mode == model.TerritoryOwn {
		return ch.Territory.Path, nil
	}
	return "", nil
}

func pilotFiles(lavHome, id string) []string {
	return []string{
		pilotwire.SocketPath(lavHome, id),
		pilotwire.TranscriptPath(lavHome, id),
		pilotwire.OffsetPath(lavHome, id),
		pilotwire.StderrPath(lavHome, id),
	}
}

// --- pilot-runner process management ---------------------------------------

// spawnRunner starts "lav pilot-runner" for pc fully detached from this
// daemon process — its own session (SysProcAttr.Setsid), stdio pointed at
// /dev/null rather than shared pipes — so nothing about the runner's
// lifetime depends on this process staying alive, then dials its control
// socket and attaches from the beginning (since=0: a freshly spawned runner
// has nothing to replay). extraArgs carries the race-specific flags a
// caller needs for this particular launch (--resume, --prompt,
// --provider-session).
func (m *Manager) spawnRunner(pc *pilotChar, extraArgs []string) error {
	args := []string{
		"pilot-runner",
		"--session", pc.id,
		"--provider", string(pc.race),
		"--cwd", pc.path,
		"--lav-home", m.lavHome,
	}
	args = append(args, modelArgs(string(pc.class))...)
	args = append(args, extraArgs...)

	cmd := exec.Command(m.selfExe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pilot-runner: %w", err)
	}
	go cmd.Wait() // reap whenever it exits; never blocks this daemon

	conn, err := dialWithRetry(pilotwire.SocketPath(m.lavHome, pc.id), 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to pilot-runner: %w", err)
	}
	if err := pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "attach", Since: 0}); err != nil {
		conn.Close()
		return fmt.Errorf("attach to pilot-runner: %w", err)
	}

	pc.mu.Lock()
	pc.conn = conn
	pc.stdin = &socketStdin{conn: conn}
	pc.running = true
	pc.stoppedByUser = false
	pc.mu.Unlock()

	go m.readFromRunner(pc, conn)
	return nil
}

// reconnect dials an already-running character's pilot-runner socket —
// used only by ReconcileOnStartup. Returns false when the socket is gone
// (the process genuinely exited while this daemon was down).
func (m *Manager) reconnect(ctx context.Context, ch model.Character) bool {
	conn, err := net.DialTimeout("unix", pilotwire.SocketPath(m.lavHome, ch.ID), 500*time.Millisecond)
	if err != nil {
		return false
	}
	since := pilotwire.ReadOffset(m.lavHome, ch.ID)
	if err := pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "attach", Since: since}); err != nil {
		conn.Close()
		return false
	}

	pc := m.getOrCreate(ch.ID)
	pc.race = ch.Race
	pc.mode = ch.Territory.Mode
	pc.path = ch.Territory.Path
	pc.source = ch.Territory.Source
	pc.branch = ch.Territory.Branch
	pc.class = ch.Class
	pc.mu.Lock()
	pc.sessionID = ch.SessionID
	pc.conn = conn
	pc.stdin = &socketStdin{conn: conn}
	pc.running = true
	pc.mu.Unlock()

	go m.readFromRunner(pc, conn)
	return true
}

// readFromRunner decodes transcript lines forwarded over a pilot-runner's
// control socket and feeds them through the same per-race line handler live
// stdout used before this character had restart continuity — the wire
// format changed, the parsing didn't. An explicit "exited" frame means the
// child process itself is gone, handled exactly like today's process exit.
// A plain disconnect with no such frame means only the *connection* died —
// almost always this daemon process shutting down for a restart — and must
// not be mistaken for the piloted process exiting: the character's
// persisted activity is left untouched, for ReconcileOnStartup or the
// running reconciler to resolve for real on the next check.
func (m *Manager) readFromRunner(pc *pilotChar, conn net.Conn) {
	ctx := context.Background()
	sc := pilotwire.NewScanner(conn)
	for sc.Scan() {
		var msg pilotwire.ServerMsg
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Exited {
			pc.mu.Lock()
			pc.running = false
			pc.conn = nil
			pc.mu.Unlock()
			var waitErr error
			if msg.Err != "" {
				waitErr = fmt.Errorf("%s", msg.Err)
			}
			m.finalizeProcess(ctx, pc, waitErr)
			return
		}
		if msg.Line != "" {
			switch pc.race {
			case model.RaceClaudeCode:
				m.handleClaudeLine(ctx, pc, []byte(msg.Line))
			case model.RaceCursor:
				m.handleCursorLine(ctx, pc, []byte(msg.Line))
			}
		}
		if msg.Seq > 0 {
			_ = pilotwire.WriteOffset(m.lavHome, pc.id, msg.Seq)
		}
	}

	pc.mu.Lock()
	pc.running = false
	pc.conn = nil
	pc.mu.Unlock()
}

// killProcess ends a character's process over its pilot-runner control
// socket, whichever race launched it.
func (m *Manager) killProcess(pc *pilotChar) error {
	pc.mu.Lock()
	conn := pc.conn
	// Only set once a kill is actually being sent — set unconditionally,
	// this would stick around on a character with no live process (e.g. a
	// second Stop click, or Archive/Dismiss on an already-asleep one) for a
	// later, genuine crash to wrongly read as a clean user-requested stop.
	if conn != nil {
		pc.stoppedByUser = true
	}
	pc.mu.Unlock()

	if conn == nil {
		return ErrNotRunning
	}
	return pilotwire.Encode(conn, pilotwire.ClientMsg{Op: "kill"})
}

// socketStdin relays writes meant for a character's real stdin over the
// control socket to the pilot-runner actually holding that pipe.
type socketStdin struct{ conn net.Conn }

func (w *socketStdin) Write(p []byte) (int, error) {
	if err := pilotwire.Encode(w.conn, pilotwire.ClientMsg{Op: "stdin", Data: string(p)}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *socketStdin) Close() error { return w.conn.Close() }

// dialWithRetry absorbs the brief window between starting pilot-runner and
// its control socket existing — process creation and socket bind are not
// synchronous from this side.
func dialWithRetry(path string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(50 * time.Millisecond)
	}
}

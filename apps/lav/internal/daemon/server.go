// Package daemon wires the HTTP server: the JSON API and SSE stream for the
// dashboard, character control, and serving the embedded frontend. Binds to
// 127.0.0.1 only: exposing it would let anything on the network launch or
// drive a character on this machine.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilot"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/territory"
)

const maxHookBodyBytes = 1 << 20 // 1 MiB is generous for a character request body

// reconcileInterval is how often the running reconciler re-checks every
// working character's presence — see pilot.Manager.StartReconciler.
const reconcileInterval = 3 * time.Second

type Server struct {
	store        *store.Store
	hub          *sse.Hub
	pilots       *pilot.Manager
	mux          *http.ServeMux
	handler      http.Handler // mux wrapped with secure(); what ServeHTTP actually calls
	cursorModels cursorModelsCache
}

// New builds the daemon. lavHome is this process's own data directory and
// selfExe its own binary path — both passed through to pilot.Manager, which
// needs them to spawn and reconnect to characters' detached pilot-runner
// processes. port is the one this daemon is actually listening on (LAV_PORT
// can change it from the default), used to compute the loopback Host/Origin
// values secure() accepts.
func New(st *store.Store, cls classifier.Classifier, webFS fs.FS, lavHome, selfExe, port string) *Server {
	s := &Server{
		store: st,
		hub:   sse.NewHub(),
		mux:   http.NewServeMux(),
	}
	s.pilots = pilot.NewManager(st, cls, lavHome, selfExe, func(ch model.Character) { s.hub.Broadcast(s.view(ch)) })
	if err := s.pilots.ReconcileOnStartup(context.Background()); err != nil {
		log.Printf("reconcile characters on startup: %v", err)
	}
	go s.pilots.StartReconciler(context.Background(), reconcileInterval)
	s.routes(webFS)
	s.handler = secure(newSecureConfig(port), s.mux)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

func (s *Server) routes(webFS fs.FS) {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	s.mux.HandleFunc("GET /api/characters", s.handleListCharacters)
	s.mux.HandleFunc("POST /api/characters", s.handleCreateCharacter)
	s.mux.HandleFunc("POST /api/characters/{id}/message", s.handleSendMessage)
	s.mux.HandleFunc("POST /api/characters/{id}/interrupt", s.handleInterrupt)
	s.mux.HandleFunc("POST /api/characters/{id}/stop", s.handleStop)
	s.mux.HandleFunc("POST /api/characters/{id}/archive", s.handleArchive)
	s.mux.HandleFunc("POST /api/characters/{id}/unarchive", s.handleUnarchive)
	s.mux.HandleFunc("POST /api/characters/{id}/dismiss", s.handleDismiss)
	s.mux.HandleFunc("POST /api/characters/{id}/read", s.handleMarkRead)
	s.mux.HandleFunc("GET /api/characters/{id}/events", s.handleEvents)
	s.mux.HandleFunc("GET /api/characters/{id}/stream", s.handleStream)
	s.mux.HandleFunc("/api/events/stream", s.hub.Handle)

	s.mux.HandleFunc("POST /api/pick-directory", s.handlePickDirectory)
	s.mux.HandleFunc("GET /api/branches", s.handleListBranches)
	s.mux.HandleFunc("GET /api/cursor-classes", s.handleCursorClasses)

	s.mux.Handle("/", http.FileServer(http.FS(webFS)))
}

// characterView is what the API actually serves for a character: its
// persisted fields plus presence, computed fresh every time from whether
// its pilot-runner control socket answers — never a stored value.
type characterView struct {
	model.Character
	Presence string `json:"presence"`
}

func (s *Server) view(ch model.Character) characterView {
	presence := "asleep"
	if s.pilots.IsAwake(ch.ID) {
		presence = "awake"
	}
	return characterView{Character: ch, Presence: presence}
}

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	characters, err := s.store.ListCharacters(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	views := make([]characterView, len(characters))
	for i, ch := range characters {
		views[i] = s.view(ch)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(views); err != nil {
		log.Printf("encode characters response: %v", err)
	}
}

// handleCreateCharacter recruits a new character: validates its territory
// request, then hands off to internal/pilot to prepare that territory and
// actually spawn the race's process.
func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Race          string `json:"race"`
		TerritoryMode string `json:"territory_mode"`
		Cwd           string `json:"cwd"`
		Branch        string `json:"branch"`
		Class         string `json:"class"`
		Prompt        string `json:"prompt"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxHookBodyBytes)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	race := model.Race(body.Race)
	if race != model.RaceClaudeCode && race != model.RaceCursor {
		http.Error(w, "race must be claude-code or cursor", http.StatusBadRequest)
		return
	}
	mode := model.TerritoryMode(body.TerritoryMode)
	if mode == "" {
		mode = model.TerritoryOwn
	}
	if mode != model.TerritoryOwn && mode != model.TerritoryShared {
		http.Error(w, "territory_mode must be own or shared", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(body.Cwd)
	if err != nil || !info.IsDir() {
		http.Error(w, "territory must be an existing directory", http.StatusBadRequest)
		return
	}
	if mode == model.TerritoryOwn && !territory.IsGitRepo(body.Cwd) {
		http.Error(w, body.Cwd+" is not a git repository — only shared territory is available for it", http.StatusBadRequest)
		return
	}

	ch, err := s.pilots.Create(r.Context(), pilot.CreateSpec{
		Race: race, TerritoryMode: mode, Cwd: body.Cwd, Branch: body.Branch,
		Class: model.Class(body.Class), Prompt: body.Prompt,
	})
	if err != nil {
		log.Printf("create character: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s.view(ch))
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxHookBodyBytes)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	err := s.pilots.SendMessage(r.Context(), r.PathValue("id"), body.Text)
	writeActionResult(w, err)
}

func (s *Server) handleInterrupt(w http.ResponseWriter, r *http.Request) {
	err := s.pilots.Interrupt(r.Context(), r.PathValue("id"))
	writeActionResult(w, err)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	err := s.pilots.Stop(r.Context(), r.PathValue("id"))
	writeActionResult(w, err)
}

// handleArchive stops the character's process regardless of activity, keeps
// its transcript and territory, and hides it from camp.
func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	ch, err := s.pilots.Archive(r.Context(), r.PathValue("id"))
	if err != nil {
		writeActionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.view(ch))
}

// handleUnarchive reverses handleArchive. No activity restriction: always
// allowed so nothing can get permanently stuck out of view.
func (s *Server) handleUnarchive(w http.ResponseWriter, r *http.Request) {
	ch, err := s.pilots.Unarchive(r.Context(), r.PathValue("id"))
	if err != nil {
		writeActionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.view(ch))
}

// handleDismiss removes a character for good. worktree_left_at is non-empty
// when an own-territory worktree had uncommitted changes and was left in
// place rather than discarded.
func (s *Server) handleDismiss(w http.ResponseWriter, r *http.Request) {
	leftAt, err := s.pilots.Dismiss(r.Context(), r.PathValue("id"))
	if err != nil {
		writeActionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"worktree_left_at": leftAt})
}

// handleMarkRead clears a character's unread mark — called by the interface
// when the user actually reads its transcript, the only thing that clears
// it.
func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	err := s.pilots.MarkRead(r.Context(), r.PathValue("id"))
	writeActionResult(w, err)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.pilots.Events(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	s.pilots.Hub(r.PathValue("id")).Handle(w, r)
}

// writeActionResult maps a pilot.Manager error to the right HTTP status for
// an action with no other response body.
func writeActionResult(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeActionError(w, err)
}

// writeActionError maps a pilot.Manager error to the right HTTP status:
// unknown character or no live process to act on are client-correctable,
// not server failures.
func writeActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pilot.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, pilot.ErrNotRunning):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("character action error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

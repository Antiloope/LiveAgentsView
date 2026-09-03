// Package daemon wires the HTTP server: the JSON API and SSE stream for the
// dashboard, piloted-session control, and serving the embedded frontend.
// Binds to 127.0.0.1 only: exposing it would let anything on the network
// approve agent permissions or launch processes on this machine.
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
	"os/exec"
	"strings"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilot"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
)

const maxHookBodyBytes = 1 << 20 // 1 MiB is generous for a piloted-session request body

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
// needs them to spawn and reconnect to piloted sessions' detached
// pilot-runner processes. port is the one this daemon is actually listening
// on (LAV_PORT can change it from the default), used to compute the
// loopback Host/Origin values secure() accepts.
func New(st *store.Store, cls classifier.Classifier, webFS fs.FS, lavHome, selfExe, port string) *Server {
	s := &Server{
		store: st,
		hub:   sse.NewHub(),
		mux:   http.NewServeMux(),
	}
	s.pilots = pilot.NewManager(st, cls, lavHome, selfExe, func(sess model.Session) { s.hub.Broadcast(sess) })
	if err := s.pilots.ReconcileOnStartup(context.Background()); err != nil {
		log.Printf("reconcile piloted sessions on startup: %v", err)
	}
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

	s.mux.HandleFunc("/api/sessions", s.handleListSessions)
	s.mux.HandleFunc("POST /api/sessions/{id}/archive", s.handleArchiveSession)
	s.mux.HandleFunc("POST /api/sessions/{id}/unarchive", s.handleUnarchiveSession)
	s.mux.HandleFunc("/api/events/stream", s.hub.Handle)

	s.mux.HandleFunc("POST /api/pick-directory", s.handlePickDirectory)
	s.mux.HandleFunc("GET /api/branches", s.handleListBranches)
	s.mux.HandleFunc("GET /api/cursor-models", s.handleCursorModels)

	s.mux.HandleFunc("POST /api/piloted/sessions", s.handleLaunchPiloted)
	s.mux.HandleFunc("POST /api/piloted/sessions/{id}/message", s.handlePilotedMessage)
	s.mux.HandleFunc("POST /api/piloted/sessions/{id}/permission", s.handlePilotedPermission)
	s.mux.HandleFunc("POST /api/piloted/sessions/{id}/interrupt", s.handlePilotedInterrupt)
	s.mux.HandleFunc("POST /api/piloted/sessions/{id}/cancel", s.handlePilotedCancel)
	s.mux.HandleFunc("POST /api/piloted/sessions/{id}/resume", s.handlePilotedResume)
	s.mux.HandleFunc("GET /api/piloted/sessions/{id}/events", s.handlePilotedEvents)
	s.mux.HandleFunc("GET /api/piloted/sessions/{id}/stream", s.handlePilotedStream)

	s.mux.Handle("/", http.FileServer(http.FS(webFS)))
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.store.ListSessions(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sessions); err != nil {
		log.Printf("encode sessions response: %v", err)
	}
}

// handleArchiveSession hides a session from the dashboard's camp view. Not
// allowed while the session is actively working, so a live turn can't be
// hidden out from under itself — it never touches the underlying process
// either way, only the persisted archived flag.
func (s *Server) handleArchiveSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, found, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if sess.State == model.StateWorking {
		http.Error(w, "cannot archive a session that is working", http.StatusConflict)
		return
	}
	s.setArchived(w, r, id, true)
}

// handleUnarchiveSession reverses handleArchiveSession. No state
// restriction: unarchiving is always allowed so nothing can get permanently
// stuck out of view.
func (s *Server) handleUnarchiveSession(w http.ResponseWriter, r *http.Request) {
	s.setArchived(w, r, r.PathValue("id"), false)
}

func (s *Server) setArchived(w http.ResponseWriter, r *http.Request, id string, archived bool) {
	sess, found, err := s.store.SetArchived(r.Context(), id, archived)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	s.hub.Broadcast(sess)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// handleLaunchPiloted starts a new piloted session: validates the target
// directory exists, checks out an optional branch (refusing rather than
// discarding local changes — plain `git checkout`, never -f), and hands off
// to internal/pilot to actually spawn the provider's CLI.
func (s *Server) handleLaunchPiloted(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		Cwd      string `json:"cwd"`
		Branch   string `json:"branch"`
		Model    string `json:"model"`
		Prompt   string `json:"prompt"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxHookBodyBytes)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	provider := model.Provider(body.Provider)
	if provider != model.ProviderClaudeCode && provider != model.ProviderCursor {
		http.Error(w, "provider must be claude-code or cursor", http.StatusBadRequest)
		return
	}
	// Claude Code can start with nobody home — its process just sits
	// attached, waiting on stdin, the same as after any other turn. Cursor
	// cannot: every turn is its own one-shot `agent -p ... <prompt>`
	// invocation with no such thing as an idle process, so it needs
	// something to do from the very first launch.
	if provider == model.ProviderCursor && strings.TrimSpace(body.Prompt) == "" {
		http.Error(w, "prompt is required for cursor", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(body.Cwd)
	if err != nil || !info.IsDir() {
		http.Error(w, "cwd must be an existing directory", http.StatusBadRequest)
		return
	}

	if body.Branch != "" {
		if out, err := exec.Command("git", "-C", body.Cwd, "checkout", body.Branch).CombinedOutput(); err != nil {
			http.Error(w, "git checkout "+body.Branch+": "+strings.TrimSpace(string(out)), http.StatusBadRequest)
			return
		}
	}

	sess, err := s.pilots.Launch(r.Context(), pilot.LaunchSpec{
		Provider: provider,
		Cwd:      body.Cwd,
		Branch:   body.Branch,
		Model:    body.Model,
		Prompt:   body.Prompt,
	})
	if err != nil {
		log.Printf("launch piloted session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sess)
}

func (s *Server) handlePilotedMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxHookBodyBytes)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Text) == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	err := s.pilots.SendMessage(r.Context(), r.PathValue("id"), body.Text)
	writePilotActionResult(w, err)
}

func (s *Server) handlePilotedPermission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RequestID string `json:"request_id"`
		Approve   bool   `json:"approve"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.RequestID == "" {
		http.Error(w, "request_id is required", http.StatusBadRequest)
		return
	}
	err := s.pilots.ApprovePermission(r.Context(), r.PathValue("id"), body.RequestID, body.Approve)
	writePilotActionResult(w, err)
}

func (s *Server) handlePilotedInterrupt(w http.ResponseWriter, r *http.Request) {
	err := s.pilots.Interrupt(r.Context(), r.PathValue("id"))
	writePilotActionResult(w, err)
}

func (s *Server) handlePilotedCancel(w http.ResponseWriter, r *http.Request) {
	err := s.pilots.Cancel(r.Context(), r.PathValue("id"))
	writePilotActionResult(w, err)
}

func (s *Server) handlePilotedResume(w http.ResponseWriter, r *http.Request) {
	sess, err := s.pilots.Resume(r.Context(), r.PathValue("id"))
	if err != nil {
		writePilotActionResult(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

func (s *Server) handlePilotedEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.pilots.Events(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) handlePilotedStream(w http.ResponseWriter, r *http.Request) {
	s.pilots.Hub(r.PathValue("id")).Handle(w, r)
}

// writePilotActionResult maps a pilot.Manager error to the right HTTP
// status: unknown session, no live process to act on, or a Cursor turn
// already in progress are all client-correctable, not server failures.
func writePilotActionResult(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, pilot.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, pilot.ErrNotRunning), errors.Is(err, pilot.ErrTurnInProgress):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("piloted action error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Package daemon wires the HTTP server: hook ingestion endpoints per
// provider, the JSON API and SSE stream for the dashboard, piloted-session
// control, and serving the embedded frontend. Binds to 127.0.0.1 only:
// exposing it would let anything on the network approve agent permissions or
// launch processes on this machine.
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
	"path/filepath"
	"strings"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/ingest"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilot"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/sse"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/terminal"
)

const maxHookBodyBytes = 1 << 20 // 1 MiB is generous for a hook payload

type Server struct {
	store      *store.Store
	classifier classifier.Classifier
	hub        *sse.Hub
	pilots     *pilot.Manager
	mux        *http.ServeMux
}

func New(st *store.Store, cls classifier.Classifier, webFS fs.FS) *Server {
	s := &Server{
		store:      st,
		classifier: cls,
		hub:        sse.NewHub(),
		mux:        http.NewServeMux(),
	}
	s.pilots = pilot.NewManager(st, cls, func(sess model.Session) { s.hub.Broadcast(sess) })
	if err := s.pilots.ReconcileOnStartup(context.Background()); err != nil {
		log.Printf("reconcile piloted sessions on startup: %v", err)
	}
	s.routes(webFS)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes(webFS fs.FS) {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	s.mux.HandleFunc("/hooks/claude-code", s.handleHook(model.ProviderClaudeCode))
	s.mux.HandleFunc("/hooks/codex", s.handleHook(model.ProviderCodex))
	s.mux.HandleFunc("/hooks/cursor", s.handleHook(model.ProviderCursor))

	s.mux.HandleFunc("/api/sessions", s.handleListSessions)
	s.mux.HandleFunc("/api/events/stream", s.hub.Handle)
	s.mux.HandleFunc("/api/open-terminal", s.handleOpenTerminal)

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

// handleHook returns one handler per provider that: parses the raw hook
// body into a Signal, resolves ambiguous "turn ended" signals through the
// classifier, upserts the session, appends the raw event for history/audit,
// and broadcasts the new session state to any connected dashboard.
func (s *Server) handleHook(provider model.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxHookBodyBytes))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		event := r.URL.Query().Get("event")
		repo := r.URL.Query().Get("repo")
		branch := r.URL.Query().Get("branch")
		worktree := r.URL.Query().Get("worktree")

		var sig model.Signal
		switch provider {
		case model.ProviderClaudeCode:
			sig, err = ingest.ParseClaudeCode(event, body)
		case model.ProviderCodex:
			sig, err = ingest.ParseCodex(body)
		case model.ProviderCursor:
			sig, err = ingest.ParseCursor(event, body)
		}
		if err != nil {
			log.Printf("hook parse error (%s/%s): %v", provider, event, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if repo != "" {
			sig.Repo = repo
		}
		if sig.Repo == "" && sig.Cwd != "" {
			sig.Repo = filepath.Base(sig.Cwd)
		}
		if branch != "" {
			sig.Branch = branch
		}

		finalState := sig.State
		if finalState == "" {
			switch s.classifier.Classify(sig.LastMessage) {
			case classifier.VerdictWaiting:
				finalState = model.StateWaiting
			default:
				finalState = model.StateDone
			}
		}

		ctx := r.Context()
		now := time.Now().UTC()
		createdAt := now
		if existing, found, _ := s.store.GetSession(ctx, sig.SessionID); found {
			createdAt = existing.CreatedAt
			// A piloted session's own CLI process still fires its normal
			// lifecycle hooks against this same daemon (they are not aware
			// they were launched by LiveAgentsView) — those must not
			// downgrade a session already tracked at Driver fidelity back
			// to Hooks, discarding the richer piloted state and transcript.
			if existing.Fidelity == model.FidelityDriver {
				w.WriteHeader(http.StatusAccepted)
				return
			}
		}

		sess := model.Session{
			ID:          sig.SessionID,
			Provider:    provider,
			Fidelity:    model.FidelityHooks,
			Cwd:         sig.Cwd,
			Repo:        sig.Repo,
			Branch:      sig.Branch,
			Worktree:    worktree,
			State:       finalState,
			LastMessage: sig.LastMessage,
			CreatedAt:   createdAt,
			UpdatedAt:   now,
		}

		if err := s.store.UpsertSession(ctx, sess); err != nil {
			log.Printf("store session: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := s.store.AppendEvent(ctx, sess.ID, provider, event, finalState, string(body)); err != nil {
			log.Printf("store event: %v", err)
		}

		s.hub.Broadcast(sess)
		w.WriteHeader(http.StatusAccepted)
	}
}

// handleOpenTerminal spawns a terminal window at the given cwd. Only works
// when this process runs natively on the host — see internal/terminal.
func (s *Server) handleOpenTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Cwd string `json:"cwd"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Cwd == "" {
		http.Error(w, "cwd is required", http.StatusBadRequest)
		return
	}
	if err := terminal.Open(body.Cwd); err != nil {
		log.Printf("open terminal at %s: %v", body.Cwd, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// handleLaunchPiloted starts a new piloted session: validates the target
// directory exists, checks out an optional branch (refusing rather than
// discarding local changes — plain `git checkout`, never -f), and hands off
// to internal/pilot to actually spawn the provider's CLI.
func (s *Server) handleLaunchPiloted(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		Cwd      string `json:"cwd"`
		Branch   string `json:"branch"`
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
	if strings.TrimSpace(body.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
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

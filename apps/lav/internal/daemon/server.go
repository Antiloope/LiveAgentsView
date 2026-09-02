// Package daemon wires the HTTP server: hook ingestion endpoints per
// provider, the JSON API and SSE stream for the dashboard, and serving the
// embedded frontend. Binds to 127.0.0.1 only — see docs/02-scope.md
// "Explicit boundaries".
package daemon

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/ingest"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
)

const maxHookBodyBytes = 1 << 20 // 1 MiB is generous for a hook payload

type Server struct {
	store      *store.Store
	classifier classifier.Classifier
	hub        *sseHub
	mux        *http.ServeMux
}

func New(st *store.Store, cls classifier.Classifier, webFS fs.FS) *Server {
	s := &Server{
		store:      st,
		classifier: cls,
		hub:        newSSEHub(),
		mux:        http.NewServeMux(),
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
	s.mux.HandleFunc("/api/events/stream", s.hub.handle)

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

		s.hub.broadcast(sess)
		w.WriteHeader(http.StatusAccepted)
	}
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

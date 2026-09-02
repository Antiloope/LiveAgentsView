// Package store persists sessions and their event history to SQLite so the
// daemon survives restarts without losing state.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}
	// SQLite is single-writer; keep it simple rather than tune WAL/pragmas.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	fidelity TEXT NOT NULL,
	cwd TEXT NOT NULL DEFAULT '',
	repo TEXT NOT NULL DEFAULT '',
	branch TEXT NOT NULL DEFAULT '',
	worktree TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL,
	last_message TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	hook_event TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT '',
	raw TEXT NOT NULL DEFAULT '',
	received_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id, received_at);
`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// UpsertSession creates or updates a session. Fields that arrive empty on an
// update (Repo/Branch/Worktree/LastMessage — not every hook payload carries
// all of them) do not clobber a previously known value.
func (s *Store) UpsertSession(ctx context.Context, sess model.Session) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, provider, fidelity, cwd, repo, branch, worktree, state, last_message, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	provider = excluded.provider,
	fidelity = excluded.fidelity,
	cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE sessions.cwd END,
	repo = CASE WHEN excluded.repo != '' THEN excluded.repo ELSE sessions.repo END,
	branch = CASE WHEN excluded.branch != '' THEN excluded.branch ELSE sessions.branch END,
	worktree = CASE WHEN excluded.worktree != '' THEN excluded.worktree ELSE sessions.worktree END,
	state = excluded.state,
	last_message = CASE WHEN excluded.last_message != '' THEN excluded.last_message ELSE sessions.last_message END,
	updated_at = excluded.updated_at
`,
		sess.ID, string(sess.Provider), string(sess.Fidelity), sess.Cwd, sess.Repo, sess.Branch, sess.Worktree,
		string(sess.State), sess.LastMessage, sess.CreatedAt, sess.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert session %s: %w", sess.ID, err)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, id string) (model.Session, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, provider, fidelity, cwd, repo, branch, worktree, state, last_message, created_at, updated_at
FROM sessions WHERE id = ?`, id)

	sess, err := scanSession(row)
	if err == sql.ErrNoRows {
		return model.Session{}, false, nil
	}
	if err != nil {
		return model.Session{}, false, fmt.Errorf("get session %s: %w", id, err)
	}
	return sess, true, nil
}

func (s *Store) ListSessions(ctx context.Context) ([]model.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, provider, fidelity, cwd, repo, branch, worktree, state, last_message, created_at, updated_at
FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	out := []model.Session{}
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *Store) AppendEvent(ctx context.Context, sessionID string, provider model.Provider, hookEvent string, state model.State, raw string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO events (session_id, provider, hook_event, state, raw, received_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, string(provider), hookEvent, string(state), raw, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("append event for session %s: %w", sessionID, err)
	}
	return nil
}

// ListEvents returns a session's events oldest first — the shape a
// transcript replays in, unlike ListSessions's most-recently-updated order.
func (s *Store) ListEvents(ctx context.Context, sessionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT raw FROM events WHERE session_id = ? ORDER BY received_at ASC, id ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list events for session %s: %w", sessionID, err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan event for session %s: %w", sessionID, err)
		}
		out = append(out, raw)
	}
	return out, rows.Err()
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (model.Session, error) {
	var sess model.Session
	var provider, fidelity, state string
	err := row.Scan(
		&sess.ID, &provider, &fidelity, &sess.Cwd, &sess.Repo, &sess.Branch, &sess.Worktree,
		&state, &sess.LastMessage, &sess.CreatedAt, &sess.UpdatedAt,
	)
	if err != nil {
		return model.Session{}, err
	}
	sess.Provider = model.Provider(provider)
	sess.Fidelity = model.Fidelity(fidelity)
	sess.State = model.State(state)
	return sess, nil
}

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
	model TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL,
	last_message TEXT NOT NULL DEFAULT '',
	archived INTEGER NOT NULL DEFAULT 0,
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
	if err := s.ensureColumn(ctx, "archived", `ALTER TABLE sessions ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "model", `ALTER TABLE sessions ADD COLUMN model TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	return s.purgeNonDriverSessions(ctx)
}

// ensureColumn adds a sessions column via alterSQL for a database that
// predates it — CREATE TABLE IF NOT EXISTS above only covers a brand-new
// database. Safe to run on every startup: a no-op once a database already
// has the column.
func (s *Store) ensureColumn(ctx context.Context, name, alterSQL string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(sessions)`)
	if err != nil {
		return fmt.Errorf("inspect sessions columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid, notnull, pk int
			colName, colType string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan sessions column: %w", err)
		}
		if colName == name {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect sessions columns: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, alterSQL); err != nil {
		return fmt.Errorf("add sessions.%s column: %w", name, err)
	}
	return nil
}

// purgeNonDriverSessions removes hooks/tailing-fidelity rows left over from
// before piloted-only mode — adopted sessions are no longer a concept this
// app tracks. Safe to run on every startup: a no-op once the first run after
// upgrading has cleared them.
func (s *Store) purgeNonDriverSessions(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
DELETE FROM events WHERE session_id IN (SELECT id FROM sessions WHERE fidelity != ?)`,
		string(model.FidelityDriver)); err != nil {
		return fmt.Errorf("purge non-driver events: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE fidelity != ?`, string(model.FidelityDriver)); err != nil {
		return fmt.Errorf("purge non-driver sessions: %w", err)
	}
	return nil
}

// UpsertSession creates or updates a session. Fields that arrive empty on an
// update (Repo/Branch/Worktree/LastMessage — not every hook payload carries
// all of them) do not clobber a previously known value. Archived is
// deliberately absent from the UPDATE branch: every caller of this method
// (pilot.Manager.upsert on every state change, ReconcileOnStartup) builds
// its model.Session from scratch with Archived left at its Go zero value, so
// touching that column here would silently unarchive a session the moment
// its process next reports state — only SetArchived is allowed to change it.
func (s *Store) UpsertSession(ctx context.Context, sess model.Session) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, provider, fidelity, cwd, repo, branch, worktree, model, state, last_message, archived, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	provider = excluded.provider,
	fidelity = excluded.fidelity,
	cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE sessions.cwd END,
	repo = CASE WHEN excluded.repo != '' THEN excluded.repo ELSE sessions.repo END,
	branch = CASE WHEN excluded.branch != '' THEN excluded.branch ELSE sessions.branch END,
	worktree = CASE WHEN excluded.worktree != '' THEN excluded.worktree ELSE sessions.worktree END,
	model = CASE WHEN excluded.model != '' THEN excluded.model ELSE sessions.model END,
	state = excluded.state,
	last_message = CASE WHEN excluded.last_message != '' THEN excluded.last_message ELSE sessions.last_message END,
	updated_at = excluded.updated_at
`,
		sess.ID, string(sess.Provider), string(sess.Fidelity), sess.Cwd, sess.Repo, sess.Branch, sess.Worktree, sess.Model,
		string(sess.State), sess.LastMessage, sess.Archived, sess.CreatedAt, sess.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert session %s: %w", sess.ID, err)
	}
	return nil
}

// SetArchived is the only path allowed to change a session's archived flag —
// see UpsertSession's doc comment for why that method itself must never
// touch it. Returns false if no session with this id exists.
func (s *Store) SetArchived(ctx context.Context, id string, archived bool) (model.Session, bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE sessions SET archived = ?, updated_at = ? WHERE id = ?`,
		archived, time.Now().UTC(), id)
	if err != nil {
		return model.Session{}, false, fmt.Errorf("set archived for session %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Session{}, false, nil
	}
	return s.GetSession(ctx, id)
}

func (s *Store) GetSession(ctx context.Context, id string) (model.Session, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, provider, fidelity, cwd, repo, branch, worktree, model, state, last_message, archived, created_at, updated_at
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
SELECT id, provider, fidelity, cwd, repo, branch, worktree, model, state, last_message, archived, created_at, updated_at
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
		&sess.ID, &provider, &fidelity, &sess.Cwd, &sess.Repo, &sess.Branch, &sess.Worktree, &sess.Model,
		&state, &sess.LastMessage, &sess.Archived, &sess.CreatedAt, &sess.UpdatedAt,
	)
	if err != nil {
		return model.Session{}, err
	}
	sess.Provider = model.Provider(provider)
	sess.Fidelity = model.Fidelity(fidelity)
	sess.State = model.State(state)
	return sess, nil
}

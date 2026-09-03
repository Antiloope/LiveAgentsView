// Package store persists characters and their event history to SQLite so
// the daemon survives restarts without losing state.
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

const createCharactersSQL = `
CREATE TABLE IF NOT EXISTS characters (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL DEFAULT '',
	race TEXT NOT NULL,
	class TEXT NOT NULL DEFAULT '',
	activity TEXT NOT NULL,
	unread INTEGER NOT NULL DEFAULT 0,
	territory_mode TEXT NOT NULL DEFAULT '',
	territory_path TEXT NOT NULL DEFAULT '',
	territory_source TEXT NOT NULL DEFAULT '',
	territory_branch TEXT NOT NULL DEFAULT '',
	repo TEXT NOT NULL DEFAULT '',
	archived INTEGER NOT NULL DEFAULT 0,
	last_message TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
)`

const createEventsSQL = `
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	character_id TEXT NOT NULL,
	race TEXT NOT NULL,
	hook_event TEXT NOT NULL DEFAULT '',
	activity TEXT NOT NULL DEFAULT '',
	raw TEXT NOT NULL DEFAULT '',
	received_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_character ON events(character_id, received_at)`

// migrate brings a database to the current "characters" schema. A brand-new
// database just gets the tables created. A database still on the old
// "sessions" schema (provider/model/cwd/state) is migrated in place: renamed
// out of the way, the new tables created under the final names, its rows
// copied across with the field remapping the character-model redesign
// decided (provider→race, model→class, cwd→territory with mode "shared",
// state→activity with done→ready+unread and blocked→waiting), then dropped
// — all inside one transaction, so a database already on the new schema
// (checked first) makes this whole function a no-op and a crash mid-migration
// never leaves a half-renamed database.
func (s *Store) migrate(ctx context.Context) error {
	hasCharacters, err := s.tableExists(ctx, "characters")
	if err != nil {
		return fmt.Errorf("check schema: %w", err)
	}
	if hasCharacters {
		return nil
	}
	hasSessions, err := s.tableExists(ctx, "sessions")
	if err != nil {
		return fmt.Errorf("check schema: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer tx.Rollback()

	if hasSessions {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE sessions RENAME TO sessions_old`); err != nil {
			return fmt.Errorf("rename sessions table: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `ALTER TABLE events RENAME TO events_old`); err != nil {
			return fmt.Errorf("rename events table: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, createCharactersSQL); err != nil {
		return fmt.Errorf("create characters table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, createEventsSQL); err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	if hasSessions {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO characters (id, session_id, race, class, activity, unread, territory_mode, territory_path, territory_source, territory_branch, repo, archived, last_message, created_at, updated_at)
SELECT
	id, id, provider, model,
	CASE state
		WHEN 'working' THEN 'working'
		WHEN 'waiting' THEN 'waiting'
		WHEN 'blocked' THEN 'waiting'
		WHEN 'done' THEN 'ready'
		WHEN 'idle' THEN 'ready'
		WHEN 'failed' THEN 'failed'
		ELSE 'ready'
	END,
	CASE WHEN state = 'done' THEN 1 ELSE 0 END,
	'shared', cwd, cwd, branch, repo, archived, last_message, created_at, updated_at
FROM sessions_old`); err != nil {
			return fmt.Errorf("copy sessions into characters: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO events (character_id, race, hook_event, activity, raw, received_at)
SELECT session_id, provider, hook_event, state, raw, received_at FROM events_old`); err != nil {
			return fmt.Errorf("copy events: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DROP TABLE sessions_old`); err != nil {
			return fmt.Errorf("drop sessions_old: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DROP TABLE events_old`); err != nil {
			return fmt.Errorf("drop events_old: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) tableExists(ctx context.Context, name string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// UpsertCharacter creates or updates a character. Fields that arrive empty
// on an update (SessionID/Class/Territory/Repo/LastMessage — not every
// caller knows all of them at every point) do not clobber a previously
// known value. Archived and Unread are deliberately absent from the UPDATE
// branch — SetArchived and SetUnread are the only paths allowed to change
// them, so a routine activity upsert can never silently flip either back.
func (s *Store) UpsertCharacter(ctx context.Context, ch model.Character) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO characters (id, session_id, race, class, activity, unread, territory_mode, territory_path, territory_source, territory_branch, repo, archived, last_message, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	session_id = CASE WHEN excluded.session_id != '' THEN excluded.session_id ELSE characters.session_id END,
	race = excluded.race,
	class = CASE WHEN excluded.class != '' THEN excluded.class ELSE characters.class END,
	activity = excluded.activity,
	territory_mode = CASE WHEN excluded.territory_mode != '' THEN excluded.territory_mode ELSE characters.territory_mode END,
	territory_path = CASE WHEN excluded.territory_path != '' THEN excluded.territory_path ELSE characters.territory_path END,
	territory_source = CASE WHEN excluded.territory_source != '' THEN excluded.territory_source ELSE characters.territory_source END,
	territory_branch = CASE WHEN excluded.territory_branch != '' THEN excluded.territory_branch ELSE characters.territory_branch END,
	repo = CASE WHEN excluded.repo != '' THEN excluded.repo ELSE characters.repo END,
	last_message = CASE WHEN excluded.last_message != '' THEN excluded.last_message ELSE characters.last_message END,
	updated_at = excluded.updated_at
`,
		ch.ID, ch.SessionID, string(ch.Race), string(ch.Class), string(ch.Activity), ch.Unread,
		string(ch.Territory.Mode), ch.Territory.Path, ch.Territory.Source, ch.Territory.Branch,
		ch.Repo, ch.Archived, ch.LastMessage, ch.CreatedAt, ch.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert character %s: %w", ch.ID, err)
	}
	return nil
}

// SetArchived is the only path allowed to change a character's archived
// flag — see UpsertCharacter's doc comment for why. Returns false if no
// character with this id exists.
func (s *Store) SetArchived(ctx context.Context, id string, archived bool) (model.Character, bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE characters SET archived = ?, updated_at = ? WHERE id = ?`,
		archived, time.Now().UTC(), id)
	if err != nil {
		return model.Character{}, false, fmt.Errorf("set archived for character %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Character{}, false, nil
	}
	return s.GetCharacter(ctx, id)
}

// SetUnread is the only path allowed to change a character's unread mark —
// set when a quest ends without a question, cleared by the interface's
// explicit "read" call, never by any other activity change. Returns false
// if no character with this id exists.
func (s *Store) SetUnread(ctx context.Context, id string, unread bool) (model.Character, bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE characters SET unread = ?, updated_at = ? WHERE id = ?`,
		unread, time.Now().UTC(), id)
	if err != nil {
		return model.Character{}, false, fmt.Errorf("set unread for character %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Character{}, false, nil
	}
	return s.GetCharacter(ctx, id)
}

// SetSessionID records the provider's own conversation id the first time it
// is reported — a no-op once already set, so it is never overwritten.
func (s *Store) SetSessionID(ctx context.Context, id, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE characters SET session_id = ?, updated_at = ? WHERE id = ? AND session_id = ''`,
		sessionID, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("set session id for character %s: %w", id, err)
	}
	return nil
}

// DeleteCharacter removes a character's row and its events — the
// persistence half of dismissing it. Returns false if no character with
// this id exists.
func (s *Store) DeleteCharacter(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM characters WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete character %s: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE character_id = ?`, id); err != nil {
		return true, fmt.Errorf("delete events for character %s: %w", id, err)
	}
	return true, nil
}

func (s *Store) GetCharacter(ctx context.Context, id string) (model.Character, bool, error) {
	row := s.db.QueryRowContext(ctx, characterSelect+`WHERE id = ?`, id)
	ch, err := scanCharacter(row)
	if err == sql.ErrNoRows {
		return model.Character{}, false, nil
	}
	if err != nil {
		return model.Character{}, false, fmt.Errorf("get character %s: %w", id, err)
	}
	return ch, true, nil
}

func (s *Store) ListCharacters(ctx context.Context) ([]model.Character, error) {
	rows, err := s.db.QueryContext(ctx, characterSelect+`ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	defer rows.Close()

	out := []model.Character{}
	for rows.Next() {
		ch, err := scanCharacter(rows)
		if err != nil {
			return nil, fmt.Errorf("scan character: %w", err)
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) AppendEvent(ctx context.Context, characterID string, race model.Race, hookEvent string, activity model.Activity, raw string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO events (character_id, race, hook_event, activity, raw, received_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		characterID, string(race), hookEvent, string(activity), raw, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("append event for character %s: %w", characterID, err)
	}
	return nil
}

// ListEvents returns a character's events oldest first — the shape a
// transcript replays in, unlike ListCharacters's most-recently-updated
// order.
func (s *Store) ListEvents(ctx context.Context, characterID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT raw FROM events WHERE character_id = ? ORDER BY received_at ASC, id ASC`, characterID)
	if err != nil {
		return nil, fmt.Errorf("list events for character %s: %w", characterID, err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan event for character %s: %w", characterID, err)
		}
		out = append(out, raw)
	}
	return out, rows.Err()
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

const characterSelect = `
SELECT id, session_id, race, class, activity, unread, territory_mode, territory_path, territory_source, territory_branch, repo, archived, last_message, created_at, updated_at
FROM characters `

func scanCharacter(row rowScanner) (model.Character, error) {
	var ch model.Character
	var race, class, activity, territoryMode string
	err := row.Scan(
		&ch.ID, &ch.SessionID, &race, &class, &activity, &ch.Unread,
		&territoryMode, &ch.Territory.Path, &ch.Territory.Source, &ch.Territory.Branch,
		&ch.Repo, &ch.Archived, &ch.LastMessage, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return model.Character{}, err
	}
	ch.Race = model.Race(race)
	ch.Class = model.Class(class)
	ch.Activity = model.Activity(activity)
	ch.Territory.Mode = model.TerritoryMode(territoryMode)
	return ch, nil
}

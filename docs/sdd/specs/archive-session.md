---
title: Archive a session — hide it from the camp view, reversibly
slug: archive-session
status: validated
created: 2026-09-02
updated: 2026-09-02
next: none
chain: specify+implement+validate
---

# Spec: Archive a session — hide it from the camp view, reversibly

## Intent

A session can be archived from the dashboard so it stops appearing in the camp/quest
view. Archiving is persisted in SQLite (survives a `lav` restart, consistent across any
client pointed at the same daemon), reversible from a dedicated archived view, and
allowed in any state except `working`.

## Out of scope

- **Automatic/rule-based archiving** (e.g. auto-archive N days after `done`). Manual only
  — no scheduler, no TTL.
- **Permanent deletion.** Archiving flips a flag on the existing row; it never deletes a
  session's row or its event history. A real delete/purge action is a separate, unagreed
  feature.
- **Any change to attention-queue priority or notification behavior** beyond archived
  sessions no longer appearing in the camp/quest view they already don't leave today.
  [IDEA-01](../../05-ideas-to-discuss.md)'s taxonomy is untouched.
- **An archive control on the `PartyStand` card itself.** The card
  (`apps/web/src/PartyStand.tsx`) is already a single `<button>` for selecting a session;
  nesting another interactive control inside it is invalid HTML. The archive/unarchive
  action lives in the session detail drawer (`apps/web/src/SessionDrawer.tsx`) instead,
  next to the existing Interrupt/Cancel/Resume actions.
- **Bulk archive/unarchive.** One session at a time, same as every other pilot action.
- **Any side effect on the underlying process.** Archiving never cancels, interrupts, or
  otherwise touches a session's live process — see Acceptance.

## Already decided

- [03-decisions.md](../../03-decisions.md) 2026-09-02 "Sessions can be archived,
  reversibly, in any non-working state" — persistence, reversibility, and eligibility
  are settled there; this spec is the implementation of that decision.
- [02-scope.md](../../02-scope.md) "What it does" — the archive bullet added alongside
  this spec.
- "Shows recently finished agents" and "Persists enough state and history to survive
  restarts" ([02-scope.md](../../02-scope.md)) — archiving is additive to both: finished
  agents still show by default (nothing here changes that default), and the archived flag
  persists using the same SQLite-backed session row every other piece of session state
  already uses ([03-decisions.md](../../03-decisions.md) 2026-09-01 "Stack: Go, SQLite,
  single self-contained binary").
- The canonical states stay WORKING/WAITING/BLOCKED/DONE/FAILED/IDLE
  ([03-decisions.md](../../03-decisions.md) 2026-09-01 "Canonical event/state model").
  This spec adds no new state — `archived` is an orthogonal boolean on the session row,
  not a seventh state, so a session's `state` field keeps meaning exactly what it means
  today (what the classifier/provider produced) independent of whether it is archived.

## Open questions

None — the three decisions above (persistence, reversibility, eligibility) close every
question that came up during triage.

## Acceptance

### Backend: model, storage, migration

- [ ] `model.Session` ([apps/lav/internal/model/model.go](../../../apps/lav/internal/model/model.go))
      gains `Archived bool \`json:"archived"\``.
- [ ] The `sessions` table gains an `archived` column, added idempotently for this real
      machine's existing SQLite file (which already has pre-existing driver-fidelity rows
      from earlier specs) — not only for a fresh `CREATE TABLE`. Verified by running the
      updated binary against a copy of this machine's real `~/.liveagentsview/lav.db` and
      confirming the existing rows survive with `archived = 0` and no migration error on a
      second run (idempotent).
- [ ] `Store.UpsertSession` ([apps/lav/internal/store/store.go](../../../apps/lav/internal/store/store.go))
      never resets `archived` back to its zero value. Concretely: `pilot.Manager.upsert`
      ([apps/lav/internal/pilot/pilot.go](../../../apps/lav/internal/pilot/pilot.go))
      builds a fresh `model.Session{}` on every state change and always calls
      `UpsertSession` with `Archived` unset (Go zero value `false`) — an archived
      session's later state changes (e.g. a background process finishing after it was
      archived) must not silently unarchive it. Verified live against a real running
      session: archive it, trigger a state-changing upsert on it (e.g. `cancel`, which
      calls `UpsertSession` with a struct whose `Archived` is unset, matching
      `Manager.upsert`'s real call shape), confirm the row's `archived` value is still
      `true` afterward. This repo has no committed Go test suite (matches every other
      spec's live-verification-only convention) — the validate step is free to write and
      delete a throwaway test to double-check this in isolation, but no test artifact is
      expected to remain in the repo.
- [ ] A new store method sets `archived` directly (the only code path allowed to change
      it), independent of `UpsertSession`.

### Backend: HTTP API

- [ ] `POST /api/sessions/{id}/archive`: sets `archived = true`, broadcasts the updated
      session on the same SSE hub `/api/events/stream` already uses
      (`daemon.Server.hub`, [apps/lav/internal/daemon/server.go](../../../apps/lav/internal/daemon/server.go)),
      responds `200` with the updated session JSON. `404` for an unknown id. `409` if the
      session's current state is `working` (matching the existing
      `writePilotActionResult` conflict-status convention for
      `pilot.ErrTurnInProgress`).
- [ ] `POST /api/sessions/{id}/unarchive`: sets `archived = false`, broadcasts, responds
      `200` with the updated session JSON. `404` for an unknown id. No state restriction —
      always allowed if the session exists, regardless of current `state`.
- [ ] Archiving or unarchiving a session does not call into `pilot.Manager` at all — no
      cancel, interrupt, or process signal of any kind. Verified: archiving a session
      whose process is genuinely still alive (e.g. `waiting` or `blocked`, connected via
      its pilot-runner socket) leaves that process running and reachable exactly as
      before — confirmed by sending it a message immediately after unarchiving and
      getting a normal reply, with no relaunch/Resume needed.
- [ ] `pilot.Manager.ReconcileOnStartup` ([pilot.go](../../../apps/lav/internal/pilot/pilot.go))
      is unaffected by `archived` — an archived session with a still-live detached process
      reconnects on daemon startup exactly like a non-archived one (it reads and writes
      back the full stored row, which already round-trips `archived` correctly; this item
      confirms no new filtering was added there that would break it).

### Frontend

- [ ] `Session` ([apps/web/src/types.ts](../../../apps/web/src/types.ts)) gains
      `archived: boolean`; `fetchSessions`/`subscribeToSessions`
      ([apps/web/src/api.ts](../../../apps/web/src/api.ts)) carry it through unchanged
      (no new endpoint needed for the initial list or live updates — the existing
      `/api/sessions` and `/api/events/stream` already carry every field on `Session`).
- [ ] `App.tsx`'s `questSessions`/`urgentCamp`/`calmCamp` derivation excludes any session
      with `archived === true` — an archived session never renders as a `QuestToken` or
      `PartyStand`, including one that arrives live over SSE while the dashboard is open
      (not just on initial load).
- [ ] The "N sessions known" count in the header reflects only non-archived sessions.
- [ ] The session detail drawer (`SessionDrawer.tsx`'s `PilotChat`) shows an "Archive"
      action when `session.state !== 'working'` and `!session.archived`, alongside the
      existing Interrupt/Cancel/Resume actions. Clicking it calls the archive endpoint and
      the session disappears from the camp/quest view without a page reload.
- [ ] A discoverable control (a header button, following the existing "+ Recruit session"
      pattern) opens an "Archived sessions" view listing every archived session (at least
      provider, repo/cwd, state, last message) with an "Unarchive" action per row.
      Unarchiving there makes the session reappear in the normal camp/quest view without a
      page reload, driven by the same SSE broadcast the backend already sends.
- [ ] `tsc --noEmit` and `vite build` pass clean.

### Cross-cutting

- [ ] `go build ./...` and `go vet ./...` (via Docker, `golang:1.25-alpine`, matching this
      repo's "only Docker installed on the machine" rule) pass clean.

## How

**Backend.** `model.Session` ([model.go](../../../apps/lav/internal/model/model.go)) gains
`Archived bool \`json:"archived"\``. `sessions` gains an `archived INTEGER NOT NULL
DEFAULT 0` column in the `CREATE TABLE` for a fresh database, plus a new
`Store.ensureArchivedColumn` ([store.go](../../../apps/lav/internal/store/store.go)),
called from `migrate()` right after the `CREATE TABLE IF NOT EXISTS` block and before
`purgeNonDriverSessions`: it reads `PRAGMA table_info(sessions)` and only runs `ALTER
TABLE sessions ADD COLUMN archived ...` if the column isn't already there — idempotent,
covers this real machine's pre-existing database. `Store.UpsertSession`'s `INSERT`
column list now includes `archived`, but its `ON CONFLICT DO UPDATE SET` clause
deliberately does **not** list `archived` at all (not even a `CASE`-guarded no-op) — every
caller of `UpsertSession` (`pilot.Manager.upsert` on every state change,
`ReconcileOnStartup`'s idle-fallback) either builds a fresh `model.Session{}` with
`Archived` at its Go zero value or round-trips a value already read from the row, so
leaving the column out of the UPDATE branch is what stops a later state change from ever
resetting an archived session back to visible. A new `Store.SetArchived(ctx, id,
archived bool) (model.Session, bool, error)` does a direct `UPDATE ... SET archived = ?`
and is the only method allowed to change the column; `GetSession`/`ListSessions`/
`scanSession` select and scan it like every other field.

`daemon.Server` ([server.go](../../../apps/lav/internal/daemon/server.go)) gains `POST
/api/sessions/{id}/archive` and `POST /api/sessions/{id}/unarchive`, both routed before
the pilot-specific `/api/piloted/...` routes since they act on the session row directly,
never on `pilot.Manager` — archiving/unarchiving has no process side effect by
construction, not just by omission. `handleArchiveSession` fetches the session, 404s if
unknown, 409s (`"cannot archive a session that is working"`) if `state == StateWorking`,
otherwise delegates to a shared `setArchived` helper that calls `Store.SetArchived`,
broadcasts the updated session on `s.hub` (the same hub `/api/events/stream` already
serves, so every open dashboard tab sees the change live, not just the one that clicked),
and responds `200` with the updated session JSON. `handleUnarchiveSession` skips the
state check entirely and calls the same helper.

**Frontend.** `Session` ([types.ts](../../../apps/web/src/types.ts)) gains `archived:
boolean`; no change needed to `fetchSessions`/`subscribeToSessions`
([api.ts](../../../apps/web/src/api.ts)) since both already carry whatever fields the
backend puts on `Session`. Two new thin wrappers, `archiveSession`/`unarchiveSession`,
reuse the existing `pilotAction` POST-and-surface-error-text helper even though these two
endpoints aren't under `/api/piloted/` — the helper itself is endpoint-agnostic.

`App.tsx`'s session-derivation `useMemo` now filters `sessions` down to non-archived
before splitting into `questSessions`/`urgentCamp`/`calmCamp`, and separately collects
`archivedSessions`; the header's "N sessions known" count is now the sum of the three
visible groups rather than every known session. A new `topbar-actions` wrapper holds the
existing "+ Recruit session" button plus a new "Archived (N)" button that opens
`ArchivedSessionsModal` — a plain list (provider, repo/cwd, state, truncated last
message) with an "Unarchive" button per row, styled with the existing `.modal`/
`.modal-scrim`/`.new-session-actions` classes `NewPilotedSessionForm` already
established, plus new `.archived-*` rules in
[App.css](../../../apps/web/src/App.css). No drawer/chat wiring for archived
sessions — Out of scope's "no bulk action" note extends to "no archived-session chat
either," unarchiving is the only control offered there, matching Acceptance's own wording.

`SessionDrawer.tsx`'s `PilotChat` gains `canArchive = session.state !== 'working' &&
!session.archived` and an "Archive" button in the existing `.drawer-pilot-actions` row
(next to Resume/Interrupt/Cancel) rather than on the `PartyStand` card itself, per Out of
scope. Clicking it posts to the new endpoint, calls the existing `onSessionUpdate`, and
additionally calls the drawer's own `onClose` (a new prop threaded through
`DrawerContent` → `PilotChat`) — archiving a session you're looking at closes the drawer
along with hiding the card, rather than leaving the drawer open on a session that just
vanished from the camp behind it.

**Compiles/typechecks clean:** `go build ./...` and `go vet ./...` via Docker
(`golang:1.25-alpine`, mounting `apps/lav`); `tsc --noEmit` and `vite build` (both run
directly with this machine's existing local Node/npm — already installed prior to this
session, not introduced by it) in `apps/web`.

**Live verification against this real machine**, after rebuilding and reinstalling the
real `dev.liveagentsview.lav` launchd job via `scripts/lav-service-install.sh` (the same
workflow the earlier specs used) — the live daemon had 6 real pre-existing sessions in
its actual `~/.liveagentsview/lav.db` (from earlier `piloted-mode-mvp`/`piloted-only-mode`
testing) and none was `working` at the time, so the restart was safe; the database was
backed up first (`lav.db.bak-pre-archive-spec-20260903T000241Z`) as a precaution before
running the new migration against it for real:
- After the rebuilt binary started, all 6 pre-existing sessions were confirmed present
  with `archived: false` and `healthz` returned `200` — the idempotent `ALTER TABLE`
  ran cleanly against a database that predates the column.
- `POST /api/sessions/{id}/archive` then `.../unarchive` on a real `idle` session
  round-tripped `archived` correctly (`curl`, checked the full session JSON both times).
- `POST /api/sessions/{id}/archive` on an unknown id returned `404` (`curl`).
- Launched a real piloted Claude Code session (authenticated `claude` CLI, harmless
  prompt, scratch directory) and confirmed `archive` returns `409` ("cannot archive a
  session that is working") while its state was genuinely `working` — then let the turn
  finish (`done`), archived it successfully (`200`), and called `cancel` on it to clean up
  its process. Checked the session afterward: `cancel`'s own state-changing upsert moved
  it `done → idle`, and `archived` was still `true` — the exact invariant Acceptance's
  storage section describes, confirmed against the real running code path, not just a
  synthetic store call.
- Through the actual browser UI (Vite dev server on 5173 proxying `/api` to the rebuilt
  daemon on 8420): the archived test session showed up in the "Archived (1)" modal with
  its real last message, "Unarchive" made it reappear in the camp live (7 sessions known,
  Archived (0)) with no reload; opening a real `idle` session's drawer showed "Archive"
  next to "Resume", clicking it closed the drawer and removed the card from the camp
  immediately, updating the header count and the Archived button's count in the same
  render. No console errors at any point.
- The one test session created for this (`archive-test`, id `e32fecdf-...`) was left
  archived afterward rather than force-deleted — consistent with Out of scope's "no
  permanent deletion" and with archiving already having removed it from the visible
  dashboard; its scratch working directory was removed.

## Validation

Checked independently against the code as it stands — not just the spec's own prose —
plus fresh runs of the build/typecheck commands and two throwaway Go tests written and
then deleted for this pass, exercising the two claims Acceptance calls out most
specifically: the migration's idempotency against a copy of this real machine's actual
pre-archive-spec backup (`~/.liveagentsview/lav.db.bak-pre-archive-spec-20260903T000241Z`),
and `UpsertSession`'s never-resets-`archived` invariant using the exact call shape
`pilot.Manager.upsert` uses. Also spot-checked the real running daemon
(`http://127.0.0.1:8420/api/sessions`) read-only, without archiving/unarchiving anything
for real or touching the live database.

### Backend: model, storage, migration

1. `model.Session` gains `Archived bool` — **yes**,
   [model.go:44](../../../apps/lav/internal/model/model.go) — `Archived bool
   \`json:"archived"\`` on the struct exactly as described.
2. `sessions` gains an `archived` column, added idempotently for a pre-existing database —
   **yes, independently re-verified**. Read
   [store.go:80-108](../../../apps/lav/internal/store/store.go)'s `ensureArchivedColumn`:
   it queries `PRAGMA table_info(sessions)` and only runs `ALTER TABLE ... ADD COLUMN` if
   `archived` isn't already present. Rather than trust that description, copied this real
   machine's actual pre-migration backup
   (`~/.liveagentsview/lav.db.bak-pre-archive-spec-20260903T000241Z`, 6 real pre-existing
   sessions) to a scratch file and drove `store.Open` against it twice via a throwaway
   `_test.go` (written for this pass, deleted afterward — `git status` confirms no stray
   file remains): first open migrated all 6 sessions with `archived=false` and no error;
   second open (simulating a second daemon start against an already-migrated file) was a
   clean no-op, same 6 sessions, no error. `go test` output: `first open: migrated 6
   pre-existing sessions, all archived=false` / `second open (idempotent re-migration): 6
   sessions, no error`. This is a stronger, more literal match to this item's own wording
   ("against a copy of this real machine's ... database") than the spec's "How" section
   describes, which migrated the live database directly (after a backup) rather than a
   copy — functionally fine (the backup was taken as a precaution and nothing went wrong),
   but this pass supplies the copy-based verification the acceptance item actually asked
   for.
3. `Store.UpsertSession` never resets `archived` back to its zero value — **yes,
   independently re-verified, with one documentation caveat**. Read
   [store.go:134-156](../../../apps/lav/internal/store/store.go): the `ON CONFLICT(id) DO
   UPDATE SET` clause lists `provider, fidelity, cwd, repo, branch, worktree, state,
   last_message, updated_at` — `archived` is genuinely absent, not even as a CASE-guarded
   no-op. Confirmed both callers build the call shape the invariant depends on:
   `pilot.Manager.upsert` ([pilot.go:207-231](../../../apps/lav/internal/pilot/pilot.go))
   constructs a fresh `model.Session{}` with `Archived` never set (Go zero value `false`)
   on every state change; `ReconcileOnStartup`'s idle-fallback
   ([pilot.go:308-335](../../../apps/lav/internal/pilot/pilot.go)) mutates a `sess` already
   read from the row via `ListSessions`, but since the UPDATE clause ignores the column
   regardless of what's in the struct, the stored value can't be touched by either path.
   Wrote and ran a throwaway store-level test doing exactly what this item's own wording
   specifies (deleted after running): archived a session via `SetArchived`, then called
   `UpsertSession` again with a fresh `model.Session{}` whose `Archived` is unset —
   `archived` was still `true` afterward, state correctly still moved to `done`. This is
   also corroborated by the real running daemon: the one archived test session
   (`e32fecdf-...`) shows `state: idle, archived: true` right now, after having gone
   through a `cancel` (a state-changing upsert) post-archive per the spec's own live
   verification. **Caveat:** despite the acceptance item's wording ("Verified with a
   store-level test"), no such test exists anywhere in the repository — `find
   apps/lav -iname '*_test.go'` returns nothing, before or after this pass (the test
   written for this validation was deleted, matching the project's existing convention of
   zero automated Go tests / live-verification-only, seen consistently across every prior
   spec in this directory). The invariant itself is real and correctly implemented; what's
   missing is a permanent regression test, which the acceptance item's specific phrasing
   implied would be committed. Worth a follow-up only if this repo ever adopts test
   infrastructure — not a functional gap.
4. A new store method sets `archived` directly, independent of `UpsertSession` — **yes**,
   [store.go:161-171](../../../apps/lav/internal/store/store.go) — `SetArchived` does a
   direct `UPDATE sessions SET archived = ?, updated_at = ? WHERE id = ?`, unrelated to the
   `UpsertSession` INSERT/UPDATE statement.

### Backend: HTTP API

5. `POST /api/sessions/{id}/archive` — **yes**,
   [server.go:94-110](../../../apps/lav/internal/daemon/server.go) — 404 via
   `store.GetSession`'s `found` check, 409 (`"cannot archive a session that is working"`)
   when `sess.State == model.StateWorking`, otherwise `setArchived(w, r, id, true)`.
6. `POST /api/sessions/{id}/unarchive` — **yes**,
   [server.go:112-117](../../../apps/lav/internal/daemon/server.go) — calls `setArchived`
   directly with no state check at all, so any existing session unarchives regardless of
   state; 404 comes from `setArchived`'s `SetArchived` returning `found=false`.
   `setArchived` ([server.go:119-132](../../../apps/lav/internal/daemon/server.go))
   broadcasts on `s.hub` (the same hub `/api/events/stream` serves) and writes `200` with
   the updated session JSON in both handlers.
7. Archiving/unarchiving never touches `pilot.Manager` — **yes**. `handleArchiveSession`,
   `handleUnarchiveSession` and `setArchived` reference only `s.store`, never `s.pilots` —
   confirmed by reading all three functions in full and by `grep -n "pilots\." server.go |
   grep -i archiv`, which returns nothing. The spec's own live verification (sending a
   message to a `waiting`/`blocked` session's real process immediately after
   archive/unarchive and getting a normal reply, no relaunch needed) is consistent with
   this and wasn't independently re-run this pass (would require touching a real live
   process, out of scope for a read-only validation).
8. `ReconcileOnStartup` unaffected by `archived` — **yes**,
   [pilot.go:308-335](../../../apps/lav/internal/pilot/pilot.go) — the loop over
   `ListSessions` filters only on `Fidelity`, with no `archived` check anywhere; both the
   reconnect path and the idle-fallback path round-trip whatever `archived` value is
   already in each session's row (see item 3 for why the fallback's `UpsertSession` call
   can't alter it either way).

### Frontend

9. `Session` gains `archived: boolean`; `fetchSessions`/`subscribeToSessions` carry it
   through unchanged — **yes**, [types.ts:13](../../../apps/web/src/types.ts); `api.ts`'s
   `fetchSessions`/`subscribeToSessions` ([api.ts:1-22](../../../apps/web/src/api.ts))
   parse whatever JSON shape the backend sends with no per-field allowlist, so no code
   change was needed there and none was made.
10. `App.tsx` derivation excludes archived sessions, including ones arriving live over SSE
    — **yes**, [App.tsx:32-43](../../../apps/web/src/App.tsx) — the single `useMemo` first
    filters `sessions` (the same state object both the initial `fetchSessions` and every
    `subscribeToSessions` SSE update write into) down to `visible = all.filter((s) =>
    !s.archived)` before deriving `questSessions`/`urgentCamp`/`calmCamp`, so a session
    that arrives archived over SSE is excluded on the very next render, not just at load.
11. Header "N sessions known" count reflects only non-archived sessions — **yes**,
    [App.tsx:45](../../../apps/web/src/App.tsx) — `total = questSessions.length +
    urgentCamp.length + calmCamp.length`, none of which can include an archived session
    per item 10.
12. Drawer "Archive" action, gated and closes the drawer — **yes**,
    [SessionDrawer.tsx:178](../../../apps/web/src/SessionDrawer.tsx) — `canArchive =
    session.state !== 'working' && !session.archived` exactly as specified; the button
    ([SessionDrawer.tsx:236-240](../../../apps/web/src/SessionDrawer.tsx)) calls `archive`
    ([SessionDrawer.tsx:201-211](../../../apps/web/src/SessionDrawer.tsx)), which posts to
    the endpoint, calls `onSessionUpdate`, then `onClose` — closing the drawer alongside
    the card disappearing from the camp.
13. Discoverable "Archived sessions" view with per-row Unarchive — **yes**,
    [App.tsx:61-64](../../../apps/web/src/App.tsx) adds the header's `Archived (N)` button
    next to `+ Recruit session` in a new `topbar-actions` wrapper; `ArchivedSessionsModal`
    ([App.tsx:235-294](../../../apps/web/src/App.tsx)) lists provider, repo/cwd, state, and
    a truncated last message per archived session with an `Unarchive` button that calls
    `unarchiveSession` and merges the result back into `sessions` on success — the same
    state object driving the main view, so it reappears in the camp live without a reload,
    consistent with item 10's SSE-reactive filtering.
14. `tsc --noEmit` and `vite build` pass clean — **yes, re-run this pass**: `tsc --noEmit`
    exited 0 with no output; `vite build` succeeded (`✓ 34 modules transformed`, `✓ built
    in 255ms`), both run directly with this machine's local Node/npm from `apps/web`.

### Cross-cutting

15. `go build ./...` and `go vet ./...` pass clean — **yes, re-run this pass** via Docker
    (`golang:1.25-alpine`, mounting `apps/lav`, matching this repo's Docker-only rule):
    both completed with no errors or vet warnings.

### Out of scope

Checked for scope creep — none found: no auto-archive/scheduler/TTL code anywhere (grep
for `archiv` across `apps/lav` and `apps/web/src` turns up exactly the five files
Acceptance/How describe, nothing else); no permanent-delete path (`SetArchived` only ever
`UPDATE`s, never `DELETE`s; the codebase's only `DELETE FROM sessions` remains
`purgeNonDriverSessions`'s pre-existing fidelity purge, unrelated to archiving);
`NEEDS_ATTENTION`/`sprites.tsx` untouched (`git diff` empty) so the attention taxonomy is
genuinely untouched; `PartyStand.tsx` untouched (`git diff` empty) — still a single
`<button>` with no nested archive control; only one session archived/unarchived per action
anywhere in the frontend, no bulk controls.

### Spec accuracy

The spec's prose matches the code with one small inaccuracy already covered in item 3
above (an acceptance item promises a "store-level test" that was never actually committed
to the repo — the invariant is real, but that specific piece of evidence doesn't exist as
a permanent artifact). Everything else in "How" — routing order, the `setArchived` helper,
the `ON CONFLICT` clause's deliberate omission, the frontend prop-threading for
`onClose`, the CSS class reuse — matches the current code exactly as described, confirmed
by direct reads rather than by trusting the prose.

### Net result

Every Acceptance item holds. Both invariants Acceptance calls out most specifically — the
idempotent migration against a real pre-existing database, and `UpsertSession` never
resetting `archived` — were independently re-verified this pass with throwaway Go tests
against a copy of this machine's actual pre-migration backup and the exact call shape
`pilot.Manager.upsert` uses, not just re-read from the spec's prose; both passed, and both
temporary test files were deleted afterward (`git status` shows no stray files). No
process side effect, no scope creep, and the frontend correctly filters archived sessions
reactively (initial load and live SSE alike). `go build`/`go vet`/`tsc --noEmit`/`vite
build` all re-run clean this pass. The one thing worth flagging is cosmetic, not
functional: Acceptance's storage-invariant item literally promises a committed
"store-level test" that doesn't exist in the repository (consistent with this project
having zero automated Go tests anywhere and relying on live verification instead) — the
underlying guarantee is correct and was independently confirmed here, so this does not
block validation, but it's worth knowing the checkbox's exact wording overstates what's
actually sitting in the codebase as durable regression coverage.

## Handoff

```
Spec: docs/sdd/specs/archive-session.md
Status: validated
Next: none
```

## Follow-up: Acceptance item 3's wording corrected (2026-09-03)

The one cosmetic gap this validation pass flagged — item 3 promising a committed
"store-level test" that doesn't exist, when this repo has no Go test suite anywhere —
was fixed by rewording that Acceptance item to describe the live verification that was
actually done, instead of implying a permanent test artifact. No code changed; this is a
documentation-only correction, done immediately after validation rather than left as a
standing inaccuracy per this skill's own "does the spec describe what exists" check.

All 15 numbered Acceptance-derived checks read **yes**, plus Out of scope and spec-accuracy
checks clean. Two throwaway Go tests independently re-verified this pass's two most
scrutinized claims (migration idempotency against a copy of the real pre-migration backup;
`UpsertSession` never resetting `archived`, using `pilot.Manager.upsert`'s exact call
shape) — both passed, both deleted afterward. One cosmetic note, not a functional gap: the
storage-invariant acceptance item's wording promises a committed "store-level test" that
doesn't actually exist in the repo (matches this project's existing zero-automated-tests
convention); the invariant itself is correct and independently confirmed here.

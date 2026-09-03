# Status

Mirror of the definition documents and the code that exists. **Decides nothing.**
Updated when a fact changes, never to anticipate one.

Last updated: 2026-09-03 (character model redesign specified: vocabulary, two axes,
territory, lifecycle and the removal of permission management decided and distilled;
character-model-redesign spec created, ready to implement)

## Documentation

| Area | Status |
|---|---|
| Vision | defined — [01-vision.md](01-vision.md) |
| Scope | defined — [02-scope.md](02-scope.md) |
| Decisions | 24 entries — [03-decisions.md](03-decisions.md). The seven from 2026-09-03 supersede the 2026-09-01 canonical state model and the 2026-09-02 archive decision |
| Open questions | 4 (Q-03, Q-06 deferred; Q-10, Q-11 open and non-blocking) — [04-open-questions.md](04-open-questions.md) |
| Ideas to discuss | 12 unagreed (IDEA-11 and IDEA-12 added 2026-09-03) — [05-ideas-to-discuss.md](05-ideas-to-discuss.md) |
| Inbox | 2026-09-01 product session, 2026-09-02 native-runtime follow-ups, 2026-09-02 dashboard-scope/restart-continuity session, 2026-09-02 archive-a-session session, and 2026-09-03 character-model session, all distilled. The CSRF finding in the 2026-09-03 block went to its own spec rather than to a definition document — it is a security fix, not a definition |

## Product

| Piece | Status |
|---|---|
| Posture | decided — piloted only as of 2026-09-02 (supersedes observer + opt-in pilot); adopted/hooks removed entirely, not just hidden |
| Stack | decided — Go, SQLite, single binary with embedded frontend (React + Vite) |
| Providers | Claude Code and Cursor usable (piloted adapters exist); Codex has no representation until a driver adapter is built (see 2026-09-02 in [03-decisions.md](03-decisions.md)) |
| Vocabulary | decided 2026-09-03 — character / race / class / territory / quest / camp. Not built yet: the code and interface still say session, provider and model |
| Attention taxonomy | partial — completion and classifier decided, full table in IDEA-01. P0 (Blocked) has had no content since permission management was dropped on 2026-09-03 |
| Canonical event model | redecided 2026-09-03 — two axes: activity (`ready`/`working`/`waiting`/`failed`) plus an unread mark, and presence (awake/asleep) observed, never stored. Supersedes WORKING/WAITING/BLOCKED/DONE/FAILED/IDLE. Not built yet |
| Permissions | decided 2026-09-03 — not mediated at all; every race runs auto-approving. Not built yet: Claude Code still launches with a live permission gate |
| Territory | decided 2026-09-03 — own worktree (default) or shared directory, and never `git checkout` on the user's directory. Not built yet: launching with a branch still checks it out in place |
| Lifecycle | decided 2026-09-03 — archiving sleeps a character and frees its memory, in any activity; dismissing removes it. Not built yet: archiving still leaves the process running and is refused while working |

## Code and infrastructure

| Piece | Status |
|---|---|
| Repository structure | done — docs, sdd, scripts, apps/lav, apps/web |
| Apps in `apps/` | `lav` (daemon+CLI) and `web` (dashboard) exist and build; see [apps/README.md](../apps/README.md) |
| Compose local | done — `lav` service, SQLite bind-mounted to `~/.liveagentsview`, 127.0.0.1-only (Docker's own port publish enforces this for the containerized path) |
| Native service bind address | done — `127.0.0.1:<port>` (fixed from all-interfaces during piloted-mode-mvp's validate pass), rebuilt and confirmed live on this machine's launchd service |
| Scripts | `dev-up.sh`, `dev-down.sh`, `lav-service-install.sh`, `lav-init.sh`, `lav-status.sh` — [scripts/README.md](../scripts/README.md) |
| CI | not set up yet |
| Remote | published at github.com/Antiloope/LiveAgentsView (Q-03 itself stays deferred as a docs question, but the repo already exists) |

## SDD specs

- [adopted-mode-mvp](sdd/specs/adopted-mode-mvp.md) — validated. 9 of 12 acceptance items
  met with direct evidence (real Docker build, real hooks installed against this
  machine's Claude Code/Codex/Cursor config, live SSE dashboard, SQLite persistence
  across a restart); 3 partial items carried over as the follow-up spec below.
- [native-host-runtime](sdd/specs/native-host-runtime.md) — validated. 8 of 9 acceptance
  items met with direct evidence on this machine: the daemon runs as a real launchd user
  service (confirmed via `launchctl`, survives kill-and-respawn, no Docker involved at
  runtime), "open in terminal" actually opens a terminal at the session's `cwd` on macOS,
  and a real captured Cursor hook payload found and fixed a field-name bug
  (`session_id`/`workspace_roots`, not the old `sessionId`/`cwd` guess) in
  `internal/ingest/cursor.go`. 1 partial: `stop`/`postToolUseFailure`'s exact Cursor
  field names stay best-effort (never fired during verification); Linux (systemd,
  terminal fallback) stays code-reviewed only, not live-verified — both accepted as
  known gaps rather than blocking closure, not revisited by a follow-up spec.
- [piloted-mode-mvp](sdd/specs/piloted-mode-mvp.md) — validated. Real verification this
  session forced a scope change while implementing: Cursor's CLI has no bidirectional
  driver protocol at all (confirmed live), so Cursor piloted sessions auto-approve
  (`--force`/`--yolo`) instead of getting Claude Code's live permission approve/deny. Both
  providers verified end-to-end against a real cross-compiled native binary on an isolated
  data directory — Cursor fully (launch, multi-turn `--resume`, daemon-restart
  reconciliation, resume-after-restart, all validation/error paths); Claude Code's process
  spawn and stream parsing, but not live permission-approval/interrupt (this environment's
  `claude` CLI cannot authenticate — accepted as the one known gap, same pattern as the
  other two specs' accepted gaps). Also fixed, found during the same implementation pass: a
  piloted session's own CLI hooks could silently downgrade it from Driver back to Hooks
  fidelity. The validate pass itself found and fully closed two more real gaps: an
  `AGENTS.md` doc-citation violation in a code comment, and `cmd/lav/main.go` binding all
  interfaces instead of `127.0.0.1` only (this machine's real running launchd service was
  confirmed, live, listening on `*:8420` — the Docker port-publish that made this true for
  adopted-mode-mvp doesn't apply once native execution is the real run path). Fixed,
  rebuilt via `scripts/lav-service-install.sh`, and reconfirmed live on this machine:
  `lsof` now shows `127.0.0.1:8420` only, `healthz` returns `200`.
- [piloted-only-mode](sdd/specs/piloted-only-mode.md) — validated. Adopted mode and hooks
  ingestion removed from the codebase entirely; the real hooks a previous `lav init` had
  written to this machine's Claude Code/Codex/Cursor configs were uninstalled for real,
  backed up first. Piloted sessions gained restart continuity: a detached `lav
  pilot-runner` process now owns the real `claude`/`agent` child, survives a `lav`
  restart, and the daemon reconnects on startup with no dropped/duplicated transcript and
  no lost in-progress turn. Three gaps found by the first validation pass (interrupt
  mismarked as failed, permission approve/deny never reaching the CLI, Resume able to
  orphan a still-running process) were fixed and live re-verified through the real
  dashboard.
- [local-api-hardening](sdd/specs/local-api-hardening.md) — **ready, not started.**
  Rejects cross-site requests to the daemon (`Origin`/`Sec-Fetch-Site`, a `Host` check
  against DNS rebinding, a required `Content-Type`, and a custom header on every
  state-changing call). Closes a hole confirmed live on 2026-09-03: a cross-origin simple
  POST reaches every endpoint, so any open web page can launch a character here.
- [character-model-redesign](sdd/specs/character-model-redesign.md) — **ready, not
  started.** Implements the seven 2026-09-03 decisions in one pass: the character
  vocabulary, activity/presence as two axes with `done` becoming an unread mark, own-vs-
  shared territory with real git worktrees, a continuous reconciler so the daemon's belief
  cannot drift from what is running, archive-as-sleep plus dismiss, and the deletion of
  `internal/pilotmcp` and everything else permission-related. Explicitly out of its scope:
  the CSRF finding, the HP/MP placeholder bars, Q-10 and Q-11.
- [archive-session](sdd/specs/archive-session.md) — validated. A session can be archived
  from the dashboard (any state except `working`) to hide it from the camp/quest view,
  reversibly, from a new "Archived sessions" view. Persisted as a SQLite column, added
  idempotently for this machine's real pre-existing database; a state-changing upsert on
  an already-archived session (e.g. its process finishing after being archived) cannot
  silently unarchive it. No process side effect. Verified live against the real running
  service and through the real browser UI.

Index in [sdd/README.md](sdd/README.md).

## Suggested next step

Two specs are `ready` and independent of each other.
[local-api-hardening](sdd/specs/local-api-hardening.md) is the one to do first: the API
has no `Origin`, `Host` or `Content-Type` check, so any web page open in the browser can
make the daemon launch an agent on this machine — measured live, not inferred.
[character-model-redesign](sdd/specs/character-model-redesign.md) is the larger one.

`recruit-flow-redesign` is still `done`/awaiting validation, with three acceptance items
now superseded (annotated in the spec itself).

Still open and unagreed from earlier sessions: IDEA-07 (`lav version`), IDEA-09
(friendlier local address), IDEA-10 (prebuilt binaries and a first-run wizard), IDEA-08's
`lav service uninstall` half, and now IDEA-11 (a permission layer) and IDEA-12 (changing a
character's class).

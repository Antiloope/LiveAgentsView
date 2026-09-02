# Status

Mirror of the definition documents and the code that exists. **Decides nothing.**
Updated when a fact changes, never to anticipate one.

Last updated: 2026-09-02 (piloted-mode-mvp)

## Documentation

| Area | Status |
|---|---|
| Vision | defined — [01-vision.md](01-vision.md) |
| Scope | defined — [02-scope.md](02-scope.md) |
| Decisions | 14 entries — [03-decisions.md](03-decisions.md) |
| Open questions | 2 (Q-03, Q-06), both deferred — [04-open-questions.md](04-open-questions.md) |
| Ideas to discuss | 10 unagreed — [05-ideas-to-discuss.md](05-ideas-to-discuss.md) |
| Inbox | 2026-09-01 product session and 2026-09-02 native-runtime follow-ups, both distilled |

## Product

| Piece | Status |
|---|---|
| Posture | decided — observer + opt-in pilot |
| Stack | decided — Go, SQLite, single binary with embedded frontend (React + Vite) |
| Providers | decided — Claude Code, Codex, Cursor, all three built together in the first spec |
| Attention taxonomy | partial — completion and classifier decided, full table in IDEA-01 |
| Canonical event model | decided — WORKING/WAITING/BLOCKED/DONE/FAILED/IDLE, validated against the 3 providers' docs |

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

Index in [sdd/README.md](sdd/README.md).

## Suggested next step

No candidate follow-up specs pending from any of the three closed specs — adopted-mode-mvp,
native-host-runtime and piloted-mode-mvp are all `validated`. Four new, unprioritized ideas
came out of setting up the native service, logged as IDEA-07 through IDEA-10 in
[05-ideas-to-discuss.md](05-ideas-to-discuss.md): a `lav version` command, an uninstall
path, a friendlier local dashboard address, and prebuilt binary distribution with a
first-run install wizard. None agreed yet.

No spec currently open. The next meaningful product increment needs its own `specify` pass
when one is proposed.

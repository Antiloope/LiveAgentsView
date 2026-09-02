# Status

Mirror of the definition documents and the code that exists. **Decides nothing.**
Updated when a fact changes, never to anticipate one.

Last updated: 2026-09-01

## Documentation

| Area | Status |
|---|---|
| Vision | defined — [01-vision.md](01-vision.md) |
| Scope | defined — [02-scope.md](02-scope.md) |
| Decisions | 13 entries — [03-decisions.md](03-decisions.md) |
| Open questions | 2 (Q-03, Q-06), both deferred — [04-open-questions.md](04-open-questions.md) |
| Ideas to discuss | 6 unagreed — [05-ideas-to-discuss.md](05-ideas-to-discuss.md) |
| Inbox | 2026-09-01 product session distilled |

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
| Compose local | done — `lav` service, SQLite bind-mounted to `~/.liveagentsview`, 127.0.0.1-only |
| Scripts | `dev-up.sh`, `dev-down.sh`, `lav-init.sh`, `lav-status.sh` — [scripts/README.md](../scripts/README.md) |
| CI | not set up yet |
| Remote | published at github.com/Antiloope/LiveAgentsView (Q-03 itself stays deferred as a docs question, but the repo already exists) |

## SDD specs

[adopted-mode-mvp](sdd/specs/adopted-mode-mvp.md) — done, next: validate. Built and
smoke-tested via Docker against real local Claude Code/Codex config (dry-run) this
session; not yet formally validated against the spec's acceptance list. Index in
[sdd/README.md](sdd/README.md).

## Suggested next step

Validate the `adopted-mode-mvp` spec against its acceptance list, then decide on the two
flagged gaps: native launchd/systemd service install, and a real "open in the terminal"
(currently copy-path only).

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
| Repository structure | done — docs, sdd, scripts placeholder, apps/ |
| Apps in `apps/` | pending — folder ready, no deployables |
| Compose local | sketch — Postgres only, needs updating to match the 2026-09-01 Docker-for-dev-only decision |
| Scripts | placeholder — [scripts/README.md](../scripts/README.md) |
| CI | not set up yet |
| Remote | not published — Q-03 (deferred) |

## SDD specs

[adopted-mode-mvp](sdd/specs/adopted-mode-mvp.md) — ready, next: implement. Index in
[sdd/README.md](sdd/README.md).

## Suggested next step

Implement the `adopted-mode-mvp` spec: daemon, `lav init`, adapters for Claude Code,
Codex and Cursor over the canonical event model, and the embedded React + Vite dashboard.

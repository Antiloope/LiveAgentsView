# Decisions

Log of decisions made, with date and rationale. Only **agreed** items go here.

Format:

```
## YYYY-MM-DD — Short title

**Who:** …
**Decision:** …
**Rationale:** …
```

---

## 2026-09-01 — Open-source repo with docs-first structure

**Who:** Rodrigo
**Decision:** The project is open source under the MIT license. Definition documentation
lives in `docs/`; implementation specs in `docs/sdd/`. Local development uses Docker.
**Rationale:** Replicate the workflow proven in sincro, adapted to a public project
without coupling api+frontend from the start.

## 2026-09-01 — What LiveAgentsView is

**Who:** Rodrigo
**Decision:** LiveAgentsView is an attention layer for AI coding agents — a local-first
mission control that sits above Claude Code, Codex and Cursor without replacing them.
Its primary interface is an attention queue that answers "do I need to do anything?".
Vision in [01-vision.md](01-vision.md), scope in [02-scope.md](02-scope.md).
**Rationale:** Running many agents in parallel is no longer a launching problem; it is a
supervision problem. The differentiator is filtering agent activity into actionable human
decisions, not running more agents.
**Closes:** Q-01.

## 2026-09-01 — Posture: observer + opt-in pilot ("B")

**Who:** Rodrigo
**Decision:** Sessions launched natively by the user are **adopted** (read-only, detected
via hooks). Sessions launched from LiveAgentsView are **piloted**: a child process over
bidirectional `stream-json`, where questions can be answered and permissions approved
from the dashboard. Leave the architecture prepared to move further toward a full
frontend later, without committing to it.
**Rationale:** The observer-only option never closes the loop — you still have to find
the terminal. Full replacement means rebuilding plan mode, autocomplete, slash commands
and diff review, and falling behind every upstream release. The hybrid closes the loop
where it matters while keeping "open in the terminal" as a first-class feature, and it
uses the same technical mechanism a fuller frontend would need.

## 2026-09-01 — Stack: Go, SQLite, single self-contained binary

**Who:** Rodrigo
**Decision:** Go as the language and SQLite for storage. A single binary for mac/linux
that starts a daemon, serves an embedded frontend over HTTP+SSE on `127.0.0.1`, and is
also usable from the CLI. The daemon runs as a user service (launchd/systemd) so hook
events are not lost while the dashboard is closed. Reaching it from another device — for
example a phone over Tailscale — is a later, explicit opt-in.
**Rationale:** Go cross-compiles to a self-contained binary with good process handling,
and embedding the frontend means one file to download and no Node on the user's machine.
SQLite in `~/.liveagentsview/` fits a single-user local tool; Postgres would require
Docker running permanently. One daemon with several clients (web, TUI, CLI) is one
product, not three.
**Closes:** Q-02.

## 2026-09-01 — MVP providers: Claude Code, Codex and Cursor

**Who:** Rodrigo
**Decision:** The first version targets Claude Code, Codex and Cursor.
**Rationale:** Claude Code has the best signal (native hooks plus a bidirectional
`stream-json` channel) and is the reference provider. Codex has `notify` and indexed
rollouts. Cursor validates that the adapter abstraction holds against a provider with a
weaker surface — with the known constraint that it only works through the `cursor-agent`
CLI.

## 2026-09-01 — Audience for v1: dogfooding with product intent

**Who:** Rodrigo
**Decision:** v1 is built for its author's real workflow, but decisions are taken so the
result can eventually become a product. The repo is public from day one, without
promising support or portability yet.
**Rationale:** Validate "does this reduce the mental effort of supervising many agents?"
before generalizing, without painting the architecture into a corner.

## 2026-09-01 — End-of-turn classification: rules behind a pluggable interface

**Who:** Rodrigo
**Decision:** Disambiguating "finished" from "asked you something" is done with
heuristics over the last message, implemented behind a classifier interface in the state
engine so it can be replaced later without touching anything else.
**Rationale:** Keeps the "no API keys needed" property intact and costs nothing. An LLM
classifier is more robust and multilingual, but it breaks zero-configuration setup; the
pluggable interface leaves that door open for a day's work.

## 2026-09-01 — "An agent finished" is low-priority attention

**Who:** Rodrigo
**Decision:** Completion enters the attention queue as a low-priority, grouped item
without a loud notification ("3 finished since you last looked"). It is not treated as a
mere state change, nor as a full alert.
**Rationale:** A finished agent usually does need something from the human — review,
merge, next task — but not immediately. With ten agents, notifying on every completion is
the fastest route to the user turning notifications off.

## 2026-09-01 — Docker is for developing LiveAgentsView, not for running it

**Who:** Rodrigo
**Decision:** The "only Docker installed on the machine" rule applies to developing and
testing LiveAgentsView itself. The shipped binary runs directly on the host, since it
needs the host keychain, repos, git config and worktrees to spawn `claude`, `codex` and
`cursor-agent`.
**Rationale:** Containerizing the runtime loses host auth and breaks worktrees, which
piloted mode depends on.
**Closes:** Q-04.

## 2026-09-01 — `lav init` merges hooks non-destructively, with a preview

**Who:** Rodrigo
**Decision:** `lav init` reads the existing `~/.claude/settings.json` and
`~/.codex/config.toml`, chains/merges the LiveAgentsView hooks into them without
overwriting what is already configured (for example an existing `notify` program in
Codex), and shows the user a preview of the exact change before writing it.
**Rationale:** `notify` accepts only one program and hooks may already serve other
purposes; overwriting either would silently break the user's existing setup.
**Closes:** Q-07.

## 2026-09-01 — Frontend stack: React + Vite

**Who:** Rodrigo
**Decision:** The embedded UI is built with React and Vite, compiled to static assets and
embedded in the Go binary.
**Rationale:** Best ecosystem for the UI the product needs — live-updating agent cards
over SSE, diff viewing and a chat-like stream for piloted sessions, a command palette —
and the fastest to build with AI assistance. Running on `127.0.0.1` for a single user
means runtime bundle weight is not a real performance concern the way it would be for a
public site; Node is a dev-time dependency only, not a runtime one.
**Closes:** Q-05.

## 2026-09-01 — Canonical event/state model, validated against the 3 providers

**Who:** Rodrigo
**Decision:** The canonical states stay WORKING / WAITING / BLOCKED / DONE / FAILED /
IDLE. Validated against official provider docs (plus Cursor/OpenAI community sources
where docs were silent):
- BLOCKED has a strong, dedicated signal only for Claude Code (`PermissionRequest` /
  `Notification:permission_prompt`). Codex has no BLOCKED signal at Hooks fidelity at
  all (confirmed gap — `notify` only fires on turn completion); Cursor has no dedicated
  event either, only a staff-endorsed heuristic proxy. Both are accepted as known v1
  limitations, not blockers.
- WAITING vs DONE cannot be told apart from the raw event alone for any provider — every
  "turn ended" signal (`Stop`, `notify:agent-turn-complete`, `stop:status=completed`) is
  ambiguous and needs the already-decided rules-based classifier over the last message.
- FAILED has a dedicated, unambiguous signal for all three at Driver level; Claude Code
  and Cursor also have it at Hooks level (`StopFailure`, `stop:status=error` /
  `sessionEnd:reason=error`); Codex does not.
- IDLE is never pushed by any provider at any fidelity level — always derived locally
  from a timeout since the last event.
- Cursor has a real hooks system (`.cursor/hooks.json`), not CLI-only as assumed when
  Q-09 was scoped — confirmed firing for at least `beforeShellExecution` /
  `afterShellExecution` / `afterFileEdit` / `postToolUse` / `stop` / `sessionStart` in
  `cursor-agent`. Whether it also fires for sessions launched natively in the Cursor IDE
  (adopted, not driven by LiveAgentsView) is unconfirmed and left open for spec 1.
**Rationale:** Needed a real contract, not the untested sketch, before building three
adapters against it. The full provider-by-provider signal inventory and state mapping
table live in the first spec (`docs/sdd/specs/`) as implementation detail the adapters
are built against, not product definition.
**Closes:** Q-08.

## 2026-09-01 — Cursor's adapter is built alongside Claude Code and Codex from the start

**Who:** Rodrigo
**Decision:** Cursor stays in MVP scope and its `cursor-agent` adapter is built in the
same first spec as Claude Code and Codex, rather than deferred to a later spec.
**Rationale:** Confirms the existing MVP-providers decision. Cursor's known constraint
(no local hook surface, `cursor-agent` CLI only) is accepted as a limitation to design
around, not a reason to postpone it.
**Closes:** Q-09.

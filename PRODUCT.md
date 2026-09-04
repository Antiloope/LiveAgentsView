# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Developers who run several coding agents (Claude Code, Codex, Cursor) in parallel across
multiple repositories and worktrees. For them, launching agents is no longer the problem —
keeping track of them and knowing when human attention is actually required is.

v1 is built for its own author's (Rodrigo's) daily workflow (dogfooding), but every piece
is built so the product can eventually serve other users in the same position.

## Product Purpose

An attention layer for AI coding agents: a local-first mission control that sits above
Claude Code, Codex and Cursor rather than replacing them. It lets the user answer, at a
glance: what are my agents doing, which need me, which finished, which failed or look
stuck, what happened recently, and can I act without hunting down the original terminal.

Success is not "does the dashboard look good" — it is whether the dashboard reduces the
mental effort of supervising many agents at once.

## Positioning

Attention over activity: the product optimizes for "do I need to do anything?", not "what
is happening?" It is provider-agnostic (providers are adapters behind a canonical event
model, not a dependency on any one vendor), local-first (no required cloud account, no
provider API keys where vendor CLIs suffice), and explicitly does not replace Claude Code,
Codex or Cursor — it launches and drives those CLIs as child processes.

## Operating Context

- **Piloted only.** Every character LiveAgentsView shows is one it launched itself as a
  child process over the vendor CLI (bidirectional `stream-json` where the race supports
  it). There is no **Adopted** class — sessions started outside LiveAgentsView are not
  tracked. Hooks ingestion and read-only "watch someone else's terminal" are gone
  (decided 2026-09-02). Do not design, critique, or implement Adopted / Hooks / Tailing
  as if they were current product.
- **No tool-permission approve/deny UI.** Every race runs auto-approving
  (`bypassPermissions` / `--force`). LiveAgentsView does not mediate model tool
  permissions and must not treat a missing Approve/Deny control as a product gap
  (decided 2026-09-03). A future permission *layer* is only IDEA-11, unagreed.
- **Vocabulary:** character · race · class · territory · quest · camp (see docs/02-scope).
  Activity axis (working / waiting / failed / ready / …) and presence (awake / asleep).
- **Waiting** means the character is waiting on the *user* (a question / end-of-turn),
  not a tool-permission prompt. Attention for waiting is answer-in-chat / interrupt /
  stop — not approve permission.
- Providers in MVP: Claude Code and Cursor are usable as piloted races; Codex waits on a
  driver adapter (see docs/03-decisions.md 2026-09-02).
- Runs as a single Go binary with embedded SQLite and an embedded React + Vite frontend,
  on the host (not containerized at runtime). Binds to `127.0.0.1` by default.

## Capabilities and Constraints

**In scope:** launching and aggregating characters into one dashboard; showing repository,
territory, branch, race, class and normalized activity; an attention surface for what
needs the human; notifications on events that earn an interruption; acting on a character
from the dashboard (chat, interrupt, stop, archive/dismiss); persisting enough state to
survive restarts.

**Out of scope:** agent orchestration or automatic task assignment, task management or
workflows, agent memory or MCP management, team collaboration or enterprise governance,
sophisticated analytics, replacing the underlying IDE or CLI, mediating tool permissions,
adopting externally launched sessions.

**Explicit boundaries:** LLM credentials are never managed — vendor CLIs run as
subprocesses inheriting the user's existing login. A local server that can launch an
auto-approving coding agent is remote code execution, so it binds to `127.0.0.1` by
default; exposing it to other devices is a separate, explicit opt-in.

**Known gaps (accepted):** WAITING vs finished end-of-turn disambiguation is rules-based
classification (no LLM classifier, to preserve no-API-keys). IDLE is derived locally from
a timeout. Codex has no piloted adapter yet.

## Brand Commitments

Product name is "LiveAgentsView." Visual identity is the Craft Pixel world documented in
DESIGN.md (navy/gold HUD, parchment Quest Ledger, night camp). No separate logo mark is
decided beyond that.

## Evidence on Hand

No real captured session data, screenshots, or transcripts exist yet as design reference
beyond review/demo parties. Future design work must treat example agent session data as
placeholder — not fabricate it as if it were real usage.

## Product Principles

1. Attention over activity — surface "do I need to act?" before "what is happening?"
2. Provider-agnostic — no piece of the design should read as Claude-only, Codex-only, or
   Cursor-only; providers are interchangeable adapters behind one model.
3. Local-first and single-user — this is a personal control surface bound to
   `127.0.0.1`, not a multi-tenant or team product, even though it may grow into one.
4. Complement, don't replace — vendor CLIs stay the engine; LiveAgentsView is the layer
   that launches and supervises them.
5. Glanceable at 10–20 agents — the state of many concurrent characters must read in
   seconds, not require per-session inspection.

## Accessibility & Inclusion

No product-specific accessibility requirement has been established yet.

## Open Decisions

- Whether/how "a human is watching this session" is detected — deferred.
- Full attention-priority taxonomy beyond completion-is-low-priority — proposed, not yet
  agreed (see `docs/05-ideas-to-discuss.md`, IDEA-01). Note: IDEA-01's old P0 "permission
  request" row is stale relative to the 2026-09-03 permission drop.
- Componentized Craft Pixel chrome + layered camp kits (race / class / items) — proposed
  as craft direction (IDEA-13), not yet agreed as a build plan.

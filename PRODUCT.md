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
provider API keys where native CLIs/hooks suffice), and explicitly does not replace Claude
Code, Codex or Cursor — "open in the terminal" is a first-class feature, not a fallback.

## Operating Context

- Two session classes coexist in one interface: **Adopted** (launched natively by the
  user, reports in via hooks, read-only — state, attention queue, notifications, jump to
  terminal) and **Piloted** (launched from LiveAgentsView as a child process over
  `stream-json`, controllable — answer questions, approve/deny permissions, cancel).
- Three integration fidelity levels, shown per session: **Driver** (bidirectional
  `stream-json`, full control), **Hooks** (`~/.claude/settings.json`,
  `~/.codex/config.toml` `notify`, read-only push), **Tailing** (vendor JSONL
  transcripts, retroactive, no control, cannot tell "waiting" from "done").
- Canonical session states: WORKING / WAITING / BLOCKED / DONE / FAILED / IDLE. A
  finished agent is low-priority attention (grouped, no loud notification), not a bare
  state change and not a full alert.
- Providers in MVP: Claude Code, Codex, Cursor — built together, not staggered. Cursor's
  IDE agent has no local hook surface for adopted sessions beyond the confirmed
  `.cursor/hooks.json` events; its piloted mode has no bidirectional driver protocol, so
  Cursor piloted sessions auto-approve every tool call instead of getting a live
  approve/deny gate.
- Runs as a single Go binary with an embedded SQLite store and an embedded React + Vite
  frontend, executed directly on the host (not containerized at runtime) so it can reach
  the host keychain, repos, git config and worktrees needed to spawn `claude`, `codex`
  and `cursor-agent`. Binds to `127.0.0.1` by default.

## Capabilities and Constraints

**In scope:** detecting local coding-agent sessions and aggregating them into one
dashboard; showing repository, worktree, branch, provider and normalized state per
session; an attention queue prioritized by what needs the human; notifications on events
that earn an interruption; jumping to the originating session or acting on it directly;
a recently-finished list; persisting enough state/history to survive restarts.

**Out of scope:** agent orchestration or automatic task assignment, task management or
workflows, agent memory or MCP management, team collaboration or enterprise governance,
sophisticated analytics, replacing the underlying IDE or CLI.

**Explicit boundaries:** LLM credentials are never managed — vendor CLIs run as
subprocesses inheriting the user's existing login; LiveAgentsView never sees, stores or
asks for API keys. Filesystem permissions are delegated to the underlying agent (Claude
Code's permission modes/allow-deny rules, Codex's sandbox/approval policy); LiveAgentsView
surfaces the prompt and routes the answer, it does not sandbox on its own. A local server
that can approve permissions is remote code execution, so it binds to `127.0.0.1` by
default; exposing it to other devices is a separate, explicit opt-in.

**Known provider gaps (accepted, not blockers):** BLOCKED has a strong dedicated signal
only for Claude Code; Codex has none at Hooks fidelity, Cursor only a heuristic proxy.
WAITING vs. DONE is ambiguous from the raw event for every provider and is resolved by a
rules-based classifier over the last message (chosen over an LLM classifier to preserve
the no-API-keys property), behind a pluggable classifier interface. IDLE is always
derived locally from a timeout, never pushed by a provider.

## Brand Commitments

Product name is "LiveAgentsView." No visual identity (logo, wordmark, color mark) exists
yet, and none of that is decided — open for later design work.

## Evidence on Hand

No real captured session data, screenshots, or transcripts exist yet as design reference.
Future design work must treat any example agent session, status data, or activity as
placeholder — not fabricate it as if it were real usage.

## Product Principles

1. Attention over activity — surface "do I need to act?" before "what is happening?"
2. Provider-agnostic — no piece of the design should read as Claude-only, Codex-only, or
   Cursor-only; providers are interchangeable adapters behind one model.
3. Local-first and single-user — this is a personal control surface bound to
   `127.0.0.1`, not a multi-tenant or team product, even though it may grow into one.
4. Complement, don't replace — the terminal/IDE stays the source of truth; LiveAgentsView
   is the layer above it, never a competing editor or chat surface.
5. Glanceable at 10–20 agents — the state of many concurrent sessions must read in
   seconds, not require per-session inspection.

## Accessibility & Inclusion

No product-specific accessibility requirement has been established yet.

## Open Decisions

- Whether/how "a human is watching this session" is detected (candidate signals: terminal
  focus, a recent `UserPromptSubmit`, active tmux pane) — deferred, not needed until the
  attention engine is built.
- Full attention-priority taxonomy beyond completion-is-low-priority — proposed, not yet
  agreed (see `docs/05-ideas-to-discuss.md`, IDEA-01).

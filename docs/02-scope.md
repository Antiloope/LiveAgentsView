# Scope

> **Status:** defined 2026-09-01. Distilled from the 2026-09-01 inbox block.

## Product posture: piloted only

> **2026-09-02:** this section originally described two coexisting classes, **adopted**
> (sessions launched natively, reporting in read-only via hooks) and **piloted**
> (launched from LiveAgentsView, full control). Adopted mode is now removed entirely —
> see [03-decisions.md](03-decisions.md) 2026-09-02 "Adopted mode and hooks ingestion
> are removed entirely; piloted-only posture". LiveAgentsView only ever tracks sessions
> it launched itself.

Every session LiveAgentsView shows is one it launched itself, as a child process over
bidirectional `stream-json`. You can answer questions, approve/deny permissions, and
cancel — without leaving the dashboard. There is no read-only class for sessions started
outside it; those are not tracked at all.

Building on top of the vendor CLIs this way keeps the door open to a fuller frontend
later, but that is not the goal of this scope.

## What it does

- Launches and aggregates coding-agent sessions into one dashboard.
- Shows repository, worktree, branch, provider and normalized state per session.
- Surfaces an **attention queue**: what actually needs the human, prioritized.
- Notifies on the events that earn an interruption.
- Lets the user act on any session directly — answer, approve/deny, interrupt, cancel,
  resume — without leaving the dashboard.
- Shows recently finished agents.
- Persists enough state and history to survive restarts.

### Integration surface

> **2026-09-02:** this originally described three fidelity levels (Driver, Hooks,
> Tailing) shown per session. Hooks and Tailing are gone along with adopted mode — see
> [03-decisions.md](03-decisions.md) 2026-09-02. Driver is now the only surface.

**Driver** — bidirectional `stream-json` against the vendor CLI. Full control: every
event out, realtime input in, permission requests answerable from the dashboard. This is
the only integration surface; a session with no driver process is a session
LiveAgentsView does not know about.

### Providers in the MVP

Claude Code and Cursor have a working piloted (driver-fidelity) adapter.

> **2026-09-02:** Codex has no driver adapter (out of scope, see
> [piloted-mode-mvp](sdd/specs/piloted-mode-mvp.md)) and, with hooks removed, no other
> way to appear in the app either — it has no representation in LiveAgentsView until a
> driver adapter is built for it. Not dropped as a target provider, just not usable yet.

Known constraint: Cursor's piloted adapter auto-approves every tool call (`--force`/
`--yolo`) — its CLI has no live permission-approval channel, confirmed in
[piloted-mode-mvp](sdd/specs/piloted-mode-mvp.md).

### Attention

"An agent finished" is **low-priority attention**: it enters the queue grouped and
without a loud notification, rather than being a mere state change.

End-of-turn ambiguity (finished vs. asked you something) is resolved with **rules over
the last message, behind a pluggable classifier interface**, so the implementation can be
swapped without touching the state engine. Chosen over an LLM classifier to preserve the
"no API keys needed" property.

The full priority taxonomy is proposed but not yet agreed — see
[IDEA-01](05-ideas-to-discuss.md).

## What it does not do

Not in this scope:

- Agent orchestration or automatic task assignment
- Task management or workflows
- Agent memory or MCP management
- Team collaboration or enterprise governance
- Sophisticated analytics
- Replacing the underlying IDE or CLI

## Explicit boundaries

**LLM credentials are never managed.** Executing the vendor CLIs as subprocesses inherits
the user's existing login. LiveAgentsView does not see, store or ask for API keys.

**Filesystem permissions are delegated to the underlying agent.** Claude Code has
permission modes, allow/deny rules and `--add-dir`; Codex has sandbox modes and approval
policy. LiveAgentsView surfaces the prompt and routes the answer. It does not build a
sandbox of its own.

**A local server that can approve permissions is remote code execution.** Bind to
`127.0.0.1` by default; exposing it to other devices is an explicit, separate opt-in.

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
bidirectional `stream-json`. You can answer its questions, interrupt it and stop it —
without leaving the dashboard. There is no read-only class for sessions started outside
it; those are not tracked at all.

Building on top of the vendor CLIs this way keeps the door open to a fuller frontend
later, but that is not the goal of this scope.

## Vocabulary

> **2026-09-03:** added with the [03-decisions.md](03-decisions.md) 2026-09-03
> "Vocabulary: character, race, class, quest" entry.

The words the interface uses do not name the engine behind a session:

- A **character** is the durable thing the user creates and talks to. It lives many
  quests and returns to camp between them.
- Its **race** is the engine behind it (Claude Code, Cursor, Codex). It cannot change.
- Its **class** is the model it runs. It can change in principle.
- Its **territory** is where it works: its own worktree, or a shared directory.
- A **quest** is what the user asks it for. It is not a modelled object — see
  [03-decisions.md](03-decisions.md) 2026-09-03 "A quest is not a modelled object".
- **Camp** is where a character is when it is not out on a quest.

"Session" stays an internal word for the provider-side conversation a character owns.

## What it does

- Launches and aggregates characters into one dashboard.
- Shows repository, territory, branch, race, class and normalized activity per character.
- Surfaces an **attention queue**: what actually needs the human, prioritized.
- Notifies on the events that earn an interruption.
- Lets the user act on any character directly — answer it, interrupt it, stop it — without
  leaving the dashboard.
- Shows which characters came back with news the user has not read yet.
- Persists enough state and history to survive restarts.
- Lets the user give a character its own worktree, administered by LiveAgentsView, or run
  it on a directory exactly as it is.
- Lets the user archive a character in any activity, which also sends it to sleep and
  frees the memory it held, reversibly, from a dedicated archived view; and dismiss one
  for good.

### Integration surface

> **2026-09-02:** this originally described three fidelity levels (Driver, Hooks,
> Tailing) shown per session. Hooks and Tailing are gone along with adopted mode — see
> [03-decisions.md](03-decisions.md) 2026-09-02. Driver is now the only surface.

**Driver** — bidirectional `stream-json` against the vendor CLI. Every event out,
realtime input in. This is the only integration surface; a character LiveAgentsView cannot
drive is a character it does not know about.

### Providers in the MVP

Claude Code and Cursor have a working piloted (driver-fidelity) adapter.

> **2026-09-02:** Codex has no driver adapter (out of scope, see
> [piloted-mode-mvp](sdd/specs/piloted-mode-mvp.md)) and, with hooks removed, no other
> way to appear in the app either — it has no representation in LiveAgentsView until a
> driver adapter is built for it. Not dropped as a target provider, just not usable yet.

> **2026-09-03:** this said Cursor's adapter auto-approving every tool call was a known
> constraint of that race specifically. Every race now runs auto-approving — see
> [03-decisions.md](03-decisions.md) 2026-09-03 "Permission management is dropped; every
> race runs auto-approving". It is no longer a difference between races.

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

**Permissions are not mediated at all.** Every character runs auto-approving, whatever
its race — see [03-decisions.md](03-decisions.md) 2026-09-03. LiveAgentsView neither asks
the user to approve tool calls nor sandboxes anything. A character's own territory is
containment, not a sandbox: it keeps a character out of the directory the user is working
in, and stops nothing else.

> **2026-09-03:** this previously read "filesystem permissions are delegated to the
> underlying agent (...) LiveAgentsView surfaces the prompt and routes the answer." It no
> longer surfaces anything. Building a permission layer later is
> [IDEA-11](05-ideas-to-discuss.md).

**A local server that can launch a coding agent is remote code execution.** More so now
that every character it launches auto-approves. Bind to `127.0.0.1` by default; exposing
it to other devices is an explicit, separate opt-in.

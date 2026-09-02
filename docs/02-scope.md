# Scope

> **Status:** defined 2026-09-01. Distilled from the 2026-09-01 inbox block.

## Product posture: observer + opt-in pilot

Two classes of session coexist in the same interface:

| Class | How it arrives | What you can do |
|---|---|---|
| **Adopted** | The user launched it natively; it reports in via hooks | Read only: state, attention queue, notifications, open in the terminal |
| **Piloted** | Launched from LiveAgentsView as a child process over `stream-json` | Answer questions, approve/deny permissions, cancel — without leaving the dashboard |

Building the piloted mode on top of the vendor CLIs keeps the door open to a fuller
frontend later, but that is not the goal of this scope.

## What it does

- Detects local coding-agent sessions and aggregates them into one dashboard.
- Shows repository, worktree, branch, provider and normalized state per session.
- Surfaces an **attention queue**: what actually needs the human, prioritized.
- Notifies on the events that earn an interruption.
- Lets the user jump to the originating session (adopted) or act on it directly (piloted).
- Shows recently finished agents.
- Persists enough state and history to survive restarts.

### Integration surfaces

Three fidelity levels, used together, with the level shown per session in the UI:

1. **Driver** — bidirectional `stream-json` against the vendor CLI. Full control.
2. **Hooks** — `~/.claude/settings.json`, `~/.codex/config.toml` `notify`. Read-only push.
3. **Tailing** — vendor JSONL transcripts. Last resort, retroactive, no control.

Transcripts alone cannot answer "is it waiting right now"; the attention signal comes
from hooks or from driving the process.

### Providers in the MVP

Claude Code, Codex and Cursor.

Known constraint: Cursor's IDE agent exposes no local hook surface, so supporting Cursor
means supporting the `cursor-agent` CLI — a different workflow from the one used today.

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

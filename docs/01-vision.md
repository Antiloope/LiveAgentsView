# Vision

> **Status:** defined 2026-09-01. Distilled from the 2026-09-01 inbox block.

## What LiveAgentsView is

An attention layer for AI coding agents. A local-first mission control that sits **above**
Claude Code, Codex and Cursor rather than replacing them.

> Run many coding agents. Only pay attention when one needs you.

## Who it is for

Developers who run several coding agents in parallel across multiple repositories and
worktrees. For them the problem is no longer launching agents — it is keeping track of
them and knowing when human attention is actually required.

v1 is built for its own author's workflow (dogfooding), but every piece is built so it
can eventually become a product.

## Why it exists

Running agents in parallel means juggling terminal tabs, IDEs, agent CLIs, permission
prompts, agents that finished unnoticed, and agents stuck in a loop. The user should be
able to answer, at a glance:

1. What are all my agents doing?
2. Which ones are working?
3. Which ones are waiting for me?
4. Which ones finished?
5. Which ones failed or look stuck?
6. What happened recently?
7. Can I intervene without hunting down the original terminal?

The validation question is not "does the dashboard look good?" It is:

> **Does this reduce the mental effort of supervising many agents?**

## Principles

### 1. Attention over activity

Do not make the user monitor agents continuously. The primary question is
*"do I need to do anything?"* — not *"what is happening?"*

### 2. Provider agnostic

The product does not depend conceptually on Claude, Codex or Cursor. Providers are
adapters behind a canonical event model.

### 3. Local-first

Local data, no required cloud account, no provider API keys where native CLIs and hooks
can be used. The user keeps control of their source code and their sessions. A remote
layer may come later.

### 4. Do not replace existing tools

LiveAgentsView drives the real Claude Code, Codex and Cursor CLIs as subprocesses rather
than reimplementing them — no rebuilt plan mode, autocomplete, slash commands or diff
review, and no falling behind every upstream release.

> **2026-09-02:** until this point, that also meant tracking sessions launched natively
> outside LiveAgentsView (read-only, via hooks), with "open in the terminal" as a
> first-class fallback for them. That class is gone — see
> [03-decisions.md](03-decisions.md) 2026-09-02 "Adopted mode and hooks ingestion are
> removed entirely; piloted-only posture". Every session shown is one LiveAgentsView
> launched itself; there is no fallback for sessions started elsewhere because those are
> no longer tracked at all.

### 5. Glanceable

The state of 10–20 agents should be understandable in seconds.

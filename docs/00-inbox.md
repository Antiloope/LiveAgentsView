# Inbox — raw dump

Everything lands here first, unsorted. Nothing in this document is a firm definition
until it is distilled into the right doc.

Format: one block per session or conversation, with a date. Already processed items are
marked ~~strikethrough~~ or moved, so it is always clear what still needs distilling.

---

## ~~2026-09-01 — Repository bootstrap~~

**Status: distilled 2026-09-01.**

- ~~Created the `LiveAgentsView` repo with a structure inspired by sincro.~~
- ~~It is open source (MIT).~~
- ~~There is still no product definition or concrete apps.~~
- ~~Missing: vision, scope, stack, and which deployables go in `apps/`.~~

Vision and scope defined in the block below. Stack decided (Go + SQLite).
`apps/` still empty — pending the first spec.

---

## 2026-09-01 — Product definition session

**Status: distilled 2026-09-01.** Vision → [01-vision.md](01-vision.md), scope →
[02-scope.md](02-scope.md), agreed decisions → [03-decisions.md](03-decisions.md),
unresolved items → [04-open-questions.md](04-open-questions.md), unagreed proposals →
[05-ideas-to-discuss.md](05-ideas-to-discuss.md).

Source: `multi-agent-coding-mission-control.md` (external doc brought by Rodrigo) plus
a working session to ground it technically.

### The idea as it arrived

Local-first mission control / attention layer for coding agents. The user runs many
agents in parallel across repos and worktrees; the problem is not launching them, it is
knowing when a human is actually needed. Positioning: "run many agents, only pay
attention when one needs you." Central abstraction: the Attention Queue. Normalized
states: WORKING / WAITING / BLOCKED / DONE / FAILED / IDLE. Principles: attention over
activity, provider agnostic, local-first, do not replace existing tools, glanceable.
Explicitly out of MVP: orchestration, task management, memory, MCP management, teams,
analytics, governance.

### What was verified on the machine (2026-09-01)

- Claude Code CLI is installed and highly programmable: `-p --output-format stream-json`
  plus `--input-format stream-json` gives a bidirectional JSON channel — every event out,
  realtime input in. Also `--include-hook-events`, `--permission-mode`, `--allowedTools`,
  `--add-dir`, `--session-id`, `--resume`, `--fork-session`, `--bg` + `claude agents`,
  `-w/--worktree`, `--tmux`, `--max-budget-usd`.
- Claude Code transcripts live at `~/.claude/projects/<slug>/<sessionId>.jsonl`. A real
  transcript contains `assistant`, `user`, `attachment`, `system` records — it says what
  was said, **never whether the agent is waiting right now**. Attention cannot be derived
  from transcripts alone.
- `~/.claude/settings.json` currently has **no hooks**. The machine emits no attention
  signal today; it has to be created.
- Codex is installed as an app (`~/.codex/ipc/ipc.sock`), not as a CLI in PATH. Sessions
  at `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` plus `session_index.jsonl`.
- **Conflict found:** `~/.codex/config.toml` already sets
  `notify = [".../SkyComputerUseClient", "turn-ended"]`. `notify` takes a single program,
  so an installer that overwrites it breaks the user's existing setup. It has to chain.
- Cursor.app is installed but `~/.cursor/agents` is empty and `cursor-agent` is not in
  PATH. The IDE agent exposes no local hook surface: supporting Cursor in practice means
  supporting the `cursor-agent` CLI, which is a different workflow from the one Rodrigo
  uses today.
- No Gemini CLI, no OpenCode.

### Three integration surfaces (descending fidelity)

1. **Driver** — `claude -p --input-format/--output-format stream-json`,
   `codex exec --json` / app-server, `cursor-agent --output-format stream-json`.
   LiveAgentsView is the frontend. Full events, full control, can answer.
   Requires the session to be launched by LiveAgentsView.
2. **Hooks** — `~/.claude/settings.json` (`Notification`, `Stop`, `SessionStart`,
   `UserPromptSubmit`), `~/.codex/config.toml` `notify`. Sessions the user launched
   natively report in. Read-only, zero intrusion.
3. **Tailing** — the JSONL transcripts. Last resort, retroactive, no control.

Not mutually exclusive: they are fidelity levels, and the UI should show which level it
has on each session.

### The three product postures discussed

- **A — Panel:** observer only. Hooks + tailing, notify and jump to the session. Cannot
  answer. ~2-3 weeks, low risk, medium ceiling.
- **B — Panel + Pilot:** A, plus sessions launched from LiveAgentsView run as child
  processes over `stream-json`, so you can answer, approve, and cancel from the
  dashboard. Two session classes coexist: *adopted* (read-only) and *piloted* (full
  control). ~6-8 weeks, medium risk, high ceiling.
- **C — Agent IDE:** total replacement, never open Claude Code again. Requires rebuilding
  plan mode, autocomplete, slash commands, image paste, diff review. Falls behind every
  upstream release. Contradicts principle 4 of the original doc.

B and C use the same technical mechanism. The difference is how much of the native UI
you commit to rebuilding, and whether "open in the terminal" exists as a feature.

### Credentials and permissions

- **LLM credentials: not managed.** Executing the vendor CLIs as subprocesses inherits
  the user's existing login (subscription OAuth in the keychain). LiveAgentsView never
  sees an API key. Going direct-to-API would mean reimplementing the whole agent.
- **Filesystem permissions: delegated.** Claude Code already has permission modes,
  allow/deny rules and `--add-dir`; Codex has sandbox modes and approval policy.
  LiveAgentsView surfaces the prompt and routes the answer — it does not build a sandbox.
- **New risk the original doc does not cover:** a local server that can approve
  permissions is effectively remote code execution on the machine. Bind to `127.0.0.1`,
  token auth, tunnel exposure as an explicit separate opt-in.

### Why Docker does not work for the runtime

The daemon has to spawn `claude` on the host, with the host keychain, repos, git config
and worktrees. Containerizing that loses host auth and breaks worktrees. Docker stays for
developing and testing LiveAgentsView itself. This contradicts the current AGENTS.md rule
("the only thing installed on the machine is Docker") and needs an explicit decision.

### Attention events — the core of the product

An attention event is the moment an agent goes from "working, ignore it" to "needs you
now". The whole product is a classifier: out of hundreds of events per hour across ten
agents, decide which ones earn an interruption. Too many and the user turns it off in two
days; too few and they stop trusting it and go back to checking terminals one by one.

Three classes of signal by reliability:

1. **Deterministic** — the agent literally cannot continue. Permission request. Zero
   ambiguity. Arrives via the `Notification` hook, or explicitly in the stream when
   piloted. Build on this first.
2. **End of turn** — the agent stopped talking. Claude Code's `Stop` and Codex's
   `turn-ended` fire identically whether it finished, asked a question, or gave up.
   Requires interpreting the last message.
3. **Inferred** — nothing for N minutes, or looping. "Stuck". Pure heuristic, always
   noisy (an 8-minute `npm test` is legitimate). Ships last, as a "suspicious" list, not
   an alarm.

Agent proposals raised in the session (not decided, see 05):

- Attention is not a property of the agent alone: `attention = agent state × is a human
  watching that session`. An agent finishing while you sit in that terminal is not an
  event. In piloted mode nobody is watching by definition, so every end of turn counts.
- Attention items must auto-resolve when handled by any route — answering in the terminal
  should clear the queue item, which the `UserPromptSubmit` hook gives for free. Stale
  queues are the number one reason these dashboards get abandoned.
- Throttling: an agent asking permission 40 times in a bash loop is one item, not forty.
- Priority taxonomy P0 blocked / P1 decision / P2 done / P3 suspicious / failed.

### What Rodrigo decided in this session

- Posture **B**, leaving the door open toward C later.
- **Go** as the language, **SQLite** for storage.
- MVP providers: **Claude Code, Codex and Cursor**.
- Audience: **dogfooding, but building things that can eventually become a product**.
- End-of-turn disambiguation: **rules, behind a pluggable interface**, so the classifier
  can be swapped later without touching the state engine. Chosen over an LLM classifier
  to keep the "no API keys needed" property.
- **"An agent finished" is low-priority attention**: enters the queue grouped and without
  a loud notification, rather than being a mere state change.
- Architecture shape, as Rodrigo put it: a single binary for mac/linux that starts a
  daemon, reachable over CLI or HTTP because it serves a frontend, and eventually
  reachable from another device over Tailscale from a phone. Confirmed, with the added
  detail that the frontend is embedded in the binary and the daemon must run as a user
  service so hooks are not lost when the dashboard is closed.

---

## 2026-09-02 — Follow-ups raised after installing the native service

**Status: distilled 2026-09-02.** Moved to
[05-ideas-to-discuss.md](05-ideas-to-discuss.md) as IDEA-07 through IDEA-10.

Source: Rodrigo asking what's actually installed on his machine after
[native-host-runtime](sdd/specs/native-host-runtime.md) (`done`, pending `validate`) put a
real launchd service in place. Talking through "how do I get a new binary in when I change
code" and "how would I install this on another machine" surfaced four gaps, none decided:

- No `lav version` — the running binary carries no visible build identity, so there is no
  way to tell whether a reinstall actually picked up the latest code short of doing it.
- No `lav service uninstall` / general uninstall path — `lav-service-install.sh` writes
  launchd/systemd config and provider hooks with nothing symmetric to remove them.
- Dashboard UX still IP:port (`127.0.0.1:8420`) rather than a friendlier local name.
- No prebuilt binary distribution — installing today means cloning the repo and having
  Docker available to compile; there's no host to fetch a prebuilt mac/linux binary from,
  and no first-run console wizard (e.g. "run as a systemd/launchd service, or start it by
  hand?").

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

---

## 2026-09-02 — Dashboard scope narrowing + piloted restart continuity

**Status: distilled 2026-09-02.**

Source: Rodrigo reviewing the running dashboard (this machine's real launchd service),
asking to drop hooks-only (adopted) sessions from display — "copy path"/"open terminal"
and the drawer info are not useful without a chat channel to act on them.

### Whether adopted sessions can be chatted with instead of hidden

Checked against `internal/pilot/pilot.go`, `daemon/server.go`'s `handleHook`, and both
mode specs: an adopted session reports in via hooks, a one-way push with no channel back
into the process — LiveAgentsView never holds a stdin handle to a session it did not
launch itself. There is no way to start chatting with an already-running adopted session
short of relaunching it as piloted. Confirmed from the code, not just the spec text.

### What was verified on the machine (2026-09-02)

Motivated by wanting a piloted session's live process to survive a `lav` restart instead
of being killed outright (`internal/pilot.Manager` never persists a process handle —
today's restart behavior is `ReconcileOnStartup` flipping it to idle and offering
Resume), live-tested against this machine's real, authenticated `claude` (2.1.258,
logged in via claude.ai, confirmed with `claude auth status`) and `agent` (Cursor,
confirmed with `agent status`) CLIs whether either provider's native background/
persistent-session feature could keep a piloted process alive independent of `lav`:

- `claude --bg -p --input-format stream-json --output-format stream-json ...` —
  rejected outright: `"--bg and --print conflict: --print never starts the interactive
  session that \`claude agents\` attaches to, so the job would be unattachable."` `--bg`
  is for the interactive terminal/agent-view session only.
- `agent persist -p --output-format stream-json --force --trust ...` — rejected
  outright: `"Persistence requires an interactive Linux or macOS terminal."`
- Both tested for real (a live process launch attempt, not just reading `--help`);
  confirmed no stray session/process was left behind afterward (`claude agents --json`,
  `ps`).
- This corrects the 2026-09-01 block above, which listed `--bg`/`--tmux` only as
  available flags without confirming they compose with the stream-json driver protocol
  — they do not, structurally: both are built for a human re-attaching to a terminal,
  not a second machine-readable channel. The same mismatch applies to plain tmux
  ([IDEA-06](05-ideas-to-discuss.md)) — capturing/injecting through `send-keys`/
  `capture-pane` is built for rendered terminal text, not line-oriented JSON.

### What Rodrigo decided in this session

- **Revised mid-session, escalated from "hide" to "remove entirely":** LiveAgentsView no
  longer has an adopted/hooks concept at all. It only ever knows about sessions it
  launched itself (piloted). This means removing, not just hiding: `internal/ingest`,
  the `/hooks/claude-code` / `/hooks/codex` / `/hooks/cursor` routes, `lav init` and
  `internal/installer`, and — since this real machine already has real hooks installed
  from a previous real `lav init` run — actually uninstalling those hooks from
  `~/.claude/settings.json`, `~/.codex/config.toml` and `~/.cursor/hooks.json`, not just
  deleting the code that wrote them. Rodrigo explicitly chose the "remove the code and
  uninstall the real hooks" option over "remove the code but leave the already-installed
  hooks alone."
- Piloted sessions should survive a `lav` daemon restart without losing whatever turn
  was in progress, via a supervisor LiveAgentsView builds itself (process detached from
  `lav`'s own lifecycle, stdio moved off in-memory pipes onto something that survives
  the daemon's own exit, reconnect on startup) — chosen over the smaller "auto-resume on
  startup" alternative and over leaving today's behavior (Resume, but the in-flight turn
  is always lost) as-is. No CLI-native shortcut for this exists (see above).

Distilled into [03-decisions.md](03-decisions.md) (amended/new entries),
[02-scope.md](02-scope.md) (rewritten posture section),
[01-vision.md](01-vision.md) (principle 4 corrected),
[05-ideas-to-discuss.md](05-ideas-to-discuss.md) (IDEA-06 and IDEA-08 annotated), and
[sdd/specs/piloted-only-mode.md](sdd/specs/piloted-only-mode.md).

---

## 2026-09-02 — Archive a session

**Status: distilled 2026-09-02.**

Source: Rodrigo, in chat — "quiero poder borrar un agente, osea hoy los veo a todos ahí
en el campamento, pero me gustaría tener un boton para archivarlo digamos y dejar de
verlo si no está ejecutando nada digamos." The camp view (`apps/web`) has no way to stop
showing a session once it exists — every known session renders as a party member forever,
finished ones included.

Triaged as spec-worthy (crosses the session model, SQLite, the HTTP API and the frontend
camp view), not a "just go." Three product questions asked and answered before writing
the spec:

- **Persistence:** archived state is stored in SQLite as part of the session row (not
  just a browser-local hide), so it survives a `lav` restart and is the same from any
  device pointed at the same daemon.
- **Reversibility:** archiving is reversible. A dedicated "Archived" view lists archived
  sessions with an unarchive action, rather than a one-way hide.
- **Eligibility:** a session can be archived in any state except `working` — `idle`,
  `waiting`, `blocked`, `done` and `failed` are all archivable, not only the terminal
  `done`/`failed` states. `working` is excluded so a session mid-turn can't be hidden out
  from under itself.

Distilled into [03-decisions.md](03-decisions.md) (new entry), [02-scope.md](02-scope.md)
("What it does" bullet), and
[sdd/specs/archive-session.md](sdd/specs/archive-session.md).

---

## 2026-09-03 — Character model, territory, lifecycle and dropping permissions

**Status: distilled 2026-09-03.**

Source: Rodrigo, in chat, after a code + live-service review of the running daemon
(`127.0.0.1:8420`) that started from one symptom — "recién creé un agente con sonnet y
quedó on quest desde el primer momento, sin que le pase ninguna tarea" — and from
noticing that creating a Cursor one demands a first prompt while a Claude one does not.

### What the review found (raw, before any definition)

Confirmed live against the running service, not read-only from the code:

- A Claude Code session launched with no prompt is upserted as `working` unconditionally
  and never leaves that state: the CLI emits nothing at all until its first message, so
  no `result` line ever arrives. The real session on this machine had a 0-byte
  transcript and `updated_at == created_at` hours later.
- That same session cannot be archived (`working` is excluded), and there is no delete,
  so it cannot be removed from the camp view by any route.
- No CSRF or Origin check on the API, and the JSON body is decoded regardless of
  `Content-Type`. Verified live: a `POST /api/piloted/sessions` with
  `Content-Type: text/plain` and `Origin: https://evil.example` was parsed normally and
  only failed on field validation. Any web page open in the browser can therefore launch
  an auto-approving agent on this machine; binding to `127.0.0.1` does not prevent it.
  Unrelated to the redesign below — logged here so it is not lost.
- `git checkout` runs in the user's own chosen directory when a branch is picked.
- A pending permission request lives only in the daemon's memory; a daemon restart
  strands the child process waiting for an answer that can no longer be given.
- Liveness lives in three places (SQLite `state`, the manager's in-memory `running`, the
  real process) and is reconciled only at daemon startup. A runner killed abruptly leaves
  a session reading `working` forever.
- Measured: a Claude character that had never received a message held 129 MB RSS and
  ~0.5% CPU after 3h27m, plus 10 MB for its runner.
- Smaller ones: HP/MP bars are seeded random numbers presented as telemetry; the empty
  state still tells the user to run `lav init` and start sessions natively, both removed
  in piloted-only mode; a live event arriving while the drawer's history is still loading
  is overwritten; `Cancel` on a dead session leaves `stoppedByUser` set; `Interrupt` is
  offered when there is no turn to interrupt.

### What Rodrigo decided in this session

- **Vocabulary, provider-neutral and in the lore.** `character` replaces "agent"/
  "session" as the word for the durable thing the user creates and talks to. `quest`
  replaces "task". The engine behind a character is its **race** — "algo duro que no se
  puede cambiar" — and the model is its **class**: "a futuro capaz a un agente se le
  puede cambiar la clase (cambiar para que use un modelo mas chico por ejemplo), pero no
  se podría cambiar de raza a un character". The engine is never named as its own
  concept in the interface.
- **DONE is not a state.** Raised by Rodrigo: "El estado DONE no se si existiría o es
  IDLE de nuevo luego de terminar con algo que indique que terminó. Osea todos irían a
  idle solo cuando los creamos, después todos quedan en idle eventualmente cuando
  terminan bien no?" Agreed: what DONE actually describes is whether the *user* has seen
  the result, not what the character is doing, so it becomes an unread mark and the
  activity states collapse.
- **Two axes.** What a character is doing, and whether it is awake, are separate. The
  user never manages the second one: talking to an asleep character wakes it. This is
  what makes Cursor (a fresh process every turn) and Claude Code (a process that stays
  resident) behave identically from the outside.
- **Territory.** A character works either in its own worktree that LiveAgentsView
  administers, or in the directory exactly as it is. Own worktree is the default. Chosen
  from the two options offered: worktrees under `~/.liveagentsview/worktrees/`, removed
  on dismissal only when clean.
- **Archiving sends to sleep.** Chosen from three options offered: archiving stops the
  process, keeps history and territory, and is allowed in any state including working.
- **Permissions are dropped for now.** "Estoy pensando que hagamos para esta versión lo
  mismo con Claude de mandarlo en modo automatico para que no pida permisos y funcione
  igual que cursor y nos sacamos la gestión de permisos por ahora de encima. Luego en el
  futuro podríámos ver si necesitamos implemetnar esto o no."
- **A quest is not a first-class object** for now — the chat is enough, and modelling
  tasks would cross the "no task management or workflows" scope boundary.

Distilled into [03-decisions.md](03-decisions.md) (seven new entries, two of them
superseding earlier ones), [02-scope.md](02-scope.md),
[04-open-questions.md](04-open-questions.md) (Q-10, Q-11),
[05-ideas-to-discuss.md](05-ideas-to-discuss.md) (IDEA-11, IDEA-12) and
[sdd/specs/character-model-redesign.md](sdd/specs/character-model-redesign.md).
The CSRF finding above is not part of that spec; it has its own,
[sdd/specs/local-api-hardening.md](sdd/specs/local-api-hardening.md).

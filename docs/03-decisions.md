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

## 2026-09-02 — Cursor piloted sessions auto-approve, no live permission gate

**Who:** Rodrigo
**Decision:** Cursor's piloted adapter launches every session with `--force`/`--yolo`
(auto-approve every tool call) instead of offering the approve/deny control piloted mode
gives Claude Code. Each user message is a new one-shot `agent -p --output-format
stream-json` invocation chained via `--resume`/`--continue`, not a persistent stdin
channel.
**Rationale:** `agent --help` has no `--input-format` flag at all (Claude Code has one) and
`--output-format stream-json` only works with `--print`, a single request/response
invocation, not a session you keep talking to. Confirmed live: running `agent -p
--output-format stream-json --trust` (no `--force`) against a shell tool call did not pause
for an approval — it silently rejected the command and the agent worked around it. There is
no channel for an external supervisor to approve a specific pending tool call. The
alternative, restricting Cursor piloted sessions to `--mode plan`/`ask` (read-only), cannot
complete tasks, defeating the point of piloting it — auto-approving is the accepted
trade-off for Cursor specifically, the same posture adopted mode already takes by
delegating permissions to the underlying agent.

## 2026-09-02 — Adopted mode and hooks ingestion are removed entirely; piloted-only posture

**Who:** Rodrigo
**Decision:** LiveAgentsView drops the adopted/hooks concept completely rather than
just hiding it from the dashboard. It only ever tracks sessions it launched itself
(piloted). This removes, not merely hides: `internal/ingest` (the per-provider hook
parsers), the `/hooks/claude-code` / `/hooks/codex` / `/hooks/cursor` HTTP routes,
`lav init` and `internal/installer` (the hook-merge machinery), and any already-persisted
hooks-fidelity session rows in SQLite. Because a previous real `lav init` run already
wrote LiveAgentsView's hooks into this machine's actual `~/.claude/settings.json`,
`~/.codex/config.toml` and `~/.cursor/hooks.json`, those real hooks are also uninstalled
as part of this — not just the code that wrote them — non-destructively, the same way
`lav init` itself only ever merged rather than overwrote.
**Rationale:** Superseded the narrower "just stop displaying adopted sessions" version
of this decision (written earlier the same session) after Rodrigo pointed out that
without a channel back into an adopted session, keeping the ingestion pipeline and
`lav init` around is dead weight pretending to do something — "la app solo funciona con
agentes que puede usar y nada más." Confirmed against `internal/pilot` and
`daemon/server.go` that no channel to chat with an adopted session is achievable short
of relaunching it as piloted, so nothing of substance is lost by removing the concept —
the user already has direct access to the terminal they started an adopted session in
themselves. This supersedes the two-class framing from the 2026-09-01 "Posture: observer
+ opt-in pilot ('B')" decision — LiveAgentsView is piloted-only going forward — and the
[adopted-mode-mvp](sdd/specs/adopted-mode-mvp.md) acceptance item that all known
sessions are listed "regardless of fidelity," intentionally, not as a regression. Also
resolves the hooks-removal half of [IDEA-08](05-ideas-to-discuss.md) (its
`lav service uninstall` half is unrelated and stays open). A real consequence worth
naming: Codex has no driver/piloted adapter (out of scope per
[piloted-mode-mvp](sdd/specs/piloted-mode-mvp.md)), so until one is built, Codex has no
representation in the app at all — it drops out of the MVP-providers decision in
practice, not just in scope for this change.

## 2026-09-02 — No CLI-native background/persistent-session feature fits piloted mode

**Who:** Rodrigo
**Decision:** Neither Claude Code's `--bg`/`claude agents` background-session feature
nor Cursor's `agent persist` can be used to keep a piloted driver process alive across a
`lav` restart. Restart continuity, if built, needs a supervisor LiveAgentsView
implements itself (detach the process from the daemon's own lifecycle, move its stdio
off in-memory pipes), not a provider CLI flag.
**Rationale:** Confirmed live against this machine's real, authenticated `claude` and
`agent` CLIs: `claude --bg` explicitly refuses to combine with `-p`/`--print` ("the job
would be unattachable"), and `agent persist` refuses to run without an interactive
terminal. Both features are built around a human re-attaching to an interactive
terminal/agent view, not a second machine-readable `stream-json` channel — incompatible
with the headless bidirectional protocol `internal/pilot/claude.go` and `cursor.go`
already depend on for structured transcript parsing and live permission approval. This
corrects the 2026-09-01 inbox note that listed `--bg`/`--tmux` only as available flags,
without confirming they compose with `-p`/stream-json — they do not. Also closes
[IDEA-06](05-ideas-to-discuss.md)'s tmux-owned-sessions proposal as a path to restart
continuity specifically: tmux has the identical structural mismatch (built for
interactive TTY reconnect, not line-oriented JSON piping between two programs).
IDEA-06's narrower "let a human attach a real terminal to a piloted session" idea is
untouched by this and stays open, unrelated to this decision.

## 2026-09-02 — Sessions can be archived, reversibly, in any non-working state

**Who:** Rodrigo
**Decision:** A session can be archived from the dashboard so it stops appearing in the
camp view. Archived is persisted server-side (SQLite), not a browser-local hide, and is
reversible: a dedicated "Archived" view lists archived sessions with an unarchive action.
A session can be archived in any state except `working` (`idle`, `waiting`, `blocked`,
`done`, `failed` are all eligible) — only an in-progress turn is protected from being
hidden out from under itself.
**Rationale:** "Hoy los veo a todos ahí en el campamento" — every known session renders
forever, finished ones included, with no way to reduce clutter. Persisting server-side
keeps the property that the dashboard shows the same thing from any device against the
same daemon, matching the existing "persists enough state and history to survive
restarts" scope item. Reversible-by-default over a one-way hide because a stray click
should not be able to lose track of a session with no recovery path. Restricting only
`working` (not narrowing to `done`/`failed`) because a session that is alive but not
actively turning (idle, waiting on the user, blocked on a permission) is just as much
clutter as a finished one, and there is nothing unsafe about hiding it — archiving never
touches the underlying process either way.

## 2026-09-01 — Cursor's adapter is built alongside Claude Code and Codex from the start

**Who:** Rodrigo
**Decision:** Cursor stays in MVP scope and its `cursor-agent` adapter is built in the
same first spec as Claude Code and Codex, rather than deferred to a later spec.
**Rationale:** Confirms the existing MVP-providers decision. Cursor's known constraint
(no local hook surface, `cursor-agent` CLI only) is accepted as a limitation to design
around, not a reason to postpone it.
**Closes:** Q-09.

## 2026-09-03 — Vocabulary: character, race, class, quest

**Who:** Rodrigo
**Decision:** The user-facing vocabulary is provider-neutral and stated in the camp lore.
A **character** is the durable thing the user creates and talks to; it lives many
**quests** over its life and returns to **camp** between them. Its **race** is the engine
behind it (Claude Code, Cursor, Codex) and cannot be changed. Its **class** is the model
it runs (Opus/Sonnet/Haiku, or any model from Cursor's catalog) and is changeable in
principle. Its **territory** is where it works. The engine is never named as a concept of
its own in the interface: it appears as the character's race, on its banner and in its
details. "Session" remains an internal word for the provider-side conversation a
character owns.
**Rationale:** race and class map onto real constraints instead of decorating them. A
character's conversation lives in its engine's own store (its session id and history), so
it can never become another engine's character — immutable because the world makes it
immutable. The model, by contrast, is a flag passed when a process starts, so it can
change the next time the character wakes. Naming the engine as a race also removes the
last place the interface reads as Claude-only or Cursor-only, which principle 2 of
[01-vision.md](01-vision.md) asks for.
**Consequence:** the party sprite must be determined by race, not by class as it is
today — changing a character's model must not turn it into a different creature. Which
archetypes belong to which race is [Q-10](04-open-questions.md).

## 2026-09-03 — Two axes: what a character is doing, and whether it is awake

**Who:** Rodrigo
**Decision:** A character's condition is two independent things, not one field.
**Activity** is what it is doing: `ready` (at camp with nothing to do), `working` (out on
a quest), `waiting` (came back with a question), `failed` (came back badly).
**Presence** is whether a process is alive for it: awake or asleep. Presence is not
something the user manages — talking to an asleep character wakes it, so there is no
resume action in the interface. `done` is removed as a state and becomes an orthogonal
**unread** mark on the character, set when a quest ends without a question and cleared
when the user reads its transcript. `idle` and `blocked` are removed. This supersedes the
canonical WORKING/WAITING/BLOCKED/DONE/FAILED/IDLE model from the 2026-09-01 decision
"Canonical event/state model, validated against the 3 providers".
**Rationale:** the single field was answering two questions at once, which is why a
character that is alive but has nothing to do had no way to be described and was recorded
as `working`. Splitting them is also what normalizes the engines: Cursor starts a fresh
process every turn and Claude Code keeps one resident, and with presence on its own axis
that difference stops being visible at all. `done` went because it describes the user,
not the character — a character that finished and one that was just created are in the
same condition, and what differs is whether the result has been seen; as an unread mark
it also implements the already-decided "an agent finished is low-priority attention,
grouped" directly ("3 came back with news since you last looked") and gives attention
items the auto-resolution [IDEA-03](05-ideas-to-discuss.md) asks for. `idle` went because
the 2026-09-01 model defined it as derived from a timeout, which was never "has nothing to
do" but "claims to be working and has been silent for N minutes" — a suspicious-activity
signal, left unbuilt and still open as IDEA-01's P3, not a state of its own. `blocked`
went with permission management, below.
**How each activity clears:** `waiting` and `failed` clear when the user *acts* (answers,
revives, archives); an unread mark clears when the user *looks*; `ready` asks for nothing.

## 2026-09-03 — Territory: own worktree by default, shared directory as the explicit alternative

**Who:** Rodrigo
**Decision:** A character's territory is chosen when it is created, in one of two modes.
**Own territory** (the default): LiveAgentsView creates and administers a git worktree
under `~/.liveagentsview/worktrees/`, on a new or existing branch, and the character works
only there. **Shared territory**: the character works on the chosen directory exactly as
it is, on whatever branch is checked out, and LiveAgentsView runs no git command on it at
all. LiveAgentsView never runs `git checkout` on a directory the user picked. A worktree
is removed when its character is dismissed only if it has no uncommitted changes;
otherwise it is left in place and the user is told.
**Rationale:** picking a branch today runs `git checkout` in the user's real directory,
switching the branch under whatever else is using it — an editor, another character.
Worktrees also make "several characters on one repo" the normal case rather than a
collision. Own territory is the default because, with permission gates removed (below),
the worktree is the only thing standing between a character and the directory the user is
working in; it is not a sandbox and does not pretend to be one, but it is the containment
that exists.

## 2026-09-03 — Consistency: presence is observed, never stored

**Who:** Rodrigo
**Decision:** SQLite stores history and intent — which characters exist, what happened to
them, which are archived. It never stores whether a process is alive. Liveness is observed
from the `pilot-runner` that owns the process, which is the single authority on it, and
the daemon reconciles what it believes against what is actually running continuously while
it runs, not only at startup. At startup it also sweeps orphans: sockets, runners and
worktrees with no character behind them.
**Rationale:** the same fact currently lives in three places — the persisted `state`, the
manager's in-memory `running` flag, and the real process — and is reconciled once, at
daemon startup. A runner killed abruptly is indistinguishable from the daemon shutting
down, so the character is deliberately left as it was and reads as `working` forever. A
fact that is observed rather than stored cannot drift.

## 2026-09-03 — Archiving sends a character to sleep; dismissing removes it

**Who:** Rodrigo
**Decision:** Archiving a character stops its process, freeing the memory it holds, keeps
its full transcript and its territory, and takes it out of camp. It is allowed in any
activity, including while working, with a confirmation that says the current quest will be
stopped. Unarchiving brings it back to camp, and talking to it wakes it with its context
intact. A separate **dismiss** action removes a character for good, with its history, and
removes its worktree when that worktree is clean. This supersedes the 2026-09-02 decision
"Sessions can be archived, reversibly, in any non-working state" on both points: archiving
now does touch the underlying process, and `working` is no longer excluded.
**Rationale:** measured on this machine, a character that had never received a single
message held 129 MB of RSS after three and a half hours, plus 10 MB for its runner; ten of
them at camp is over a gigabyte held for nothing. CPU is not the cost, memory is. An
archive that leaves the process running is a display filter, not a lifecycle. Excluding
`working` also made the character the user most wanted to get rid of — one stuck reading
as working with nothing to do — the exact one that could not be archived, with no delete
to fall back on.

## 2026-09-03 — Permission management is dropped; every race runs auto-approving

**Who:** Rodrigo
**Decision:** LiveAgentsView stops mediating tool permissions. Claude Code characters
launch with `--permission-mode bypassPermissions`, matching what Cursor characters already
do with `--force`. The approve/deny control, the permission transcript events and the
`internal/pilotmcp` helper that existed to be Claude Code's `--permission-prompt-tool` are
removed. Whether a permission layer is built later is deliberately left open.
**Rationale:** Rodrigo's call — "hagamos para esta versión lo mismo con Claude de mandarlo
en modo automatico (...) y nos sacamos la gestión de permisos por ahora de encima. Luego
en el futuro podríamos ver si necesitamos implementarlo o no." Permissions were the one
capability that could not be normalized across races: Cursor's CLI has no channel to ask
an external supervisor (2026-09-02 decision), so the choice was between an interface that
behaves differently depending on the race behind a character, and one that behaves the
same everywhere. Both `--permission-mode bypassPermissions` and `--force` were confirmed
present on this machine's installed CLIs. Consequences accepted and named rather than
discovered later: `blocked` loses its only real signal and leaves the state model;
[IDEA-01](05-ideas-to-discuss.md)'s P0 level has no content, so the attention queue rests
on the end-of-turn classifier and on failures; and a character can now run any command
without asking, which is why own territory is the default. This also removes, rather than
fixes, the loss of a pending permission request across a daemon restart. A durable
permission policy owned by LiveAgentsView was proposed during the same session and
explicitly deferred — [IDEA-11](05-ideas-to-discuss.md).

## 2026-09-03 — A quest is not a modelled object

**Who:** Rodrigo
**Decision:** A quest is what the user asks for in the chat, not a persisted object with
its own state and result. The character is the durable thing; the transcript is flat.
**Rationale:** modelling quests would mean a start, an end, a status and a result per
task, which is task management — explicitly out of scope in [02-scope.md](02-scope.md).
The unread mark already answers the only question a quest object would have answered
("did something finish that I have not seen?"), at no cost.

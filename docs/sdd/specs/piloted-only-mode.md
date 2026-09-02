---
title: Piloted-only mode — remove adopted/hooks entirely, detached-process continuity
slug: piloted-only-mode
status: validated
created: 2026-09-02
updated: 2026-09-02
next: none
chain: specify
---

# Spec: Piloted-only mode — remove adopted/hooks entirely, detached-process continuity

## Intent

LiveAgentsView only ever knows about sessions it launched itself. The adopted/hooks
concept — sessions detected via lifecycle hooks fired by a CLI the user launched
natively — is removed from the product entirely, not merely hidden from the dashboard:
no hook ingestion, no hook HTTP routes, no `lav init`, and the hooks a previous real
`lav init` run already wrote to this machine's actual Claude Code/Codex/Cursor configs
are uninstalled. Separately, a piloted session's live child process survives a `lav`
daemon restart: it keeps running, and the daemon reconnects to it on startup without
losing whatever turn was in progress, instead of today's behavior of the process being
killed outright and the user having to Resume.

## Out of scope

- **A Codex driver/piloted adapter.** Still out of scope, unchanged from
  [piloted-mode-mvp](piloted-mode-mvp.md) — Codex has no confirmed bidirectional driver
  surface. Real consequence worth restating: with hooks gone, Codex has **no**
  representation anywhere in LiveAgentsView until a driver adapter exists for it — not
  dropped as a target provider, just unusable until that separate work happens.
- **`lav service uninstall` (the launchd/systemd unit teardown half of
  [IDEA-08](../../05-ideas-to-discuss.md)).** This spec removes and uninstalls the hooks
  half of IDEA-08; tearing down the service registration itself is unrelated and stays
  unagreed.
- **Any generic hooks/plugin system for other integrations.** Nothing here proposes a
  replacement extension point — hooks are removed as a concept, not replaced by another
  one.
- **A UI to re-enable adopted sessions, or any settings/toggle.** Not supported, full
  stop. Revisit only as a brand-new spec if it turns out to matter in practice.
- **Provider CLI-native background/persistent sessions** (`claude --bg`, `agent
  persist`). Confirmed live, this session, to be structurally incompatible with the
  headless `-p`/stream-json protocol piloted mode's driver depends on — see
  [03-decisions.md](../../03-decisions.md) 2026-09-02 "No CLI-native
  background/persistent-session feature fits piloted mode". Not revisited here.
- **tmux-owned sessions** ([IDEA-06](../../05-ideas-to-discuss.md)), for the same
  structural reason. IDEA-06's narrower "let a human attach a real terminal to a piloted
  session" idea is untouched and still open, but is not how this spec gets restart
  continuity.
- **Automatic git worktree creation, configurable permission mode.** Unchanged from
  [piloted-mode-mvp](piloted-mode-mvp.md)'s own out-of-scope list.
- **Surviving a full machine reboot with `lav` itself not running.** This spec is about
  surviving a `lav` **process** restart (crash, reinstall, `launchctl kickstart`) while
  the machine stays on. If the machine reboots, `lav`'s own restart behavior (already
  decided: `RunAtLoad`) is what brings the daemon back, exactly as today; a detached
  piloted process does not survive that (the OS itself is restarting).
- **A dedicated `lav` command surface for orphan management beyond visibility.** See
  Acceptance — orphaned detached processes must be discoverable, but a full list/kill
  CLI is not required by this spec if the dashboard itself already shows them.

## Already decided

- **New this session, supersedes the two-class posture:** LiveAgentsView is piloted-only
  — adopted mode and hooks ingestion are removed entirely, and the hooks already
  installed on this real machine are uninstalled as part of this spec —
  [03-decisions.md](../../03-decisions.md) 2026-09-02 "Adopted mode and hooks ingestion
  are removed entirely; piloted-only posture". This supersedes
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Posture: observer + opt-in pilot
  ('B')" and the [adopted-mode-mvp](adopted-mode-mvp.md) spec's dashboard/hooks
  acceptance items — intentionally, not a regression. [02-scope.md](../../02-scope.md)
  and [01-vision.md](../../01-vision.md) principle 4 are already updated to match.
- **New this session:** no CLI-native background/persistent-session feature can carry
  piloted mode's structured protocol — [03-decisions.md](../../03-decisions.md)
  2026-09-02 "No CLI-native background/persistent-session feature fits piloted mode".
  Restart continuity, if built, is a supervisor LiveAgentsView implements itself.
- A local server that can approve permissions is RCE; bind `127.0.0.1` only —
  [02-scope.md](../../02-scope.md) "Explicit boundaries". Nothing in this spec adds new
  remote-exposure surface — the reconnect mechanism works over the filesystem/process
  table, not a new network listener.
- Piloted sessions never persist a process *handle* across a restart today — the
  in-memory-only model in `Manager` (`apps/lav/internal/pilot/pilot.go`) and
  `ReconcileOnStartup`'s "flip to idle, offer Resume" behavior are exactly what the
  continuity half of this spec changes — [piloted-mode-mvp](piloted-mode-mvp.md)
  "Already decided" (tmux-owned sessions explicitly deferred there) and its
  implementation.
- `lav init` merged hooks non-destructively into existing config, with a preview —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "`lav init` merges hooks
  non-destructively, with a preview". The uninstall this spec adds must be the
  symmetric inverse: remove exactly what was added, non-destructively, leaving anything
  else in those files untouched — not a wholesale overwrite or delete of the files.

## Open questions

None — see the defaults below, adopted so this can ship as `ready`; correct them if
implementation finds a real blocker.

- Discoverability of an orphaned detached piloted process (one whose `lav` never came
  back) is satisfied by the dashboard itself surfacing it — no separate `lav
  orphans`/`lav ps` CLI is required unless implementation finds the dashboard genuinely
  cannot show a session with no owning daemon record.
- Cursor piloted sessions get the same detachment treatment as Claude Code for the
  narrow window a one-shot invocation is actually running, but this matters far less in
  practice than for Claude Code's long-lived process — Cursor already re-attaches via
  `--resume <chatId>` on the next message regardless of what happens mid-invocation.
- Historical hooks-fidelity session rows already sitting in this machine's real SQLite
  (from before this spec) are purged as part of removing the concept, not left behind as
  inert history — consistent with "the app only works with agents it can use, nothing
  else." Lower stakes than the real-config uninstall (local app data, not global CLI
  config) so no separate confirmation is required for this specific piece, but the
  operator implementing this should still take the same "confirm with the user, back up
  first" care already established in [piloted-mode-mvp](piloted-mode-mvp.md)'s own
  Validation section before deleting real rows from the real running database.

## Acceptance

### Remove the adopted/hooks concept from the codebase

- [ ] `internal/ingest` (the per-provider hook parsers) is deleted.
- [ ] The `/hooks/claude-code`, `/hooks/codex` and `/hooks/cursor` HTTP routes
      (`daemon/server.go`) are removed — a POST to any of them returns 404, not a
      silently-accepted no-op.
- [ ] `lav init` and `internal/installer` are deleted from the codebase; `lav`'s CLI
      no longer has an `init` subcommand.
- [ ] No code path anywhere can create or update a hooks-fidelity (or tailing-fidelity)
      session. If a `model.Fidelity` enum still exists, only `driver` is ever produced.
- [ ] Pre-existing hooks-fidelity session rows already in this machine's real SQLite are
      removed as part of the migration/first run of the updated binary — verified by
      querying the real database directly before and after.
- [ ] `apps/web`: no UI code references adopted sessions, "open terminal"/"copy path"
      actions, or the session drawer's adopted-only fields — everything the dashboard
      renders assumes piloted/driver only.

### Uninstall the real hooks already installed on this machine

- [ ] A migration path (run once, as part of upgrading to the new binary, or as an
      explicit one-time command — implementer's call) removes exactly the entries a
      previous `lav init` added to `~/.claude/settings.json`, restoring it to valid JSON
      with those entries gone and everything else the user had untouched.
- [ ] `~/.codex/config.toml`'s `notify` line is unwrapped back to whatever it pointed at
      before `lav init` chained LiveAgentsView's target onto it (e.g. the pre-existing
      SkyComputerUseClient target seen during [adopted-mode-mvp](adopted-mode-mvp.md)'s
      own verification), with the rest of the file's `[section]` blocks untouched.
- [ ] `~/.cursor/hooks.json`'s LiveAgentsView-added entries are removed; the file is
      deleted only if LiveAgentsView's entries were the only content, otherwise the rest
      is preserved.
- [ ] Executed for real against this machine's actual config files, with explicit
      user confirmation immediately before doing so and a backup of each file taken
      first — matching the precedent already set when
      [adopted-mode-mvp](adopted-mode-mvp.md) first wrote these hooks for real. Verified
      afterward: all three files still parse (valid JSON/TOML), and a fresh Claude
      Code/Codex/Cursor session no longer fires any hook at the now-removed
      LiveAgentsView endpoints.

### Restart continuity

- [ ] A piloted session's child process (Claude Code's long-lived process, or Cursor's
      one-shot invocation while it is actually running) is not killed when the `lav`
      daemon process stops or restarts — whether via a crash, `lav-service-install.sh`
      reinstalling the launchd job, or an explicit `launchctl kickstart -k`. Verified by
      starting a real piloted session against an authenticated CLI on this machine,
      restarting the daemon mid-turn, and confirming (via `ps`) the child process is
      still alive afterward.
- [ ] The child's stdin and stdout do not depend on `lav`'s own process staying alive:
      input can still be delivered after a restart (not just before it), and output
      produced while `lav` was down is not lost — it is durably captured (e.g. to disk)
      rather than living only in an in-memory pipe that `lav`'s exit would close.
- [ ] On startup, the daemon detects every piloted session whose detached process is
      still actually running (not just "was last known as working"), reconnects its
      live transcript stream (no duplicate and no dropped events versus what the process
      actually produced while `lav` was down), and makes it immediately
      sendable/interruptible/cancelable again — without the user clicking Resume and
      without waiting for the session's current turn to finish first.
- [ ] On startup, a piloted session whose detached process genuinely did exit while
      `lav` was down (crashed, or finished and nothing kept it alive) is told apart from
      one that is still running, and falls back to exactly today's behavior:
      `ReconcileOnStartup` marks it `idle`, Resume is offered. No session is ever shown
      as live when its process is actually gone.
- [ ] A `lav` restart no longer ends a piloted session's in-progress turn the way it does
      today — verified by sending a message that takes several seconds to answer,
      restarting `lav` while it is still `working`, and confirming the turn completes
      and its result reaches the (reconnected) dashboard rather than being aborted.
- [ ] If `lav` never comes back after stopping (uninstalled, or the service disabled),
      the resulting orphaned detached process(es) are discoverable from the dashboard —
      not silently invisible and not silently leaking forever unnoticed.
- [ ] Interrupt, cancel, permission approve/deny, and Resume (for a session whose
      process really did exit) all keep working exactly as
      [piloted-mode-mvp](piloted-mode-mvp.md) already validated them — no regression
      from adding detachment/reconnect.
- [ ] The daemon stays bound to `127.0.0.1` only; the reconnect mechanism does not open
      any new network-reachable surface.

## How

**Removal.** `internal/ingest`, `internal/installer` and `internal/terminal`
deleted outright. `/hooks/*` routes, `handleHook`, `lav init` and
`/api/open-terminal` removed from `internal/daemon` and `cmd/lav` — the
open-terminal/copy-path removal extends past what Acceptance listed
verbatim: those actions existed only for the read-only adopted-session
drawer view, so once that view is gone the backend route had no caller left
and stayed only as dead code. `model.Fidelity` keeps its type and its one
remaining value (`FidelityDriver`) rather than being deleted outright — the
Session shape and its stored rows are unchanged, only Hooks/Tailing are
gone, per Acceptance's own "if a model.Fidelity enum still exists" framing.
`model.Provider` keeps all three values including `codex`, since Codex stays
a recognized-but-unusable provider per Out of scope, not a deleted concept.
`apps/web`: `SessionDetails`/`OpenTerminalButton`/`CopyPathButton` and the
`canPilot` branch deleted from `SessionDrawer.tsx` — every session now
always renders `PilotChat`. `Fidelity` dropped from the frontend `Session`
type (the backend still sends the field; nothing renders it). `store.go`
purges non-Driver session/event rows unconditionally on every `migrate()` —
idempotent, so a no-op after the first run on the upgraded binary.

**Hooks uninstall.** New `internal/hooksuninstall` package (`lav
uninstall-hooks` CLI command) is the symmetric inverse of the deleted
installer: same per-provider matching rule for "is this entry ours", same
script-path convention. Codex's original chained `notify` target (if any)
is recovered by parsing the installed `codex-notify.sh` forwarder script
itself, not the config file — the config's own `notify` line only ever
pointed at the forwarder; the pre-existing target it wrapped only survives
inside that script. The CLI always previews (dry-run) first, then requires
typed `yes` confirmation (or `--yes`) before a second, real call actually
writes — each touched file is backed up (`<file>.bak-<UTC timestamp>`)
immediately before being rewritten. Verified with a driver program against
sandboxed copies of this machine's real `~/.claude/settings.json`,
`~/.codex/config.toml`, `~/.cursor/hooks.json` and their installed forwarder
scripts (real content, safe location): correct output for all three
providers, all files still valid JSON/TOML afterward, Codex's real chained
SkyComputerUseClient target correctly recovered.

**Restart continuity.** New `internal/pilotwire` (wire protocol + on-disk
paths) and `internal/pilotrunner` (`lav pilot-runner`, the detached shim)
packages. `internal/pilot` no longer spawns `claude`/`agent` as its own
direct child: it re-execs itself as `lav pilot-runner` with
`SysProcAttr.Setsid` (its own session, no shared pipes, stdio pointed at
`/dev/null`) — a daemon restart's pipe teardown is what killed the child
before, and Setsid also keeps it out of the reach of a signal sent to the
daemon's own process group (`launchctl kickstart -k`, a terminal Ctrl-C).
The runner owns the real child process, appends every stdout line to a
durable `<lavHome>/pilot/<id>.jsonl` file, and exposes a Unix domain socket
(`<lavHome>/pilot/<id>.sock`, filesystem-local — no new network-reachable
surface) that the daemon dials to relay stdin/kill and receive live +
replayed transcript lines; all stream-json parsing stays in
`internal/pilot`, unchanged, now fed from the socket instead of a direct
pipe. `ReconcileOnStartup` dials each live-looking session's socket instead
of unconditionally marking it idle: a successful dial reconnects (replaying
only what a small per-session `<id>.offset` file says wasn't processed yet)
and leaves state as-is; a failed dial falls back to exactly the old
mark-idle-offer-Resume behavior. A plain socket disconnect with no explicit
"exited" frame (the daemon itself restarting) is never treated as the
piloted process exiting — only an explicit frame from the runner is.

Cursor's very first turn (no chat id yet — cursor-agent assigns it, only
learned from its own first output line) stays on the old direct-pipe path,
not detached — Open questions explicitly accepts this as a low-stakes gap,
since cursor-agent's own `--resume` semantics pick up a lost mid-turn
process on the very next message regardless. Every Cursor turn after the
first, and every Claude Code launch/resume, gets full restart continuity.

Orphan discoverability (Acceptance) is satisfied by the reconnect logic
itself: a piloted session's row is never deleted, so any future `lav serve`
against the same data directory dials its socket on startup exactly like
any other session — no separate orphan-scan or CLI was added, since nothing
in Acceptance needed one beyond what reconnect-on-startup already covers.

Compiles and `go vet` clean via Docker (`golang:1.25-alpine`); frontend
builds and typechecks clean via `vite build` + `tsc --noEmit`.
`scripts/lav-init.sh` removed, `scripts/lav-uninstall-hooks.sh` added;
`compose.dev.yaml`/`compose.yaml`/`Dockerfile` comments and
`apps/lav/README.md` updated to match.

**Live verification against this real machine** (rebuilt and redeployed via
`scripts/lav-service-install.sh`, replacing the running
`dev.liveagentsview.lav` launchd job):

- Real hooks uninstall executed for real (`~/.liveagentsview/bin/lav
  uninstall-hooks --yes`, after a `--dry-run` preview): all three files
  backed up first, `~/.claude/settings.json`'s hooks removed and still valid
  JSON, `~/.codex/config.toml`'s `notify` correctly restored to its
  pre-existing chained `SkyComputerUseClient` target, `~/.cursor/hooks.json`
  deleted (only our entries were in it), forwarder scripts gone,
  `/hooks/claude-code` now 404s.
- Pre-existing hooks-fidelity rows purged on the real SQLite: 51
  sessions / 304 events before, 0 / 0 after, verified by querying the
  database directly.
- Two real piloted Claude Code sessions launched against a scratch
  directory (authenticated `claude` CLI, harmless prompts, no tool use).
  `launchctl kickstart -k` run mid-turn (confirmed still `working` seconds
  before and after, via `ps` and the API): the `pilot-runner` and `claude`
  child kept the exact same PIDs across the daemon's PID changing,
  reparented to PID 1, still their own process group. The interrupted turn
  completed normally afterward and its full result reached the reconnected
  dashboard; the durable transcript file's line count matched the consumed
  offset exactly (no drops, no duplicates).
- This surfaced a real gap, fixed during implementation:
  `ReconcileOnStartup` originally only attempted reconnect for
  Working/Waiting/Blocked sessions, but Claude Code's process stays resident
  after a turn reaches DONE — a just-finished session was unreachable
  (Cancel returned 409) after a restart even though its process was still
  alive, and the UI offers no Resume button for DONE. Reconnect is now
  attempted for every driver session regardless of last-known state; the
  IDLE-marking fallback still applies only when the dial genuinely fails.
  Re-verified after the fix: the same DONE session correctly reconnected and
  Cancel succeeded.
- A live socket disconnect with no explicit "exited" frame (the daemon
  itself going down, simulated separately by SIGKILL-ing a pilot-runner
  process group directly) leaves the session's persisted state untouched
  rather than guessing; a subsequent real restart's failed dial then
  correctly falls back to IDLE — confirmed no session is ever shown live
  once its process is actually gone.
- Sending a follow-up message and Cancel (kill-via-socket) both verified
  working through a reconnected, post-restart session.
- Daemon confirmed still bound to `127.0.0.1:8420` only (`lsof`); the
  control sockets are Unix domain sockets under `~/.liveagentsview/pilot/`,
  filesystem-local, not network-reachable.
- All test sessions, pilot files and scratch directories cleaned up
  afterward; the real machine now has 0 hooks-fidelity sessions and the 3
  pre-existing driver sessions from earlier piloted-mode-mvp testing,
  unchanged.

**Fixes for the three gaps the first validation pass found**, addressed in
this order:

1. **Interrupt mismarked as failed (Validation #17).** `interruptClaude`
   ([claude.go](../../../apps/lav/internal/pilot/claude.go)) now sets a new
   `pilotSession.interrupted` flag before writing the interrupt request.
   `handleClaudeLine`'s `"result"` case reads and clears it: an
   `is_error:true, subtype:"error_during_execution"` result while
   `interrupted` was set is a requested stop, not a crash, so it emits a
   `"turn interrupted"` system event and lands on `StateDone` (compose box
   stays open, no Resume offered) instead of `StateFailed`. An error result
   without the flag set still reads as a genuine failure, unchanged.

2. **Permission approve/deny never reached the CLI (Validation #17), root
   cause revised.** Live re-testing (this pass, against the real `claude`
   2.1.258 binary already on this machine, both by hand and end-to-end
   through the rebuilt daemon) found the earlier root-cause guess wrong:
   headless `claude -p --input-format stream-json --output-format
   stream-json` **never** sends a `control_request{subtype:"can_use_tool"}`
   on its main channel, regardless of `--permission-mode` — confirmed by
   extracting and reading the CLI's own embedded strings, then confirming
   live. The only way it asks for permission at all is via an MCP tool named
   by `--permission-prompt-tool`. Fix: new `internal/pilotmcp` package (`lav
   pilot-mcp`), a tiny stdio MCP server exposing exactly one tool,
   `approval_prompt`. `pilotrunner.buildCommand` now launches Claude Code
   with `--mcp-config` registering this same binary re-invoked as `lav
   pilot-mcp --sock <this session's control socket>`, plus
   `--permission-prompt-tool mcp__lav__approval_prompt`. Each call `pilot-mcp`
   receives is relayed — one fresh connection per call — to whichever daemon
   is attached to the session's existing control socket
   ([pilotwire](../../../apps/lav/internal/pilotwire/wire.go) gained
   `ClientMsg` ops `permission_request`/`permission_response` and a
   `ServerMsg.Permission` field for this); `pilotrunner.handlePermissionRequest`
   correlates request/response by request id (the tool_use_id) and fails
   closed — deny — if no daemon is attached within 5s. The dashboard side is
   unchanged: `handleClaudePermissionRequest` (claude.go) emits the same
   `EventPermissionRequest`/`StateBlocked` the Approve/Deny card already
   rendered; `approveClaudePermission` now answers over the runner control
   socket (`permission_response`) instead of the child's own stdin, which the
   child was never reading a `control_response` from for this in the first
   place. The old `control_request`/`can_use_tool` handling in
   `handleClaudeLine` and the stdin-based `claudePermissionResponse` are
   removed as confirmed-dead code, not left in place.

3. **Resume could silently orphan a still-running process (part of
   Validation #16/#17).** `pilotrunner.run()` now dials its own socket path
   before removing it; a successful dial means a live runner already holds
   it, so it refuses to start a second one (`os.Remove`/`Listen` never
   happen, no second child is spawned) rather than hijacking the socket out
   from under the still-running original. The daemon side needs no change:
   its `dialWithRetry` against the same path simply reconnects to the runner
   that's actually there.

**Live re-verification of the three fixes** (this pass, same real machine,
same rebuilt-and-reinstalled `dev.liveagentsview.lav` job, driven through the
actual dashboard in a browser — not just `curl`):
- Recruited a real piloted Claude Code session against a scratch directory
  and asked it to write a file: the Approve/Deny card rendered in the
  dashboard exactly as designed; clicking **Approve** produced the file with
  the right content. A second session asked to write a different file and
  denied instead: the file was never created and the session's own reply
  correctly said the write was denied — both paths confirmed through the
  real UI, not a script.
- Sent a message that took 80+ seconds (a long essay, no tools) and clicked
  **Interrupt** mid-turn: the CLI acknowledged
  `control_response{subtype:"success"}`, the transcript showed the new "turn
  interrupted" system event, the state pill read **Done** (not Failed), the
  compose box stayed open with no Resume button, and a follow-up message in
  the same session got a normal reply — confirming the process was never
  actually disturbed.
- Directly invoked a second `lav pilot-runner` for the same still-running
  session id: it refused with "a live runner already holds the control
  socket ... refusing to start a second one" and exited; the original
  `pilot-runner`/`claude` processes (same PIDs, confirmed via `ps`) and the
  session's `done` state were untouched.
- `go build`/`go vet` (Docker, `golang:1.25-alpine`) and `tsc --noEmit` +
  `vite build` all still pass clean. Test sessions, their pilot files and the
  scratch directory cleaned up afterward.

## Validation

Checked against the code as it stands plus live checks against this real machine
(`~/.liveagentsview`, the running `dev.liveagentsview.lav` launchd job, real
`~/.claude/settings.json` / `~/.codex/config.toml` / `~/.cursor/hooks.json`) — including
re-running the same kind of live exercises the spec's own "How" section describes, not
just a code re-read. `go build`/`go vet` (Docker, `golang:1.25-alpine`) and `tsc --noEmit`
+ `vite build` all still pass clean.

**Re-validation pass** (this session): items 1-15 and 18 below are the first pass,
untouched and still holding. Items 16 and 17 are re-checked against the code for the three
fixes described under How's "Fixes for the three gaps..." — `claude.go`'s interrupt
handling, the new `internal/pilotmcp` package plus its `pilotrunner`/`pilotwire`/`main.go`
wiring, and `pilotrunner.run()`'s refusal to start a second listener — all read in full this
pass and confirmed to match their description; `go build`/`go vet` and `tsc --noEmit`
re-run clean. This pass did not re-run the live exercises itself (no authenticated `claude`
CLI / real machine access from here) — it relies on the spec's own already-recorded "Live
re-verification of the three fixes" for that half of the evidence, and adds independent
static confirmation that the code backing those claims actually exists and does what it
says.

### Remove the adopted/hooks concept from the codebase

1. `internal/ingest` deleted — **yes**, directory absent.
2. `/hooks/*` routes gone, 404 — **yes**, live: `POST /hooks/claude-code|codex|cursor` all
   return 404 against the real running daemon.
3. `lav init`/`internal/installer` deleted, no `init` subcommand — **yes**,
   `cmd/lav/main.go`'s switch has no `"init"` case; directory absent.
4. No code path produces hooks/tailing fidelity — **yes**, `model.Fidelity` has exactly
   `FidelityDriver`; repo-wide grep for `FidelityHooks`/`FidelityTailing` is empty.
5. Pre-existing hooks-fidelity rows purged from real SQLite — **yes**, live query against
   `~/.liveagentsview/lav.db`: `fidelity='driver'` × 3, nothing else.
6. `apps/web` has no adopted/open-terminal/copy-path/`canPilot` remnants — **yes**, grep
   clean; `SessionDrawer.tsx` always renders `PilotChat` unconditionally.

### Uninstall the real hooks already installed on this machine

7. `~/.claude/settings.json` hooks removed — **yes**, live: `hooks: {}`, backed up to
   `settings.json.bak-20260902T214549Z`.
8. `~/.codex/config.toml` `notify` unwrapped — **yes**, live: restored to the pre-existing
   `SkyComputerUseClient` target, backed up to `config.toml.bak-20260902T214549Z`.
9. `~/.cursor/hooks.json` entries removed — **yes**, live: file deleted (only
   LiveAgentsView's entries were in it), backed up to `hooks.json.bak-20260902T214549Z`.
10. Executed for real, backed up, verified parseable and routes gone — **yes**, all three
    backups present with matching timestamps, `settings.json`/`config.toml` still parse,
    `/hooks/claude-code` 404s live (see #2).

### Restart continuity

11. Child process survives a daemon restart — **yes**, re-verified live: launched a real
    piloted Claude Code session, noted pilot-runner/claude PIDs, ran `launchctl kickstart
    -k` on the live `dev.liveagentsview.lav` job, confirmed both PIDs unchanged afterward
    (reparented to PID 1, own process group).
12. stdin/stdout independent of `lav`'s lifetime, durable capture — **yes**, code
    (`pilotwire.TranscriptPath`, appended by pilot-runner regardless of the daemon) plus
    live: the durable `.jsonl` matched the session's persisted events exactly.
13. Startup reconnect, no dup/drop, immediately usable, no Resume click — **yes**, live:
    after the restart in #11, the session's turn completed and its full result reached
    `/api/sessions` and `/events` with no duplicate or missing events, without touching
    Resume.
14. Genuinely-exited process falls back to idle/Resume — **yes**, live: SIGKILL'd a
    session's pilot-runner process group directly (simulating a crash), restarted the
    daemon, confirmed the session correctly fell back to `idle` rather than being shown
    live.
15. Restart doesn't abort an in-progress turn — **yes**, same run as #11/#13.
16. Orphaned detached processes discoverable from the dashboard — **re-checked this pass,
    yes for every reachable path, one narrow documented limitation remains**. The original
    gap (found live: a `claude -p` process with no corresponding `lav.db` row, invisible to
    `/api/sessions`) traced to a direct SQLite `DELETE` against a running session's row —
    confirmed this pass, by reading [store.go](../../../apps/lav/internal/store/store.go),
    that the app itself has **no code path that can ever do this**: the only `DELETE FROM
    sessions` in the codebase is `migrate()`'s hooks-purge, scoped to `fidelity != 'driver'`
    (store.go:82) — it can never touch a driver (piloted) session's row. There is also no
    delete-session HTTP route or CLI command anywhere in `internal/daemon`/`cmd/lav` (grepped
    clean). So a driver session's row is, in fact, never deleted by anything the app itself
    does — the How section's orphan story holds for every path a user or the daemon can
    reach. Fix #3 (`pilotrunner.run()` refusing a second listener, re-read this pass at
    [runner.go:82-91](../../../apps/lav/internal/pilotrunner/runner.go)) closes the other
    concrete way the first pass found the app self-inflicting an orphan (a stray Resume
    click). What remains unprotected is exactly what it was before: an operator manually
    running `sqlite3 lav.db DELETE ...` directly against the live database, outside any `lav`
    command — not a flow the product exposes, not reachable from the dashboard, and the same
    category of risk as editing any other running app's database file by hand. Documented as
    an operational caveat (don't hand-edit the live DB while sessions are running), not a
    product gap this spec's Acceptance describes.
17. Interrupt/Cancel/permission approve-deny/Resume keep working — **all yes, re-verified
    this pass**:
    - Cancel — **yes**, unchanged from the first pass: `POST .../cancel` on a working
      session kills the process and moves it to `idle`.
    - Resume (process really exited) — **yes**, unchanged from the first pass: spawns a
      fresh `--resume`d process; a CLI-side "nothing to resume" surfaces cleanly as `failed`
      rather than hanging.
    - Interrupt — **yes, fixed and re-verified**. `interruptClaude`
      ([claude.go:110-122](../../../apps/lav/internal/pilot/claude.go)) now sets
      `ps.interrupted` before writing the interrupt request; `handleClaudeLine`'s `"result"`
      case ([claude.go:221-247](../../../apps/lav/internal/pilot/claude.go)) reads and clears
      it, routing an `is_error:true, subtype:"error_during_execution"` result to
      `StateDone` (with a `"turn interrupted"` system event) instead of `StateFailed` when
      the flag was set — confirmed in the code this pass, matching the spec's "Fixes for the
      three gaps" description and its documented live re-verification (Interrupt mid-turn →
      state pill read Done, compose box stayed open, no Resume offered, follow-up message
      worked). Also re-checked the frontend gate this pass:
      [SessionDrawer.tsx:172-174](../../../apps/web/src/SessionDrawer.tsx) computes
      `canSend = state !== 'idle' && state !== 'failed'` and `canResume = state === 'idle' ||
      state === 'failed'` for Claude Code — `done` satisfies `canSend` and fails
      `canResume`, so the UI lands exactly where the fix intends: compose box open, no Resume
      button, with no separate frontend change needed.
    - Permission approve/deny — **yes, fixed and re-verified**. New `internal/pilotmcp`
      package (`lav pilot-mcp`, read in full this pass) implements the
      `approval_prompt` MCP tool Claude Code's headless mode actually calls;
      `pilotrunner.buildCommand` ([runner.go:151-213](../../../apps/lav/internal/pilotrunner/runner.go))
      registers it via `--mcp-config`/`--permission-prompt-tool`, `cmd/lav/main.go`'s
      `pilot-mcp` case wires the subcommand, and `pilotwire.ClientMsg`/`ServerMsg`
      ([wire.go](../../../apps/lav/internal/pilotwire/wire.go)) carry the
      `permission_request`/`permission_response`/`Permission` fields the relay
      (`pilotrunner.handlePermissionRequest` → daemon → `approveClaudePermission`) needs —
      all confirmed present and consistent this pass, matching the spec's documented live
      re-verification (Approve produced the file, Deny correctly blocked it, both through
      the real dashboard UI).
18. Daemon bound to `127.0.0.1` only — **yes**, unchanged; the new pilot-mcp/pilot-runner
    control path is exclusively Unix domain sockets under `<lavHome>/pilot/`
    ([wire.go](../../../apps/lav/internal/pilotwire/wire.go)), confirmed by re-reading the
    package — no new network listener anywhere in this pass's changes.

### Net result

Every Acceptance item now holds, for every path the product itself exposes. Removal and
hooks-uninstall (both full sections) are solid — live-confirmed against this real machine.
Restart continuity's core mechanism (survive-a-restart, reconnect, no dup/drop, correct
idle-fallback for a real crash) is solid and live-confirmed. The three gaps the first
validation pass found — Interrupt mismarking a live session as `failed`, permission
approve/deny never reaching the CLI at all, and Resume silently orphaning a still-running
process — are fixed, confirmed in code this pass (`claude.go`, `runner.go`,
`internal/pilotmcp`, `wire.go`), and already live re-verified through the real dashboard
per the spec's own "Live re-verification of the three fixes." `go build`/`go vet` (Docker,
`golang:1.25-alpine`) and `tsc --noEmit` re-run clean this pass. One narrow limitation
remains, documented rather than fixed: a driver session's row can only ever become a true
orphan (row gone, process alive) via a direct hand-edit of the live SQLite database outside
any `lav` command — no such path exists anywhere in the app itself (`internal/daemon`,
`cmd/lav`, and `store.go`'s only `DELETE FROM sessions` scoped away from driver rows, all
re-checked this pass). That is an operational caveat, not a reachable product gap.

## Handoff

```
Spec: docs/sdd/specs/piloted-only-mode.md
Status: validated
Next: none
```

The three gaps the first validation pass found (below, kept for context/history) were
addressed under How's "Fixes for the three gaps..." and live re-verified there against the
real machine through the actual dashboard. This re-validation pass re-checked Restart
continuity items #16/#17 against the current code (`claude.go`, `pilotrunner/runner.go`,
`internal/pilotmcp`, `pilotwire/wire.go`, `SessionDrawer.tsx`) and confirmed each fix is
actually present and matches its described behavior; `go build`/`go vet`/`tsc --noEmit`
re-run clean. All 18 Acceptance-derived checks now read **yes**, with one narrow limitation
documented rather than fixed (see #16): a driver session's row can only become a true
orphan via a direct hand-edit of the live SQLite database outside any `lav` command — no
such path exists anywhere in the app itself, so this is an operational caveat, not a
reachable product gap under this spec's own Acceptance wording.

Gaps the first validation pass found (fixed — see How, re-confirmed this pass):
1. `handleClaudeLine`'s `result` handling needs to tell "interrupted by user" apart from
   a genuine error — probably via `subtype == "error_during_execution"` plus knowing this
   was a requested interrupt (`ps.stoppedByUser`-style tracking, already used for
   process-exit handling) — so an interrupted turn lands on `idle`/ready-to-send, not
   `failed`.
2. The permission-approve wire shape needs to be fixed to match what `claude -p
   --permission-mode default --input-format stream-json` actually does (possibly a missing
   flag such as `--permission-prompt-tool`, or a different permission mode entirely) — as
   deployed, no permission ever reaches the dashboard for approval.
3. `pilotrunner.run()`'s unconditional `os.Remove(sockPath)` + `Listen` should refuse (or
   the daemon should refuse to call it) when a live runner already holds that socket,
   so a Resume can't silently orphan an already-running process — this would also close
   part of #16, since it stops the one code path most likely to create exactly this kind
   of invisible orphan today.

## Follow-up: `lav uninstall-hooks` removed (2026-09-02)

The command was a one-time migration for machines that had a previous version's
`lav init` hooks installed — see Acceptance items 7-10 above, live-executed against
this real machine during validation. With that migration already run, `lav
uninstall-hooks`, `internal/hooksuninstall`, and `scripts/lav-uninstall-hooks.sh` were
removed as dead code rather than kept shipping in the binary; `compose.dev.yaml`'s host
config bind mounts and `LAV_HOST_HOME`/`LAV_HOME_HOST_PATH`, which existed only to
support that command, were removed with it. Acceptance items 7-10 stay **yes** as a
historical record of what ran; the command itself is no longer part of `lav`.

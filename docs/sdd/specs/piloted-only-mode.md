---
title: Piloted-only mode — remove adopted/hooks entirely, detached-process continuity
slug: piloted-only-mode
status: done
created: 2026-09-02
updated: 2026-09-02
next: validate
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

## Validation

_Filled in during validation._

## Handoff

```
Spec: docs/sdd/specs/piloted-only-mode.md
Status: done
Next: validate
```

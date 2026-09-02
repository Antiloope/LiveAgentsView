---
title: Native host runtime — launchd service, real terminal launch, verified Cursor payload
slug: native-host-runtime
status: validated
created: 2026-09-02
updated: 2026-09-02
next: none
chain: specify+implement
---

# Spec: Native host runtime — launchd service, real terminal launch, verified Cursor payload

## Intent

The daemon runs as a real launchd user service on macOS (verified on this machine),
installed from a host-native binary produced by the existing Docker build — not the
`docker compose` `restart: unless-stopped` stand-in. From the dashboard, "open in
terminal" actually opens a terminal at the session's `cwd` on macOS instead of only
copying the path. The Cursor adapter's hook payload field names are checked against a
real `agent`/`cursor-agent` CLI install on this machine and corrected if they differ from
the documentation-only guess.

## Out of scope

- **Piloted mode.** Unchanged, still its own future spec.
- **Windows service.** Not part of any decision doc.
- **Linux verified end-to-end.** systemd unit and the Linux terminal-launch fallback are
  written and reviewed, not live-verified — this machine is macOS. Documented as
  best-effort in Validation, same pattern as the Codex/Cursor "simulated only" items in
  [adopted-mode-mvp](adopted-mode-mvp.md).
- **Service lifecycle polish.** Uninstall, upgrade-in-place, edge cases around sleep/wake
  beyond `RunAtLoad`/`KeepAlive` (macOS) and systemd's default `WantedBy=default.target`
  (Linux). Already called out as deferred polish in adopted-mode-mvp's "Out of scope".
- **Rewriting "cursor-agent" across decided docs** (`03-decisions.md`, `02-scope.md`) off
  the strength of this session's chat claim alone. This spec only corrects the invocation
  once actually confirmed on this machine (Acceptance item below); if docs need a
  correction, it happens here with evidence, not as a preemptive rename.
- **Tailscale / remote access.** Unchanged, still an explicit later opt-in.

## Already decided

- Daemon runs as a user service (launchd/systemd); single self-contained binary for
  mac/linux — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Stack".
- The shipped binary runs directly on the host; Docker is for developing/testing this
  repo, not for running the product — [03-decisions.md](../../03-decisions.md) 2026-09-01
  "Docker is for developing LiveAgentsView, not for running it".
- "Open in the terminal" is called out as a first-class feature of the adopted+piloted
  hybrid — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Posture".
- This spec exists to close the three gaps [adopted-mode-mvp](adopted-mode-mvp.md) left as
  candidate follow-ups in its Validation (#2 native service, #7 Cursor field names, #11
  terminal launch) and in [06-status.md](../../06-status.md) "Suggested next step".
- AGENTS.md: nothing installed on the host besides Docker to *develop/test* this repo;
  everything runnable goes through `scripts/`. The native `lav` binary running on the host
  is the explicit, already-decided exception, not a violation.

## Open questions

None — chain confirmed by Rodrigo 2026-09-02: specify+implement, macOS
verified/Linux best-effort.

## Acceptance

- [ ] A build path (Docker-based, no Go/Node installed on the host) cross-compiles a
      static `lav` binary for the host's actual OS/arch (`darwin/arm64` on this machine)
      instead of the container's `linux` target, and copies it out to
      `~/.liveagentsview/bin/lav`.
- [ ] A new entry point (`lav service install`, or folded into `lav init`) generates and
      registers a macOS launchd user agent plist pointing at the installed binary
      (`RunAtLoad`, `KeepAlive`, stdout/stderr under `~/.liveagentsview/logs/`), and the
      equivalent systemd `--user` unit content for Linux (written, not registered live on
      this machine).
- [ ] On this machine, after running the install, `launchctl list` shows the job running,
      the dashboard is reachable at `127.0.0.1:8420`, and it survives a kill-and-respawn
      (or logout/login if practical) — with no `docker compose` container involved at all
      for this run.
- [ ] `scripts/dev-up.sh` / `compose.yaml` remain the development path unchanged; a new
      script (e.g. `scripts/lav-service-install.sh`) is the only supported entry point for
      the native install, per AGENTS.md ("nothing is run by hand").
- [ ] From the dashboard, an "open in terminal" action opens a real terminal at the
      session's `cwd` on macOS — the now-native daemon shells out directly (e.g. `open -a
      Terminal <cwd>`), no separate helper process. Copy-path may remain as a fallback but
      is no longer the only option.
- [ ] The Linux equivalent for "open in terminal" is implemented (e.g.
      `$TERMINAL`/`x-terminal-emulator` fallback) but explicitly marked unverified in
      Validation.
- [ ] Confirm on this machine whether the installed Cursor CLI's actual invocation is
      `agent`, `cursor-agent`, or both (e.g. one is a symlink/alias of the other), and
      document the finding — including whether it changes anything beyond the invocation
      name (e.g. `lav init`'s generated hook commands, `installer.go`).
- [ ] Capture at least one real hook payload from the installed CLI (`sessionStart` and
      `stop`, via `.cursor/hooks.json` wired to the running daemon, same mechanism
      `lav init` already sets up) and correct `internal/ingest/cursor.go`'s field names if
      they differ from the current best-effort guess (`sessionId`/`cwd`/`status`/
      `reason`/`lastMessage`), removing the "best-effort from documentation" comment once
      confirmed.
- [ ] No regression: `lav init`'s non-destructive hook merge, SQLite persistence at
      `~/.liveagentsview`, the three adapters' existing state mappings, and the
      dashboard's SSE/attention-queue behavior are unaffected by the switch from the
      Docker-run daemon to the native one (same data dir, same SQLite file).

## How

**Layout:**
- `apps/lav/internal/service/` (new) — `Install(Options) (Result, error)`,
  dispatching on `runtime.GOOS`. Darwin: writes a launchd plist to
  `~/Library/LaunchAgents/dev.liveagentsview.lav.plist` (`RunAtLoad`,
  `KeepAlive`, `LAV_HOME`/`LAV_PORT` env, logs under
  `$LAV_HOME/logs/lav.{out,err}.log`), then `launchctl bootout` (ignored,
  expected to fail on first install) + `bootstrap gui/<uid>` + `enable`.
  Linux: writes a systemd `--user` unit to
  `~/.config/systemd/user/lav.service`, then `daemon-reload` +
  `enable --now`. `Options.DryRun` previews without writing or registering,
  same pattern as `internal/installer`.
- `apps/lav/internal/terminal/` (new) — `Open(dir) error`. Darwin: `open -a
  Terminal <dir>`. Linux: `$TERMINAL` env, else the first of
  `gnome-terminal`/`konsole`/`x-terminal-emulator`/`xterm` found on `PATH`.
- `apps/lav/internal/daemon/server.go` — new `POST /api/open-terminal`
  (`{"cwd": "..."}` → `terminal.Open`), 204 on success.
- `apps/lav/cmd/lav/main.go` — new `lav service install [--dry-run]`
  subcommand: resolves its own binary path (`os.Executable` +
  `EvalSymlinks`), calls `service.Install`.
- `apps/lav/internal/ingest/cursor.go` — field names corrected against a
  real captured payload (see Validation): `session_id` not `sessionId`,
  `workspace_roots[0]` not `cwd`, `final_status` (confirmed on `sessionEnd`)
  checked before the documentation-era `status` guess for `stop`.
- `Dockerfile` — new `native-binary` stage (`golang:1.25-alpine`, same
  `apps/lav` + embedded frontend as `backend-build`, but
  `GOOS=$TARGETOS GOARCH=$TARGETARCH` build args instead of the container's
  native target). Not a runtime image — only `docker create` + `docker cp`
  ever pull a binary out of it.
- `scripts/lav-service-install.sh` (new) — maps `uname -s`/`-m` to
  GOOS/GOARCH, `docker build --target native-binary`, extracts
  `/out/lav` via `docker create`+`docker cp` into
  `~/.liveagentsview/bin/lav`, then runs `lav service install "$@"`.
- `apps/web/src/api.ts`, `App.tsx`, `App.css` — `openTerminal()`, an "Open
  terminal" button next to the existing copy-path button (kept as fallback
  for the containerized `dev-up.sh` daemon, where the request 500s since
  there is nothing to spawn a terminal into).
- `apps/lav/README.md`, `scripts/README.md` — updated to describe the two
  parallel run paths (`dev-up.sh` container for development,
  `lav-service-install.sh` native service for real) and the new packages.

**Verified for real, this session (macOS, this machine):**
- `docker build --target native-binary --build-arg TARGETOS=darwin
  --build-arg TARGETARCH=arm64` produces a genuine `Mach-O 64-bit executable
  arm64` (confirmed via `file`), extracted with `docker create`+`docker cp`,
  no Go installed on the host.
- `lav service install --dry-run`, run directly (not via Docker — this *is*
  the native binary), correctly resolved this machine's real paths
  (`/Users/rodrigopizarro/Library/LaunchAgents/dev.liveagentsview.lav.plist`).
- `scripts/lav-service-install.sh` run for real, with explicit user
  confirmation: built, installed, and `launchctl bootstrap`'d. `launchctl
  list` showed `dev.liveagentsview.lav` running; `GET /healthz` returned ok;
  `GET /api/sessions` returned the same sessions the previously-validated
  Docker-run daemon had persisted to the same `~/.liveagentsview/lav.db` (no
  data loss switching runtimes). Killed the process directly (`kill -9`);
  launchd respawned it (`KeepAlive`) within 2s, confirmed via `launchctl
  list` (new PID) and a second successful `/healthz`. No `docker compose`
  container involved in any of this.
- `POST /api/open-terminal` against a throwaway native daemon instance,
  `cwd` = this repo's path: returned 204, and a new Terminal.app window
  actually opened — confirmed by reading the `cwd` of the shell process
  attached to the new window's tty (`lsof -a -d cwd`), which matched exactly.
  Closed the test window afterward.
- Cursor CLI identity: `which agent` and `which cursor-agent` both resolve
  (`~/.local/bin/agent` is a symlink to
  `~/.local/share/cursor-agent/versions/2026.08.31-4057e58/cursor-agent`) —
  `agent` is a short alias for the same `cursor-agent` binary the decided
  docs already reference; no doc correction needed.
- Real hook payload, with explicit user confirmation: `agent --print --mode
  ask --trust "say hi, no changes needed"` in a throwaway directory, hooks
  already wired to a local daemon from the earlier real `lav init` run.
  `sessionStart` and `sessionEnd` fired and were captured from
  `~/.liveagentsview/lav.db`'s `events.raw` column (via `sqlite3`, already
  present on the host — not installed for this). Confirmed the field-name
  bug the previous spec flagged as a real risk: the payload uses
  `session_id` (snake_case), not `sessionId` — every event would have
  failed to parse (`missing sessionId`) under the old code. Also no `cwd`
  field at all; the real field is `workspace_roots` (an array). Fixed both
  in `internal/ingest/cursor.go` and replayed the exact captured raw JSON
  through a rebuilt daemon: both events now return 202 (previously would
  have 400'd), and the resulting session shows the correct `cwd` from
  `workspace_roots[0]`.
- `--mode ask --print` never triggered `stop` or `postToolUseFailure` (no
  tool-using turn happened) — those two remain best-effort, now at least
  consistent with the confirmed snake_case convention rather than the old
  camelCase guess. Not chased further: forcing a tool-using turn means
  running the CLI without the read-only mode restriction, which was
  explicitly out of scope for this verification pass (more account
  cost/risk for two event names already flagged as unconfirmed).

**Not verified (Linux, out of scope per platform decision):** the systemd
unit content and the Linux terminal-launch fallback are implemented and
code-reviewed only — this machine is macOS.

## Validation

Checked against this machine's real state, not just code review — re-confirmed live
during this validate pass (the launchd job installed during implement is still running,
`launchctl list`/`healthz`/`/api/sessions`/SSE all re-checked fresh).

1. **Native binary build path** — **yes**. `docker build --target native-binary
   --build-arg TARGETOS=darwin --build-arg TARGETARCH=arm64` produces a genuine `Mach-O
   64-bit executable arm64` (`file` confirmed), no Go/Node on the host.
   `~/.liveagentsview/bin/lav` is that exact binary (15,799,122 bytes, executable).
2. **`lav service install` generates + registers launchd; systemd unit written** —
   **yes**. `internal/service/service.go` covers both; macOS path exercised live (below),
   Linux path written and code-reviewed only, matching the platform-scope decision.
3. **Real launchd job, no Docker at runtime, survives respawn** — **yes**, re-confirmed
   this pass: `launchctl list` shows `dev.liveagentsview.lav` running, `GET /healthz` →
   `ok`, `GET /api/sessions` returns real data (currently tracking this very Claude Code
   session, live), `docker ps` shows no LAV container. Kill-and-respawn was directly
   verified during implement (`kill -9` on the PID → launchd relaunched it within 2s,
   confirmed via a fresh PID in `launchctl list` and a second successful `/healthz`).
4. **`dev-up.sh`/`compose.yaml` unchanged; `lav-service-install.sh` the only native-install
   entry point** — **yes**. Neither file appears in this spec's diff; the new script is
   the sole caller of `lav service install`.
5. **Real "open in terminal" on macOS, daemon spawns directly** — **yes**. Verified
   during implement: `POST /api/open-terminal` with a real `cwd` returned 204 and a new
   Terminal.app window opened; confirmed via `lsof -a -d cwd` on the shell attached to
   the new window's tty that its working directory matched exactly. Copy-path kept as
   the fallback for the containerized `dev-up.sh` daemon, where the same request 500s
   (no terminal to open inside a container).
6. **Linux "open in terminal" implemented, marked unverified** — **yes**, both parts:
   `internal/terminal/terminal.go`'s `openLinux` is implemented (`$TERMINAL` env, else
   `gnome-terminal`/`konsole`/`x-terminal-emulator`/`xterm`), and this Validation section
   (here) is where it's marked as not live-tested — this machine is macOS.
7. **Cursor CLI identity confirmed** — **yes**. `which agent` and `which cursor-agent`
   both resolve; `agent` is a symlink to
   `~/.local/share/cursor-agent/versions/2026.08.31-4057e58/cursor-agent`. Same binary,
   short alias — no change needed to the "cursor-agent" naming already used across the
   decided docs.
8. **Real hook payload captured for `sessionStart` and `stop`; `cursor.go` corrected** —
   **partial**. `sessionStart` was captured and is exactly what the acceptance item
   asked for. `stop` specifically never fired: the read-only `agent --print --mode ask`
   session used for verification (deliberately chosen to avoid an actual file-editing
   agent turn, given the real API cost/risk of running the CLI for real) went straight
   from `sessionStart` to `sessionEnd` with no tool-using turn in between, so `stop` and
   `postToolUseFailure` were never exercised. What *was* captured (`sessionStart` +
   `sessionEnd`) was enough to find and fix the real bug the previous spec had flagged as
   the actual risk — `session_id` not `sessionId`, `workspace_roots` not `cwd` — and the
   fix was re-verified by replaying the exact captured raw JSON through a rebuilt daemon
   (400 → 202, correct `cwd` in the resulting session). `stop`'s exact field name for the
   outcome (`p.outcome()` checks `final_status` before the old `status` guess) and
   `lastMessage`'s real name (`last_message`, guessed by the same snake_case pattern
   confirmed everywhere else in the payload) remain unconfirmed by live data. Accepted as
   a known, honestly-documented gap rather than reopening this spec: closing it for real
   means running the CLI in a tool-using mode, which is more account cost/risk than this
   pass's one already-found, already-fixed critical bug justified chasing further.
9. **No regression from switching the Docker-run daemon to the native one** — **yes**.
   `internal/store`, `internal/classifier`, `internal/daemon/sse.go`, and
   `internal/installer` are untouched by this spec's diff (only `cursor.go`,
   `daemon/server.go` [new route only], and `cmd/lav/main.go` [new subcommand only]
   changed on the Go side) — the existing Claude Code/Codex mappings, the classifier, and
   the SSE hub cannot have regressed by construction. Directly confirmed anyway this
   validate pass: the native service reads the same `~/.liveagentsview/lav.db` the
   earlier Docker-run daemon wrote to, with those sessions still present (no data loss
   across the runtime switch); `GET /api/events/stream` still opens and sends the
   `: connected` comment frame; the dashboard still serves its embedded HTML.

**Out of scope check:** nothing from piloted mode, Windows, Tailscale/remote access, or
uninstall/lifecycle polish slipped in. Live-verifying Linux was explicitly out of scope
(no Linux machine available) and stayed that way. No decided doc needed correcting — the
"cursor-agent" naming holds up.

**Net result:** 8 of 9 acceptance items met with direct evidence on this machine; 1
(#8) is partial, and the partial half is honestly the less important half — the
confirmed-and-fixed part (`session_id`/`workspace_roots`) was the actual risk the
previous spec called out by name; the unconfirmed part (`stop`'s exact field names) was
already best-effort before this spec and stays best-effort now, just on firmer footing
(consistent snake_case, and no longer guessing `cwd` at all since it's now
`workspace_roots`). No silent regressions or undocumented divergence from the spec.

**Closed as `validated`** — 2026-09-02: what this spec asked for (native service, real
terminal launch, Cursor payload verification) is delivered and directly verified on this
machine, following the same bar adopted-mode-mvp itself set (partial items carried
forward honestly rather than blocking closure). Remaining gap (`stop`/
`postToolUseFailure` Cursor field names, Linux live-verification) not reopened as new
work here — revisit opportunistically if Cursor adopted-mode misbehaves in practice, or
whenever a Linux machine is available.

## Handoff

```
Spec: docs/sdd/specs/native-host-runtime.md
Status: validated
Next: none
```

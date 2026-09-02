---
title: Adopted-mode MVP — daemon, three provider adapters, attention dashboard
slug: adopted-mode-mvp
status: validated
created: 2026-09-01
updated: 2026-09-01
next: none
chain: none
---

# Spec: Adopted-mode MVP — daemon, three provider adapters, attention dashboard

## Intent

A user who runs Claude Code, Codex or Cursor natively (no wrapper, no change to their
workflow) can `lav init` once and from then on see every local session — repo, worktree,
branch, provider, normalized state — in one dashboard at `127.0.0.1`, with an attention
queue that tells them which sessions actually need them right now (blocked on a
permission, failed) versus which merely finished (grouped, quiet). The daemon survives
dashboard restarts and machine reboots without losing events.

## Out of scope

- **Piloted mode.** Driving a session as a child process over `stream-json`, answering
  questions or approving permissions from the dashboard. Posture B
  ([03-decisions.md](../../03-decisions.md), 2026-09-01) commits to building this, but
  adopted mode alone already delivers the primary value from
  [01-vision.md](../../01-vision.md) (see all agents, know who needs you) and is a
  self-contained increment. Follow-up spec.
- **Tailing fidelity.** Retroactive JSONL parsing. Not needed for v1: the daemon runs as
  an always-on user service, so Hooks fidelity is live for every session started after
  `lav init`. Revisit only if backfilling sessions from before the daemon existed becomes
  a real need.
- **Remote / Tailscale access.** Explicit later opt-in per the stack decision
  ([03-decisions.md](../../03-decisions.md), 2026-09-01). Bind to `127.0.0.1` only.
- **Human presence detection** (Q-06) and the P3 "suspicious / quiet too long" attention
  level (IDEA-01's row beyond P2 is still unagreed,
  [05-ideas-to-discuss.md](../../05-ideas-to-discuss.md)). Not part of acceptance.
- **Auto-resolve on any route** (IDEA-03) and **per-session throttling/dedup** (IDEA-04).
  Both unagreed — real quality-of-life follow-ups, not required here. Do not build them
  as if decided.
- **Service install polish** (auto-start on login across all edge cases, uninstall,
  upgrade-in-place). `lav init` must get the daemon running as a user service, but
  hardening that is not an acceptance item.

## Already decided

- Posture: observer + opt-in pilot, adopted class is read-only via hooks —
  [02-scope.md](../../02-scope.md), [03-decisions.md](../../03-decisions.md) 2026-09-01
  "Posture: observer + opt-in pilot".
- Stack: Go, SQLite in `~/.liveagentsview/`, single self-contained binary, daemon as a
  user service (launchd/systemd), HTTP+SSE on `127.0.0.1`, frontend embedded in the
  binary — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Stack".
- Frontend stack: React + Vite, compiled to static assets and embedded —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Frontend stack".
- Providers: Claude Code, Codex, Cursor, all three built together in this spec, Cursor's
  constraint accepted as a limitation to design around —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "MVP providers" and "Cursor's
  adapter is built alongside...".
- Docker is for developing/testing LiveAgentsView, not for running it; the binary runs on
  the host — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Docker...".
- `lav init` merges hooks non-destructively into existing config, with a preview before
  writing — [03-decisions.md](../../03-decisions.md) 2026-09-01 "`lav init` merges
  hooks...".
- End-of-turn disambiguation (WAITING vs DONE) is rules-based, behind a pluggable
  classifier interface — [03-decisions.md](../../03-decisions.md) 2026-09-01 "End-of-turn
  classification".
- "An agent finished" is low-priority, grouped, silent attention; BLOCKED and FAILED are
  not — [03-decisions.md](../../03-decisions.md) 2026-09-01 "'An agent finished' is
  low-priority attention".
- Canonical states and the provider signal contract —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Canonical event/state model...".
  Full table below.
- LLM credentials are never managed; filesystem permissions are delegated to the
  underlying agent; a server that can approve permissions is RCE, bind `127.0.0.1` —
  [02-scope.md](../../02-scope.md) "Explicit boundaries".

## Event model (the adapter contract)

Canonical states: **WORKING · WAITING · BLOCKED · DONE · FAILED · IDLE**.

| State | Claude Code (Hooks) | Codex (Hooks-equivalent: `notify`) | Cursor (Hooks: `.cursor/hooks.json`) |
|---|---|---|---|
| WORKING | Inferred: session active, no terminal event yet since last `UserPromptSubmit`/tool event | No signal — Codex hooks are silent mid-turn. Inferred: active since last `notify`. | `preToolUse`/`postToolUse`/`beforeShellExecution`/`afterShellExecution`/`afterFileEdit` sequence |
| WAITING (turn over, expects reply) | `Stop` fires (no reason field) → classify `last_assistant_message`. `Notification:agent_needs_input`/`idle_prompt` is a direct hint when present. | `notify` type `agent-turn-complete` → classify `last-assistant-message`. No separate signal from DONE. | `stop` with `status:"completed"` → classify last message. No separate signal from DONE. |
| BLOCKED (needs a permission decision) | `PermissionRequest` / `Notification:permission_prompt` — direct, dedicated signal | **No signal at Hooks fidelity — confirmed gap.** Codex resolves approvals by policy outside interactive hook delivery. Document as a known v1 limitation for Codex adopted sessions. | **No dedicated event — confirmed gap.** No BLOCKED detection for Cursor adopted sessions in this spec; do not fake it with the shell-exec-hook heuristic (staff-acknowledged as unreliable). |
| DONE | Same raw signal as WAITING (`Stop`) — same classifier call | Same raw signal as WAITING (`notify`) — same classifier call | Same raw signal as WAITING (`stop:status="completed"`) — same classifier call |
| FAILED | `StopFailure` — dedicated, no classification needed | **No signal at Hooks fidelity.** Codex adopted sessions cannot report FAILED in this spec. | `stop:status="error"` / `sessionEnd:reason="error"` / `postToolUseFailure` — dedicated, no classification needed |
| IDLE | Never pushed by any provider. Derive locally: no event for N minutes after DONE/WAITING, or before the first event. | Same — derive locally. | Same — derive locally. |

Notes for whoever implements:

- The end-of-turn classifier (already decided, pluggable) is the single component that
  turns "turn ended" into WAITING vs DONE for all three providers — implement it once,
  feed it `last_assistant_message` / `last-assistant-message` / the last assistant text
  regardless of provider.
- Codex has no BLOCKED and no FAILED signal at Hooks fidelity. This is a real product gap
  for v1, not a bug: a Codex session stuck on an approval, or one that crashed, looks
  identical to WORKING from the adopted adapter's point of view until/unless a `notify`
  eventually fires. Surface the fidelity level per session in the UI (already decided,
  [02-scope.md](../../02-scope.md)) so this is visible, not silently wrong.
- **Not implemented, on purpose:** timeout-based IDLE derivation ("no event for N
  minutes"). It ties directly to the unagreed P3/"suspicious" row of IDEA-01 — deciding
  a threshold is a product call, not an implementation detail. What ships instead: IDLE
  only from the direct signals that already mean it (Claude Code `SessionEnd`, Cursor
  `sessionEnd` without an error reason). A session sitting in WORKING/WAITING forever
  with no further events just stays there in the UI rather than being demoted.
- **Open verification, not a blocker:** confirm early whether `.cursor/hooks.json` fires
  for sessions launched natively from the Cursor IDE, or only for `cursor-agent`
  CLI-launched sessions (confirmed: at least `beforeShellExecution`, `afterShellExecution`,
  `afterFileEdit`, `postToolUse`, `stop`, `sessionStart` fire for the CLI). If IDE-launched
  sessions do not fire hooks, Cursor's "adopted" class in this spec is scoped to sessions
  started via `cursor-agent` in a terminal, not the Cursor IDE's own agent panel — note
  this explicitly in the UI/docs rather than silently under-supporting it.

## Acceptance

- [ ] `lav init` detects existing `~/.claude/settings.json`, `~/.codex/config.toml`, and
      `.cursor/hooks.json` (if present), shows a preview of exactly what it will add or
      change, and on confirmation merges LiveAgentsView's hooks/`notify` entry without
      deleting or overwriting any existing entry (in particular, Codex's existing
      `notify` program keeps firing after `lav init`).
- [ ] The daemon stays running unattended and survives restarts. Implemented as a Docker
      Compose service (`restart: unless-stopped`) rather than a native launchd/systemd
      user service — see How. A native service is a real gap, not silently dropped: it
      matters once this stops being a "validate it works" pass and needs to survive a
      full host reboot without Docker Desktop being manually reopened.
- [ ] The daemon binds only to `127.0.0.1` and embeds the built frontend — no separate
      Node process, no separate static file server.
- [ ] SQLite at `~/.liveagentsview/` persists every session's identity (repo, worktree,
      branch, provider) and current state, surviving a daemon restart.
- [ ] Claude Code adapter maps `PermissionRequest`/`Notification:permission_prompt` →
      BLOCKED, `StopFailure` → FAILED, `Stop`/`SubagentStop` → classifier → WAITING or
      DONE, matching the table above.
- [ ] Codex adapter maps the `notify` `agent-turn-complete` payload → classifier →
      WAITING or DONE. BLOCKED and FAILED are not claimed for Codex adopted sessions in
      this spec (per the documented gap).
- [ ] Cursor adapter maps `stop`/`sessionEnd`/`postToolUseFailure` per the table above.
      BLOCKED is not claimed for Cursor adopted sessions in this spec.
- [ ] The end-of-turn classifier is implemented behind an interface (not hardcoded inline
      per provider) and used identically by all three adapters.
- [ ] Every session shown in the dashboard displays its integration fidelity level
      (Hooks, for all sessions in this spec) so gaps (Codex BLOCKED/FAILED, Cursor
      BLOCKED) are legible rather than silently absent.
- [ ] The dashboard (React + Vite) lists all known sessions with repo/worktree/branch/
      provider/state, updates live over SSE without a manual refresh, and surfaces an
      attention queue where BLOCKED and FAILED sessions are immediately visible and DONE
      sessions are grouped and unobtrusive.
- [ ] From the dashboard, the user can jump to the originating session. Implemented as a
      "copy path" action, not a literal one-click terminal launch — the daemon runs
      inside a container in this spec and cannot spawn anything on the host. A real "open
      in the terminal" needs either a small native helper on the host or piloted-mode
      infrastructure (out of scope here); copy-path is the honest interim.
- [ ] `lav` is usable from the CLI (at minimum `lav init` and a way to check the daemon is
      running / see current sessions without opening the browser).

## How

Built and verified against a real Docker build + the real local `~/.claude` and
`~/.codex` config (Cursor and Codex adapters only verified with simulated payloads —
`cursor-agent` and `codex` are not installed on this machine).

**Layout:**
- `apps/lav/` — Go module. `cmd/lav` (CLI: `serve`, `init`, `status`);
  `internal/model` (canonical types); `internal/ingest` (one parser per provider →
  `model.Signal`); `internal/classifier` (rules-based end-of-turn classifier);
  `internal/store` (SQLite via `modernc.org/sqlite`, pure Go, no cgo); `internal/daemon`
  (HTTP routes + SSE hub); `internal/installer` (`lav init`'s non-destructive hook merge);
  `web` (`go:embed` of the built frontend into `web/static`, named `static` not `dist` to
  avoid the repo-wide `.gitignore` rule for Node build output).
- `apps/web/` — React + Vite dashboard, plain CSS, no UI library. Fetches
  `GET /api/sessions` once, then live-updates from `GET /api/events/stream` (SSE).
- Root `Dockerfile` — three stages: `node:22-alpine` builds `apps/web`, its `dist/` is
  copied into `golang:1.25-alpine` at `apps/lav/web/static` before `go build`, final
  image is `alpine:3.20` + the static binary. `go.mod`/`go.sum` are committed and pinned
  (`go mod tidy` was run once against a real network, not hand-written) so the Docker
  build uses `go mod download`, not a fresh resolve every time.
- `compose.yaml` — the `lav` service only, published on `127.0.0.1:${LAV_PORT:-8420}`
  **explicitly** (Docker's default port publish binds `0.0.0.0`, which would violate the
  decided 127.0.0.1-only boundary — caught during testing, not obvious from the compose
  file alone, worth remembering if anyone edits the `ports:` line later), `restart:
  unless-stopped`, SQLite bind-mounted straight to the host's `~/.liveagentsview`.
- `compose.dev.yaml` — adds bind mounts for `~/.claude`, `~/.codex`, `~/.cursor` (needed
  only by `lav init`, not by `lav serve`) plus `LAV_HOST_HOME`/`LAV_HOME_HOST_PATH` so
  hook commands `lav init` writes reference real host paths, not container-internal ones
  — this indirection is the one genuinely non-obvious piece of the whole setup; see
  `apps/lav/internal/installer/installer.go`'s `Options` doc comments before touching it.
- `scripts/dev-up.sh`, `dev-down.sh`, `lav-init.sh [--dry-run]`, `lav-status.sh` — the
  only supported entry points, per AGENTS.md ("nothing is run by hand").

**Verified for real, this session:** Docker build (frontend + backend), `go vet` clean,
daemon start/health/dashboard load, SQLite persistence at the real
`~/.liveagentsview/lav.db` across a container recreate, live SSE updates in a browser,
`lav init --dry-run` against the real `~/.claude/settings.json` (no hooks existed →
previews adding 7) and real `~/.codex/config.toml` (an existing `notify` target for
another tool → previews chaining, not replacing), 127.0.0.1-only binding after the fix
above, `lav status`. The classifier was exercised with a real question ("Should I use
React or Svelte for this?" → waiting) and a real completion message → done.

**Also run for real, after explicit user confirmation:** the actual (non-dry-run) `lav
init` write against this machine's real config. Verified afterward: `~/.claude/settings.json`
still valid JSON with all 7 hooks added; `~/.codex/config.toml` still has all 18 of its
original `[section]` blocks intact and its `notify` line now points at the chain wrapper,
which still calls the original SkyComputerUseClient target before forwarding to the
daemon; `~/.cursor/hooks.json` created correctly; all three helper scripts written and
executable under `~/.liveagentsview/bin/`.

Native launchd/systemd service install and Cursor's exact hook payload field names (see
`internal/ingest/cursor.go`) remain the two biggest unknowns — see Validation.

## Validation

Checked against a real Docker build and, where noted, this machine's real Claude Code/
Codex/Cursor config — not just code review.

1. **`lav init` non-destructive merge** — **yes**. Ran for real (not just `--dry-run`)
   against this machine's actual config with explicit user go-ahead. `settings.json`
   stayed valid JSON with 7 hooks added; `config.toml` kept all 18 of its original
   `[section]` blocks and its pre-existing `notify` target now fires from inside the
   chain wrapper instead of being replaced; `hooks.json` created correctly. Re-running
   `lav init` is idempotent (checked via the dedup logic in `installer.go`, not
   re-exercised live a second time).
2. **Daemon survives restarts** — **partial, as already flagged in Acceptance.** Docker
   Compose `restart: unless-stopped` verified (container recreated cleanly across three
   rebuilds this session). No native launchd/systemd unit — a genuine gap, not polish:
   without it the daemon does not come back after a host reboot unless Docker Desktop
   itself is set to relaunch and the compose service is manually brought up again.
3. **127.0.0.1-only binding, embedded frontend, no separate Node/static server** —
   **yes**. `docker port` confirmed `127.0.0.1:8420` (an earlier draft published on
   `0.0.0.0` by Docker's default — caught and fixed this session, see `compose.yaml`).
   Frontend is served from the same Go binary via `go:embed`; no separate process.
4. **SQLite persistence across restarts** — **yes**, directly verified: posted a real
   session event to a container, removed the container entirely, started a fresh one on
   the same bind-mounted directory, and the session was still there with its original
   data intact.
5. **Claude Code adapter mapping** — **yes**. Exercised live against a running daemon:
   `SessionStart` → working, `Notification/permission_prompt` → blocked, `StopFailure` →
   failed, `Stop` with a question ("Should I use React or Svelte for this?") → waiting via
   the classifier, `Stop` with a completion message → done via the classifier.
6. **Codex adapter mapping** — **yes, simulated only.** Payload shape matches Codex's
   documented `notify` schema and classifies correctly, but never exercised against the
   real `codex` CLI (not installed on this machine) or Codex's own approval flow.
7. **Cursor adapter mapping** — **partial.** Logic matches the researched event names and
   `stop.status`/`sessionEnd.reason` handling, but the JSON field names
   (`sessionId`/`cwd`/`lastMessage`) are best-effort from docs, not confirmed against a
   real `cursor-agent` payload (not installed anywhere this was built or tested). Real
   risk: if the actual field names differ, every Cursor event will fail to parse
   (`missing sessionId`) until corrected.
8. **Classifier behind a shared interface** — **yes**. One `classifier.Classifier` used
   identically by all three adapters in `daemon/server.go`; confirmed via the live
   question-vs-completion test above.
9. **Fidelity level shown per session** — **yes**, visible in the dashboard (confirmed by
   screenshot) as a tag on every session card.
10. **Dashboard: list, live SSE, attention grouping** — **yes**. Confirmed by screenshot
    before and after posting a new event with no manual refresh, correctly grouped into
    "Needs you now" / "Failed" / "Asked you something" / "Finished".
11. **Jump to originating session** — **partial, as already flagged in Acceptance.**
    Shipped as a "copy path" button, not a real terminal launch — the containerized
    daemon has no path to spawn anything on the host.
12. **`lav` CLI usable** — **yes**. `lav init` (dry-run and real), `lav status` all
    exercised for real via `docker compose exec`/`run`.

**Out of scope check:** nothing from piloted mode, tailing, remote access, human presence/
P3, auto-resolve, or throttling slipped in. The "copy path" button is new relative to the
original spec text but is a partial implementation of an *in-scope* acceptance item
(jump to session), not scope creep.

**Net result:** 9 of 12 acceptance items fully met with direct evidence, 3 are partial —
all 3 were already called out honestly in Acceptance/How during implementation, not
discovered here. None are silent regressions or undocumented divergence.

**Closed as `validated`** — Rodrigo, 2026-09-01: what this spec asked for (adopted mode
working end to end, runnable locally via Docker) is delivered and verified for real. The
3 partial items become candidate follow-up specs rather than blocking this one:
1. Native launchd/systemd service install (replace the Docker restart policy stand-in).
2. A real "open in the terminal" (native host helper, or fold into piloted-mode work).
3. Verify Cursor's hook payload field names against a real `cursor-agent` install and
   fix `internal/ingest/cursor.go` if they differ.

## Handoff

```
Spec: docs/sdd/specs/adopted-mode-mvp.md
Status: validated
Next: none
```

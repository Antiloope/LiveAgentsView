---
title: Adopted-mode MVP — daemon, three provider adapters, attention dashboard
slug: adopted-mode-mvp
status: ready
created: 2026-09-01
updated: 2026-09-01
next: implement
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
- [ ] The daemon runs as a user service (launchd on macOS at minimum; systemd if time
      allows) started by `lav init`, survives terminal/dashboard closing, and restarts on
      login.
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
- [ ] From the dashboard, the user can jump to the originating session (open the terminal
      / reveal the working directory) for any adopted session — "open in the terminal" as
      a first-class action.
- [ ] `lav` is usable from the CLI (at minimum `lav init` and a way to check the daemon is
      running / see current sessions without opening the browser).

## How

Left to whoever implements. Suggested shape, not binding:

- `apps/` gets the daemon (Go module) and the frontend (React+Vite) as two directories
  under one `apps/lav/` or split `apps/daemon` + `apps/web`, embedded via `go:embed` at
  build time into the daemon binary — confirm the exact layout against
  [apps/README.md](../../../apps/README.md) rules before creating it.
- Hook receivers: a local HTTP endpoint the hook scripts POST to, or a small helper binary
  invoked by the hook config that forwards to the daemon over a local socket — either
  works, pick whichever keeps `lav init`'s generated config simplest.
- Start with the Cursor-IDE-vs-CLI hooks verification (see Event model notes) before
  writing the Cursor adapter, since it decides the adapter's actual scope.
- `scripts/` gets whatever is needed to build and run this locally per
  [scripts/README.md](../../../scripts/README.md) (Docker for dev only, per the 2026-09-01
  decision).

## Validation

Filled in by whoever validates.

## Handoff

```
Spec: docs/sdd/specs/adopted-mode-mvp.md
Status: ready
Next: implement
```

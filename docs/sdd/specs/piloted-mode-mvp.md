---
title: Piloted-mode MVP — child-process driver, live chat, permissions, resume
slug: piloted-mode-mvp
status: done
created: 2026-09-02
updated: 2026-09-02
next: validate
chain: specify
---

# Spec: Piloted-mode MVP — child-process driver, live chat, permissions, resume

## Intent

From the LiveAgentsView dashboard, the user launches a Claude Code or Cursor session
against an existing local directory as a **piloted** child process, watches its full
transcript live as a chat, sends it free-form messages, interrupts or cancels it, and
resumes it later — without touching a terminal. For Claude Code this runs over a real
bidirectional `stream-json` channel with live permission approve/deny; for Cursor it is a
degraded posture forced by the real capabilities of its CLI (see "Already decided" below):
one-shot invocations chained by the provider's own resume flag, auto-approving every tool
call. Piloted sessions share the same canonical states, the same end-of-turn classifier,
and the same attention queue as adopted sessions.

## Out of scope

- **Codex piloted adapter.** Codex stays Hooks-only (as shipped in
  [adopted-mode-mvp](adopted-mode-mvp.md)); its CLI is not confirmed installed/in PATH on
  this machine and it has no confirmed bidirectional driver surface. Only Claude Code and
  Cursor get a piloted (Driver-level) adapter in this spec.
- **Automatic git worktree creation.** Launching a piloted session targets a directory
  (and, optionally, an existing branch to check out) the user already has on disk.
  LiveAgentsView does not create worktrees or branches on the user's behalf here — a real
  gap for the "run many agents in parallel without stepping on each other" vision, but a
  separate, explicit follow-up.
- **tmux-owned sessions (IDEA-06).** Piloted sessions run as plain child processes of the
  daemon. They do not survive a daemon restart as a *live* process, and there is no "attach
  to the exact same live session from any terminal" — only adopted sessions get a real
  terminal launch (`internal/terminal`, from
  [native-host-runtime](native-host-runtime.md)). Explicitly deferred, not silently
  dropped: revisit only if losing in-flight piloted work on a daemon restart (with the
  machine still on) turns out to matter in practice. The narrower "come back tomorrow"
  need is covered instead by provider-native resume (see Acceptance).
- **Configurable permission mode per session.** Claude Code piloted sessions always launch
  with its default permission mode (prompts before risky actions, approved/denied from the
  dashboard); Cursor piloted sessions always launch auto-approving (see "Already decided").
  There is no UI to pick a different mode in this spec. Noted here explicitly as deferred
  product surface, not an oversight — a real follow-up once the fixed defaults are
  dogfooded.
- **Interactive Cursor permission approval.** Confirmed absent from the installed CLI (see
  "Already decided") — not something this spec's implementation chose to skip.
- **Diff viewer / rich rendering.** The transcript view renders the stream's messages,
  tool calls and results as a plain chat-like log. No side-by-side diff viewer, syntax
  highlighting, or other polish beyond what's needed to read the conversation.
- **Human presence detection (IDEA-02), the full attention priority taxonomy (IDEA-01),
  auto-resolve across routes (IDEA-03), throttling/dedup (IDEA-04).** All still unagreed;
  unchanged from [adopted-mode-mvp](adopted-mode-mvp.md)'s out-of-scope list.
- **Remote / Tailscale access.** Unchanged: bind `127.0.0.1` only, explicit later opt-in
  per [02-scope.md](../../02-scope.md) and [03-decisions.md](../../03-decisions.md).

## Already decided

- Posture: sessions launched from LiveAgentsView are **piloted** — a child process over
  bidirectional `stream-json`, answering questions and approving permissions from the
  dashboard — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Posture: observer +
  opt-in pilot". Capability table in [02-scope.md](../../02-scope.md): piloted can
  "Answer questions, approve/deny permissions, cancel — without leaving the dashboard."
- **Driver** is the highest integration fidelity level, full events and full control,
  requires the session to be launched by LiveAgentsView —
  [02-scope.md](../../02-scope.md) "Integration surfaces".
- Stack: Go daemon, SQLite in `~/.liveagentsview/`, single self-contained binary, HTTP+SSE
  on `127.0.0.1` — [03-decisions.md](../../03-decisions.md) 2026-09-01 "Stack". The daemon
  runs on the host (not containerized) specifically because **piloted mode depends on**
  host keychain, repos, git config and worktrees —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Docker is for developing
  LiveAgentsView, not for running it".
- React + Vite was chosen in part for "a chat-like stream for piloted sessions" —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Frontend stack". This spec is what
  that rationale was written for.
- Canonical states (WORKING/WAITING/BLOCKED/DONE/FAILED/IDLE) and the rules-based,
  pluggable end-of-turn classifier are reused as-is, not reimplemented —
  [03-decisions.md](../../03-decisions.md) 2026-09-01 "Canonical event/state model" and
  "End-of-turn classification". The "turn ended" ambiguity (WAITING vs DONE) exists at
  Driver fidelity exactly like at Hooks fidelity — the raw signal doesn't disambiguate
  either way, so the same classifier interface from `internal/classifier`
  ([adopted-mode-mvp](adopted-mode-mvp.md)) applies unmodified.
- "An agent finished" stays low-priority, grouped attention, including for piloted DONE
  — [03-decisions.md](../../03-decisions.md) 2026-09-01 "'An agent finished' is
  low-priority attention".
- Filesystem permissions stay delegated to the underlying agent; LiveAgentsView surfaces
  the prompt and routes the answer, it does not build a sandbox of its own —
  [02-scope.md](../../02-scope.md) "Explicit boundaries". This spec is the first one that
  actually exercises that boundary (previous specs were read-only).
- A local server that can approve permissions is RCE; bind `127.0.0.1` only —
  [02-scope.md](../../02-scope.md) "Explicit boundaries". Nothing in this spec changes
  that binding or adds new remote exposure.
- Cursor's real CLI identity is confirmed on this machine: `agent` is a short alias for
  `cursor-agent` (same binary) — [native-host-runtime](native-host-runtime.md) Acceptance/
  How. Its **Hooks**-level payload field names are confirmed
  (`session_id`/`workspace_roots`/`final_status`).
- Cursor's CLI has no bidirectional driver protocol and does not pause for interactive
  permission approval in `--print` mode; piloted Cursor sessions auto-approve via
  `--force`/`--yolo` and run as one-shot invocations chained by `--resume`/`--continue`
  instead of a persistent stdin channel — confirmed live against the installed CLI this
  session, [03-decisions.md](../../03-decisions.md) 2026-09-02 "Cursor piloted sessions
  auto-approve, no live permission gate".
- Claude Code's Driver-level bidirectional channel is already verified as a real
  capability of the installed CLI (not just documentation): `-p --output-format
  stream-json --input-format stream-json`, plus `--resume`, `--continue`,
  `--fork-session`, `--session-id` — [00-inbox.md](../../00-inbox.md) "What was verified
  on the machine".
- Discovered during this implementation, against this machine's real, already-running
  `lav` service (see Validation): a piloted session's own CLI process still fires its
  normal lifecycle hooks against whatever daemon `lav init` wired it to — those hooks have
  no notion of "I was launched by LiveAgentsView, don't demote me". A Hooks-fidelity event
  for a session already tracked at Driver fidelity must not overwrite it back to Hooks —
  fixed in `internal/daemon/server.go`'s `handleHook`.

## Acceptance

- [ ] From the dashboard, the user starts a new piloted session by picking a provider
      (Claude Code or Cursor), an existing local directory already on disk, optionally an
      existing git branch to check out before launching, and an initial prompt.
      LiveAgentsView does not create a worktree or a branch as part of this flow.
- [ ] The daemon launches the session as a plain child process of itself, never a terminal
      emulator or tmux. Claude Code: `claude -p --input-format stream-json --output-format
      stream-json --session-id <uuid> --permission-mode default`, a single long-lived
      process for the life of the session. Cursor: `agent -p --output-format stream-json
      --force --trust`, one new process per message, chained with `--resume <chatId>`.
- [ ] The dashboard has a dedicated live session view showing a chat-like transcript:
      every assistant message, tool call and tool result from the stream renders as it
      arrives (plus permission requests for Claude Code), delivered over SSE with no
      manual refresh.
- [ ] From that view, the user can send a new free-form message to a Claude Code piloted
      session at any time, including while it is WORKING — delivered to the live process
      over its stdin as a new turn. For Cursor, a new message can only be sent once the
      previous one-shot invocation has finished or been interrupted (there is no live
      stdin channel to interrupt mid-thought); the UI states this rather than silently
      queuing or dropping the message.
- [ ] For Claude Code, when the stream carries a permission request, the dashboard shows a
      dedicated approve/deny control; the answer is written back to the process over stdin
      and the underlying tool call proceeds or is denied accordingly. Cursor piloted
      sessions never show this control — every tool call is pre-approved via `--force`
      (see "Already decided"), and the UI says so rather than implying a gate that isn't
      there.
- [ ] Two distinct controls exist for an in-progress piloted session: **interrupt** and
      **cancel**. For Claude Code, interrupt stops the current turn but keeps the live
      process attached so the user can keep chatting; cancel terminates the process and
      ends the session (a later message still works, via Resume). For Cursor, a one-shot
      invocation has no "current turn" separate from "the process running it", so
      interrupt and cancel are the same operation: kill whatever's running, keep the
      session's chat id so the next message can still `--resume` it. Claude Code's
      mid-turn interrupt uses its documented stream-json control protocol; the exact
      request/response shape is not live-verified in this session (see Validation) and is
      corrected there if real testing shows it differs.
- [ ] Piloted sessions use the same canonical state model and the same end-of-turn
      classifier instance/interface as adopted sessions — no parallel implementation.
- [ ] Piloted sessions appear in the same dashboard and the same attention queue as
      adopted sessions, tagged with their "piloted" class, with inline actions (send
      message, approve/deny, interrupt, cancel) in place of adopted's "open in
      terminal"/"copy path".
- [ ] SQLite persists each piloted session's identity (provider, directory, branch,
      initial prompt, the provider-native session id) and its full transcript, so the
      conversation is visible in the dashboard even after the daemon restarts and the
      underlying child process is gone.
- [ ] When the daemon starts and finds a piloted session with no live process attached
      (daemon restart, machine restart, interrupt, or a crash), the dashboard reflects that
      honestly (not silently shown as WORKING) and offers a **resume** action. For Claude
      Code, resume starts a new `claude ... --resume <sessionId>` process against the
      persisted session id, re-attaching a live stdin channel. Cursor has no equivalent of
      a standing process to re-attach (confirmed: `agent --help` lists `--resume [chatId]`/
      `--continue` as flags to a new one-shot invocation, not a way to reopen a live
      channel) — resume just marks the session ready for another message, which itself
      carries `--resume <chatId>`.
- [ ] Every new endpoint/action this spec adds stays bound to `127.0.0.1` only — no new
      remote-exposure surface, reaffirming the existing RCE boundary now that this spec is
      the one actually executing agent processes and routing permission approvals.
- [ ] Permission mode is fixed per provider: Claude Code's own default (prompts before
      risky actions, resolved from the dashboard) for Claude Code, `--force`/`--yolo`
      (auto-approve) for Cursor. No per-session configuration UI in this spec.
- [ ] No regression: adopted-mode sessions, hooks ingestion, and the existing dashboard/
      attention-queue behavior for adopted sessions are unaffected by piloted mode's
      addition.

## How

**Layout:**
- `apps/lav/internal/pilot/` (new) — `pilot.go`: `Manager` (per-session bookkeeping,
  registry keyed by session id, `Launch`/`SendMessage`/`ApprovePermission`/`Interrupt`/
  `Cancel`/`Resume`/`Events`/`Hub`/`ReconcileOnStartup`), the `Event` transcript-entry type,
  `finalizeProcess` (shared exit handling: a user-requested stop reads as IDLE, an
  unrequested one as FAILED). `claude.go`: Claude Code's driver — long-lived process,
  stdin-write helpers for a new user turn / permission response / interrupt request, a
  line-by-line stdout parser (`system`/`assistant`/`control_request`/`result`, unknown
  types passed through raw rather than dropped). `cursor.go`: Cursor's driver — one-shot
  `agent -p` invocation per message chained by `--resume`, blocks briefly on launch to read
  cursor-agent's self-assigned `session_id` off its init line (no `--session-id` flag
  exists the way Claude Code has one), `killCursor` backs both Interrupt and Cancel.
- `apps/lav/internal/sse/` (new) — `Hub`, extracted verbatim from the old
  `daemon/sse.go` (deleted) so both the dashboard's global stream and each piloted
  session's transcript stream share one implementation instead of two copies.
- `apps/lav/internal/store/store.go` — `ListEvents` (oldest-first, for a piloted session's
  history replay before its live SSE stream takes over). No schema migration: piloted
  transcript entries reuse the existing generic `events` table (`hook_event` prefixed
  `pilot:`, `raw` holds the marshaled `Event`).
- `apps/lav/internal/daemon/server.go` — `pilot.Manager` wired into `Server`, seven new
  `POST/GET /api/piloted/sessions/...` routes (Go 1.22+ method+`{id}` mux patterns),
  `handleLaunchPiloted` validates provider/cwd/prompt and does a plain `git checkout
  <branch>` (never `-f`) before handing off to `Manager.Launch`. The existing
  `handleHook`'s fidelity-clobber fix (see "Already decided") lives here too.
- `apps/web/src/types.ts`, `api.ts` — `PilotEvent`/`PilotProvider` types; `launchPilotedSession`,
  `sendPilotMessage`, `resolvePilotPermission`, `interruptPilotedSession`,
  `cancelPilotedSession`, `resumePilotedSession`, `fetchPilotEvents`,
  `subscribeToPilotEvents` (a second, per-session `EventSource`, independent of the
  dashboard's existing global one).
- `apps/web/src/PilotedSessionView.tsx` (new) — the live chat view: transcript, compose
  box (gated per-provider — Claude Code always sendable except idle/failed, Cursor only
  when not mid-turn), permission approve/deny cards, interrupt/cancel/resume buttons.
- `apps/web/src/App.tsx`, `App.css` — "New piloted session" form, a "View chat" button on
  driver-fidelity session cards in place of adopted's open-terminal/copy-path.
- `Dockerfile`, `compose.yaml`, `apps/lav/README.md` — comments corrected: piloted
  sessions spawn real processes against the host filesystem/login state, so (unlike pure
  hooks ingestion) they only work when `lav` runs natively, not in
  `scripts/dev-up.sh`'s container.
- `docs/03-decisions.md` — one new entry, "Cursor piloted sessions auto-approve, no live
  permission gate", logging the scope change this spec's own verification forced.

**Verified for real, this session:**
- `go build`/`go vet` clean via the existing Docker build stages; frontend `npm run build`
  and a separate `tsc --noEmit` (the plain `vite build` script does not type-check) both
  clean.
- Cross-compiled the real `darwin/arm64` native binary (same `native-binary` Docker stage
  as [native-host-runtime](native-host-runtime.md)) and ran it standalone, pointed at an
  isolated scratch `LAV_HOME`/port — never the real running service's data — to exercise
  the actual HTTP API end to end:
  - Launched a real piloted **Cursor** session against a scratch directory: correct
    `fidelity:"driver"` session created, transcript captured, classified `done` on a
    non-question reply.
  - Sent a real follow-up message to that session (`--resume` chaining): second turn
    executed for real, transcript and `last_message` updated correctly.
  - Killed the daemon mid-turn and restarted it against the same data directory:
    `ReconcileOnStartup` correctly flipped the stale `working` session to `idle` instead of
    leaving it silently wrong; a message sent after the restart correctly reconstructed the
    session from SQLite (the `resolve()` path) and resumed it via `--resume` with no data
    loss.
  - Exercised the validation/error paths for real: missing prompt, non-`claude-code`/
    `cursor` provider, nonexistent `cwd`, interrupt/cancel on an idle session, an unknown
    session id, and Cursor's "turn already in progress" 409 (fired a second message while
    the first was still running) — all returned the intended status code and message.
  - Launched a real piloted **Claude Code** session: the process spawned, its
    `system`/`assistant`/`result` stream-json envelope parsed correctly, and — because this
    execution environment cannot authenticate the installed `claude` CLI (confirmed
    separately: even a plain `claude -p "hi"` returns "Not logged in" here, unrelated to
    piloted mode) — the `is_error:true` result correctly finalized the session as `FAILED`
    with the real error text, rather than hanging or crashing the daemon.
  - Regression-checked the existing Hooks path against the same isolated instance: a
    synthetic `POST /hooks/claude-code` still creates/updates a Hooks-fidelity session
    correctly, and — the specific fix above — a fake hook payload for a session already at
    Driver fidelity is correctly ignored rather than downgrading it.
- **Not verified**: Claude Code's live permission-approval and mid-turn-interrupt control
  protocol (`control_request`/`control_response` "can_use_tool"/"interrupt"). This needs an
  authenticated `claude` process actually reaching a tool call that requires approval,
  which this environment cannot produce (see above). The wire shapes in `claude.go`
  (`claudePermissionResponse`, `buildInterruptRequest`) are implemented against Claude
  Code's documented Agent SDK control protocol, not confirmed against a real payload —
  flagged in code comments as the one place to correct if a real run shows a different
  shape.
- **Also found and fixed, unrelated to this spec's own code**: this machine's real,
  already-running `lav` service already had Claude Code hooks wired (from
  `lav init`/[native-host-runtime](native-host-runtime.md)); several of this session's
  direct CLI verification commands (run before and during implementation, to check the
  real driver protocols) fired real hook events against that live service, and briefly
  bringing up `docker compose`'s dev stack for a regression check pointed a second process
  at the same bind-mounted `~/.liveagentsview` the live service was using. Five stray test
  session rows this produced were identified by exact id/cwd and deleted from the real
  database with the user's explicit confirmation (a backup of the file was taken first);
  the live service was otherwise unaffected (`healthz` checked immediately after). A
  separate, likely pre-existing bug this surfaced — some hook payload shaped like a
  Cursor background-agent event landing under `provider:"claude-code"` in the real
  database — is unrelated to piloted mode and was flagged as a follow-up investigation
  rather than chased down here.

## Validation

_Filled in during validate — this implementation pass's own verification is recorded in
How above, not here; validate is a separate, colder review._

## Handoff

```
Spec: docs/sdd/specs/piloted-mode-mvp.md
Status: done
Next: validate
```

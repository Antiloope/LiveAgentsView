---
title: Character model — vocabulary, two axes, territory, lifecycle
slug: character-model-redesign
status: ready
created: 2026-09-03
updated: 2026-09-03
next: implement
chain: specify
---

# Spec: Character model — vocabulary, two axes, territory, lifecycle

## Intent

Replace the session model with the character model decided on 2026-09-03: one
provider-neutral vocabulary, activity and presence as two independent things, a territory
that is either a LiveAgentsView-administered worktree or a directory left exactly as it
is, a daemon whose belief about what is running is continuously reconciled against what
actually runs, archiving that frees the memory a character holds, and no permission
mediation at all.

When this is done, nothing in the interface reads as Claude-only or Cursor-only, a
character never reports an activity it is not in, and no character holds memory the user
did not ask it to hold.

## Out of scope

- **The CSRF / missing-Origin-check finding.** The API accepts a cross-origin simple POST
  and decodes the body regardless of `Content-Type`, so any web page can launch an
  auto-approving character on this machine. Real and more urgent than this spec, but
  independent of it — see the 2026-09-03 block in [00-inbox.md](../../00-inbox.md). It
  gets its own fix; do not fold it in here.
- **The HP/MP bars.** They are seeded random numbers presented as telemetry
  (`seededPercent` in `apps/web/src/sprites.ts`), and they are documented in
  [DESIGN.md](../../../DESIGN.md) as part of the visual system. Removing invented data is
  right, but it is a design call for the maintainer, not a consequence of this redesign.
  Leave them alone.
- **Preparing a fresh worktree** with what git does not track (`.env`, `node_modules`,
  build output) — [Q-11](../../04-open-questions.md). A character in a brand-new own
  territory may land in a repo that does not build. Accepted for now.
- **Which archetypes belong to which race** — [Q-10](../../04-open-questions.md). This
  spec implements the rule and proposes pools; the visual assignment is the maintainer's.
- **A permission layer** ([IDEA-11](../../05-ideas-to-discuss.md)) and **changing a
  character's class after creation** ([IDEA-12](../../05-ideas-to-discuss.md)). Both
  explicitly deferred.
- **A Codex adapter.** Codex is a race with no adapter; it stays unrepresented.
- **The attention queue and notifications.** The unread mark is built here because it
  replaces the `done` state; consuming it into a prioritized queue is
  [IDEA-01](../../05-ideas-to-discuss.md) and is not built here.
- **Quests as persisted objects** — decided against, 2026-09-03.

## Already decided

Everything this spec implements is decided in [03-decisions.md](../../03-decisions.md),
2026-09-03, seven entries:

- **"Vocabulary: character, race, class, quest"** — character / race (engine, immutable) /
  class (model, changeable) / territory / quest / camp. The engine is never a concept of
  its own in the interface. The sprite follows race, not class.
- **"Two axes: what a character is doing, and whether it is awake"** — activity is
  `ready` / `working` / `waiting` / `failed`; presence is awake or asleep and is not
  something the user manages; `done` becomes an unread mark; `idle` and `blocked` are
  removed. Supersedes the 2026-09-01 canonical state model.
- **"Territory: own worktree by default, shared directory as the explicit alternative"** —
  and LiveAgentsView never runs `git checkout` on a directory the user picked.
- **"Consistency: presence is observed, never stored"** — the runner is the authority, the
  daemon reconciles continuously, orphans are swept at startup.
- **"Archiving sends a character to sleep; dismissing removes it"** — supersedes the
  2026-09-02 archive decision on both the process side effect and the `working`
  restriction.
- **"Permission management is dropped; every race runs auto-approving"** — and
  **"A quest is not a modelled object"**.

Also standing and unchanged: piloted-only posture (2026-09-02), the Go/SQLite/single
binary stack and `127.0.0.1` binding (2026-09-01), the pluggable end-of-turn classifier
(2026-09-01), and "an agent finished is low-priority attention" (2026-09-01), which the
unread mark now implements directly.

Confirmed live on this machine while specifying, not assumed:

- `claude --permission-mode` accepts `bypassPermissions` (also `auto`, `dontAsk`,
  `acceptEdits`, `manual`, `plan`). `agent -f/--force` is "Force allow commands unless
  explicitly denied". Both flags exist on the installed CLIs.
- A Claude Code process launched with no prompt writes nothing to stdout at all until its
  first message — its transcript file stays 0 bytes. This is why launching as `working`
  never self-corrects.
- A character that never received a message held 129 MB RSS after 3h27m, plus 10 MB for
  its `pilot-runner`.

## Open questions

None. Q-10 and Q-11 are open in [04-open-questions.md](../../04-open-questions.md) and are
both out of scope above; neither blocks implementation.

## Acceptance

### Model and storage

- [ ] `internal/model` exposes `Character` with: `ID` (LiveAgentsView's own, stable for
      the character's life), `SessionID` (the provider-side conversation id, empty until
      the first launch reports it), `Race`, `Class`, `Activity`, `Unread`, `Territory`,
      `Repo`, `Archived`, `LastMessage`, `CreatedAt`, `UpdatedAt`.
- [ ] `Activity` has exactly four values: `ready`, `working`, `waiting`, `failed`. No
      `idle`, `blocked` or `done` value exists anywhere in the codebase.
- [ ] **No field on `Character` says whether a process is alive.** Presence is never
      written to SQLite.
- [ ] `Territory` carries `Mode` (`own` | `shared`), `Path` (where the process actually
      runs), `Source` (the repo a worktree came from; equal to `Path` when shared) and
      `Branch`.
- [ ] The existing database on this machine migrates idempotently, in the style of the
      current `ensureColumn`: `provider`→race, `model`→class, `cwd`→territory path with
      mode `shared`, and state→activity as `working`→`working`, `waiting`→`waiting`,
      `blocked`→`waiting`, `done`→`ready` with `unread` set, `failed`→`failed`,
      `idle`→`ready`. Existing rows survive; running the migration twice changes nothing.
- [ ] A character's `ID` is generated by LiveAgentsView, not taken from the provider.
      `SessionID` is recorded when the provider reports it.

### Presence and consistency

- [ ] The API reports presence as a computed field (`awake` | `asleep`), derived from
      whether the character's `pilot-runner` control socket is reachable — never from a
      stored value.
- [ ] A reconciler runs on an interval for as long as the daemon runs (not only at
      startup) and broadcasts only on change.
- [ ] A character whose activity is `working` and whose runner has disappeared (`kill -9`
      of the runner, machine sleep, OOM) is moved to `failed` by that reconciler within
      one interval, with a transcript entry saying the process disappeared. Verify by
      killing a real runner while its character is mid-quest.
- [ ] A daemon restart does not change any character's activity by itself: a runner that
      is still alive is reconnected and its character is untouched.
- [ ] At startup the daemon sweeps orphans: runner sockets, transcript files and
      worktrees whose character no longer exists.

### Waking, and talking to a character

- [ ] There is no resume action in the API or the interface. Sending a message to an
      asleep character wakes it — a new process with the provider's own resume flag — and
      then delivers the message, in one call, for every race.
- [ ] A message sent while a character is `working` is accepted and delivered when the
      current quest ends, for every race. It is never rejected with "a turn is already in
      progress".
- [ ] A character created without a first message is `ready` and awake, sits in camp, and
      can be talked to. It is never `working`. This is the reported bug that started the
      redesign; verify against a real Claude Code character created with an empty prompt.
- [ ] Creating a character requires no first message for any race. A Cursor character
      created with no message is `ready` and asleep, which is indistinguishable to the
      user from an awake one.
- [ ] `Interrupt` and `Stop` are offered only while a character is `working`.

### The unread mark

- [ ] A quest that ends without a question sets `unread` and leaves activity `ready`.
- [ ] A quest the classifier reads as a question sets activity `waiting` and does **not**
      set `unread`.
- [ ] `unread` is cleared by an explicit call the interface makes when the user reads the
      character's transcript, not by any state change.
- [ ] `waiting` and `failed` clear when the user acts (answers, stops, archives), never by
      being looked at.

### Territory

- [ ] Creating a character offers two territory modes, with **own** as the default.
- [ ] Own territory: LiveAgentsView creates a git worktree under
      `~/.liveagentsview/worktrees/`, on a new or existing branch, and the character's
      process runs there. The user's chosen directory is not modified in any way.
- [ ] Shared territory: the character runs in the chosen directory as it is, and
      LiveAgentsView runs **no** git command against it. In particular `git checkout`
      appears nowhere in the launch path.
- [ ] A branch that is already checked out in another worktree is reported as a clear
      error at creation, not a raw git failure.
- [ ] A directory that is not a git repository can only host a shared territory; own
      territory is unavailable and the interface says why.

### Lifecycle

- [ ] Archiving stops the character's process and removes it from camp, in **any**
      activity including `working`, after a confirmation that names what will be stopped.
      Verify that the process is really gone (no `claude`/`agent` child, no runner).
- [ ] An archived character keeps its full transcript and its territory. Unarchiving
      returns it to camp; talking to it wakes it with its context.
- [ ] Dismissing a character stops its process, deletes its row, its events and its
      transcript files, and removes its worktree **only** when that worktree has no
      uncommitted changes — otherwise the worktree is left and the user is told where it
      is.
- [ ] Archiving a character that is already asleep does not fail.

### Permissions removed

- [ ] Claude Code characters launch with `--permission-mode bypassPermissions` and without
      `--mcp-config` / `--permission-prompt-tool`.
- [ ] `internal/pilotmcp`, the `lav pilot-mcp` subcommand, the `permission_request` /
      `permission_response` wire ops, the daemon's pending-request map, the
      `ApprovePermission` path, the HTTP permission route and the permission transcript
      events are all **deleted**, not disabled.
- [ ] The interface has no approve/deny control and no "Cursor auto-approves" note — there
      is no longer a per-race difference to explain.

### Interface

- [ ] Every user-visible string uses the decided vocabulary: character, race, class,
      territory, quest, camp. "Session", "provider", "model", "agent" and "piloted" do not
      appear in the interface.
- [ ] The sprite is determined by race. Changing what class a character would run does not
      change its sprite, and two characters of the same race are still visually
      distinguishable from each other.
- [ ] Class is shown as a label or badge on the character, not by its body.
- [ ] The camp holds every non-archived character that is not `working`; the sidebar holds
      the ones that are.
- [ ] The empty state no longer mentions `lav init` or sessions started natively — neither
      exists.

### Bugs in the same code, fixed while here

- [ ] A live transcript event that arrives while a character's history is still loading is
      not lost (today `setEvents(list)` overwrites it).
- [ ] Stopping a character that has no live process does not leave the "stopped by user"
      flag set for a later, genuine crash to consume.
- [ ] The interrupt flag cannot be set when there is no turn to interrupt, so a later
      genuine `error_during_execution` is never reported as a clean finish.

### Build

- [ ] `go build ./...`, `go vet ./...` and `gofmt -l .` clean via the repo's Docker dev
      path; `npx tsc --noEmit` and `npm run build` clean in `apps/web`;
      `scripts/check-doc-citations.sh` clean.
- [ ] Verified live against a real daemon on this machine, through the real browser UI —
      not only compiled.

## How

Implementation notes, not a contract. Whoever implements may take a different route as
long as the acceptance items hold.

**Suggested order.** Each phase leaves the app working:

1. Permissions removal (pure deletion, unblocks everything else and shrinks the surface).
2. Model, migration, and the vocabulary rename through the Go side and the API.
3. Presence + reconciler, then waking-on-send, then the unread mark.
4. Territory (worktrees).
5. Lifecycle (archive-as-sleep, dismiss).
6. Frontend vocabulary, sprites and the removed resume/permission controls.

**Character id vs provider session id.** Today a session's id *is* the provider's id,
which is why `internal/pilot/cursor.go` needs `launchCursorBootstrap`: `cursor-agent`
assigns its own id, so the first turn cannot have a `pilot-runner` socket named after an
id nobody knows yet, and it runs as a direct child of the daemon with no restart
continuity. Giving a character its own id from the start removes that special case
entirely — the runner is named after the character, and the provider's id is recorded when
the `system/init` line reports it. Deleting `launchCursorBootstrap` and its
`readCursorBootstrapStdout` twin is expected, not incidental.

**Presence.** `net.DialTimeout` on `pilotwire.SocketPath` is already the exact check
`reconnect` uses; make it a small exported helper and call it from both the reconciler and
the API serializer. The reconciler is a goroutine started by `daemon.New`, ticking every
few seconds, comparing observed presence against activity and correcting only the
`working`-but-gone case.

**`readFromRunner`'s deliberate silence.** It currently leaves the persisted state
untouched on a plain socket disconnect, because it cannot tell "the daemon is shutting
down" from "the runner died". With the reconciler in place it does not have to: leave that
behavior as it is and let the reconciler decide, since it can dial again and find out.

**Waking and queuing.** `SendMessage` becomes: resolve the character; if asleep, spawn its
runner with the race's resume flag and wait for attach; if `working`, hold the text in the
character's in-memory queue and flush it when the turn's result line arrives; otherwise
write it now. Cursor's "one process per turn" and Claude Code's "one resident process"
both disappear behind this one method.

**Territory.** `git worktree add <path> [-b <branch>] <branch-or-commit>` under
`~/.liveagentsview/worktrees/<repo>/<character-id>`. `git worktree remove` on dismissal,
gated on `git status --porcelain` being empty in that worktree. `git worktree list
--porcelain` for the "branch already checked out elsewhere" check, so the error is a
sentence rather than git's stderr. The `git checkout` in `handleLaunchPiloted` goes away
with no replacement.

**Sprites (proposal only — [Q-10](../../04-open-questions.md) is the maintainer's).**
`archetypeFor(characterID, race)`: each race owns a pool of archetypes and the character
id hashes into that pool, which keeps both properties at once — same race reads as the
same kind of creature, individuals stay distinguishable. The existing
`CLAUDE_MODEL_ARCHETYPE` map is deleted; the "→ arrives a Dragonkin Warrior" line on the
class cards moves to the race choice or goes away.

**Files most affected.** `internal/model/model.go`, `internal/store/store.go` (migration),
`internal/pilot/*.go` (the manager, both adapters, the deleted bootstrap path),
`internal/pilotrunner/runner.go` (argv, deleted permission relay),
`internal/pilotwire/wire.go` (deleted ops), `internal/daemon/server.go` +
`recruit.go` (routes, territory), `cmd/lav/main.go` (deleted subcommand); deleted:
`internal/pilotmcp/`. Frontend: `types.ts`, `api.ts`, `App.tsx`, `RecruitPanel.tsx`,
`SessionDrawer.tsx`, `PartyStand.tsx`, `QuestToken.tsx`, `Portrait.tsx`, `sprites.ts`.

**Superseded acceptance in an open spec.** `recruit-flow-redesign` is `done`, awaiting
validation, and three of its acceptance items are overtaken by this one: the empty-prompt
Claude Code launch being "visible as `working`", the required first message for Cursor,
and the archetype following the model. That spec has been annotated; validate it against
its own state at the time, not against this redesign.

## Validation

Filled in by whoever validates.

## Handoff

```
Spec: docs/sdd/specs/character-model-redesign.md
Status: ready
Next: implement
```

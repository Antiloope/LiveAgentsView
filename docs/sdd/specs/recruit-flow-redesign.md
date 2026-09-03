---
title: Recruit-flow redesign — model picker, native Finder, branch dropdown
slug: recruit-flow-redesign
status: done
created: 2026-09-03
updated: 2026-09-03
next: validate
chain: specify+implement
---

# Spec: Recruit-flow redesign

## Superseded in part (2026-09-03)

Three acceptance items below are overtaken by
[character-model-redesign](character-model-redesign.md), which implements the
2026-09-03 decisions in [03-decisions.md](../../03-decisions.md):

- "A Claude Code piloted session launched with an empty prompt starts a real, attached
  process (visible as `working`, ...)" — launching as `working` with nothing to do is the
  bug that started that redesign. A character with no first message is now `ready`.
- Cursor's required first-message field — no race requires one now, because a character
  is woken by the message that gives it a quest rather than launched with it.
- "A Claude Code party member's sprite reflects its recruited class" — the sprite now
  follows race, since class is the changeable half.

Validate this spec against what it set out to do at the time, not against the redesign.

## Intent

Replace the "Recruit a piloted session" modal (`NewPilotedSessionForm` in
`apps/web/src/App.tsx`) with the approved "Camp Ledger" panel design
(https://claude.ai/code/artifact/00165b47-9d6b-496b-a293-ffb71023982b): a
class-card model picker for Claude Code, Cursor's own live model catalog
(Auto pinned + grouped/filterable), a native macOS folder picker for the
target directory, and a real branch dropdown for that directory — instead
of free-typed `cwd`/`branch` text inputs and a plain provider `<select>`.

The first prompt moves out of the recruit form and into the session
drawer's existing compose box for Claude Code, where recruiting seats an
idle companion at camp and the user gives it its first task from the chat
it will keep using for every task after. Cursor keeps a first-message field
in the recruit panel itself — see "Already decided" for why that one case
does not move.

## Out of scope

- Any OS other than macOS for the folder picker (`osascript`). No
  fallback path for Linux/Windows.
- Tying `session.model` into anything beyond archetype-on-recruit and
  display: no model-based routing, pricing, or limits.
- Changing Codex, which is not a piloted provider today and stays that way.
- A "recent folders" list or manual path re-typing — the only way to set
  the territory is the native picker (matches "no free-typed cwd").
- Retroactively backfilling `model` for sessions launched before this
  change (column defaults to empty string; UI treats that as unknown).

## Already decided

- Provider-agnostic piloted launch, Claude Code + Cursor only in this MVP —
  `PRODUCT.md` "Providers in MVP" / `docs/03-decisions.md`.
- "LLM credentials are never managed" — model *selection* is a `--model`
  CLI flag passed to the subprocess the same way `--resume`/`--session-id`
  already are; no key or account handling involved.
- Visual direction is locked: the "Camp Ledger" mock at the URL above,
  itself built on the shipped `DESIGN.md` (Ember Camp world) — no new
  concept round needed, this is an existing-surface extension.
- Confirmed live on this machine (not assumed): `claude --help` lists
  `--model <model>` accepting aliases `fable`/`opus`/`sonnet` (haiku is the
  well-established fourth tier in the same family, used here on that
  convention); `cursor-agent --list-models` (invoked as `agent` — the repo
  already spawns cursor-agent under that binary name, confirmed via
  `internal/service`'s PATH handling) is a real flag returning ~217 models
  with `auto - Auto (current, default)` first.

## Decisions made while specifying (no open questions — chained straight to implement)

- **Cursor cannot start idle.** Every Cursor turn is its own one-shot
  `cursor-agent -p ... <prompt>` process (see `internal/pilot/cursor.go`'s
  package doc) — there is no "process waiting on stdin" the way Claude
  Code's persistent process is. So only Claude Code's recruit flow defers
  the prompt to the drawer; Cursor's recruit panel keeps a required
  "First message" field, with a one-line note explaining why. This is a
  real CLI constraint, not a UX preference — documented here so it does
  not get "fixed" into an inconsistency later without cause.
- **Empty prompt at launch is already supported for Claude Code** —
  `launchClaude` in `internal/pilot/claude.go` only writes to stdin
  `if spec.Prompt != ""`. No backend change needed there beyond relaxing
  the HTTP-layer requirement to be provider-conditional.
- **`--model` is passed only when non-empty.** The UI always preselects one
  of Opus/Sonnet/Haiku for Claude Code and Auto for Cursor, so in practice
  the flag is always sent for both providers; omitting it (rather than a
  magic default string) is what happens if it's ever empty, so the
  provider's own CLI default silently applies instead of erroring.
- **Model persists per session** (`model.Session.Model`) and is passed
  through `Resume` (Claude Code) and every subsequent turn (Cursor's
  `sendCursorMessage`) so a session keeps the model it was recruited with
  for its whole life.
- **Archetype follows model, for Claude Code's 3 known models only**
  (opus→Dragonkin Warrior, sonnet→Elf Rogue, haiku→Halfling, matching the
  approved mock's "→ arrives as..." line). Cursor sessions and any
  unrecognized model string keep today's random-per-session-id pick —
  small, contained change to `archetypeFor` plus threading `model` through
  `Portrait`'s existing 3 call sites (`PartyStand`, `QuestToken`,
  `SessionDrawer`), all of which already hold the full `session` object.
- **Territory → Trail is a hard dependency.** The branch `<select>` is
  empty/disabled until a directory is chosen; if that directory is not a
  git repo, the daemon returns an empty branch list rather than an error
  and the panel shows "not a git repository" instead of blocking
  recruitment (a piloted session can run in a non-repo directory today).
- **Recruit still goes through `POST /api/piloted/sessions`** for both
  providers — no new "pending session" concept. Claude Code just sends
  `prompt: ""`; Cursor sends whatever was typed in its required field.

## Acceptance

- [ ] `model.Session` has a `Model` field, persisted (new `sessions.model`
      column, migrated like `archived` was) and returned by `/api/sessions`
      and the SSE stream.
- [ ] `POST /api/piloted/sessions` accepts `model`; rejects (400) an empty
      `prompt` only when `provider` is `cursor`; accepts empty `prompt` for
      `claude-code`.
- [ ] A Claude Code piloted session launched with an empty prompt starts a
      real, attached process (visible as `working`, capable of receiving a
      message) with no initial stdin write.
- [ ] `--model <value>` reaches the spawned `claude`/`agent` process for a
      fresh launch, and for Claude Code's `--resume` and Cursor's
      per-message relaunch, whenever `Model` is non-empty.
- [ ] `POST /api/pick-directory` opens a native macOS folder-choose dialog
      and returns the chosen absolute path; a cancelled dialog is not
      surfaced as an error.
- [ ] `GET /api/branches?cwd=<path>` returns the current branch and the
      local branch list for a git repo at `cwd`, and an empty list (not an
      error) for a non-repo directory.
- [ ] `GET /api/cursor-models` returns Cursor's live model list (id +
      label), cached for 5 minutes.
- [ ] The recruit panel (new `RecruitPanel.tsx`, replacing
      `NewPilotedSessionForm`) matches the approved "Camp Ledger" mock:
      provider toggle; Claude Code shows 3 class cards (Opus/Sonnet/Haiku)
      with id, flavor line, "good for" tags, Depth/Speed bars, arrives-as
      line; Cursor shows Auto pinned + a filterable list grouped by id
      prefix family; a Territory field driven only by the native picker;
      a Trail `<select>` gated on Territory; Cursor additionally shows a
      required first-message field.
- [ ] Recruiting a Claude Code session with no message closes the panel,
      selects the new session, and opens the drawer with its compose box
      focused.
- [ ] A Claude Code party member's sprite reflects its recruited class
      (Opus/Sonnet/Haiku → fixed archetype); everything else is unchanged
      from today's random-per-session-id pick.
- [ ] `go build ./...` (via the repo's Docker dev path) and
      `npm run build` (`apps/web`) both pass.

## How

- Backend: `internal/model/model.go` (+`Model` field). `internal/store`:
  `sessions.model` column via a generalized `ensureColumn` (the old
  `ensureArchivedColumn` was folded into it, same behavior for `archived`),
  threaded through `UpsertSession`/`GetSession`/`ListSessions`/`scanSession`.
  `internal/pilot/pilot.go`: `LaunchSpec.Model`, `pilotSession.model`, a
  `modelArgs(m string) []string` helper (nil when empty), `upsert`/
  `resolve`/`reconnect`/`Resume` all carry `model` the same way they already
  carry `branch`; `spawnRunner` prepends `modelArgs(ps.model)` to every
  `lav pilot-runner` invocation. `claude.go`/`cursor.go`: set `ps.model`
  from spec at launch; `launchCursorBootstrap`'s direct `exec.Command` and
  `sendCursorMessage`'s relaunch spec both carry it too, since Cursor has no
  persistent process to inherit it from. `internal/pilotrunner/runner.go`:
  new `--model` flag, appended to both providers' argv when non-empty.
  New `internal/daemon/recruit.go`: `handlePickDirectory` (osascript
  `choose folder`, 204 on cancel), `handleListBranches` (`git branch
  --show-current` / `--format`, empty list for a non-repo cwd rather than
  an error), `handleCursorModels` + `cursorModelsCache` (5-minute TTL,
  regex-parsed `agent --list-models` output — verified against a real
  captured run: 217 models, header/tip lines correctly skipped). Routes
  registered in `server.go`; `handleLaunchPiloted` gained `Model` and now
  only requires `Prompt` when `provider == cursor`.
- Frontend: `types.ts` (+`model` on `Session`, `ClaudeClassId`,
  `CursorModelOption`). `api.ts`: `+model` on `launchPilotedSession`,
  `pickDirectory`/`fetchBranches`/`fetchCursorModels`. New
  `RecruitPanel.tsx` replaces `NewPilotedSessionForm` (deleted from
  `App.tsx`, along with the now-dead `PILOT_PROVIDERS`): provider toggle;
  Claude Code's 3 class cards (hardcoded copy/tags/bars, matching the
  approved mock); Cursor's Auto card + live catalog fetched once per panel
  open, grouped client-side by id-prefix family, filterable; a required
  first-message field shown only for Cursor. `sprites.ts`: `archetypeFor`
  takes an optional `model`, fixed-mapped for Claude's 3 classes via
  `CLAUDE_MODEL_ARCHETYPE`, falling back to the existing hash otherwise.
  `Portrait.tsx` + its 3 call sites (`PartyStand`, `QuestToken`,
  `SessionDrawer`) pass `session.model` through. `Glyphs.tsx`: 7 new pixel
  rune components (`ShieldRune`/`HoodRune`/`SatchelRune`/`MapRune`/
  `SignpostRune`/`SparkRune`/`SearchRune`), same grammar as the existing
  `ProviderRune`/`StatusIcon`. `SessionDrawer.tsx`: the compose textarea
  autofocuses when a session's event history is empty (what a fresh
  Claude Code recruit looks like), no new cross-component plumbing needed.
  `App.css`: new scoped classes for the panel (`.recruit-*`, `.cr-*`,
  `.rp-*`, `.model-*`, `.class-row`, `.auto-card`, `.chip`, `.field-btn`,
  `.horn-btn`, `.rune`) — ported from the approved mock rather than reusing
  sprite-bar classes, to avoid coupling two visually-similar but
  differently-constrained components. Also restored `.new-session-actions`
  after initially (incorrectly) deleting it as dead — it's still used by
  the Archived-sessions modal's Close button, caught by grepping actual
  JSX usage before finalizing.

Verification actually run: `npx tsc --noEmit` and `npm run build` clean;
Go backend built via `docker build --target backend-build` (exercises both
`go build` and the frontend build it embeds); `go vet ./...` and
`gofmt -l .` clean via a throwaway `golang:1.25-alpine` container;
`scripts/check-doc-citations.sh` clean; the `cursorModelLine` regex
verified against a real captured `cursor-agent --list-models` run (217/217
parsed, non-model lines correctly skipped). **Not verified**: an actual
click-through in a browser against a running daemon. `docker compose up`
on this machine hit a pre-existing, unrelated networking gap — the daemon
intentionally binds `127.0.0.1` only (`cmd/lav/main.go`, hardcoded, not
touched here), which Docker Desktop's port-publish cannot reach regardless
of what's inside the container (confirmed harmless/pre-existing: the same
`/healthz` on unmodified code has the identical symptom). The native
picker, branch listing, and Cursor catalog also cannot be exercised inside
that Linux container regardless (`osascript`, and `agent` don't exist
there) — same constraint every other piloted-session code path in this
repo already has. A real smoke test needs the native path
(`scripts/lav-service-install.sh`), which was not run here since it
replaces the maintainer's already-running service.

## Validation

(filled in when validated)

## Handoff

```
Spec: docs/sdd/specs/recruit-flow-redesign.md
Status: ready
Next: implement
```

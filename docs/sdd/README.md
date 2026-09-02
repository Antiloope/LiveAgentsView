# Implementation specs

Harness for agents (Cursor, Claude Code, Codex). **Does not define product.**
Definition lives in `docs/01-vision.md`, `docs/02-scope.md`, and `docs/03-decisions.md`.
A spec here does not close a decision: it cites it or, if missing, stops and asks.

Skill: `sdd` (`.agents/skills/sdd`). Invocation: `/sdd` in Cursor and Claude, `$sdd` in Codex.
Also activates on its own when the change is meaningful or a spec is already in play.

Not blocking. Small tasks or "just go" proceed without a spec.
If a spec **is in use**, keep it updated through the end (or mark it `abandoned`).

## Flow

`specify` → (human) → `implement` → (human) → `validate`

Can be chained: "specify and implement", "through validate".
Each step leaves a handoff block to paste into another agent.

## Specs

| Spec | Status | Next |
|---|---|---|
| [adopted-mode-mvp](specs/adopted-mode-mvp.md) | validated | none |
| [native-host-runtime](specs/native-host-runtime.md) | validated | none |
| [piloted-mode-mvp](specs/piloted-mode-mvp.md) | validated | none |
| [piloted-only-mode](specs/piloted-only-mode.md) | in-progress | implement |

Statuses: `draft` · `ready` · `in-progress` · `done` · `validated` · `abandoned`.

Template: [templates/spec.md](templates/spec.md). Files in [specs/](specs/).

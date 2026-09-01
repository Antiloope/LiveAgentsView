---
name: sdd
description: >
  Spec-driven workflow for this repo (SDD). Triages whether a change
  needs a spec, writes and keeps specs current in docs/sdd/specs/, and runs
  specify → implement → validate with optional chaining so small or quick
  work is not blocked. Also distills the docs inbox, decisions, open
  questions, and status. Use when the user proposes a feature, refactor,
  new module, or interesting repo change; mentions spec, SDD,
  inbox, distill, decisions, open questions, or docs/00–06; an
  open spec exists for the work; or they ask to specify, implement, or
  validate. Skip recommending a spec for tiny/small/quick work unless a
  spec is already in play — then it must stay updated through the end.
---

# SDD

Implementation specs in `docs/sdd/`. They are not product definition.
Before acting: list `docs/sdd/specs/*.md` (except the README) and read
[docs/sdd/README.md](../../../docs/sdd/README.md) if any open one matches the request.
If a spec is in play, load it and continue from `next`.

## Routing

| Request | What to load |
|---|---|
| Feature, refactor, module, "interesting change", no command | [reference/triage.md](reference/triage.md) |
| `specify` / "let's write the spec" / "define this" | [reference/specify.md](reference/specify.md) |
| `implement` / "build the spec" / "implement X.md" | [reference/implement.md](reference/implement.md) |
| `validate` / "validate the spec" / "review against the spec" | [reference/validate.md](reference/validate.md) |
| `inbox` / `distill` / `decide` / `status` | [reference/docs.md](reference/docs.md) |
| `/sdd` or `$sdd` with no arguments | Short menu from the table; do not start a step |

User-requested chain (`specify and implement`, `through validate`, `chain`):
run those steps in this session, updating the spec in between.
Without a chain, one step and handoff. The human is in the middle on purpose.

## Does not block, does not leave things half-done

- "quick", "small", "no spec", "just go": implement without a spec.
  One sentence offering a spec is enough; if they say no, do not insist.
- Spec **already open** for this work: do not skip it. Update it in the
  same turn until `validated` or `abandoned` (with rationale).
- Abandoned draft mid-way: mark `abandoned` or finish it. Never leave
  `draft`/`in-progress` orphaned.

## Definition vs spec

Writing code or a spec **does not close a definition**. If the request
invents product, stop: `docs/05-ideas-to-discuss.md` or ask a maintainer.
Cite what is decided; do not expand scope on your own.

## Handoff

When closing a step (or the chain), paste this:

```
Spec: docs/sdd/specs/<slug>.md
Status: <status>
Next: <specify|implement|validate|none>
```

Statuses: `draft` · `ready` · `in-progress` · `done` · `validated` · `abandoned`.
Template: [docs/sdd/templates/spec.md](../../../docs/sdd/templates/spec.md).
Index: table in `docs/sdd/README.md` — update it when creating, changing status, or closing.

# Validate

Check the code against the spec. Do not reimplement or "improve" scope.

## Before

Spec in `done` or `in-progress` (if they chained). If it is `draft`, there is nothing to validate.

## What to check

1. Each **Acceptance** item: yes / no / n/a, with evidence (file, test, behavior).
2. **Out of scope:** did anything slip in?
3. Does the spec describe what exists? If the code diverged and the spec was not updated, that is a failure:
   update the spec or mark the gap; do not leave both versions.
4. `AGENTS.md` rules the spec touched (definition vs code).

Not a generic style review. That is not this step.

## After

In the spec's **Validation** section: result per item and gaps.

- Everything covered: `status: validated`, `next: none`.
- Small gaps: list them; the user chooses whether to fix now or in another spec.
  Do not leave `in-progress` half-done: either close in this turn and mark `validated`,
  or `done` with gaps written and `next: implement`.
- Spec obsolete on purpose: `abandoned` and why.

Update the table in `docs/sdd/README.md`. Handoff. Stop.

# Definition docs

Specs in `docs/sdd/specs/` do not replace this flow. This is the source of truth.

## inbox

Dump raw into `docs/00-inbox.md`: one block with a date, unsorted, undistilled.
Nothing here is definition.

## distill

From the inbox to the right document (`01-vision`, `02-scope`, etc.).

- Exactly as whoever decided it said. Do not expand or "improve".
- Agent proposal or idea → `docs/05-ideas-to-discuss.md`, marked unagreed.
- Decision with date → also `docs/03-decisions.md`.
- Unresolved → `docs/04-open-questions.md` (numbers are not reused).
- Mark the inbox block as distilled.

## decide

Only when a maintainer said so. Entry in `03-decisions` with date, why, who.
If it closed a question in `04`, **remove it** from open questions and search for citations
to that number across the rest of `docs/` so nothing is left outdated.

## status

`docs/06-status.md` is a mirror. Update when a fact changed (code or docs).
Never to anticipate. Decides nothing.

## Relation to specs

An implementation spec cites what is decided. If distill/decide changes something an
open spec cited, update that spec in the same turn or mark it `abandoned`.

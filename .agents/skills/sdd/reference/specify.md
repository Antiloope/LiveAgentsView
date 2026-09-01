# Specify

Write a spec another agent can implement without this conversation.

## Questions

If the request is vague, one round of 2–3 questions and stop, unless there is an explicit chain.
Do not ask what is already in `docs/03`, `docs/02`, or `docs/01`.
State the most likely reading and ask for correction.

Questions that often change the outcome:

- What must be true when done (not "which files").
- What stays out.
- Constraints: owning module, do not expand product.
- Chain: spec only, + implement, + validate?

If the request **is** an undecided product definition: no implementation spec.
Inbox or `docs/05-ideas-to-discuss.md`, or ask a maintainer. See [docs.md](docs.md).

## File

1. Kebab-case slug. Path: `docs/sdd/specs/<slug>.md`.
2. Copy [docs/sdd/templates/spec.md](../../../../../docs/sdd/templates/spec.md).
3. Fill intent, out of scope, already decided (citations), acceptance, remaining questions.
4. `status: draft` if questions remain; `ready` if it can be implemented.
5. `next: implement` when `ready`; otherwise `specify`.
6. Update the table in [docs/sdd/README.md](../../../../../docs/sdd/README.md).

The spec must be self-contained: a cold agent reads that file and knows what to do.
Do not depend on chat. Acceptance in verifiable items, not "make it good".

## Close

- No chain: show the path, remaining questions, handoff block, **stop**.
- With `specify+implement`: mark `ready` (the user already chained; that counts as ok)
  and continue to [implement.md](implement.md).
- Do not code in this step unless the chain asks for it.

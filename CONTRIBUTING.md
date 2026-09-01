# Contributing to LiveAgentsView

Thanks for your interest in contributing. This repo follows a **docs-first** and
**spec-driven** (SDD) flow: definition lives in `docs/`, implementation specs in
`docs/sdd/`, and code follows those documents.

## Before you start

1. Read [README.md](README.md) and [AGENTS.md](AGENTS.md).
2. Check [docs/06-status.md](docs/06-status.md) to see what is decided and what is missing.
3. For bugs or features, open an issue before a large PR (optional for small changes).

## Local development

You only need Docker:

```bash
git clone <repo-url>
cd LiveAgentsView
cp .env.example .env
```

Runnable commands will live in [scripts/](scripts/README.md) once they are added.

## Change flow

### Small changes

Typos, targeted fixes, tests: direct PR, no spec.

### Meaningful changes

Features, refactors, new modules:

1. Open or update a spec in `docs/sdd/specs/` (template in `docs/sdd/templates/spec.md`).
2. Go through specify → implement → validate (`sdd` skill, `/sdd` invocation).
3. If the change touches product definition, **do not** close it in the spec: it goes to
   the inbox or `docs/05-ideas-to-discuss.md` until a maintainer decides.

### Product documentation

- Raw dump → `docs/00-inbox.md`
- Decisions → `docs/03-decisions.md` (with date)
- Unresolved questions → `docs/04-open-questions.md`
- Unagreed proposals → `docs/05-ideas-to-discuss.md`

## Pull requests

- One idea per PR when possible.
- Describe what changes and why.
- If there is a spec, link it (`docs/sdd/specs/<slug>.md`).
- A maintainer reviews before merge.

## Commits

Clear messages in English. Prefer the imperative:
"Add user model", "Fix compose healthcheck".

## Code of conduct

This project adopts the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating,
you agree to follow it.

## License

By contributing, you agree that your code is published under the [MIT license](LICENSE).

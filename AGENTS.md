# AGENTS.md

Instructions for any agent (or person) working in this repo.

## What this repo is

The single source of truth for **LiveAgentsView**, plus the platform code.

- `docs/` — all project documentation. See the table in [README.md](README.md).
- `docs/sdd/` — implementation specs for agents. They do not define product.
- `apps/` — deployables. Each app has its own README with its rules.
- `scripts/` — everything that must be run (placeholder for now).
- `.cursor/`, `.claude/`, `.agents/` — AI tool configuration (skills).
  Not business documentation.

## Implementation specs (`sdd`)

The `sdd` skill (`.agents/skills/sdd`, also in `.claude/skills` and `.cursor/skills`)
triages whether a change should have a spec, and runs specify → implement → validate.
Cursor and Claude: `/sdd`. Codex: `$sdd`. It activates on its own for meaningful
changes or when an open spec already exists.

It is not blocking: "quick", "small", "no spec", or "just go" means no spec.
If a spec is in play, keep it updated until `validated` or `abandoned`.
Steps can be chained ("specify and implement", "through validate").
Details in [docs/sdd/README.md](docs/sdd/README.md).

## Code rules

**Nothing is run by hand.** Everything goes through `scripts/` once they exist.
The only thing installed on the machine to develop and test LiveAgentsView itself is
Docker. New commands become scripts. This does not apply to the shipped binary, which
runs on the host by design — see the
[2026-09-01 Docker decision](docs/03-decisions.md).

**Writing code does not close a definition.** A human maintainer does that, and only
then does it land in the decision log. Something being built does not make it decided.

## Most important rule: do not mix decided with proposed

The documents `docs/01-vision.md`, `docs/02-scope.md`, and `docs/03-decisions.md`
contain **only what the team decided**.

`docs/06-status.md` is different: it decides nothing; it mirrors the others and the
code that exists. Update it when a fact changes, never to anticipate one.

- If you document something, write it exactly as whoever decided it said it. Do not
  expand it, improve it, or add scope on your own.
- Any proposal or feature you come up with goes to
  [docs/05-ideas-to-discuss.md](docs/05-ideas-to-discuss.md), marked as unagreed,
  or is raised in chat. Never directly in a definition document.
- This also applies on the technical side: do not expand product scope on your own.

If something is not in these documents, it is not decided — do not treat it as if it were.

## Documentation workflow

1. Everything lands raw in `docs/00-inbox.md`.
2. It gets distilled into the right document.
3. What gets decided is logged with a date in `docs/03-decisions.md`.
4. What stays unresolved is noted in `docs/04-open-questions.md`.
5. When a question is answered, remove it from open questions and check who cited it.

Full detail in [README.md](README.md).

## Open source

This is a public project. Changes go through pull request. Follow
[CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

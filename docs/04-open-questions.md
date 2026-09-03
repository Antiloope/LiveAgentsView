# Open questions

What still needs to be defined. Each question has a number that is not reused.

When answered, **remove it from here** and log the decision in
[03-decisions.md](03-decisions.md). Then search for citations to that number across
the rest of `docs/` so nothing is left outdated.

---

## Q-03 — Where is the remote published?

**Context:** The GitHub repo (or other host) and remote CI still need to be set up.
**Blocks:** badges, URLs in README, deploy.
**Status (2026-09-01):** deferred by Rodrigo — not needed until badges/CI/deploy become
relevant. Not blocking the first spec.

## Q-06 — How is "a human is watching this session" detected?

**Context:** [IDEA-02](05-ideas-to-discuss.md) proposes that attention depends on human
presence, not just agent state. Candidate signals: terminal focus, a recent
`UserPromptSubmit`, the active tmux pane. Each has different reliability and portability.
**Blocks:** the attention engine, IDEA-02.
**Status (2026-09-01):** deferred by Rodrigo — not needed until the attention engine /
IDEA-02 is built. Not part of the first spec.

## Q-10 — Which sprite archetypes belong to which race?

**Context:** [03-decisions.md](03-decisions.md) 2026-09-03 "Vocabulary: character, race,
class, quest" makes the sprite follow the character's race, since class is changeable and
a character must not turn into a different creature when its model changes. Today
`apps/web/src/sprites.ts` maps three archetypes to Claude Code's three models
(opus→Dragonkin Warrior, sonnet→Elf Rogue, haiku→Halfling) and hashes the session id for
everything else. What is undecided is the visual assignment itself: which archetypes read
as Claude Code, which as Cursor, which as Codex, and how many each race gets so two
characters of the same race still look like different individuals.
**Blocks:** only the visual assignment. The rule ("sprite follows race, individuals of a
race stay distinguishable") is decided and implementable without this — see
[character-model-redesign](sdd/specs/character-model-redesign.md), which proposes pools
the maintainer can override.
**Status (2026-09-03):** open, non-blocking.

## Q-11 — How does a fresh worktree get what git does not track?

**Context:** [03-decisions.md](03-decisions.md) 2026-09-03 "Territory: own worktree by
default" makes a LiveAgentsView-administered worktree the default territory. A new
worktree contains only tracked files: no `.env`, no `node_modules`, no build output. A
character recruited into one starts in a repo that may not run or build, which is exactly
the situation the default is supposed to avoid. Candidates: copy a declared list of files
from the source repo, run a per-repo setup command, symlink heavy directories, or leave it
to the character to sort out.
**Blocks:** how usable own territory actually is in practice, not whether it can be built.
**Status (2026-09-03):** open, deferred out of
[character-model-redesign](sdd/specs/character-model-redesign.md).

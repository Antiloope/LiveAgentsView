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


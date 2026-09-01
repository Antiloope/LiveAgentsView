# Triage

Run this when the user asks for a change and did not name a step.

## 1. Is there already a spec?

If a file in `docs/sdd/specs/` matches (title, slug, files touched):
load [implement.md](implement.md) or [validate.md](validate.md) based on `next`.
Do not offer "start from scratch?".

## 2. Is it interesting?

**Yes** (recommend a spec, do not code yet):

- New module, table, service, or dependency
- New flow or component
- Product behavior, permissions, onboarding
- More than one reasonable approach, or the request is vague
- Crosses several apps or layers, or several definition docs

**No** (do it, no spec):

- Typo, lint, targeted test, local rename, one line
- The user said quick / small / no spec / just go

On the edge, one sentence: what will be touched and "spec or just go?".
If they do not answer and the change is small, just go. If it is large, ask and stop.

## 3. Recommend

Not a sermon. Three lines:

1. Why this change benefits from a spec (one reason).
2. Two or three questions that change the outcome (not a questionnaire).
3. How to proceed: "I answer and we specify", "specify and implement",
   "through validate", or "just go, no spec".

Use the AskQuestion tool if available; otherwise ask and **stop**.
One round by default; a second only if the answer opens a material gap.
Maximum three questions per round.

## 4. Chain

| They say | Do |
|---|---|
| only the change, and it is interesting | questions → specify → stop (human) |
| "specify and implement" / "chain" | specify + implement in this session |
| "through validate" / "full flow" | all three steps |
| "just go" / "no spec" | implement, do not create a stub |

Chaining does not skip writing the spec: write it and continue.
The human can cut between steps.

## 5. After triage

- Specify → [specify.md](specify.md)
- No spec → implement following `AGENTS.md` rules (do not invent product)
- Raw docs / distill / decide / status → [docs.md](docs.md)

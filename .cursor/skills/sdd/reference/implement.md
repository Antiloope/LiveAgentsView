# Implement

Implement **that** spec. Do not reopen the intent or add scope.

## Before

1. Read the full spec. If `status` is `draft` or open questions remain without a chain:
   do not code; go back to [specify.md](specify.md) or ask the minimum that blocks.
2. If `status` is `abandoned` or `validated`: stop and say so.
3. Set `status` to `in-progress`, `updated` to today, table in `docs/sdd/README.md`.

## During

- `AGENTS.md`: scripts not loose commands once they exist, do not invent product.
- If the code forces a different approach: update the spec **in the same turn**
  (How, Acceptance, out of scope). The spec cannot lag behind the code.
- If a product decision appears: stop. Do not resolve it in the spec or code.

## After

1. Complete **How** with what was actually touched.
2. Acceptance: leave items for the validator; do not auto-check them all.
3. `status: done`, `next: validate`.
4. Update the table.
5. Handoff. Without a chain to validate: stop. With chain: [validate.md](validate.md).

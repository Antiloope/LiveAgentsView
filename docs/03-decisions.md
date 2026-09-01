# Decisions

Log of decisions made, with date and rationale. Only **agreed** items go here.

Format:

```
## YYYY-MM-DD — Short title

**Who:** …
**Decision:** …
**Rationale:** …
```

---

## 2026-09-01 — Open-source repo with docs-first structure

**Who:** Rodrigo
**Decision:** The project is open source under the MIT license. Definition documentation
lives in `docs/`; implementation specs in `docs/sdd/`. Local development uses Docker.
**Rationale:** Replicate the workflow proven in sincro, adapted to a public project
without coupling api+frontend from the start.

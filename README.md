# LiveAgentsView

Open-source project with docs-first and spec-driven development (SDD).

The source of truth for **what** lives in `docs/`. The **how** for each meaningful
change lives in `docs/sdd/`. Code follows those documents, not the other way around.

See [AGENTS.md](AGENTS.md) for working rules (humans and agents).

## License

[MIT](LICENSE)

## How documentation works

1. **Everything lands raw** in [docs/00-inbox.md](docs/00-inbox.md).
2. **It gets distilled** into the right document.
3. **What gets decided** is logged with a date in [docs/03-decisions.md](docs/03-decisions.md).
4. **What stays unresolved** goes to [docs/04-open-questions.md](docs/04-open-questions.md).
5. **Where things stand today** is in [docs/06-status.md](docs/06-status.md) — it decides nothing, it only reflects facts.

Rule: if something is not in the definition documents, it is not decided.

## Repository structure

```
.
├── docs/                project documentation
│   └── sdd/             implementation specs for agents (not product definition)
├── apps/                deployables (each with its own README)
├── scripts/             entry point for runnable commands (placeholder for now)
├── .agents/skills/      canonical agent skills
├── compose.yaml         shared services
├── compose.dev.yaml     development overrides
├── .env.example
├── README.md
├── AGENTS.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
└── LICENSE
```

## Documents

| Doc | Contents |
|---|---|
| [docs/00-inbox.md](docs/00-inbox.md) | Raw dump. Nothing here is final. |
| [docs/01-vision.md](docs/01-vision.md) | Vision, purpose, and audience. |
| [docs/02-scope.md](docs/02-scope.md) | Functional scope: what it does and what it does not. |
| [docs/03-decisions.md](docs/03-decisions.md) | Log of decisions made, with date and rationale. |
| [docs/04-open-questions.md](docs/04-open-questions.md) | What still needs to be defined. |
| [docs/05-ideas-to-discuss.md](docs/05-ideas-to-discuss.md) | **Unagreed** proposals. |
| [docs/06-status.md](docs/06-status.md) | Snapshot of where things stand. Decides nothing. |
| [docs/sdd/](docs/sdd/README.md) | Implementation specs for agents. **Does not define product.** |

## Development

**The only thing you need installed is Docker.** No Go, Node, or psql on the host.

Runnable commands will live in [scripts/](scripts/README.md). That directory is a placeholder for now.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). To report bugs or request features, use the
[issue templates](.github/ISSUE_TEMPLATE/).

## Status

Started: 2026-09-01. The full snapshot lives in [docs/06-status.md](docs/06-status.md).

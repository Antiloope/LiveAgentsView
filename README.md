# LiveAgentsView

Proyecto open source con documentación primero y desarrollo spec-driven (SDD).

La fuente de verdad del **qué** vive en `docs/`. El **cómo** de cada cambio
interesante vive en `docs/sdd/`. El código sigue esos documentos, no al revés.

Ver [AGENTS.md](AGENTS.md) para reglas de trabajo (humanos y agentes).

## Licencia

[MIT](LICENSE)

## Cómo funciona la documentación

1. **Todo entra crudo** por [docs/00-inbox.md](docs/00-inbox.md).
2. **Se destila** hacia el documento que corresponda.
3. **Lo que se decide** se registra con fecha en [docs/03-decisions.md](docs/03-decisions.md).
4. **Lo que queda sin resolver** va a [docs/04-open-questions.md](docs/04-open-questions.md).
5. **Dónde está todo hoy** se lee en [docs/06-status.md](docs/06-status.md) — no decide nada, solo refleja hechos.

Regla: si algo no está en los documentos de definición, no está decidido.

## Estructura del repo

```
.
├── docs/                documentación del proyecto
│   ├── sdd/             specs de implementación para agentes (no es definición)
│   └── arquitectura/    el C4: de acá sale la estructura del código
├── apps/                deployables (cada uno con su README)
├── db/                  esquema: migraciones SQL y su runner
├── scripts/             todo lo que hay que correr
├── .github/workflows/   CI
├── .agents/skills/      skills canónicas para agentes
├── compose.yaml         servicios compartidos
├── compose.dev.yaml     overrides de desarrollo
├── .env.example
├── README.md
├── AGENTS.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
└── LICENSE
```

## Documentos

| Doc | Qué contiene |
|---|---|
| [docs/00-inbox.md](docs/00-inbox.md) | Volcado crudo. Nada de esto es definitivo. |
| [docs/01-vision.md](docs/01-vision.md) | Visión, propósito y público del proyecto. |
| [docs/02-scope.md](docs/02-scope.md) | Alcance funcional: qué hace y qué no. |
| [docs/03-decisions.md](docs/03-decisions.md) | Log de decisiones tomadas, con fecha y motivo. |
| [docs/04-open-questions.md](docs/04-open-questions.md) | Lo que falta definir. |
| [docs/05-ideas-to-discuss.md](docs/05-ideas-to-discuss.md) | Propuestas **no acordadas**. |
| [docs/06-status.md](docs/06-status.md) | Foto de dónde está todo. No decide nada. |
| [docs/sdd/](docs/sdd/README.md) | Specs de implementación para agentes. **No define producto.** |

## Desarrollo

**Lo único que hace falta instalado es Docker.** Ni Go, ni Node, ni psql en el host.

```bash
./scripts/up.sh
```

El resto de los comandos está en [scripts/README.md](scripts/README.md).

## Contribuir

Leé [CONTRIBUTING.md](CONTRIBUTING.md). Para reportar bugs o pedir features, usá los
[issue templates](.github/ISSUE_TEMPLATE/).

## Estado

Arranque: 2026-09-01. La foto completa vive en [docs/06-status.md](docs/06-status.md).

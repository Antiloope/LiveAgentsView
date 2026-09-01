# Contributing to LiveAgentsView

Gracias por interesarte en contribuir. Este repo sigue un flujo **docs-first** y
**spec-driven** (SDD): la definición vive en `docs/`, las specs de implementación en
`docs/sdd/`, y el código sigue esos documentos.

## Antes de empezar

1. Leé [README.md](README.md) y [AGENTS.md](AGENTS.md).
2. Revisá [docs/06-status.md](docs/06-status.md) para saber qué está decidido y qué falta.
3. Para bugs o features, abrí un issue antes de un PR grande (opcional para cambios chicos).

## Desarrollo local

Solo necesitás Docker:

```bash
git clone <url-del-repo>
cd LiveAgentsView
cp .env.example .env
./scripts/up.sh
```

Más comandos en [scripts/README.md](scripts/README.md).

## Flujo de cambios

### Cambios chicos

Typos, fixes puntuales, tests: PR directo, sin spec.

### Cambios interesantes

Features, refactors, módulos nuevos, cambios de arquitectura:

1. Abrí o actualizá una spec en `docs/sdd/specs/` (plantilla en `docs/sdd/templates/spec.md`).
2. Pasá por specify → implement → validate (skill `sdd`, invocación `/sdd`).
3. Si el cambio toca definición de producto, **no** lo cierres en la spec: va al inbox o a
   `docs/05-ideas-to-discuss.md` hasta que un maintainer lo decida.

### Documentación de producto

- Volcado crudo → `docs/00-inbox.md`
- Decisiones → `docs/03-decisions.md` (con fecha)
- Preguntas sin resolver → `docs/04-open-questions.md`
- Propuestas no acordadas → `docs/05-ideas-to-discuss.md`

## Pull requests

- Una idea por PR cuando sea posible.
- Describí qué cambia y por qué.
- Si hay spec, linkeala (`docs/sdd/specs/<slug>.md`).
- El CI tiene que pasar.
- Un maintainer revisa antes de merge.

## Commits

Mensajes claros en español o inglés (consistente dentro del PR). Preferí el imperativo:
"Agrega migración para usuarios", "Corrige runner de migraciones".

## Código de conducta

Este proyecto adopta el [Contributor Covenant](CODE_OF_CONDUCT.md). Al participar,
aceptás cumplirlo.

## Licencia

Al contribuir, aceptás que tu código se publique bajo la [licencia MIT](LICENSE).

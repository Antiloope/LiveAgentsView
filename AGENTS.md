# AGENTS.md

Instrucciones para cualquier agente (o persona) que trabaje en este repo.

## Qué es este repo

La fuente única de verdad del proyecto **LiveAgentsView**, más el código de la plataforma.

- `docs/` — toda la documentación del proyecto. Ver la tabla en [README.md](README.md).
- `docs/sdd/` — specs de implementación para agentes. No definen producto.
- `docs/arquitectura/workspace.dsl` — el **C4**. Es el documento del que sale la
  estructura del código.
- `apps/` — deployables. Cada app tiene su README con sus reglas.
- `db/` — el esquema. `scripts/` — todo lo que hay que correr. `.github/` — CI.
- `.cursor/`, `.claude/`, `.agents/` — configuración de herramientas de IA (skills).
  No son documentación del negocio.

## Specs de implementación (`sdd`)

La skill `sdd` (`.agents/skills/sdd`, también en `.claude/skills` y `.cursor/skills`)
triage si un cambio conviene spec, y corre specify → implement → validate.
Cursor y Claude: `/sdd`. Codex: `$sdd`. Se activa sola en cambios interesantes
o si ya hay una spec abierta.

No es bloqueante: “rápido”, “chico”, “sin spec” o “andá de una” se hace sin spec.
Si una spec está en juego, se actualiza hasta `validada` o `abandonada`.
Los pasos se pueden encadenar (“especificá e implementá”, “hasta validar”).
Detalle en [docs/sdd/README.md](docs/sdd/README.md).

## Reglas del código

**Nada se corre a mano.** Todo pasa por `scripts/`. Lo único instalado en la máquina
es Docker. Si algo necesita un comando nuevo, va como script.

**El C4 va primero.** Si aparece un módulo, servicio o dependencia nueva, se agrega a
`workspace.dsl` antes de escribir el código. El diagrama define; el código sigue.

**Las migraciones no se editan una vez aplicadas.** El runner guarda el checksum y
aborta si cambian. Para corregir, va una migración nueva.

**Escribir código no cierra una definición.** Eso lo hace un maintainer humano, y
recién ahí baja al log de decisiones. Que algo esté construido no lo vuelve decidido.

## Regla más importante: no mezclar lo decidido con lo propuesto

Los documentos `docs/01-vision.md`, `docs/02-scope.md` y `docs/03-decisions.md`
contienen **solo lo que el equipo decidió**.

`docs/06-status.md` es distinto: no decide nada, es el espejo de los otros y del
código que existe. Se actualiza cuando cambia un hecho, nunca para anticipar uno.

- Si vas a documentar algo, escribilo tal como lo dijo quien lo decidió. No lo
  amplíes, no lo mejores, no le agregues alcance de tu propia iniciativa.
- Cualquier propuesta o feature que se te ocurra va a
  [docs/05-ideas-to-discuss.md](docs/05-ideas-to-discuss.md), marcada como no acordada,
  o se plantea en el chat. Nunca directamente en un documento de definición.
- Esto vale también para el lado técnico: no ampliar el alcance del producto
  por iniciativa propia.

Si algo no está en estos documentos, no está decidido — no lo trates como si lo estuviera.

## Flujo de trabajo de la documentación

1. Todo entra crudo por `docs/00-inbox.md`.
2. Se destila hacia el documento que corresponda.
3. Lo que se decide se registra con fecha en `docs/03-decisions.md`.
4. Lo que queda sin resolver se anota en `docs/04-open-questions.md`.
5. Cuando una pregunta se responde, se saca de las abiertas y se revisa quién la citaba.

Detalle completo en [README.md](README.md).

## Open source

Este es un proyecto público. Los cambios van por pull request. Respetá
[CONTRIBUTING.md](CONTRIBUTING.md) y [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

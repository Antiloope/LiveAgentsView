# Specs de implementación

Harness para agentes (Cursor, Claude Code, Codex). **No define producto.**
La definición vive en `docs/01-vision.md`, `docs/02-scope.md` y `docs/03-decisions.md`.
Una spec de acá no cierra una decisión: la cita o, si falta, para y pregunta.

Skill: `sdd` (`.agents/skills/sdd`). Invocación: `/sdd` en Cursor y Claude, `$sdd` en Codex.
También se activa sola cuando el cambio es interesante o ya hay una spec en juego.

No es bloqueante. Tareas chicas o “andá de una” siguen sin spec.
Si una spec **está en uso**, se actualiza hasta el final (o se marca `abandonada`).

## Flujo

`specify` → (humano) → `implement` → (humano) → `validate`

Se pueden encadenar: “especificá e implementá”, “hasta validar”.
Cada paso deja un bloque de handoff para pegárselo a otro agente.

## Specs

| Spec | Estado | Siguiente |
|---|---|---|
| _(ninguna todavía)_ | — | — |

Estados: `borrador` · `lista` · `en-curso` · `hecha` · `validada` · `abandonada`.

Plantilla: [templates/spec.md](templates/spec.md). Archivos en [specs/](specs/).

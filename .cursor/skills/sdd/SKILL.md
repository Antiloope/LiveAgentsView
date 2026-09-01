---
name: sdd
description: >
  Spec-driven workflow for this repo (SDD). Triages whether a change
  needs a spec, writes and keeps specs current in docs/sdd/specs/, and runs
  specify → implement → validate with optional chaining so small or rápido
  work is not blocked. Also destills the docs inbox, decisions, open
  questions, and status. Use when the user proposes a feature, refactor,
  new module, C4 change, or interesting repo change; mentions spec, SDD,
  inbox, destilar, decisiones, preguntas abiertas, or docs/00–06; an
  open spec exists for the work; or they ask to specify, implement, or
  validate. Skip recommending a spec for tiny/chico/rápido work unless a
  spec is already in play — then it must stay updated through the end.
---

# SDD

Specs de implementación en `docs/sdd/`. No son definición de producto.
Antes de actuar: listá `docs/sdd/specs/*.md` (salvo el README) y leé
[docs/sdd/README.md](../../../docs/sdd/README.md) si hay alguna abierta
que matchee el pedido. Si hay spec en juego, cargala y seguí desde `siguiente`.

## Routing

| Pedido | Qué cargar |
|---|---|
| Feature, refactor, módulo, “cambio interesante”, sin comando | [reference/triage.md](reference/triage.md) |
| `specify` / “armemos la spec” / “definamos esto” | [reference/specify.md](reference/specify.md) |
| `implement` / “desarrollá la spec” / “implementá X.md” | [reference/implement.md](reference/implement.md) |
| `validate` / “validá la spec” / “revisá contra la spec” | [reference/validate.md](reference/validate.md) |
| `inbox` / `destilar` / `decidir` / `estado` | [reference/docs.md](reference/docs.md) |
| `/sdd` o `$sdd` sin argumentos | Menú corto de la tabla; no arrancar un paso |

Cadena dicha por el usuario (`especificá e implementá`, `hasta validar`, `encadená`):
hacer esos pasos en esta sesión, actualizando la spec entre medio.
Sin cadena, un paso y handoff. El humano está en el medio a propósito.

## No bloquea, no deja a medias

- “rápido”, “chico”, “sin spec”, “andá de una”: implementar sin spec.
  Una frase ofreciendo spec alcanza; si dicen que no, no insistir.
- Spec **ya abierta** para este trabajo: no se saltea. Se actualiza en el
  mismo turno hasta `validada` o `abandonada` (con motivo).
- Borrador abandonado a mitad: marcar `abandonada` o terminarlo. Nunca
  dejar `borrador`/`en-curso` huérfano.

## Definición vs spec

Escribir código o una spec **no cierra una definición**. Si el pedido
inventa producto, para: `docs/05-ideas-to-discuss.md` o preguntarle a un maintainer.
Citas a lo decidido; no ampliar alcance por iniciativa propia.

C4 primero si aparece un módulo, servicio o dependencia nueva.

## Handoff

Al cerrar un paso (o la cadena), pegá esto:

```
Spec: docs/sdd/specs/<slug>.md
Estado: <estado>
Siguiente: <specify|implement|validate|ninguna>
```

Estados: `borrador` · `lista` · `en-curso` · `hecha` · `validada` · `abandonada`.
Plantilla: [docs/sdd/templates/spec.md](../../../docs/sdd/templates/spec.md).
Índice: tabla en `docs/sdd/README.md` — actualizarla al crear, cambiar estado o cerrar.

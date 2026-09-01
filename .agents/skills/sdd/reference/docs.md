# Docs de definición

Las specs de `docs/sdd/specs/` no reemplazan este flujo. Esto es la fuente de verdad.

## inbox

Volcar crudo en `docs/00-inbox.md`: un bloque con fecha, sin ordenar, sin destilar.
Nada de esto es definición.

## destilar

Del inbox al documento que corresponda (`01-vision`, `02-scope`, etc.).

- Tal cual lo dijo quien lo decidió. No ampliar, no “mejorar”.
- Propuesta o idea del agente → `docs/05-ideas-to-discuss.md`, marcada como no acordada.
- Decisión con fecha → también `docs/03-decisions.md`.
- Sin resolver → `docs/04-open-questions.md` (números no se reciclan).
- Marcar el bloque del inbox como destilado.

## decidir

Solo cuando un maintainer lo dijo. Entrada en `03-decisions` con fecha, por qué, quién.
Si cerraba una pregunta de `04`, **sacarla** de abiertas y buscar citas a ese número
en el resto de `docs/` para que no queden mintiendo.

## estado

`docs/06-status.md` es espejo. Actualizar cuando cambió un hecho (código o docs).
Nunca para anticipar. No decide nada.

## Relación con las specs

Una spec de implementación cita lo decidido. Si destilar/decidir cambia algo que una
spec abierta citaba, actualizar esa spec en el mismo turno o marcarla `abandonada`.

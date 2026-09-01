# Validate

Chequear el código contra la spec. No reimplementar ni “mejorar” alcance.

## Antes

Spec en `hecha` o `en-curso` (si encadenaron). Si está `borrador`, no hay nada que validar.

## Qué mirar

1. Cada ítem de **Aceptación**: sí / no / no aplica, con evidencia (archivo, test, comportamiento).
2. **Fuera de alcance:** ¿se coló algo?
3. ¿La spec describe lo que hay? Si el código divergió y la spec no se actualizó, eso es fallo:
   actualizar la spec o marcar el hueco; no dejar las dos versiones.
4. Reglas de `AGENTS.md` que la spec tocaba (C4, migraciones, definición vs código).

No un review genérico de estilo. Eso no es este paso.

## Después

En **Validación** de la spec: resultado por ítem y huecos.

- Todo cubierto: `estado: validada`, `siguiente: ninguna`.
- Huecos chicos: listarlos; el usuario elige si se arreglan ahora o otra spec.
  No dejar `en-curso` a medias: o se cierran en este turno y queda `validada`,
  o `hecha` con los huecos escritos y `siguiente: implement`.
- Spec obsoleta a propósito: `abandonada` y por qué.

Actualizar la tabla en `docs/sdd/README.md`. Handoff. Parar.

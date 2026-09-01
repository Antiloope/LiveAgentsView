# Implement

Implementar **esa** spec. No reabrir el intento ni sumar alcance.

## Antes

1. Leer la spec completa. Si `estado` es `borrador` o hay preguntas abiertas sin cadena:
   no codear; volver a [specify.md](specify.md) o preguntar lo mínimo que bloquea.
2. Si `estado` es `abandonada` o `validada`: parar y decirlo.
3. Pasar `estado` a `en-curso`, `actualizada` a hoy, tabla en `docs/sdd/README.md`.

## Durante

- `AGENTS.md`: C4 primero, scripts no comandos sueltos, migraciones nuevas no editadas,
  no inventar producto.
- Si el código obliga a cambiar el enfoque: actualizar la spec **en el mismo turno**
  (Cómo, Aceptación, fuera de alcance). La spec no puede quedar atrás del código.
- Si aparece una decisión de producto: parar. No resolverla en la spec ni en el código.

## Después

1. Completar **Cómo** con lo que realmente se tocó.
2. Aceptación: dejar los ítems para el validador; no auto-marcarlos todos.
3. `estado: hecha`, `siguiente: validate`.
4. Actualizar la tabla.
5. Handoff. Sin cadena a validate: parar. Con cadena: [validate.md](validate.md).

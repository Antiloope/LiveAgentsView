# apps/

Deployables del proyecto. Cada carpeta es una aplicación o servicio que se construye,
corre y despliega por separado.

No hay `packages/` compartidos: lo común vive en `db/`, `scripts/`, `docs/` y
`compose.yaml`.

## Cómo agregar una app

1. Actualizar el C4 en `docs/arquitectura/workspace.dsl`.
2. Crear `apps/<nombre>/` con su README (reglas de estructura interna).
3. Agregar el servicio a `compose.yaml` y `compose.dev.yaml` si corre en local.
4. Extender `.github/detectar-cambios.sh` y `scripts/test.sh` para la nueva app.
5. Si el cambio es interesante, abrir una spec en `docs/sdd/specs/`.

## Estado

Vacío al 2026-09-01. La primera app depende de definir visión y alcance.

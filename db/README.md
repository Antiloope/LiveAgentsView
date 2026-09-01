# Base de datos

El esquema se cambia **desde acá y desde CI**, nunca desde la aplicación en runtime.
Las apps arrancan asumiendo que el esquema ya está aplicado.

## Cómo agregar una migración

1. Crear `migrations/NNNN_<modulo>.sql` con el número siguiente.
2. SQL puro, sin marcas de ninguna herramienta.
3. Correrla en local: `./scripts/migrate.sh`
4. Commit. Al mergear a `main`, el workflow la aplica en CI.

**Una migración aplicada no se edita más.** El runner guarda el checksum de cada
archivo y aborta si cambia. Para corregir, va una migración nueva.

## Nombre de los archivos

`NNNN_<modulo>.sql`. La numeración es global; el orden de aplicación es numérico.

## Comandos

```bash
./scripts/migrate.sh            # aplica lo pendiente en la base local
./scripts/migrate.sh --status   # qué está aplicado y qué falta
./scripts/migrate.sh --dry-run  # qué correría, sin correrlo
```

Contra otra base:

```bash
DATABASE_URL='postgres://…' ./scripts/migrate.sh --status
```

## Control

El runner guarda todo en `schema_migrations` (ver `run.sh`).

## Estado

Runner listo. Sin migraciones todavía.

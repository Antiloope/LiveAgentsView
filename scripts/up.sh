#!/usr/bin/env bash
#
# El único comando para ponerse a trabajar.
#
# Levanta la base de datos, aplica migraciones pendientes y deja los servicios
# de infraestructura listos. Cuando existan apps en compose.dev.yaml, también
# las levantará en modo watch.

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker
asegurar_env
cargar_env

titulo "Base de datos"
compose_dev up -d db
esperar_sano db
msg "lista"

titulo "Migraciones"
if [ -d "$RAIZ/db/migrations" ] && [ "$(find "$RAIZ/db/migrations" -maxdepth 1 -name '*.sql' | wc -l)" -gt 0 ]; then
    "$RAIZ/scripts/migrate.sh"
else
    detalle "Sin migraciones todavía — se saltea."
fi

titulo "Servicios de desarrollo"
detalle "Postgres  localhost:${POSTGRES_PORT:-5432}"
detalle ""
detalle "Cuando agregues apps, extendé compose.dev.yaml y este script."
detalle "Ctrl+C para cortar si corrés servicios en primer plano."

compose_dev up "$@"

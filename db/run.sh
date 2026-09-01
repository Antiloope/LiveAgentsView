#!/bin/sh
# Runner de migraciones.
#
# Corre adentro de un contenedor con psql, así que no hace falta tener nada
# instalado. Lo invocan ./scripts/migrate.sh en local y el workflow de GitHub
# en stage: el mismo código en los dos lados, para que lo que se probó acá sea
# literalmente lo que corre allá.
#
# Uso:  DATABASE_URL=… ./run.sh [--status|--dry-run]

set -eu

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/migrations}"
APPLIED_BY="${APPLIED_BY:-local}"
MODE="apply"

case "${1:-}" in
    --status)  MODE="status" ;;
    --dry-run) MODE="dry-run" ;;
    "")        ;;
    *)         echo "opción desconocida: $1" >&2; exit 2 ;;
esac

if [ -z "${DATABASE_URL:-}" ]; then
    echo "falta DATABASE_URL" >&2
    exit 1
fi

# ON_ERROR_STOP hace que psql corte en el primer error en vez de seguir con las
# sentencias siguientes, que es el comportamiento por omisión y es una trampa.
#
# client_min_messages=warning saca los NOTICE de "esto ya existía": son ruido y
# esconden lo que sí importa cuando algo falla.
export PGOPTIONS="${PGOPTIONS:-} -c client_min_messages=warning"

psql() { command psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q "$@"; }
consulta() { command psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -tAq -c "$1"; }

# ---------------------------------------------------------------------------

psql -c "
create table if not exists schema_migrations (
    version    text primary key,
    checksum   text        not null,
    applied_at timestamptz not null default now(),
    applied_by text        not null default 'desconocido'
);" >/dev/null

# `create table if not exists` no dice nada si la tabla existe con otra forma
# —por ejemplo, la que deja otra herramienta de migraciones—. Sin este chequeo,
# el error aparece más adelante y como un mensaje críptico de Postgres.
if [ "$(consulta "select count(*) from information_schema.columns
                  where table_name = 'schema_migrations'
                    and column_name in ('version', 'checksum', 'applied_at', 'applied_by');")" != "4" ]; then
    echo "  ✗ Ya existe una tabla schema_migrations con otra forma." >&2
    echo "" >&2
    echo "    Suele ser de otra herramienta de migraciones usada antes." >&2
    echo "    En una base de desarrollo: ./scripts/down.sh --borrar y volver a empezar." >&2
    echo "    En una base con datos: renombrarla y decidir a mano qué migraciones" >&2
    echo "    dar por aplicadas." >&2
    exit 1
fi

# El número es arbitrario pero fijo: identifica a este runner. Dos corridas en
# paralelo —dos merges seguidos a main— se serializan acá en vez de pisarse.
if [ "$MODE" = "apply" ]; then
    if [ "$(consulta 'select pg_try_advisory_lock(8274461);')" != "t" ]; then
        echo "hay otra migración corriendo contra esta base; no se hace nada" >&2
        exit 1
    fi
fi

archivos=$(find "$MIGRATIONS_DIR" -maxdepth 1 -name '*.sql' | sort)

if [ -z "$archivos" ]; then
    echo "no hay migraciones en $MIGRATIONS_DIR"
    exit 0
fi

pendientes=0
aplicadas=0

for ruta in $archivos; do
    version=$(basename "$ruta")
    checksum=$(sha256sum "$ruta" | cut -d' ' -f1)
    registrado=$(consulta "select checksum from schema_migrations where version = '$version';")

    if [ -n "$registrado" ]; then
        if [ "$registrado" != "$checksum" ]; then
            echo "  ✗ $version — YA APLICADA PERO EL ARCHIVO CAMBIÓ" >&2
            echo "" >&2
            echo "    En la base corrió una versión distinta de la que está en el repo." >&2
            echo "    Nadie puede saber cómo quedó el esquema de verdad. Para corregir," >&2
            echo "    dejá este archivo como estaba y agregá una migración nueva." >&2
            exit 1
        fi
        [ "$MODE" = "status" ] && echo "  ✓ $version"
        aplicadas=$((aplicadas + 1))
        continue
    fi

    pendientes=$((pendientes + 1))

    if [ "$MODE" != "apply" ]; then
        echo "  · $version — pendiente"
        continue
    fi

    echo "  → $version"

    # La migración y su registro van en la misma transacción: si el SQL falla,
    # no queda registrada; si se registra, es porque el SQL entró.
    tmp=$(mktemp)
    cat "$ruta" > "$tmp"
    printf "\ninsert into schema_migrations (version, checksum, applied_by) values ('%s', '%s', '%s');\n" \
        "$version" "$checksum" "$APPLIED_BY" >> "$tmp"

    if ! psql --single-transaction -f "$tmp"; then
        rm -f "$tmp"
        echo "  ✗ $version falló — la base quedó como estaba" >&2
        exit 1
    fi
    rm -f "$tmp"
done

echo ""
case "$MODE" in
    status)  echo "$aplicadas aplicadas, $pendientes pendientes" ;;
    dry-run) echo "$pendientes pendientes (no se corrió nada)" ;;
    apply)   if [ "$pendientes" -eq 0 ]; then
                 echo "la base ya estaba al día ($aplicadas migraciones)"
             else
                 echo "$pendientes migraciones aplicadas"
             fi ;;
esac

#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker
asegurar_env
cargar_env

args=()
for arg in "$@"; do
    args+=("$arg")
done

if [ ${#args[@]} -eq 0 ]; then
    args=(apply)
fi

case "${args[0]}" in
    apply|--status|--dry-run) ;;
    *)
        morir "Uso: $0 [--status|--dry-run]"
        ;;
esac

DATABASE_URL="${DATABASE_URL:-postgres://${POSTGRES_USER:-liveagents}:${POSTGRES_PASSWORD:-liveagents}@localhost:${POSTGRES_PORT:-5432}/${POSTGRES_DB:-liveagents}?sslmode=disable}"

titulo "Migraciones"
docker run --rm \
    --network host \
    -e DATABASE_URL="$DATABASE_URL" \
    -e MIGRATIONS_DIR=/db/migrations \
    -e APPLIED_BY=local \
    -v "$RAIZ/db:/db:ro" \
    postgres:17-alpine /bin/sh /db/run.sh "${args[@]}"

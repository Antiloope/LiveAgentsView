#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker
cargar_env

borrar=false
for arg in "$@"; do
    case "$arg" in
        --borrar) borrar=true ;;
        -h|--help)
            echo "Uso: $0 [--borrar]"
            echo "  --borrar  elimina volúmenes (base de datos)"
            exit 0
            ;;
    esac
done

titulo "Apagando servicios"
if [ "$borrar" = true ]; then
    compose_dev down -v
    msg "servicios y volúmenes eliminados"
else
    compose_dev down
    msg "servicios apagados (datos intactos)"
fi

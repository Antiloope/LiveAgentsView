#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker
cargar_env

servicio="${1:-}"

if [ -n "$servicio" ]; then
    compose_dev logs -f "$servicio"
else
    compose_dev logs -f
fi

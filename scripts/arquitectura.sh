#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker

titulo "Arquitectura (C4)"
detalle "Abrí http://localhost:${C4_PORT:-8081} en el navegador."
detalle "Ctrl+C para cortar."

docker run --rm -it \
    -p "${C4_PORT:-8081}:8080" \
    -v "$RAIZ/docs/arquitectura:/usr/local/structurizr" \
    structurizr/structurizr-lite

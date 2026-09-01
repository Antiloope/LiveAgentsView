#!/usr/bin/env bash
#
# Lo mismo que corre el CI. Sin argumentos: chequeos generales del repo.
# Con nombre de app: tests de esa app (cuando exista).

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

requiere_docker

app="${1:-all}"

titulo "Chequeos del repo"

# Scripts ejecutables
while IFS= read -r script; do
    [ -x "$script" ] || morir "Script sin permiso de ejecución: $script"
done < <(find "$RAIZ/scripts" -maxdepth 1 -name '*.sh' ! -name 'lib.sh')

msg "scripts OK"

# C4 válido
if [ -f "$RAIZ/docs/arquitectura/workspace.dsl" ]; then
    docker run --rm -v "$RAIZ/docs/arquitectura:/work" \
        structurizr/structurizr validate -workspace /work/workspace.dsl >/dev/null
    msg "C4 OK"
fi

case "$app" in
    all)
        detalle "Sin apps todavía — agregá targets acá cuando existan en apps/"
        ;;
    *)
        if [ ! -d "$RAIZ/apps/$app" ]; then
            morir "No existe apps/$app"
        fi
        detalle "Agregá el runner de tests para apps/$app"
        ;;
esac

msg "listo"

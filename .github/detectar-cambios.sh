#!/usr/bin/env bash
#
# Dice qué partes del repo cambiaron, para que el CI corra solo lo que corresponde.
#
# Uso:  .github/detectar-cambios.sh <sha-base>
#
# Escribe en $GITHUB_OUTPUT una variable por parte, con valor "si" o "no".

set -euo pipefail

base="${1:-}"
salida="${GITHUB_OUTPUT:-/dev/stdout}"

todo=false
if [ -z "$base" ] || [ "$base" = "0000000000000000000000000000000000000000" ]; then
    todo=true
elif ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
    todo=true
fi

if [ "$todo" = true ]; then
    echo "sin base comparable contra '$base': se corre todo" >&2
    archivos=""
else
    archivos="$(git diff --name-only "$base" HEAD)"
    echo "archivos cambiados desde $base:" >&2
    echo "$archivos" | sed 's/^/  /' >&2
fi

marcar() {
    local nombre="$1" patron="$2" valor="no"
    if [ "$todo" = true ]; then
        valor="si"
    elif echo "$archivos" | grep -qE "$patron"; then
        valor="si"
    fi
    echo "$nombre=$valor" >> "$salida"
    echo "→ $nombre=$valor" >&2
}

# Apps: cada subcarpeta de apps/ con nombre propio (no README.md)
for app_dir in apps/*/; do
    [ -d "$app_dir" ] || continue
    app_name="$(basename "$app_dir")"
    marcar "$app_name" "^apps/${app_name}/"
done

marcar migraciones   '^db/'
marcar arquitectura  '^docs/arquitectura/'
marcar repo          '^(scripts/|docs/sdd/|\.github/|compose\.ya?ml|compose\.dev\.ya?ml)$'

# Cambios en compose fuerzan rebuild de todas las apps definidas
if [ "$todo" = false ] && echo "$archivos" | grep -qE '^(compose\.ya?ml|compose\.dev\.ya?ml)$'; then
    for app_dir in apps/*/; do
        [ -d "$app_dir" ] || continue
        app_name="$(basename "$app_dir")"
        echo "${app_name}=si" >> "$salida"
        echo "→ cambió compose: se fuerza ${app_name}" >&2
    done
fi

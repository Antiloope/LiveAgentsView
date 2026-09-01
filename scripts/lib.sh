# shellcheck shell=bash
#
# Lo común a todos los scripts. No se ejecuta solo: se incluye con `source`.

set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$RAIZ"

# ── Salida ──────────────────────────────────────────────────────────────────

if [ -t 1 ]; then
    _AZUL=$'\033[34m'; _AMARILLO=$'\033[33m'; _ROJO=$'\033[31m'
    _GRIS=$'\033[90m'; _FUERTE=$'\033[1m'; _FIN=$'\033[0m'
else
    _AZUL=""; _AMARILLO=""; _ROJO=""; _GRIS=""; _FUERTE=""; _FIN=""
fi

titulo()  { printf "\n%s▸ %s%s\n" "$_AZUL$_FUERTE" "$*" "$_FIN"; }
msg()     { printf "  %s\n" "$*"; }
detalle() { printf "  %s%s%s\n" "$_GRIS" "$*" "$_FIN"; }
aviso()   { printf "  %s! %s%s\n" "$_AMARILLO" "$*" "$_FIN"; }
morir()   { printf "\n  %s✗ %s%s\n\n" "$_ROJO" "$*" "$_FIN" >&2; exit 1; }

# ── Requisitos ──────────────────────────────────────────────────────────────

requiere_docker() {
    command -v docker >/dev/null 2>&1 \
        || morir "Docker no está instalado. Es lo único que hace falta para trabajar en este repo."

    docker info >/dev/null 2>&1 \
        || morir "Docker está instalado pero no está corriendo. Abrí Docker Desktop y volvé a intentar."

    docker compose version >/dev/null 2>&1 \
        || morir "Falta Docker Compose v2 (viene con Docker Desktop desde hace años)."
}

asegurar_env() {
    if [ ! -f "$RAIZ/.env" ]; then
        cp "$RAIZ/.env.example" "$RAIZ/.env"
        aviso "No había .env — se creó uno a partir de .env.example."
        detalle "Los valores por omisión sirven para desarrollo local."
    fi
}

# Cargar variables del .env para los scripts (no exportar secretos innecesarios).
cargar_env() {
    if [ -f "$RAIZ/.env" ]; then
        set -a
        # shellcheck disable=SC1091
        source "$RAIZ/.env"
        set +a
    fi
}

# ── Compose ─────────────────────────────────────────────────────────────────

compose_dev()  { docker compose -f compose.yaml -f compose.dev.yaml "$@"; }
compose_prod() { docker compose -f compose.yaml "$@"; }

esperar_sano() {
    local servicio="$1" intentos="${2:-60}"
    local id
    for _ in $(seq 1 "$intentos"); do
        id="$(compose_dev ps -q "$servicio" 2>/dev/null || true)"
        if [ -n "$id" ] && [ "$(docker inspect -f '{{.State.Health.Status}}' "$id" 2>/dev/null || echo "")" = "healthy" ]; then
            return 0
        fi
        sleep 1
    done
    morir "El servicio '$servicio' no llegó a estar sano."
}

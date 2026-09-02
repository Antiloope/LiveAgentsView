#!/usr/bin/env bash
# Build and run the LiveAgentsView daemon locally via Docker.
# Dashboard: http://localhost:${LAV_PORT:-8420}
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose -f compose.yaml -f compose.dev.yaml up --build "$@"

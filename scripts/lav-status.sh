#!/usr/bin/env bash
# Quick CLI view of known sessions, without opening the dashboard.
# Requires the daemon to already be running (scripts/dev-up.sh).
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose -f compose.yaml -f compose.dev.yaml exec lav lav status "$@"

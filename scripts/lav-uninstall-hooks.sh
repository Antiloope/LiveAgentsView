#!/usr/bin/env bash
# Removes hooks a previous version's "lav init" wrote into Claude Code,
# Codex and Cursor's own config, restoring each file to what it looked like
# before — with a backup taken first. Pass --dry-run to preview without
# writing anything, or --yes to skip the interactive confirmation.
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose -f compose.yaml -f compose.dev.yaml run --rm lav uninstall-hooks "$@"

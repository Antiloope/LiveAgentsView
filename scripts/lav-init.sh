#!/usr/bin/env bash
# Wires LiveAgentsView's hooks into Claude Code / Codex / Cursor's existing
# config, non-destructively (see docs/03-decisions.md, closes Q-07).
# Pass --dry-run to preview without writing anything.
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose -f compose.yaml -f compose.dev.yaml run --rm lav init "$@"

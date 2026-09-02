#!/usr/bin/env bash
# Fails if a tracked source file's comments cite a docs/ definition or spec
# path as the reason for something — AGENTS.md requires the reasoning
# stated directly in the comment instead. Markdown files and .github/ are
# exempt: a README or an issue template pointing a human at docs/ is a
# legitimate cross-reference, not a comment explaining code.
set -euo pipefail
cd "$(dirname "$0")/.."

hits=$(git grep -n -E 'docs/(0[1-6]-|sdd/)' -- ':!*.md' ':!.github/**' || true)

if [ -n "$hits" ]; then
  echo "Comments must state reasoning directly, not cite docs/ (AGENTS.md):" >&2
  echo "$hits" >&2
  exit 1
fi
echo "ok: no doc citations in code comments"

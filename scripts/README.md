# scripts/

Everything runnable against the project lives here. No long Docker commands
typed from memory, and no dependencies installed on the host beyond Docker.

- `dev-up.sh` — build and run the `lav` daemon locally. Dashboard at
  `http://localhost:8420` once it's up.
- `dev-down.sh` — stop it.
- `lav-init.sh [--dry-run]` — wire LiveAgentsView's hooks into Claude Code,
  Codex and Cursor's existing config, without overwriting anything already
  there. Run this once the daemon is up.
- `lav-status.sh` — list known sessions from the CLI, without opening the
  dashboard.

See [apps/lav/README.md](../apps/lav/README.md) for what the daemon does.

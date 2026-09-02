# scripts/

Everything runnable against the project lives here. No long Docker commands
typed from memory, and no dependencies installed on the host beyond Docker.

- `dev-up.sh` — build and run the `lav` daemon locally via `docker compose`.
  Dashboard at `http://localhost:8420` once it's up. This is the development
  path — fast rebuilds, disposable state.
- `dev-down.sh` — stop it.
- `lav-service-install.sh [--dry-run]` — the real install path: cross-compiles
  a host-native `lav` binary via Docker (no Go installed on the host), copies
  it to `~/.liveagentsview/bin/lav`, and registers it as a launchd (macOS) /
  systemd `--user` (Linux) service so the daemon survives a full host reboot
  without Docker involved at runtime. Mutually exclusive with `dev-up.sh` on
  port 8420 — stop one before starting the other.
- `lav-status.sh` — list known sessions from the CLI, without opening the
  dashboard. Talks to `dev-up.sh`'s container; against the native service use
  `~/.liveagentsview/bin/lav status` directly.
- `check-doc-citations.sh` — fails if a comment in tracked source cites a
  `docs/` definition or spec path as its reasoning (AGENTS.md: state the
  reasoning directly instead). Run before committing code changes.

See [apps/lav/README.md](../apps/lav/README.md) for what the daemon does.

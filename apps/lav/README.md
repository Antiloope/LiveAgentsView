# lav

The LiveAgentsView daemon and CLI: a Go binary that ingests Claude Code, Codex
and Cursor hook events over HTTP, persists sessions to SQLite, and serves an
embedded React dashboard over HTTP+SSE on `127.0.0.1`.

Do not build or run this by hand — see `scripts/` at the repo root. Local
development uses Docker (`scripts/dev-up.sh`) so nothing needs to be installed
on the host beyond Docker itself. Running it for real uses
`scripts/lav-service-install.sh`, which cross-compiles a host-native binary
via the same Docker build and registers it as a launchd/systemd user service
— the daemon itself always runs directly on the host, never in a container
(see "Docker is for developing LiveAgentsView, not for running it" in
[03-decisions.md](../../docs/03-decisions.md)).

## Layout

- `cmd/lav` — CLI entrypoint (`serve`, `init`, `status`, `service install`).
- `internal/model` — canonical Provider/State/Fidelity/Session types.
- `internal/ingest` — one parser per provider, raw hook payload → `model.Signal`.
- `internal/classifier` — end-of-turn classifier (rules-based v1, pluggable).
- `internal/store` — SQLite persistence.
- `internal/daemon` — HTTP routes, SSE hub, the `/api/open-terminal` endpoint.
- `internal/installer` — `lav init`: non-destructive hook merge per provider.
- `internal/service` — `lav service install`: registers the running binary as
  a launchd (macOS) / systemd `--user` (Linux) service.
- `internal/terminal` — spawns a terminal window at a session's `cwd`; only
  works when the daemon runs natively on the host, not inside a container.
- `web` — `go:embed` of the built frontend (`apps/web`); `web/static` is
  populated by the Docker build, not by hand.

## Why hooks call an HTTP endpoint instead of the daemon spawning anything

Sessions are launched natively by the user, not by LiveAgentsView. Ingesting a
hook event only ever receives a POST and writes to SQLite — it never touches
the host filesystem or spawns a process. `/api/open-terminal` is the one
exception: it exists specifically to spawn a terminal on the host, which is
why it only works when the daemon runs natively
(`scripts/lav-service-install.sh`), not inside `scripts/dev-up.sh`'s
container.

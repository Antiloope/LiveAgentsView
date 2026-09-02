# lav

The LiveAgentsView daemon and CLI: a Go binary that launches and pilots
Claude Code and Cursor sessions, persists them to SQLite, and serves an
embedded React dashboard over HTTP+SSE on `127.0.0.1`. Piloted-only: the only
sessions this app knows about are ones it launched itself.

Do not build or run this by hand — see `scripts/` at the repo root. Local
development uses Docker (`scripts/dev-up.sh`) so nothing needs to be installed
on the host beyond Docker itself. Running it for real uses
`scripts/lav-service-install.sh`, which cross-compiles a host-native binary
via the same Docker build and registers it as a launchd/systemd user service
— the daemon itself always runs directly on the host, never in a container
(see "Docker is for developing LiveAgentsView, not for running it" in
[03-decisions.md](../../docs/03-decisions.md)).

## Layout

- `cmd/lav` — CLI entrypoint (`serve`, `uninstall-hooks`, `status`,
  `service install`, `pilot-runner`, `pilot-mcp`).
- `internal/model` — canonical Provider/State/Session types.
- `internal/classifier` — end-of-turn classifier (rules-based v1, pluggable).
- `internal/store` — SQLite persistence.
- `internal/daemon` — HTTP routes, the SSE hub, and piloted-session
  endpoints.
- `internal/pilot` — launches Claude Code and Cursor piloted sessions,
  streams their transcript live, and routes messages, permission decisions,
  interrupts and cancellation to them. Spawns each session's real process
  through a detached `pilot-runner` (below) rather than as its own direct
  child, so a daemon restart doesn't kill an in-progress turn.
- `internal/pilotrunner` — `lav pilot-runner`: the detached shim that owns a
  piloted session's actual `claude`/`agent` process, durably logs its stdout
  to disk, and exposes a control socket so the daemon can reconnect after a
  restart. Not meant to be invoked by hand.
- `internal/pilotmcp` — `lav pilot-mcp`: a tiny stdio MCP server Claude Code
  spawns per session as its `--permission-prompt-tool` target (the only way
  headless Claude Code asks for tool permission at all — confirmed live, it
  never sends one over its main stream-json channel). Relays each call to
  the session's own `pilot-runner` over the same control socket. Not meant
  to be invoked by hand.
- `internal/pilotwire` — the control-socket protocol and on-disk file
  layout shared between `internal/pilot`, `internal/pilotrunner` and
  `internal/pilotmcp`.
- `internal/sse` — the Server-Sent Events hub, shared by the dashboard's
  global session stream and each piloted session's transcript stream.
- `internal/hooksuninstall` — `lav uninstall-hooks`: removes exactly what a
  previous version's `lav init` added to Claude Code/Codex/Cursor's own
  config, backing up each file first.
- `internal/service` — `lav service install`: registers the running binary as
  a launchd (macOS) / systemd `--user` (Linux) service.
- `web` — `go:embed` of the built frontend (`apps/web`); `web/static` is
  populated by the Docker build, not by hand.

## Why piloted sessions need to run natively

`internal/pilot` and `internal/pilotrunner` spawn `claude`/`agent` directly
against the host's real filesystem, git checkouts and login state — that
only works when the daemon runs natively (`scripts/lav-service-install.sh`),
not inside `scripts/dev-up.sh`'s container.

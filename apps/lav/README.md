# lav

The LiveAgentsView daemon and CLI: a Go binary that creates and drives
Claude Code and Cursor characters, persists them to SQLite, and serves an
embedded React dashboard over HTTP+SSE on `127.0.0.1`. The only characters
this app knows about are ones it created itself, every one auto-approving —
there is no permission gate to mediate.

Do not build or run this by hand — see `scripts/` at the repo root. Local
development uses Docker (`scripts/dev-up.sh`) so nothing needs to be installed
on the host beyond Docker itself. Running it for real uses
`scripts/lav-service-install.sh`, which cross-compiles a host-native binary
via the same Docker build and registers it as a launchd/systemd user service
— the daemon itself always runs directly on the host, never in a container.

## Layout

- `cmd/lav` — CLI entrypoint (`serve`, `status`, `service install`,
  `pilot-runner`).
- `internal/model` — canonical Character type: race, class, activity,
  territory. No field on it says whether a process is alive — see
  `internal/daemon`'s presence field.
- `internal/classifier` — end-of-turn classifier (rules-based v1, pluggable).
- `internal/store` — SQLite persistence.
- `internal/territory` — sets up and tears down where a character's process
  runs: either a git worktree this app administers, or a directory the user
  picked, left exactly as it is.
- `internal/daemon` — HTTP routes, the SSE hub, presence and the reconciler
  that keeps a character's recorded activity honest against what is
  actually running.
- `internal/pilot` — creates Claude Code and Cursor characters, streams
  their transcript live, and routes messages, interrupts and stops to them.
  Spawns each character's real process through a detached `pilot-runner`
  (below) rather than as its own direct child, so a daemon restart doesn't
  kill an in-progress turn.
- `internal/pilotrunner` — `lav pilot-runner`: the detached shim that owns a
  character's actual `claude`/`agent` process, durably logs its stdout to
  disk, and exposes a control socket so the daemon can reconnect after a
  restart. Not meant to be invoked by hand.
- `internal/pilotwire` — the control-socket protocol and on-disk file
  layout shared between `internal/pilot` and `internal/pilotrunner`.
- `internal/sse` — the Server-Sent Events hub, shared by the dashboard's
  global character stream and each character's own transcript stream.
- `internal/service` — `lav service install`: registers the running binary as
  a launchd (macOS) / systemd `--user` (Linux) service.
- `web` — `go:embed` of the built frontend (`apps/web`); `web/static` is
  populated by the Docker build, not by hand.

## Why characters need to run natively

`internal/pilot` and `internal/pilotrunner` spawn `claude`/`agent` directly
against the host's real filesystem, git checkouts and login state — that
only works when the daemon runs natively (`scripts/lav-service-install.sh`),
not inside `scripts/dev-up.sh`'s container.

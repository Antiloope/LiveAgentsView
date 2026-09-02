# lav

The LiveAgentsView daemon and CLI. Implements
[docs/sdd/specs/adopted-mode-mvp.md](../../docs/sdd/specs/adopted-mode-mvp.md): a Go
binary that ingests Claude Code, Codex and Cursor hook events over HTTP, persists
sessions to SQLite, and serves an embedded React dashboard over HTTP+SSE on
`127.0.0.1`.

Do not build or run this by hand — see `scripts/` at the repo root. Local
development uses Docker (`scripts/dev-up.sh`) so nothing needs to be installed
on the host beyond Docker itself.

## Layout

- `cmd/lav` — CLI entrypoint (`serve`, `init`, `status`).
- `internal/model` — canonical Provider/State/Fidelity/Session types.
- `internal/ingest` — one parser per provider, raw hook payload → `model.Signal`.
- `internal/classifier` — end-of-turn classifier (rules-based v1, pluggable).
- `internal/store` — SQLite persistence.
- `internal/daemon` — HTTP routes, SSE hub.
- `internal/installer` — `lav init`: non-destructive hook merge per provider.
- `web` — `go:embed` of the built frontend (`apps/web`); `web/static` is
  populated by the Docker build, not by hand.

## Why hooks call an HTTP endpoint instead of the daemon spawning anything

Adopted-mode sessions are launched natively by the user, not by LiveAgentsView.
The daemon only ever receives a POST and writes to SQLite — it never touches
the host filesystem or spawns a process. That is what makes it safe to run
this daemon inside Docker even though
[the Docker decision](../../docs/03-decisions.md) says the *shipped binary*
runs on the host: that constraint is about piloted mode (spawning `claude`/
`codex`/`cursor-agent` as child processes with host keychain access), which is
out of scope for this spec.

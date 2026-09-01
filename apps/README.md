# apps/

Project deployables. Each folder is an application or service that is built,
run, and deployed separately.

There is no shared `packages/`: common pieces live in `scripts/`, `docs/`, and
`compose.yaml`.

## How to add an app

1. Create `apps/<name>/` with its README (internal structure rules).
2. Add the service to `compose.yaml` and `compose.dev.yaml` if it runs locally.
3. If the change is meaningful, open a spec in `docs/sdd/specs/`.

## Status

Empty as of 2026-09-01. The first app depends on defining vision and scope.

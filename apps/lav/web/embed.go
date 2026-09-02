// Package web embeds the built frontend (apps/web, React + Vite) into the
// daemon binary: one binary, no separate Node process, no separate static
// server.
//
// static/ here is populated by the Docker build (frontend stage builds
// apps/web, then copies its dist/ output here before `go build` — named
// "static" rather than "dist" to avoid the repo-wide .gitignore rule for
// Node build output). A placeholder is checked in so `go:embed` never fails
// on an empty directory.
package web

import "embed"

//go:embed all:static
var Dist embed.FS

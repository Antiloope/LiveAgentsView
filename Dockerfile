# Builds the LiveAgentsView daemon: the React dashboard (apps/web) compiled
# to static assets, embedded into the Go binary (apps/lav) via go:embed.
#
# Docker is how this project is built without installing Go or Node on the
# host. The final runtime image below (what scripts/dev-up.sh actually runs)
# can ingest adopted-mode hook events over HTTP with no host access needed,
# but piloted sessions (internal/pilot) spawn `claude`/`agent` against the
# host's real filesystem, git config and login state — those only work when
# the daemon runs natively, via the native-binary stage below and
# scripts/lav-service-install.sh, not in this container.

# --- frontend ---------------------------------------------------------
FROM node:22-alpine AS frontend-build
WORKDIR /app
COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm install
COPY apps/web ./
RUN npm run build

# --- backend ------------------------------------------------------------
FROM golang:1.25-alpine AS backend-build
WORKDIR /src/apps/lav
COPY apps/lav/go.mod apps/lav/go.sum ./
RUN go mod download
COPY apps/lav ./
COPY --from=frontend-build /app/dist ./web/static
RUN CGO_ENABLED=0 go build -o /out/lav ./cmd/lav

# --- native binary (cross-compiled for the host OS/arch; extracted with
# `docker cp`, not run as a container — see scripts/lav-service-install.sh)
FROM golang:1.25-alpine AS native-binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src/apps/lav
COPY apps/lav/go.mod apps/lav/go.sum ./
RUN go mod download
COPY apps/lav ./
COPY --from=frontend-build /app/dist ./web/static
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/lav ./cmd/lav

# --- runtime --------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend-build /out/lav /usr/local/bin/lav
ENV LAV_HOME=/data
VOLUME ["/data"]
EXPOSE 8420
ENTRYPOINT ["lav"]
CMD ["serve"]

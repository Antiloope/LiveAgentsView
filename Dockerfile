# Builds the LiveAgentsView daemon: the React dashboard (apps/web) compiled
# to static assets, embedded into the Go binary (apps/lav) via go:embed.
#
# Docker is how this project is built and run locally without installing Go
# or Node on the host. The daemon itself is safe to run in a container
# because it only receives HTTP hook events and writes to SQLite — it never
# touches the host filesystem or spawns a process.

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

# --- runtime --------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend-build /out/lav /usr/local/bin/lav
ENV LAV_HOME=/data
VOLUME ["/data"]
EXPOSE 8420
ENTRYPOINT ["lav"]
CMD ["serve"]

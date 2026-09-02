#!/usr/bin/env bash
# Cross-compiles a host-native lav binary via Docker (no Go installed on the
# host), installs it at ~/.liveagentsview/bin/lav, and registers it as a
# launchd (macOS) / systemd --user (Linux) service — replacing dev-up.sh's
# `docker compose` restart policy with a real OS-level service. Pass
# --dry-run to preview the service registration without writing anything.
set -euo pipefail
cd "$(dirname "$0")/.."

case "$(uname -s)" in
  Darwin) goos=darwin ;;
  Linux) goos=linux ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  arm64|aarch64) goarch=arm64 ;;
  x86_64) goarch=amd64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

docker build --target native-binary \
  --build-arg TARGETOS="$goos" --build-arg TARGETARCH="$goarch" \
  -t lav-native-build .

container=$(docker create lav-native-build)
trap 'docker rm -f "$container" >/dev/null' EXIT

mkdir -p "$HOME/.liveagentsview/bin"
docker cp "$container:/out/lav" "$HOME/.liveagentsview/bin/lav"
chmod +x "$HOME/.liveagentsview/bin/lav"

"$HOME/.liveagentsview/bin/lav" service install "$@"

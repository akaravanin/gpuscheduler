#!/usr/bin/env bash
# Build the dev image and start the gpuscheduler dev container.
# Source is bind-mounted, so host edits are live inside the container.
set -euo pipefail

IMAGE="gpuscheduler:dev"
CONTAINER="gpuscheduler-dev"
MODCACHE_VOL="gpuscheduler-modcache"
BUILDCACHE_VOL="gpuscheduler-buildcache"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

log "Building $IMAGE"
docker build -t "$IMAGE" "$REPO_ROOT"

# Already running? Nothing to do.
if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || echo false)" = "true" ]; then
  log "$CONTAINER is already running"
else
  # Exists but stopped, or built from a stale image: recreate so the new image is used.
  if docker inspect "$CONTAINER" >/dev/null 2>&1; then
    log "Removing stale container $CONTAINER"
    docker rm -f "$CONTAINER" >/dev/null
  fi

  docker volume create "$MODCACHE_VOL"   >/dev/null
  docker volume create "$BUILDCACHE_VOL" >/dev/null

  log "Starting $CONTAINER"
  docker run -d \
    --name "$CONTAINER" \
    --hostname gpusched \
    -v "$REPO_ROOT":/app \
    -v "$MODCACHE_VOL":/go/pkg/mod \
    -v "$BUILDCACHE_VOL":/go/build-cache \
    -w /app \
    "$IMAGE" >/dev/null
fi

cat <<MSG

  container : $CONTAINER   (source at /app, live-mounted from $REPO_ROOT)
  shell     : docker exec -it $CONTAINER sh
  test      : docker exec -it $CONTAINER go test ./...
  run       : docker exec -it $CONTAINER go run ./cmd/gpusched
  stop      : ./scripts/stop.sh

MSG

#!/usr/bin/env bash
# Stop and remove the gpuscheduler dev container.
#
#   ./scripts/stop.sh              stop + remove the container (caches kept)
#   ./scripts/stop.sh --purge      also remove the image and the Go module/build caches
set -euo pipefail

IMAGE="gpuscheduler:dev"
CONTAINER="gpuscheduler-dev"
MODCACHE_VOL="gpuscheduler-modcache"
BUILDCACHE_VOL="gpuscheduler-buildcache"
PURGE=0

for arg in "$@"; do
  case "$arg" in
    --purge) PURGE=1 ;;
    -h|--help) sed -n '2,7p' "$0"; exit 0 ;;
    *) echo "unknown flag: $arg (try --help)" >&2; exit 2 ;;
  esac
done

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

if docker inspect "$CONTAINER" >/dev/null 2>&1; then
  log "Removing container $CONTAINER"
  docker rm -f "$CONTAINER" >/dev/null
else
  log "No container named $CONTAINER"
fi

if [ "$PURGE" -eq 1 ]; then
  log "Purging image and caches"
  docker rmi -f "$IMAGE" >/dev/null 2>&1 || true
  docker volume rm -f "$MODCACHE_VOL" "$BUILDCACHE_VOL" >/dev/null 2>&1 || true
fi

log "Done"

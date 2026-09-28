# Dev container for gpuscheduler.
#
# This image intentionally does NOT compile the project. The source tree is
# bind-mounted at /app by scripts/start.sh, so edits on the host are live inside
# the container and you build/run/test interactively:
#
#   docker exec -it gpuscheduler-dev sh
#   go test ./...
#   go run ./cmd/gpusched
#
# Module and build caches live in named volumes (see start.sh) so rebuilds are fast
# and survive container recreation.

FROM golang:1.26-alpine

# git: needed by the go tool for VCS stamping and any future module fetches.
# make/bash: convenience for scripted runs. tzdata: sane timestamps in sim output.
RUN apk add --no-cache git make bash tzdata

ENV CGO_ENABLED=0 \
    GOFLAGS=-buildvcs=false \
    GOPATH=/go \
    GOCACHE=/go/build-cache

WORKDIR /app

# Idle. All real work happens via `docker exec`.
CMD ["sleep", "infinity"]

# ---------------------------------------------------------------------------
# Phase 6: when you want a shippable binary, add a multi-stage build here.
# Sketch (uncomment and adapt — this is scaffolding, the Go is yours to write):
#
#   FROM golang:1.26-alpine AS build
#   WORKDIR /src
#   COPY go.mod go.sum ./
#   RUN go mod download
#   COPY . .
#   RUN go build -trimpath -o /out/gpusched ./cmd/gpusched
#
#   FROM gcr.io/distroless/static-debian12
#   COPY --from=build /out/gpusched /gpusched
#   ENTRYPOINT ["/gpusched"]
# ---------------------------------------------------------------------------

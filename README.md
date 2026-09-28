# gpuscheduler

Topology-aware GPU cluster scheduler and discrete-event simulator, in Go.

Places gang-scheduled training jobs onto a modeled `cluster > rack > node > GPU`
hierarchy while minimizing **fragmentation** — free GPUs that exist but not in a shape
any queued job can use. Compares placement policies (first-fit, best-fit, topology-aware)
on identical seeded workload traces, and layers on preemption, fair-share, and backfill.

## Why this is not a CPU scheduler

A CPU scheduler hands out cores one at a time. A training job wants 64 GPUs **all at
once** — a half-placed gang makes zero progress while holding resources. And *which* 64
matters: 64 GPUs packed into 8 nodes of one rack train far faster than 64 scattered
across the cluster, because inter-node and inter-rack communication dominates step time.
So this is topology-aware bin-packing under all-or-nothing constraints, with priorities
and fairness — NP-hard in general, hence heuristics, hence a simulator to measure them.

## Quick start

```sh
./scripts/start.sh                              # build image + start dev container
docker exec -it gpuscheduler-dev sh             # shell in (source live-mounted at /app)
docker exec -it gpuscheduler-dev go test ./...
docker exec -it gpuscheduler-dev go run ./cmd/gpusched
./scripts/stop.sh                               # stop + remove (--purge also drops caches)
```

Host edits appear in the container immediately. Go module and build caches live in named
volumes, so rebuilds stay fast across container restarts.

## Layout

| path | what lives there |
|---|---|
| `cmd/gpusched/` | entry point |
| `cluster/` | topology model, allocation state, fragmentation metric |
| `job/` | job model: gang size, priority, duration, checkpoint state, team |
| `schedule/` | `Policy` seam + placement heuristics, preemption, backfill |
| `sim/` | event loop, arrival process, metrics |
| `docs/` | design notes, phase write-ups, plots |

Each package directory has a `README.md` with the design decisions and traps for that
area — read those before writing the package.

## Status

Scaffolding only. No Go written yet; `cmd/gpusched/main.go` is the first thing to write.
See `CLAUDE.md` for the build plan and the working agreement (short version: the human
writes all Go here; Claude designs, reviews, and stays out of the source).

## Results

*To be filled in with measured numbers and plots as each phase lands. Nothing goes in
this section that didn't come from a real run.*

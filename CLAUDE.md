# gpuscheduler — working agreement

Topology-aware GPU cluster scheduler + simulator, in Go. **The human writes the code.
Claude is the design partner, reviewer, and rubber duck — not the author.**

This project exists to build interview-grade fluency in gang scheduling, fragmentation,
bin-packing, preemption, fairness, and backfill. Fluency comes from typing it, hitting the
bug, and fixing it. Code that Claude writes teaches the human nothing.

## Hard rule: Claude does not write implementation Go in this repo

Claude MUST NOT create or edit any non-test `*.go` file. Not implementations, not "just
the struct definitions", not "here's the interface to get you started", not a one-line fix
to a compile error. If a non-test `.go` file needs to change, the human changes it.

Test files are the one standing exception — see below.

This holds even when:
- the human is stuck and it would be faster to just write it;
- the change is trivial or mechanical (a rename, a missing import, a typo);
- the human describes exactly what they want written;
- Claude is midway through a task and a `.go` edit would unblock it.

### The one escape hatch

The human may hand over a specific file by saying **"take the wheel on <file>"**. That
grants Claude write access to *that file only*, for *that request only*. It expires as
soon as the request is done. Anything vaguer than that phrase — "can you just...",
"go ahead", "fix it" — is NOT a handover; ask whether they mean to use the hatch.

### Standing exception: `*_test.go`

Claude MAY create and edit `*_test.go` files without asking.

The rationale: tests are a *specification* of behavior, and writing the spec doesn't rob
the human of implementing it — arguably it sharpens the exercise, because the target is
unambiguous and failure is legible. Red-to-green is the loop we want.

The boundaries that keep this from becoming a loophole:

- **No implementation smuggled into helpers.** A test may hand-compute an expected value,
  build fixtures, and assert. It must not contain a second, parallel implementation of the
  logic under test (e.g. a helper that computes fragmentation its own way and compares).
  If the test file could be lifted into the package and pass as the implementation, it is
  too clever — rewrite it with literal expected values.
- **Tests do not drag implementation along.** When a test fails, Claude reports the
  failure, explains what it means, and points at the likely line. Claude does not touch
  the non-test file to make it pass. Ever. That is the whole point.
- **Tests pin an API; say so.** A test file inevitably fixes method names and signatures.
  Claude states plainly which surface a new test file assumes, so the human can rename
  freely rather than discovering the constraint by compile error.
- **Prefer `package <pkg>_test`** (external test package) so tests exercise the public
  surface only. It proves the package is actually usable from `schedule` and `sim`, and it
  stops tests from reaching into internals the real callers can't see.
- **No fabricated results.** Claude runs the tests and reports what actually happened,
  including "this doesn't compile yet because the methods don't exist" — which is the
  expected state when tests are written first.

## What Claude SHOULD do (be genuinely useful, not withholding)

- **Explain the concept properly.** Gang scheduling, DRF, EASY backfill, why fragmentation
  is hard to even define. Depth is welcome. Concepts are not code.
- **Design in prose and pseudocode.** Sketch the shape of a type, the fields a struct
  probably needs, the steps of an algorithm — in chat, in a fenced block that is clearly
  pseudocode or clearly illustrative, never written to a `.go` file. Prefer describing an
  interface in words ("a Policy takes a cluster snapshot and a job, returns a placement or
  a reason it failed") over dictating Go verbatim.
- **Write and maintain tests** (see the `*_test.go` exception above), including tests
  written before the implementation exists.
- **Review code the human wrote.** This is the highest-value mode. Be specific and honest:
  point at the line, name the failure case, say what breaks. Do not soften real bugs.
- **Interrogate the design.** "What happens to that when two jobs finish in the same tick?"
  "How does this behave when every node has exactly 1 GPU free?" Ask the questions an
  interviewer would ask.
- **Read the code freely** to ground answers. Reading is always allowed.
- **Debug by direction, not by patch.** Read the error, explain what it means and where to
  look. Say "your event loop pops before it checks the reservation" — not a diff.
- **Run things**: `go build ./...`, `go test ./...`, `go vet`, the sim, docker. Reporting
  what broke is fine and useful; fixing it in the file is not.
- **Interview-drill on demand.** When asked, play the interviewer on any of these topics.

## What Claude owns outright (edit freely, no permission needed)

Everything that is not Go source, and not the human's learning objective:
- `*_test.go` (per the standing exception above)
- `Dockerfile`, `.dockerignore`, `scripts/*.sh`
- CI config (`.github/workflows/*`), `.gitignore`, `Makefile`
- `README.md`, `docs/*`, plots and write-up prose — *except* claims about results, which
  must come from a real run, never from what the numbers "should" look like.
- `go.mod` / `go.sum` housekeeping when a tool (`go mod tidy`) produces it.

Non-Go scaffolding is plumbing, not the lesson. Don't make the human yak-shave Docker.

## Style of assistance

- Lead with the answer or the key tradeoff. No preamble, no rehearsing the question.
- One good question beats five. Don't run a Socratic interrogation when a direct
  explanation is what was asked for.
- When the human's approach is worse than an alternative, say so plainly and say why,
  then let them choose. Don't quietly steer.
- Never fabricate simulator results. If a number isn't from a run that actually happened,
  it doesn't get stated.

## Project shape

    cmd/gpusched/   entry point
    cluster/        topology model (cluster > rack > node > GPU), allocation state
    job/            job model: gang size, priority, duration, checkpoint state, owner/team
    schedule/       Policy interface + first-fit, best-fit, topology-aware, backfill, preemption
    sim/            discrete-event loop, arrival process, metrics
    docs/           design notes, phase write-ups, plots

Run everything in Docker: `./scripts/start.sh` (build + start dev container),
`./scripts/stop.sh` (stop + remove). Source is bind-mounted, so edits on the host are
live in the container. Shell in with `docker exec -it gpuscheduler-dev sh`.

## Goals (the commitment; each needs a "done when" to be met)

These five goals define what the project must demonstrate. Judge scope questions
against them: work that serves a goal comes first.

1. **Gang scheduling.** All-or-nothing placement.
   *Done when:* tests show a job is never running with fewer GPUs than its gang size, a
   job that doesn't fit waits rather than partially starting, and a rejected `Allocate`
   leaves the cluster unchanged.
2. **A fragmentation metric.**
   *Done when:* the starter metric (free GPUs not on a fully-free node) is implemented
   and tested, sampled over time in the sim, and the README explains where it fails,
   ideally with one better definition implemented beside it for comparison.
3. **First-fit vs best-fit vs topology-aware on the same job trace.**
   *Done when:* all three policies run on identical seeded traces, and a results table
   reports utilization, avg/p95 wait, fragmentation, and nodes/racks spanned, as means
   over N seeds with confidence intervals. Every number comes from a real run.
4. **Preemption with a checkpoint/restart cost.**
   *Done when:* a preempted job loses the work since its last checkpoint and pays a
   restart cost; it is requeued as the same job with its progress kept; victim selection
   minimizes lost work while freeing enough GPUs in a usable shape; and a measurement
   shows the trade-off (high-priority wait saved vs GPU-hours lost).
5. **A small HTTP API.**
   *Done when:* there are endpoints to submit a job, get a job's status, and get cluster
   state and fragmentation, as a thin layer over the same scheduler core the sim uses
   (no scheduling logic in handlers), with a curl walkthrough in the README.
   *Open question:* the sim runs in virtual time; the API needs either a live wall-clock
   mode or a "step the sim" endpoint. Decide this before building it.

Goal-to-phase map: 1 → Phase 0/1, 2 → Phase 0 then 2, 3 → Phases 1–2, 4 → Phase 3,
5 → Phase 6.

## Build plan (the human drives; Claude advises per phase)

- **Phase 0** — packages, `go test ./...`, CI green.
- **Phase 1** — `placementCost = a·nodesSpanned + b·racksSpanned`, real search over candidates.
- **Phase 2** — best-fit-decreasing, Poisson arrivals, heavy-tailed job sizes, policy comparison table.
- **Phase 3** — preemption coupled to checkpoint cost; minimal victim set.
- **Phase 4** — teams, quotas, DRF / hierarchical fair-share.
- **Phase 5** — EASY backfill with reservation for the head-of-line job.
- **Phase 6** — small HTTP API (goal 5), metrics dashboard, README with plots, write-up.

Each phase deliverable is a *measurement*, not just working code. "Topology-aware cut
fragmentation from X to Y over N runs" is the interview payload.

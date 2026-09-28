# schedule — placement policies

## The Policy seam
Every policy answers one question: *given the cluster state and a job, where does it go —
or why can't it go anywhere?* Keep that seam narrow and everything else plugs in:
alternative heuristics, backfill, preemption, and eventually a Kubernetes/Volcano-style
plugin story you can talk about in an interview.

Return a placement **and** a reason on failure. "Doesn't fit" vs "fits but violates quota"
vs "fits only if we preempt" are different signals the queue logic needs.

## Policies, in build order
- **First-fit** — walk nodes, take what you find. Fast, scatters badly. Your baseline.
- **Best-fit-node** — prefer the node that leaves the least slack. Packs partially-used
  nodes and so *preserves whole-empty nodes* for big jobs. That preservation is the
  anti-fragmentation move; be able to say it in one sentence.
- **Topology-aware** — tiered: fits on one node → fits in one rack → minimize racks.
  Phase 1 replaces the tiers with a continuous cost `a·nodesSpanned + b·racksSpanned`
  and a real search over candidates. Continuous scoring is tunable and comparable;
  tiers are neither.
- **Best-fit-decreasing** — sort the queue largest-first. Classic bin-packing result:
  big items placed first pack better.

## The two hard behaviors
- **Preemption** (Phase 3): choosing a *minimal* victim set is itself an optimization
  problem — minimize wasted compute (time since each victim's last checkpoint), subject
  to freeing enough GPUs in a good enough shape. Only preempt when net-positive.
- **Backfill** (Phase 5, EASY): compute a reservation time for the blocked head-of-line
  job, then run anything that provably finishes before it. Fixes head-of-line blocking
  *without* violating priority — the correctness property you must be able to state.

## Testing
These are pure functions over a cluster state. Table-driven tests with hand-built tiny
clusters (2 racks × 2 nodes × 4 GPUs) catch nearly everything. Assert on the *shape* of
the placement (nodes spanned, racks spanned), not on specific GPU IDs, or every test
breaks the first time you change iteration order.

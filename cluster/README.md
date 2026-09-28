# cluster — topology model and allocation state

Models the physical machine so placement quality is measurable.

## Hierarchy
`Cluster > Rack > Node > GPU`. Two things make a placement good or bad:
- GPUs on the **same node** talk over NVLink (fast).
- Nodes in the **same rack** share an IB leaf switch (medium). Cross-rack goes up
  through spine (slow). This is the tier structure the whole project optimizes for.

## Things to decide up front (they shape every later phase)
- **GPU identity.** Are GPUs individually addressed (`rack3/node7/gpu2`), or is a node
  just a count of free GPUs? Counting is simpler; individual identity is what you need
  if you ever model NVLink domains within a node (e.g. 8-GPU node = two 4-GPU domains).
- **Allocation representation.** A placement is a set of (node, count) pairs. Store it
  with the job, and store a back-reference on the node, or you'll be scanning the whole
  cluster to answer "who owns this GPU?" during preemption.
- **Snapshot vs live.** Policies want to *evaluate* candidate placements without
  mutating. Either give policies a read-only view, or make allocate/free cheap and
  reversible. Decide now — retrofitting this in Phase 3 is painful.

## Fragmentation
The starter definition: *fraction of free GPUs that are not part of a fully-free node.*
Cheap to compute, easy to explain, and wrong in interesting ways — it says nothing about
which job sizes you can still place. Better definitions to try in later phases:
- **Placement-weighted**: given the job-size distribution, what fraction of sizes can no
  longer be placed at all? Or expected largest placeable job.
- **Locality-weighted**: free GPUs discounted by how much NVLink/rack locality they've lost.
Implement the simple one first, then argue about it in the README. *Naming the
metric-definition problem is the senior signal.*

# job — the unit of work

A training job, not a task. The defining property: it needs **all** its GPUs at once
(gang scheduling). A half-placed job makes zero progress while holding resources.

## Fields you'll need eventually
- `GPUs` — gang size. Heavy-tailed in reality: many 1–8, few 256+.
- `Priority` — drives queue order and who preempts whom.
- `Duration` — for the simulator, the *true* runtime; for backfill, the user's
  *estimate*. Real schedulers only have the estimate, and users lie. Modeling that gap
  is a great stretch goal.
- `Team` / `User` — needed in Phase 4 for fair-share.
- Checkpoint state (Phase 3): last checkpoint time, checkpoint interval, cost to write
  one. Preemption without checkpointing throws away hours of compute — the two are
  coupled, and saying so unprompted is one of the two things that reads as senior.
- Lifecycle timestamps: submit, start, end. Wait time = start − submit; that's the
  headline metric for every phase.

## Watch out
- A preempted-and-requeued job is the *same* job. Keep its identity and its accumulated
  progress; don't create a new one, or your wait-time stats will lie.

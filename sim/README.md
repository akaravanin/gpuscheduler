# sim — discrete-event simulator and metrics

The scheduler is validated as a simulator; that's how real schedulers are developed too,
so this is a legitimate engineering choice, not a shortcut. Say it that way.

## Event loop
A priority queue of events ordered by time — `JobArrives`, `JobCompletes`, and later
`PreemptionDone`. Pop the earliest, mutate state, try to schedule, push new events.
Virtual time only; never sleep.

Two traps:
- **Ties.** Two events at the same timestamp must have a deterministic order or runs
  aren't reproducible. Break ties on a monotonic sequence number.
- **When to re-run the scheduler.** After every completion (resources freed) and every
  arrival. Anything else and you'll silently leave the cluster idle with a runnable queue.

## Workload generation (Phase 2)
- Poisson arrivals — one tunable knob (λ) that sweeps you from idle to overloaded.
  Most interesting behavior lives near saturation; report load as offered utilization.
- Heavy-tailed sizes — many small jobs, a few huge ones. Powers of two (1,2,4,8,…) with
  decreasing probability is realistic enough and makes gang-scheduling pain visible.
- **Fixed seed.** Every policy comparison must run the *same* trace, or you're comparing
  random noise. Seed it, log the seed.

## Metrics
Utilization; avg and p95 wait by priority; fragmentation sampled over time; makespan;
and in Phase 4 a fairness index (Jain's, or per-team share vs quota). Report a mean over
N seeded runs **with confidence intervals** — a single run proves nothing, and being the
candidate who says that is worth more than the scheduler itself.

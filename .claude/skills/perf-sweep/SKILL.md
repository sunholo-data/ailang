---
name: perf-sweep
description: Monthly AILANG runtime performance and memory sweep — re-measures workload p95 latency, peak RSS of five memory-shaped programs (live-data recursion, effect results, log lines, string building, deep-trace ratio) and allocation-per-op of the evaluator and builtins, diffs against the last banked snapshot in .ailang/state/perf/, names what got slower or fatter with the command that produced each number, and exits 2 on a regression. Use when the user says "perf sweep", "monthly sweep", "is ailang getting slower", "memory regression", "performance audit", "check the latency budget", "profile ailang", "where does it allocate", or asks whether a change made the runtime slower or use more memory, even without the word sweep. NOT for eval-suite pass rates (eval-analyzer), file sizes (codebase-organizer), or count/duplication (simplicity-audit).
---

# Perf Sweep

The file-size gate and the simplicity audit see the code; nothing saw the runtime until
M-V1-MEMORY-FOOTPRINT (2026-09-16) found that "logging on" multiplied peak RSS 3–20× and
that two external reporters had measured the tracer and blamed `concat` and `cons`. This
skill re-measures the runtime once a month and diffs the numbers so a regression is a
row in a table, not a host OOM three weeks later.

## Current State

- **Last banked snapshot**: !'ls .ailang/state/perf/ 2>/dev/null | tail -1 || echo none'
- **Machine**: !'sysctl -n machdep.cpu.brand_string 2>/dev/null || grep -m1 "model name" /proc/cpuinfo | sed "s/.*: //"'
- **Binary under test**: built from this tree by the sweep, never the PATH one
- **Ledger targets**: `benchmarks/budget_ledger.md` (hand-edited, 50/50 rule)

## Run it

```bash
make perf-sweep            # full: 5 workload runs, count=3 benches, hotspots; ~4 min
make perf-sweep-quick      # 3 runs, count=1; ~30 s — fine for a first look
make perf-sweep-control    # positive control: MUST print WORSE and exit 2, banks nothing
```

Under the hood `scripts/sweep.sh` runs `tools/perf_sweep.sh` (the measurement, one JSON),
compares it with the newest snapshot under `.ailang/state/perf/`, prints a before/now/delta
table with each metric's tolerance, lists workloads over their ledger target and every
regression with the exact command that produced the number, prints the top allocation
sites of the `list_large` workload, banks today's snapshot, and exits 2 if any metric moved
past its tolerance **on the same machine class**. Across machine classes it shows the diff
and refuses to gate; re-baseline instead.

Snapshots are tracked in git so the diff works on any machine. **Commit the new snapshot**
or next month's diff is against a stale base.

## Measurement rules (each learned the hard way)

1. **Tracing off on every measured run.** The default tier is `standard`, and it is
   load-bearing on any number anyone reports. `AILANG_NO_TRACE=1` is set by the sweep;
   set it yourself for any ad-hoc probe.
2. **RSS only for live-data shapes, allocation counts for "does it copy".** Under the CLI's
   GOGC=500, peak RSS is where the GC cycles land: one binary read 0.96× on the rig and
   1.37× on the ubuntu runner. The sweep pins GOGC=100 for its RSS rows and still moved a
   copy-on-write map probe +34% with no change — that row is now a Go benchmark (B/op).
   If you add an RSS row and it moves past tolerance with no code change, it is
   garbage-dominated: move it to `go test -bench … -benchmem`.
3. **A memory limit cannot fix live data.** 772 MB of live accumulator under
   `--max-memory 64MB` spent 11 s in GC and changed nothing. `GODEBUG=gctrace=1` tells you
   which kind of peak you have before you reach for a limit.
4. **AILANG recursion keeps every frame's bindings until it unwinds.** A "tail-recursive"
   per-item loop frees nothing; per-item work belongs in a step function the loop calls
   (`foldlE`, `mapE`): 870 MB → 212 MB for the same 40 file reads.
5. **Machine class is part of the number.** The ledger's seed rows are an Apple M2 at
   v0.11; today's M4 Max measures `list_large` at ~102 ms against a 900 ms target. When the
   class changes, re-baseline the ledger, do not celebrate.

## What to do with the output

1. **A `WORSE` row** is a regression against last month. Find the commit:
   `git log --since=<last snapshot date> -- internal/eval internal/builtins internal/effects internal/trace`,
   then confirm with the row's own `how` command on the suspect commit and its parent —
   the sweep gives you the reproduction, not the cause.
2. **A workload over its ledger target** is an SLO breach; the ledger's 50/50 rule says
   how to pay for it (offset, flag-gate, or an accepted-regression row in a design doc).
3. **A `better` row** is a deposit: record the p95 delta in the ledger per its process.
4. **The hotspot list** is context, not a finding — `evalCore`/`applyFunction`/`Environment`
   dominate every month. Something new at the top is the finding.
5. Route fixes through a design doc and `sprint-planner`; the sweep is the observe step.
   The known structural rows (`rss_cons_6000_mb`, `rss_string_build_20k_mb`,
   `eval_mapinsert5k_b_op`) are owned by M-LIST-CONS-QUADRATIC (parked) and the string
   builder / persistent map items in the M-V1-MEMORY-FOOTPRINT Future Work. They are
   expected to be flat until those land, and to drop sharply when they do.

Report in this shape: the delta table, the regressions with their `how`, the over-target
workloads, and one recommended action per regression. Do not fix during the sweep.

## Monthly schedule

First Monday of the month, after the weekly `simplicity-audit`:

- **Claude Code routine** (cloud): `/schedule` with the prompt
  `run the perf-sweep skill, commit the snapshot, and report regressions` on `0 10 1-7 * 1`.
  The cloud runner is a different machine class from the rig, so the first run there
  banks a baseline and the diff only gates from the second.
- **Rig launchd**: a `dev.ailang.perf-sweep` job is a `mission-loop-change` task so the
  plist inherits the memory-admission gate and the HOLD protocol; the rig is the class the
  ledger should be re-baselined on.

## Positive control

Once a quarter, before trusting a green run: `make perf-sweep-control`. It runs the
live-data probe at 1.7× depth, which must print `rss_cons_6000_mb … WORSE` and exit 2
without banking. If it passes green, the instrument is broken, not the runtime.

## Resources

- [resources/instruments.md](resources/instruments.md) — which question, which instrument; what each sweep row exists for
- [resources/principles.md](resources/principles.md) — optimisation principles (profile first, algorithms over micro-opts, batch, fast path)
- [resources/go_patterns.md](resources/go_patterns.md) — Go-side patterns for the runtime
- Design record: `design_docs/implemented/v1_0_0/m-v1-memory-footprint.md` (the audit that found the rows)
- Memory guide: `docs/docs/guides/debugging.md#memory` (every control, the probe protocol)

## Boundaries

- `benchmark-runner` and `eval-analyzer` own model pass rates and cost; this skill owns
  the runtime's own speed and memory. `simplicity-audit` owns count and duplication.
- The scripts are bash 3.2 / BSD grep / macOS awk safe (the rig). Keep them so.
- Replaces the April `perf-reviewer` skill (cross-language fibonacci timings, no snapshot);
  its principles and Go patterns live on under `resources/`.

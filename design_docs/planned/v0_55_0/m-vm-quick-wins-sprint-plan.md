# M-VM-QUICK-WINS Sprint Plan: Phases 0–1 (runtime-only)

**Design doc:** [m-vm-quick-wins.md](m-vm-quick-wins.md)
**Sprint ID:** M-VM-QUICK-WINS
**Scope:** Phase 0 (baseline and parity gate) and Phase 1 (W1–W4, `internal/vm` only).
Phase 2 (W5–W7, compiler) is **not in this sprint**. It waits for M-VM-COVERAGE Phase 1
(PR #1989, which rewrites `internal/bytecode/compiler/call.go`). Phase 3 (GC) is measurement-only
and runs as part of M5.
**Duration:** ~2 days, 5 milestones, each sized for one executor run of 60–90 minutes.
**Risk:** low–medium. All changes are localized and reversible, with a keep/drop rule per win.
**Mode:** attended, executed in-session, committed straight to `dev` (dev-workflow rule).

## Pre-flight findings (2026-10-28)

- **Overlap with PR #1989** (M-VM-COVERAGE, open). It edits `internal/vm/vm.go` in `CALL`/`TAIL_CALL`:
  it adds an over-application branch after `argCount := int(inst.B())`, and moves `CallClosure`
  to `closure_call.go`. W2 changes `TAIL_CALL`'s argument stash about 30 lines below that branch.
  The hunks are adjacent, not overlapping, so whichever lands second rebases. W1 (`OpClosure`),
  W3 (arith dispatch) and W4 (`frame.go`) are untouched by the PR.
- **W1 invariant holds against PR #1989.** Its local-`letrec` lowering emits `OpClosure` with
  pseudo-MOVE captures and never mutates `Captures` after creation (diff grep: no `Captures[...] =`
  and no `.Captures =`).
- **Closure equality is structural** (`Value.Equal` compares `Proto` and then `Captures`), so sharing
  zero-capture closures cannot change an equality result.
- **perf-sweep (`tools/perf_sweep.sh`) has no VM metric.** It already turns `go test -bench`
  output into `ns/op` (tol 0.20) and `B/op` (tol 0.05) metrics. VM metrics reuse that path.
- **No CI target runs VM parity.** `scripts/verify_bytecode_parity.go --json` is the instrument.
- **Registry reuse:** `none` for every milestone. This is VM-internal runtime work with no
  package-shaped capability.
- **AILANG syntax gate:** the only `.ail` code is three benchmark programs, already checked with
  `ailang check` during design (prompt v0.16.7). No showcase module: contracts skip (benchmark
  kernels, not specs), effects `! {IO}` only on `main`, inline tests skip (expected outputs are
  asserted by the Go benchmark harness).

## Baseline (from the design doc, V1/V2)

| Workload | VM today | Target | Allocated today | Target |
|---|---|---|---|---|
| `fib(32)` | 0.50 s | ≤ 0.35 s | 330 MB | < 10 MB |
| 10M tail loop | 1.07 s | ≤ 0.65 s | 1.7 GB | < 50 MB |
| map + fold 2M | 0.18 s | no regression | — | — |

## Milestones

### M1_BASELINE_AND_GATE: VM benchmarks, perf-sweep metrics, parity baseline (~120 LOC)

- Add `internal/runner/vm_bench_test.go`. It compiles each workload once (pipeline →
  `CompileBytecodeFromResult`), then times only VM execution in `b.Loop()`, with
  `b.ReportAllocs()` and an assertion on the expected result:
  - `BenchmarkVMFib25` (`fib(25)` = 75025; small enough for `-count 3`)
  - `BenchmarkVMTailLoop1M` (`loop(0, 1_000_000, 0)`; the expected value is computed in Go in the test)
  - `BenchmarkVMMapFold200K`
- Wire them into `tools/perf_sweep.sh` §3 as `vm_<name>_ns_op` (le, tol 0.20) and `vm_<name>_b_op`
  (le, tol 0.05).
- Bank a perf snapshot: run `tools/perf_sweep.sh --quick` and keep the JSON.
- Record the parity baseline: `go run ./scripts/verify_bytecode_parity.go --json`, plus a
  `--strict-bytecode` pass/fail per file over `examples/runnable`, written to
  `.ailang/state/perf/vm-parity-baseline-2026-10-28.json`. Add a small diff script
  `tools/vm_parity_diff.sh <baseline.json>`: exit 1 if any file changes outcome.
- **Acceptance:** the benchmarks pass and report allocs; perf_sweep emits the six `vm_*` metrics;
  the parity baseline is banked and `tools/vm_parity_diff.sh` against itself exits 0; a positive
  control (a deliberately broken outcome row) exits 1.

### M2_W1_W2_ALLOC: shared zero-capture closures + tail-call scratch (~150 LOC)

- W1 in `OpClosure`: when `NumCaptures == 0`, use a lazily built per-VM cache indexed by image
  prototype.
- W2 in `TAIL_CALL`: use a VM-owned scratch slice instead of `make` per call.
- Tests in `internal/vm/vm_quickwins_test.go`:
  - a zero-capture `OpClosure` executed twice yields the same `*ClosureObj`; a capturing one yields
    distinct objects;
  - `testing.AllocsPerRun` on a 10k-step tail loop is O(1) and independent of step count (compare
    1k and 10k);
  - a tail call whose arguments come from a HOF callback (re-entrancy) still gives the right answer.
- Mutation check: revert each fix in a scratchpad copy and confirm the allocation test fails.
- **Acceptance:** `BenchmarkVMFib25` and `BenchmarkVMTailLoop1M` B/op fall by at least 90%; the
  parity diff is empty; `make test-changed` passes; the keep rule is met and numbers are in the commit.

### M3_W3_INT_FASTPATH: inline int arithmetic and comparison (~120 LOC)

- In the dispatch cases for `OpAdd/OpSub/OpMul` and `OpLt/OpLe/OpEq`, handle int×int inline.
  `Div`/`Mod` keep `CheckIntDivisor` (RT001).
- Table test: for every op over every tag pair (including mismatches, `minInt`, and zero divisors),
  the fast path and `arith()`/`runtimeEq`/the comparison helper give identical value or error text.
- **Acceptance:** the table test passes; `vm_divzero_test.go` is unchanged and passing; the parity
  diff is empty; the keep rule is applied (≥ 5% on some VM benchmark, or the change is reverted and
  recorded as dropped).

### M4_W4_FRAME_RESET: cheaper tail-call register reset (~80 LOC)

- First prove or refute the invariant: no compiled prototype reads a non-parameter, non-capture
  register before writing it. Add a test that walks every prototype in the images compiled from
  `examples/runnable` and checks read-before-write over straight-line and branch paths, using a
  conservative dataflow pass.
- If the invariant holds, `reuseFor` zeroes only `[NumParams + len(captures), NumRegs)`, after the
  arguments are written. If it fails, drop W4 and record why.
- **Acceptance:** the invariant test exists and passes (or W4 is recorded as dropped with the
  counterexample); the parity diff is empty; the keep rule is applied.

### M5_MEASURE_AND_REPORT: final numbers, GC measurement, docs (~60 LOC docs)

- Re-run the CLI timings (best of 3, `fib(32)`, 10M tail loop, map+fold 2M) and `--memprofile`
  allocations; run `tools/perf_sweep.sh` against M1's snapshot (exit 0 required: no metric over tolerance).
- Phase 3: if GC is still above 10% of VM CPU samples, measure `GOGC=200/400` and `GOMEMLIMIT` versus
  RSS. Write a recommendation only.
- Update the LIMITATIONS numbers (both files), add a changelog fragment, and fill the design doc's
  implementation report for Phases 0–1, with Phase 2 left planned.
- **Acceptance:** the targets in the Baseline table are met, or the shortfall is reported with its
  numbers; perf-sweep shows no regression; `make check-changelog check-file-sizes` passes.

## Validation (every milestone)

`make test-changed`, `go test ./internal/vm/... ./internal/runner/... ./internal/bytecode/...`,
`make lint` on touched packages, `make check-file-sizes`, and the parity diff
(`tools/vm_parity_diff.sh`). Never `make test` in full: CI runs it.

## Out of scope

W5–W7 (compiler side, after PR #1989), the 40-byte `Value`, GC default changes, and evaluator performance.

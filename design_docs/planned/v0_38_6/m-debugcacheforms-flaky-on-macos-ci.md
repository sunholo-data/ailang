# M-DEBUGCACHEFORMS-FLAKY-ON-MACOS-CI — drain the capture pipe before closing its read end

**Status**: Planned
**Target**: v0.38.6
**Priority**: P1 (non-blocking red on `dev` — `Build macos-latest` is not a required context, but every occurrence costs a re-run and hides other reds)
**Estimated**: 0.5 day (one milestone; M2 does not exist — see the audit)
**Dependencies**: None
**Planner-Lane**: codex-ok
**Mission**: V1, iteration 353 (designer). Queue row `m-debugcacheforms-flaky-on-macos-ci`; the red was introduced by V1's own landing [#1071](https://github.com/sunholo-data/ailang/pull/1071) (`202358fd3`).

> **Quorum trigger statement (design-doc-creator skill):** none of the four attended triggers fires —
> no design-freeze item needs a human, nothing overrides shared machinery (the only change is to a
> test-file helper with three call sites in the same file), no cost/KPI/banked-data surface, and
> every premise is verifiable in-repo or from CI logs fetched in-session. This doc is produced
> unattended, so the controller runs the quorum anyway; the log table is at the end.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The test helper becomes deterministic: the captured bytes no longer depend on goroutine scheduling |
| A2: Replayability | 0 | Test-only change; no trace or replay surface |
| A3: Effect Legibility | 0 | No effect signatures change |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | The helper gains its own local, hermetic regression test that goes red on the exact defect |
| A6: Safe Concurrency | +1 | Removes a reader/closer race in the test harness (close-after-drain ordering) |
| A7: Machines First | 0 | No prompt, syntax or token-cost impact |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | 0 | No composition surface |
| A11: Structured Failure | 0 | No error-shape changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the opposite; one is removed
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

`TestPipelineModulePhases_DebugCacheFormsAndCounters` (`internal/pipeline/pipeline_module_phases_test.go:200`)
fails intermittently on the `Build macos-latest` job of the `Build and Release` workflow and has never
failed locally. Every one of the four most recent `Build macos-latest` failures on `dev` names this
test in its job log (V2, controller-measured); the run `34774738497` / job `103770669326` instance
(commit `e7b8787f7`, a docs-only change) reads:

```text
=== RUN   TestPipelineModulePhases_DebugCacheFormsAndCounters
    pipeline_module_phases_test.go:232: cold miss diagnostic missing: "[CACHE] std/option: MISS\n"
--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters (0.04s)
```

The captured `cold` stderr contains ONLY the first of the four lines the pipeline writes on a cold
compile (`[CACHE] std/option: MISS`, `[CACHE] std/result: MISS`, `[CACHE] answer: MISS`,
`[CACHE] Summary: …` — order measured in V9). The capture is **truncated, not wrong**: the bytes the
assertion wants were written, and lost. The sibling `Build macos-latest` job of the same run
(`103770669327`, same commit, started three minutes earlier) passed.

**Mechanism (measured, V6/V7):** the test-file helper `capturePipelineStderr`
(`pipeline_module_phases_test.go:28-44`) swaps `os.Stderr` for the write end of an `os.Pipe()`,
starts a goroutine that `io.Copy`s the read end into a buffer, runs `f()`, and then tears down in
this order:

```go
os.Stderr = old
_ = w.Close()
_ = r.Close()   // <-- closes the READ end while the copier may still be mid-drain
<-done
```

Closing an `*os.File` out from under a reader makes the pending `Read` return an error, and
`io.Copy` stops; whatever is still in the kernel pipe buffer is discarded. On a 12-core laptop the
copier has drained everything before `f()` returns, so the order never matters. On a loaded CI
runner the copier is scheduled late and the tail of the output is lost. A standalone probe with a
reader that is slow between reads loses bytes in **50/50** iterations under the HEAD ordering and
**0/50** with `<-done` moved before `r.Close()` — same reader, same writes, one variable changed.
The partial-capture shape the probe produces (`"[CACHE] std/opti"`, V7) is the CI shape.

**Current State:**
- `4/25` of the most recent `Build and Release` runs on `dev` concluded `failure`, three on
  docs-only commits and one on a `tools/` fix (the briefing said all four were docs-only), every one with a `Build macos-latest` job as the failing job [`gh run list`, V1].
- **All 4 of the 4 failed macOS jobs fail on this test** [V2, controller-measured from the raw job
  logs]: `103770676134`, `103763201443`, `103758453565` each print
  `pipeline_module_phases_test.go:245: warm skip diagnostic missing: ""` (the WARM capture came back
  completely EMPTY — total loss, the reader had read nothing before the close), and `103770669326`
  prints the `:232` first-line-only cold capture above. Three loss shapes (empty, first line only,
  last line missing) — all consistent with one mechanism and inconsistent with a wrong oracle.
  The `cancelled` jobs beside each failure are the run's OTHER matrix legs (the second
  `macos-latest` entry, `windows`, `ubuntu`), cancelled seconds after the failure by the matrix's
  default `fail-fast: true` (V14) — they are a *consequence* of this test's failure, not a separate
  failure mode. (A first draft of this doc read them as an independent "external cancellation"
  defect; that was an instrument error — `gh run view --log` on a job whose log carries terminal
  escape sequences returns an error line, and the empty grep was read as absence. Rule 3a.)
- The charter row's earlier instance ("`:268 invalid summary counters missing` at `5debf4e10`")
  could not be re-derived from a fetchable log (the run for that commit, `34119398470`, shows
  `--- PASS` in the jobs still fetchable, i.e. the failed job's log has expired or was re-run). The
  claim is not load-bearing — the mechanism explains a lost last line exactly as it explains a
  lost tail — but it is recorded as UNVERIFIED.
- Local non-repro (control): `-count=20 -race` → `ok`; `-count=1 -v` prints `--- PASS:` so the
  selector matches [V3].

**Impact:**
- Every V1 landing on `dev` currently has a small chance of a red `Build macos-latest` badge
  that a human then has to read past. Two standing reds is how the third gets missed.
- The same helper shape, if copied, would carry the race to any future stderr-capture test.

## Goals

**Primary Goal:** make `capturePipelineStderr` return every byte written to `os.Stderr` during
`f()` regardless of how late the reader goroutine is scheduled, and prove it with a test that is
red on the pre-fix ordering.

**Success Metrics:**
- `TestCapturePipelineStderr_GatedReaderLosesNothing` (new): exact-equality capture on every
  iteration post-fix, and `rc=1` on **every** run under MUT-1 (`r.Close()` moved before the gate
  release) — a structural kill, no parameters to tune.
- `TestCapturePipelineStderr_PanicRestoresStderr` and `TestCapturePipelineStderr_DrainTimeoutIsLoud`
  (new): the teardown is panic-safe and its wait is bounded with a loud diagnostic.
- `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` stays `rc=0` (baseline
  `rc=0`, V10).
- CI watch (secondary, explicitly weak — see Success Criteria): no `Build macos-latest` job whose
  log contains `--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters` across the next 25
  `dev` runs, counting only jobs in which the test reached a `--- PASS`/`--- FAIL` line.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix the **helper's teardown order** (writer → wait → reader), not the assertions | The assertions are correct once the capture is complete; a structural-counter oracle read through the same broken capture truncates identically. Fixing the oracle would leave the root cause in place and hide it | agent (controller recommendation adopted; argued in Ruled-out) | design | low |
| Inject a **channel-gated reader** and a **drain-wait hook** through an options seam (`capturePipelineStderrWith(opts, f)`), keeping `capturePipelineStderr(f)` as the production-facing signature with `opts == nil` | The three existing call sites must not change; the regression test needs to *hold the reader before its first underlying `Read` until the teardown releases it*, so that the broken ordering (MUT-1) is red by construction rather than by scheduling luck. A `nil` options value must reproduce today's behaviour exactly (nil-hook invariant) | agent (quorum r1: gpt6-astra's fix adopted) | design | low |
| The teardown is a **named-return `defer`** and its wait on the copier is **bounded by a fixed local deadline** with a panic diagnostic, not an unbounded `<-done` | Without `defer`, a panicking `f()` leaves `os.Stderr` swapped and the copier leaked for every later test in the package; a naive `defer` with `return buf.String()` evaluates the operand *before* the deferred `<-done` (data race, AC4). `testutil.HangGuard` (V18) needs a `*testing.T` the helper does not have; a 10 s local deadline is six orders of magnitude above the microsecond drain of a ≤4 KiB payload and far below both `go test`'s default 10 m and the 30 m CI job timeout (V14), so it bounds the hang without threading `t` through the frozen call sites | agent (quorum r1: gemini-3-1-pro's fix + gpt6-astra's bounded-wait ask) | design | low |
| **M2 does not exist**: the audit (V5) finds the close-before-wait order in exactly one helper; every other pipe-capture helper in the repo drains synchronously or waits before closing | Padding the sprint with "harmless" reorders of correct helpers would be scope without a killer test | agent (measured) | design | low |
| Primary acceptance is the **deterministic helper test**; the CI green streak is a **secondary observation with stated power**, and a single green re-run is NOT evidence | The mechanism is a scheduling race; local greens certify nothing about the runner. At the measured base rate (4 failures in ≤50 macOS job executions across 25 runs, ~8% per execution, V2) a 50-execution clean streak has ~1.5% probability with no fix, so the streak corroborates but the helper test is the proof | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Teardown runs inside a **`defer` with a named return** `(out string)` in this order:
  `os.Stderr = old; w.Close(); <drain-wait hook, if set>; bounded wait on done; r.Close();
  out = buf.String()`. The named return is mandatory: a bare `return buf.String()` evaluates its
  operand before the deferred wait and races the copier. No `time.Sleep`, no `runtime.Gosched()`,
  no retry loop (all three would be silent fallbacks, Principle 2).
- [x] The wait on `done` is bounded by a **fixed local deadline (10 s, overridable via the options
  seam for the timeout test only)**. On expiry the teardown closes `r` (which unblocks a production
  reader parked in `Read`) and **panics** with a diagnostic naming the deadline and the two possible
  causes (a writer still holds the pipe's write end — `f()` leaked `os.Stderr` into a goroutine —
  or an injected reader never released). `os.Stderr` is already restored at that point. The
  diagnostic does not read `buf` (the copier may still own it). `testutil.HangGuard` is *not*
  reused — it needs `*testing.T` (V18) and the seam deliberately does not thread `t`; the
  rationale is in High-Impact Decisions.
- [x] The regression test's fixed payload is the four V9 cold lines (~150 bytes) and the test
  asserts `len(payload) <= 4096` with a comment stating why: below pipe capacity, `f()`'s writes
  never block while the gated reader is parked (V19: 64 KiB measured locally; 4 KiB is under every
  documented default on any platform).
- [x] `capturePipelineStderr(f)` is `capturePipelineStderrWith(nil, f)`; with `nil` options the
  helper installs no wrapper and no hook — production call sites see only the reorder.
- [x] The regression test lives in the same package and file family
  (`internal/pipeline/pipeline_module_phases_test.go` or a sibling `_test.go`) and uses an
  injected reader; it does not exercise the compiler pipeline at all.
- [x] The Windows `t.Skip` at `:201-215` is untouched (different, pre-existing defect —
  `m-cache-module-id-encoding`, parked on D-57).

Nothing here needs a human: every row is agent-resolvable and measured.

## Solution Design

### Overview

The helper's teardown closes the read end of the pipe before the copier goroutine has finished
draining it. The fix is to close in the order **writer → wait for drain → reader**: closing the
writer is what delivers EOF to the copier (the write end has exactly one fd, `w`, so `w.Close()`
guarantees EOF once the buffer is drained); waiting on `done` then guarantees the buffer holds every
byte; closing the reader last releases the fd. This is the ordering already used by
`internal/eval/prelude_cap_gate_test.go:73-77` (V5, positive control for the audit scanner).

A regression test for the helper itself injects a **channel-gated reader**: its `Read` parks on a
channel before its first underlying `Read`, so the copier cannot consume anything until the
teardown's **drain-wait hook** closes the gate. The payload is fixed and far below pipe capacity, so
`f()`'s writes complete into the kernel buffer without blocking. Under the fixed order the hook
runs after `w.Close()`, the reader then drains the whole buffer to EOF and `done` closes: exact
capture, every time. Under MUT-1 (`r.Close()` before the hook) the reader's *first* underlying
`Read` is on an already-closed file: it returns an error, `io.Copy` stops with nothing captured, and
the exact-equality assertion is red — regardless of goroutine scheduling, because the reader was
never runnable before the close. The kill is structural, not timing-based; nothing is tuned.

### Architecture

**Components:**
1. **`captureOpts`** (test-local struct) — `wrap func(io.Reader) io.Reader` (reader wrapper),
   `release func()` (the drain-wait hook: invoked by the teardown after `w.Close()` and before the
   bounded wait; the helper's own wait is the wait-for-completion, so the hook has no timing role),
   `drainTimeout time.Duration` (zero → the 10 s default; set only by the timeout test).
2. **`capturePipelineStderrWith(opts *captureOpts, f func()) (out string)`** — the helper body,
   with the named-return `defer` teardown. `opts == nil` → no wrapper, no hook, default deadline.
3. **`capturePipelineStderr(f func()) string`** — unchanged signature, now `return
   capturePipelineStderrWith(nil, f)`. All three call sites (`:226`, `:239`, `:256`, V4) are
   byte-for-byte untouched.
4. **`gatedReader`** (test-local type) — holds the underlying reader, a `gate chan struct{}` and an
   atomic `reads` counter. `Read` blocks on `<-gate`, increments `reads`, then delegates. The test
   closes the gate from `opts.release`.
5. **`TestCapturePipelineStderr_GatedReaderLosesNothing`** — N (≥50) iterations; each installs a
   fresh gated reader, writes the four fixed V9 cold lines to `os.Stderr` inside `f`, and asserts
   (a) the capture equals the payload exactly and (b) `reads ≥ 1` (the wrapper was actually
   installed and invoked — kills MUT-2). `t.Fatalf` with the iteration index, `reads`, and the
   captured bytes on the first miss. Post-condition: `os.Stderr` restored after each iteration.
6. **`TestCapturePipelineStderr_PanicRestoresStderr`** — `f()` panics; the test recovers and asserts
   `os.Stderr` is the original and `w` is closed (kills MUT-3a).
7. **`TestCapturePipelineStderr_DrainTimeoutIsLoud`** — a wrapper whose gate is never released,
   `drainTimeout: 50ms`; asserts the helper panics with a message containing `did not drain` and
   that `os.Stderr` is restored (kills MUT-3b; see the mutation table for the leak note).

```go
// After (sketch — the executor writes the real code; probed end-to-end in V20):
type captureOpts struct {
	wrap         func(io.Reader) io.Reader // test-only reader wrapper
	release      func()                    // drain-wait hook: runs after w.Close(), before the wait
	drainTimeout time.Duration             // 0 → 10s
}

func capturePipelineStderrWith(opts *captureOpts, f func()) (out string) {
	r, w, err := os.Pipe()
	if err != nil {
		panic("os.Pipe: " + err.Error())
	}
	old := os.Stderr
	os.Stderr = w
	var src io.Reader = r
	if opts != nil && opts.wrap != nil {
		src = opts.wrap(r)
	}
	timeout := 10 * time.Second
	if opts != nil && opts.drainTimeout > 0 {
		timeout = opts.drainTimeout
	}
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = io.Copy(&buf, src); close(done) }()
	defer func() {
		os.Stderr = old // 1. restore first: even a panicking f() leaves no swapped stderr behind
		_ = w.Close()   // 2. writer: the only write-end fd; EOF reaches the copier once drained
		if opts != nil && opts.release != nil {
			opts.release() // 3. drain-wait hook (nil in production): lets a gated reader start reading
		}
		select { // 4. bounded wait: every byte is in buf when done closes
		case <-done:
		case <-time.After(timeout):
			_ = r.Close() // unblocks a production reader parked in Read; do NOT read buf here
			panic("capturePipelineStderr: copier did not drain within " + timeout.String() +
				" after the write end was closed (f leaked os.Stderr into a goroutine, or an injected reader never released)")
		}
		_ = r.Close()     // 5. reader: release the fd only after the drain
		out = buf.String() // 6. named return: assigned AFTER <-done, so no race with the copier
	}()
	f()
	return
}

func capturePipelineStderr(f func()) string { return capturePipelineStderrWith(nil, f) }
```

**Why MUT-1 is deterministic under this design:** MUT-1 moves step 5 (`r.Close()`) above step 3.
The gated reader has not executed a single underlying `Read` before step 3 — it is parked on the
gate, not on the file — so when it is released its first `Read` is on a closed `*os.File`, which
returns `os.ErrClosed` unconditionally. `io.Copy` returns, `done` closes, `buf` is empty, and the
exact-equality assertion fails on iteration 0. No scheduler interleaving can make the reader
consume the buffer before the close, because it cannot run until after the close. In production
(nil options) MUT-1 is exactly the HEAD defect.

**Bounded wait, stated:** the mechanism is a local `select` on `done` versus `time.After(10s)`.
`testutil.HangGuard`/`HangGuardContext` (V18) were examined and not reused: both take `*testing.T`
(their value is capping by `t.Deadline()`), the helper has none, and threading `t` would change the
seam to `(t, opts, f)` for a wait that completes in microseconds. On expiry: `r.Close()`, then a
panic whose text names the deadline and the two possible causes; `os.Stderr` is restored before the
panic. The panic is loud (Principle 2), not a silent fallback.

### Implementation Plan

**M1: reorder the teardown + regression test of the helper** (~3 hours)
- [ ] H1 — `pipeline_module_phases_test.go:28-44`: introduce `captureOpts` and
  `capturePipelineStderrWith(opts, f) (out string)` with the named-return `defer` teardown
  (restore → `w.Close()` → hook → bounded wait → `r.Close()` → `out = buf.String()`); keep
  `capturePipelineStderr` as a one-line wrapper passing `nil`. Update the doc comment to state the
  ordering invariant and why (the pipe read end must outlive the drain; the named return must be
  assigned after the wait).
- [ ] H2 — add `gatedReader` + the three tests (`GatedReaderLosesNothing`, `PanicRestoresStderr`,
  `DrainTimeoutIsLoud`). Before committing, run the mutation drill (MUT-1, MUT-2, MUT-3a, MUT-3b
  from the table); each mutant must be `rc≠0` on every run, the restored tree `rc=0`. Record the
  rcs in the sprint checkpoint. Nothing is tuned: a mutant that is not red is a design defect to
  report, not a parameter to raise.
- [ ] H3 — one line in `changelogs/v0.32-current.md` under the unreleased section (`Fixed`).
- [ ] Acceptance sweep (commands below) on the branch; then open the PR. The CI watch starts at
  the merge commit.

**M2: does not exist.** The audit in V5 read all 14 pipe-using test files by eye; the
close-before-wait order occurs in one helper only. Nothing else to reorder.

### Files to Modify/Create

**Modified files:**
- `internal/pipeline/pipeline_module_phases_test.go` (+~110/−8 LOC) — H1 helper reorder + options
  seam + bounded `defer` teardown, H2 gated-reader type and the three helper tests. Call sites at
  `:226`, `:239`, `:256` unchanged. The file is 403 LOC today; H2 in a sibling file keeps it in
  the comfort band (Deferred Decisions).
- `changelogs/v0.32-current.md` (+1 LOC) — H3.

**New files:** none. (The agent may put H2 in a sibling `pipeline_stderr_capture_test.go` if the
phases file would cross the 500-LOC comfort band; either is acceptable — Deferred Decisions.)

## Examples

### Example 1: the failing capture, before and after

**Before** (HEAD ordering; reproduced with the probe, V7 — `head=true lost=50/50`):
```text
captured: "[CACHE] std/opti"
assertion: strings.Contains(cold, "[CACHE] answer: MISS")  → false
pipeline_module_phases_test.go:232: cold miss diagnostic missing: "[CACHE] std/option: MISS\n"
```

**After** (writer → wait → reader; V7 — `head=false lost=0/50`):
```text
captured: "[CACHE] std/option: MISS\n[CACHE] std/result: MISS\n[CACHE] answer: MISS\n[CACHE] Summary: 0 hits, 3 misses (3 modules cached)\n"
assertion: true
```

### Example 2: the regression test's mutation drill

```bash
# green at the fixed ordering
go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=1 -timeout 60s -v        # rc=0, three --- PASS lines
# MUT-1: move `_ = r.Close()` above the release hook / bounded wait (one-line edit), then:
for i in 1 2 3 4 5; do go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1; done
#   → rc=1 all five times: "iteration 0: reads=1 got="" want <payload>" (V20: 50/50 lost, 5/5 runs red)
# restore; whole package green again
go test ./internal/pipeline/ -count=1                                                          # rc=0
```

## Success Criteria

Acceptance commands, each baselined on the pristine worktree at `05a6457d1` (rc recorded in the
Verification Log). `go build ./...` is **not** listed: it is `rc=1` at base on this repo
(`cmd/wasm` has no native `main`, V11).

- [ ] **AC1** `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` → `rc=0`
  (baseline `rc=0`, V10). Can fail for this diff: H1 changes the helper all six assertions read
  through.
- [ ] **AC2** `go test ./internal/pipeline/ -run 'TestCapturePipelineStderr_' -count=1 -timeout 60s -v`
  → `rc=0` AND the output contains all three of
  `--- PASS: TestCapturePipelineStderr_GatedReaderLosesNothing`,
  `--- PASS: TestCapturePipelineStderr_PanicRestoresStderr`,
  `--- PASS: TestCapturePipelineStderr_DrainTimeoutIsLoud`
  (baseline: `no tests to run`, so the `--- PASS` lines are the load-bearing half).
- [ ] **AC3 — deterministic MUT-1 drill.** With `_ = r.Close()` moved above the release hook, run
  `go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1`
  **five times**: `rc=1` on every run, first failure at iteration 0 with an empty capture. Restore:
  `rc=0`. No parameter is tuned between runs; if any run is `rc=0` the gate is not holding the
  reader before its first `Read` and the test is wrong, not slow. All rcs go in the checkpoint.
  The same drill for MUT-2, MUT-3a, MUT-3b (one run each suffices — each is structural) is
  recorded alongside.
- [ ] **AC4** `go test ./internal/pipeline/ -run TestPipelineModulePhases_DebugCacheFormsAndCounters -count=20 -race`
  → `rc=0` (baseline `rc=0`, V3). Cannot *prove* the flake is gone (see AC6) but can catch a
  regression in the reordered teardown under the race detector.
- [ ] **AC5** `go build ./internal/pipeline/ && go vet ./internal/pipeline/` → `rc=0` (baseline
  `rc=0`, V11).
- [ ] **AC6 — CI observation, stated as such.** The mechanism is a scheduling race; local greens
  certify nothing about the runner. After the merge commit lands on `dev`, the controller measures
  with `gh run list --workflow "Build and Release" --branch dev --limit 25` and, for every
  `Build macos-latest` job that concludes `failure`, reads the job log for
  `--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters`. **Pass:** zero such jobs across
  25 consecutive post-landing runs (≤50 macOS job executions), counting only jobs in which the test
  reached a `--- PASS`/`--- FAIL` line (fail-fast-cancelled sibling legs are neither).
  **Baseline:** 4 attributable failures in the 25 pre-landing runs (≤50 macOS executions, ~8% per
  execution, V2), plus a 5th on merge SHA `45f02deb3` (job `102044971960`). **Power, honestly:**
  at ~8% per execution a clean 50-execution streak has ~1.5% (0.92^50) probability with no fix, so
  AC6 is strong corroboration but still not proof — AC2/AC3 are the proof. A single green re-run of
  a failed job is NOT evidence of anything (the sibling macOS job in the failing run was green at
  the same commit, V2).
- [ ] All tests passing
- [ ] Changelog updated (H3)

### Named mutations, anchored to the diff (rule 3n)

| Hunk | Mutation | Killer test | Measured basis |
|------|----------|-------------|----------------|
| H1 (helper reorder) | **MUT-1**: move `_ = r.Close()` above the release hook / bounded wait (= the HEAD defect for nil options) | `TestCapturePipelineStderr_GatedReaderLosesNothing` → `rc=1` **on every run** | **Deterministic by construction:** the gated reader has performed no underlying `Read` before the hook releases it, so under MUT-1 its first `Read` is on a closed fd → `os.ErrClosed` → `io.Copy` returns with an empty `buf` → exact-equality fails at iteration 0. No interleaving lets the reader run before the close. Probed: `50/50` lost, `5/5` runs `rc=1`, `0/50` lost fixed, race-clean (V20). Mechanism link to CI: V6/V7 |
| H1 (seam) | **MUT-2a**: `capturePipelineStderrWith` ignores `opts.wrap` (always reads `r` directly) | `TestCapturePipelineStderr_GatedReaderLosesNothing` — the `reads ≥ 1` assertion on the gated reader is red (the wrapper was never invoked) | Structural: `reads` is only incremented inside the wrapper's `Read` |
| H1 (seam) | **MUT-2b**: `capturePipelineStderrWith` ignores `opts.release` (hook never invoked) | `TestCapturePipelineStderr_GatedReaderLosesNothing` — the gate is never opened, the bounded wait expires, the helper panics → `rc=1` (the test sets `drainTimeout: 2s` so the mutant is red in seconds, not 10 s × N) | Structural: nothing else closes the gate |
| H1 (teardown) | **MUT-3a**: the `defer` removed (teardown inlined after `f()`) | `TestCapturePipelineStderr_PanicRestoresStderr` — a panicking `f()` skips the inlined teardown, `os.Stderr` stays swapped → assertion red | Structural; probed green at the fixed form (V20) |
| H1 (teardown) | **MUT-3b**: the bounded wait replaced by a bare `<-done` | `TestCapturePipelineStderr_DrainTimeoutIsLoud` — the never-released wrapper makes the bare wait hang; the test is red via `go test`'s `-timeout` (AC2 passes `-timeout 60s`; the mutant exits `rc=2` with a goroutine dump). Non-hermetic kill, declared as such: the mutant is red in 60 s, not instantly | Structural; probed green at the fixed form in 50 ms (V20). Note: the test deliberately leaks one goroutine parked on its own gate; `os.Stderr` is restored and `r`/`w` are closed before the panic, so nothing leaks into later tests |
| H2 (new tests) | — | none: H2 *is* the killer; declared as such | — |
| H3 (changelog) | — | none: prose | — |

**Determinism argument (replaces any tuning clause).** Every mutant above is red because of a
happens-before relation the test constructs, not because of a duration: MUT-1 because the reader
cannot read before the close; MUT-2a because the counter lives in the wrapper; MUT-2b/MUT-3b
because a gate nobody closes is never open; MUT-3a because a panic skips non-deferred code. There
are no sleeps, byte caps or iteration counts to raise. If the executor finds any of them not red,
the correct action is to report the design defect, not to adjust a parameter.

## Testing Strategy

**Unit tests:**
- `TestCapturePipelineStderr_GatedReaderLosesNothing` (new, hermetic, no pipeline involvement):
  N≥50 iterations, the four fixed V9 cold lines (~150 B, asserted ≤ 4 KiB), gated reader released
  by the drain-wait hook, exact-equality assertion, `reads ≥ 1` on the wrapper (kills MUT-2a),
  `drainTimeout: 2s` (kills MUT-2b quickly), `os.Stderr` restored post-condition.
- `TestCapturePipelineStderr_PanicRestoresStderr` (new): panicking `f()`; asserts restore and a
  closed write end (kills MUT-3a).
- `TestCapturePipelineStderr_DrainTimeoutIsLoud` (new): never-released wrapper, `drainTimeout:
  50ms`; asserts a panic containing `did not drain` and a restored `os.Stderr` (kills MUT-3b).

**What the regression test is, and is not.** The regression test is a structural invariant proof,
not a natural-failure reproduction: under the fixed ordering, a complete drain is guaranteed by
`w.Close()` → EOF → `<-done` regardless of reader speed, so the gated reader serves only to make
MUT-1 (broken ordering) deterministically red. The CI failure mode (scheduler-late reader) is the
same mechanism the gate forces — a reader that has not yet read when the close happens; V6/V7
establish the causal link, and the fixed-ordering guarantee is unconditional. (Recorded from
oc-glm-5-2's round-1 objection.)

**Integration tests (existing, unchanged):**
- The six `[CACHE]` assertions in `TestPipelineModulePhases_DebugCacheFormsAndCounters` continue
  to read through the fixed helper; their string oracles are deterministic once capture is complete
  (V9: the same four lines in the same order on every cold run).

**Manual / CI:**
- AC6 watch by the controller across the next 25 `dev` runs, with the exclusion rule above.

## Ruled-out (do not inherit from the queue row)

- **"Timing-dependent `cached 0s ago` text trips the assertion."** REFUTED by reading the test:
  the warm assertion is `strings.Contains(warm, "[CACHE] answer: SKIP (cached ")` — a prefix; no
  duration is compared (`:247`, V8). The invalid-phase assertion is likewise a prefix
  (`"[CACHE] answer: INVALID, recompiling"`, `:265`).
- **"Assert on the counters structurally rather than on the rendered summary."** A legitimate
  hardening, but it does not touch the measured mechanism: a structural oracle would still be read
  through the same truncated capture (the counters are only observable via the `Summary` line the
  pipeline writes to `os.Stderr` at `pipeline_module_cache.go:123`, V8) and would lose the same
  bytes. Recommendation: leave the string oracle alone — it is deterministic once capture is
  correct — and fix the helper. Declined as scope; may be queued independently if someone wants a
  counters seam for other reasons.
- **Concurrent writers to `os.Stderr` interleaving.** Not the mechanism: no goroutines in the
  module-phase code path (`grep "go func|errgroup|sync.WaitGroup"` over the non-test
  `pipeline_module*.go` + `cache_runtime.go` → 0 hits, control `metrics.go` hits) and zero
  `t.Parallel()` in `internal/pipeline/*_test.go` (control: `internal/lsp` has two) — V8.
- **A second `t.Skip`.** Would remove the last platform on which this behaviour is tested;
  the Windows skip is for a different defect and stays.
- **`time.Sleep` / `runtime.Gosched()` before `r.Close()`.** A silent, timing-based fallback
  (Principle 2); the correct fix is an ordering invariant, not a delay.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether H2 lives in `pipeline_module_phases_test.go` or a sibling
  `pipeline_stderr_capture_test.go` — agent may choose (file-size band).
- Exact `N` (≥50) for the gated-reader test — agent may choose; it has no bearing on the kill.
- Names of the seam function, the options struct and its fields — agent may choose;
  `capturePipelineStderr(f)` must keep its signature and the `nil`-options path must be
  behaviour-identical to today apart from the reorder and the bounded wait.

## Non-Goals

**Not attempted in this feature:**
- The `cancelled` sibling matrix legs in each failing run: they are `fail-fast: true` cancelling
  the other legs after this test fails (V2, V14), not a separate defect. Nothing to queue.
- The Windows `t.Skip` at `:201-215` — a pre-existing drive-letter cache-path defect owned by
  `m-cache-module-id-encoding` (parked on D-57). Not touched.
- A structural counters oracle — see Ruled-out.
- A gatelint rule (`internal/testutil/gatelint`, rules R1–R3 exist, V13) that flags
  `r.Close()`-before-wait in pipe helpers — worth a Future Work line, not this sprint: the audit
  shows one instance and a heuristic scanner would need care around the synchronous-drain shapes.
- Any change to the `[CACHE]` writers in `pipeline_module_cache.go` or the
  `cacheDependencies.stderr` seam in `cache_runtime.go` — production code is not the cause.

## Timeline

**Day 1** (~4 hours, single milestone):
- H1 helper reorder + seam (0.5h)
- H2 gated-reader + panic + timeout tests, mutation drill MUT-1/2a/2b/3a/3b (2h)
- H3 changelog, acceptance sweep, PR (1h)
- Controller: AC6 watch runs passively over the following `dev` pushes

**Total: ~4 hours, 1 milestone**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The fixed payload grows past pipe capacity (a future edit adds lines) → `f()` blocks writing while the gated reader is parked → deadlock | Low | The test asserts `len(payload) <= 4096` with the reason in a comment; capacity measured at 64 KiB locally (V19) and the smallest documented default anywhere is 4 KiB (V19, EXTERNAL PREMISE for other kernels). If it ever happens the bounded wait does not help (the hang is inside `f()`), but `go test -timeout` reds it loudly and the comment names the cause |
| The options seam or the hook leaks into production call sites and changes the behaviour the six `[CACHE]` assertions read through | Low | Nil-hook invariant (Design Freeze): `capturePipelineStderr(f)` passes `nil`; with `nil` no wrapper is installed, no hook runs, the default deadline applies. AC1 (`-count=1`) and AC4 (`-count=20 -race`) exercise exactly this path |
| The bounded wait expires spuriously on a loaded runner and panics a test that would have drained | Very low | 10 s for a microsecond drain of ≤ 4 KiB after the only write end is closed; the CI job's own budget is 30 min (V14). A spurious expiry would be loud with a diagnostic naming the two real causes — not a silent drop, which is the failure mode being fixed |
| A future `f()` leaks `w` (via `os.Stderr`) into a goroutine, so EOF never arrives | Low | Now bounded: the wait expires, `r` is closed, and the panic names this exact cause. Previously an unbounded hang |
| AC6 cannot distinguish "fixed" from "lucky" at the measured base rate | Med | Stated in AC6 with the power estimate; AC2/AC3 are the proof; the controller does not close the row on AC6 alone |
| A different macOS red appears after this lands and gets blamed on this test | Low | AC6 counts only job logs carrying `--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters`; any other `Build macos-latest` red is a new row |

## Conflict Surface

This change touches no parser/typechecker/codegen/effects path; the section is written anyway
because the helper swaps a process-global (`os.Stderr`).

### Positions touched

- The teardown of `capturePipelineStderr` at `internal/pipeline/pipeline_module_phases_test.go:39-43`
  (global `os.Stderr` restore + pipe close order).
- The helper's signature is preserved by adding a second function; nothing else in the package
  references the pipe.

### What else lives here

| Position | Existing occupant | Shape | Decision |
|----------|-------------------|-------|----------|
| Callers of `capturePipelineStderr` | 3 call sites, all in `pipeline_module_phases_test.go` (`:226` cold, `:239` warm, `:256` invalid) — the queue briefing said six; grep shows 3 calls + 1 definition + 1 doc-comment mention = 5 hits (V4) | `x := capturePipelineStderr(func() { … })` | reuse — signature unchanged |
| Writers to the captured stream | `pipeline_module_cache.go:65,85,87,123` (`fmt.Fprintf(os.Stderr, "[CACHE] …")`) — direct global; `cache_runtime.go:110-131` writes `CACHE_INVALID`/`CACHE_SOURCE_UNAVAILABLE`/`CACHE_WRITE_FAILED` through the `cacheDependencies.stderr` seam, which `productionCacheDependencies()` binds to `os.Stderr` at construction (V8) | `fmt.Fprintf(os.Stderr, …)` / `fmt.Fprintf(runtime.stderr, …)` | untouched |
| Other test helpers that swap `os.Stderr` (must not be affected; none imports this one) | `cmd/ailang/messages_send_test.go:184`, `cmd/ailang/pkg_lock_ratchet_test.go:27,65,127,155`, `internal/ai/cache_warnings_test.go:20`, `internal/apiserver/debug_sink_test.go:27`, `internal/coordinator/daemon_test.go:230,285`, `internal/effects/fs_sandbox_debug_test.go:29,138,165`, `internal/parser/delimiter_trace_test.go:137,180,220,269,392,449,587` (V4) | per-package helpers, all in their own package | untouched; each was read for the audit (V5) and none has the close-before-wait order |
| Shared test utilities | `internal/testutil` has **no** pipe/capture helper (`grep 'os\.Pipe\|func Capture'` → 0; control: 6 exported helpers present, V12). It does have `HangGuard(t, cap)` / `HangGuardContext(t, cap)` (V18), both `*testing.T`-bound | — | no capture helper to reuse; do not create one in this sprint (one consumer). `HangGuard` examined and not reused — the helper has no `t`; a fixed 10 s local deadline is used instead (High-Impact Decisions). `internal/pipeline` tests already import `testutil` (3 files) and there is no cycle (V18), so reuse would be *possible* later if the seam ever threads `t` |

### Programs / tests that MUST still work

- `TestPipelineModulePhases_DebugCacheFormsAndCounters` — all six assertions (AC1).
- The other seven `TestPipelineModulePhases_*` tests in the file (`grep -c '^func Test'` = 8, V12) — they do not use the helper but share the package run (AC1).
- `internal/eval/prelude_cap_gate_test.go` `captureStdout` — the already-correct ordering; untouched, cited as the pattern.

### What deliberately changes

- Bytes written to `os.Stderr` inside `f()` after the copier was scheduled late are now captured
  instead of dropped. No test relied on the drop (a truncated capture only ever made assertions
  fail).

## Verification Log

Every codebase/CI claim above maps to a row. Commands were run in the worktree at `05a6457d1`
(= `origin/dev`) on 2026-09-14 unless stated; empty/negative results carry a known-positive control.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | 4 of the last 25 `Build and Release` runs on `dev` failed; each failing job is `Build macos-latest`; each is a docs-only commit | `gh run list --workflow "Build and Release" --branch dev --limit 25 --json databaseId,conclusion,headSha,displayTitle` | `failure`: `34774741191` (f23f78d51 docs), `34774738497` (e7b8787f7 docs), `34772012484` (29b252639 docs(registry)), `34770261581` (4c61f1143 fix(agent-set) — a `tools/` change, not design_docs; the briefing's "all docs-only" is 3 of 4); other 21 `success`. `gh run view <id> --json jobs` for each: the `failure` job is `Build macos-latest` in all 4 (`103770676134`, `103770669326`, `103763201443`, `103758453565`) |
| V2 | All 4 recent failed macOS jobs fail on THIS test; three with an EMPTY warm capture, one with a first-line-only cold capture; the `cancelled` jobs are fail-fast siblings | (controller, 2026-09-14) `gh api --allow-escape-sequences repos/sunholo-data/ailang/actions/jobs/<job>/logs > log; grep -aE 'pipeline_module_phases_test.go:[0-9]+' log` for jobs 103770676134, 103763201443, 103758453565, 103770669326; `gh run view <run> --json jobs --jq '.jobs[] \| "\(.name): \(.conclusion) \(.startedAt) \(.completedAt)"'` for runs 34774741191, 34772012484, 34770261581, 34774738497; `grep -n fail-fast .github/workflows/build.yml` | `:245: warm skip diagnostic missing: ""` ×3, `:232: cold miss diagnostic missing: "[CACHE] std/option: MISS\n"` ×1 — 4 of 4 `--- FAIL: TestPipelineModulePhases_DebugCacheFormsAndCounters`. In every run the `cancelled` legs completed 8–25 s after the `failure` leg; `fail-fast` is unset (default true). The designer's first reading ("1 of 6") came from `gh run view --log`, which errors on logs containing terminal escape sequences — an empty grep read as absence (rule 3a); superseded by this row |
| V3 | Local non-repro; selector matches | `go test ./internal/pipeline/ -run TestPipelineModulePhases_DebugCacheFormsAndCounters -count=20 -race`; `… -count=1 -v` | `ok … 3.889s` rc=0; `--- PASS: TestPipelineModulePhases_DebugCacheFormsAndCounters (0.03s)` |
| V4 | 3 call sites of the helper, all in one file; the set of test files that swap `os.Stderr` | `grep -rn 'capturePipelineStderr' --include='*.go' .`; `grep -rn 'os\.Stderr = ' --include='*_test.go' .` | 5 hits: `:19` (comment), `:28` (def), `:226`, `:239`, `:256` (calls). Stderr swaps: the 8 files listed in the Conflict Surface table |
| V5 | Systemic audit: 14 test files use `os.Pipe()`; exactly one closes the read end before waiting; by-eye per-file verdicts | `grep -rl 'os\.Pipe()' --include='*_test.go' .` (14 files); awk window "`r.Close()` then `<-done`/`wg.Wait()` within 3 lines" (HEAD order) and the mirrored window (positive control); then `sed -n` of each helper body | HEAD-order scan: **`internal/pipeline/pipeline_module_phases_test.go:41→42` only**. Mirror scan (control): `internal/eval/prelude_cap_gate_test.go:76→77`. By-eye: `cmd/ailang/eval_suite_flags_test.go:239-262` goroutine reads, `w.Close()` then `return <-done`, `r` never closed → safe; `ext_registry_gen_test.go:109-117` `w.Close()` then synchronous read → safe; `messages_send_test.go:180-194` goroutine, `w.Close()`, `<-done` → safe; `mission_activation_unix_test.go:64-73` pipe is a child's stdin (`defer read.Close()`, parent closes `write`) → not a capture, safe; `pkg_commands_test.go:723-742,760-777` `writeEnd.Close()` → `io.ReadAll(readEnd)` → `readEnd.Close()` synchronous → safe; `pkg_lock_ratchet_test.go:26-38 (+64,126,154)` `w.Close()` → `buf.ReadFrom(r)` synchronous → safe; `internal/ai/cache_warnings_test.go:15-27` `w.Close()` → `io.Copy` synchronous → safe; `internal/apiserver/debug_sink_test.go:22-39` goroutine, `w.Close()`, `<-done` → safe; `internal/coordinator/daemon_test.go:226-251,281-304` `w.Close()` → single synchronous `r.Read(4096)` → `r.Close()` → safe for the one-line message it captures; `internal/effects/fs_sandbox_debug_test.go:25-37 (+134,161)` `w.Close()` → `io.Copy` synchronous → safe; `internal/effects/io_test.go:19-37,51-69` synchronous; `:140-146` writer goroutine feeding stdin → safe; `internal/eval/prelude_cap_gate_test.go:60-77` goroutine, `w.Close()`, `<-done`, `r.Close()` → the correct order; `internal/parser/delimiter_trace_test.go:136-156 (+6 more)` `w.Close()` → synchronous `buf.ReadFrom(r)` → safe. **Verdict: one defective helper; M2 does not exist** |
| V6 | Mechanism: closing the read end before the drain loses bytes; waiting first loses none | Standalone probe `/tmp/iter353_probe/main_test.go` (Go, `os.Pipe`, copier goroutine, reader sleeps 200µs before each ≤16-byte `Read`), 50 iterations per arm, `go test -count=1 -v` | `HEAD ordering (r.Close before <-done): lost 50/50` (first loss `got ""`); `FIX ordering (<-done before r.Close): lost 0/50` |
| V7 | The CI partial-capture shape reproduces (reader prompt on first read, late afterwards) | probe `variant_b_test.go`: first `Read` prompt, later reads sleep 200µs, ≤16 bytes | `head=true lost=50/50 partial(non-empty)=24 sample="[CACHE] std/opti"`; `head=false lost=0/50` |
| V8 | Writers/seam/no goroutines/no `t.Parallel`; warm & invalid assertions are prefixes | `grep -n 'Fprintf(os.Stderr' internal/pipeline/pipeline_module_cache.go`; `sed -n 10,36p internal/pipeline/cache_runtime.go` + `grep -n 'runtime.stderr' …`; `grep -rn "go func\|errgroup\|sync.WaitGroup" internal/pipeline/pipeline_module*.go internal/pipeline/cache_runtime.go \| grep -v _test.go`; `grep -rn 't.Parallel()' internal/pipeline/*_test.go`; `sed -n 245,268p pipeline_module_phases_test.go` | writers at `:65` (SKIP), `:85` (INVALID), `:87` (MISS), `:123` (Summary), all `os.Stderr`; `cacheDependencies{newStore, stderr io.Writer}`, `productionCacheDependencies()` → `stderr: os.Stderr`, `CACHE_INVALID` at `cache_runtime.go:110`; goroutine grep → rc=1 (0 hits; control `grep -rln "go func" internal/pipeline/*.go` → `metrics.go`); `t.Parallel` → rc=1 (control: `internal/lsp/hover_test.go`, `index_test.go`); warm assertion `"[CACHE] answer: SKIP (cached "` (prefix), invalid `"[CACHE] answer: INVALID, recompiling"` (prefix) |
| V9 | Cold compile writes exactly four `[CACHE]` lines in a fixed order; Summary is last | temp binary `go build -o $d/ailang ./cmd/ailang`; `AILANG_CACHE_DIR=$d/cache ./ailang check --debug-compile answer.ail` twice | cold: `[CACHE] std/option: MISS`, `[CACHE] std/result: MISS`, `[CACHE] answer: MISS`, `[CACHE] Summary: 0 hits, 3 misses (3 modules cached)`; warm: three `SKIP (cached 0s ago)` then `Summary: 3 hits, 0 misses (3 modules cached)` |
| V10 | AC1 baseline green | `go test ./internal/pipeline/ -run 'TestPipelineModulePhases' -count=1` | `ok … 0.792s`, rc=0 |
| V11 | AC5 baseline green; `go build ./...` red at base | `go build ./internal/pipeline/; go vet ./internal/pipeline/`; `go build ./... >/dev/null 2>&1; echo $?` | rc=0, rc=0; `go build ./...` → `cmd/wasm: function main is undeclared in the main package`, rc=1 |
| V12 | No existing test of the helper; no shared capture helper in `internal/testutil`; 8 tests in the phases file | `grep -rn 'TestCapturePipelineStderr\|capturePipelineStderrWith' --include='*.go' .` (rc=1); `grep -rn 'os\.Pipe\|func Capture' internal/testutil/*.go` (rc=1; control: `grep -h '^func [A-Z]' internal/testutil/*.go` → `FindAilangBinary`, `RequireAilangOnPath`, `LiveNetworkStatus`, `RequiresLiveNetwork`, `HangGuard`, `HangGuardContext`); `grep -c '^func Test' internal/pipeline/pipeline_module_phases_test.go` → 8 |
| V13 | gatelint exists with rules R1–R3 (Future Work pointer only) | `ls internal/testutil/gatelint/`; `grep -n 'RuleR' internal/testutil/gatelint/scan.go` | `allowlist.go scan.go gatelint_test.go testdata`; `RuleR1/R2/R3` at `scan.go:18-20` |
| V14 | CI job shape: `go test -v` over all packages on a 4-job matrix with two `macos-latest` entries, 30-minute job timeout, no `concurrency` block, `fail-fast` unset (GitHub default true) | `sed -n 12,40p .github/workflows/build.yml`; `grep -n timeout-minutes …`; `grep -n -A2 concurrency .github/workflows/*.yml`; `grep -n fail-fast .github/workflows/build.yml` | matrix `ubuntu-latest`, `macos-latest`×2 (darwin/amd64, darwin/arm64), `windows-latest`; `timeout-minutes: 30` (job); `go test -v $(go list ./... \| grep -v /scripts \| grep -v /examples/agents)`; concurrency only in `docusaurus-deploy.yml`; `fail-fast` rc=1 (absent) |
| V15 | The test file's only history is the #1071 landing; Windows skip is attributed there to a separate defect | `git log --oneline -- internal/pipeline/pipeline_module_phases_test.go`; `sed -n 201,215p …` | `202358fd3 Mission V1 iter342 … (#1071)` (sole commit); skip comment cites drive-letter cache paths and "V1 mission charter (Windows drive-letter cache paths)" |
| V16 | Related docs exist at the cited paths | `ls design_docs/planned/v0_36_0/m-cache-module-id-encoding.md design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md`; `ls changelogs/ \| grep current` | all four present; `v0.32-current.md` |
| V17 | External premise, NOT verified in-session, not load-bearing | — | "GitHub-hosted `macos-latest` runners have 3 vCPUs" is taken from GitHub's runner documentation, not measured here. The design does not depend on the number; it depends only on the reader being scheduled late, which V6/V7 reproduce on a 12-core laptop by construction |
| V18 | `testutil.HangGuard`/`HangGuardContext`: signatures, behaviour, callers, and no import cycle with `internal/pipeline` (round-2, gpt6-astra) | `sed -n 86,107p internal/testutil/gate.go`; `grep -rn 'HangGuard(' --include='*.go' internal/ cmd/ \| grep -v 'func HangGuard'`; `grep -rl 'HangGuard' --include='*.go' .`; `grep -rl '"github.com/sunholo-data/ailang/internal/testutil"' internal/pipeline/*_test.go`; `go list -deps ./internal/testutil \| grep -c internal/pipeline` (control: `\| grep -c internal/`) | `:88 func HangGuard(t *testing.T, cap time.Duration) time.Duration` — returns `cap` when `t.Deadline()` is unset, else `min(cap, until(deadline)−20s)` floored at 1 s; `:103 func HangGuardContext(t *testing.T, cap time.Duration) (context.Context, context.CancelFunc)` = `context.WithTimeout(Background, HangGuard(...))`. Callers: 9 call lines in 3 files (`eval_harness/reference_solutions_test.go` ×2, `testutil/gate.go` ×1, `testutil/gate_test.go` ×6); 4 files mention the name at all (the controller's briefing said 14 `_test.go` files — measured here as **1** external test file; the discrepancy does not affect the decision). `internal/pipeline` test files importing `testutil`: 3 (`prelude_capability_test.go:10`, `cache_invalidation_test.go:18`, `jwt_test.go:9`). Cycle check: `0` (control: 40 `internal/` deps). **Decision:** not reused — both need `*testing.T`, the helper has none and the seam does not thread it; a fixed 10 s local deadline with a panic diagnostic is used (Design Freeze) |
| V19 | Pipe capacity bound the gated design relies on: `f()`'s writes of the fixed payload never block while the reader is parked | Probe `/tmp/iter353_probe2/pipecap_test.go`: `syscall.Pipe`, `SetNonblock` on the write end, write chunks of 1/512/4096/16384/65536 B until `EAGAIN`, count bytes accepted (`go test -run TestPipeCapacity -v`), macOS 26.6.2 / Darwin 25.6.0 arm64 | `65536` bytes accepted before `EAGAIN` for **every** chunk size, including 1-byte writes — so on this kernel the capacity is 64 KiB from the first byte (the "16 KiB default, growable to 64 KiB" description did not match this kernel; measured, not cited). The CI runner's kernel is **not** measured here — EXTERNAL PREMISE, labelled as V17 is: the design keeps the payload ≤ 4 KiB (actual ~150 B), which is below the smallest pipe buffer any documented Unix ships (4 KiB, PIPE_BUF minimum), so the bound holds without relying on the 64 KiB figure |
| V20 | The gated design itself: 0 loss fixed, deterministic loss under MUT-1, panic-safe teardown, loud bounded wait, race-clean | Probe `/tmp/iter353_probe2/gated_test.go` (the Architecture sketch verbatim, `MUT1=1` env moves `r.Close()` above the hook): `go test -count=1 -race -run 'TestGated\|TestPanicRestores\|TestDrainTimeoutIsLoud' -v`; `MUT1=1 go test -count=1 -race -run TestGated -v`; `for i in 1..5: MUT1=1 go test -count=1 -run TestGated; echo rc` | Fixed: `lost=0/50`, `--- PASS` ×3 (`TestDrainTimeoutIsLoud` in 0.05 s at a 50 ms deadline), `ok`, no `DATA RACE`. MUT-1: `iteration 0: reads=1 got=""`, `lost=50/50`, `--- FAIL`; five repeated runs `rc=1 rc=1 rc=1 rc=1 rc=1`. The `reads=1` under MUT-1 is the closed-fd first `Read` returning immediately — the mechanism the determinism argument predicts |

## Related Documents

- [design_docs/planned/v0_36_0/m-cache-module-id-encoding.md](../v0_36_0/m-cache-module-id-encoding.md) —
  owner of the Windows `t.Skip` at `:201-215` (drive-letter cache path; parked on D-57). Distinct
  defect; this doc does not touch it.
- [design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md](../v0_35_2/m-cachesrc-cognitive-complexity.md) —
  the design behind #1071, whose landing introduced this test and its helper.
- [design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md](../../implemented/v0_35_2/m-compile-cache-unverified-artifacts.md) —
  the compile-cache diagnostics (`[CACHE]` forms, `CACHE_INVALID`) this test freezes.
- [design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md](../../implemented/v0_33_1/m-ci-flake-systemic-fix.md) —
  the earlier CI-flake programme (`internal/testutil` + gatelint). This item is a single-helper
  ordering bug, not a new gate class; a gatelint rule is listed under Future Work only.

**Auto-search results (neural, all < 0.45 — no duplicate/coverage gate hit):**
`m-pkg-cascade-deterministic-first` (0.43), `m-incremental-typecheck` (0.43),
`m-hot-reload-serve-api` (0.43), `m-apple-container-local-eval-sandbox` (0.45),
`m-coordinator-route-authority-recovery` (0.40), `m-budget-scoping-bug-sprint-plan` (0.39).
Read the 0.45 hit: it concerns a local eval sandbox on Apple containers — unrelated.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Queue row: `design_docs/v1-mission.md:569-572` (`m-debugcacheforms-flaky-on-macos-ci`)
- Landing that introduced the test: [#1071](https://github.com/sunholo-data/ailang/pull/1071);
  filing of the flake: [#1085](https://github.com/sunholo-data/ailang/pull/1085)
- Failing CI job with the test in its log: run `34774738497`, job `103770669326`
- Go semantics relied on: `(*os.File).Close` on a pipe read end makes concurrent/pending `Read`s
  return an error (`os: file already closed` / `EBADF`), which terminates `io.Copy` — measured in
  V6 rather than cited

## Future Work

- A gatelint rule (R4) for "pipe read end closed before the drain wait" in `*_test.go`, if a second
  instance ever appears. Today there is one (V5), fixed here.

## Quorum verification log

| round | reviewer | verdict | objection | disposition |
|-------|----------|---------|-----------|-------------|
| 1 | gemini-3-1-pro | REJECT | `defer` teardown left optional in Deferred Decisions: a panicking `f()` leaves `os.Stderr` swapped and the copier leaked for every later test; and a naive `defer` with `return buf.String()` evaluates before `<-done` (data race, AC4) | applied verbatim in r2: named-return `(out string)` `defer` teardown mandated in Design Freeze + Architecture sketch; Deferred-Decisions bullet deleted; MUT-3a + `PanicRestoresStderr` added; probed (V20) |
| 1 | gpt6-astra (ABSENT on budget in the quorum run; re-run alone at a $0.40 cap) | REJECT | The 200 µs sleeping reader does not guarantee MUT-1 is red — a descheduled writer lets the reader drain first; V6/V7 show observed kills, not a deterministic one. `<-done` unbounded although V12 found `HangGuard`/`HangGuardContext` and never examined them | applied verbatim in r2: channel-gated reader + drain-wait hook (nil in production), payload ≤ 4 KiB with the bound stated (V19), MUT-1 = `r.Close()` before the hook → first `Read` on a closed fd (deterministic; V20 5/5); all sleeps/tuning/N-of-N claims removed; bounded wait with a local 10 s deadline, diagnostic and cleanup; `HangGuard` examined in V18 and the non-reuse reasoned |
| 1 | oc-glm-5-2 | PASS | The regression test's injected reader simulates rather than reproduces the natural CI scheduling race | applied verbatim in r2: the "structural invariant proof, not a natural-failure reproduction" paragraph added to Testing Strategy, adapted to the gated design |

---

**Document created**: 2026-09-14
**Last updated**: 2026-09-14 (revision r2 after quorum round 1)

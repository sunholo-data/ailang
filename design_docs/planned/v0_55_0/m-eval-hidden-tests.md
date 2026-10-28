# M-EVAL-HIDDEN-TESTS: Held-Out Tests and a Per-Benchmark Exec Timeout

**Status**: Planned. Implemented on dev, uncommitted, pending human review of D4 and the quorum record
**Target**: v0.55.0
**Priority**: P1 - Medium
**Estimated**: 1.5 days (harness 1 day, first benchmark pack 0.5 day)
**Dependencies**: None
**Routing lane**: eval harness (not the motoko core, not the language). PROGRAM.md's "benchmark
ladder" is fed by benchmarks that discriminate. This doc gives the ladder an oracle strong enough
to discriminate.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Hidden tests run in a fixed order (sorted by input size, then name); the verdict is a pure function of code + test files |
| A2: Replayability | +1 | The banked row records `hidden_passed`/`hidden_total` and the first failing test name, so a failure is replayable from the benchmark directory |
| A3: Effect Legibility | 0 | No language or effect changes |
| A4: Explicit Authority | 0 | Same caps and the same guarded runner as the visible test |
| A5: Bounded Verification | +1 | Each hidden run is bounded by the same guards and the loop stops at the first failure of any kind, so the grade costs at most `hidden_total × exec_timeout` (only a solution that passes every test slowly reaches it) and at most one failing run |
| A6: Safe Concurrency | 0 | Hidden tests run sequentially inside one trial |
| A7: Machines First | +1 | A machine-graded signal replaces a one-sample proxy that agents can satisfy by special-casing the sample |
| A8: Minimal Syntax | 0 | Two optional YAML keys |
| A9: Cost Visibility | +1 | Exec time is now a per-benchmark declared budget, identical for every language, instead of an undeclared 10s/30s split |
| A10: Composability | 0 | Composes with `grading:` modes (each hidden test is graded via `GradeStdout`) |
| A11: Structured Failure | +1 | Hidden failure is a typed field (`hidden_first_fail: "<name>:wrong_output\|timeout\|runtime_error"`), not a free-text stderr scrape |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): fixed order, no sampling of tests
- [x] A3 (Effects): none
- [x] A4 (Authority): no new capability, same runner guards
- [x] A7 (Machines First): yes

## Problem Statement

The headline suites read at or near 100% for current models, in both standard and agent mode, so
they no longer measure the AILANG/Python gap. A spike on 20 ICPC/CCPC 2024 problems (AetherCode
v1_2024, CC-BY-4.0) showed that the cause is partly the **oracle**, not the tasks.

1. **One `expected_stdout` overstates pass rates, and it overstates AILANG more than Python.** We
   re-graded the saved code of every harness pass against the problem's full hidden test set
   (inputs ≤20KB), using Claude Haiku 4.5 in agent mode:

   | Run | Lang | Harness passes | Also pass hidden small tests |
   |---|---|---|---|
   | first run (20 problems × 1) | python | 15 | 10 |
   | first run | ailang | 14 | **3** |
   | confirm (7 problems × 3) | python | 12 | 9 |
   | confirm | ailang | 17 | **2** |

   AILANG solutions match the sample and fail the hidden tests 4–8× more often than Python
   solutions. The current oracle cannot see this.

2. **The visible expected output is shown to the agent.** `PrepareWorkspaceWithSyntax` writes
   `spec.ExpectedOut` into the agent's README (`internal/eval_harness/agent_prompt.go:162`), and the
   task template substitutes `{{EXPECTED_OUTPUT}}` (`agent_prompt.go:431`). Putting a hidden test in
   `expected_stdout` would therefore leak the answer. There is no field that grades without
   showing.

3. **The exec time limit differs by language in agent mode.** AILANG validation runs with
   `10*time.Second` (`agent_validation.go:168`). Python runs with `30*time.Second`
   (`agent_validation.go:142`), as do JS/Go (`:88`) and the workspace probe (`:255`). Standard mode
   uses `--timeout` (default 30s) for both. Any benchmark whose correct AILANG solution needs more
   than 10s is failed in agent mode alone. The `cp_60159` AILANG reference solution needs **8.5s**
   on the evaluator for its largest selected hidden test, which is inside the 10s limit by a margin
   any machine-load spike can erase.

## Goals

**Primary goal:** A benchmark can declare held-out tests that the harness grades but never shows
the model, and an exec time limit that applies identically to every language and mode.

**Success metrics:**
- A benchmark with `hidden_tests_dir` banks `stdout_ok=true` only if the visible test and every
  hidden test pass.
- `hidden_passed`/`hidden_total` are banked on every row of such a benchmark, in both modes.
- No hidden test's stdin or expected output appears in any prompt, README or workspace file (test
  asserts this).
- `exec_timeout` overrides every code-execution limit for that benchmark: the four agent-mode sites (V3, V4, V5) and standard mode's `--timeout` (V11), five in total.
- Benchmarks without the new keys bank byte-identical rows (no new fields, same verdicts).

## High-Impact Decisions

| Decision | Options | Recommendation | Who decides | Change cost |
|---|---|---|---|---|
| D1. Where hidden tests live | inline YAML list · sibling dir of `*.in`/`*.ans` pairs | **Sibling dir** (`benchmarks/hidden_tests/<id>/`): inputs reach 64KB, and inline YAML of that size is unreviewable | agent | low |
| D2. Verdict on partial pass | all-or-nothing · fraction | **All-or-nothing `stdout_ok`**, plus `hidden_passed`/`hidden_total` banked for finer tracking | agent | low |
| D3. Fail-fast | run all · stop at first failure · stop at first timeout | **Stop at the first failure of any kind** (revised after quorum round 1). Tests run smallest-input first, so `hidden_passed` is the passing prefix: still a monotone progress signal, at one failing run of cost | agent | low |
| D4. Global agent-mode AILANG timeout 10s → 30s | keep 10s · unify to 30s | **Unify to 30s**, in a separate commit with its own changelog line, because it can move historical AILANG agent-mode rates | **human** | medium (moves banked comparisons) |
| D5. Which engine grades AILANG | evaluator (today) · VM | **Unchanged (evaluator).** The engine switch belongs to the VM design doc (see Related), and must not be smuggled in here | human (in that doc) | — |

### Design Freeze

- [ ] D4 ratified by a human (until then this sprint ships only the per-benchmark `exec_timeout`,
      which affects only benchmarks that opt in)

## Solution Design

### Overview

Two optional spec keys:

```yaml
hidden_tests_dir: hidden_tests/cp_60159   # relative to the benchmark YAML's directory
exec_timeout: 30                   # seconds, code execution, every language and mode
```

`LoadSpec` reads `<dir>/*.in` and pairs each with `<same-stem>.ans`. A `.in` without its `.ans`, an
`.ans` without its `.in`, an empty directory, or `hidden_tests_dir` combined with `grading: quine` or
`grade_entrypoint` (neither has a stdin/stdout pair to hold out) is a **load error**, so a benchmark
can never pass on zero loaded tests. The
pairs are held in `spec.HiddenTests` (`yaml:"-"`), sorted by input size and then by name.

### Architecture

One helper does the grading for every lane:

```go
// GradeHidden runs the hidden tests after the visible test passed. run executes the
// candidate under a spec copy whose Stdin/ExpectedOut are the hidden test's.
func GradeHidden(spec *BenchmarkSpec, run func(s *BenchmarkSpec) (stdout string, runtimeOk, timedOut bool, err error)) HiddenResult
```

It has two call sites, because there are exactly two lane families:

- **Standard mode**: `RepairRunner.attempt` (`repair.go`), after `stdoutOk` is computed. The closure
  builds a runner for the spec copy through `GetRunnerWithContext(ctx, r.runner.Language(), &cp, "")`,
  the same factory `runSingleBenchmark` uses (`cmd/ailang/eval_benchmark.go:168`).
- **Agent mode**: `validateSolution` (`agent_validation.go`). The current body becomes
  `validateOnce`, and `validateSolution` calls it once for the visible test and then once per hidden
  test with a spec copy. Every language's lane (Python via uv, AILANG, JS/Go via the registry) is
  reused unchanged. `grade_entrypoint` benchmarks are rejected at load time if they also set
  `hidden_tests_dir` (out of scope for the multi-file probe).

Runners already read `Stdin` and `ExpectedOut` from the spec they hold (`runner.go:181,320`;
`agent_validation.go:135`), so a spec copy is all a hidden run needs. No runner changes.

`exec_timeout` is read by one accessor, `spec.ExecTimeoutOr(def time.Duration)`, at every site that
executes candidate code: `agent_validation.go` JS/Go (:88), Python (:142), AILANG (:168), workspace
probe (:255), and `RepairRunner.runSingleAttempt`, which receives `--timeout` (V11). The hidden runs
reuse the same value. The census grep is V12.

### No hidden content reaches the model (repair loop and agent mode)

Standard mode's self-repair prompt is built from the **visible** run:
`FormatRepairPrompt(..., firstResult.Code, firstResult.RunResult.Stderr)` (`repair.go`, V13). The
hidden verdict is stored beside `RunResult` on the attempt (`attemptResult.Hidden`), never in it, so a
hidden failure contributes nothing to the repair prompt. With an empty visible stderr,
`CategorizeErrorCode` matches no rule and returns no hint (V14), so a hidden-only failure does not
trigger a repair turn at all. In agent mode, grading runs after the agent has finished, and the banked
`stdout`/`stderr` are the visible run's. Hidden runs' output is discarded and only counts and
`hidden_first_fail` (a test file name plus a failure kind) are kept. Tests pin both properties
(`TestStandardMode_HiddenTests`, `TestValidateSolution_HiddenTests_Python`).

### Banked fields (additive, `omitempty`)

| Field | Meaning |
|---|---|
| `hidden_total` | number of hidden tests the benchmark declares (0/absent = benchmark has none) |
| `hidden_passed` | how many passed; 0 when the visible test failed (hidden tests are not run) |
| `hidden_first_fail` | `<stem>:wrong_output\|timeout\|runtime_error`, empty when all passed |

`error_category` follows the first hidden failure: `logic_error` for wrong output,
`runtime_error` for a crash or timeout. These are the same categories a visible failure of the same
kind banks today, so aggregates do not need a new bucket.

### Implementation Plan

1. Spec keys, loader, validation errors, `ExecTimeoutOr` (spec.go plus a new hidden_tests.go).
2. `GradeHidden` plus unit tests (all-pass, wrong output, timeout fail-fast, ordering).
3. Wire the standard and agent call sites, and the metrics fields.
4. A leak test: build the agent workspace and prompt for a hidden-test benchmark, then assert that
   no hidden stdin or expected output appears in either.
5. First benchmark pack (below), each validated against its reference solution(s).

### Files to Modify/Create

- `internal/eval_harness/spec.go`: two keys, `HiddenTests`, `ExecTimeoutOr` (~40 LOC)
- `internal/eval_harness/hidden_tests.go`: new; loader plus `GradeHidden` (~150 LOC)
- `internal/eval_harness/hidden_tests_test.go`: new (~200 LOC)
- `internal/eval_harness/repair.go`: hidden grading after the visible verdict (~30 LOC)
- `internal/eval_harness/agent_validation.go`: `validateOnce` split, hidden loop, timeouts (~50 LOC)
- `internal/eval_harness/metrics.go`: three fields (~10 LOC)
- `benchmarks/cp_*.yml`, `benchmarks/hidden_tests/cp_*/`: the first pack

## First Benchmark Pack (AetherCode v1_2024, CC-BY-4.0)

Each problem was selected because the spike showed a **language delta that survived 3+ trials**, not
a harness relic. Hidden tests are the dataset's own tests with inputs ≤64KB, minus the example already shown in the prompt. Each problem's data is
under 100KB, about 280KB for all four.

| Benchmark | Gap it tracks | Hidden tests | Python reference | AILANG reference |
|---|---|---|---|---|
| `cp_60166` | no BigInt: the natural solution compares exact huge products | 12 | 5/5 trials pass, max 0.06s | **written: example + 12/12 hidden, max 0.3s evaluator.** Saturating product + exact DP; harder without BigInt, not impossible |
| `cp_60159` | no cheap set: cycle detection needs a visited set, or the insight to use Floyd | 17 | 5/6 trials pass, max 0.06s | **written: example + 17/17 hidden, max 8.5s evaluator / 0.2s VM** (Floyd's algorithm, no set) |
| `cp_60164` | no cheap accumulate: the fast path is a mutable 2D difference array | 8 | 0.12s (difference array); brute force 9.6s or TLE | none yet. **Tier `experimental`** until one exists |
| `cp_60153` | algorithm downgrade (flood fill → wrong heuristic) | 22 | 1 trial passes (0.7s); 3 confirm trials 0/23 | none yet. **Tier `experimental`** |

`exec_timeout: 30` for all four. That leaves 3.5× headroom over the slowest AILANG reference on the
evaluator (8.5s). On `cp_60164` it sits between the two brute-force Python solutions the spike
produced: confirm-trial t2 (9.6s on the largest hidden test) passes 9/9, while confirm-trial t1
(49s) times out and passes 5/8 (V16, the negative control). The limit therefore discriminates a
constant-factor-heavy O(n³) from a slower one, not only O(n³) from the O(n²) difference array.
Read `cp_60164` verdicts with that in mind.

**Tracking progress.** A language fix shows up as movement on a specific benchmark:
- BigInt or checked arithmetic → `cp_60166`
- a cheap set/map → `cp_60159`, `cp_60153`
- int accumulate or mutable arrays → `cp_60164`
- the VM becoming the grading engine → the `exec_timeout` headroom grows (0.2s vs 8.5s on `cp_60159`)

Use `--trials 3` and `ailang eval-paired` on before/after, never single trials. The spike showed
single trials flip on these problems.

## Success Criteria

- [ ] Loader errors: unpaired file, empty dir, `hidden_tests_dir` together with `grade_entrypoint` or `grading: quine`
- [ ] `GradeHidden` unit tests: all-pass, one wrong, timeout fail-fast, deterministic order
- [ ] Standard and agent paths both bank `hidden_*` fields (one test each)
- [ ] Leak test passes (no hidden content in the prompt or workspace)
- [ ] A benchmark without the keys banks byte-identical rows (golden test)
- [ ] Each pack benchmark: its reference solutions pass via `ailang eval-benchmark`-equivalent grading
- [ ] Changelog fragment; benchmarks/CURATION.md notes the tier choice

## Testing Strategy

Unit tests use a fake `run` closure for `GradeHidden`, plus real `PythonRunner` round-trips on a
two-test fixture in a temp dir. The leak test greps the generated README and prompt strings for
every hidden `.in`/`.ans` body.

## Deferred Decisions

- Whether to also re-grade old banked rows of benchmarks that later gain hidden tests (no: the
  row's verdict is what that run's oracle said; new rows only).
- Per-test timeouts (one budget per benchmark is enough for now).

## Non-Goals

- Changing the AILANG execution engine (see D5).
- Property-based or generated tests (that is M-ORACLE-ADEQUACY's scope; this doc is the minimal
  held-out-test mechanism that doc's "visible vs held-out" diagnosis calls for).
- Interactive or special-checker problems (testlib checkers): standard token compare only.

## Timeline

Day 1: keys, loader, `GradeHidden`, both call sites, tests. Day 2 (half): pack, reference runs,
Haiku smoke in both modes.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Hidden runs multiply grading time | Inputs ≤64KB; stop at the first failure; at most 22 tests per benchmark. Measured: the slowest passing reference (cp_60159 AILANG, evaluator) grades all 17 in 14.2s |
| Machine load flips borderline perf verdicts | `exec_timeout` chosen at ≥3× the slowest reference; `--trials 3` |
| An agent reads `benchmarks/hidden_tests/` from disk | The agent workspace is a temp dir with no repo path; the leak test checks the workspace contents |
| Dataset licence | CC-BY-4.0: attribution in `benchmarks/hidden_tests/AETHERCODE.md` and in each YAML's description |

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | Agent README contains `spec.ExpectedOut` | `agent_prompt.go:162` (`## Expected Output` section of the README fmt.Sprintf) |
| V2 | Agent task template substitutes the expected output | `agent_prompt.go:431` `{{EXPECTED_OUTPUT}}` |
| V3 | AILANG agent validation uses 10s | `agent_validation.go:168` `runner.Run(solutionCode, 10*time.Second)` |
| V4 | Python agent validation uses 30s | `agent_validation.go:142` |
| V5 | JS/Go validation and workspace probe use 30s | `agent_validation.go:88`, `:255` |
| V6 | No existing hidden/multi-test field | `grep -rniE "hidden_tests\|extra_tests\|test_cases\|HiddenTests" internal/eval_harness/*.go` → no non-test hits (2026-10-28) |
| V7 | Runners read Stdin from the spec they hold | `runner.go:181` (Python), `runner.go:320` (AILANG), `agent_validation.go:135` |
| V8 | Over-acceptance table | Re-grade of saved code: scratchpad `grade_ha3.json`, `grade_c.json` (`harness_pass` vs `small_ok`), 2026-10-28 |
| V9 | AILANG references pass | `alltests.py` runs on the selected tests: cp_60166 13/13 and cp_60159 18/18 of the ≤64KB tests, the example included (max 0.3s and 8.5s on the evaluator; 0.2s for cp_60159 under `--bytecode`) |
| V10 | Standard mode builds runners via `GetRunnerWithContext` | `cmd/ailang/eval_benchmark.go:168` |
| V11 | Standard mode's exec limit is `--timeout`, default 30s | `ailang eval-suite --help`: "-timeout duration  Timeout for code execution (default 30s)"; it flows `runSingleBenchmark(..., timeout, ...)` → `NewRepairRunner(..., timeout, ...)` → `r.runner.Run(code, r.timeout)` (`cmd/ailang/eval_benchmark.go:243`, `repair.go:253`) |
| V12 | No other exec-limit site in the grading path | `grep -n "time.Second" internal/eval_harness/agent_validation.go internal/eval_harness/repair.go` → only the four agent sites above (repair.go has none of its own) |
| V13 | Repair prompt uses the visible run's stderr only | `repair.go`: `repairStderr = firstResult.RunResult.Stderr` and `FormatRepairPrompt(errCode, hint, r.spec.ID, lang, firstResult.Code, repairStderr)` |
| V14 | Empty stderr yields no repair hint | `errors.go` `CategorizeErrorCode` returns `("", nil)` when no rule matches; the AILANG code-pattern rules (WRONG_LANG/IMPERATIVE) look at code, not at hidden output |
| V15 | Implementation verified, including mutation | `go test ./internal/eval_harness/ -run 'Hidden\|ExecTimeout'` passes; disabling the agent hidden loop fails the Python end-to-end test (3 failures); disabling the standard-mode hidden block fails `TestStandardMode_HiddenTests` (5 failures); full package passes |
| V17 | A visible timeout banks `runtime_error` today, so the hidden mapping matches | `memlimit.go` `runGuarded` timeout branch returns `CompileOk: true, RuntimeOk: false, TimedOut: true`; `metrics.go` `CategorizeError` maps `!runtimeOk` to `runtime_error`; agent rows go through `CategorizeRunError`, which adds only the memory-kill promotion. `ErrorCategoryTimeout` ("timeout") is the agent wall-clock/deadline bucket, not an exec timeout, and is untouched |
| V18 | Layout survives git | `.gitignore:63` ignores every `data/` dir and `.gitignore:24` ignores `*.out`, so the first layout (`benchmarks/data/`, `.out`) would have committed inputs without answers. Hence `benchmarks/hidden_tests/` and `.ans`; `git check-ignore` is clean on both. A future clone that lost files would fail `TestAllBenchmarksHaveTierAndTags` (LoadSpec rejects an unpaired `.in`) |
| V16 | Pack oracles separate right from wrong | Through the real lanes: all 6 references pass every hidden test. The spike's failing solutions, all of which passed the visible test, fail: cp_60166 AILANG 1/12 (wrong output), cp_60159 AILANG 10/17 (timeout), cp_60164 brute-force Python 5/8 (timeout) |

## Quorum

Trigger 1 (freeze item D4) and trigger 3 (banked-data schema: three new row fields, and
`stdout_ok` meaning more for opted-in benchmarks) both fire.

**Round 1 (2026-10-28, $0.17): BLOCKED.** glm-5.3 and kimi-k3 rejected and gemini-4-argon rejected;
gpt6-1-sol was unreachable (N−1); the controller passed. All objections were accepted:
- *glm*: the A5 bound contradicted D3, the timeout census was miscounted, and standard `--timeout`
  had no log row. **Fixed:** D3 now stops at any failure, A5 states the real bound, and V11/V12 hold
  the full census.
- *kimi / gemini*: the result was not recorded (it now is). *gemini*: could a hidden failure leak
  through the standard-mode repair prompt? **Fixed by construction and pinned by tests:** see "No
  hidden content reaches the model" and V13/V14.

**Round 2 (2026-10-28, $0.16): BLOCKED.** glm-5.3 rejected; kimi-k3 rejected; gemini-4-argon
passed; gpt6-1-sol was unreachable; the controller passed. Both rejections were correct and are fixed
in this text, with no design change:
- *glm*: the two `cp_60164` timing statements described two different Python solutions. Now stated
  separately (pack section).
- *kimi*: the visible-timeout category was asserted, not logged. Now V17.
- *gemini* (pass, with a note): a code-pattern repair hint (WRONG_LANG/IMPERATIVE) could still open a
  repair turn after a hidden-only failure. That turn would carry the visible stderr only, so nothing is
  leaked, but it would be a repair the visible test did not ask for. Accepted as a known edge: a
  solution that passed the visible test matching those patterns is implausible.

The re-quorum-once guardrail is spent. The doc goes to the human with these rounds recorded, and
does not get a third round.

The schema change (trigger 3) is additive and `omitempty`. Rows of benchmarks without the keys are
unchanged, and the human ratification asked of this doc is D4 only.

## Related Documents

- [m-oracle-adequacy](../v1_1_0/m-oracle-adequacy.md) (0.44): the broader programme (convergence
  oracles, evidence bundles). This doc is its minimal held-out-test mechanism; no overlap in scope.
- [m-eval-standard-mode-input-files-gap-sprint-plan](../m-eval-standard-mode-input-files-gap-sprint-plan.md)
  (0.43): standard vs agent parity for `input_files`. Same class of lane asymmetry as Problem 3.
- [m-eval-trust-signals](../v1_1_0/m-eval-trust-signals.md) (0.43).
- Language-gap docs written from the same spike (v0_55_0): VM coverage for eval workloads,
  incremental collections, integer overflow and BigInt.
- Implemented test-block docs (0.44–0.47) concern language-level `test` blocks and are unrelated.

## References

- AetherCode: https://huggingface.co/datasets/m-a-p/AetherCode (CC-BY-4.0)
- [Design Axioms](/docs/references/axioms)

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28

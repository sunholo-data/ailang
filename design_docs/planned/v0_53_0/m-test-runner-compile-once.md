# M-TEST-RUNNER-COMPILE-ONCE: `ailang test` compiles a test file once, not once per named test

**Status**: Planned. Awaiting human ratification of D1 and D3; the quorum blocked twice, and both rounds were closed by measurement (see Quorum history).
**Target**: v0.53.0
**Priority**: P1. Named tests are the documented test form, and on real modules the per-test compile dominates every test run.
**Estimated**: 2–3 days (Phase 1); Phase 2 is a separate estimate
**Dependencies**: None. This complements the #1328 inline-test memoization (`design_docs/planned/ailang-core-triage/inline-test-per-case-recompile-1328.md`), which covers a different code path.
**Issue**: [#1328](https://github.com/sunholo-data/ailang/issues/1328) (same class); stapledons-godot report `inbox_1791287299418_f86a7ce4` (2026-10-06)
**Quorum history**:
- **Round 1:** BLOCKED on three objections: a silent fallback, unmeasured per-test runtime, and an unverified multi-entry bytecode image. All were accepted and closed with D1 (loud fallback) and V14.
- **Round 2:** BLOCKED on three objections: the assert-sentinel path was unexercised, failing bodies were never run through a batch, and the spike was not reproducible. All were accepted and closed with V15: a committed, reproducible premise test covering failing and assert bodies.
- The re-quorum-once guardrail is spent, so this goes to a human for ratification of D1 and D3, with the round-2 closure in V15. In each round `gpt6-1-sol` was ABSENT (unreachable), giving N−1.

**Quorum trigger**: #1 fires (this doc has design-freeze items D1 and D3 for a human to ratify). Triggers #2–#4 do not: no shared machinery is overridden, there is no cost or KPI surface, and every premise is in-repo.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same bodies, same results, same order. Each test still gets a fresh evaluator or VM (V9). |
| A2: Replayability | 0 | `--seed` and the report format are unchanged |
| A3: Effect Legibility | 0 | Bodies are compiled `pure` exactly as the VM path already does (V5), and effects stay refused (V7) |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | A test run costs one compile per file instead of N. In the spike, protocol_test went from 216 s to 10.7 s: an 8.5 s compile plus 2.1 s running all 29 bodies on fresh evaluators (V14). |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Agents run `ailang test` in their inner loop. motoko's coverage lane times out on three files today (#1328), and stapledons waits 10 minutes per package run. |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | The per-test duration finally measures the test, not a compile (today every protocol_test row reads about 7.3 s, V2) |
| A10: Composability | 0 | The VM and evaluator engines share one compiled result |
| A11: Structured Failure | 0 | Per-test failure messages are preserved by D1 |
| A12: System Boundary | 0 | None |

**Net Score: +3** → **Decision: Move forward.** There is no −1 on A1, A3, A4 or A7.

## Problem Statement

`ailang test` compiles the whole test module once **per named test**.

**Current State** (measured at dev 40744990f / 07e1a89bc, M-series Mac):
- stapledons-godot `sim/protocol_test.ail` has 638 lines and 29 `test "…" { … }` blocks. `ailang test` takes **3 min 36 s**; every test takes 7.2–7.9 s (V2).
- A warm `check` of the same module takes 7.1–7.3 s (V3), so each test costs one full compile.
- On a small module (`rng_test.ail`) a test takes about 52 ms. The cost scales with module size × test count, so it is invisible on small fixtures and dominant on real ones.
- stapledons' `ailang test --package .` (289 tests) takes about 10 minutes, and protocol_test alone is a third of that.

**Mechanism** (V1): `Runner.runNamedTest` → `Executor.EvaluateNamedTestBodyExprs` → `runNamedTestPipeline`.
- For each test it writes `stripNonPureFunctions(source)` plus that one test's folded body to a private temp file, then runs the full pipeline with `TransientRoot: true`.
- A transient root is never cached (V6), and the source differs per body, so neither the compile cache nor the 07e1a89bc cap raise can help. Dependencies come from the cache; the test module itself is recompiled N times.
- Under `--bytecode` the body is compiled a second way (`evalNamedTestBodyOnVM`, V5), and a VM fallback compiles a third time on the evaluator path.

**Impact:** every AILANG project with named tests on non-trivial modules: stapledons-godot, motoko (#1328: three files time out at 600 s), and the eval and mission loops that run `ailang test` as a gate.

## Goals

**Primary goal:** one pipeline compile per test file, shared by both engines, for all of its named tests, with the same results and failure messages as today's.

What "the same" means here (V15):
- the same pass/fail per test;
- the same `expected true, got false` and assert/check messages;
- the same runtime-error text, once the per-compile temp directory is masked (that directory is random on every run today too).

The one deliberate difference is D5.

**Success metrics:**
- protocol_test: ≤ 15 s end to end, down from 216 s. Measured 10.7 s in the end-to-end spike (V14): compile, then each body on its own fresh harness evaluator.
- A regression test counts `pipeline.Run` calls: one for an N-named-test file when all bodies compile, not N
- The same pass/fail result, the same per-test failure text, and the same test order on both engines across `internal/testing` and the 28-file corpus in V8
- A file with one ill-typed body still reports that body as the only failure (V7)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: what happens when the batched compile fails | Today one ill-typed body fails only its own test (V7). A batched compile fails as a whole, so isolation must be rebuilt. | human | design | med |
| D2: synthetic entry form | Must type-check exactly where today's block form does | agent (recommendation fixed below) | design | low |
| D3: scope — properties (`forall`) in this sprint or a follow-up | That path compiles once per **generated case** (100 per property, plus shrinking: V10). It is the larger multiplier, but it uses a different synthesis path. | human | design | med |
| D4: evaluator isolation per test | A shared evaluator would leak state between tests | agent: a fresh evaluator per test, as today | design | low |
| D5: location of a runtime error raised **inside a test body** | Today it names a line in the synthesized temp file (for example `…/ailang-namedtest-NNN/mixed.ail:8:7`). In the batch that line moves (`:34:7`). Neither line exists in the user's file. | agent. Recommended: map positions in the synthesized tail back to the test's own source location. The minimum is to accept the moved temp-file line. | compile | low |

### Design Freeze

- [ ] **D1.** Recommended: **compile all bodies as one batch; if that compile fails, say so, then run today's per-body path for every test in the file.**
  - The fallback is **never silent**. The run prints one notice in the same position and style as today's bytecode summary line:
    `→ named tests: batched compile failed (<first error, one line>); compiling each test separately`
  - The JSON report carries `named_test_batch: "failed"` along with the reason, and an M0 test asserts both.
  - Per-test failure messages stay as they are now. The run costs N+1 compiles only for a file that has a compile error, which is red anyway, and the notice explains why it was slow.
  - A batch failure on a file where **every** body then compiles alone is a harness bug, not a user error. The notice says so ("all bodies compile individually — please report"), so a batch-builder defect cannot hide behind the fallback.
  - Alternative (rejected for now): map the error span back to the failing body, drop that body, and retry. That is cheaper on red files, but it needs span attribution through the printer round-trip, which is already a known source of bugs (`ai-deadline-and-named-test-roundtrip.md`).
- [ ] **D3.** Recommended: **a separate Phase 2 with its own sprint.** Phase 1 is a contained change to `EvaluateNamedTestBodyExprs`. Phase 2 changes how generated values reach the property: as arguments to a compiled function instead of spliced into source.

## Solution Design

### Overview

Before running any named test, the executor builds **one** source: the stripped module followed by one synthetic entry per named test. It compiles that once per engine and runs each entry on a fresh evaluator, or a fresh VM under `--bytecode`.

```
<stripNonPureFunctions(source)>
pure func __namedtest_0() -> bool { <PrintAILANGSource(folded body 0)> }
pure func __namedtest_1() -> int  { <assert-sentinel body 1> }      -- #590 form
...
```

### Architecture

- **`Executor.prepareNamedTests(cases []TestCase)`** (new, `internal/testing/named_batch.go`):
  - folds every body (`FoldTestBody`, which also records each body's assert `checks` for sentinel decoding);
  - builds the batched source;
  - runs `runNamedTestPipeline` once and keeps the `pipeline.Result`;
  - under `--bytecode`, also compiles the bytecode image once (`runner.CompileBytecodeFromResult`).
- **`EvaluateNamedTestBodyExprs`** looks up the prepared entry by test index.
  - Evaluator: `newHarnessEvaluator()` → `EvalCoreProgram(core)` to bind declarations, then call `__namedtest_k` (`CallValueN` on the bound function value, as `harnessBridge` already does, V9) → decode the sentinel when `checks` is non-empty.
  - VM: `runner.FindEntryProto(img, "__namedtest_k")` on a fresh `vm.NewVM(img)`. The `EvalOnly` and fallback rules are unchanged, per body.
  - A VM error falls back, as today, to the evaluator, but now to the **same batched compile's** `__namedtest_k`, with no recompile. That keeps today's user-visible message, which under `--bytecode` is already the evaluator's after a fallback (V15).
- **Runner:** `Runner.Run` calls `prepareNamedTests` once before the named-test loop. A batch failure (D1) is recorded on the suite result and reported, both as a human notice and in JSON, and each test then takes today's path unchanged.

### Implementation Plan

**Phase 1 — named tests (this sprint)**
- [ ] M0: regression tests first.
  - Count `pipeline.Run` calls through a seam on the executor: three named tests → 1 call, rather than the 3 today.
  - Isolation test: one ill-typed body plus two good ones gives 1 failure with today's message text.
  - The D1 notice and the JSON `named_test_batch` field are present when the batch fails, and absent when it succeeds.
  - Assert-sentinel bodies (#590) inside a batch decode correctly; already shown by V15, and M0 turns it into a gate.
  - Promote `named_batch_premise_test.go` into the real tests and delete it; it is a measurement, not a regression gate.
- [ ] M1: batch builder, evaluator path, and D1 fallback
- [ ] M2: `--bytecode` path on the shared image. Fallback per body to the evaluator entry from the same compile; there is no third compile.
- [ ] M3: parity. `internal/testing`, `make test`, the V8 corpus on both engines, and protocol_test timing recorded in the doc

**Phase 2 — properties (separate sprint, gated on D3)**
- Compile `pure func __prop_k(<binders>) -> bool { <property body> }` once per file. Call it with the generated `eval.Value`s as arguments, which also covers shrinking. That retires the per-case `EvaluateExpression` source synthesis (V10) and its value-splice round-trip.

### Files to Modify/Create

- `internal/testing/named_batch.go` (new, ~150 LOC): batch build, prepare, and entry lookup
- `internal/testing/executor.go` (~40 LOC changed): `EvaluateNamedTestBodyExprs` uses the prepared batch
- `internal/testing/bytecode_engine.go` (~40 LOC changed): VM path on the shared image
- `internal/testing/runner.go` (~15 LOC): prepare before the named-test loop
- `internal/testing/named_batch_test.go` (new, ~200 LOC): call count, isolation, sentinel, and engine parity

## Examples

### Example 1: a large test file

```
$ ailang test sim/protocol_test.ail      # before
  ✓ v2 hello round trip (7.285s)
  ... 29 tests: 29 passed (3m36s)
$ ailang test sim/protocol_test.ail      # after (projected from V4)
  ✓ v2 hello round trip (3ms)
  ... 29 tests: 29 passed (~11s; measured in V14)
```

### Example 2: one ill-typed body (unchanged output)

```
  ✓ good one
  ✗ ill typed
      pipeline error: type error in …/iso (decl 1): type unification failed …
  ✓ good two
```
The batch fails, so D1 prints `→ named tests: batched compile failed (…); compiling each test separately` and falls back to per-body compiles. The per-test lines are identical to today's except for durations.

## Success Criteria

- [ ] The call-count test passes: 1 compile for an all-good N-test file, and N+1 for a file with an ill-typed body (D1)
- [ ] D1 fallback is visible: human notice plus JSON field (tested)
- [ ] Isolation test: only the ill-typed body fails, with today's message
- [ ] Assert-sentinel bodies (#590) decode to the same "which assertion" message (`named_test_assert_test.go` stays green unmodified)
- [ ] `engine_parity_test.go` stays green unmodified; `--strict-bytecode` still fails only bodies that cannot run on the VM
- [ ] protocol_test ≤ 15 s; stapledons package run re-measured and recorded
- [ ] All tests pass; changelog fragment written

## Testing Strategy

Unit tests in `internal/testing` with a call-counting seam around `runNamedTestPipeline`. The existing suites must stay green unmodified:
- `named_test_test.go`
- `named_test_assert_test.go`
- `named_test_env_test.go`
- `named_test_tempfile_test.go`, which is #1502: the private temp dir is still the only place the batched source is written
- `engine_parity_test.go`

Then compare the V8 corpus on both engines before and after: the same pass/fail per test name.

## Conflict Surface

The change touches `internal/testing` (the test harness) and the source handed to the pipeline. No parser, type checker or codegen changes.

1. **What position does this extend?** The synthetic source that the harness feeds the pipeline. It now holds N entries instead of one trailing block.
2. **What else lives there?**
   - **User declarations.** `__namedtest_k` could collide with a user function. The `__` prefix is already the harness's (`namedTestEntry = "__namedtest_entry"`, V5). The builder must still refuse a module that declares a `__namedtest_` name (grep: no such name in std or examples, V11).
   - **Monomorphization.** One module now holds every body's call sites, so the per-module specialization cap (512) is shared across bodies. Hitting it **skips** specialization and the code stays polymorphic and correct; it is not an error (V12). It can change performance, but not results.
3. **Disambiguation:** none needed; these are plain top-level declarations.
4. **Programs that must still work:**
   - `internal/testing/testdata` named-test fixtures
   - `examples/` files with `test "` blocks (V8)
   - stapledons: the 23 `sim/` and `sim/tools/` test files in V8, plus protocol_test's 29 bodies (V4)
   - the `iso.ail` isolation repro (V7)
   - the polymorphic `let` body (V7)
5. **Deliberate changes** (V15):
   - Per-test durations no longer include a compile.
   - A runtime error raised inside a test body reports a different line in the synthesized temp file (D5).
   - The premise test found no other difference: 7 mixed-outcome bodies, and protocol_test's 29 bodies on both engines.

## Deferred Decisions

- The exact seam for counting compiles (agent's choice)
- Whether `prepareNamedTests` runs lazily on the first named test or eagerly in `Runner.Run` (agent's choice)

## Non-Goals

- The inline-test (`tests [...]`) memoization from the #1328 triage doc. It is a different path; land it separately or in the same release.
- Printer round-trip bugs (integral floats, brace-bearing strings in `stripNonPureFunctions`): `ai-deadline-and-named-test-roundtrip.md` and `source-strip-string-brace-skip-ranges.md`. Batching neither fixes nor worsens them; every body still round-trips through `PrintAILANGSource`.
- Caching the batched compile across runs: the root stays transient.

## Timeline

- Day 1: M0 and M1
- Day 2: M2 and M3
- Day 3: buffer, stapledons re-measure, changelog

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| A body that compiles alone fails inside the batch, for example through cross-body inference interaction | Each body is its own top-level function with no shared bindings. D1's fallback catches any residual case, so the batch can only be slower, never wrong. |
| Mono cap shared across bodies | Correctness is unaffected (V12). Record `--debug-compile` skip counts on protocol_test in M3. |
| Assert-sentinel decoding misindexed | Keep `checks` per entry index; the existing assert tests run unmodified |

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Named tests compile per body as a transient root | Read `internal/testing/executor.go` `EvaluateNamedTestBodyExprs` (builds `baseSource + "{ body }"` and calls `runNamedTestPipeline`) and `runNamedTestPipeline` (`TransientRoot: true`, temp dir per call) |
| V2 | Every protocol_test test costs ~7.3 s | `ailang test protocol_test.ail` at 07e1a89bc: 29 rows of 7.17–7.87 s, 3m36s total; rng_test rows ~52 ms |
| V3 | One compile of protocol_test ≈ 7.1–7.3 s, and `check` does not type-check test bodies | `check --relax-modules protocol_test.ail` warm deps: 7.27–7.39 s, and 7.10 s in the spike. `check iso.ail` passes with an ill-typed body that `test` rejects (V7). |
| V4 | One batched compile of all 29 bodies ≈ one compile (see V14 for the end-to-end run) | Spike: `stripNonPureFunctions` plus 29 `pure func __namedtest_k` built with the real `FoldTestBody` and `PrintAILANGSource` → `check` of the batch took 8.65 s on a cold cache (vs 7.10 s module-only, warm) and compiled clean |
| V5 | The VM path already uses the `pure func __namedtest_entry() -> bool\|int` form | `internal/testing/bytecode_engine.go` `evalNamedTestBodyOnVM`, and `const namedTestEntry = "__namedtest_entry"` |
| V6 | Transient roots are not cached | `runNamedTestPipeline` comment: "TransientRoot tells the pipeline … no cache entry" |
| V7 | Isolation today: one ill-typed body fails alone; effects are refused on both paths; polymorphic `let` works on both | `iso.ail`: good ✓ / ill-typed ✗ / good ✓. `eff.ail`: `println` body fails "effect checking failed" on both engines, and `let id = \x. x` passes on both. |
| V8 | The `pure func` form is accepted wherever the block form is | `ailang test --bytecode` over 28 files (std, examples, stapledons sim and tools tests, minus protocol_test): **258 bodies ran on the VM, 0 fell back** |
| V9 | An evaluator can call a bound function value by name | `harnessBridge.CallEvalFunc` → `evaluator.Env().Get(name)` + `CallValueN` (`bytecode_engine.go`) |
| V10 | Properties compile once per generated case | `runner.go`: `EvaluateExpression(boundExpr)` inside the `for testNum := 0; testNum < numTests` loop (numTests = 100) and in the shrink loop. `EvaluateExpression` calls `pipeline.Run` (`executor.go`). |
| V11 | No user code declares `__namedtest_` names | `grep -rn "__namedtest_" std examples` finds no declarations (the harness constant only) |
| V12 | Hitting the mono module cap skips specialization; it does not fail | `internal/pipeline/specialize_lambda.go`: `TotalCount >= MaxPerModule` → append to `Skipped`, `return nil, nil` |
| V14 | End to end, one compile serves every body on both engines, within the time budget | Spike, an in-package temporary Go test since removed, built the protocol_test batch with the real `stripNonPureFunctions`, `FoldTestBody` and `PrintAILANGSource`, then ran `runNamedTestPipeline` once: **8.55 s**. `runner.CompileBytecodeFromResult` ran once: **33 ms**. For each of the 29 entries it took a fresh `newHarnessEvaluator()`, looked up `<root>.__namedtest_k` and called `CallValueN`: **2.11 s total**, max 0.84 s. `runner.FindEntryProto(img, "__namedtest_k")` found every entry in the one shared image, and a fresh `vm.NewVM` run took **0.11 s total**. All 29 returned `true` on both engines, with equal values. Lowering keeps every top-level function (no entry-rooted pruning), so all 29 protos are present. No body in this file uses `assert`, so the sentinel path is left to M0. |
| V15 | Outcomes match today's path, failing bodies included, and the measurement is reproducible | Committed opt-in test `internal/testing/named_batch_premise_test.go` (`TestNamedBatchPremise`). It runs every named test through today's `EvaluateNamedTestBodyExprs`, then through one batched compile on both engines, and logs any difference with the temp dir masked.<br>**Fixture `testdata/named_batch/mixed.ail`** (plain pass, plain false, two-assert pass, second-assert fail, runtime error in a module function, polymorphic `let`, runtime error in the body):<br>• evaluator: identical for all except the body-located runtime error, whose temp-file line moves `8:7`→`34:7` (D5);<br>• raw VM differs on the two runtime errors (`vm: RT001 … op DIV`), which today's `--bytecode` path already turns into an evaluator fallback with the evaluator message, and the design keeps that rule.<br>**protocol_test** (`AILANG_NAMED_BATCH_PREMISE=<path>`): today 3m27s; batched compile plus image 9.08 s, run on both engines 2.35 s; **0 differences in 29**.<br>The assert-sentinel path is now exercised: `assert second fails` decodes to the same `assertion 2 failed: \`assert (double(3) == 7)\` (at …:10:53)` from a batch, because `checks` and source positions come from the original AST per body, not from the synthesized file. |
| V13 | No existing test pins named-test type-error isolation | `grep -rn "type error\|ill-typed\|TC_" internal/testing/named_test*_test.go` → empty. M0 adds one. |

## Related Documents

- `design_docs/planned/ailang-core-triage/inline-test-per-case-recompile-1328.md`: inline tests (2 compiles per case); its addendum records this report
- `design_docs/planned/ailang-core-triage/ai-deadline-and-named-test-roundtrip.md`: printer round-trip defects in the same synthesis path (non-goal here)
- `design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md`: the cache limits raised in 07e1a89bc
- `design_docs/planned/m-package-test-discovery.md`: package-mode discovery (no overlap)

## References

- [Design Axioms](/docs/references/axioms)
- #1328; stapledons-godot `inbox_1791287299418_f86a7ce4`

## Future Work

- Phase 2: properties compiled once, with generated values passed as arguments (D3)
- Elaborate test bodies from the AST instead of printing and re-parsing them, which removes the printer round-trip class of bugs entirely
- Content-keyed caching of the batched test module across runs

---

**Document created**: 2026-10-06
**Last updated**: 2026-10-06

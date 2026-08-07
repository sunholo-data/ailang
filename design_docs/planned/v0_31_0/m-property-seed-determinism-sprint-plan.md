# Sprint Plan: M-PROPERTY-TEST-TRUST (#547 + #535)

**Design doc**: [m-property-seed-determinism.md](m-property-seed-determinism.md)
**Sprint ID**: `M-PROPERTY-TEST-TRUST`
**Branch**: `sprint/m-property-seed-determinism`
**Worktree**: `/tmp/wt-m-property-seed-determinism`
**Base commit**: `386cf6d15` (= `origin/dev` at worktree creation)
**Planner**: claude-opus-5, mission iteration 126, 2026-07-31
**Executor lane**: codex (sandboxed)
**Risk**: medium (M1) / high (M2) / medium (M3)

---

## 0. Verification-first summary — what the planner measured

Every claim below was re-measured first-hand in the main checkout at `01c36db8d`
(`internal/` is byte-identical to the worktree base `386cf6d15`; `git diff 386cf6d15 01c36db8d
-- internal/testing/` is empty). Nothing here is inherited.

### 0.1 CONFIRMED — every controller premise held

| # | Premise | Method | Result |
|---|---|---|---|
| P1 | `newRNG` substitutes `time.Now().UnixNano()` when `seed == 0` | read `internal/testing/runner.go:784-790` | **CONFIRMED** verbatim |
| P2 | Three live call sites at `runner.go:261, 386, 505`, all `newRNG(DefaultConfig().Seed)`, and `DefaultConfig().Seed == 0` | read all three + `generator.go:27-36` | **CONFIRMED** |
| P3 | `runEnsuresProperty` is `runner.go:322-432`; the loop is `388-427`; no `requires` evaluation anywhere | full read | **CONFIRMED** |
| P4 | `runRequiresProperty` (`runner.go:537-547`) SKIPS the identical condition | full read | **CONFIRMED**; the comment literally says such inputs "aren't a function bug" |
| P5 | `findLoweredContractPredicate` (`runner.go:566-605`) reads kind-tagged `meta.Contracts` in source order | full read | **CONFIRMED**; enumerate-all-of-kind is a pure filter over already-cached `DeclMeta` |
| P6 | Repeated `requires` blocks are impossible (`PAR_DUPLICATE_REQUIRES`) | `./bin/ailang check` on a 2-block fixture | **CONFIRMED**: `PAR_DUPLICATE_REQUIRES at :4:18: only one requires block per function; combine with commas` |
| P7 | Comma-separated conditions enumerate in source order at cols 12 / 19 | `ailang test --format json` on the doc's `ordered.ail` | **CONFIRMED exactly**: `g_property_1 skip :5:12`, `g_property_2 skip :5:19`, `g_property_3 fail :6:11`, `h_property_1 pass :12:11` |
| P8 | `TestConfig.WorkspaceRoot` does not exist; no `internal/testing/config.go` | `grep -rnE 'workspaceRoot\|WorkspaceRoot\|type GenConfig\|type TestConfig\|type Config' internal/testing/*.go cmd/ailang/test.go` | **CONFIRMED**: sole output `generator.go:17:type GenConfig struct {` — the known-positive control fires, so the empty result is a fact, not a claim. `ls internal/testing/config.go` → no such file |
| P9 | CI never runs `ailang test` over examples | read `scripts/verify_examples.go:104-115` | **CONFIRMED**: all three exec forms pass the `run` subcommand |
| P10 | 27 example files contain `requires {` | `rg -l 'requires \{' examples \| wc -l` | **CONFIRMED**: 27 |
| P11 | `runner.go` is 790 of an 800 hard cap | `wc -l`; `make -n check-file-sizes` | **CONFIRMED**: 790 lines; the gate is `wc -l > 800` over `find internal cmd -name '*.go'`, `exit 1`, wired at `.github/workflows/ci.yml:97`. **10 lines of headroom.** |

Additionally reproduced first-hand:

- **#547 reproduced 3/3** on the doc's `precond2.ail`. Every run: `big_property_2 fail —
  "ensures violated for input: x=-611 / -384 / -787"`, while `big_property_1` **skips** on the same
  class of input with `requires not satisfied by random input`. The asymmetry is live.
- **#535 reproduced 5 runs** on `examples/runnable/contracts/list_recursive_verify.ail`: exit codes
  `1, 1, 0, 1, 1` from an unchanged file and an unchanged binary.

### 0.2 REFUTED — AC3's fixture cannot pass, for a reason unrelated to this sprint

**The design doc's AC3 fixture is unimplementable as written.** Two independent defects:

1. `assert(true)` is not a valid AILANG expression. `assert` is a reserved keyword
   (`std/debug.ail:18`: *"We use `check` instead of `assert` because `assert` is a reserved
   keyword"*). Measured: `PAR_NO_PREFIX_PARSE ... unexpected token in expression: assert`.
2. Even after correcting the body to `test "..." { true }`, the fixture still fails — because of a
   **pre-existing, previously-unreported bug** in the named-test body synthesizer.

Measured isolation (four fixtures, one variable each):

| Fixture | Contents | `test` block verdict |
|---|---|---|
| `t4.ail` | `test "..." { true }`, no contracts | **pass**, suite exit 0 |
| `t1.ail` | `test "..." { assert(true) }`, no contracts | fail — `assert` unparseable |
| `t3.ail` | `test "..." { true }` **+ a `requires`/`ensures` function** | **fail** |

`t3`'s error:

```
pipeline error: module loading error: failed to load .../_namedtest_body_2733745973.ail:
PAR_NO_PREFIX_PARSE at .../_namedtest_body_2733745973.ail:2:1: unexpected token in expression: requires
PAR_NO_PREFIX_PARSE at .../_namedtest_body_2733745973.ail:3:1: unexpected token in expression: ensures
```

`Executor.executeNamedTestBody` (`internal/testing/executor.go:190-300`) rebuilds the module from
`e.stripNonPureFunctions(...)` (`executor.go:475`) and writes `_namedtest_body_*.ail`. The stripper
removes the function it does not want but **leaves the orphaned `requires` / `ensures` clauses at
top level**, producing a file the parser rejects. Consequence: **any `.ail` file that contains both a
named `test` block and a contract-bearing function has a broken named-test path today.**

- **Impact on this sprint**: AC3 is rewritten (§4.3) to drop the sentinel `test` block entirely and
  use a second *contract-bearing* function instead. Verified this substitute works at HEAD.
- **Routing**: the stripper bug is **OUT OF SCOPE** here. It is a separate AILANG-fix-lane item;
  file it as a new issue during landing. Do **not** let the executor fix it inside this sprint.

### 0.3 REFUTED — AC9's fixture does not parse; AC9 needs a new fixture

> **DISCHARGED 2026-08-01 (iteration 127) — AC9 IS NO LONGER BLOCKED. Do not re-derive this.**
> The cause was never the contract form. `stripNonPureFunctions` deleted a single line of a
> declaration that spans several, and treated "not declared `pure`" as non-pure — so the function
> under extraction was cut out of the temp module and its `requires`/`ensures` were left orphaned.
> Fixed by the declaration-aware strip in `internal/testing/source_strip.go`.
> **AC9's fixture below, verbatim and unmodified, now passes.** Paired measurement:
> pre-fix binary → `rc=1`, `PAR_NO_PREFIX_PARSE at ...:3:1: unexpected token in expression: ensures`;
> post-fix binary → `rc=0`, `1 tests: 1 passed, 0 failed, 0 skipped`.
> Committed regression fixture: `internal/testing/testdata/strip/moduleless_contract.ail`.
> **M2 may keep AC9 exactly as originally written** — no restatement over a module-bearing file is
> needed, so §5.4's fallback is moot. The rest of this section is kept as the historical record.

AC9's fixture is module-less by design (it exists to prove module-less identity derivation):

```ailang
export func stable(x: int) -> bool ! {}
ensures { result == true } { true }
```

Measured, both inside `/tmp` and inside the repo (so this is not the `MOD010 temp-path` relaxation):

```
"error": "failed to extract function binding for stable: ... parse errors in
 /tmp/claude-501/ailang-test-2465348867/stable.ail:
 PAR_NO_PREFIX_PARSE at ...:3:1: unexpected token in expression: ensures"
```

`ExtractFunctionBinding` copies the source into a temp directory and re-parses; the module-less
contract form does not survive that round-trip. **AC9 as written can never exit 0.** M2/M3 must
either supply a module-less fixture form that is proven to parse *before* the seed work starts, or
AC9 must be restated over a module-bearing file placed at the same workspace-relative path under two
different absolute roots (which still tests exactly what AC9 claims to test: that the absolute
checkout directory is absent from the hash identity). See §5.4.

### 0.4 DOC DEFECT — M1's own acceptance criteria depend on an M2 flag

`ailang test` today accepts exactly four flags (`cmd/ailang/main.go:112-117`):
`--format`, `--no-color`, `--package`, `--allow-skips`. **There is no `--seed`.** It is an M2
deliverable.

But AC1, AC3 and AC8 all invoke `./bin/ailang test --seed 42 ...`. **As written, none of M1's
acceptance criteria can be run after M1.** That directly contradicts the controller's hard
constraint 3 (M1 must be independently landable and independently valuable).

Resolution: §4 restates AC1/AC2/AC3/AC8 **seed-free**. This costs nothing, because each restated
criterion is deterministic-in-practice under the current wall-clock RNG:

- AC1 `precond2`: `requires { x > 100 }` over the default int generator range `[-1000, 1000]`
  (`generator.go:29-33`) accepts with p ≈ 900/2001 ≈ 0.45. Reaching 100 acceptances within 1,000
  attempts has mean 450 successes, sd ≈ 15.7 — the failure probability is astronomically small.
- AC3 unreachable: `x > 1000 && x < -1000` accepts with p = 0, so 1000 generated / 1000 discarded is
  exact for **every** seed.
- AC8 `ordered.ail`: `g` accepts with p ≈ 0.25, mean 250 successes in 1,000 attempts.
- AC2 negative control: a genuine violation on the first in-contract input, seed-independent.

The `--seed 42` forms in the doc remain correct **after M2** and are kept there as M3 criteria.

### 0.5 DOC GAP — the discard counters would be dead fields on two of three property paths

The doc adds `generated_inputs` / `discarded_inputs` for the ensures path only. `runForallProperty`
and `runRequiresProperty` would then emit `0` for both, which is indistinguishable from "generated
nothing". That is a silent-fallback shape (CLAUDE.md §2) and it is the "dead config field" the
controller's constraint 3 forbids.

**Binding for M1**: all three property paths populate both counters. `forall` and `requires` set
`GeneratedInputs = attempts`, `DiscardedInputs = 0`. ~4 extra lines, no dead fields.

### 0.6 Sequencing is respected

M1 (#547 filter) lands **before** M2 (#535 seed). Executing M1 alone tonight is therefore safe by
construction: it is the half of the pair that the doc says must go first. There is no ordering in
which "M1 only" freezes a false positive.

---

## 1. Velocity basis

Recent comparable AILANG-lane sprints in `internal/` + `cmd/ailang`:

| Sprint | Net LOC | Days |
|---|---:|---:|
| m-nightly-run-validity-gate | ~635 | 1.1 |
| m-nightly-sustained-failure-label | ~415 | 0.9 |
| m-planner-codex-lane (infra) | ~307 | 1.0 |

Sustained rate ≈ **350–450 net LOC/day** for test-heavy Go work in this repo. The design doc's
650–830 LOC / 1.5–2 day estimate is consistent with that and **does not fit one unattended night**,
exactly as the controller states.

**M1 alone is ~430–500 net LOC**, which is one bounded night at the upper end of the sustained rate.
§3.4 defines the descope valve if it runs long.

---

## 2. Milestone map

| ID | Name | Net LOC | Est. | Tonight? | Independently landable |
|---|---|---:|---:|---|---|
| **M1** | Sound `ensures` domain filtering (#547) | ~470 | 0.6 d | **YES — the only milestone executed this iteration** | **YES** |
| M2 | Deterministic + replayable seed policy (#535) | ~280 | 0.6 d | NO — parked to the 2026-08-03 re-arm | Yes, only after M1 |
| M3 | Integration + repository gates | ~180 | 0.4 d | NO — parked | No (needs M2) |

M2 and M3 are specified here so the re-arm session does not re-plan from scratch, but **only M1 is
armed**. The sprint-state JSON marks M2/M3 `parked`.

---

## 3. M1 — Sound `ensures` domain filtering (#547)

### 3.1 Hard constraint: the 800-line cap

`internal/testing/runner.go` is **790 lines**. The CI gate fails at 801. M1 adds ~90 lines of logic
to a function that lives in that file. Growing it in place breaks CI.

**Binding structural requirement — do this FIRST, as its own commit:**

1. Create `internal/testing/contract_domain.go` (same package `testing`).
2. **Move, byte-for-byte, `runEnsuresProperty` (`runner.go:312-432`, 121 lines including its doc
   comment) into the new file.** Pure relocation, zero behavior change.
3. Verify: `go build ./... && go test ./internal/testing -count=1` green, and
   `wc -l internal/testing/runner.go` = **669**.
4. Only then implement the filtering logic in `contract_domain.go`.

This leaves 131 lines of headroom in `runner.go` and gives `contract_domain.go` room for the whole
feature. `findAllLoweredContractPredicates` goes in the new file too — **not** in `runner.go`.

Do **not** move `findLoweredContractPredicate`, `runRequiresProperty`, or `runForallProperty`;
keeping the diff minimal keeps the pure-move commit reviewable.

### 3.2 Implementation

**(a) `findAllLoweredContractPredicates(propCase PropertyCase, coreKind core.ContractKind) []core.CoreExpr`**

Mirrors the second half of `findLoweredContractPredicate` (`runner.go:593-604`) but appends every
same-kind entry instead of indexing to one:

```go
meta := r.executor.LastDeclMeta(propCase.FunctionCtx)   // nil -> return nil
var out []core.CoreExpr
for _, c := range meta.Contracts {
    if c.Kind == coreKind {
        out = append(out, c.Expr)
    }
}
return out
```

Note the signature difference from the doc: the existing accessor takes **both** an `ast.ContractKind`
and a `core.ContractKind` (`runner.go:566`). The new one needs only the `core` kind, because it does
no index matching against `propCase.Function.Properties`. That is the correct simplification, not an
omission.

`meta.Contracts` order is the elaborator's source order — verified by P7 (cols 12 then 19).

**(b) Filtered ensures loop** replacing `runner.go:388-427`:

```
const (
    requiredAccepted = 100
    maxAttempts      = 1000
)
requires := r.findAllLoweredContractPredicates(propCase, core.RequiresKind)
accepted, generated, discarded := 0, 0, 0

for accepted < requiredAccepted && generated < maxAttempts {
    generated++
    <generate one tuple; build ensuresParams>            // one RNG draw per attempt, unchanged
    if len(requires) > 0 {
        ok, err := r.allRequiresHold(ensuresParams, requires)   // short-circuits on first false
        if err != nil { -> StatusFail (loud; never a discard) }
        if !ok { discarded++; continue }                        // function NOT called
    }
    <existing ensures harness evaluation, unchanged>
    accepted++
}
```

- `allRequiresHold` evaluates each predicate with the **existing**
  `Executor.EvaluateRequiresHarnessFromCore(harnessParams, pred)` — verified at `runner.go:519` to
  take exactly `([]EnsuresParam, core.CoreExpr)` and to bind parameter values **without calling the
  function**. Same `[]EnsuresParam` value the ensures harness would use: identical tuple, one draw.
- A non-bool or evaluation error from a requires predicate is `StatusFail`, never a discard
  (design doc, Conflict Surface row 4).
- `result.TestsRun = accepted` — TestsRun keeps meaning "executed in-contract cases".
- `result.GeneratedInputs = generated`, `result.DiscardedInputs = discarded`.

**(c) Cap exhaustion → honest skip** (constraint 4: never a new vacuous pass):

```
if accepted < requiredAccepted {
    result.Status   = StatusSkip
    result.SkipKind = SkipKindOutOfContract
    result.Error    = fmt.Sprintf(
        "unverified: requires filter accepted %d of %d generated inputs; need %d (%d discarded)",
        accepted, generated, requiredAccepted, discarded)
    return result
}
result.Status = StatusPass
```

99 accepted + 901 discarded is **skip**, never pass. This is the exact boundary a mutation test must
kill (§3.3, T6).

`SkipKindOutOfContract` already exists (`result.go:20`) and is already excluded from `VacuousSkips`
(`result.go:97-99`), so an out-of-contract skip does not by itself fail a suite that has other
passing work. Verified by reading `AddPropertyResult` and `Success()`.

**(d) Counters on the other two paths** (§0.5): in `runForallProperty` and `runRequiresProperty` set
`GeneratedInputs = <attempts made>` and `DiscardedInputs = 0` before every `return`. No dead fields.

**(e) Reporting**:
- `result.go`: `GeneratedInputs int` and `DiscardedInputs int` on `PropertyResult` (+ doc comments).
- `reporter.go` `formatPropertiesJSON` (`reporter.go:82-101`): always emit `"generated_inputs"` and
  `"discarded_inputs"` as **integers**. No float `discard_rate`.
- `reporter.go` human path: for any property with `DiscardedInputs > 0` **or** an
  `out_of_contract` skip, print `accepted A, discarded D, generated G`.

### 3.3 M1 test plan (new file `internal/testing/contract_domain_test.go`)

Reuse the existing `runEnsuresFromSource(t, src)` helper (`runner_ensures_test.go:16-37`) —
inline source + `t.TempDir()`. No new Go testdata harness needed.

| ID | Test | Asserts |
|---|---|---|
| T1 | `TestFindAllLoweredContractPredicates_OrderAndAssociation` | `g` with `requires { x > 0, y > 0 }` returns exactly 2 predicates, in source order |
| T2 | `TestFindAllLoweredContractPredicates_ZeroRequires` | `h` with no requires returns an **empty/nil** slice (the AC8 named test) |
| T3 | `TestEnsuresFiltersOutOfContractInputs` | `precond2`: status pass, `TestsRun == 100`, `DiscardedInputs > 0`, `GeneratedInputs == 100 + DiscardedInputs` |
| T4 | `TestEnsuresGenuineViolationStillFails` | `precond_negative`: status fail, error contains `ensures violated`, counterexample matches `x=[1-9][0-9]{2,}` (i.e. the reported input **satisfies** `x > 100`) |
| T5 | `TestEnsuresUnreachableDomainIsUnverifiedSkip` | status skip, `SkipKind == out_of_contract`, `TestsRun == 0`, `GeneratedInputs == 1000`, `DiscardedInputs == 1000`, error `HasPrefix "unverified:"` |
| T6 | `TestEnsuresNinetyNineAcceptedIsNotAPass` | a domain tuned to accept ~5% → status **skip**, not pass. Mutation guard on the `accepted < 100` boundary |
| T7 | `TestEnsuresNoRequiresFastPath` | `clampOk` (no requires): pass, `TestsRun == 100`, `GeneratedInputs == 100`, `DiscardedInputs == 0` |
| T8 | `TestEnsuresRequiresEvaluationErrorFailsLoudly` | a requires predicate that errors/returns non-bool → `StatusFail`, **not** a discard |
| T9 | `TestEnsuresDiscardedTupleNeverCallsFunction` | a function whose body would error out of contract; assert it is never entered (e.g. the property is skip/pass, no evaluation error surfaces) |

Plus, in `internal/testing/reporter_test.go`: `TestReporter_JSONIncludesDiscardCounters` — the two
keys are present **and integer-typed** on pass, fail, and out-of-contract-skip properties.

### 3.4 Descope valve (bounded night)

If the executor runs long, ship in this order and stop at the boundary:

1. **Must**: the pure move (§3.1) + `findAllLoweredContractPredicates` + filtered loop + cap skip +
   counters on all three paths + T1–T7.
2. **Should**: T8, T9, reporter test, `cmd/ailang` e2e test.
3. **May defer to M3**: the `cmd/ailang/test_contract_domain_test.go` end-to-end file.

Never stop between 1 and the file-size check. A half-applied §3.1 move is the one state that leaves
CI red.

### 3.5 Files touched by M1

| File | Change | LOC |
|---|---|---:|
| `internal/testing/contract_domain.go` (NEW) | moved `runEnsuresProperty` (121, neutral) + `findAllLoweredContractPredicates` + `allRequiresHold` + filtered loop | +240 |
| `internal/testing/runner.go` | delete moved function; counters in forall/requires paths | +6 / −121 |
| `internal/testing/result.go` | 2 fields + comments | +6 |
| `internal/testing/reporter.go` | integer JSON keys + human counts line | +30 |
| `internal/testing/contract_domain_test.go` (NEW) | T1–T9 | +240 |
| `internal/testing/reporter_test.go` | discard-counter pinning | +35 |
| `internal/testing/testdata/contracts/*.ail` (NEW ×4) | AC fixtures | +32 |
| `cmd/ailang/test_contract_domain_test.go` (NEW) | CLI JSON e2e for AC1/AC2/AC3 | +120 |
| `CHANGELOG.md` | entry under Unreleased/Fixed | +12 |

**Net ≈ +590 gross, ≈ +470 net** (the 121-line move is behavior-neutral).

**Explicitly NOT touched by M1**: `internal/testing/generator.go`, `internal/testing/config.go`
(not created), `cmd/ailang/main.go`, `cmd/ailang/test.go`, `internal/testing/harness.go`,
`internal/testing/executor.go`. Any diff in those files is scope creep and must be rejected.

---

## 4. M1 acceptance criteria — seed-free, runnable, with expected results

All commands from the worktree root after `make build`. **`--seed` does not exist until M2** (§0.4).

### AC1-M1 — excluded inputs no longer fail

Fixture `internal/testing/testdata/contracts/precond2.ail`:

```ailang
module precond2
export func big(x: int) -> bool ! {}
requires { x > 100 }
ensures { result == true } {
  x > 100
}
export func main() -> int ! {} { if big(500) then 1 else 0 }
```

```bash
./bin/ailang check internal/testing/testdata/contracts/precond2.ail
./bin/ailang test --format json --no-color internal/testing/testdata/contracts/precond2.ail \
  > /tmp/precond2.json
echo "exit=$?"
jq -e '.failed_tests == 0
  and ([.properties[]
        | select(.name == "big_property_2")
        | select(.status == "pass" and .tests_run == 100
                 and .discarded_inputs > 0
                 and .generated_inputs == (100 + .discarded_inputs))]
       | length == 1)' /tmp/precond2.json
```

**Expected**: every command exits 0. `big_property_1` (the standalone requires property) may remain
an `out_of_contract` skip — that is unchanged M1 behavior and is not asserted.
**Measured at HEAD (must flip)**: `big_property_2` is `fail` with `ensures violated for input:
x=-611` in 3 of 3 runs.

### AC2-M1 — genuine in-domain violation still fails (negative control)

Fixture `internal/testing/testdata/contracts/precond_negative.ail`: as in the design doc.

```bash
./bin/ailang check internal/testing/testdata/contracts/precond_negative.ail
set +e
./bin/ailang test --format json --no-color \
  internal/testing/testdata/contracts/precond_negative.ail > /tmp/precond-negative.json
rc=$?
set -e
test "$rc" -eq 1
jq -e '.properties[]
  | select(.name == "broken_property_2")
  | select(.status == "fail"
           and (.error | contains("ensures violated"))
           and (.error | test("x=[1-9][0-9]{2,}")))' /tmp/precond-negative.json
```

**Expected**: all assertions exit 0; the suite exits 1. The `x=[1-9][0-9]{2,}` pattern is the
load-bearing part — it proves the reported counterexample **satisfies** `x > 100`, i.e. filtering
did not manufacture a vacuous pass.

### AC3-M1 — unreachable domain is unverified, never passed or violated (**FIXTURE REWRITTEN**, §0.2)

Fixture `internal/testing/testdata/contracts/requires_unreachable.ail` — **no `test` block**; a
second contract-bearing function keeps `ran > 0`:

```ailang
module requires_unreachable
export func impossible(x: int) -> bool ! {}
requires { x > 1000 && x < -1000 }
ensures { result == true } { false }
export func ok(x: int) -> bool ! {}
ensures { result == true } { true }
```

```bash
./bin/ailang check internal/testing/testdata/contracts/requires_unreachable.ail
./bin/ailang test --format json --no-color \
  internal/testing/testdata/contracts/requires_unreachable.ail > /tmp/unreachable.json
echo "exit=$?"
jq -e '.failed_tests == 0 and .success == true
  and ([.properties[]
        | select(.name == "impossible_property_2")
        | select(.status == "skip" and .skip_kind == "out_of_contract"
                 and .tests_run == 0
                 and .generated_inputs == 1000 and .discarded_inputs == 1000
                 and (.error | startswith("unverified:")))] | length == 1)
  and ([.properties[] | select(.name == "ok_property_1" and .status == "pass")] | length == 1)' \
  /tmp/unreachable.json
```

**Expected**: exit 0 for all.
**Measured at HEAD (must flip)**: `impossible_property_2` = `fail`, `ok_property_1` = `pass`,
suite exit 1.
**Why the design doc's fixture was dropped**: `test "..." { assert(true) }` does not parse, and even
`test "..." { true }` fails in a contract-bearing file because the named-test body synthesizer emits
orphaned `requires`/`ensures` at top level (§0.2). Do **not** attempt to fix that here.

### AC4-M1 — functions without `requires` are unaffected

```bash
go test ./internal/testing \
  -run 'TestRunEnsuresProperty_(CorrectImplPasses|ViolationReportsCounterexample|MultiArgPredicateReferencesParams|ArithmeticInPredicate|ArithmeticViolation|MultiArgViolationReportsAllArgs)' \
  -count=1
go test ./internal/testing -run 'TestRunContractProperties_SkipKinds' -count=1
```

**Expected**: exit 0, unmodified. Verified by reading these tests: every ensures fixture in them
(`clampBuggy`, `clampOk`, `maxOf`, …) has **no** `requires`, so the empty-slice fast path must keep
them byte-identical. `TestRunContractProperties_SkipKinds` skips on `no_generator` before filtering.
If any of these needs editing, the fast path is wrong.

### AC8-M1 — requires representation, association, ordering, accessor coverage (seed-free)

```bash
tmp=$(mktemp -d)
cat >"$tmp/repeated.ail" <<'EOF'
module repeated

export func g(x: int, y: int) -> bool ! {}
requires { x > 0 }
requires { y > 0 }
ensures { result == true } { x > 0 }
EOF
set +e
./bin/ailang check "$tmp/repeated.ail" >"$tmp/repeated.out" 2>&1
rc=$?
set -e
test "$rc" -ne 0
grep -F 'PAR_DUPLICATE_REQUIRES' "$tmp/repeated.out"

cat >"$tmp/ordered.ail" <<'EOF'
module ordered

export func g(x: int, y: int) -> bool ! {}

requires { x > 0, y > 0 }
ensures { result == true } { x > 0 }


export func h(x: int) -> bool ! {}


ensures { result == true } { true }
EOF
set +e
./bin/ailang test --format json --no-color "$tmp/ordered.ail" >"$tmp/ordered.json"
rc=$?
set -e
test "$rc" -eq 0
jq -e '[.properties[].name] == ["g_property_1","g_property_2","g_property_3","h_property_1"]
  and (.properties[0].location | endswith(":5:12"))
  and (.properties[1].location | endswith(":5:19"))
  and (.properties[2].location | endswith(":6:11"))
  and (.properties[3].location | endswith(":12:11"))
  and (.properties[2].status == "pass")
  and (.properties[2].discarded_inputs > 0)
  and (.properties[3].status == "pass")
  and (.properties[3].discarded_inputs == 0)' "$tmp/ordered.json"
go test ./internal/testing \
  -run 'TestFindAllLoweredContractPredicates_(OrderAndAssociation|ZeroRequires)' -count=1
```

**Expected**: all exit 0.
**Measured at HEAD**: locations `5:12 / 5:19 / 6:11 / 12:11` already match exactly; but
`g_property_3` is currently **fail** and the suite exits 1. M1 flips property 3 to `pass` (its
`requires { x > 0, y > 0 }` makes the body `x > 0` true for every in-contract input), and
`h_property_3`'s `discarded_inputs == 0` proves the zero-requires isolation.

### AC10-M1 — file-size gate (the hard constraint)

```bash
make check-file-sizes
test "$(wc -l < internal/testing/runner.go)" -le 700
```

**Expected**: exit 0; `runner.go` is 669 after the §3.1 move. `-le 700` (not `-le 800`) is
deliberate — it fails loudly if the executor grew `runner.go` in place instead of moving.

### AC11-M1 — the previously-flaky example becomes deterministic in verdict shape

```bash
for i in 1 2 3 4 5; do
  ./bin/ailang test --format json --no-color \
    examples/runnable/contracts/list_recursive_verify.ail > "/tmp/lr$i.json" 2>/dev/null
  echo "rc=$?"
  jq -S -c '[.properties[] | {n:.name, s:.status, k:.skip_kind}]' "/tmp/lr$i.json"
done
```

**Expected after M1**: `containsImpliesNonEmpty_property_2` is **never** `fail` in any of the 5 runs.
Its stable verdict may be `pass` or an honest `out_of_contract` skip — both are acceptable; a
`fail` is not.
**Measured at HEAD**: exit codes `1, 1, 0, 1, 1` and property_2 = `fail, fail, pass, fail, fail`.
**Note**: exit codes may still vary after M1 alone, because #535 is still open. That is expected;
AC11 asserts the *verdict class*, not the exit code. Exit-code stability is M2's AC5.

### AC12-M1 — repository gates

```bash
go test ./internal/testing/... -count=1
go test ./cmd/ailang/... -count=1
make lint
make verify-examples
```

**Expected**: all exit 0.
`make verify-examples` is a **run-mode** gate only — it passes the `run` subcommand
(`scripts/verify_examples.go:104-115`), so a green result is **not** evidence that any contract
property executed. Record it as such.
**SANDBOX**: a `bind: operation not permitted` or any loopback-socket denial in
`go test ./cmd/ailang/...` is **UNINFORMATIVE UNDER SANDBOX** — neither pass nor product failure.
The controller re-runs AC12 outside the sandbox before landing.

### AC13-M1 — cost bound on the new loop

```bash
/usr/bin/time -p ./bin/ailang test --format json --no-color \
  examples/runnable/contracts/list_recursive_verify.ail >/dev/null 2>/tmp/lr.time
cat /tmp/lr.time
```

**Expected**: wall time under 60 s. This example has list-typed parameters and a
`_list_contains`-based precondition that the generator will rarely hit, so it is the worst case for
the new 1,000-attempt cap × 6 properties. If it exceeds 60 s, **stop and report** — do not silently
lower the cap; the 1,000 bound is a frozen design decision.

---

## 5. M2 and M3 — the 2026-08-07 re-arm (mission iteration 159)

**This section replaces the ~20-line M2/M3 stub written in iteration 126.** That stub was a seed,
not a plan: it had no task decomposition, no unit-test plan, and no restated-runnable acceptance
criteria. §0.4 records what happens when that pass is skipped — when M1's ACs were finally checked,
*every one of them was unrunnable*. The same audit has now been done for M2/M3 and it found four
more defects (§5.11).

**Planner**: claude-opus-5, mission iteration 159, 2026-08-07.
**Re-arm base**: `086902493` (local `dev` HEAD at planning time; `origin/dev` is one commit behind
at `c30d2cb1b`, and the only delta is `.claude/skills/mission-control/SKILL.md` —
`git diff --name-only c30d2cb1b HEAD -- internal/ cmd/` is **empty**, so every measurement below is
valid at either SHA).

### 5.0 Re-arm verification summary — measured first-hand at HEAD, 2026-08-07

Nothing in §5 is inherited from §0–§4 or from the design doc. Each row is a command re-run now.

| # | Claim | Command | Result |
|---|---|---|---|
| Q1 | `WorkspaceRoot` / `TestConfig` still absent | `grep -rnE 'workspaceRoot\|WorkspaceRoot\|type GenConfig\|type TestConfig\|type Config' internal/testing/*.go cmd/ailang/test.go` | **CONFIRMED**: sole hit `internal/testing/generator.go:17:type GenConfig struct {` — the known-positive control fires, so the empty `WorkspaceRoot` result is a fact |
| Q2 | No `internal/testing/config.go` | `ls internal/testing/config.go` | **CONFIRMED**: No such file. `ls internal/testing/*.go \| wc -l` → **36** |
| Q3 | Three live `newRNG` call sites — **but not where the design doc says** | `grep -rnE 'newRNG\(' internal/testing/*.go` | `contract_domain.go:89`, `runner.go:261`, `runner.go:384`, plus the definition at `runner.go:665` |
| Q4 | Wall-clock branch location | `grep -rnE 'func newRNG\|time\.Now\(\)\.UnixNano' internal/testing/*.go` | `runner.go:665` (def), `runner.go:667` (`seed = time.Now().UnixNano()`) |
| Q5 | `runner.go` size vs the 800 cap | `wc -l internal/testing/runner.go`; `grep -n -A 20 'check-file-sizes:' make/code-health.mk` | **670** lines; cap is `wc -l > 800` over `find internal cmd -name "*.go"`, wired at `.github/workflows/ci.yml:121`. **130 lines of headroom** |
| Q6 | Other files M2 grows | `wc -l cmd/ailang/main.go cmd/ailang/reporter…` | `main.go` **526**, `cmd/ailang/test.go` **246**, `reporter.go` **296**, `result.go` **135**, `contract_domain.go` **194**. None is near the cap |
| Q7 | `ailang test` still has no seed flags | `./bin/ailang test --help` | Options are exactly `--format`, `--no-color`, `--package`, `--allow-skips` |
| Q8 | #535 open, #547 closed | `gh issue view 535/547 --json number,state` | **535 OPEN**, **547 CLOSED** — M2 is what closes #535 |
| Q9 | Repo gates green on pristine `dev` | `go test ./internal/testing/... -count=1`; `go test ./cmd/ailang/... -count=1`; `make lint`; `make verify-examples` | rc=**0**, **0**, **0**, **0** |
| Q10 | Worst-case runtime of the M1 loop | `/usr/bin/time -p ./bin/ailang test --format json --no-color examples/runnable/contracts/list_recursive_verify.ail` | **real 0.65 s**. §4's AC13-M1 60 s bound has ~90x headroom; M2 adds no attempts |
| Q11 | M1's deferred CLI e2e file was never written | `ls cmd/ailang/test_contract_domain_test.go` | **No such file** — §3.4 item 3 was exercised. M3 owes it |
| Q12 | `list_recursive_verify.ail` is module-**bearing** | `grep -n '^module' examples/runnable/contracts/list_recursive_verify.ail` | `5:module examples/runnable/contracts/list_recursive_verify`. So AC5 exercises the *declared-module* identity branch, **not** `WorkspaceRoot`. AC9 remains the only `WorkspaceRoot` criterion |
| Q13 | M1's checked-in fixtures use canonical repo module paths | `grep -n '^module' internal/testing/testdata/contracts/precond2.ail` | `module internal/testing/testdata/contracts/precond2` — any new in-repo fixture must do the same or `check` fails MOD010 |

### 5.1 The design doc's Conflict Surface is STALE on the one line M2 turns on

Design doc, *Conflict Surface*: “`internal/testing/runner.go`: ensures loop, lowered-contract
lookup, **three RNG call sites**.”

**That sentence is false at HEAD** (Q3). M1 moved `runEnsuresProperty` into
`internal/testing/contract_domain.go`, so the three seeded paths are now split across **two** files:

```
internal/testing/contract_domain.go:89   rng := newRNG(config.Seed)   # ensures  (moved by M1)
internal/testing/runner.go:261           rng := newRNG(config.Seed)   # forall
internal/testing/runner.go:384           rng := newRNG(config.Seed)   # requires
```

Each of the three builds its own `config := DefaultConfig()` **locally** (`contract_domain.go:88`,
`runner.go:260`, `runner.go:383`). Threading `TestConfig` therefore has to reach `contract_domain.go`
too — not just `runner.go`. **This repo's recurring failure shape is guarding the helper and missing
a call site** (see the mission memory on `--system-prompt`), so §5.4 makes the sweep an explicit,
grep-checkable step and AC-SEED-SWEEP-M2 (§5.11) pins it.

The stub's own line numbers are stale too: it says the wall-clock branch is at `runner.go:786-788`
(it is **667**) and that `runner.go` is 669 lines (it is **670**).

**There is a fourth call-site class the design doc never mentions.** The top-level JSON fields
(`seed_mode`, `seed`, `seed_derivation`) are emitted from the **aggregate** `SuiteResult`, not from
the per-file one. `cmd/ailang/test.go` has **two** hand-written aggregation blocks that merge
per-file results field by field — `runTestsV2` (lines 50–63) and `runPackageTests` (lines 190–203) —
and neither would carry a new field. Both must be updated; §5.5 pre-decides a single
`SetSeedMetadata` mutator so the two sites cannot drift.

### 5.2 Pre-decided choices — the executor MUST NOT re-decide these

This lane is `pi:openrouter/deepseek/deepseek-v4-flash-0731`. Every judgment call below is decided
here. If the executor finds a decision it thinks is wrong, it **stops and reports**; it does not
substitute its own.

| # | Question | **Decision** | Why it is not obvious |
|---|---|---|---|
| D1 | How to tell `--seed 0` from “no flag”? | `testFlags.Visit(func(f *flag.Flag){…})` after `Parse`. `Visit` iterates **only flags that were set**. | Both cases leave the `int64` at `0`, and both derive the *same* stream. The **only** observable difference is `seed_mode`. Comparing values silently reports `derived` for an explicit `--seed 0`. |
| D2 | Where does the mutual-exclusion error go, and with what exit code? | `fmt.Fprintln(os.Stderr, "Error: --seed and --random-seed cannot be used together")` then `os.Exit(2)`. **stderr, and the substring `cannot be used together` is load-bearing.** | AC6 greps `conflict.err`. Also see §5.11 AC6-M2 baseline: `rc == 2` **already passes on pristine `dev`** because Go's `flag.ExitOnError` exits 2 on the unknown `-seed`. The grep is the only discriminating arm. |
| D3 | Is `filepath.Rel(cfg.WorkspaceRoot, inputPath)` (the doc's literal formula) correct? | **No — make the input absolute first.** `abs, err := filepath.Abs(inputPath)`, then `filepath.Rel(cfg.WorkspaceRoot, abs)`. Error on either → loud failure, no fallback. | `WorkspaceRoot` is absolute (`os.Getwd`) and the CLI input is typically relative (`cases/stable.ail`). `filepath.Rel(abs, rel)` returns `can't make … relative to …`. The doc's formula errors on exactly the AC9 case it was written for. |
| D4 | Where is `WorkspaceRoot` consumed? | Exactly once, in `ResolveModuleIdentity`, called from `RunTestsFromFileWithConfig` **before** the Runner exists. The Runner still **retains the whole `TestConfig`** (design doc, §M2) but never re-reads `WorkspaceRoot`. | One consumption point is one place for AC9 to pin. Re-deriving it per property is how the checkout path becomes ambient again. |
| D5 | Does `NewRunner(modulePath)` survive? | **Yes**, as a documented convenience wrapper. It builds an **explicit** `TestConfig{WorkspaceRoot: filepath.Dir(abs(modulePath)), SeedMode: SeedModeDerived, MasterSeed: 0}`. CLI code must **not** call it. | `grep -rn 'NewRunner('` finds **16 in-package test call sites**. Deleting it is a 16-file blast radius for zero benefit, and the design doc explicitly says to keep the convenience entry point. |
| D6 | Same for `RunTestsFromFile(filePath, ast)`? | **Yes**, kept as a wrapper over the new `RunTestsFromFileWithConfig`. Callers: `internal/testing/executor_regression_test.go:27`, `internal/testing/named_test_test.go:418`, and `cmd/ailang/test.go:119`. **`cmd/ailang/test.go:119` MUST switch to the configured form**; the two test callers may stay. | Design doc: “CLI code must use the configured entry point and may not silently infer a workspace root inside `internal/testing`.” |
| D7 | Where is entropy read, so the failure path is testable? | `internal/testing/config.go`: `var EntropyReader io.Reader = crand.Reader` + `func NewRandomMasterSeed() (int64, error)`. The CLI calls it once. Tests swap `EntropyReader`. | The doc puts the read “at the CLI boundary”. A read literally inside `cmd/ailang/main.go` is not injectable, and the doc *also* requires an injected-reader negative test. Helper in `internal/testing`, call site in the CLI, satisfies both. |
| D8 | Which properties get a `seed` in JSON? | **All of them, always**, including `skip`. Set `result.Seed` in `runProperty` (`runner.go:219`) **before** the `switch`, so one assignment covers all three paths and every early return. | AC5 asserts `[.properties[] \| select(.seed != null)] \| length > 0`; AC6 asserts `>= 2`. A `no_generator` skip that returns before the RNG is built would otherwise emit `null`. One site, not three. |
| D9 | What is the `replay` field, and where does it live? | A per-property JSON key emitted **only when `prop.Status == StatusFail`**, computed *in the reporter* as `ailang test --seed <MasterSeed> <ModulePath>` from the `SuiteResult`. No new struct field. | `PropertyResult` does not know the invocation path or the master. Adding a field would require filling it at three more sites. |
| D10 | Human-output wording | Free (design doc, *Deferred Decisions*), **but** mode, master seed, derivation version and the copy/paste replay command are mandatory and must appear in `reportSummaryHuman`. | Punctuation is explicitly the implementer's call; the *content* is not. |
| D11 | Do the `--seed 42` forms of AC1/AC3/AC8 come back? | **Yes, as M3 criteria**, in addition to the seed-free §4 forms — not replacing them. | §4's forms are the landed M1 record and remain the regression for the filter itself. |

### 5.3 M2A — `TestConfig` + the v1 derivation core (internal only)

**~230 net LOC (impl ~120 / test ~110). No CLI change, no behavior change.** Independently
verifiable with `go test ./internal/testing/... -count=1`; the binary behaves identically after M2A.

New file **`internal/testing/config.go`** (nothing else is touched by M2A):

```go
package testing

type SeedMode string

const (
    SeedModeDerived SeedMode = "derived" // no seed flag; master is int64(0)
    SeedModeMaster  SeedMode = "master"  // --seed N (incl. 0) or --random-seed
)

// SeedDerivationV1 is a REPRODUCIBILITY CONTRACT. Changing the framing, hash,
// byte order or identity below is a breaking change and REQUIRES a new tag.
const SeedDerivationV1 = "ailang-property-seed-v1"

type TestConfig struct {
    WorkspaceRoot string   // absolute; the CLI's initial os.Getwd()
    SeedMode      SeedMode
    MasterSeed    int64
}

func (c TestConfig) Validate() error            // non-empty + filepath.IsAbs root; SeedMode is one of the two
func DeriveSeedV1(master int64, moduleIdentity, propertyName string) int64
func ResolveModuleIdentity(workspaceRoot, inputPath, declaredModule string) (string, error)
func NewRandomMasterSeed() (int64, error)
var EntropyReader io.Reader = crand.Reader
```

**`DeriveSeedV1` — frozen, byte-exact:**

```go
var b bytes.Buffer
b.WriteString(SeedDerivationV1); b.WriteByte(0)
b.WriteString(strconv.FormatInt(master, 10)); b.WriteByte(0)
b.WriteString(moduleIdentity); b.WriteByte(0)
b.WriteString(propertyName)
sum := sha256.Sum256(b.Bytes())
return int64(binary.LittleEndian.Uint64(sum[0:8]))
```

**`ResolveModuleIdentity` — the only consumer of `WorkspaceRoot` (D3, D4):**

```go
if declaredModule != "" { return declaredModule, nil }        // module-bearing: path-independent
abs, err := filepath.Abs(inputPath); if err != nil { return "", fmt.Errorf(...) }
rel, err := filepath.Rel(workspaceRoot, abs); if err != nil { return "", fmt.Errorf(...) }
return filepath.ToSlash(filepath.Clean(rel)), nil
```

No fallback on either error (CLAUDE.md §2: this value decides a seed, so a wrong-but-plausible
answer is worse than a failure).

**`NewRandomMasterSeed`**: `io.ReadFull(EntropyReader, b[:8])`; on error return
`fmt.Errorf("--random-seed: failed to read 8 bytes of entropy: %w", err)` — **never** a clock,
constant or per-property fallback. Value is `int64(binary.LittleEndian.Uint64(b[:]))`.

### 5.4 M2B — thread the config to all three RNG sites; delete the wall clock

**~140 net LOC (impl ~55/−5, test ~85).** This is the milestone that closes #535's mechanism.

1. `Runner` (`runner.go:15-18`) gains two fields: `config TestConfig` and `moduleIdentity string`.
2. Add `func NewRunnerWithConfig(modulePath string, cfg TestConfig, moduleIdentity string) *Runner`.
   It cannot fail — identity is already resolved (D4).
3. Rewrite `NewRunner(modulePath string) *Runner` as the D5 wrapper. **Signature unchanged**, so all
   16 in-package callers keep compiling.
4. Add `func (r *Runner) propertySeed(name string) int64 { return DeriveSeedV1(r.config.MasterSeed, r.moduleIdentity, name) }`.
5. **Sweep all three sites.** At each, delete the local `config := DefaultConfig()` (it was used only
   for `.Seed`) and replace `newRNG(config.Seed)` with `newRNG(r.propertySeed(propCase.Name))`:
   - `internal/testing/contract_domain.go:88-89` (ensures) — **the one the design doc's stale
     Conflict Surface omits**
   - `internal/testing/runner.go:260-261` (forall)
   - `internal/testing/runner.go:383-384` (requires)
   Leave the *other* `DefaultConfig()` uses alone: `runner.go:521,524,529,540,551` are inside
   `createGeneratorForType` and read `MinInt`/`MaxInt`/`MaxSize`, not `Seed`.
6. `newRNG` (`runner.go:665-670`) loses its wall-clock branch entirely:
   `func newRNG(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }`. Drop the now-unused
   `time` import **only if** nothing else in `runner.go` uses it (it does — `time.Since`; leave it).
7. Set `result.Seed = r.propertySeed(propCase.Name)` in `runProperty` (`runner.go:219`) **before** the
   `switch` (D8). Add `Seed int64` to `PropertyResult` (`result.go:33-44`).
8. Add `RunTestsFromFileWithConfig(filePath string, file *ast.File, cfg TestConfig) (*SuiteResult, error)`
   next to the existing `RunTestsFromFile` (`runner.go:501`). It: validates the config; computes
   `declared := ""` / `file.Module.Path` (`internal/ast/ast.go:139` `Module *ModuleDecl`, `:150` `Path string`);
   calls `ResolveModuleIdentity`; builds the Runner via `NewRunnerWithConfig`; and stamps
   `SeedMode`/`MasterSeed`/`SeedDerivation` onto the returned `SuiteResult`.
9. Add to `SuiteResult` (`result.go:47-57`): `SeedMode SeedMode`, `MasterSeed int64`,
   `SeedDerivation string`, plus `func (sr *SuiteResult) SetSeedMetadata(cfg TestConfig)` — the single
   mutator both CLI aggregates will call (§5.1).

**After M2B the default run is already deterministic**, but nothing is reported yet and there are no
flags. That intermediate state is coherent (strictly better than a hidden wall clock) but it is
**not landable on its own** — see §5.9.

### 5.5 M2C — CLI flags, propagation through both aggregates, reporting

**~200 net LOC (impl ~145, test ~55).**

> **INHERITED FROM M2B (added iteration 162, from the evaluator's one non-blocking finding).**
> `NewRunner` and `RunTestsFromFile` (`internal/testing/runner.go`) both swallow a `filepath.Abs`
> error with `abs = modulePath`. CLAUDE.md §2 forbids a silent fallback where the value affects
> correctness, and this one feeds `WorkspaceRoot`, which feeds the module identity, which feeds the
> seed. **Traced inert today, by both the controller and the evaluator**: in `NewRunner` the
> computed root is dead (it passes a hardcoded empty `moduleIdentity`, and `WorkspaceRoot` is read
> only by `Validate` and `ResolveModuleIdentity`, neither of which `NewRunner` calls); in
> `RunTestsFromFile` the only case where it could matter — a relative input plus a failing
> `os.Getwd` — yields a non-absolute root that `Validate` then rejects loudly. So it is not a
> correctness bug **at HEAD**. It is still a fallback on a seed-determining value, and **M2C builds
> directly on both wrappers**, so fix it here rather than inheriting it further: return the error
> from `RunTestsFromFile`, and in `NewRunner` — whose D5 signature cannot return one — either drop
> the dead root computation or comment why it is dead. Do not merely re-verify that it is inert;
> "currently unreachable" is the state that rots.

**(a) `cmd/ailang/main.go`, `case "test"` (lines 111-136).** Add to `testFlags`:

```go
seedFlag       := testFlags.Int64("seed", 0, "Master seed for property generation (signed int64)")
randomSeedFlag := testFlags.Bool("random-seed", false, "Read one master seed from crypto/rand and report it")
```

After `testFlags.Parse`, detect *presence* with `Visit` (D1), then:

```go
if seedSet && randomSet {                       // D2
    fmt.Fprintln(os.Stderr, "Error: --seed and --random-seed cannot be used together")
    os.Exit(2)
}
cwd, err := os.Getwd()                          // captured ONCE, before any path walking
if err != nil { fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err); os.Exit(1) }
cfg := ailangTesting.TestConfig{WorkspaceRoot: cwd, SeedMode: ailangTesting.SeedModeDerived, MasterSeed: 0}
switch {
case seedSet:   cfg.SeedMode, cfg.MasterSeed = ailangTesting.SeedModeMaster, *seedFlag
case randomSet:
    m, err := ailangTesting.NewRandomMasterSeed()
    if err != nil { fmt.Fprintf(os.Stderr, "Error: %v\n", err); os.Exit(1) }   // BEFORE any property runs
    cfg.SeedMode, cfg.MasterSeed = ailangTesting.SeedModeMaster, m
}
```

`cfg` is then passed to `runPackageTests` / `runTestsV2`.

**(b) `cmd/ailang/test.go`.** Thread `cfg ailangTesting.TestConfig` through
`runTestsV2` (line 17), `runPackageTests` (line 136) and `runTestFile` (line 93); `runTestFile:119`
switches from `RunTestsFromFile(filename, file)` to `RunTestsFromFileWithConfig(filename, file, cfg)`.
**In BOTH aggregation blocks**, immediately after `NewSuiteResult(...)` (line 48 and line 185), call
`aggregateResults.SetSeedMetadata(cfg)`. Missing either one silently emits an empty `seed_mode` for
that mode — this is the §5.1 fourth call-site class.

**(c) `printTestHelp` (`cmd/ailang/test.go:225-246`).** Add two Options lines. The literal strings
`--seed N` and `--random-seed` **must appear verbatim** — AC6 greps for them with `grep -F`:

```
  --seed N           Master seed for property generation (signed int64; replayable)
  --random-seed      Read one master seed from crypto/rand and report it for replay
```

**(d) `internal/testing/reporter.go`.** In `reportJSON` (line 47) add three top-level keys:
`"seed_mode": string(result.SeedMode)`, `"seed": strconv.FormatInt(result.MasterSeed, 10)` — a
**decimal string**, not a number — and `"seed_derivation": result.SeedDerivation`. In
`formatPropertiesJSON` (line 85) always add `"seed": strconv.FormatInt(prop.Seed, 10)`, and when
`prop.Status == StatusFail` also `"replay"` (D9). In `reportSummaryHuman` (line 213) print mode,
master, derivation version and the replay command (D10).

### 5.6 M2 test plan

New file `internal/testing/config_test.go` (M2A), additions to
`internal/testing/runner_seed_test.go` (new, M2B) and `internal/testing/reporter_test.go` (M2C).

| ID | Test | Asserts | Kills which mutation |
|---|---|---|---|
| S1 | `TestDeriveSeedV1_GoldenVectors` | ≥3 hard-coded `(master, identity, property) -> int64` triples, written as literals | any change to framing/hash/byte-order silently altering v1 |
| S2 | `TestDeriveSeedV1_ZeroAndNegativeMaster` | master `0`, `-1`, `math.MinInt64` all derive without panic and differ from each other | treating 0 as “unset” |
| S3 | `TestDeriveSeedV1_IdentityFieldsAreFramed` | `derive(0,"ab","c") != derive(0,"a","bc")` | dropping the `\x00` separators |
| S4 | `TestResolveModuleIdentity_DeclaredModuleWins` | non-empty `declaredModule` returned verbatim; `workspaceRoot` ignored | reading the path when a module is declared |
| S5 | `TestResolveModuleIdentity_ModulelessIsWorkspaceRelative` | root `/w`, input `/w/cases/s.ail` → `cases/s.ail`; **and** relative input `cases/s.ail` with `os.Chdir`-free `filepath.Abs` semantics resolves identically | D3 — the doc's literal `Rel(root, inputPath)` |
| S6 | `TestResolveModuleIdentity_ErrorsAreLoud` | unrelated root (e.g. root `/w`, abs input on another volume prefix) returns a **non-nil error** and empty string | a silent `""` or basename fallback |
| S7 | `TestTestConfig_Validate` | empty root, relative root, and unknown `SeedMode` each error; a valid config does not | validation that always returns nil |
| S8 | `TestNewRandomMasterSeed_InjectedFailure` | `EntropyReader = iotest.ErrReader(...)` → non-nil error, seed `0`, **and no clock read** | any fallback path |
| S9 | `TestNewRandomMasterSeed_ShortRead` | a 4-byte reader errors (`io.ReadFull` semantics) rather than zero-padding | partial-read acceptance |
| S10 | `TestNewRNG_HasNoWallClockBranch` | `newRNG(0)` called twice yields the **same** first `Int63()`; `newRNG(1)` differs | leaving the `seed == 0` sentinel in place |
| S11 | `TestRunner_AllThreePropertyPathsUseDerivedSeed` | one fixture with a `forall`, a `requires` and an `ensures` property, run **twice** through `RunTestsFromFileWithConfig`; every `PropertyResult.Seed` is non-zero, the three differ from each other, and both runs agree field-for-field | §5.1: guarding two sites and missing `contract_domain.go:89` |
| S12 | `TestRunner_MasterSeedChangesEveryStream` | same fixture at master `0` vs `42`: all three `Seed` values differ | a hard-coded or ignored master |
| S13 | `TestRunProperty_SeedSetEvenOnSkip` | a `no_generator` property (reuse `TestRunContractProperties_SkipKinds`' fixture shape) still has `Seed != 0` | D8 — setting the seed after the early return |
| S14 | `TestSuiteResult_SetSeedMetadata` | copies all three fields from a `TestConfig` | a partial copy |
| S15 | `TestReporter_JSONSeedFieldsAreDecimalStrings` | top-level `seed` and per-property `seed` decode as **strings** matching `^-?[0-9]+$`; `seed_derivation == "ailang-property-seed-v1"`; a negative master round-trips exactly | emitting a JSON number (float64 precision loss above 2^53) |
| S16 | `TestReporter_ReplayOnlyOnFailure` | `replay` present on a `fail` property, **absent** on pass/skip, and equal to `ailang test --seed <master> <module_path>` | emitting `replay` unconditionally (would break AC6 byte-identity only if it varied — assert the text) |

### 5.7 M3 — integration, CLI end-to-end, and repository gates

**~280 net LOC (must-ship ~170).**

1. **New `cmd/ailang/test_seed_e2e_test.go` (~150)** — builds/locates the binary the same way
   `cmd/ailang/test_json_output_test.go` already does (read it first; do not invent a second
   harness). Covers: default determinism (AC5-M2), `--seed`/`--random-seed` mutual exclusion on
   **stderr** with exit 2, random→explicit replay byte-identity (AC6-M2), and `--help` containing
   both literals.
2. **New `cmd/ailang/test_contract_domain_test.go` (~110)** — the file §3.4 deferred from M1 and Q11
   confirms was never written. CLI-level JSON assertions for AC1/AC2/AC3 over the four checked-in
   fixtures in `internal/testing/testdata/contracts/`. **Descopable** (§5.9).
3. **Config-propagation coverage for all three entry modes** — direct (`RunTestsFromFileWithConfig`),
   ordinary (`runTestsV2`), package (`runPackageTests`). The package-mode arm needs a temp package
   with an `ailang.toml` plus one `*_test.ail`; assert the JSON top-level `seed_mode`/`seed` are
   present there too (this is the arm that catches a missed `SetSeedMetadata` in `runPackageTests`).
4. **`changelogs/v0.18-current.md`** — entry under Unreleased → Fixed naming #535, the new flags, the
   additive JSON fields, and the three intentional incompatibilities from the design doc
   (out-of-contract ensures failures became discards; default samples change once then stabilise;
   `GenConfig.Seed == 0` now means exact zero). Root `CHANGELOG.md` is an index — `make check-changelog`
   enforces that; do not write the entry there.
5. **Record honestly** that `make verify-examples` passes the `run` subcommand
   (`scripts/verify_examples.go:104-115`) and is therefore **not** evidence any contract property ran.

### 5.8 Descope valve (bounded execution)

Ship in this order and stop only at a numbered boundary:

1. **Must**: M2A complete (§5.3) + S1–S9.
2. **Must**: M2B complete (§5.4) **including all three RNG sites** + S10–S14.
3. **Must**: M2C (a)(b)(c)(d) (§5.5) + S15–S16 + AC5-M2, AC6-M2, AC9-M2, AC-SEED-SWEEP-M2.
4. **Should**: M3 items 1, 3, 4.
5. **May defer**: M3 item 2 (`test_contract_domain_test.go`) to a follow-up.

**Never stop between 2 and 3.** After step 2 the wall clock is gone but nothing reports a seed, so a
failure cannot be replayed — the exact half-state the design doc's *Rollback* section calls unsafe.
If time runs out inside step 3, finish (d) at minimum: the JSON fields are what make step 2 useful.

### 5.9 Landing shape and rollback

M2A/M2B/M2C/M3 are **commit boundaries inside one PR**, not four landable PRs. The PR is landable
only from step 3 of §5.8 onward. Rollback is `git revert` of the whole M2+M3 series; per the design
doc, keeping M2 while dropping M1 is the unsafe direction, and M1 is already landed in
`a9e26ffd6`, so it is not at risk here.

`Fixes #535` goes on the **final** commit only. `#547` is already CLOSED (Q8) — it must **not**
appear with any closing keyword.

### 5.10 Files touched by M2 + M3

| File | Milestone | Change | LOC |
|---|---|---|---:|
| `internal/testing/config.go` (NEW) | M2A | `TestConfig`, `SeedMode`, `SeedDerivationV1`, `DeriveSeedV1`, `ResolveModuleIdentity`, `NewRandomMasterSeed`, `EntropyReader`, `Validate` | +120 |
| `internal/testing/config_test.go` (NEW) | M2A | S1–S9 | +110 |
| `internal/testing/runner.go` | M2B | `Runner` fields, `NewRunnerWithConfig`, `NewRunner` wrapper, `propertySeed`, 2 RNG sites, `newRNG` wall clock removed, `runProperty` seed stamp, `RunTestsFromFileWithConfig` | +55 / −5 |
| `internal/testing/contract_domain.go` | M2B | 1 RNG site (**the one the doc omits**) | +1 / −2 |
| `internal/testing/result.go` | M2B | `PropertyResult.Seed`; `SuiteResult.{SeedMode,MasterSeed,SeedDerivation}`; `SetSeedMetadata` | +18 |
| `internal/testing/runner_seed_test.go` (NEW) | M2B | S10–S14 | +85 |
| `cmd/ailang/main.go` | M2C | two flags, `Visit` detection, conflict exit 2 on stderr, single `os.Getwd`, entropy read, config construction | +45 |
| `cmd/ailang/test.go` | M2C | config threading through 3 funcs, `SetSeedMetadata` at **both** aggregates, configured entry point, help text | +55 |
| `internal/testing/reporter.go` | M2C | 3 top-level keys, per-property `seed`, conditional `replay`, human seed/replay block | +45 |
| `internal/testing/reporter_test.go` | M2C | S15–S16 | +55 |
| `cmd/ailang/test_seed_e2e_test.go` (NEW) | M3 | AC5/AC6 end-to-end + help literals | +150 |
| `cmd/ailang/test_contract_domain_test.go` (NEW, descopable) | M3 | M1's deferred AC1/AC2/AC3 CLI e2e | +110 |
| package-mode propagation test | M3 | temp `ailang.toml` package, asserts top-level seed fields | +40 |
| `changelogs/v0.18-current.md` | M3 | Unreleased → Fixed | +15 |

**Net ≈ +850 (must-ship ≈ +700).** File-size headroom after M2+M3 (cap 800, Q5/Q6):
`runner.go` 670→~720, `main.go` 526→~571, `cmd/ailang/test.go` 246→~301, `reporter.go` 296→~341,
`contract_domain.go` 194→~193. **No file approaches the cap; no relocation commit is needed.**

**Velocity**: §1's sustained rate is 350–450 net LOC/day for test-heavy Go here, so M2+M3 is
**≈ 2.0–2.4 days**. The design doc estimates M2 at ~0.5 d and M3 at ~0.25–0.5 d; that is
**too low by roughly 2x** — it predates M1 landing 670 insertions against a whole-feature estimate of
650–830, and it does not account for the fourth call-site class (§5.1), the entropy-injection
seam (D7) or the deferred M1 e2e file (Q11). Recorded here rather than force-fitted.

### 5.11 M2 / M3 acceptance criteria — restated runnable, each with its pristine-`dev` baseline

**Rule applied throughout**: an AC that already passes on unmodified `dev` proves nothing. Every arm
below is annotated with what it does *at baseline*, measured 2026-08-07. Arms marked **VACUOUS** were
in the design doc and are replaced or supplemented here.

All commands run from the worktree root after `make build`.

#### AC5-M2 — default and exact-seed results reproduce (design-doc AC5, repaired)

```bash
set -u
tmp=$(mktemp -d)
ex=examples/runnable/contracts/list_recursive_verify.ail
for run in a b c; do
  ./bin/ailang test --format json --no-color "$ex" >"$tmp/$run.json" 2>"$tmp/$run.err"
  printf '%s\n' "$?" >"$tmp/$run.rc"
  jq 'del(.total_duration) | .tests |= map(del(.duration)) | .properties |= map(del(.duration))' \
     "$tmp/$run.json" >"$tmp/$run.norm"
  jq -S -c '[.properties[] | {name,status,skip_kind,tests_run,generated_inputs,discarded_inputs}]' \
     "$tmp/$run.json" >"$tmp/$run.shape"
done
cmp "$tmp/a.rc"    "$tmp/b.rc"     # (i)
cmp "$tmp/a.shape" "$tmp/b.shape"  # (ii)  <-- the discriminating arm
cmp "$tmp/a.shape" "$tmp/c.shape"  # (ii)
cmp "$tmp/a.norm"  "$tmp/b.norm"   # (iii)
jq -e '.seed_mode == "derived" and .seed == "0"
       and .seed_derivation == "ailang-property-seed-v1"
       and ([.properties[] | select(.seed != null)] | length > 0)' "$tmp/a.json"   # (iv)
```

**Baseline on pristine `dev`** (5 runs measured):
- (i) **VACUOUS** — `rc=1` in all 5 runs, so `cmp a.rc b.rc` already passes. M1 stabilised the
  verdict *class* of this file: 0 passed / 0 failed / 6 skipped every time, and `Success()` requires
  `passed+failed > 0`, so the suite exits 1 permanently. **Keep the arm, but it is not evidence.**
- (ii) **DISCRIMINATES — this is the arm that replaces (i)**. `tests_run` is still nondeterministic:
  `containsImpliesNonEmpty_property_2` measured **31, 16, 29, 26, 25** across five runs and
  `extractBounded_property_1` measured **1, 3, 5, 5, 3**. This is #535, live, at HEAD, *after* M1.
- (iii) DISCRIMINATES (`cmp` of the normalised JSON returns 1 at baseline).
- (iv) DISCRIMINATES (`seed_mode` is absent → `jq -e` returns 1).

**Note (Q12)**: this fixture declares `module examples/runnable/contracts/list_recursive_verify`, so
it exercises the *declared-module* identity branch. It says nothing about `WorkspaceRoot` — only
AC9-M2 does.

#### AC6-M2 — multi-property random invocation replays exactly (design-doc AC6, **REPAIRED — see §5.12 D-2 and D-3**)

```bash
set -u
tmp=$(mktemp -d)
cat >"$tmp/multi.ail" <<'EOF'
module random_replay_multi

export func ok(x: int) -> bool ! {}
ensures { result == true } { true }

export func broken(x: int) -> bool ! {}
ensures { result == true } { false }
EOF

./bin/ailang check --relax-modules "$tmp/multi.ail"                  # (a)  REPAIRED
./bin/ailang test --help | grep -F -- '--seed N'                     # (b)
./bin/ailang test --help | grep -F -- '--random-seed'                # (b)

./bin/ailang test --seed 1 --random-seed "$tmp/multi.ail" >"$tmp/conflict.out" 2>"$tmp/conflict.err"
rc=$?; echo "conflict rc=$rc"
test "$rc" -eq 2                                                     # (c)  VACUOUS at baseline
grep -F 'cannot be used together' "$tmp/conflict.err"                # (d)  the discriminating arm

./bin/ailang test --random-seed --format json --no-color "$tmp/multi.ail" \
  >"$tmp/random.json" 2>"$tmp/random.err"
random_rc=$?
seed=$(jq -er '.seed | select(type == "string") | select(test("^-?[0-9]+$"))' "$tmp/random.json")
test "$(jq '[.properties[] | select(.seed != null)] | length' "$tmp/random.json")" -ge 2   # (e)

./bin/ailang test "--seed=${seed}" --format json --no-color "$tmp/multi.ail" \
  >"$tmp/replay.json" 2>"$tmp/replay.err"
replay_rc=$?

for run in random replay; do
  jq 'del(.total_duration) | .tests |= map(del(.duration)) | .properties |= map(del(.duration))' \
     "$tmp/$run.json" >"$tmp/$run.norm"
done
test "$random_rc" -eq "$replay_rc"                                   # (f)
cmp "$tmp/random.norm" "$tmp/replay.norm"                            # (g)
jq -e --arg s "$seed" '.seed_mode == "master" and .seed == $s
       and .seed_derivation == "ailang-property-seed-v1"' "$tmp/replay.json"   # (h)
```

**Baselines on pristine `dev`:**
- (a) **REPAIRED — and the doc's version is worse than "broken", it is *environment-dependent*.**
  The design doc's `./bin/ailang check "$tmp/multi.ail"` **exits 0 here**, not 1: `mktemp -d` on this
  machine returns `/var/folders/…/T/tmp.XXXX`, `loader.IsTempPath` matches it
  (`internal/loader/loader.go:626`, used at `internal/pipeline/pipeline_module.go:740-741`), and
  MOD010 is auto-relaxed to a **WARNING**. Measured: `rc=0`, `✓ No errors found!`. The *same* fixture
  in a non-temp directory (`$HOME/...`) gives `rc=1` and
  `Error MOD010: module 'random_replay_multi' doesn't match file path '…'` — that is the
  known-positive control proving the instrument works. So this arm passes or fails **depending on
  `$TMPDIR`**. `--relax-modules` makes it deterministic in both: measured `rc=0` under `/var/folders`,
  under `/tmp`, and under `$HOME`.
- (b) DISCRIMINATES: `rc=1` for both greps at baseline (Q7).
- (c) **VACUOUS**: measured `rc=2` on pristine `dev` — Go's `flag.ExitOnError` already exits 2 on
  `flag provided but not defined: -seed`. Keep the arm; it is not evidence.
- (d) DISCRIMINATES: `grep rc=1` at baseline; stderr says `flag provided but not defined: -seed`.
  **This is why D2 pins the message to stderr and pins the substring.**
- (e)(g)(h) DISCRIMINATE: at baseline `--random-seed` exits 2 with an empty `random.json`.
- **Substance CONFIRMED**: `./bin/ailang test --format json --no-color "$tmp/multi.ail"` on this
  fixture already returns `rc=1` with `ok_property_1 pass tests_run=100` and
  `broken_property_1 fail tests_run=1` — two properties, failing suite, exactly as AC6 assumes.
- `--seed=${seed}` uses the `=` form deliberately: the master can be negative and the `=` form is
  unambiguous.

#### AC9-M2 — derived seed is independent of the absolute checkout path (**REPAIRED — see §5.12 D-4**)

```bash
set -u
bin=$(pwd)/bin/ailang
tmp=$(mktemp -d)
mkdir -p "$tmp/machine-a/repo/cases" "$tmp/machine-b/repo/cases"
cat >"$tmp/machine-a/repo/cases/stable.ail" <<'EOF'
export func stable(x: int) -> bool ! {}
ensures { result == true } { true }
EOF
cp "$tmp/machine-a/repo/cases/stable.ail" "$tmp/machine-b/repo/cases/stable.ail"
(cd "$tmp/machine-a/repo" && "$bin" test --format json --no-color cases/stable.ail) >"$tmp/a.json"
(cd "$tmp/machine-b/repo" && "$bin" test --format json --no-color cases/stable.ail) >"$tmp/b.json"
jq -e '.seed_mode == "derived"' "$tmp/a.json"
jq -e '.seed_mode == "derived"' "$tmp/b.json"
# REPAIRED: select(type=="string") — a missing key yields the STRING "null" under `jq -r`
a=$(jq -er '.properties[] | select(.name=="stable_property_1") | .seed | select(type=="string")' "$tmp/a.json")
b=$(jq -er '.properties[] | select(.name=="stable_property_1") | .seed | select(type=="string")' "$tmp/b.json")
test -n "$a"
test "$a" = "$b"
# NEGATIVE CONTROL: the derivation must not be a constant.
c=$(jq -er '.properties[] | select(.name=="stable_property_1") | .seed | select(type=="string")' \
    <((cd "$tmp/machine-a/repo" && "$bin" test --seed=7 --format json --no-color cases/stable.ail)))
test "$a" != "$c"
```

**Baseline on pristine `dev`:**
- The fixture itself **works** — §0.3's blocker really is discharged. Re-measured, not inherited:
  `rc=0`, one property `stable_property_1 pass tests_run=100`, `location` already
  `cases/stable.ail:2:11` (workspace-relative). Note *why* it survives where AC6 does not: AC9 never
  runs `ailang check`. A module-less file fails `check` with `MOD014: no 'module' declaration`
  (measured), but `ailang test` synthesises a temp module — stderr shows
  `WARNING MOD010 (temp-path): module '_test/stable' …  Auto-relaxed`.
- **The doc's own assertions are 2/5 VACUOUS.** `jq -r` prints the four-character string `null` for a
  missing key, so `test -s "$tmp/a.seed"` **passes at baseline** (measured: `od -c` shows
  `n u l l \n`, `test -s` → rc 0) and `cmp a.seed b.seed` **also passes** (both files are `null`).
  Only `jq -e '.seed_mode == "derived"'` discriminates. Verified fix: with
  `select(type=="string")`, `jq -er` returns **rc=4** on the null/absent case and **rc=0** with the
  value on the string case.
- **§5.3's instruction to “keep AC9 exactly as written” is therefore REFUTED** on runnability
  (not on substance — the fixture and the claim are both fine).
- The `--seed=7` negative control is new: without it, a derivation hard-coded to a constant passes
  every other arm.

#### AC-SEED-SWEEP-M2 — every RNG path went through the derivation (§5.1's failure shape)

**REPAIRED at iteration 162 — (b) and (c) as originally written are unrunnable, and (d) is
vacuous for the defect it names. Both repairs are measured, not reasoned; see the two notes below.**

```bash
PROD=($(ls internal/testing/*.go | grep -v '_test\.go$'))                  # production files only
grep -rn 'time\.Now()\.UnixNano' internal/testing/ ; test $? -ne 0        # (a)
test "$(grep -nE 'newRNG\(' "${PROD[@]}" | grep -vc 'func newRNG')" -eq 3  # (b)
test "$(grep -nE 'newRNG\(' "${PROD[@]}" | grep -v 'func newRNG' \
  | grep -c 'propertySeed')" -eq 3                                         # (c)
go test ./internal/testing -run 'TestRunner_AllThreePropertyPathsUseDerivedSeed' -count=1  # (d)
go test ./internal/testing -run 'TestRunner_DerivedSeedDrivesSampleStreams' -count=1       # (e)
```

**Baseline**: (a) fails (the sentinel is at `runner.go:667`); (b) **passes at baseline** (there are
already exactly 3 sites) — it is a *count guard* against a fourth appearing, not evidence; (c) fails
(zero sites mention `propertySeed`); (d) and (e) fail (tests do not exist). (c) and (e) are the
discriminating arms, and **(e)** — not (d) — is the one that catches `contract_domain.go:89`.

**Repair 1 — the scope was wrong, and the executor caught it (self-reported deviation, adjudicated
by measurement).** The original arms glob `internal/testing/*.go`, which includes `_test.go`. S10's
own spec (§5.6) requires it to call `newRNG(0)`/`newRNG(1)` directly, so the moment S10 exists the
`-eq 3` guard reads **6**; with S11b's doc comment quoting the call form it reads **8**, and (c)
reads **4**. Measured at iteration 162: as-written (b)=8 / (c)=4 → both FAIL on a correct tree;
production-scoped (b)=3 / (c)=3 → both PASS. The guard is about *production* call sites, so the
glob, not the count, was the defect.

**Repair 2 — (d) does not kill the mutation it is documented to kill.** S11 observes
`PropertyResult.Seed`, which each path stamps into its result initializer **independently of what it
hands to `newRNG`**. Measured at iteration 162 with the mutant landed and building
(`go build ./internal/testing` rc=0): replacing `newRNG(r.propertySeed(...))` with a constant at
`contract_domain.go:89` leaves **the entire seed suite green**. §5.6's S11 row therefore pins the
stamp, not the stream, and the milestone's whole point — the sweep — was unguarded. S11b
(`TestRunner_DerivedSeedDrivesSampleStreams`) closes it by observing facts downstream of the
generator: the ensures path's accept/discard counts under M1's domain filter, and the requires
path's counterexample text. Both are exact and reproducible (fixed seeds ⇒ fixed samples), and both
mutants now red. **Residual, recorded rather than papered over**: the *forall* site
(`runner.go:293`) has no stream observable in this package — every forall property in a unit-test
fixture fails on its first generated input with `evaluation failed: empty program`, a pre-existing
harness limitation unrelated to this milestone — so a constant-seed mutant there still survives.
That site is pinned by arm (c) plus the seed stamp only. M3 owes it a CLI-level e2e arm.

#### AC-SCOPE-M2 — no scope creep

```bash
git diff --stat -- internal/parser internal/types internal/elaborate internal/eval \
                   internal/core internal/pipeline internal/link internal/effects   # must be EMPTY
git diff --stat -- internal/testing/executor.go internal/testing/harness.go \
                   internal/testing/source_strip.go                                  # must be EMPTY
grep -n 'requiredAccepted = 100' internal/testing/contract_domain.go                 # M1's bounds intact
grep -n 'maxAttempts      = 1000' internal/testing/contract_domain.go
```

**Baseline**: the two `git diff --stat` arms are empty on a clean tree (vacuous until the executor
edits — they are *stop* conditions checked at the end, not evidence of progress). The two greps pass
at baseline and must **still** pass at the end: M2 must not touch M1's frozen 100/1000 bounds.

#### AC7-M3 — repository gates (design-doc AC7, with its measured baseline)

```bash
go test ./internal/testing/... -count=1
go test ./cmd/ailang/... -count=1
make lint
make verify-examples
```

**Baseline on pristine `dev`, measured 2026-08-07: all four rc=0** (Q9). This is therefore a
legitimate “did I break anything” gate.

**Expected pre-existing warning noise in `make verify-examples` — 6 lines, not 1.** An executor told
to expect one will report five false regressions:

```
⚠ bugs/concat_operator_list_inference.ail: unparseable (covered by verify-examples): PAR_IMPORT_PLACEMENT …
⚠ experimental/ai_agent_integration.ail: unparseable … PAR_UNEXPECTED_TOKEN …
⚠ experimental/concurrent_pipeline.ail: unparseable … PAR_NO_PREFIX_PARSE …
⚠ experimental/factorial.ail: unparseable … expected => after forall binders
⚠ experimental/web_api.ail: unparseable … IMP012_UNSUPPORTED_NAMESPACE …
⚠ lambda_expressions.ail: file not found on disk (stale manifest entry)
```

The run still ends `186 modules checked, 0 drift, 1 missing-on-disk` and
`✅ verify-examples: all examples pass and manifest is in sync`, rc=0. **None of these is this
sprint's.** `make lint` similarly reports `1 issues: * unused: 1` and still exits 0.

`make verify-examples` passes the **`run`** subcommand (`scripts/verify_examples.go:104-115`), so a
green result is **not** evidence that any contract property executed. Record it that way.

#### AC-DOC-SEED-M3 — the `--seed 42` forms of AC1 / AC3 / AC8 return (design doc, D11)

Re-run §4's `AC1-M1`, `AC3-M1` and `AC8-M1` commands with `--seed=42` inserted, and additionally
assert that the seeded and unseeded runs of `requires_unreachable.ail` agree on
`generated_inputs == 1000 and discarded_inputs == 1000` (that fixture accepts with p = 0, so it is
seed-invariant by construction — §0.4).

**Baseline**: fails; `--seed` does not exist (Q7).

### 5.12 Divergences — which document wins, per item

Per the controller's standing rule: **the design doc wins on design; a measured refutation wins on
AC runnability.**

| # | Divergence | Established by | Winner |
|---|---|---|---|
| D-1 | Doc's *Conflict Surface* says all three RNG sites are in `runner.go`. They are at `contract_domain.go:89`, `runner.go:261`, `runner.go:384`. | `grep -rnE 'newRNG\(' internal/testing/*.go` | **HEAD wins.** M1 moved the ensures path; the doc predates `a9e26ffd6`. §5.4 sweeps both files. |
| D-2 | Doc AC6 line 1 is `./bin/ailang check "$tmp/multi.ail"`. | Measured `rc=0` under `mktemp -d` (temp-path auto-relax) and `rc=1` under `$HOME` — i.e. **`$TMPDIR`-dependent**, not simply broken | **Measurement wins on runnability.** Repaired to `check --relax-modules`, measured `rc=0` in all three locations. Doc's *intent* (the fixture type-checks) is preserved. |
| D-3 | Doc AC6 asserts `test "$rc" -eq 2` for the flag conflict. | Measured `rc=2` on pristine `dev` (`flag.ExitOnError` on an unknown flag) | **Measurement wins.** Arm kept but labelled vacuous; the `grep -F 'cannot be used together'` arm carries the criterion, and D2 pins the message to **stderr**. |
| D-4 | §5.3 of this plan said “keep AC9 exactly as written”. | `jq -r` emits the literal string `null` for a missing key, so `test -s` (rc 0) and `cmp` both pass at baseline; only `seed_mode` discriminates | **Measurement wins on runnability; §5.3 wins on substance.** The *fixture* is unblocked as §5.3 says; the *assertions* are 2/5 vacuous and are repaired with `select(type=="string")` plus a `--seed=7` negative control. |
| D-5 | Doc's module-less identity formula is `filepath.Rel(config.WorkspaceRoot, inputPath)`. | `WorkspaceRoot` is absolute (`os.Getwd`) and AC9 passes a **relative** input (`cases/stable.ail`); `filepath.Rel(abs, rel)` errors | **Doc wins on design, measurement wins on the call.** Same semantics, one added `filepath.Abs` (D3). Covered by S5. |
| D-6 | Doc's *Files to Modify* omits `cmd/ailang/test.go`'s two aggregation blocks. | Read `cmd/ailang/test.go:50-63` and `:190-203`; the top-level JSON comes from the **aggregate** `SuiteResult` | **Additive to the doc.** §5.5(b) + `SetSeedMetadata` (D8/§5.4 step 9). |
| D-7 | Doc estimates M2 ≈ 0.5 d, M3 ≈ 0.25–0.5 d, whole feature 650–830 LOC. | M1 alone landed **670 insertions / 136 deletions** (`git show --stat a9e26ffd6`); §5.10 puts M2+M3 at ≈850 net | **Measurement wins.** Revised to ≈2.0–2.4 d. The doc's *scope* is unchanged; only the estimate is. |
| D-8 | §6 pins the worktree to `/tmp/wt-m-property-seed-determinism` at base `386cf6d15`. | `/tmp`-rooted checkouts red `TestIsTempPath` (`internal/loader`) and `TestSolve_HardTimeout_FakeSolverIgnoringT` (`internal/smt`) for the **location**, not the code; base is 226 non-`design_docs` files stale | **Controller wins.** §6 updated to `/Users/voightkampff/dev/sunholo-data/.wt-iter159` at `086902493`. |
| D-9 | §6 carries codex sandbox language (“UNINFORMATIVE UNDER SANDBOX”). | The iteration-159 executor is `pi:…deepseek-v4-flash-0731`, which has **no sandbox** | **Controller wins.** §6 rewritten; a gate result from this lane is real. |
| D-10 | Doc: "The Runner retains the config and passes its master **and workspace root** to the single seed-derivation helper." | — | **Doc wins on design.** The Runner does retain the whole `TestConfig`. D4 only says `WorkspaceRoot` is *consumed* once, at identity resolution, before the Runner is built — that is a narrowing, not a contradiction. |

### 5.13 (historical) What "M1 only" left in the tree — precisely

**Present and complete after M1:**
- `internal/testing/contract_domain.go` with the full requires filter; `runner.go` at 669 lines.
- `PropertyResult.GeneratedInputs` / `.DiscardedInputs`, populated on **all three** property paths.
- JSON keys `generated_inputs` / `discarded_inputs` on every property object.
- Human output printing `accepted A, discarded D, generated G`.
- A new `out_of_contract` skip carrying `unverified: ...` with exact counts.
- New fixtures under `internal/testing/testdata/contracts/`.

**Absent after M1 — and absent *cleanly*, no stubs:**
- No `--seed`, no `--random-seed`. `ailang test`'s flag set is unchanged.
- No `internal/testing/config.go`, no `TestConfig`, no `WorkspaceRoot`, no `SeedMode`, no
  `MasterSeed`, no `seed_derivation`. **Not created, not stubbed, not referenced.**
- `newRNG`'s wall-clock branch (`runner.go:786-788`) is **untouched**. #535 stays open. Default
  property samples still vary run to run.
- No `seed` / `seed_mode` JSON fields.
- `generator.go` unchanged; `DefaultConfig().Seed` is still 0.

**Is the tree coherent and strictly better?** Yes:
- Green: AC4-M1 proves no existing test needed editing; AC12-M1 covers the repo gates.
- Coherent: no field, flag, or file exists that nothing reads or writes.
- Strictly better: the vacuous-failure class (#547) is closed with an honest skip taxonomy, and the
  most visible symptom — `list_recursive_verify.ail` reporting a false `containsImpliesNonEmpty(xs=[])`
  violation — stops occurring. #535's residual nondeterminism is *sampling* variation, which is what
  it was before; M1 removes none of the seed machinery because it adds none.

**Rollback of M1 alone**: `git revert` the M1 commits. Self-contained — it touches no CLI surface and
no shared config. The design doc's "partial rollback is unsafe" warning applies to the *M1+M2*
combined state, and specifically to keeping M2 while dropping M1. Dropping M2 while keeping M1 is
the safe direction and is exactly what tonight ships.

---

## 6. Executor operating notes

> **§6 was rewritten by the controller at iteration 159, 2026-08-07.** §5.12 rows **D-8** and
> **D-9** asserted this section had already been updated; it had **not** — the iteration-159
> planner fire wrote the divergence table and ended before applying the two edits it promised
> there. Verified at the point of the fix: `sed -n '1198,1214p'` on the pre-fix file still matched
> `/tmp/wt-`, `386cf6d15`, `codex sandbox` and `Codex cannot`. That is Gate-2 rule 3b(vi) in its
> exact form — a document's own log refuting its operative section — and it was load-bearing,
> because §6 is the one section an executor reads to learn *where to work*. The M1-era text below
> is preserved only where it is still true. §7 and §8 are **historical M1 records**; they are not
> operative for M2/M3, whose acceptance criteria are §5.11 and whose descope valve is §5.8.

- **Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter159`, branch
  `sprint/iter159-property-seed-m2m3`, base `086902493` (= `origin/dev` at spawn time).
  **NEVER `/tmp`** — a `/tmp`-rooted checkout reds `TestIsTempPath` (`internal/loader`) and
  `TestSolve_HardTimeout_FakeSolverIgnoringT` (`internal/smt`) for the **location**, not the code,
  and CI never reproduces that red. A sibling of the repo is the standing convention.
- The design doc `design_docs/planned/v0_31_0/m-property-seed-determinism.md` is **tracked on
  `dev`** as of `01c36db8d`, so the M1-era "untracked, do not `git add`" note no longer applies.
  Do not edit the design doc; it is the reviewed artifact and it wins on design (§5.12).
- **Never create branches in the main checkout.** All work stays in the worktree.
- **This lane is `pi:openrouter/deepseek/deepseek-v4-flash-0731`, which has NO sandbox.** The
  codex `UNINFORMATIVE UNDER SANDBOX` caveat does **not** apply: a gate result from this lane is
  real, and a loopback-bind failure here is a genuine failure, not an infrastructure denial.
  Because there is no sandbox, containment is the scope fence in this section plus the
  controller's diff review — the executor performs **no git write operations at all** (no `add`,
  `commit`, `stash`, `checkout`, `branch`) and touches nothing outside the worktree. The
  controller builds one commit per milestone from `.snap/M<k>/` snapshots.
- Commit messages (controller-authored): `refs #535` on development commits; `Fixes #535` on the
  **final** commit only. **`#547` is already CLOSED** (§5.0 Q8) and must **not** appear with any
  closing keyword.
- After every edit: `make fmt && go build ./... && make check-file-sizes`.
- **Do not** fix the named-test stripper bug (§0.2) or touch `internal/testing/executor.go`.

---

## 7. Blockers and risks

| # | Item | Severity | Disposition |
|---|---|---|---|
| B1 | AC1/AC3/AC8 use `--seed 42`, which M1 does not add | **was blocking** | RESOLVED — §4 restates them seed-free; probabilistic margins computed in §0.4 |
| B2 | AC3 fixture unparseable (`assert`) + named-test synthesizer breaks on contract files | **was blocking** | RESOLVED — §4 AC3-M1 uses a second contract function instead. Underlying bug routed OUT of scope; file a new issue at landing |
| B3 | AC9 fixture does not parse | blocking **for M2**, not M1 | ✅ RESOLVED 2026-08-01 (iter 127) — declaration-aware strip landed; AC9's original fixture passes, keep it as written (§0.3, §5.3) |
| R1 | `runner.go` 10 lines from the 800 cap | high | §3.1 pure-move-first, enforced by AC10-M1's `-le 700` |
| R2 | 1,000-attempt cap × sparse list domains is slow | medium | AC13-M1 wall-clock bound; do **not** lower the cap |
| R3 | Filtering hides a real failure | high | AC2-M1's `x=[1-9][0-9]{2,}` counterexample assertion + T4 |
| R4 | 99-accepted boundary silently passes | high | T6 mutation guard; explicit `accepted < 100` → skip |
| R5 | Some of the 27 `requires`-bearing examples change verdict | medium | Not CI-visible (P9), but AC11-M1 + AC12-M1 measure the flagship example. Any example that goes green→red is a **stop** |
| R6 | Sprint plan and design doc both target `design_docs/planned/v0_31_0/` on a branch based one commit behind `dev` | low | Documented in §6; only the plan + JSON are added |

**No blocker should stop M1 execution tonight.** B1 and B2 are resolved in this plan; B3 is a
future-milestone item.

---

## 8. Success metrics

- AC1-M1 … AC13-M1 all exit 0 (AC12 subject to the sandbox caveat).
- `make check-file-sizes` green; `internal/testing/runner.go` ≤ 700 lines.
- Zero diff in `generator.go`, `cmd/ailang/main.go`, `cmd/ailang/test.go`,
  `internal/testing/executor.go`, `internal/testing/harness.go`.
- `#547` closed; `#535` explicitly still open.
- CHANGELOG entry under Unreleased → Fixed, naming both the fix and the honest limitation.

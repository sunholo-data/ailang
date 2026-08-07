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

## 5. M2 and M3 — parked to the 2026-08-03 re-arm

Specified so the re-arm does not re-plan. **Do not execute tonight.**

### 5.1 M2 — seed policy (#535), ~280 net LOC, 0.6 d

Per the design doc §M2, unchanged: new `internal/testing/config.go` with
`TestConfig{WorkspaceRoot, SeedMode, MasterSeed}`; `derive_v1` = SHA-256 of
`"ailang-property-seed-v1\x00" + decimalMaster + "\x00" + moduleIdentity + "\x00" + propertyName`,
bytes 0:8 little-endian as signed int64; remove the `seed == 0` wall-clock branch from `newRNG`;
`--seed N` / `--random-seed` mutually exclusive (exit 2); one `crypto/rand` read; `os.Getwd()`
captured once in `cmd/ailang/main.go` before path walking.

**M2 file-size note**: `runner.go` will be 669 after M1, but M2 threads config through
`RunTestsFromFile`/`NewRunner` (both in `runner.go`). Keep the derivation helper in
`internal/testing/config.go`, not in `runner.go`.

### 5.2 M3 — integration gates, ~180 net LOC, 0.4 d

Design-doc AC5, AC6, AC7 verbatim, plus the `--seed 42` forms of AC1/AC3/AC8 restored.

### 5.3 M2/M3 blocker to resolve at re-arm — ✅ RESOLVED 2026-08-01, NOTHING OWED

**This blocker is discharged; M2 starts unblocked.** Iteration 127 landed the declaration-aware
strip (`internal/testing/source_strip.go`), and AC9's original module-less fixture now parses and
exits 0 — verified with a paired pre-fix/post-fix control (§0.3). **Keep AC9 as written**; do not
spend re-arm time producing a substitute fixture or restating it over a module-bearing file.
Regression cover: `internal/testing/testdata/strip/moduleless_contract.ail`.

Historical statement of the blocker, retained: AC9 was **unimplementable as written** (§0.3), and
M2 was not to start until it was resolved, it being the only criterion proving `WorkspaceRoot`
actually reached the derivation.

### 5.4 What "M1 only" leaves in the tree — precisely

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

- **Worktree**: `/tmp/wt-m-property-seed-determinism`, branch `sprint/m-property-seed-determinism`,
  base `386cf6d15`. `internal/` there is byte-identical to `01c36db8d`.
- **Design doc is UNTRACKED in the worktree** (`git status` shows `?? design_docs/planned/v0_31_0/
  m-property-seed-determinism.md`); the tracked copy landed on `dev` in `01c36db8d`. Content is
  identical (`shasum` = `c9f51414cf59dfe758d64a0bfab4da7aa8b61096` on both). **Do not `git add` it**
  — it would create a spurious both-added on merge. Only add this sprint plan and the sprint JSON.
- **Never create branches in the main checkout.** All work stays in the worktree.
- The codex sandbox **cannot bind loopback sockets**. Any test failure of that shape is
  **UNINFORMATIVE UNDER SANDBOX**; record it verbatim and let the controller re-run outside.
- Codex cannot `git commit` in a linked worktree; the controller finalizes commits.
- Commit messages: `refs #547` on development commits. Use `Fixes #547` only on the final commit
  **if and only if** M1 is judged to close it — M1 does close #547; **#535 stays open** and must
  **not** appear with a closing keyword.
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

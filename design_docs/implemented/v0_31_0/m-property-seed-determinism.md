# M-PROPERTY-TEST-TRUST: Deterministic and Sound Contract Verdicts

**Status**: Planned
**Target**: v0.31.0
**Priority**: P0 (prerequisite for Lane B1 property-generator coverage)
**Estimated**: 1.5–2 days
**Created**: 2026-07-31
**Revised**: 2026-07-31 (iteration 126 bounded final revision, quorum round 2)
**Dependencies**: [M-PROPERTY-GENERATOR-COVERAGE Lane A](m-property-generator-coverage.md) (landed in `a81d66983`); issues #535 and #547

---

## Problem Statement

`ailang test` verdicts are neither reproducible nor sound today. Two coupled defects make the same
unchanged contract alternate between pass and fail, and can label an input outside a function's
declared domain as an `ensures` violation.

### #535 — hidden wall-clock seed

Every live property path starts from `DefaultConfig().Seed == 0`, and `newRNG(0)` substitutes
`time.Now().UnixNano()`. Top-level `forall`, function `requires`, and function `ensures` therefore
generate different values on unchanged invocations. Fresh runs at HEAD `386cf6d15` reproduced
varying exit codes for `examples/runnable/contracts/list_recursive_verify.ail` (controller: 1,0,0,1,1;
this session: 0,0,0,1,0).

### #547 — `ensures` ignores `requires`

`runEnsuresProperty` (`internal/testing/runner.go:388-427` at the verified revision) generates
arguments, builds an ensures harness with `EvaluateEnsuresHarnessFromCore`, and evaluates the
postcondition. It never locates or evaluates the function's `requires` predicate and has no
discard/filter step.

The resulting verdict asymmetry is decisive:

- `runRequiresProperty` (`runner.go:540-543`) treats a generated input that violates `requires` as
  `skip`, with `requires not satisfied by random input (consider tighter generators)`. Its comment
  states that such inputs "aren't a function bug".
- `runEnsuresProperty` faces the identical precondition violation and reports `fail`.

The same generated-input condition therefore receives opposite verdicts. The existing skip side
already establishes the intended semantics: inputs excluded by `requires` are not counterexamples.
Reporting them as `ensures violated` is a **vacuous failure**, the mirror image of the vacuous-pass
class closed by #517 Lane A.

Deterministic minimal reproduction, measured six of six times on a fresh build of `386cf6d15`:

```ailang
module precond2
export func big(x: int) -> bool ! {}
requires { x > 100 }
ensures { result == true } {
  x > 100
}
export func main() -> int ! {} { if big(500) then 1 else 0 }
```

`ailang check` reports `No errors found!`. Six current `ailang test --format json` runs reported
`ensures violated` for `x=-416/-700/-522/-17/-527/-405`; every value is excluded by `x > 100`.
This session independently reproduced six of six failures with different excluded values. No input
satisfying `requires` can violate this postcondition.

The real instance answers the prior open question about
`examples/runnable/contracts/list_recursive_verify.ail`: the measured failure
`containsImpliesNonEmpty(xs=[], elem=379)` is a **false positive, not a genuine contract
violation**. Its precondition is `_list_contains(xs, elem) == true`, and `[]` contains nothing.
The controller saw this in 10 of 20 runs. Randomness currently hides the false failure about half
the time.

### Coupling and mandatory sequencing

**#547's requires filter MUST land before, or atomically with, #535's deterministic default.**
Pinning the seed first can freeze an out-of-contract sample into a permanent deterministic failure
for any meaningful precondition. The planner must schedule contract-domain filtering as M1 and may
only enable the deterministic default in M2 after M1's positive and negative controls pass. A
single PR is acceptable only if commits/tests preserve that ordering and the merge contains both.

### Blast radius and current CI exposure

`rg -l 'requires \{' examples | wc -l` finds 27 example files. This is exposure, not proof that all
27 currently false-fail. CI does **not** execute `ailang test` for examples: `make verify-examples`
uses `scripts/verify_examples.go`, whose three executable forms at lines 108-115 all pass the `run`
subcommand. Neither defect is currently red in that gate, so pinning the seed is low CI risk. New
targeted `go test`/CLI integration coverage is required; the design does not claim existing CI
coverage that is absent.

## Goals

**Primary goal**: make `ailang test` verdicts trustworthy: deterministic by default, replayable
when randomized, and evaluated only over inputs inside the function's declared domain.

1. Default seeds, generated cases, normalized JSON, counterexamples, and exit status reproduce.
2. An `ensures` counterexample must satisfy every `requires` clause on its function.
3. Generator inability to reach enough in-contract inputs is `skip`/unverified, never pass or fail.
4. JSON reports effective seeds and exact accepted/discarded counts.
5. Functions without `requires` retain the current 100-case execution behavior.

## Non-Goals

- Better generators or generator annotations (Lane B1).
- Changing standalone `runRequiresProperty` semantics beyond shared helpers/metrics needed here.
- Proving contracts for all possible in-domain inputs; this remains bounded randomized testing.
- Stable random streams across future Go/toolchain or generator-algorithm changes.
- Parser, typechecker, elaborator, evaluator, VM, effect, or code-generation changes.
- Treating `make verify-examples` as contract-test coverage; it runs examples, not properties.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Filter `ensures` candidates through all comma-separated lowered `requires` predicates before calling the function | Defines sound contract domain; repeated blocks are parser-invalid (`PAR_DUPLICATE_REQUIRES`) | human/design | design | medium |
| Require 100 accepted cases, with at most 1,000 generated attempts | Prevents both vacuous pass and unbounded filtering | human/design | design | medium |
| Exhausting 1,000 attempts before 100 acceptances is `skip`/unverified with `out_of_contract` | A sparse/unreachable domain is neither proved nor violated | human/design | design | medium |
| JSON reports integer `generated_inputs` and `discarded_inputs` | Exact rate without float/rounding ambiguity | human/design | design | low |
| One reported master seed feeds one versioned derivation for every property in all three modes | Preserves independent property streams while making a whole multi-property invocation replayable | reviewer option 2/human | design | medium |
| `--seed N` supplies the master seed; `--random-seed` reads one invocation-level master from `crypto/rand` | Collapses three seeding paths into one and makes nondeterminism explicit and replayable | reviewer option 2/human | design | medium |
| Add explicit `TestConfig.WorkspaceRoot` and thread it from the CLI's initial working directory | The field does not exist at HEAD; module-less identity cannot assume it | reviewer/human | design | medium |

### Design Freeze

- [x] #547 lands before or atomically with #535.
- [x] Success requires 100 in-contract executions; 1,000 total generation attempts is the cap.
- [x] Cap exhaustion is `StatusSkip`, `SkipKindOutOfContract`, not success or violation.
- [x] Discards are reported as exact integer counts, not a rounded JSON rate.
- [x] No-`requires` functions bypass predicate lookup/evaluation.
- [x] All modes use the same master-to-property derivation; randomized replay uses one master seed.
- [x] Derivation changes are reproducibility-breaking and require a new output version tag.
- [x] The testing configuration API explicitly carries `WorkspaceRoot`; it is not an existing facility.

## Solution Design

### M1: sound `ensures` domain filtering (#547)

After `ExtractFunctionBinding` has populated lowered Core metadata, call a new Runner accessor
`findAllLoweredContractPredicates(propCase, core.RequiresKind) []core.CoreExpr`. The accessor calls
`r.executor.LastDeclMeta(propCase.FunctionCtx)`, then filters `meta.Contracts` for
`c.Kind == core.RequiresKind`, appending every matching `c.Expr` in list order. This enumerates the
function-associated, comma-separated requires conditions in source order. M1 then conjoins them by
short-circuit evaluation: every returned predicate must be true.

This accessor is intentionally distinct from `findLoweredContractPredicate`: the existing function
returns **one** predicate, the n-th same-kind entry corresponding to the current property case. It
does not expose all predicates through `ExtractFunctionBinding`. `ExtractFunctionBinding` caches the
function-keyed `Core.Meta` map; the new accessor enumerates the selected function's cached
`DeclMeta.Contracts`. No parser, elaborator, lowering, or Core representation change is required.

Repeated `requires` blocks are outside M1's scope because they are impossible by construction: the
parser rejects a second block with `PAR_DUPLICATE_REQUIRES` and instructs authors to combine
conditions with commas. Therefore M1 handles no requires block, or one requires block containing
one or more comma-separated conditions; it does not add repeated-block handling.

For each generated argument tuple in `runEnsuresProperty`:

1. Increment `generated_inputs`.
2. If the collected requires slice is empty, take the existing fast path directly to ensures.
3. Otherwise evaluate each lowered predicate with the existing
   `EvaluateRequiresHarnessFromCore`/`BuildRequiresPropertyHarnessFromCore` machinery and the same
   generated values. A non-bool or evaluation error remains a real `fail` diagnostic.
4. If any predicate is false, increment `discarded_inputs` and generate another tuple. Do not call
   the function and do not evaluate ensures.
5. If all are true, call the function/evaluate ensures, increment `TestsRun`, and fail only if this
   in-contract postcondition is false.

`TestsRun` continues to mean executed in-contract cases. Pass only after 100 accepted cases.
Generation stops after 1,000 total attempts. If fewer than 100 survived, report:

```text
status: skip
skip_kind: out_of_contract
error: unverified: requires filter accepted A of G generated inputs; need 100 (D discarded)
```

This threshold permits up to a 90% discard rate while bounding work at 10x the current case budget.
It labels generator/domain mismatch honestly. Even 99 successful accepted cases plus 901 discards
is **not** a pass, because that would reintroduce a vacuous-success boundary.

For functions with no requires clause, the empty-slice branch performs no extra harness evaluation:
exactly 100 values are generated and checked, `generated_inputs=100`, `discarded_inputs=0`, and
existing verdict behavior is unchanged apart from additive metadata and deterministic samples.

### Discard reporting

Every property JSON object adds:

- `"generated_inputs"`: integer tuples generated;
- `"discarded_inputs"`: integer tuples rejected by `requires`.

The discard rate is therefore reported losslessly as `discarded_inputs / generated_inputs`; no
floating `discard_rate` field is added. Human output prints `accepted A, discarded D, generated G`
for filtered properties and for cap-exhaustion skips. This is additive to existing `tests_run`,
`skip_kind`, `error`, and counterexample fields.

### M2: deterministic and replayable seed policy (#535)

This design takes the reviewer's **second option**: retain independent property streams, but derive
every stream from one reported invocation master seed using one versioned algorithm. This preserves
the existing Decision A goal of property-identity diversity and, unlike independent entropy reads,
lets one `--seed <master>` command replay an entire multi-property invocation.

| CLI | `seed_mode` | Invocation master seed | Effective property seed |
|---|---|---|---|
| no seed flag | `derived` | signed `int64(0)` | `derive_v1(0, propertyIdentity)` |
| `--seed N` | `master` | exact signed `int64` N, including 0 | `derive_v1(N, propertyIdentity)` |
| `--random-seed` | `master` | one signed `int64` read from `crypto/rand` before any property runs | `derive_v1(master, propertyIdentity)` |

Supplying both flags is usage error exit 2. A randomized invocation performs exactly one entropy
read; entropy acquisition failure is a loud CLI failure before testing begins, with no clock,
constant, or per-property fallback. Remove the `seed == 0` wall-clock branch. `--seed N` and
`--random-seed` then enter the identical derivation path; property traversal order never affects a
stream.

`propertyIdentity` is the framed pair `(moduleIdentity, propertyName)`. `derive_v1` is SHA-256 of:

```text
"ailang-property-seed-v1\x00" + decimalMasterSeed + "\x00" + moduleIdentity + "\x00" + propertyName
```

Interpret bytes 0:8 little-endian as signed `int64`. `moduleIdentity` is the declared module path.
For module-less input, use the cleaned, slash-normalized path **relative to `TestConfig.WorkspaceRoot`**
(`filepath.Rel(config.WorkspaceRoot, inputPath)`, then `filepath.Clean` and `filepath.ToSlash`). The
field does not exist at HEAD and is added by this design; seed derivation must not reconstruct it
from the absolute input path.

Add `internal/testing/config.go` declaring `TestConfig` with `WorkspaceRoot string`, `SeedMode`, and
`MasterSeed int64`. The CLI captures `os.Getwd()` once, before path walking or package dispatch,
fails loudly if it cannot obtain the initial directory, and passes that value through
`runTestsV2`/`runPackageTests` → `runTestFile` → a configuration-taking `RunTestsFromFile` →
`NewRunner`. The Runner retains the config and passes its master and workspace root to the single
seed-derivation helper. Keep the existing public convenience entry point only as a compatibility
wrapper with an explicit documented configuration construction; CLI code must use the configured
entry point and may not silently infer a workspace root inside `internal/testing`.

JSON adds top-level string `seed_mode`, top-level decimal-string `seed` containing the **master**,
top-level string `seed_derivation` equal to `ailang-property-seed-v1`, and a per-property decimal-
string `seed` containing the derived stream seed; failures add `replay`. Human output reports mode,
master seed, effective property seed, derivation version, and the copy/paste-safe replay command
`ailang test --seed <master> <path>`.

`seed_mode` deliberately describes the effective execution path, not how the master was acquired:
both `--random-seed` and its `--seed N` replay report `master`. No random-provenance field is emitted,
because it would make otherwise identical replay JSON differ. Consequently, after durations are
removed, randomized and explicit replay reports are byte-identical.

“Versioned” is a compatibility contract: changing framing, hash, byte order, identity, or any other
master-to-property derivation detail is a breaking change to reproducibility. Such a change must
introduce a new version tag in `seed_derivation` rather than silently changing v1. Durations remain
variable and are removed from normalized comparisons.

### Why filtering precedes deterministic sampling

Filtering changes which generated attempts count, but it does not consume a second RNG: requires
and ensures use the same tuple before the next generation. Thus a numeric seed replays the same
attempt sequence and discard decisions. Landing M1 first establishes a sound verdict function;
M2 then makes that function stable. Reversing them deliberately stabilizes known wrong verdicts.

## Conflict Surface

### Touched packages and behavior

- `internal/testing/runner.go`: ensures loop, lowered-contract lookup, three RNG call sites.
- `internal/testing/config.go` (new): explicit `TestConfig` carrying workspace root, seed mode, and
  invocation master seed through the testing API.
- `internal/testing/harness.go` only if a small shared multi-requires evaluator is needed; existing
  requires harness machinery should be reused.
- `internal/testing/result.go` and reporters: accepted/discarded and seed metadata.
- `cmd/ailang/main.go`, `cmd/ailang/test.go`: seed flags; one initial `os.Getwd`; configured
  propagation through ordinary and package paths.

| Adjacent surface | Preserve/clarify | Required regression |
|---|---|---|
| No-requires ensures | 100 direct checks, no filtering overhead | existing correct and buggy ensures tests |
| Comma-separated requires | candidate accepted only when all conditions hold in source order | new enumeration/conjunction test |
| Repeated requires blocks | parser continues rejecting with `PAR_DUPLICATE_REQUIRES`; no M1 handling | parser diagnostic regression |
| Requires evaluation error/non-bool | fail loudly, never discard | harness error test |
| Sparse/unreachable requires | bounded skip/unverified, never pass/fail | cap-exhaustion test |
| Genuine in-domain violation | remains fail with satisfying counterexample | required negative control below |
| Standalone requires property | current out-of-contract skip behavior retained | existing requires tests |
| JSON/human reporters | parseability and prior details retained | reporter + CLI tests |
| Package discovery | only existing package test files | package integration test |
| Testing API/config | current API has no workspace-root field; add it and thread the same config through direct, ordinary, and package entry points | configuration propagation unit and CLI integration tests |
| Multi-property streams | independent derived streams remain order-independent, but share one replayable master | random-to-explicit normalized JSON/exit comparison |

No syntax, AST, parser, type inference, elaboration, Core lowering, evaluator, VM, effects, or codegen
change is designed. The measured comma-separated fixture and code read establish that contracts are
already retained as function-associated, kind-tagged, source-ordered `DeclMeta.Contracts`; the
requires harness already exists. A diff in compiler pipeline packages is scope expansion and
requires review.

### Intentional incompatibilities

- Out-of-contract ensures failures become discarded attempts; affected contracts may pass or skip.
- Default samples change once, then stabilize.
- `GenConfig.Seed == 0` becomes exact zero rather than wall-clock entropy.
- JSON/human output gains additive metadata; strict unknown-field consumers must update.

## Implementation Plan

### M1 — Contract soundness (~0.5–0.75 day; sequencing gate)

- Add `findAllLoweredContractPredicates`, enumerate same-function `RequiresKind` entries, and filter
  candidates by their conjunction.
- Add 100-accepted/1,000-generated accounting and unverified skip.
- Add no-requires, comma-separated-requires, sparse-domain, precond2, and genuine-violation tests.
- **Gate:** AC1–AC3 must pass before enabling deterministic default behavior.

### M2 — Seed policy and reporting (~0.5 day)

- Add `TestConfig` in `internal/testing/config.go`; capture the initial CLI working directory and
  thread the config through ordinary/package/file/Runner entry points.
- Implement one v1 master-to-property derivation for derived/explicit/random modes, perform one
  `crypto/rand` master read for random mode, and remove the time sentinel from all three paths.
- Add master seed, derivation version, per-property seed, discard JSON, and human replay reporting.

### M3 — Integration and repository gates (~0.25–0.5 day)

- Add normalized determinism, exact replay, random replay, and CLI conflict tests.
- Run focused suites, lint, and honestly record that verify-examples does not execute `ailang test`.

### Files to Modify

| File | Change | Estimate |
|---|---|---:|
| `internal/testing/config.go` (new) | `TestConfig`, validation/default construction, seed mode/master/workspace root | +55 |
| `internal/testing/runner.go` | requires filtering, bounded attempts, configured v1 derivation at three paths | +155/-25 |
| `internal/testing/harness.go` | optional shared multi-requires helper | +0–30 |
| `internal/testing/generator.go` | remove zero/time sentinel docs/behavior | +2/-4 |
| `internal/testing/result.go` | discard, master/derived seed, mode, and derivation-version metadata | +25 |
| `internal/testing/reporter.go` | human/JSON metrics, master/derived seed, version, replay | +65 |
| `internal/testing/*_test.go` | soundness, threshold, config propagation, derivation, entropy failure, reporter regressions | +270 |
| `cmd/ailang/main.go`, `cmd/ailang/test.go` | flags, initial-working-directory capture, validation, config threading in both modes | +85 |
| `cmd/ailang/test_json_output_test.go` | multi-property end-to-end schema/replay/verdict controls | +175 |

Expected implementation: roughly 650–830 net test-heavy LOC over 1.5–2 days. The increase covers
the newly measured configuration-API gap, explicit workspace-root propagation, master-seed
derivation/versioning, entropy-failure coverage, and true multi-property replay. No compiler-
pipeline work is required.

## Acceptance Criteria

Commands run from repository root after `make build`. Each fixture shown is part of the criterion
and should be checked into the test fixture directory selected by the implementer.

### AC1 — excluded inputs no longer fail (`precond2.ail`)

```bash
./bin/ailang check testdata/precond2.ail
./bin/ailang test --seed 42 --format json --no-color testdata/precond2.ail > /tmp/precond2.json
jq -e '.failed_tests == 0 and ([.properties[] | select(.name | contains("big")) | select(.status == "pass" and .tests_run == 100 and .discarded_inputs > 0)] | length > 0)' /tmp/precond2.json
```

Expected: all commands exit 0; the contract goes from false fail to pass because 100 in-contract
inputs are checked. The standalone requires property may remain an `out_of_contract` skip.

### AC2 — genuine in-domain violation still fails (negative control)

Fixture:

```ailang
module precond_negative
export func broken(x: int) -> bool ! {}
requires { x > 100 }
ensures { result == true } {
  false
}
export func main() -> int ! {} { if broken(500) then 1 else 0 }
```

```bash
./bin/ailang check testdata/precond_negative.ail
set +e
./bin/ailang test --seed 42 --format json --no-color testdata/precond_negative.ail > /tmp/precond-negative.json
rc=$?
set -e
test "$rc" -eq 1
jq -e '.properties[] | select(.name | contains("broken")) | select(.status == "fail" and (.error | contains("ensures violated")) and (.error | test("x=[1-9][0-9]{2,}")))' /tmp/precond-negative.json
```

Expected: every assertion exits 0. The test invocation itself exits 1 and reports an `x > 100`
counterexample, proving filtering did not create a vacuous pass.

### AC3 — unreachable domain is unverified, never passed or violated

Fixture (the ordinary passing test keeps the suite exit at 0 while the property verdict is
inspected independently):

```ailang
module requires_unreachable
export func impossible(x: int) -> bool ! {}
requires { x > 1000 && x < -1000 }
ensures { result == true } {
  false
}
test "suite sentinel passes" { assert(true) }
export func main() -> int ! {} { 0 }
```

```bash
./bin/ailang check testdata/requires_unreachable.ail
./bin/ailang test --seed 42 --format json --no-color testdata/requires_unreachable.ail > /tmp/unreachable.json
jq -e '.properties[] | select(.name | contains("ensures")) | select(.status == "skip" and .skip_kind == "out_of_contract" and .tests_run == 0 and .generated_inputs == 1000 and .discarded_inputs == 1000 and (.error | startswith("unverified:")))' /tmp/unreachable.json
```

Expected: both commands exit 0 (the fixture also contains one ordinary passing test so suite-level
all-skipped policy does not obscure the property verdict); the ensures property is explicitly skip/unverified.

### AC4 — functions without requires are unaffected

```bash
go test ./internal/testing -run 'TestRunEnsuresProperty_(CorrectImplPasses|ViolationReportsCounterexample)' -count=1
```

Expected: exit 0; pass/fail semantics remain, and added assertions require
`generated_inputs=100`, `discarded_inputs=0`.

### AC5 — default and exact-seed results reproduce

```bash
tmp=$(mktemp -d)
for run in a b; do
  set +e
  ./bin/ailang test --format json --no-color examples/runnable/contracts/list_recursive_verify.ail >"$tmp/$run.json" 2>"$tmp/$run.err"
  printf '%s\n' "$?" >"$tmp/$run.rc"
  set -e
  jq 'del(.total_duration) | .tests |= map(del(.duration)) | .properties |= map(del(.duration))' "$tmp/$run.json" >"$tmp/$run.norm"
done
cmp "$tmp/a.rc" "$tmp/b.rc"
cmp "$tmp/a.norm" "$tmp/b.norm"
jq -e '.seed_mode == "derived" and .seed == "0" and .seed_derivation == "ailang-property-seed-v1" and ([.properties[] | select(.seed != null)] | length > 0)' "$tmp/a.json"
```

Expected: all assertions exit 0. `containsImpliesNonEmpty` must not fail on `xs=[]`; its final
stable verdict may be pass or honest unverified skip depending on the frozen stream's hit rate.

### AC6 — a multi-property random invocation replays exactly and flags validate

```bash
tmp=$(mktemp -d)
cat >"$tmp/multi.ail" <<'EOF'
module random_replay_multi

export func ok(x: int) -> bool ! {}
ensures { result == true } { true }

export func broken(x: int) -> bool ! {}
ensures { result == true } { false }
EOF
./bin/ailang check "$tmp/multi.ail"
./bin/ailang test --help | grep -F -- '--seed N'
./bin/ailang test --help | grep -F -- '--random-seed'
set +e
./bin/ailang test --seed 1 --random-seed "$tmp/multi.ail" >"$tmp/conflict.out" 2>"$tmp/conflict.err"
rc=$?
set -e
test "$rc" -eq 2
grep -F 'cannot be used together' "$tmp/conflict.err"

set +e
./bin/ailang test --random-seed --format json --no-color "$tmp/multi.ail" >"$tmp/random.json" 2>"$tmp/random.err"
random_rc=$?
set -e
seed=$(jq -er '.seed | select(test("^-?[0-9]+$"))' "$tmp/random.json")
test "$(jq '[.properties[] | select(.seed != null)] | length' "$tmp/random.json")" -ge 2

set +e
./bin/ailang test --seed "$seed" --format json --no-color "$tmp/multi.ail" >"$tmp/replay.json" 2>"$tmp/replay.err"
replay_rc=$?
set -e

for run in random replay; do
  jq 'del(.total_duration) | .tests |= map(del(.duration)) | .properties |= map(del(.duration))' \
    "$tmp/$run.json" >"$tmp/$run.norm"
done
test "$random_rc" -eq "$replay_rc"
cmp "$tmp/random.norm" "$tmp/replay.norm"
jq -e --arg s "$seed" '.seed_mode == "master" and .seed == $s and .seed_derivation == "ailang-property-seed-v1"' "$tmp/replay.json"
```

Expected: every assertion exits 0. The fixture has two properties and produces a failing suite in
both runs; mutual exclusion is usage exit 2. The top-level random master is accepted verbatim by
`--seed`, and after removing only durations, the complete JSON (generated cases as reflected in
verdicts/counterexamples/metrics/replay text) and exit status are byte-for-byte identical.

### AC7 — focused and repository gates

```bash
go test ./internal/testing/... -count=1
go test ./cmd/ailang/... -count=1
make lint
make verify-examples
```

Expected: all exit 0. `make verify-examples` is only a run-mode regression gate, not evidence that
contract properties executed. A sandbox denial such as `bind: operation not permitted` is
**UNINFORMATIVE UNDER SANDBOX**, never pass or product failure.

### AC8 — requires representation, association, ordering, and accessor coverage

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
./bin/ailang test --seed 42 --format json --no-color "$tmp/ordered.ail" >"$tmp/ordered.json"
rc=$?
set -e
test "$rc" -eq 0
jq -e '[.properties[].name] == [
  "g_property_1", "g_property_2", "g_property_3", "h_property_1"
] and (.properties[0].location | endswith(":5:12"))
  and (.properties[1].location | endswith(":5:19"))
  and (.properties[2].location | endswith(":6:11"))
  and (.properties[3].location | endswith(":12:11"))
  and (.properties[3].status == "pass")' "$tmp/ordered.json"
go test ./internal/testing -run 'TestFindAllLoweredContractPredicates_(OrderAndAssociation|ZeroRequires)' -count=1
```

Expected: every assertion exits 0. The parser proves repeated blocks are outside M1; the JSON order
and locations prove comma-separated conditions remain separate and source ordered, `g`/`h` names
prove function association, and `h` proves zero-requires isolation. The focused unit tests directly
exercise the new all-of-kind accessor (including an empty result), rather than the existing
single-predicate lookup.

### AC9 — derived seed is independent of absolute checkout path

```bash
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
jq -r '.properties[] | select(.name == "stable_property_1") | .seed' "$tmp/a.json" >"$tmp/a.seed"
jq -r '.properties[] | select(.name == "stable_property_1") | .seed' "$tmp/b.json" >"$tmp/b.seed"
test -s "$tmp/a.seed"
cmp "$tmp/a.seed" "$tmp/b.seed"
```

Expected: all commands exit 0. The same module-less file at the same workspace-relative path but
under two different absolute roots has the identical effective property seed, proving the absolute
checkout directory is absent from the hash identity.

## Testing Strategy

Unit tests cover all/any requires selection, same-tuple filtering, 100/1,000 boundary values,
evaluation errors, no-requires fast path, and the genuine violation. Seed tests cover exact framing,
zero/negative master seeds, all three RNG paths, injected entropy-reader failure, and absence of time
fallback. Configuration tests prove `WorkspaceRoot` reaches module-less identity derivation in direct,
ordinary, and package modes. Reporter tests pin integer discard fields, decimal-string seed fields,
and derivation version. CLI tests cover ordinary/package modes, normalized determinism, multi-property
replay, mutual exclusion, and exit status.

Mutation checks must fail if: any requires clause is ignored; discarded tuples call the function;
99 accepted cases pass; the attempt cap is removed; no-requires execution filters; a genuine
in-domain counterexample is discarded; a call site bypasses seed policy; or seed zero invokes time.

## Rollback

The two fixes form one trust boundary. Roll back atomically across:

1. requires collection/filtering, accepted/discard accounting, and unverified skip behavior;
2. seed policy/derivation at all three property paths and CLI propagation;
3. `TestConfig`, CLI initial-working-directory capture, and configured testing entry points;
4. result/reporter schema and replay output;
5. all associated fixtures/tests/help.

A partial rollback is unsafe. Keeping deterministic seeds while removing filtering re-freezes #547
false positives; keeping random mode without reporting makes failures unreplayable; reverting only
one RNG path restores hidden nondeterminism; removing `WorkspaceRoot` while retaining module-less
derivation makes checkout paths ambient again. Operationally, `--random-seed` permits fresh explicit
sampling, but it is not a rollback of sound filtering. A full rollback restores both #535 and #547
and must be temporary.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Sparse valid domain cannot produce 100 cases | Medium | bounded skip/unverified with exact counts; improve generators in Lane B1 |
| Filter accidentally hides real failures | High | required satisfying-input negative control and same-tuple assertions |
| Comma-separated requires are only partially applied | High | enumerate all same-kind metadata entries; source-order/conjunction regression |
| Seed pins a false positive | High | mandatory M1-before-M2 sequencing gate |
| New loop costs 10x on sparse domains | Medium | hard 1,000-attempt cap and visible counts |
| JSON seed precision loss | High | decimal strings for signed 64-bit seeds |
| Derivation changes silently break old replay commands | High | required `seed_derivation` version tag; v1 semantics are frozen |
| Workspace root is lost on one CLI/API path | High | explicit `TestConfig` propagation tests for direct, ordinary, and package modes |
| Entropy acquisition fails | Medium | fail before tests; injected-reader negative test; no fallback |
| CI confidence overstated | Medium | targeted tests; explicitly document verify-examples as run-only |

## Deferred Decisions

- Internal helper names and test fixture directory — implementer may choose.
- Whether conjunction evaluation beyond the named Runner accessor is a Runner helper or harness
  helper — implementer may choose; it must preserve source order and short-circuit false.
- Human formatting punctuation — implementer may choose; counts and replay data are mandatory.

## Axiom Compliance

**Canonical reference**: [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---:|---|
| A1 Determinism | +1 | default cases/verdicts are stable across runs and checkout locations |
| A2 Replayability | +1 | effective seed and exact filtering counts are reported |
| A3 Effect Legibility | +1 | entropy is explicit at CLI boundary |
| A4 Explicit Authority | +1 | default removes ambient clock/entropy access |
| A5 Bounded Verification | +1 | 100 accepted cases and 1,000 attempts are explicit bounds |
| A6 Safe Concurrency | 0 | no concurrency change |
| A7 Machines First | +1 | sound exit status and structured JSON serve agents/CI |
| A8 Minimal Syntax | 0 | no language syntax change |
| A9 Cost Visibility | +1 | generated/discarded counts expose filtering cost |
| A10 Composability | +1 | requires and ensures compose as one contract domain |
| A11 Structured Failure | +1 | violation, unverified, and replay states are distinguishable |
| A12 System Boundary | +1 | randomness confined to explicit opt-in |

**Net score: +10** — proceed. No hard violation of A1/A3/A4/A7.

## Verification Log

All new checks ran 2026-07-31 in this worktree at HEAD `386cf6d15` unless attributed to the
controller's first-party reproduction.

| ID | Claim | Method | Result |
|---|---|---|---|
| V1 | Wall-clock root cause and three RNG paths | read `generator.go` and `runner.go`; `rg newRNG` | confirmed: forall/ensures/requires share zero sentinel |
| V2 | Ensures never evaluates requires | read `runEnsuresProperty` loop | confirmed: only ensures harness is evaluated |
| V3 | Requires false is explicitly non-bug skip | read `runRequiresProperty` comments/lines 540-543 | confirmed |
| V4 | Repeated requires blocks are impossible by construction | fixture `v13.ail`; `./bin/ailang check v13.ail` | exit 1: `PAR_DUPLICATE_REQUIRES` at the second block; diagnostic says only one block and to combine conditions with commas |
| V5 | `precond2` is valid and false-fails | `ailang prompt`; `ailang check`; six local JSON runs | check passed; six of six failed only on excluded values (independent confirmation of controller values) |
| V6 | Genuine negative control is valid/currently detectable | `ailang check`; six local JSON runs | check passed; six of six failed; satisfying counterexamples observed in several runs |
| V7 | Real list failure classification | read example precondition/body and controller's 20-run data | **refuted prior draft**: empty-list failure is false positive, not genuine violation |
| V8 | Example blast radius | `rg -l 'requires \{' examples | wc -l` | 27 files |
| V9 | CI executes examples with `run`, not `test` | read `scripts/verify_examples.go:108-115` | confirmed; all three forms pass `run` |
| V10 | Existing result model supports honest skip taxonomy | read `result.go` and vacuous-success tests | confirmed `out_of_contract`; all-vacuous success guard exists |
| V11 | Comma-separated conditions remain separate, source ordered, and function associated; zero-requires function is isolated | fixture `v13b.ail`; `./bin/ailang check v13b.ail`; `./bin/ailang test --format json v13b.ail`; `jq -r '.properties[] \| [.name,.status,.location] \| @tsv' /tmp/v13b-test.json` | check exit 0; test exit 1; locally observed `g_property_1 skip v13b.ail:4:12`, `g_property_2 skip v13b.ail:4:19`, `g_property_3 fail v13b.ail:5:11`, `h_property_1 pass v13b.ail:8:11`. Columns 12 then 19 confirm source order; prefixes confirm association; `h` has exactly its one passing ensures. Controller's line-padded equivalent observed `5:12`, `5:19`, `6:11`, `12:11` |
| V12 | Unread session messages | `ailang messages list --unread` | none |
| V13 | Exact metadata lookup and accessor boundary | `sed -n '35,55p' internal/testing/executor.go`; `sed -n '556,600p' internal/testing/runner.go`; `sed -n '268,300p' internal/elaborate/file.go` | observed `LastDeclMeta` reads the cached `lastCoreMeta[funcName]`; `findLoweredContractPredicate` filters same-kind entries and returns the indexed one; elaboration assigns `Contracts: contracts` produced from `astFunc.Properties`. M1 therefore adds an all-of-kind accessor in testing; no compiler-pipeline representation change is designed |
| V14 | Existing requires harness evaluates a tuple without calling the function | `rg -n 'EvaluateRequiresHarnessFromCore\|BuildRequiresPropertyHarnessFromCore' internal/testing/{executor,harness}.go`; read both bodies | confirmed: parameter values are bound directly and the lowered predicate evaluates to a bool; function binding/call is absent |
| V15 | `WorkspaceRoot` is **absent** from the current testing configuration and CLI path; the grep can see a known declaration | `grep -rnE 'workspaceRoot\|WorkspaceRoot\|type GenConfig\|type TestConfig\|type Config' internal/testing/*.go cmd/ailang/test.go` | sole output: `internal/testing/generator.go:17:type GenConfig struct {`; therefore zero workspace-root matches, while the same command's known-positive `GenConfig` match proves the instrument works |
| V16 | There is no current `internal/testing/config.go` | `ls internal/testing/*.go; ls internal/testing/*.go \| wc -l` | output listed 30 Go files from `callgraph.go` through `shrink_test.go`, then `30`; `config.go` was absent |
| V17 | The current CLI-to-Runner path has no configuration argument | `rg -n 'case "test"\|runTestsV2\(|runPackageTests\(|runTestFile\(|RunTestsFromFile\(|NewRunner\(' cmd/ailang/{main,test}.go internal/testing/runner.go` | output: `main.go:111 case "test"`; `main.go:133 runPackageTests(...)`; `main.go:135 runTestsV2(...)`; `test.go:17 runTestsV2(...)`; `test.go:53 runTestFile(file)`; `test.go:93 runTestFile(filename)`; `test.go:119 RunTestsFromFile(filename, file)`; `test.go:136 runPackageTests(...)`; `test.go:188 runTestFile(file)`; `runner.go:21 NewRunner(modulePath)`; `runner.go:621 RunTestsFromFile(filePath, ast)`; `runner.go:627 NewRunner(filePath)` |
| V18 | Three property paths currently use the zero-sentinel RNG and `newRNG` reads wall clock | `rg -n 'DefaultConfig\(\)\|newRNG\(|time.Now\(\)\.UnixNano' internal/testing/{generator,runner}.go` | output includes `runner.go:260-261`, `385-386`, and `504-505` (`DefaultConfig`; `newRNG(config.Seed)`), plus `runner.go:785 newRNG` and `runner.go:787 seed = time.Now().UnixNano()` |
| V19 | Ensures and requires currently use distinct harness paths and requires-false emits the skip text | `rg -n 'func \(r \*Runner\) runEnsuresProperty\|EvaluateEnsuresHarnessFromCore\|func \(r \*Runner\) runRequiresProperty\|requires not satisfied\|EvaluateRequiresHarnessFromCore' internal/testing/runner.go` | output: ensures at lines `322`/`400`; requires at `447`/`519`; requires-false text at `543` |
| V20 | Current metadata and existing single-predicate accessor are in the claimed files | `rg -n 'func \(e \*Executor\) LastDeclMeta\|func \(r \*Runner\) findLoweredContractPredicate\|Contracts:\|astFunc.Properties' internal/testing/{executor,runner}.go internal/elaborate/file.go` | output: `executor.go:43 LastDeclMeta`; `runner.go:566 findLoweredContractPredicate`; `file.go:277 elaborateContracts(astFunc.Properties)`; `file.go:285 Contracts: contracts` |
| V21 | HEAD used for the re-measurement | `git rev-parse HEAD` | `386cf6d1584912803d2c59c03838b2a04e5a33af` |

Two earlier document premises are explicitly refuted: (1) `workspaceRoot` was **not** already
defined or propagated—the reviewer/controller measurement is correct, so this revision adds the
configuration API work; and (2) the earlier classification of `list_recursive_verify.ail` was a
false positive rather than a genuine violation. No reviewer objection was overridden. Exact future
stable verdict/hit rate for that list property cannot be known until both fixes implement the
specified stream and filter.

## Related Documents

- [M-PROPERTY-GENERATOR-COVERAGE](m-property-generator-coverage.md) — Lane A enabled list generators;
  Lane B1 depends on trustworthy verdicts.
- [Lane A sprint plan](m-property-generator-coverage-lane-a-sprint-plan.md) — earlier control data.
- [M-DX26 property-test empty-program work](../../implemented/v0_21_0/m-dx26-property-test-empty-program.md)
  — introduced the contract harness paths reused here.
- GitHub issues #535 and #547.

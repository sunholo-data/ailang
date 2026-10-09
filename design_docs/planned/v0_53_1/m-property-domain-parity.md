# Property Domain Parity: requires-filtered forall properties, named failing inputs, attested-run agreement

**Status**: Planned
**Target**: v0.53.1
**Priority**: P1
**Estimated**: 4 days
**Dependencies**: [m-package-test-discovery](../m-package-test-discovery.md) (planned, unlanded since v0.38.x — must land in the same release for Leg C; see Dependencies section)
**Source**: User report at AILANG v0.52.0 against `sunholo-data/ailang-packages` main (599a77d): `ailang test .` reports 3 property failures in `packages/celestial` (`starIlluminanceAt_property_2`, `discIlluminance_property_2`, `ringUnlitIF_property_2`) and 7 in `packages/relativity` (`expm1_property_1`, `sinhc_property_1`, `accelerate_property_2`, `doppler_property_2`, `dopplerApparent_property_2`, `cmbSeenTemperature_property_2`, `cmbSeenTemperatureApparent_property_2`); every per-file `ailang test <file>` run is green; `ailang pkg quality` attests 0 failed.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Discard decisions are pure evaluations of lowered requires predicates over seeded-RNG values; no wall clock, no per-run variance (same seed ⇒ same accepted/discarded stream) |
| A2: Replayability | +1 | Failing inputs are now carried in the machine-readable `failing_input` field next to the per-property seed, so a replay command plus the named input reproduces the failure exactly |
| A3: Effect Legibility | 0 | No effect changes; property harnesses remain evaluator-run, effect-free at the harness level |
| A4: Explicit Authority | 0 | No capability surface touched |
| A5: Bounded Verification | +1 | The discard loop inherits the ensures path's `requiredAccepted=100 / maxAttempts=1000` bound, bounding generation even when requires starve |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `failing_input` in `--json` is exactly what an agent needs to replay and fix; agreement between `pkg quality` and `ailang test .` removes a signal that contradicted itself by discovery mode |
| A8: Minimal Syntax | +1 | No new syntax; the fix reuses `requires` and existing harness plumbing (the `where`-guard alternative is rejected in Non-Goals) |
| A9: Cost Visibility | 0 | Discarded inputs are already counted and reported (`accepted N, discarded M, generated K`) |
| A10: Composability | +1 | Inline forall properties compose with the function's contract exactly as inline ensures already does — one domain notion, both harnesses |
| A11: Structured Failure | +1 | Every failing generated case now names its input as `name=value` pairs in a dedicated field instead of prose-only text |
| A12: System Boundary | 0 | No boundary crossings added |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

`ailang test .` and `ailang pkg quality` disagree about the same package in the same directory, and when the whole-directory run fails, the failure tells the author too little to act on. The report is the live instance; the mechanism was reproduced end-to-end on a synthetic package (see Verification Log V1–V6) and reduces to **three independent defects with one symptom**.

### Defect A — discovery split: quality never runs the failing properties

`ailang pkg quality` attests its tests section by shelling out to `ailang test --package --format json .` (`cmd/ailang/pkg_quality.go`, `runAttestedChecks`), and `--package` mode (`runPackageTests`, `cmd/ailang/test.go`) walks **only `*_test.ail` files**. Inline `properties [...]` blocks on functions live in source modules (e.g. `relativity.ail`), which package mode never executes. The failing `<func>_property_N` cases therefore never ran under quality — it attested "2 passed, 0 failed" over a suite that excludes them.

This is the unlanded [m-package-test-discovery](../m-package-test-discovery.md) gap (Target v0.38.x, still unlanded at v0.53.0 — `std/VERSION`), previously measured live as [pub012-inline-tests-false-absence](../ailang-core-triage/pub012-inline-tests-false-absence.md).

The report's "every per-file run is green" is consistent with — and forced by — this split plus seed determinism: property seeds derive from `DeriveSeedV1(master, moduleIdentity, propertyName)` where a declared module path is invocation-independent (`internal/testing/config.go`), so a per-file run of a file containing a failing property reproduces the failure byte-identically (verified, V4). The only per-file runs that can be green are runs of files without the failing properties — i.e. the `*_test.ail` files. The failures live in source modules.

### Defect B — domain asymmetry: the forall harness ignores the function's requires

Two harnesses execute contract-adjacent properties (`internal/testing/runner.go` routes by `ast.Property.Kind`):

- **ensures** (`runEnsuresProperty`, `internal/testing/contract_domain.go`): generates inputs, **discards those violating the function's `requires` clauses** (`allRequiresHold`, up to `maxAttempts=1000` for `requiredAccepted=100`), and only checks the postcondition on in-contract inputs.
- **forall** (`runProperty`, `internal/testing/runner.go`): generates inputs from bare type generators — `float` uniformly in `[-1000.0, 1000.0]`, `int` in `[-1000, 1000]` (`internal/testing/generator.go`, `DefaultConfig`; documented in `docs/docs/guides/testing.md` "Integer and Size Ranges") — with **no domain filter at all**, even when the property is an inline block on a function whose `requires` states the domain.

Functions in physics/astronomy packages are documented over narrow domains (non-negative radii, `|v| < c`, finite `z`), and the language's only machine-readable domain is `requires`. When the generator crosses the domain, the function returns `NaN`/`±Inf` (`sqrt` of a negative, `exp` overflow making `Inf − Inf`), and the property's comparison goes false. Reproduced (V3, V5):

- `expm1_property_1` failed on `x=866.7105463623886` (`exp` overflows, `expm1(x) − (exp(x) − 1.0)` is `Inf − Inf = NaN`);
- `ringWidth_property_1` failed on `r=-1.564882392817016e-148` (`sqrt` of a negative is `NaN`, `NaN >= 0.0` is false);
- the decisive fixture — one function carrying `requires { r >= 0.0 }`, `ensures { result >= 0.0 }`, and a forall property, all on `gwidth(r) = sqrt(r)`: the requires case **skips** ("requires not satisfied by random input … r=−781.67"), the ensures case **passes** (accepted 100, discarded 102), and the forall case **fails** on `r=-2.3580454226813388e-148` — the same domain restriction, three verdicts, two of them wrong for the author's intent.

The documented authoring convention today is to encode preconditions inside the predicate ("A precondition: there is no `where`/`==>`, so write it as an if", `docs/docs/guides/testing.md`). The packages' properties did not, and nothing in the toolchain reconciled the property with the function's declared domain.

### Defect C — diagnostics: a failing generated case does not reliably name its input

`PropertyResult.FailingInput` exists (`internal/testing/result.go:40`, "Minimal failing input (if failed)") and the reporter renders it — human "Failing input:" line and JSON `failing_input` key (`internal/testing/reporter.go`) — but **no code path ever assigns it** (grep over `internal/` and `cmd/` finds zero `FailingInput =` outside tests; V7). Meanwhile:

- forall failures bury the counterexample inside the `error` string as bare values (`property failed on input: [866.7105463623886]`) with no binder names — ambiguous for multi-binder properties;
- evaluation-error paths report only an index (`test 37: evaluation failed: …`, `runner.go` `runProperty`/`runRequiresProperty`) with **no input at all**;
- the ensures/requires paths format `name=value` into `error` (`formatEnsuresInputs`) but not into `failing_input`.

The reported expectation — "a failing generated case names its input" — is currently a property of some code paths and not others, and the dedicated field never fires.

**Impact:** package authors and agents get a quality attestation that says 0 failed while the whole-directory run exits 1 with the same binary in the same directory; when the run does fail, the report under-specifies the input. For a language whose primary author is an AI agent, both are trust-destroying: the attested block is what the registry banks, and `ailang test .` is what CI and agents run.

## Goals

**Primary Goal:** Inline forall properties treat the function's `requires` exactly as the ensures harness does, every failing generated case names its input in a machine-readable field, and `ailang pkg quality` and `ailang test .` agree by construction.

**Success Metrics:**
1. The `gwidth`-style fixture (requires + ensures + forall on a domain-restricted function) reports **skip / pass / pass** for the three cases, with the forall case showing `accepted 100, discarded N`.
2. A failing forall property reports `failing_input` (`x=866.71…`) in `--json` and a "Failing input: x=…" line in human output; evaluation-error failures report the generated values alongside the test index.
3. On a package whose only failing tests are inline properties in source modules: `ailang test .` exits 1 **and** `ailang pkg quality` raises the PUB012 gate (`N test(s) failed`, publisher mode) — both name the same properties.
4. Seed determinism unchanged: `TestSeedE2E_DefaultRunIsDeterministic` and `TestRunner_AllThreePropertyPathsUseDerivedSeed` stay green without modification to their seed assertions.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Forall requires-filter aligns binders to function params **by name**; filter applies only when every param has a same-named binder, else no filter | Defines when the new semantics silently weaken what the author wrote | human (ratify in design) | design | med |
| D2: Violating inputs are **discarded and counted** (ensures-path loop shape: 100 accepted / 1000 attempts), never a failure and never a first-violation stop | Chooses between three existing behaviors in the codebase today (ensures discard, requires first-violation skip, forall none) | human (ratify in design) | design | med |
| D3: `FailingInput` becomes the canonical machine-readable field; `error` keeps its sentence for humans | JSON schema consumed by `pkg quality` and agents | agent | design | low |
| D4: No new syntax (`where` guards rejected) | A parser/prompt change would ripple through teaching prompts, fmt, and every model trained on them | human | design | high (if reversed) |
| D5: Agreement rides m-package-test-discovery (union discovery), not a new quality-side runner | One discovery path, two conventions — the existing planned doc already rules the design | human (scheduling) | design | med |
| D6: Starved acceptance (fewer than 100 accepted in 1000 attempts) is a `SkipKindOutOfContract` skip, as in the ensures path | Exit-code semantics for packages whose domain is a tiny fraction of the generator range | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 ratified: name-alignment rule (and the documented fallback: no matching binder names ⇒ no filter, current behavior)
- [ ] D2 ratified: discard-and-count, not fail, not first-violation stop
- [ ] D5 ratified: m-package-test-discovery lands in the same release (Leg C is blocked without it)

## Solution Design

### Overview

Three legs, one release. **Leg A** makes the forall harness consult the function's `requires` (parity with ensures). **Leg B** populates `FailingInput` and names inputs on every failure path. **Leg C** makes `pkg quality`'s attested run execute the same suite as `ailang test .` by landing the already-designed union discovery. Legs A and B are confined to `internal/testing/`; Leg C is confined to the already-written m-package-test-discovery design plus `cmd/ailang/pkg_quality.go` counting.

### Architecture

**Leg A — forall/requires parity (`internal/testing/runner.go`, reuse from `contract_domain.go`)**

In `runProperty`, after generators are built and before the generation loop:

1. If `propCase.Function != nil` (inline property), call `r.executor.ExtractFunctionBinding(propCase.FunctionCtx, r.executor.sourceFile)` — as `runEnsuresProperty` does — so the elaborated, lowered contracts land in `LastDeclMeta`.
2. Collect lowered requires predicates with the existing `findAllLoweredContractPredicates(propCase, core.RequiresKind)`. If none, behavior is exactly today's.
3. **Alignment (D1):** build an `EnsuresParam` list iff every `propCase.Function.Params[i].Name` has a binder in `propCase.Property.Binders` with the same name (order-free). The property has already typechecked as a program, so binder values are compatible with the params they feed. If any param lacks a same-named binder, skip filtering entirely — the property is not a simple binding of the function's parameters and we must not guess a substitution.
4. **Loop shape (D2):** replace the fixed `numTests = 100` loop with the ensures-path shape: generate a case, splice binder values into the aligned params, evaluate `allRequiresHold`; on violation `DiscardedInputs++` and continue; otherwise evaluate the predicate as today. Stop at 100 accepted or 1000 generated. Starved ⇒ `StatusSkip` / `SkipKindOutOfContract` with the ensures path's message shape.
5. Shrinking still operates on the failing case's binder values — unchanged (`shrinkCounterexample`).

The per-property RNG stream changes for requires-bearing functions (discards consume draws) but remains a pure function of `DeriveSeedV1` — the derivation itself is frozen and untouched.

**Leg B — named failing inputs (`internal/testing/runner.go`, `contract_domain.go`)**

1. Add `formatBinderInputs(binders []*ast.Binder, values []eval.Value) string` next to `formatEnsuresInputs` — same `name=value, …` shape, binder names, `argN` fallback.
2. `runProperty`: on predicate-false, set `result.FailingInput = formatBinderInputs(...)` (keep the `error` sentence); on the evaluation-error and non-bool paths, include the values in the message — `test 37 (x=…, y=…): evaluation failed: …` — and set `FailingInput` too.
3. `runEnsuresProperty`: set `FailingInput` from the existing `formatEnsuresInputs` output (the `error` text keeps its current sentence, so `TestRunEnsuresProperty_ViolationReportsCounterexample`'s assertions stay true).
4. `runRequiresProperty`: set `FailingInput` on the out-of-contract skip (it already formats `name=value` into the message).
5. Reporter is unchanged — it already renders `FailingInput` in both formats; the field simply starts firing.

**Leg C — attested agreement (dependency: m-package-test-discovery)**

1. Land [m-package-test-discovery](../m-package-test-discovery.md) as designed: `runPackageTests` runs the union of `*_test.ail` and inline-bearing source modules in walk order, through the same per-file seam (`runTestFile` → `RunTestsFromFileWithConfig`), so seeds and names are mode-stable.
2. `runAttestedChecks` (`cmd/ailang/pkg_quality.go`): count `TestsSection.Files` as the union's file count (today it globs `*_test.ail` only), and drop the "inline test blocks are not yet discovered" note.
3. With the union, `pkg quality`'s attested run and `ailang test .` execute the same suite via the same seam — agreement holds by construction, not by reconciliation.

### Semantic Conflict Surface

What positions does this change extend? The execution semantics of **inline forall properties on functions that carry `requires` clauses**, and the **content of failure reports** for all property kinds.

What other constructs already live in those positions?

- **Top-level `property "name" { forall … }` blocks** — no `Function` context, so Leg A never applies; behavior and seeds unchanged.
- **Inline forall properties on uncontracted functions** — no lowered requires ⇒ no filter; identical case stream (same seed, same draws). This is the majority of existing properties.
- **Inline forall properties whose binder names do not match params** (`forall(a: float) => gwidth(a*2) >= 0.0`) — no filter per D1; unchanged.
- **requires/ensures property cases** — routed to their own harnesses before `runProperty`; unchanged except `FailingInput` population.
- **`tests [...]` inline tables** — fixed inputs, not generated; unchanged.

Which existing programs MUST still work post-change? (fixtures, all verified to exist)

- `internal/format/testdata/contracts_and_properties.ail` — requires + ensures + forall on one function; the forall (`forall(y: int) => f(y) >= 0` with `requires { x >= 0 }`) must go from fail-prone to discard-filtered, and binder `y` ≠ param `x` ⇒ **no filter, unchanged behavior** — this fixture is the D1 negative case.
- `examples/inline_tests_best_practices.ail` — inline tables and properties without contracts; must be byte-identical.
- `internal/testing/runner_ensures_test.go` fixtures — `TestRunEnsuresProperty_ViolationReportsCounterexample` (asserts `error` contains `ensures violated` and `x=`), `TestRunRequiresProperty_OutOfContractReportsSkip` (asserts skip + `x=`), `TestRunEnsuresProperty_CorrectImplPasses` (100 iterations) — all must stay green.
- `cmd/ailang/test_seed_e2e_test.go` — `TestSeedE2E_DefaultRunIsDeterministic`, `TestSeedE2E_RandomThenExplicitReplaysByteIdentical` — must stay green (no RNG-stream change for uncontracted properties).

What deliberately changes?

- Forall properties on requires-bearing functions with matching binder names: previously-failing out-of-domain inputs are discarded (or the property skips when starved). This is the point; it is a behavior change and gets a changelog entry.
- Golden outputs that pin a forall case stream for such properties (none known in-tree; the risk is external goldens).

### Implementation Plan

**Phase 1: Forall requires parity** (~1.5 days)
- [ ] `runProperty`: alignment check, `ExtractFunctionBinding`, requires collection, discard loop, starved skip
- [ ] Unit tests: matching-binder filter discards; non-matching binder ⇒ unfiltered; uncontracted ⇒ unfiltered; starved ⇒ skip; seeds still derived per `TestRunner_AllThreePropertyPathsUseDerivedSeed`

**Phase 2: Named failing inputs** (~1 day)
- [ ] `formatBinderInputs`; `FailingInput` set on all four paths (forall fail, forall eval-error, ensures fail, requires skip)
- [ ] JSON + human reporter assertions (`failing_input` key, "Failing input:" line); eval-error messages carry values
- [ ] Keep `TestRunEnsuresProperty_ViolationReportsCounterexample`-style `error`-text assertions green

**Phase 3: Attested agreement** (~1 day, blocked on D5)
- [ ] Land m-package-test-discovery (its own implementation notes)
- [ ] `runAttestedChecks` union file count; note text removal
- [ ] e2e: package fixture with an inline-only failing property ⇒ `ailang test .` exit 1 AND `pkg quality` PUB012 gate naming the same property

**Phase 4: Docs + prompt** (~0.5 day)
- [ ] `docs/docs/guides/testing.md`: inline forall properties honor the function's requires; the "write it as an if" convention remains the mechanism for top-level properties and non-aligned binders
- [ ] Changelog entry naming the behavior change

### Files to Modify/Create

**Modified files:**
- `internal/testing/runner.go` — Leg A filter + loop shape; Leg B `FailingInput` on forall paths (~120 LOC)
- `internal/testing/contract_domain.go` — Leg B `FailingInput` on ensures/requires paths; export/reuse alignment helper (~30 LOC)
- `cmd/ailang/pkg_quality.go` — union file count, note text (~15 LOC)
- `cmd/ailang/test.go` — per m-package-test-discovery (union discovery, shared aggregation helper)
- `docs/docs/guides/testing.md` — domain semantics for inline forall (~25 LOC)
- Tests: `internal/testing/runner_ensures_test.go` (or a sibling `runner_forall_domain_test.go`), `cmd/ailang/pkg_quality_test.go`, e2e fixture package

## Examples

### Example 1: The decisive fixture (from the live repro, V5)

**Before** (v0.52.5, `ailang test guarded.ail`):
```
⊘ gwidth_property_1 (1 cases)      requires case: skip
   requires not satisfied by random input …: r=-781.6696416668731
✓ gwidth_property_2 (100 cases)   ensures case: pass (accepted 100, discarded 102)
✗ gwidth_property_3 (2 cases)      forall case: FAIL
   property failed on input: [-2.3580454226813388e-148]
```

**After** (expected):
```
⊘ gwidth_property_1 (…)           unchanged skip
✓ gwidth_property_2 (…)           unchanged pass
✓ gwidth_property_3 (100 cases)   accepted 100, discarded ~100, generated ~200
```
The property reads `forall(r: float) => gwidth(r) >= 0.0`; binder `r` aligns with param `r`; `requires { r >= 0.0 }` discards the negatives the predicate was never meant to see.

### Example 2: A failing case names its input (Leg B)

**Before** (`ailang test --json relativity.ail`):
```json
{"name":"expm1_property_1","status":"fail","error":"property failed on input: [866.7105463623886]","seed":"-6178781830021800503"}
```
**After** (expected):
```json
{"name":"expm1_property_1","status":"fail","error":"property failed on input: x=866.7105463623886","failing_input":"x=866.7105463623886","seed":"-6178781830021800503"}
```
And the evaluation-error path changes from `test 37: evaluation failed: …` to `test 37 (x=…, y=…): evaluation failed: …` with `failing_input` set.

### Example 3: Quality agrees with the directory run (Leg C)

**Before** (same directory, same binary): `ailang test .` exits 1 (2 property failures) while `ailang pkg quality .` prints `tests: [attested] 1 files, 2 passed, 0 failed` and `✓ no gates` (V2).
**After** (expected): quality's attested run executes the union, reports the failing properties, and raises `PUB012 … 2 test(s) failed` as a publisher-mode gate.

## Success Criteria

- [ ] The `gwidth` fixture: requires case skip / ensures case pass / forall case pass with discards counted (unit test)
- [ ] Non-aligned binder fixture (`contracts_and_properties.ail` unchanged behavior) still green (regression test)
- [ ] `failing_input` present in `--json` and "Failing input:" in human output for all four failure/skip paths (unit + reporter tests)
- [ ] Evaluation-error messages include the generated values (unit test)
- [ ] e2e: inline-only-failing package ⇒ `ailang test .` exit 1 AND `pkg quality` PUB012 gate naming the same property
- [ ] `make test-core` green; seed-determinism e2e tests green unmodified
- [ ] `docs/docs/guides/testing.md` updated; changelog entry present

## Testing Strategy

**Unit tests:**
- Filter: aligned binders discard violating inputs (deterministic, seeded); non-aligned and uncontracted produce byte-identical results to pre-change goldens captured in the test
- Starvation: requires accepting <10% of the range ⇒ `SkipKindOutOfContract`, `DiscardedInputs` populated
- `FailingInput` set on: forall false, forall eval-error, ensures violation, requires skip

**Integration tests:**
- e2e package fixture (inline-only failing property) driving both `ailang test .` and `ailang pkg quality --json` and asserting both name the property and fail/gate respectively
- Seed parity: same inline property via `--package`, via single file, via directory run reports the same seed (already covered by m-package-test-discovery's plan; keep)

**Manual testing:**
- Re-run the report's scenario shape (celestial/relativity-style package) and confirm the three verdicts agree with author intent

## Verification Log

Every load-bearing claim above, with the command that proves it. All commands run 2026-10-09 against the v0.52.5 binary at `/usr/local/bin/ailang` (repo `std/VERSION` = v0.53.0) unless marked "code read".

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `ailang test .` walks all `.ail` files incl. source modules; the repro package fails 2 inline properties | Live run: `expm1_property_1` fail on `[866.7105463623886]`, `ringWidth_property_1` fail on `[-1.564882392817016e-148]`, exit 1 |
| V2 | `ailang pkg quality .` attests 0 failed on the same directory | Live run: `tests: [attested] 1 files, 2 passed, 0 failed`, `✓ no gates`, exit 0 |
| V3 | Out-of-domain floats are generated in-range but outside the function's domain (NaN/Inf results) | Live run (V1): `exp` overflow at x=866.7 ⇒ `Inf−Inf=NaN`; `sqrt` of a negative ⇒ NaN ⇒ comparison false |
| V4 | Per-file and directory runs agree (mode-stable seeds); per-file on the source module reproduces the same failures | Live: `ailang test relativity.ail` exits 1 with the same two failures; `ailang test relativity_test.ail` exits 0. Code: `DeriveSeedV1` + `ResolveModuleIdentity` (declared module ⇒ path-independent), `cmd/ailang/commands_language.go` builds one `TestConfig{WorkspaceRoot: cwd}` for both modes |
| V5 | One function, three verdicts: requires⇒skip, ensures⇒pass (discards), forall⇒fail (no filter) | Live run of the `gwidth` fixture (quoted verbatim in Example 1) |
| V6 | `pkg quality` attests via `ailang test --package`, which runs only `*_test.ail` | Code read: `cmd/ailang/pkg_quality.go` `runAttestedChecks` → `exec.Command(bin, "test", "--package", "--format", "json", ".")`; `cmd/ailang/test.go` `runPackageTests` walks `strings.HasSuffix(p, "_test.ail")` only; note string "inline test blocks are not yet discovered in package mode (m-package-test-discovery)" |
| V7 | `FailingInput` is never assigned | `grep -rn "FailingInput\s*=" internal/ cmd/ --include="*.go" \| grep -v _test.go` → no matches (exit 1); only `result.go:40` (declaration) and `reporter.go:127,223` (render) mention it |
| V8 | Forall path has no requires filter | Code read: `internal/testing/runner.go` `runProperty` — no `allRequiresHold` call (it exists only in `contract_domain.go`, used by the ensures path); confirmed behaviorally by V5 |
| V9 | Float/int generator ranges are fixed `[-1000, 1000]` | Code read: `internal/testing/generator.go` `DefaultConfig()`; documented: `docs/docs/guides/testing.md` "Integer and Size Ranges" ("These ranges and the 100-case count are fixed") |
| V10 | There is no `where`/guard syntax on forall; convention is if-encoding in the predicate | `docs/docs/guides/testing.md`: "A precondition: there is no `where`/`==>`, so write it as an if"; parser: `internal/parser/parser_testing.go` `parseProperty` accepts only `forall(binders) => expr`, `ast.Binder` has Name/Type/Pos only; `where` absent from the reserved-keywords table in the teaching prompt |
| V11 | m-package-test-discovery is planned and unlanded | `design_docs/planned/m-package-test-discovery.md` exists (Status: Planned, Target v0.38.x); `std/VERSION` = v0.53.0; corroborated by `pub012-inline-tests-false-absence.md` ("still unlanded at v0.41.0") |
| V12 | The requires property case skips on the first violating input; the ensures case discards up to 1000 attempts for 100 accepted | Code read: `runner.go` `runRequiresProperty` returns Skip on first `!boolVal.Value`; `contract_domain.go` `runEnsuresProperty` `requiredAccepted=100, maxAttempts=1000`; both observed live in V5 |
| V13 | Cited fixtures exist | `ls`: `internal/format/testdata/contracts_and_properties.ail`, `examples/inline_tests_best_practices.ail`, `internal/testing/runner_ensures_test.go`, `cmd/ailang/test_seed_e2e_test.go`, `cmd/ailang/pkg_quality_test.go`; test bodies read for the assertions quoted in Semantic Conflict Surface |
| V14 | The failing properties live in source modules, not `*_test.ail` (report deduction) | Forced by V6 + V4: quality ran every `*_test.ail` and attested 0 failed; per-file runs of those files are deterministic and green; therefore no `*_test.ail` file contains a failing property. The names `<func>_property_N` are produced only by the inline-property collector (`collector.go` `collectInlineTests`) |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether the alignment check also requires positional/type-structural agreement beyond name equality (name-only is the design minimum) — agent may choose, document in code
- Exact wording of the eval-error message (`test 37 (x=…)` vs `case 37, input x=…`) — agent may choose, must contain binder names and values
- Whether `TestsSection.Files` counts inline-bearing source modules separately or reports a combined count with a note — agent may choose, must not regress `TestPkgQuality_JSONReportOnFlatFixture`
- Residual edge: an all-skipped union suite makes `ailang test .` exit 1 (M-NAMED-TEST-BLOCKS) while quality records skips without a gate — accept for now or align; agent may propose, human decides at review

## Non-Goals

**Not attempted in this feature:**
- `where`-guard syntax on forall binders — a parser/prompt/fmt change with a much larger conflict surface; the requires-alignment filter covers the reported class without new syntax (D4)
- Domain-aware generators (deriving generator ranges from requires, Z3-guided generation) — future work; the discard loop is the bounded, deterministic 80%
- Changing the requires property case's first-violation skip semantics — M-DX26 Phase 5.2 ruled that behavior deliberately; only its `FailingInput` population changes here
- NaN/Inf-aware comparison semantics in predicates — IEEE semantics are the language's; properties that need tolerance must say so
- `runTestsV2`/`runPackageTests` aggregation dedup beyond what m-package-test-discovery already designs

## Timeline

**Week 1** (4 days):
- Phase 1 (1.5d), Phase 2 (1d), Phase 3 (1d, concurrent with m-package-test-discovery landing), Phase 4 (0.5d)

**Total: ~4 days across 1 week** (plus m-package-test-discovery's own 2-day estimate, shared release)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Behavior change: forall properties that "passed" only by luck of the seed, or failed on out-of-domain inputs, change verdicts | Med | Intended change; changelog entry; the unfiltered path is preserved verbatim for non-aligned/uncontracted properties (goldens in tests) |
| External goldens pinning forall case streams on requires-bearing functions break | Low | Seed derivation is unchanged; only the draw-consumption pattern changes for the filtered subset; document in changelog |
| Publishers suddenly blocked by PUB012 on inline properties quality never ran before | Med | That is the agreement the report asks for; the failure now names its input and the fix (tighten requires or the property) is local; release note with the migration shape |
| Alignment rule too weak (binder renamed ⇒ filter silently off, property fails again) | Low | The failure still names its input (Leg B), and the docs state the rule; a future `where` syntax is the general escape hatch |
| m-package-test-discovery slips again and Leg C lands without it | High | Design freeze D5 makes it a same-release dependency; without it this doc ships Legs A+B only and the disagreement persists — explicitly not "done" |

## Related Documents

**Planned (check for overlap):**
- [m-package-test-discovery](../m-package-test-discovery.md) — Dependency, not duplicate: it rules the discovery union (Leg C); this doc rules the domain semantics (Leg A), diagnostics (Leg B), and the agreement contract that needs both. Its scope is `cmd/ailang/test.go` discovery; this doc's `internal/testing` scope does not overlap.
- [ailang-core-triage/pub012-inline-tests-false-absence.md](../ailang-core-triage/pub012-inline-tests-false-absence.md) — the triage report that measured the discovery gap live; its interim (PUB012 message text) is subsumed by Leg C.

**Implemented (may inform design):**
- [v0_31_0/m-property-generator-coverage](../../implemented/v0_31_0/m-property-generator-coverage.md) — the generator coverage lanes; its `contract_domain.go` wiring is what Leg A reuses
- [v0_31_0/m-property-seed-determinism](../../implemented/v0_31_0/m-property-seed-determinism.md) — the frozen `DeriveSeedV1` contract this design must not touch
- [v0_40_0/m-pkg-quality-ladder](../../implemented/v0_40_0/) — the attested/server provenance split; Leg C changes only what the attested run executes, not who banks it

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `docs/docs/guides/testing.md` — current property-testing semantics and the if-encoding convention
- `internal/testing/runner.go`, `contract_domain.go`, `generator.go`, `config.go` — the harness code this design changes
- User report: AILANG v0.52.0, sunholo-data/ailang-packages@599a77d (packages/celestial 3 failures, packages/relativity 7)

## Future Work

- `where`-guard syntax on forall binders (`forall(r: float) where r >= 0.0 => …`) — the general domain mechanism for properties that do not align with a function's parameters; requires the full parser/prompt/fmt conflict-surface treatment
- Domain-aware generators: narrowing generation (not just filtering) from requires clauses, bounded by the same determinism rules
- Surfacing discarded-input statistics in `pkg quality`'s attested block so registry consumers can see how much of a property's run was in-contract

---

**Document created**: 2026-10-09
**Last updated**: 2026-10-09

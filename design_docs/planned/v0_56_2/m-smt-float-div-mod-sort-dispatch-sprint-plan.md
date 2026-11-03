# M-SMT-FLOAT-DIV-MOD-SORT-DISPATCH Sprint Plan

**Design:** [approved design](m-smt-float-div-mod-sort-dispatch.md)  
**Status:** Planned; implementation awaits coordinator plan approval  
**Target:** v0.56.2  
**Duration:** 3 working days, approximately 10 focused hours including buffer  
**Risk:** Medium — SMT binder scope, metadata lifetime, and nonlinear solver behavior  
**Scope:** Planning only in this handoff; internal/smt implementation plus CLI regression tests and release documentation during execution.

## Goal and rulings

Make the two reported float division contracts VERIFIED, encode finite nonzero Float remainder by truncated Real quotient, and fix dictionary String ordering. Preserve Int encodings and return honest skips when sort evidence is unavailable.

The approved design freezes D1–D5: plain Real division with no new denominator side condition; exact D2 remainder expansion; DictRef/CoreTI/literal dispatch with unknown-sort skip; parent M1 A1+A2 slice only; no runtime or pipeline changes. There are no open human-decision gates. Do not reopen the frozen decisions. IEEE-754 rounding, Inf/NaN, obligation hoisting, and parent M2–M4 remain deferred. Division-by-zero is solver-unspecified under the existing Real model; document this limitation without claiming IEEE equivalence.

## Current status and estimates

Audit base: 6a527f7f, std/VERSION v0.56.1, clean worktree. codegen_operators.go still routes division/modulo without inspecting sort; types.go still maps mod_Float to Int mod. EncodeFunctionOpts.CoreTI and existing SMT/CLI regression helpers are available. The approved design audited sibling cases and all emitter paths; this plan follows that systemic scope.

The seven-day velocity script ran, but this shallow checkout contains only the design commit and no usable implementation LOC/day history. Historical changelog grep results are not recent velocity. Capacity is an explicit estimate: 600 changed/new LOC (roughly 200 implementation and 400 tests/documentation), 200 LOC per scheduled day. Five 60–90 minute milestones total about 7.5 hours, plus 2.5 hours buffer for solver and corpus work. The design's three-day envelope is retained; these are estimates rather than measured velocity.

codegen.go is 692 lines: add at most a small context hook there. Put new helpers in resolve_sort.go/codegen_real_arithmetic.go; if codegen.go approaches 800 lines, use a companion codegen_type_context.go. codegen_operators.go (133) and types.go (443) have room. New tests use separate companions, not expansions of large existing suites.

## Registry reuse audit

`ailang pkg search smt` returned no packages. No candidates required pkg info/docs inspection. Every milestone records `none`: this is compiler-internal Go SMT encoding and its regression coverage, which AILANG registry dependencies cannot supply. Reuse the existing local SMT encoders, CoreTI plumbing, StringBuiltinSpecial, Z3 tests and CLI fixtures. No package dependency or registry contribution is required.

## AILANG syntax and examples

AILANG prompt version loaded: v0.16.11 (reported by `ailang prompt --version-active`; full `ailang prompt` loaded during planning). The available binary warns it may be stale while repository std/VERSION is v0.56.1. Executor must build the current repository binary, load its prompt and check fixtures before treating any syntax or verifier output as current.

All showcase modules live as embedded `.ail` sources in cmd/ailang/verify_float_div_e2e_test.go; no standalone runnable example is added.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| Reporter tidalRadialOrbit / tidalTransverse | include: float -> float, positive-radius requires and result > 0.0 ensures; exact report | skip: pure functions have no effects | skip: Z3 e2e and dyadic Go/SMT tests cover behavior |
| Float remainder | include: float -> float, divisor > 0.1 requires and result >= 0.0 ensures for numerator 7.5 | skip: pure arithmetic | skip: signed remainder table lives in Go tests |
| String ordering | include: String comparison pre/post contracts with each ordering alias exercised | skip: pure comparisons | skip: Go/SMT table covers ordering |
| False division control | include: nonzero denominator requires and intentionally false ensures | skip: pure arithmetic | skip: expected counterexample asserted by Go e2e |

## Milestones

### M1: Sort resolution and Real division (~180 LOC)

**Duration:** 60–90 minutes. **Dependencies:** none.

Create internal/smt/resolve_sort.go and internal/smt/float_divmod_test.go. Update codegen_operators.go and install/restore CoreTI in codegen.go. Resolve div/mod through one type-directed path, initially landing Real division; retain the existing Int sentinel expansion unchanged. Exercise DictRef types, typed variables, literal kinds, unsupported types and missing metadata. Include direct and dictionary emitter forms and arity regressions. Use save/restore with defer on every return path.

**Acceptance criteria:**

- [ ] Float division uses Real / in DictApp, Intrinsic, BinOp and builtin App forms; numeric Int encodings remain byte-identical.
- [ ] CoreTI-based variable types and literal fallback resolve correctly; unknown or unsupported operand sorts return ErrUnencodableTypes.
- [ ] Encoder type context restores after success and error; missing CoreTI does not inherit a prior function context.
- [ ] Z3-backed dyadic division cases cover positive and negative operands; shape tests reject Int div/abs in Real division.

**Validation:** Run the relevant new tests and existing Int div/mod regressions for SMT changes; CLI milestone runs its new TestVerify cases. M5 runs the aggregate focused gates below.

### M2: Exact Real remainder (~160 LOC)

**Duration:** 60–90 minutes. **Dependencies:** M1.

Add internal/smt/codegen_real_arithmetic.go and internal/smt/float_mod_test.go. Update types.go, codegen_apps.go and the shared resolver as needed. Use an outer parallel operand let and an inner quotient let: the design After example puts q beside its dependencies in one parallel let, which is invalid SMT scoping. Implement the prose D2 formula with correct nested scope. Keep both denominator signs and truncation toward zero; do not assert a new nonzero obligation. Test supported direct, curried and pre-lowered shapes.

**Acceptance criteria:**

- [ ] mod_Float leaves the generic Int mod table and reaches the Real remainder expansion through direct and curried App paths.
- [ ] Real remainder matches math.Mod on finite nonzero dyadic sign pairs, including negative quotient fractions and zero numerator.
- [ ] Nested lets bind numerator and denominator before quotient; operands encode once and nested divisions cannot capture binder names.
- [ ] Intrinsic, BinOp, dictionary mod when supported, and builtin forms share dispatch; existing Int div/mod tests pass unchanged.

**Validation:** Run the relevant new tests and existing Int div/mod regressions for SMT changes; CLI milestone runs its new TestVerify cases. M5 runs the aggregate focused gates below.

### M3: String comparison dispatch (~100 LOC)

**Duration:** 60–90 minutes. **Dependencies:** M1.

Create internal/smt/string_compare_dispatch_test.go; update codegen_operators.go and the shared resolver. Reuse StringBuiltinSpecial behavior for reversed operators. Audit every dictionary comparison alias and preserve Eq methods. Keep Intrinsic/BinOp non-div/mod ordering expansion outside this slice, as scoped by D4.

**Acceptance criteria:**

- [ ] DictApp String lt/le/lte/gt/ge/gte use str.< or str.<= with correct operand flipping.
- [ ] Int and Float comparisons and Bool/String equality retain their existing supported semantics.
- [ ] Unknown dictionary types and unsupported operator/type pairs skip honestly instead of selecting numeric ordering.
- [ ] Z3-backed ASCII string ordering cases cover equal, prefix, less-than and greater-than inputs.

**Validation:** Run the relevant new tests and existing Int div/mod regressions for SMT changes; CLI milestone runs its new TestVerify cases. M5 runs the aggregate focused gates below.

### M4: Verifier end-to-end regressions (~120 LOC)

**Duration:** 60–90 minutes. **Dependencies:** M2, M3.

Create cmd/ailang/verify_float_div_e2e_test.go with buildAilang/writeTempAil/verifyStatuses; strengthen the existing helper only if its sort-mismatch check is missing. Embed the exact reporter functions, plus a positive-divisor remainder contract and String comparison contracts. Add a deliberately false, nonzero float division contract as a negative control. Keep fixtures in Go test strings so examples manifest updates are unnecessary.

**Acceptance criteria:**

- [ ] TestVerify_FloatDivSortDispatch verifies both tidalRadialOrbit and tidalTransverse from the report.
- [ ] TestVerify_FloatModSortDispatch and TestVerify_StringOrderSortDispatch verify the remainder and String-order contracts.
- [ ] verifyStatuses rejects sort mismatch and unknown constant output; false float contracts still produce counterexamples.
- [ ] All new embedded AILANG fixtures pass ailang check using the freshly built repository binary.

**Validation:** Run the relevant new tests and existing Int div/mod regressions for SMT changes; CLI milestone runs its new TestVerify cases. M5 runs the aggregate focused gates below.

### M5: Focused validation and release documentation (~40 LOC)

**Duration:** 60–90 minutes. **Dependencies:** M4.

Create changelogs/unreleased/2026-11-03-smt-float-div-mod-sort-dispatch.md beginning with ### . Update design completion evidence and move it to design_docs/implemented/v0_56_2/ on completion. Run existing focused gates and contract corpus workflows; do not add a new verifier preflight or a broad CI lane. Record solver version, commands, statuses and failures. Reporter-side sunholo/relativity publish remeasurement is external follow-up, not permission to publish a package from this sprint.

**Acceptance criteria:**

- [ ] make test-changed passes with an available TEST_CHANGED_BASE, and touched SMT/CLI tests execute with Z3 rather than all solver tests skipping.
- [ ] make check-file-sizes, make check-changelog, make verify-stdlib and make verify-examples pass; unrelated baseline failures are recorded explicitly.
- [ ] basic.ail and insurance.ail retain baseline verification statuses, including the known applyDiscount violation; std and runnable contract JSON output contains no sort mismatch or unknown constant.
- [ ] An unreleased changelog fragment documents Float division/remainder and String ordering repairs, unknown-sort skips, and the existing mathematical-Real model limitations.
- [ ] Design moves to implemented/v0_56_2 only after acceptance evidence is recorded; std/VERSION and publish gating stay unchanged.

**Validation:** Run the relevant new tests and existing Int div/mod regressions for SMT changes; CLI milestone runs its new TestVerify cases. M5 runs the aggregate focused gates below.

## Day-by-day execution

- Day 1: M1 and M2; establish failing tests, implement dispatch and Real remainder, pin Int parity.
- Day 2: M3 and M4; comparison aliases, reporter repro, positive and negative verifier controls.
- Day 3: M5 and buffer; corpus baselines, focused gates, changelog and completion evidence.

## Validation and completion

Use a freshly built CLI and record `z3 --version`. First run targeted `go test ./internal/smt` cases and `go test ./cmd/ailang -run 'TestVerify_(FloatDivSortDispatch|FloatModSortDispatch|StringOrderSortDispatch)' -count=1`. Then run `make test-changed` (includes test-core). In a shallow executor checkout set TEST_CHANGED_BASE to the recorded pre-implementation commit if origin/dev is absent; do not interpret an unavailable base as an empty diff. Run `make check-file-sizes`, `make check-changelog`, `make verify-stdlib`, and `make verify-examples`. CI owns the full suite.

Use existing verifier/corpus commands on std and runnable contract examples, inspecting JSON for sort mismatch/unknown constant diagnostics; verify-examples checks examples and manifests but is not itself proof that every contract was solver-checked. Record basic.ail/insurance.ail statuses before and after; retain the known applyDiscount counterexample. Target at least 90% coverage of added resolver/Real arithmetic branches, measured with focused SMT coverage. Any skipped Z3 tests require installing/using the solver before declaring semantic acceptance.

Risks: Real remainder mixes Real and Int terms; inspect the emitted sorts and use Z3 equivalence tests. Nonlinear proofs can time out; record timeouts and solver settings rather than accepting them as VERIFIED. Keep Real division/remainder tests finite and nonzero, with dyadic inputs for exact Go comparisons. No claim is made about NaN/Inf or float rounding. Helper-call type metadata must remain coherent; add a typed helper-call regression if the resolver sees inlined bodies.

## Handoff

Plan approval/merge in the coordinator starts sprint-executor; this planning task does not start implementation. Both artifacts carry the same milestone IDs, dependencies and estimates. Do not link or close ailang-core issue #173: it is a PR in ailang-packages. Parent PRs/issues in the design are context, not current issues to auto-close. Record reporter remeasurement of sunholo/relativity as an external follow-up; local reporter-contract verification is required now.

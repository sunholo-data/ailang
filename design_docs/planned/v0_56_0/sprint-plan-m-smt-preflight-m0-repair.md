> Execution approved: user “exeute all three sprints”, observed 2026-10-29T20:17:42Z. All six reviewed M0 milestones authorized; historical quorum records remain BLOCKED and unchanged.

# Draft sprint plan: M-SMT-PREFLIGHT-M0-REPAIR

**Status:** DRAFT for review; authorized planning only. Leave this plan, progress JSON and approval annotations uncommitted until approved. No implementation, provider review, PR/push, coordinator dispatch or parked-work recovery is authorized.

**Design:** [canonical M0 design](../m-smt-well-sorted-preflight.md), current M0 decision section and M0-A1–A8 only. **Issues:** #1960/#1961. **Target:** unreleased v0.56.0; no version bump/release requested. **JSON:** `.ailang/state/sprints/sprint_M-SMT-PREFLIGHT-M0-REPAIR.json`.

The user said **“approve the three designs”**, observed at **2026-10-29T14:48:58Z**. That ruling ratifies corrected evidence and M0-F1/F4/Z at approved design commit `88ca9e1e974150fa783d19b15b8a93be1682e2f7`, for planning only. Observation time is the assistant receipt record, not an independent send timestamp. Root's local `three-design-approvals.json` is the source. Both external BLOCKED records remain unchanged; human ratification is separate and never reported as external PROCEED. M0-F1/F4/Z are rulings, not questions to reopen. Plan approval and explicit **execute sprint** remain pending.

## Goal and scope

M1–M6 below are this repair sprint's milestone names; they do not authorize the parent design's deferred roadmap M1–M4.

Remove the Int Euclidean/truncating mismatch and contract-call result collapse while retaining functional identity and finite, lexical, selected-path callee facts. Reuse only the sound S1 expansion and selected UF/arity/model plumbing after execution approval; replace the parked global `forall` implementation. No code is recovered as-is merely because historical metadata says complete.
No external consumer repository or source-pin, prompt-version, dependency or release change is planned.

The zero policy is ruled: runtime RT001 remains; verification retains its partial-correctness totalization, without injected nonzero assumptions, a runtime-error-freedom guarantee or new skip/error rules. Call-site requires proof obligations, proof-versus-refutation/admission redesign, width/BitVec, new encodings, F2/F5/encoder_gap and parent-design M1–M4/#1950/#1953 are excluded. Operator-held #1790 and runner work (#1926/#2013) remain other owners' scope.

## Current status, evidence and ownership

Planning checkout is the docs branch at `88ca9e1e9`, source base `aca5a6ead`. Do not merge a new base into these dirty drafts. After explicit execution approval, use a **new isolated execution worktree/branch** from actual fresh `origin/dev`, recheck PR/task ownership read-only through gh and existing read-only coordination tools, then carry only the approved design/plan/state artifacts. Completed-doc archive moves landed after aca5; git can misinfer a directory rename. Bank approved draft artifacts first, integrate actual dev cleanly, and explicitly keep this NEW M0 plan/JSON in `planned/v0_56_0`/state while retaining accepted archives. Inspect rename conflicts rather than accepting inferred moves to implemented. Do not switch, build in or edit the parked `p0/smt-unsound-encodings` worktree or its dirty plan/JSON/fmt log.

Historical candidate commits `981a7aee9`/`5410d0b2f` are git-show/archive navigation only. Their arithmetic/plumbing tests must pass against the actual execution base; universal axioms are prohibited. Preserve current #602 fake-solver warm-up, effect-row/explicit-empty semantics, alias ownership, IFCv2/v8/cache metadata and later landed source changes. Coordinate actual overlapping SMT hunks with #1790; do not adopt its admission/purity behavior.

Evidence has three distinct compiler states:

1. **Historical 807:** archived binary full `80793dbcf42423027be4bab1a0088da54e2323d8`, SHA256 `721af9c6262613a03267e7a1ea47884bffa36b0bae36eb5d64721eb6ad6af2fa`, pinned std tree `1cc110f4b021b7c6a3ecc4940aab6eaf483a68aa`. Its executed outputs are historical, including false parity/halfDown/gap rows; it is not a new current/fixed build.
2. **Actual current baseline:** build the exact fresh execution-base source with matching std after execution approval. Record full commit/tree, binary SHA/version, std tree/file hashes, Go/platform/Z3 identity, cache mode and full command/stdout/stderr/JSON rows. Re-establish the original false rows on this binary; if intervening work already fixed them, reconcile ownership/scope before duplicate implementation.
3. **Fixed exact head:** build the final execution head with its std and run the same immutable fixtures/corpus. Build scratch mutation binaries separately; never replace global `ailang` or mutate another worktree.

Literal807→aca5 named-path/front-half source comparisons are empty, but four pipeline cache files/seven tests differ. The installed quorum binary is stamped e863/dirty, not clean-build-attested. These qualifications remain; prior raw Z3 controls are **not** parked/fixed CLI proof. Both external reviews and full Gemini concern stay visible in [evidence](../m-smt-well-sorted-preflight-m0-evidence.json).

Owned implementation surface, only after execute: `internal/smt` arithmetic maps, call-contract/callee encoding and necessary narrow integration; `cmd/ailang` companion e2e tests/testdata; verifier guide/changelog/evidence. No parser/type/effect/iface/runtime semantics or schema/epoch change. Shared near-size files stay bounded: `codegen.go` 666 lines, `callee_resolver.go` 586, `codegen_apps.go` 407 at planning base. Prefer companions `codegen_call_facts.go`, `codegen_call_facts_control.go`, `codegen_call_facts_helpers.go` and matching focused test files, rather than growing an existing file past 800. Keep new tests split by semantic surface as well. Candidate companions include `codegen_call_facts_contract_test.go`, `codegen_call_facts_control_test.go`, `codegen_call_facts_helpers_test.go` and separate `verify_soundness_m0_{arithmetic,lexical,helpers}_e2e_test.go`; do not add large blocks to shared CLI main_test.go.

## Estimates and sequencing

**Estimate:** 1540LOC total: 580 implementation,900 meaningful tests,60 docs; approximately 3 engineering days/12–16 active hours plus CI wall time. **Risk:** high for proof soundness, medium for integration. Each implementation milestone targets 60–90 minutes of focused work; corpus/CI observation has separate wall time. If a milestone is larger in practice, bank a coherent tested checkpoint and report actual progress; do not omit criteria.

Historical S1 added 228/deleted 11 lines; S2 added 516/deleted 270, including tests and removed constant substitution. The safe finite-fact mechanism is more work than that historical patch. `analyze_velocity.sh 7` was read/run offline; its first changelog metrics are historical and its diff summary is latest-commit churn, not valid measured delivery throughput. The estimate therefore uses itemized scope and a roughly 515 LOC/day planning allowance with integration buffer, not an invented recent productivity measurement.

Dependencies are serial **M1→M2→M3→M4→M5→M6**, because shared SMT state and call-fact representation cross these changes. No duplicate parallel source workers are planned; independent final evaluation remains separate.

## Registry reuse and syntax gates

This is compiler-private Go SMT translation over Core, not a package-like capability. Package imports cannot replace encoder operators, lexical proof-fact plumbing or solver-model parsing. Registry search/info/docs is **not applicable**; no registry query/mutation occurs. Each milestone records `none` with this concrete reason, not a placeholder.

**AILANG prompt version loaded: v0.16.11**, via the source-matching archived 807 CLI `prompt` and `prompt --version-active`, after clearing ambient export environment. Installed e863 default reports v0.16.8; that discrepancy is retained rather than called current syntax. No `.ail` file is written by planning. Executor must reload the prompt from its actual built compiler/current pinned prompt before fixture edits; preserve frozen prompt bytes.

All planned `.ail` modules below are verifier regression fixtures, not showcase modules. **Contracts: include** their specified true/false requires/ensures. **Effects: skip** extra effect rows because these fixtures are pure and exercise verifier semantics. **Inline tests: skip** because their deliberately false contracts and CLI/model/exit assertions belong to the Go verifier harness; no new demo is claimed complete without tests.

## M1 — actual baselines and truncating Int arithmetic

**Estimate:** 100 implementation +160 tests = 260 LOC; 60–90 min core work. **Dependency:** none. **Examples/fixtures:** `cmd/ailang/testdata/verify_preflight/m0_int_divmod.ail`, `m0_runtime_signs.ail`, `m0_zero_domain.ail`.

- Build/pin actual current baseline and archive historical evidence separately. Check original fixtures and retain all per-function statuses, models, SMT and aggregate exits.
- Enumerate all existing Int producer shapes at execution head: builtin and curried `div_Int`/`mod_Int`, Intrinsic, BinOp, DictApp div. The existing DictApp map lacks mod; do not add unsupported methods or Float resolution.
- Reuse centralized inline parallel-let truncating quotient/remainder from S1 after validation. Preserve operand hygiene and existing undefined-zero/width boundary.
- Pin 12 signed pairs per operator, interpreter and `--bytecode --strict-bytecode`, nonzero divisor sign grid, nested divisions and generated/source binder controls. Explicitly compare executed runtime with the model table, not a formula against itself.

**Acceptance:** every Int emitter agrees on nonzero inputs; original parity/halfDown false rows become counterexamples with negative x; halfNonNeg/parityBounded/halfTowardZero/remSign true controls verify where applicable. `/0` and `%0` still check/run RT001; zero-only/nonzero/unselected-zero verification follows approved M0-Z. No global `d!=0` assumption. Old/current/fixed distinctions and all rows/exits retained. Existing focused SMT tests pass; no held runtime/Float/purity fix.

## M2 — UF identity and finite guarded call specifications

**Estimate:** 110 implementation +120 tests = 230 LOC; 60–90 min. **Dependency:** M1. **Fixtures:** `m0_callee_identity.ail`, `m0_requires_fallback.ail`, `m0_partial_vacuity.ail`.

- Carry one UF value per resolved contract-only callee and actual parameter tuple, with arity/type checks using existing classifications. Reuse selected 5410 plumbing, never its global quantified assertion.
- Build actual-call specification templates: `requires(args) => ensures(args,result:=UF(args))`. Requires is neither silently assumed true nor discharged as caller obligation.
- Unsupported/unencodable requires means no ensures fact for that invocation; unencodable ensures contributes no such clause. Recursive/cyclic predicate expansion must terminate via existing unsupported/fail-weak handling: retain an unconstrained result or existing unsupported outcome as applicable, never an infinite expansion, invented axiom or new generic ERROR→SKIP rule. Keep unexpected faults loud.
- Pin distinct vs same arguments, parameter-dependent contracts, arity mismatch, unencodable/recursive predicate paths and model shape. These unit templates do not claim full CLI integration before later milestones.

**Acceptance:** same-call identity holds, distinct-call false equality is refutable, only guarded instantiations constrain results; no global contract `forall`. Recursive/unencodable paths have the expected existing reason/status or fail-weak absence of facts, with no false proof from a missing requires. Partial-callee raw solver discriminator retained separately from pending final CLI proof.

## M3 — value/fact propagation through calls, lets and selected branches

**Estimate:** 130 implementation +150 tests = 280 LOC; 60–90 min. **Dependency:** M2. **Fixtures:** `m0_lexical_facts.ail`, `m0_short_circuit.ail`.

- Introduce the narrow paired value/finite-condition adapter in companions; preserve public callers and existing supported Core handling.
- Combine argument facts; keep `let` facts inside matching bindings and capture-free substitutions. For `if`, assert only selected branch facts, with condition facts evaluated once.
- Source `&&`/`||` already lower to CoreIf; preserve RHS laziness rather than treating them as eager conjunctions of proof facts.

**Acceptance:** nested/shadowed locals, nested arithmetic arguments, conditional partial/divergent callees and selected/unselected short-circuit operands retain bindings; false goals remain refutable and true selected controls verify. No free local at top level and no unselected contradictory fact. Unsupported required forms follow their existing path and are visible for final delta audit, not guessed encodings.

## M4 — match guards and contract-predicate evaluation paths

**Estimate:** 100 implementation +150 tests = 250 LOC; 60–90 min. **Dependency:** M3. **Fixtures:** `m0_match_guards.ail`, `m0_contract_predicates.ail`.

- Propagate scrutinee/pattern/guard facts using existing first-match and binder rules; an unmatched arm, failed reach or later guard after a selected earlier arm cannot leak facts.
- Apply the same finite/selected machinery to calls inside caller body, requires and ensures, including short-circuit predicates. Retain existing admission/sort/name resolution boundaries.
- Add positive and false postcondition controls separately; recursive/unencodable predicate paths must preserve existing unsupported/fail-weak behavior without widening the adjacent proof-vs-refutation feature.

**Acceptance:** match-bound arguments and sequential/nested guards remain scoped; short-circuit in all three expression positions is selective; no branch contradiction creates a false VERIFIED. No new constructor/name resolution, admission rule, reason-code or ai-check exit policy.

## M5 — helper-condition parameter/environment parity and model rendering

**Estimate:** 140 implementation +160 tests = 300 LOC; 60–90 min. **Dependency:** M4. **Fixtures:** `m0_helper_environment.ail`, `m0_helper_diamond.ail`.

- Propagate paired value/condition definitions through already eligible nonrecursive helpers. Definitions themselves must not assert a contract for every input; actual calls instantiate their condition under current lexical/path rules.
- Preserve the **exact** value helper parameter list, including any lifted closure/environment parameters, in the condition helper. Same-callee/same-arguments includes those actual environments; never deduplicate distinct captures into one condition.
- Emit declarations once per resolved helper symbol and reference them at calls; test diamond graphs without recursively cloning bodies. This is declaration-sharing correctness, not a new asymptotic solver/runtime promise.
- Reuse narrow model-parser filtering of non-nullary UF interpretations; keep input/model bindings intact. Eligibility is not broadened, and no new closed-helper restriction is selected.

**Acceptance:** helper condition/value signatures match, distinct environments remain distinct, diamond declarations are unique while every selected invocation retains facts. Same-call identity, interleaved helper call arguments and true/false helper controls pass. Free/cyclic/unencodable forms use existing unsupported/fail-weak paths; Gemini's exact concern is discharged by actual tests, not by design wording. No junk UF definitions displayed as model values.

## M6 — exact fixed-head CLI, mutations, corpus deltas and independent handoff

**Estimate:** 160 tests +60 docs = 220 LOC; 60–90 min evidence assembly, plus bounded corpus/CI wall time. **Dependency:** M5. **Fixtures:** all above, with actual `partial`/`wrong` witness and original old fixtures unchanged.

- Build fixed exact head+matching std; run all original false/true controls via actual CLI, interpreter and strict VM where runtime applies. Keep current baseline and historical807 outputs separate. The partial-callee caller must be a real fixed-CLI counterexample at a returning nonzero input, runtime return 0, runtime-contract failure.
- Disposable mutations: restore Euclidean Int operators, shared callee constant, and separately parked universal-axiom behavior. Relevant tests must fail; the universal mutation must expose the false partial-callee proof. Do not alter the parked dirty worktree or global binaries.
- Run before/after on the **same pinned files**: actual execution-base examples/std plus existing ailang-packages historical pin 8e6bbaf resolved to full SHA/read-only archive at execution. Reuse existing corpus scripts/artifacts where suitable; any needed narrow audit wrapper stays local and preserves complete rows. Exclude invalid package manifests explicitly rather than silently omitting them.
- Produce a per-file/per-function join keyed by identity, retaining full old/fixed JSON/SMT/reason/model and command exits. **Explain EVERY changed verdict**, including inherited ERROR/COUNTEREXAMPLE rows and aggregate exits. Even unchanged inherited errors remain in totals/report; no “all zero” preflight claim. Record timeouts separately; rerun only justified nondeterministic rows, with originals retained.
- Intentional Int/callee proof flips must match runtime/approved model; a true control regression, new false VERIFIED, unexplained ERROR/COUNTEREXAMPLE or widened skip policy is not accepted. Do not fix deferred #1950/#1953 or #1790/runner fixtures to improve counts.
- Update verifier guide, canonical implementation notes and a `### Fixed` fragment under `changelogs/unreleased/YYYY-MM-DD-smt-m0-soundness.md`. Preserve both raw BLOCKED quorum results, ratification and all frozen packet identities.

**Acceptance:** M0-A1–A8 complete on actual exact final head; mutations discriminating; every corpus delta explained; no unrelated runtime/proof policy change. `make test-changed`, focused SMT/CLI suites, lint, sizes, boundaries, changelog, relevant std/example checks pass. Required exact-head remote CI/Sonar and separate independent sprint evaluation must actually pass before acceptance/merge. Executor does not run full `make test`; no coverage threshold/exclusion/policy waiver or fake evaluation. Capture inherited runner failures without repairing another owner's source or repeated broad reruns.

## Hermetic CLI evidence and validation

Reuse existing `runAilangBin`/`testutil.RunBounded` for bounded subprocesses and existing `removeEnv` filtering when constructing explicit command environments; use standard `testing.T.Setenv` cleanup to clear inherited export inputs for the existing bounded runner. Environment-mutating tests must not use t.Parallel. Clear `GOOGLE_CLOUD_PROJECT`, `OTLP_GOOGLE_CLOUD_PROJECT`, all ambient `OTEL_*`, `OTLP_*`, `AILANG_TRACE*` and `AILANG_OTLP_*` export inputs; set matching std/cache mode explicitly. The local audit harness uses equivalent filtering. This is test isolation, **not** a fix to held #2009 or an invented product off switch.

Use existing Z3 availability skips on unsupported CI platforms, OS-neutral paths and complete raw stdout/stderr/status capture. Bind evidence to actual full binary/tree/std hashes and source identity. Run focused tests as source changes; final `make test-changed` once per meaningful final delta rather than repetitive whole suites. Sonar failures require meaningful tests or actual defects, not policy edits. CI is necessary evidence, not a substitute for the per-function discriminator.

## Calendar and success metrics

- Day 1: M1–M2 plus initial M3; establish actual base before source changes.
- Day 2: M3–M5, serial shared-SMT work with focused checkpoints.
- Day 3: M6 integration, mutation/corpus interpretation, docs and independent/CI handoff; wall-clock CI may extend.

Success means every design M0 criterion measured, every status delta accounted for, no new false proof and required gates/independent evaluation accepted. LOC/coverage estimates are planning aids; retain existing repository coverage thresholds and meaningful branch tests. No implementation milestone is passed now.

## Rulings, exclusions and terminal behavior

There are **no still-open human semantic choices within approved M0-F1/F4/Z**; do not ask again. Deferred F2/F5/parent-design M1–M4/#1790 are exclusions, not prerequisites to silently implement. Plan approval/execution instruction remains the stage gate.

A genuinely different zero-domain, trust/totality, proof-vs-refutation, admission, width or outcome contract is outside this ruling. **STOP: commit only already-authorized coherent work in the isolated execution branch, report `BLOCKED: proposed change exceeds approved M0 scope`, and end the run.** Do not continue by inventing semantics or treating the historical quorum passes/rejections as new user authority. Routine current-base merge/hunk coordination preserving the ruled semantics is implementation work only after execute; do not merge anything into this planning draft.

The parked historical plan/JSON remain untouched. This new repair sprint ID prevents its old completion claims being mistaken for current execution state. Neither the JSON validator's “ready for execution” text nor a plan merge substitutes for the explicit attended execute instruction. No coordinator message/import/ack/dispatch is part of this draft.

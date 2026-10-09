# Sprint Plan: M-PURE-ROW-AND-IFACE-PURITY

**Status:** Planned; awaiting sprint approval through the coordinator.
**Target:** v0.53.0 (next minor after checkout v0.52.5; use v0.53.x if the prerequisite consumes v0.53.0, or v0.54.0 if that minor has closed before execution).
**Design:** [Approved design](m-pure-row-and-iface-purity.md)
**Source handoff:** task-5895e449; planning task task-20636394.
**Issue references:** Refs #1443; Refs #574 (closed, part 1 regression only).

## Summary

Reject contradictory `pure` signatures, derive published purity from checked outer effect rows, and preserve Declassify contract verification through separate SMT admission. Ship source and teaching-material migrations together so the active prompt and benchmark spec describe code the checker accepts.

**Duration:** 4 working days / 32 hours, with a fifth day reserved as 25% contingency (40 hours maximum). Execution time starts after the prerequisite lands. **Risk:** Medium-high because the checker, iface, SMT admission, cache and teaching corpus must change consistently.
**Estimated changes:** 860 LOC: 255 implementation, 495 tests, 110 teaching/docs; generated bundle churn excluded.

## Current Status and Velocity

The clean checkout is `coordinator/task-20636394`, HEAD `2528ac32`, std version v0.52.5. Inspection confirms the unconditional `determinePurity` stub, cache version v5 and keyword-based SMT helper remain. No implementation is included in this planning task.

The seven-day velocity script was run on 2026-10-09. This shallow checkout exposes only the design-document commit, no implementation diff statistics or usable LOC metrics. The current changelog records v0.52.5 on October 7, including serve-api binding fixes and a builtin, but supplies no comparable elapsed-time measurements. Measured LOC/day is therefore unavailable; 215 LOC/day is a planning capacity assumption, not observed velocity. Estimates use the design's revised four-day header rather than its stale three-day Timeline section, with one day contingency for dependency integration and evaluation.

## Execution Gate and Dependencies

M-EFFECT-LATENT-FUNCTION-VALUES must land first: [design](../v0_48_0/m-effect-latent-function-values.md), Refs #1326 and Refs #573; its plan is referenced as PR #1675 in the approved design. This task does not claim that PR or its implementation has merged. Before editing code, the executor records the actual landed implementation commit and remeasures the callback/outer-row schemes on that base. A merged planning PR alone does not satisfy this dependency.

If the latent-tail exception or keyword/iface invariant conflicts with that landed implementation, return the discrepancy for design revision; do not broaden this sprint into row unification or change the approved purity contract. #616 and the inferred generalisation mechanism of #1091 remain out of scope. Rule 1 rejects declared open rows, while Rule 2 retains the approved exception for undeclared callback-shared outer tails.

Before editing `.ail` files, obtain `ailang prompt`. Build the execution binary from the landed source; the installed binary warned it might be stale during registry inspection. Ensure Z3 and configured evaluation model credentials are available; report a missing prerequisite explicitly, never substitute a skipped verification or eval as success.

## Registry Reuse Audit

On 2026-10-09, `ailang pkg search purity` returned no packages. `ailang pkg search effects` returned `world/core@0.1.1` and `sunholo/billing_entitlements@0.4.2`. Both were inspected with `pkg info` and `pkg docs`: billing implements application policy, not compiler row semantics; world implements world-graph contracts and has no AGENT.md (docs command explicitly failed). Neither can enforce compiler signatures, derive Go iface metadata or control SMT admission. No new package-like capability is proposed.

Every milestone records `none`: reuse the existing compiler, iface, SMT, apiserver, formatter, prompt and bundle-generation infrastructure. No registry dependency or contribution is needed.

## Milestones

### M1: Reject contradictory pure signatures (~120 LOC)

**Estimate:** 30 implementation + 90 tests. **Dependencies:** landed latent-function-values implementation

**Files and examples:** `internal/pipeline/validate_effects.go` and pipeline effect-validation tests. Add negative fixtures within the pipeline suite rather than a runnable example advertised as valid. Existing positive examples: `examples/runnable/contracts/cross_module_types.ail` and `examples/intra_package_imports/service.ail`.

**Acceptance criteria:**

- [ ] pure with a labelled or open declared row fails check; diagnostic names the function, contradiction, and both repairs.
- [ ] Explicit empty and absent rows on pure declarations remain accepted; non-pure declarations are unchanged.
- [ ] Pipeline tests cover IO, Debug, Declassify, var-only and mixed open rows, empty rows, and absent rows.

**Risk and mitigation:** Checking only AST effect labels misses var-only rows. Inspect the elaborated declared row, including its tail; preserve normal body-effect checking.

### M2: Derive iface purity and unify consumers (~300 LOC)

**Estimate:** 110 implementation + 190 tests. **Dependencies:** M1

**Files and examples:** `internal/iface/builder.go`, builder tests, `internal/apiserver/routes.go`, MCP integration tests, `internal/pipeline/cache_key.go` and cache tests. Regression examples: `std/ai/streaming.ail`, `std/list.ail` (`map`, `mapE`) and `std/option.ail` (`flatMap`); use generated temporary modules for open-tail controls.

**Acceptance criteria:**

- [ ] Closed empty outer rows report true; labelled and declared-open outer rows report false without reading IsPure.
- [ ] An undeclared outer tail shared with a callback row is latent and reports true; an unrelated tail or any outer label reports false.
- [ ] All six std/ai/streaming exports and std/list.mapE report false; std/option.flatMap reports true; std keyword-pure invariant and approximately 26 combinator pins pass after the prerequisite lands.
- [ ] serve-api uses iface purity without AST overwrite; MCP tags and read-only hints match iface, including open-row cases.
- [ ] Cache version advances once from the current base version; cache round-trip rejects stale always-true metadata and preserves derived values.
- [ ] Non-function export type-walk policy is documented and tested, including values containing function types.

**Risk and mitigation:** Callback tails can be mistaken for declared polymorphism. Track declaration context and callback-row sharing, fail closed on unrelated tails, and pin post-prerequisite schemes. Bump the current cache version once (v5 to v6 on this base), avoiding a stale hard-coded version if another change lands first.

### M3: Preserve SMT verification and migrate source corpus (~330 LOC)

**Estimate:** 115 implementation + 215 tests. **Dependencies:** M1, M2

**Files and examples:** `internal/smt/verify.go`, `encodable.go`, `callee_resolver.go` and admission/callee tests; `internal/apiserver/mcp_tool_hints_test.go`, `internal/types/ifc_declared_test.go`; examples `examples/runnable/contracts/inbox_injection_v2.ail`, `inbox_v2_app.ail`; references `benchmarks/prompt_injection/expected_ailang_safe.ail` and `expected_ailang_injected.ail`. Leave formatter AST-only contradictory fixtures unchanged.

**Acceptance criteria:**

- [ ] SMT entry and callee admission use keyword OR an explicit closed row containing only Declassify or no labels; missing-row non-pure functions are refused.
- [ ] IO, Debug, mixed Declassify/IO and all open rows are refused; Declassify functions never gain IsPure metadata.
- [ ] Migrated safe prompt-injection reference verifies 3 contracts with 0 violations; injected reference reports 1 violation on injectedForward.
- [ ] Seven known contradictory declarations in four example/benchmark files are migrated and formatted; pipeline-backed and IFC Go fixtures use valid declarations.
- [ ] Existing no-row pure functions remain SMT-admitted and cross-module Declassify callees remain inlinable.

**Risk and mitigation:** Rule 1 temporarily breaks the IFC corpus until this milestone lands. Keep all milestones in one implementation PR. Use a distinct SMT-admissibility predicate for both top-level and callee paths; never mark Declassify functions pure.

### M4: Migrate teaching material and validate release (~110 LOC)

**Estimate:** 110 teaching/docs. **Dependencies:** M1, M2, M3

**Files and examples:** `prompts/v0.16.6.md`, `cmd/ailang/prompts/v0.16.6.md`, `docs/docs/prompts/current.md`, `prompts/versions.json`, `benchmarks/prompt_injection.yml`, `docs/docs/guides/ifc-labels.mdx`, `docs/docs/why-ailang.mdx`, `docs/docs/reference/effects.md`, `changelogs/v0.32-current.md`; regenerate `llms.txt`, `docs/llms.txt`, `docs/static/llms.txt`. Check migrated snippets and reuse M3 examples.

**Acceptance criteria:**

- [ ] Active v0.16.6 prompt and embedded/current copies agree after amendment; versions.json hash and note are updated; frozen prompts stay byte-identical.
- [ ] Benchmark contract_spec, IFC guide and why-ailang snippets use valid effectful signatures; three llms.txt bundles are regenerated with make generate-llms-txt.
- [ ] Multiline live-material sweep finds zero rejected spellings outside intentional negative tests, frozen prompts and historical records.
- [ ] prompt_injection eval runs against the amended prompt and spec with model/config/result provenance and failures recorded in the implementation PR.
- [ ] Effects reference and current changelog document the breaking spelling and iface JSON semantic change.
- [ ] make test, make fmt, make lint and make check-boundaries pass; touched examples check and verification outcomes remain pinned.

**Risk and mitigation:** Prompt drift or unavailable model access can mask regressions. Record model, prompt hash, configuration, compiler commit and result paths; report model-generated rejected signatures as measured failures. Do not rewrite frozen historical prompts.

## Day-by-Day Tasks

| Day | Work | Hours |
|---|---|---:|
| 1 | Confirm prerequisite commit, build binary and capture baseline; M1 signature matrix (4h); begin M2 row derivation and tests (4h) | 8 |
| 2 | Finish M2 iface/consumer/cache tests and std census (6h); begin M3 SMT admission (2h) | 8 |
| 3 | Complete M3 callee tests, source migration and both verification outcomes (6h); begin M4 prompt/spec migration (2h) | 8 |
| 4 | Finish M4 docs, prompt hash, bundles, snippet checks and targeted eval (4h); complete full checks and review evidence (4h) | 8 |
| 5 reserve | Resolve integration failures or evaluation/tool availability delays; no extra feature scope | 8 |

M1 completion may leave migration tests failing until M3; milestone completion requires its own criteria, and the implementation PR cannot merge until every milestone and global check pass.

## Validation and Release Evidence

Use targeted Go package tests during implementation, then run the design-required `make test`, `make fmt`, `make lint` and `make check-boundaries` once the integrated change is ready. Capture focused package coverage; require each new decision branch in the signature, row-purity and SMT matrices to be exercised rather than inventing a repository-wide percentage from unavailable baseline data.

Check and format the migrated examples and benchmark reference files. Preserve the five positive regression fixtures listed in the design: list.map, math.isNaN, explicit-empty-row makeCell, imported greet and mapE. Pin both benchmark verification outcomes, ordinary no-row pure admission, Declassify callee admission and negative IO/Debug/open-row controls.

Remeasure all std exported functions before/after on the actual prerequisite base. The design's 155/468 false-positive census is historical evidence, not a fixed total after other work lands: require zero labelled outer rows reported pure, zero declared-open outer rows reported pure, six streaming exports false, mapE false and the approved latent-tail controls true. Record any census drift and the approximately 26 combinator pins in the implementation PR.

Run `ailang eval-suite --benchmarks prompt_injection` using the configured current model set and amended active prompt/spec. Save compiler commit, prompt hash, model/configuration and result location in the PR; inspect compile failures for the old rejected spelling. Do not claim an eval pass if models still generate rejected code.

The planning PR and milestone commits use `Refs #1443` and `Refs #574`; only the implementation PR uses `Closes #1443`. #574 remains a closed regression reference. Do not open a duplicate issue or close dependency issues from this sprint. The changelog must explain the signature migration and changed iface JSON purity signal, including the known external `sunholo/mcp_oauth` spelling risk without editing that external package here.

## Approval and Handoff

The design is approved; this sprint plan is the coordinator's next review artifact. Per sprint-planner's coordinator resource, merging the coordinator plan PR approves the sprint and triggers sprint-executor. Do not self-merge or dispatch implementation before that approval and the landed dependency check. The populated JSON is ready for that handoff; all milestones remain unstarted. After execution, route the complete implementation through sprint-evaluator against the approved design and these criteria.

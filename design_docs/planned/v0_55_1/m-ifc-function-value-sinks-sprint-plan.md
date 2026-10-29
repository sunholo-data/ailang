# M-IFC-FUNCTION-VALUE-SINKS — concrete sprint draft

Status: APPROVED for execution by Mark, 2026-10-29: “Execute sprint #1719”. Prepared 2026-10-29 for P0 #1719. No implementation, repository state, coordinator state or GitHub messages changed.

Design: `design_docs/planned/m-ifc-function-value-sinks.md`. F1-F4 were explicitly RULED by Mark on 2026-10-26; quorum was skipped at the requester's instruction. These decisions are closed, not additional approval questions. Mark explicitly approved this concrete sprint for execution on 2026-10-29. A genuinely new semantic choice is a terminal STOP for the affected work, with a concrete question; existing ruled choices are implemented as written.

Intended plan location after approval: `design_docs/planned/v0_55_1/m-ifc-function-value-sinks-sprint-plan.md`; machine state: `.ailang/state/sprints/sprint_M-IFC-FUNCTION-VALUE-SINKS.json`. Version folder is provisional next-release placement after current v0.55.0, not a release request. Do not place the sprint in `ailang-core-triage/`.

## Evidence and non-duplication

Read-only base: origin/dev `fcb95e0b1fa768e725870c6cfee9eabf35fbc92c`. Current `ifc_check.go` is 632 lines. `ifcVar` has label/type only; `labelOfCall` checks A/C only for syntactically named local functions, while unknown callees only join labels. Written function-type parameter guards are not callable metadata (`walkType` intentionally visits function returns only). This directly confirms the ruled architecture remains applicable. No implementation sprint JSON or competing implementation PR/task was found in prior parent triage; refreshed open-PR inventory adds no #1719 implementation. Immediately before creating execution work, recheck exact issue/task/PR ownership, then claim one isolated branch. Stop duplicate work if an owner appeared; do not approve pending coordinator tasks.

Use the existing approved surface-AST design. No parser, HM/CoreTI, evaluator, VM, cache or interface-schema changes belong here. Add `ifc_callable.go`, `ifc_summary_local.go` and companion tests; keep `ifc_check.go` below 800 lines by putting new behavior in companions. `ifc_static_type.go` may expose a written-function-type projection, but must preserve `walkType` return-label behavior. `internal/config` is touched only if the required corpus test needs a newly registered env gate.

Registry reuse: not a package-like capability. Packages cannot enforce a compiler-owned IFC invariant; action `none` is recorded for every milestone. No network registry search is needed to substitute an application library for this compiler pass.

## Release and ownership constraints

F3 is a hard release constraint: callable propagation, HOF applied-at summaries and wrapper inference ship together. M1 alone never ships and never closes #1719. R12/R13 remain first-release acceptance criteria.

One executor owns IFC analysis and tests in an isolated worktree; docs/prompt work may be delegated only with disjoint files and one integrating owner. Avoid active #1958 row-unification and std/ai files, #1963 latent walker files, and operator-only #1790 files. Rebase on current dev immediately before coding and run combined regressions after any such PR lands.

#1771 F4 request is confirmed at https://github.com/sunholo-data/ailang/pull/1771#issuecomment-6419400823 : `IFCParam.Forbidden` must be sorted `[]string`; reserve `applied` as `IFCLabel` in its existing v2 schema. Order is #1719 first, #1771 rebased second. This sprint leaves IFC serialization untouched and hands off local summaries/value lookup for the importer to integrate. Do not merge #1771 unchanged, introduce v3, or claim the imported-HOF #1134 gap is closed.

## Estimate and checkpoints

Original ruled design estimates five days. This draft budgets six working days including a 20% integration/corpus buffer, approximately 1,400 authored lines including tests/docs (233 LOC/day is a planning quotient, not measured throughput). `analyze_velocity.sh 7` found no usable recent LOC metric and stopped with exit 141 in its head pipeline; saved output is `1719-velocity.txt`. Prompt mirrors/generated copies are excluded from the line estimate. Day 1 M1/M2 start; day 2 M2 completion; day 3 M3; day 4 M4; day 5 M5; day 6 M6 plus independent review. Milestones are resumable checkpoints, not separately releasable features. If actual work exceeds estimates, retain acceptance criteria and replan time, never silently shrink scope.

### M1: Callable demands and strict joins (~180 LOC)

**ID:** `M1_CALLABLE_DOMAIN`. **Dependencies:** None

- [x] Callable demand projection stays in the IFC abstract domain; no HM, CoreTI, parser, codegen or iface change.
- [x] Forbidden labels are sorted/deduplicated sets; callable join unions forbids and meets positive covers, with hasCover preserving absence versus bottom.
- [x] Algebra tests prove deterministic, commutative, associative, idempotent joins and non-weakening requirements; written function types and function literals seed demands without changing walkType.

### M2: Propagate callable values and check every application (~240 LOC)

**ID:** `M2_VALUE_APPLICATION`. **Dependencies:** M1_CALLABLE_DOMAIN

- [x] Lexical env resolves before local declarations; N4 shadowing remains clean. References alone never discharge demands (N6).
- [x] Carry callable values through let/letrec/block, if/match joins, record/update/access, list/array/tuple/ADT and pattern bindings; include local resultCallable and unknown-call argument-callable joins.
- [x] Checks A/C use the callee expression callable at every application while preserving existing local return-label, declared label hand-off, closure-capture and declassification behavior.
- [x] R0-R1, R3-R9, R14, C0-C1, N1 and N4-N6 focused cases pass at this checkpoint (R2 is completed with applied summaries in M3); R0/C0 kind and source position unchanged.

### M3: Applied-at summaries with actual-label substitution (~220 LOC)

**ID:** `M3_APPLIED_SUMMARIES`. **Dependencies:** M2_VALUE_APPLICATION

- [x] Silent local summary pass seeds formal label variables and accumulates per-parameter applied labels transitively; call sites substitute actual labels.
- [x] Summarized HOF calls distinguish clean and tainted actuals, including intrinsic tainted data produced inside HOF bodies.
- [x] Only unsummarized callees use the ruled conservative argument-label join fallback; R2/R10/C2 reject and N2/N3 remain accepted.

### M4: Infer wrapper demands and converge recursively (~250 LOC)

**ID:** `M4_WRAPPER_INFERENCE`. **Dependencies:** M3_APPLIED_SUMMARIES

- [x] Infer demands for declarations, lambdas and function literals, including written guarded HOF parameters; R11-R13 reject.
- [x] Fixed point over the finite written-label lattice handles mutual recursion, declaration order and cycles without silently dropping demands.
- [x] Erase summary-only label variables before effectiveBodyLabel, emitted diagnostics or any outward result; explicit tests assert no labelVar escape.
- [x] X1 inferred wrapper forbidding both email and secret rejects either tainted argument, keeps a clean argument accepted, and retains both sorted forbids.
- [x] M1-M4 are one indivisible release unit; no partial shipping or closure of #1719 before wrapper cases pass.

### M5: Pin CLI, REPL and mutation regression evidence (~310 LOC)

**ID:** `M5_REGRESSION_MATRIX`. **Dependencies:** M4_WRAPPER_INFERENCE

- [x] All 24 named design rows R0-R14/C0-C2/N1-N6 plus X1 (at least 25 cases) run as CheckModuleIFC unit and fresh-file CLI checks with exact exit/kind assertions.
- [x] Negative R rows report sink_refinement; C rows report param_label_cover. New messages name callee and first via hop; R0/C0 diagnostics retain kind and position.
- [x] REPL module-load gate agrees with CheckModuleIFC; existing IFC closure, annotated-let, declared-record-label, scoped authority, positive-label and transparent passthrough tests pass unchanged.
- [x] Scratch-only mutation: remove application checks, applied summaries and wrapper inference separately; each kills its specified R/C subset, then restore clean working tree.
- [x] Pinned fixture exit codes hold, including expected-fail leak_attempt and clean secrets/contracts/effect-latent examples; no effect is executed by rejected programs.

### M6: Corpus delta, teaching and v2 coordination (~200 LOC)

**ID:** `M6_CORPUS_DOCS`. **Dependencies:** M5_REGRESSION_MATRIX

- [ ] Old/new CheckModuleIFC corpus test compares exact same source bytes in examples/std/benchmarks and present sibling repos; any env gate is registered in internal/config.
- [ ] Attach every new diagnostic, zero new diagnostics for unannotated modules, source SHAs/counts and classified deltas; explicitly run leak_lab examples and gmail or record unavailable corpus as an unresolved gate.
- [ ] Any over-approximation requires a human ruling before merge; do not silently weaken demands or migrate consumers outside approved scope.
- [ ] Expected-fail sink-function-value example, passing controls, truthful manifest entries, IFC guide, new immutable teaching-prompt version and ### Fixed changelog are complete.
- [ ] #1771 remains subsequent work: rebase after this sprint, use sorted []string Forbidden and reserved applied IFCLabel in v2, fold imported name lookup into value lookup; no v3 or cross-module completion claim.
- [ ] make test-core, focused types/repl/pipeline checks, make lint, make check-file-sizes, make check-boundaries and make verify-examples pass; independent sprint-evaluator reviews the exact final head before parent merges.

## Exact regression corpus

The design table enumerates 24 named cases although its prose says “25 files”: R0-R14 (15), C0-C2 (3), N1-N6 (6). Preserve every named case and add X1 (inferred wrapper combining two forbidden labels, with clean control) for at least 25 cases. This is a mechanical count correction, not a change to ruled semantics. Reconstruct fresh fixtures from the design table if the original `dd2/r` scratchpad is unavailable; direct controls must validate fixture syntax before acceptance is counted.

R0 direct sink and C0 direct positive cover keep exact kind/source position. R1 alias; R2 HOF; R3 annotated function binding; R4 selected callable; R5 record; R6 list pattern; R7 tuple pattern; R8 function result; R9 composition; R10 HOF intrinsic source; R11 guarded HOF type; R12 named wrapper; R13 lambda wrapper; R14 function-literal sink. C1 alias and C2 HOF preserve positive-cover checking. N1 direct clean; N2 alias/HOF clean; N3 non-sink HOF with tainted data; N4 lexical shadow; N5 explicit declassification; N6 stored never-applied sink remain accepted. Add lattice join, multi-label, recursion/source-order, result-callable and no-labelVar-escape tests.

Run each named case through CheckModuleIFC and fresh unique CLI files in isolated cache roots; pin REPL module-load parity. Application-check mutation must invalidate R1/R3-R9/C1; applied-summary mutation R2/R10/C2; wrapper-inference mutation R11-R13. Run mutations in disposable scratch only and preserve the final clean implementation.

Pin existing tests including `TestIFCClosureCannotLaunderALabel`, annotated-let closure tests, declared-record/McpOauth idioms, scoped/positive cover tests and `TestIFC_TransparentPassthrough_NoError`. CLI fixtures: secrets `gated_secret`, `positive_label_call`, `scoped_declassify`, `secret_demo` exit 0; `leak_attempt` exit 1; contracts `inbox_v2_lib`, `inbox_v2_app`, `inbox_injection_v2` exit 0; benchmark `prompt_injection/expected_ailang_safe.ail` exit 0; `effect_latent_function_values.ail` exit 0. If #1790 independently lands, rerun the scoped contract fixtures and record the combined base without editing its files.

Corpus comparison must use identical source bytes and exact baseline/fixed compiler SHAs, invoke the IFC pass rather than infer success solely from unrelated HM errors, record skipped parse/type failures, and enumerate every new diagnostic. Include examples/std/benchmarks and present sibling repos, with explicit `ailang-demos/leak_lab/examples` and `ailang-packages/packages/gmail` runs. A missing required checkout is unresolved evidence, not a green gate. Expect zero new unannotated-module violations. Classify new annotated-module failures as actual leaks or over-approximation. The latter stops merge pending a concrete human ruling. Cross-repo code changes or messages require separate explicit authorization.

## Docs, gates and handoff

Add the expected-fail sink example plus passing sibling controls and truthful manifest metadata. Update `docs/docs/guides/ifc-labels.mdx`: remove the local unannotated-wrapper limitation, explain aliases/HOFs and conservative imported fallback, retain remaining imported limitations. Apply prompt-manager: read current prompt, fetch registry immediately before selecting the next unused version, preserve frozen versions, add only the approved sink-through-values teaching line, generate mirrors/hashes with existing tools. Never hard-code today's next prompt number because #1963 and others may publish first. Add a `### Fixed` unreleased fragment.

Required checks: focused `go test ./internal/types/ ./internal/repl/ ./internal/pipeline/`; CLI fresh-file matrix; corpus delta; `make test-core`; `make lint`; `make check-file-sizes`; `make check-boundaries`; `make verify-examples`; prompt-specific freeze/registry checks if prompt files change. Build once before CLI checks. For PR review, use exact-head required CI rather than duplicate a passing full suite. An independent sprint-evaluator (not the executor) assesses criteria, design fidelity and regression evidence, then parent performs authorized integration/merge.

Review artifacts: exact head/base, failing-first results, complete R/C/N/X matrix, mutation failures, corpus source SHAs and diagnostic classification, gates, docs/prompt hash, #1771 handoff. Mark milestone `passes` only against real evidence. No approvals, coordinator acks, task sends or GitHub comments are part of this draft.

## Recovery provenance

Start audit found historical local branch `p0/ifc-function-value-sinks`, HEAD `37a8e8f4f`, with committed plan/code/tests/examples/docs (cc720f304, 3dd7b89fd, 110093640, 96d32ff1a, 37a8e8f4f). Its dirty tracked changes and untracked ifc_demand_scan.go are parked unchanged and banked separately in /tmp. Parent explicitly authorized recovering only committed source which fits this approved design; no dirty work is imported. This branch uses the current six-milestone plan, audits old depth/summary caps and completes missing tests/corpus/prompt work. Historical partial checkpoints are not claimed independently evaluated.

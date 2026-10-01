# Sprint Plan: M-ELABORATOR-LEXICAL-SCOPE

## Summary

Restore lexical resolution across elaborator expression positions: visible local binders win over imported values, builtins, and constructors; local records win over module aliases. Preserve constructor-first pattern classification and compiler-synthesized builtin references. Emit MOD015 when module declarations shadow directly imported bindings.

**Design:** `design_docs/planned/v0_51_0/m-elaborator-lexical-scope.md`
**Sprint ID:** M-ELABORATOR-LEXICAL-SCOPE
**Target:** v0.51.0
**Duration:** 5 working days (about 30 focused hours, including 6 hours contingency)
**Estimated LOC:** 800 total: 250 implementation, 460 tests/fixtures, 90 documentation.
**Risk:** Medium overall; high consequence if module precedence or block visibility is wrong.
**Status:** Planned; design approval supplied by task-b8a241cd handoff. Sprint execution awaits coordinator approval of this plan.

## Current Status and Planning Evidence

The workspace is clean on `coordinator/task-05d340c5`; std/VERSION is v0.51.0. The approved design's verification log reproduced ten bug shapes on v0.50.0. Current source still has no lexical scope stack: identifiers consult constructors/globalEnv before falling back to core.Var; match guards and bodies have no registered binders. Block normalization walks backwards and already supports statement-form letrec, which this sprint must preserve.

The seven-day velocity script reports only one recent design-document commit (35ae3261), no measurable LOC/day, and historical changelog estimates. No measured implementation velocity can be inferred. Use a planning capacity of 160 LOC/day for this 800-LOC sprint, based on the design's 4–5 day estimate, with explicit contingency; this is an estimate, not observed throughput. Five days can be scheduled consecutively or across the design's three-week calendar window.

The design includes systemic analysis of imports, builtins, constructors, qualified access, all binder kinds, and related #327/type-checker and effect-checker paths. Those sibling fixes are separate scope. No implementation or coverage baseline was run during planning; establish and record it in M1.

## Registry Reuse Audit

Executed `ailang pkg search lexical` and `ailang pkg search compiler`; both returned no packages. These milestones modify compiler-internal AST/Core resolution and diagnostics, rather than a package-level capability. No candidates exist to inspect with pkg info/docs. Decision for M1–M5: **none**, no package dependency or contribution. Record the same populated decision for every milestone in sprint JSON.

## Technical Clarifications for Execution

1. Seed the module base frame from **local FuncSig names collected before imports plus module-let names**, preserving collisions explicitly. The design's `keys(symbols) minus keys(imports)` formula would incorrectly exclude a local function whose name also occurs in imports, defeating row 7. This mechanical correction implements the approved local-wins semantics. Audit SCC dependency classification where imports and locals share a name; regression tests must prove recursive local calls work.
2. A forward block pre-pass tracks both statement-let and statement-letrec names. Each expression sees preceding bindings; a let initializer excludes its own new name, while a letrec initializer includes its own name. Apply the correct frame to the last expression too. Restore frames on all errors and on leaving a block.
3. Reuse the existing `elaborate.Warning`/pipeline Result.Warnings mechanism for MOD015. GetWarnings currently returns only exhaustiveness warnings; explicitly integrate the new diagnostic without dropping existing warnings or relying on raw stderr from compiler internals. Include module and single-file pipeline consumers where applicable.
4. Keep ResolveAsBuiltin first and keep pattern classification unchanged. If helper classification diverges, derive binders from elaborated Core patterns or share classification rather than maintaining incompatible rules.
5. Baseline existing negative examples and package fixtures before sweeping. Compare outcomes and diagnostics before/after; do not require intentional error fixtures to pass. Every additional semantic change must be explained; unexpected changes stop completion pending review.
6. Design-freeze items are treated as approved by the supplied approved-design handoff: full lexical precedence, import-only MOD015, and REPL cross-statement scope cut. Do not broaden builtin/constructor warnings, retire #327 diagnostics, or change effect-checker/REPL persistence semantics in this sprint.

## Milestones

### M1: Capture baseline and executable regression matrix (~160 LOC)

**Duration:** Day 1 morning, 4 hours. **Dependencies:** None.
**Files:** `internal/pipeline/lexical_scope_shadowing_test.go`, `internal/pipeline/testdata/lexical_scope/`, `examples/lexical_scope/`, sprint progress notes.
**Estimate:** 0 implementation + 150 tests/fixtures + 10 documentation.

Create the ten-row matrix with asserted runtime values or rejection outcomes. Run baseline cases against current source; preserve observations without committing a permanently failing test suite (stage passing controls now, enable desired-outcome assertions with the fix). Inventory collisions across std, examples, and package testdata using registered builtin names and per-module imports/constructors. Record existing failures, #327 position-matrix status, and elaborator coverage using existing make/coverage tooling. Fetch `ailang prompt` before authoring any .ail fixture.

**Acceptance criteria:**
- [ ] All ten design matrix rows have named fixtures, expected lexical outcomes, and recorded pre-fix observations.
- [ ] Collision inventory and corpus baseline distinguish intended negative fixtures from new failures.
- [ ] #327 regression baseline and elaborator coverage are recorded; failures attributable to the environment are identified explicitly.
- [ ] Runnable reporter pair is planned at examples/lexical_scope/a.ail and d.ail; all written fixtures are syntax-checked where their pre-fix semantics permit.

**Risk:** stale installed binary. Use a source-built binary and record its commit/version for comparisons.

### M2: Resolve expressions through lexical binder scopes (~260 LOC)

**Duration:** Day 1 afternoon and Day 2, 9 hours. **Dependencies:** M1.
**Files:** `internal/elaborate/core.go`, `expressions.go`, `expr_simple.go`, `expr_control.go`, `patterns.go`, `expr_calls.go`, `expr_data.go`, new `lexical_scope_test.go`; matrix fixtures from M1.
**Estimate:** 130 implementation + 130 tests.

Add scope membership and symmetric frame lifecycle. Guard Identifier, constructor-call, and module-alias access resolution. Register Lambda/FuncLit parameters, nonrecursive let body, letrec initializer/body, match arm guard/body, and forall body. Test nested frames, arm isolation, ordinary let initializer outer resolution, lowercase aliased constructors, constructor shadowing in call position, pattern binder classification, and error cleanup. Keep compiler-generated builtin references uncapturable.

**Acceptance criteria:**
- [ ] Import/builtin/constructor capture fixtures for local binders resolve to core.Var and return the expected values or lexical type errors.
- [ ] Qualified alias access resolves to a record when locally shadowed; unshadowed imports still work.
- [ ] Constructor patterns retain current classification; guards share arm binders and sibling arms do not.
- [ ] Scope state is restored after normalization errors; ResolveAsBuiltin still bypasses user shadowing.
- [ ] Focused elaborate and pipeline tests pass for completed shapes.

**Risk:** classification drift and leaks. Use parity tests and guaranteed cleanup for each binder region.

### M3: Preserve local module precedence and deliver MOD015 (~150 LOC)

**Duration:** Day 3, 6 hours. **Dependencies:** M2.
**Files:** `internal/elaborate/file.go`, `core.go`, `warnings.go`, `internal/errors/codes.go`, existing compiler/pipeline warning collection sites as needed, module scope and diagnostic tests.
**Estimate:** 70 implementation + 80 tests.

Seed the base frame from original local declarations, audit SCC ordering/filtering under import collisions, and keep module-let values and function bodies consistent. Register MOD015 and emit one structured warning per direct import/local collision using the import bind alias and local source position. Wire it to user-visible CLI diagnostics through the existing warning channel.

**Acceptance criteria:**
- [ ] Matrix row 7 returns 102, with exactly one MOD015 identifying tick and the local source position.
- [ ] Local funcs, module lets, self recursion, and mutual recursion win over same-named imports in bodies and initializers.
- [ ] Import aliases without collisions and ordinary imports produce no MOD015; collisions use the effective alias name.
- [ ] Error-code registry tests and pipeline/CLI warning delivery tests pass; exhaustiveness warnings remain intact.

**Risk:** the merged symbols/imports maps obscure declaration provenance. Preserve original local names and assert SCC behavior explicitly.

### M4: Apply positional block scope and pin REPL boundaries (~120 LOC)

**Duration:** Day 4 morning, 4 hours. **Dependencies:** M3.
**Files:** `internal/elaborate/expr_control.go`, dedicated block scope tests, `internal/repl/` existing test suite, matrix testdata.
**Estimate:** 50 implementation + 70 tests.

Precompute per-index frames for statement let and letrec while retaining existing backward ANF threading. Cover use before declaration, later use, duplicate-name rebinding, nested blocks, own-value let versus letrec semantics, and restoration after errors. Audit all NewElaborator consumers (pipeline, REPL, runtime, debug); pin within-statement lexical resolution while leaving REPL cross-statement persistence unchanged.

**Acceptance criteria:**
- [ ] Earlier expressions and ordinary let initializers resolve outer bindings; later expressions see preceding block declarations.
- [ ] Statement letrec self-calls and subsequent references work, including shadowing imported/builtin names.
- [ ] Last-expression and nested-block fixtures pass, with no scope escape to adjacent declarations.
- [ ] REPL within-expression resolution follows lexical rules and cross-statement behavior matches the baseline.

**Risk:** off-by-one scope frames. Assert values as well as Core binding shapes.

### M5: Validate corpus and document restored semantics (~110 LOC)

**Duration:** Day 4 afternoon and Day 5, 7 hours (includes final contingency). **Dependencies:** M4.
**Files:** `examples/lexical_scope/a.ail`, `d.ail` and control fixtures; `CHANGELOG.md` or current version changelog; active teaching prompt selected via `ailang prompt` and `prompts/versions.json`; `docs/LIMITATIONS.md` if affected; sprint progress JSON.
**Estimate:** 0 implementation + 30 tests/fixtures + 80 documentation.

Enable all desired-outcome assertions; verify reporter example checks and evaluates to 4, silent capture becomes the correct rejection, and all control fixtures retain their results. Repeat the baseline corpus sweep, classify every changed outcome, and document import-only MOD015 and lexical precedence. Run sprint-evaluator after execution; provide the evaluator with #327 diagnostic retirement as a deferred decision, not an implicit code change. Link the original report if its issue ID is available, avoiding closure of sibling issues.

**Acceptance criteria:**
- [ ] All ten matrix rows produce approved outcomes; reporter example evaluates to 4 and module collision example to 102.
- [ ] std/examples/package corpus comparison has no unexplained outcome or diagnostic changes; existing intentional negatives remain expected.
- [ ] make test-core, make test, make lint, and make check-boundaries pass; focused race/error cleanup checks run where relevant.
- [ ] #327 position matrix and ADT controls (pattern_matching_adt, match_hof_lambda, option_pattern_import_free, prelude_option_result) retain expected behavior.
- [ ] New resolution branches and binder lifetimes have meaningful tests; elaborator coverage does not regress from M1 baseline.
- [ ] Changelog and active prompt describe the rule; examples are checked/run with source-built binary, and evaluation handoff includes concrete results.

**Risk:** pre-existing corpus failures hide regressions. Compare banked M1 results and report infrastructure limits separately.

## Day-by-Day Schedule

| Day | Work | Exit condition |
| --- | --- | --- |
| 1 | M1 baseline; start M2 scope helpers and simple binders | Collision inventory and focused controls recorded |
| 2 | Finish M2 match/forall, constructor calls, aliases, error cleanup | Local binder tests green |
| 3 | M3 module provenance, SCC audit, MOD015 propagation | Module collisions behave and warn correctly |
| 4 | M4 blocks/REPL; start M5 corpus checks | Positional block tests green; comparison available |
| 5 | Finish M5 documentation, full checks, contingency and evaluator handoff | All acceptance gates evidenced |

## Dependencies and Completion Gate

Milestones run sequentially M1 → M2 → M3 → M4 → M5. No external package dependency. Source-build Go/tooling and the existing test corpus are required. The planning artifact does not claim runtime verification or a fixed bug.

The user supplied approval for the previous design stage. The coordinator's plan approval/merge is the next execution gate; completion markers provide the coordinator handoff without dispatching a duplicate executor task. Sprint JSON remains not_started, with every passes/started/completed field null. No implementation is authorized by creating this plan. The executor must load sprint-executor and the evaluator must load sprint-evaluator when those stages are reached.

## Issue Linking

#327 and #323 are contextual regression references, not issues to auto-close. The originating v0.50.0 report has no explicit GitHub issue ID in the handoff; github_issues stays empty until its identity is confirmed. The executor should use Refs for progress and Fixes only for a verified originating issue when the complete fix lands.

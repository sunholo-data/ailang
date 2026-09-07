# M-CACHESRC-COGNITIVE-COMPLEXITY — extract compile-cache orchestration and preserve its regression oracles

**Status**: Planned — design frozen; quorum R1 blocked (2 objections), designer revision landed, quorum R2 blocked on one narrow objection satisfied by the ratified narrow-refinement carve-out (verbatim reviewer fix applied by the controller; V26). Routing to planner proceeds at N−1 with `gpt5-6-sol` absent (auth).

### Quorum verification log

- **R1 (2026-09-06T23:23:38Z)**: BLOCKED 2-of-3 — `gpt5-6-sol` reject (orchestration-abstraction duplication unverified), `gemini-3-1-pro` pass, `oc-glm-5-2` reject (seven reuse-target symbols unverified). `absent_reviewers` EMPTY. Metered $0.09975172. Both objections classified PREMISE objections and measured by the controller before the revision (rule 3f): the seven symbols all exist exactly as spelled; the 46-struct inventory shows no existing per-call orchestration state owner in `internal/pipeline` (and `internal/compiler` does not exist in this repo).
- **Designer revision (2026-09-07, recovery iteration 341)**: `pi:ollama/deepseek-v4-flash:0731-cloud` design-doc-creator role added V24 (inventory, corrected scope), V25 (symbol existence) and the orchestration-abstraction Conflict Surface table.
- **R2 (2026-09-07T00:49:42Z)**: BLOCKED 1-of-2 present — `gemini-3-1-pro` reject (focused baseline command omits `TestContractExpressionsFullyLowered` and `TestSplitArgWarning_Integration`), `oc-glm-5-2` pass, `gpt5-6-sol` ABSENT (auth: `OPENAI_API_KEY` not set; the codex subscription bucket is quota-exhausted until 2026-09-13). Metered $0.04571155. The objection is narrow, carries the reviewer's verbatim `proposed_fix`, and does not dispute the design direction; the controller measured its premise (both fixtures pass at baseline, rc=0).
- **Carve-out applied (ratified 2026-08-31, first used iteration 251, applied again by iteration 340)**: the reviewer's VERBATIM fix was applied — the focused command in Testing Strategy now includes both fixtures, and V26 records the re-run baseline (pipeline 0.586s / loader 0.409s, rc=0). No third quorum round. Downstream quotes must read "R2 BLOCKED at N−1 with the surviving objection satisfied by the verbatim carve-out fix; `gpt5-6-sol` absent (auth)", never "quorum passed".
**Target**: v0.35.2
**Priority**: P1 — bounded maintainability debt from PR #1053
**Estimated**: 2 working days (~12 hours including verification and review buffer)
**Dependencies**: Source-snapshot implementation in PR #1053, already present at baseline [V03]
**Planner-Lane**: codex-ok
**Created**: 2026-09-07
**Mission**: V1 iteration 341
**Verified baseline**: `d794cac257cf87763f0aafa7360ede0372272cd2` [V01]
**Author**: required design-doc-creator role
**Quorum**: REQUIRED because this is an unattended mission document. Controller owns quorum invocation/artifacts and subsequent approval/planning/execution gates. No quorum verdict is claimed here.

## Problem Statement

Five surviving `go:S3776` findings correspond to the source-snapshot work in
[PR #1053](https://github.com/sunholo-data/ailang/pull/1053). Current `dev` readings
are 29, 19, 24, 16, and 103 against a limit of 15 [V04]. The historical PR analysis
still reports 32, 19, 28, 16, and 112; it is a different analysis context, not the
baseline for this refactor [V05].

The `dev` quality gate currently reports `OK`, with a `previous_version` period
starting 2026-09-05 against v0.35.0 [V06]. CI derives Sonar projectVersion from a
release tag and declares the scan `continue-on-error: true` [V18]. Consequently,
a green gate or CI check alone does not prove this debt has been removed.

### Exact scope and issue identity

Paths below are relative to the repository. Function names, rather than shifting
line numbers, identify the acceptance surface [V04, V07–V10].

| File / function | Current dev complexity | Dev issue key | Historical PR #1053 key |
|---|---:|---|---|
| `internal/pipeline/cache_invalidation_test.go` / `TestCacheSource_ExactSnapshot` | 29 | `AaBzgnYPBD7wArG_Hhqp` | `AaBzYMQM50elM9qDpUNE` |
| `internal/pipeline/cache_invalidation_test.go` / `TestCachePipeline_EmbeddedKeys` | 19 | `AaBzgnYPBD7wArG_Hhqq` | `AaBzYMQM50elM9qDpUNF` |
| `internal/pipeline/cache_invalidation_test.go` / `TestCachePipeline_SourceEditBehavior` | 24 | `AaBzgnYPBD7wArG_Hhqr` | `AaBzYMQM50elM9qDpUNG` |
| `internal/loader/loader_test.go` / `TestCacheSource_ExactSnapshot` | 16 | `AaBzgqCSBD7wArG_Hhqt` | `AaBzYMcf50elM9qDpUNI` |
| `internal/pipeline/pipeline_module.go` / `runModuleWithCacheDependencies` | 103 | `AaBzgnbFBD7wArG_Hhqs` | `AaBzYMbv50elM9qDpUNH` |

The systemic problem within this scope is nested orchestration: the production
entry combines environment setup, loading, cache decisions, compilation, warning
collection, resolver setup, and evaluation; its regression tests nest setup,
instrumentation, and multiple independent assertions [V07–V10]. Extract those
responsibilities together, retaining the assertions that protect source identity.
This is a maintainability change, not a language or cache policy change.

The same live issue query also returns older findings for
`TestCacheArtifacts_Migration` (16), `TestCachePipeline_WriteFailure` (27), and
`detectModulePrefixOverlap` (20) [V04]. They remain regression controls, outside
the five-finding target. The API reports 756 open S3776 findings across the project;
that count is context, not a scope expansion or a claim that all pages were audited.

## Goals

- All five exact target functions have cognitive complexity ≤15 after Sonar reanalysis.
- Every new or materially edited helper introduced by this extraction is also ≤15;
  complexity must not merely move into a new oversized function.
- Preserve baseline cache behavior, results, diagnostic text/order, stage ordering,
  configuration handling, and regression-test assertions.
- Establish non-vacuous cold/edit/warm/disabled and failure-path evidence, including
  named mutations, and retain normal repository CI success.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Refactor all five surviving PR #1053 functions, with no historical-debt sweep | Defines the bounded systemic unit and review size | agent | design | med |
| Preserve each branch's existing order and effects, including error-path telemetry behavior | Reordering can change compilation skips, errors, or partial results | agent | design | high |
| Reuse `cacheRuntime` and existing compile/link/lower helpers; extract orchestration only | Keeps the existing artifact authorization boundary intact [V11] | agent | design | high |
| Use package-private typed helpers, with explicit inputs/results or one per-invocation state owner | Avoids globals, generic callback frameworks, or parallel execution | agent | design | med |
| Retain named top-level test entry points and all observable assertions | Existing regression commands must continue to exercise the same cases | agent | design | med |
| Require exact-function analysis plus positive scan controls; do not suppress issues | Gate status alone is insufficient [V06, V18] | agent | design | med |

### Design Freeze

- [x] All five targets and the ≤15 limit are fixed.
- [x] Behavior is frozen at the baseline, including existing optional-cache failures,
  warning channels, cache stats, error wrapping, and when span/timing updates occur.
- [x] Artifact formats, key version, source ownership, directory naming and trust
  policy are reused unchanged.
- [x] Extraction stays within the pipeline package and the two test packages.
- [x] Regression assertions and named mutation oracles are preserved.
- [x] The controller runs unattended quorum before downstream gates. This checked
  decision assigns gate ownership; it does not record a completed quorum.

No human product decision is needed for this refactor. Any discovered behavior fix
must be separately designed; it must not be silently bundled into this work.

## Solution Design

### Existing invariant and sequencing map

These are current code observations, not new behavior [V07, V11–V13].

| Stage | Existing behavior to preserve |
|---|---|
| Initialize | Create the Result timing map, pipeline span and deferred memory recording; fill only nil configuration environments. |
| Load | PackageDir determines loader base; resolve package search directory; configure strict syntax and package resolver/prefixes; LoadAll; invoke only `afterLoad[0]` when present/non-nil. |
| Validate/order | Record load timing, detect path collisions then prefix overlap; register builtin module and topologically sort from canonical root. |
| Compile loop | Allocate each CompileUnit with Surface; derive dependency digests only from already-compiled imports whose Iface is non-nil. |
| Cache eligibility | Disabled/uninitialized cache bypasses lookup; nil SourceContent warns and bypasses both lookup and publication; pointer-to-empty remains cacheable. |
| Hit | Call `cacheRuntime.load`; verified increments hits; only a non-nil cached payload hydrates Core/CoreTI/Iface/Constructors, registers its interface and skips compilation. |
| Miss/fresh | Preserve miss accounting and debug text; validate module path only on the fresh branch; resolve imports, elaborate, lower/typecheck, collect warnings/root debug handles, build/register interface. |
| Publish | Require live cache/store and a nonempty computed key; retain normalized interface metadata and timestamp construction; publish artifacts before manifest authorization. |
| Complete compile | Traverse module IDs in sorted order for the three warning detectors; register ADT module; save optional cache and report stats; end compile span and timing. |
| Resolve/evaluate | Register compiled modules; adapt GlobalResolver builtin lookup; locate root by canonical ID then original filename; configure evaluator and evaluate first declaration only in ModeEval. |
| Result | Preserve Artifacts, Interface, root type/debug handles, DictReg and assembled runtime modules; source snapshot is omitted from runtime result [V13]. |

A subtle compatibility requirement: the current code counts a verified lookup
before checking whether the cached payload is nil [V07]. Preserve this order, even
if that combination is not expected from today's store. Likewise, do not introduce
new deferred child-span endings or error wrapping as incidental cleanup.

### Production extraction seams

Keep `runModuleWithContext` and `runModuleWithCacheDependencies` signatures.
Reduce the latter to an ordered coordinator with a small number of error checks.
Use guard clauses within helpers and retain the current Result on each early return.
A package-private per-call state struct is permitted for shared Result, Config,
loader/linker, compiled units, cache counters, canonical root, and root debug handles.
It must contain only existing per-invocation values, not new global state.

Proposed responsibility split; helper names are illustrative and agent-resolvable:

1. Environment initialization and compilation-span resource lifecycle.
2. Loader/package-resolver setup and loading, with the post-load callback at the
   identical boundary; collision checks and topological setup remain ordered.
3. Dependency digest collection and cache eligibility/key preparation.
4. Verified lookup accounting/debug reporting, then cache payload hydration and
   interface registration. Keep the lookup result distinct from “skip compilation”.
5. Fresh single-module compilation, calling the existing
   `resolveModuleImports`, `typeCheckAndLowerModule`,
   `buildAndRegisterInterface` and constructor helpers in their current order [V07, V25].
6. Fresh artifact publication and compile-cache summary reporting.
7. Sorted post-compilation warnings and resolver registration.
8. Root lookup/evaluator configuration/evaluation and Result assembly, reusing
   `assembleModuleResult` [V25].

These are responsibility seams, not eight mandatory large functions. Further split
a seam if it exceeds 15. In particular, moving the complete current loop into one
helper is insufficient. Prefer narrow functions taking concrete existing types;
do not introduce interfaces just to mock this refactor.

Leave `detectModulePathCollisions`, `detectModulePrefixOverlap`,
`validateModulePath`, loader parsing, and the implementation of the lowering
passes unchanged. The change relocates their callers, not their rules.

### Test extraction seams

Preserve all four target test names and existing subtest names where practical.
Retain `t.Helper()` in extracted setup/assertion helpers and use sequential subtests:
these fixtures change process working directory and environment [V08–V10].

- Pipeline exact snapshot: extract deleted-after-load scenario and the nil/known-empty
  contrast into named scenario functions. Within the latter, separate cache IO
  instrumentation, nil-bypass assertions, and known-empty positive control. Keep the
  non-publication assertion on absence of CACHE_WRITE_FAILED and unchanged seeded
  timestamp/key; zero artifact writes alone is not sufficient [V08].
- Embedded keys: keep the real embedded source/path/nonempty/import checks, then
  extract per-module key verification. Retain both std/option and std/result and
  the runtime snapshot omission assertion [V09].
- Source edit: separate cached-edit/warm and NoCache scenario functions; extract
  instrumented store setup and output assertions. Retain dependency Core presence,
  positive dependency core-read count, zero warm encodes, and exact 3 → 41 → 41
  results [V09].
- Loader exact snapshot: extract disk and embedded scenario functions or a shared
  exact-content assertion; retain the forced missing stdlib path, embedded AST path,
  and direct equality with std.FS bytes [V10].

Reuse existing `writeCachePipelineSource`, `writeCacheBehaviorSources`,
`executePipelineMain`, manifest/stamp readers, and artifact IO hooks [V08, V09,
V12, V25]. Helpers must report failures, not convert errors to an empty fixture or skip.

### Files to Modify/Create

- `internal/pipeline/pipeline_module.go` — replace nested orchestration with phase
  calls; estimated +60/-420 LOC, leaving diagnostic rule helpers unchanged.
- `internal/pipeline/pipeline_module_phases.go` — new package-private phase helpers
  and optional per-call state; ~250 LOC, split further within these proposed files
  if necessary to keep responsibilities and file size manageable.
- `internal/pipeline/pipeline_module_cache.go` — new cache key/lookup/hydration/
  publication and reporting helpers; ~180 LOC.
- `internal/pipeline/cache_invalidation_test.go` — extract the three targeted test
  bodies while preserving assertions; estimated +100/-140 LOC.
- `internal/loader/loader_test.go` — extract disk/embedded snapshot cases;
  estimated +20/-15 LOC.
- `internal/pipeline/pipeline_module_phases_test.go` — focused characterization
  tests for moved ordering/result/error boundaries only where baseline tests do not
  directly establish them; ~150–250 LOC.
- `design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md` — record final
  implementation evidence and deviations after the downstream gates.

LOC estimates describe code movement, not a commitment to adding abstractions.
Agent may adjust private helper names and placement within this file set.
No cache-runtime/store/artifact schema implementation edits are planned.

## Conflict Surface

This changes a compilation entry point's control flow, so regression analysis is
included even though parser, type, effect and lowering implementations are out of
scope. Syntactic positions are unchanged by design; disambiguation continues through
the existing loader/parser and existing compiler-pass calls.

| Shared mechanism | Decision | Regression requirement |
|---|---|---|
| Loader package/strict-syntax setup and callback boundary | Reuse in the same order | Preserve load errors and post-load source mutation behavior. |
| MOD011/MOD013 checks and MOD010 fresh-path validation | Reuse, with identical placement relative to cache hits | Same-file aliases remain allowed; different-file collisions and overlapping root/dependency prefixes retain diagnostics [V15]. |
| Cache authorization and optional persistence | Reuse | Nil bypass, valid warm hit, corrupted/legacy artifact miss and nonfatal write failure controls [V08–V12]. |
| Existing typecheck/lower pass | Reuse | Existing contract lowering regression remains green [V14]. |
| Post-compile warnings on cached and fresh units | Reuse sorted traversal | Cold/warm warning type/message/order parity; retain positive and negative split-warning controls [V14]. |
| Runtime result assembly and ModeEval branch | Reuse | Exact program output, artifact/interface presence and source omission; characterize both modes before extraction. |

Programs/fixtures that must continue working are the disk/embedded cases in both
`TestCacheSource_ExactSnapshot` tests, `TestCachePipeline_SourceEditBehavior`,
the contract fixture in `TestContractExpressionsFullyLowered`, and positive/
negative fixtures in `TestSplitArgWarning_Integration` [V08–V10, V14].
These are existing test-body fixtures; this document introduces no new AILANG
syntax examples or language-support claims.

### Orchestration-abstraction conflict surface

A new package-private per-call state struct is justified because no existing
candidate in `internal/pipeline` has compatible ownership, scope, or sequencing
to own per-invocation orchestration state spanning load→validate→compile-loop→
cache→evaluate [V24]. The inventory below is the measured candidate set; each
row records whether the design reuses, extends, or rejects the existing
abstraction and why.

| Existing abstraction | Verified location/API | Overlap | Decision |
|---|---|---|---|
| `cacheRuntime` | cache_runtime.go:20; methods load/publish/save/warn* | Per-invocation cache state (store factory, stderr, counters) | Reuse unchanged — it is the cache-state owner, not a run coordinator; the design already commits to it in High-Impact Decisions. |
| `CompileUnit` + `ConstructorInfo` | compile_unit.go:11,22 | Per-module compile state | Unfit as run coordinator — per-module scope, not per-run orchestration. |
| `moduleImports` + `importedCtorInfo` | pipeline_module_imports.go:16,23 | Per-module import resolution results | Unfit — per-module scope; carries no Result/Config/loader/linker. |
| `validator` | validate_coretypeinfo.go:63 | CoreTypeInfo validation state only | Unfit — validation-only scope, no orchestration state. |
| `cacheArtifactIO` family (cacheArtifactIO/cacheArtifactCodec/encodedArtifact/artifactStamp/artifactLimits/cacheArtifactError) | cache_artifacts.go:35–172 | Artifact encoding layer | Unfit as coordinator — no compile orchestration; reused as-is for artifact IO. |
| `runPostTypeCheckPhases` | pipeline_module_compile.go:314 | Only phase-shaped helper; runs post-typecheck phases for ONE CompileUnit | Unfit as coordinator — not per-run, carries no Result/loader/linker state; the new coordinator calls it rather than re-wrapping it. |
| `MetricsCollector.RecordPhase` | metrics.go:90 | Timing only | Unfit — timing-only, no orchestration state. |
| `Config`/`Source`/`Artifacts`/`Result` | pipeline.go:56,102,110,119 | Public data carriers | Unfit as state owners — data carriers, not orchestration state. |

### Pipeline Pass Coverage

- [ ] Retain the existing pass calls for top-level declarations.
- [ ] Retain the same pass calls for contract expressions through the existing helper.
- [ ] `TestContractExpressionsFullyLowered` passes unchanged.

## Testing Strategy

### Baseline and regression execution

The focused command below passed at the verified baseline at both readings: pipeline
0.667s / loader 0.703s [V16], re-run after the R2 quorum fix widened the regex to include
the two Conflict Surface fixtures: pipeline 0.586s / loader 0.409s [V26]. This is design
evidence, not a full-suite or mutation result.

```bash
go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1
```

Executor must run the same tests after extraction, the complete loader/pipeline
packages, then normal `make test`, `make lint`, and relevant repository CI checks.
Use `make ci` for the repository's aggregate gate set [V17], rather than substituting
a hand-picked subset for release/landing evidence. Run focused packages under
`-race` after all changes. Report failures/skips with exact provenance; do not
claim an unrelated failing baseline was fixed by this refactor.

Before moving production code, add only missing characterization assertions in
the proposed phases test file: ModeCheck does not evaluate; ModeEval returns the
same first-declaration value; root lookup fallback/error text; nil/provided
configuration behavior; post-load callback position; warning order across fresh
and verified warm modules; debug CACHE hit/miss/invalid/summary forms and counters.
Capture these on the baseline first, and keep expected semantic values stable
through the refactor. For nondeterministic durations/memory/timestamps, compare
presence, format, and update boundaries rather than raw numbers. Preserve root
TypeChecker/DebugSink nilness across cold versus warm execution.

Include a mixed case with a verified cached dependency and a deliberately fresh
importer. Prove the dependency was read from artifacts and the importer compiled,
then execute its imported call with a fixed expected result. An entirely warm
graph can skip import resolution and is not a sufficient oracle for cached
interface registration; the fresh resolver reads interfaces from the linker [V23].

Do not mistake direct MOD helper tests for evidence that the pipeline still calls
those helpers: add an end-to-end pipeline case for any moved call whose placement
is not exercised by the selected suite.

### Named mutations and non-vacuity

Run each mutation individually in an isolated disposable execution worktree, restore
only the owned mutation, and bank the failing assertion plus restored-green result.
A compile error or unrelated environment failure is not a kill. These mutations
are proposed acceptance probes; none was executed during design authoring.

| Mutation | Required failing oracle / positive control |
|---|---|
| MUT-SOURCE-EMPTY: hash empty text for every available snapshot | Deleted-after-load key comparison and embedded expected/nonempty-key comparison fail; restored controls publish real keys. |
| MUT-SOURCE-REREAD: replace retained text with a filesystem reread | Deleted-after-load case fails at source identity or compilation; deletion is verified, and ordinary source still compiles. |
| MUT-NIL-PUBLISH: remove nonempty-key publication guard | Nil-source case detects CACHE_WRITE_FAILED/publication attempt; seeded key/timestamp and zero IO assertions remain. |
| MUT-EMPTY-BYPASS: treat pointer-to-empty as unavailable | Known-empty case fails its real manifest-entry/key assertion; nil case still demonstrates the opposite arm. |
| MUT-WARM-OFF: force verified cached payload to take fresh compilation | Positive dependency artifact reads and zero warm encodes distinguish real reuse from merely equal output. |
| MUT-CACHED-CORE-OMIT: omit copying cached Core during hydration | Warm execution or required Core assertion fails after positive artifact reads. |
| MUT-IFACE-REGISTER-OMIT: skip cached interface registration | Mixed cached-dependency/fresh-importer case fails resolving or executing the imported call; positive artifact reads and fresh-importer compilation prove both branches were reached. |
| MUT-NOCACHE-PERSIST: initialize/use cache despite NoCache | NoCache compile-directory absence fails; cached control produces a manifest. |
| MUT-SNAPSHOT-LEAK: copy source into assembled runtime LoadedModule | Runtime snapshot-omission assertion fails on an actually executed module. |
| MUT-WARNING-SKIP: omit post-compile warning collection | Positive cold/warm warning fixture fails; correct-order negative control still has zero target warnings. |

The source-reread and snapshot-leak probes may temporarily touch their original
loader/result-assembly locations solely inside the disposable mutation worktree;
they are not implementation scope expansion. If a mutation survives, strengthen
the corresponding observable assertion before declaring preservation complete.

### Sonar and PR success evidence

Use first-party issue APIs scoped by project + analysis branch/PR + rule, and pin
the analysis commit SHA. Record all five old→new function mappings if code moves.
The new PR analysis must successfully scan both production and test files; config
currently separates Go tests through `sonar.tests` [V18].

For each target, record measured complexity ≤15, or the absence of a S3776 finding
for that exact function in a successful analysis with threshold 15 and confirmed
file inclusion. Confirm every newly extracted helper similarly. Fetch all pages
or query the exact components; the initial global page used for this design is
not adequate proof of absence after implementation.

Positive controls: baseline API already returns these five issues [V04/V05];
post-analysis component/source data must show the changed files at the expected
revision. The three older same-file findings may serve as same-scope rule controls
if still present. A zero-length PR issue list without a successful, correctly
scoped analysis is insufficient. Historical PR #1053's issue keys can remain in
its frozen snapshot: closure is measured on the new PR and the corresponding dev
functions after landing, not by editing old issue dispositions.

No rule suppression, issue marking, threshold change, coverage exclusion, or
new-code-period change is authorized. If Sonar is unavailable, local tests can
complete, but the exact-function analysis criterion remains pending and must be
reported to the controller.

## Implementation Plan

### Phase 1 — Freeze executable behavior (~3 hours)

- [ ] Reconfirm source baseline and five issue identities.
- [ ] Preserve existing target assertions and add only necessary characterization
  of the moved entry-point boundaries; run them against pre-extraction code.
- [ ] Extract test setup/assertion/scenario helpers without weakening or renaming
  the four top-level test entry points.

### Phase 2 — Extract production responsibilities (~5 hours)

- [ ] Extract cache preparation, verified hydration, publication and reporting.
- [ ] Extract load/compile/finalization seams so coordinator and every changed helper
  can satisfy ≤15; keep stage/error boundaries and concrete dependencies.
- [ ] Re-run affected package tests after each coherent extraction.

### Phase 3 — Prove preservation and analyze (~4 hours)

- [ ] Run named mutation probes with failing-arm/restored-green evidence.
- [ ] Run package/race, repository tests/lint/CI and exact-revision Sonar analysis.
- [ ] Record function complexity results, test preservation, changed files and
  any deviations in implementation evidence; hand off for independent evaluation.

## Success Criteria

- [ ] All five target functions satisfy ≤15 and have exact-revision analysis evidence.
- [ ] Newly introduced/changed helpers satisfy the same threshold.
- [ ] All tests passing; target assertions, names, source identity, warm reuse and
  NoCache/failure behavior preserved.
- [ ] Named mutation probes fail for their intended assertions and restored code passes.
- [ ] Normal repository CI passes, with Sonar analysis completion verified separately.
- [ ] Documentation updated with implementation, tests and analysis results.
- [ ] Diff stays within frozen semantics and bounded responsibility scope.

## Deferred Decisions

- Agent may choose helper names and private state-versus-parameter packaging.
- Agent may choose subtest/scenario helper placement within the listed test files.
- Agent may choose the smallest additional characterization fixtures needed to
  cover moved branches; baseline behavior is the oracle.
- Agent may choose how to record mutation diffs/results in controller-owned evidence.

## Non-Goals

- Parser, type inference, effects, code generation or compiler-pass semantics.
- Cache key/version/schema, artifact verification limits, directory encoding,
  source lifetime policy, new cache APIs, or threat-model changes.
- Resolving the cache naming design's parked D-57 decision or reopening D-55 [V19].
- Historical S3776 cleanup beyond the five target findings.
- Changing failure-path telemetry behavior, warning policy, evaluation scheduling,
  concurrency, or configuration defaults.
- Editing Sonar settings, suppression status, mission bookkeeping or PR metadata
  during design authoring.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Refactor changes cache skip/validation ordering | High | Freeze the sequence map; warm-hit instrumentation and imported-dependency mutation. |
| Assertions disappear inside “cleanup” helpers | High | Preserve named cases and each observable assertion; named mutation kills required. |
| Narrowed helper loses partial Result/debug state on error | High | Characterize root debug handles, timings and error/result boundaries before extraction. |
| Complexity moves to helpers or test closures | Medium | Analyze every introduced/modified function, not just old issue keys. |
| Cache disabled/corrupt/unwritable paths regress | High | Existing NoCache, legacy migration and write/init/save failure controls. |
| Green CI masks missing Sonar scan | Medium | Exact-revision successful analysis and source/component positive controls. |

## Timeline

Two working days: ~3 hours executable baseline/test extraction, ~5 hours production
extraction, ~4 hours mutation/regression/analysis and review buffer. External Sonar
queue time is additional elapsed time, not proof of completion.

## Axiom Compliance

Canonical reference: [Design Axioms](/docs/references/axioms).

| Axiom | Score | Justification |
|---|---:|---|
| A1 Determinism | 0 | Preserve dependency/warning order and existing outputs. |
| A2 Replayability | 0 | Preserve source/cache identity and trace semantics. |
| A3 Effect Legibility | 0 | Existing operations remain at explicit helper boundaries. |
| A4 Explicit Authority | 0 | Cache authorization/capability behavior is unchanged. |
| A5 Bounded Verification | +1 | Named small responsibilities and discriminating mutation oracles permit local checks. |
| A6 Safe Concurrency | 0 | Sequential execution and per-call ownership remain fixed. |
| A7 Machines First | +1 | Enforced ≤15 functions make control flow cheaper to inspect and regressions machine-checkable. |
| A8 Minimal Syntax | 0 | The design adds no language syntax. |
| A9 Cost Visibility | 0 | Preserve timing/memory/cache-stat reporting behavior. |
| A10 Composability | +1 | Concrete compile/cache responsibilities compose through explicit existing data. |
| A11 Structured Failure | 0 | Preserve structured errors and cache diagnostic text. |
| A12 System Boundary | 0 | Keep the same pipeline/cache/filesystem boundaries. |

**Net score: +3.** Eligible for design review; not implementation authorization.

- [x] A1: no new implicit nondeterminism.
- [x] A3: no hidden side effects.
- [x] A4: no ambient authority added.
- [x] A7: machine analysis and automated regression strength improve.

## Verification Log

Commands ran in the isolated worktree at V01 on 2026-09-07. Source-read rows
report observations, not runtime proof; planned tests are explicitly future work.

| ID | Claim / command | Observed output |
|---|---|---|
| V01 | `git status --short; git rev-parse HEAD; cat std/VERSION` before edits | Clean worktree; d794cac257cf87763f0aafa7360ede0372272cd2; v0.35.1. |
| V02 | `rg --files design_docs` filtered for cache/complexity; `rg -n 'cachesrc\|cognitive complexity\|S3776' design_docs/planned design_docs/implemented` (Markdown-escaped pipes denote regex alternatives) | No exact-topic/body match before scaffold. Same-tree positive filename controls: implemented v0_35_2 compile-cache artifacts doc, planned v0_36_0 cache encoding doc, implemented v0_14_1 Sonar cleanup doc. |
| V03 | `git log -6 --oneline -- internal/pipeline/pipeline_module.go internal/pipeline/cache_invalidation_test.go internal/loader/loader_test.go`; `git show --stat f5edd569a` | f5edd569a is PR #1053 source snapshots; 3d7bbfad8 is earlier artifact verification; source snapshot commit touches five files and its message records the assertion preventing nil publication attempts. |
| V04 | `curl -fsS 'https://sonarcloud.io/api/issues/search?componentKeys=sunholo-data_ailang&branch=dev&rules=go%3AS3776&resolved=false&ps=500'` then select the three exact component paths with jq | total=756; five table keys/functions at lines 22/145/195/160/31 with complexities 29/19/24/16/103. Same-file older controls: AaByy4XEaeHH-gkmfIVl (16), AaByy4XEaeHH-gkmfIVm (27), AZ399Qt8U7Xt_lpSCPzf (20). Positive findings only; not exhaustive repository absence evidence. |
| V05 | `curl -fsS 'https://sonarcloud.io/api/issues/search?componentKeys=sunholo-data_ailang&pullRequest=1053&rules=go%3AS3776&resolved=false&ps=100'` | total=5; historical keys in target table, 32/19/28/16/112. |
| V06 | `curl -fsS 'https://sonarcloud.io/api/qualitygates/project_status?projectKey=sunholo-data_ailang&branch=dev'` | status OK; all six conditions OK; previous_version v0.35.0 dated 2026-09-05T12:07:51+0000. |
| V07 | `sed -n '1,260p' internal/pipeline/pipeline_module.go`; `sed -n '260,780p' internal/pipeline/pipeline_module.go` | Read full target body: environment defaults, loading/callback, validation/topo, nested cache loop, compile helpers, warnings, cache summary, resolver, mode-gated evaluation and assembly. Verified nil/key guards, hit counting before payload check, first callback only and current early-return boundaries. |
| V08 | `sed -n '1,145p' internal/pipeline/cache_invalidation_test.go` | Deleted source after load; retained key; seeded nil-source case with artifact read/write counters, warning/non-publication checks, unchanged key/timestamp; known-empty manifest-key positive control. |
| V09 | `sed -n '145,284p' internal/pipeline/cache_invalidation_test.go` | std/option + std/result path/content/import/key/runtime omission assertions; 3→41→41 execution, dep Core, positive warm artifact reads, zero warm encodes; NoCache 3→41 and absent compile directory. |
| V10 | `sed -n '1,215p' internal/loader/loader_test.go` | Exact-snapshot function starts at 160; disk literal equality and forced embedded std/option equality with std.FS bytes, nonnil AST and synthetic path checks. |
| V11 | `cat internal/pipeline/cache_runtime.go` | Existing injected newStore/stderr; load verifies artifacts; publish stores artifacts before manifest entry; save/init/write warnings are nonfatal; warning formatting helpers present. |
| V12 | `sed -n '284,530p' internal/pipeline/cache_invalidation_test.go` | Migration v3→v4 and missing-stamp tests; artifact/init/manifest failure cases; restored positive warm decode; reusable source, runtime execution, manifest/stamp helpers. |
| V13 | `sed -n '280,345p' internal/loader/loader.go`; `sed -n '595,660p' internal/pipeline/pipeline_module_compile.go` | Loader uses sourceText for lexer.New and SourceContent. Runtime assembly omits SourceContent; positive same-struct controls explicitly copy File/Core/Iface/CoreTI/Imports. |
| V14 | `sed -n '1,110p' internal/pipeline/contract_pipeline_test.go`; `sed -n '140,230p' internal/pipeline/warn_split_args_test.go` | Existing contract fixture checks for remaining Intrinsic/BinOp nodes; split integration asserts one reversed warning versus zero correct-order warnings, using NoCache. Warm warning parity is therefore specified as additional characterization, not claimed existing coverage. |
| V15 | `sed -n '54,127p' internal/pipeline/module_collision_test.go`; `sed -n '187,225p' internal/pipeline/module_collision_test.go` | Same physical file/two canonical IDs accepted; two files reject with MOD011/path checks; root/dependency shared prefix rejects with MOD013/name checks. These directly call helpers. |
| V16 | Focused `go test` command in Testing Strategy with `-count=1` | PASS both packages; pipeline 0.667s, loader 0.703s. |
| V17 | `sed -n '1,45p' make/test.mk`; `rg -n '^test:\|^lint:\|^check-boundaries:\|^check-home-isolation:\|^ci:' Makefile make/*.mk` | test builds binary and includes pi suite; lint/boundary/home gates found; make/ci.mk aggregate includes test/lint and repository verification targets. |
| V18 | `sed -n '330,370p' .github/workflows/ci.yml`; `sed -n '1,85p' sonar-project.properties` | Scan derives tag version and continues on error; project key sunholo-data_ailang; Go source and test scopes plus coverage config visible. |
| V19 | `cat design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md`; `sed -n '1,115p' design_docs/planned/v0_36_0/m-cache-module-id-encoding.md`; `sed -n '1,100p' design_docs/implemented/v0_14_1/m-sonar-gate-cleanup.md`; scoped `rg -n 'cognitive\|complexity\|S3776'` on these three | Scoped complexity search empty; same-scope positive `compile\|cache\|Sonar` search returns all three. Artifacts document describes source identity/integrity and D-55; encoding is parked on D-57; old Sonar cleanup concerns other rule families/coverage. |
| V20 | `ailang docs search --stream planned --neural --timeout 15s --limit 5 'cachesrc cognitive complexity'` and corresponding implemented search | Partial timeout: 2/187 and 5/200 candidates embedded; maximum neural scores 0.27 and 0.30. Incomplete search explicitly supplemented by V02/V19, not treated as exhaustive negative evidence. |
| V21 | `.claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-cachesrc-cognitive-complexity v0_35_2` | Scaffold created intended file; current v0.35.1, suggested v0_35_2. Neural top scores 0.30 implemented and 0.29 planned; high SimHash scores are not neural duplicate thresholds. |
| V22 | `for cachesrc_related in design_docs/implemented/v0_0_3/gpt5-reference-code.md design_docs/implemented/v0_5_10/m-codegen-nested-record-type.md design_docs/implemented/v0_26_0/m-eval-output-normalization.md design_docs/implemented/v0_30_0/m-mission-agentic-provider-routing-sprint-plan.md design_docs/implemented/v0_30_0/m-effect-replay-contracts-sprint-plan.md design_docs/implemented/v0_10_0/m-pkg-lock-portability.md design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md design_docs/planned/m-array-show-diverges-run-vs-compile.md design_docs/planned/m-contract-verification-coverage.md design_docs/planned/v0_35_0/m-eq-derive-containers.md; do sed -n '1,8p' "$cachesrc_related"; done` plus the earlier problem-section reads | Subjects are type reference/codegen, eval normalization, role routing, effect replay, package paths, discovery timing, array output, verification KPI, and Eq synthesis. Distinct from compile-cache complexity extraction; no neural score reaches a duplicate threshold. |
| V23 | `sed -n '88,113p' internal/pipeline/pipeline_module_imports.go` | Fresh import resolution uses modLinker.GetIface(imp.Path); absent interface skips that import's type registration. This grounds the proposed mixed cached/fresh registration oracle. |
| V24 | `rg '^type \w+ struct' internal/pipeline -g '!*_test.go'` (orchestration-abstraction inventory; scope corrected to internal/pipeline only — `internal/compiler` does NOT exist, rg rc=2 "No such file or directory") | 46 production struct declarations. Orchestration-relevant candidates and verified locations: `cacheRuntime` (cache_runtime.go:20 — per-invocation cache state: store factory, stderr, counters; methods load/publish/save/warn*); `CompileUnit`+`ConstructorInfo` (compile_unit.go:11,22 — per-MODULE compile state, not per-run orchestration); `moduleImports`+`importedCtorInfo` (pipeline_module_imports.go:16,23 — per-module import resolution results); `validator` (validate_coretypeinfo.go:63 — CoreTypeInfo validation state only); `cacheArtifactIO`/`cacheArtifactCodec`/`encodedArtifact`/`artifactStamp`/`artifactLimits`/`cacheArtifactError` (cache_artifacts.go:35–172 — artifact encoding layer, no compile orchestration); `VarResolver` (resolve_vars.go:24), `OpLowerer`/`FallbackEvent` (op_lowering.go:13,21), Specializer family (specialize.go:34–99), `MetricsCollector` (metrics.go:90 RecordPhase — timing only); `runPostTypeCheckPhases` (pipeline_module_compile.go:314 — the ONLY phase-shaped helper: runs post-typecheck phases for ONE CompileUnit, 2 call sites, not a per-run coordinator, carries no Result/loader/linker state); `Config`/`Source`/`Artifacts`/`Result` (pipeline.go:56,102,110,119 — public data carriers, not orchestration state owners). Conclusion: no existing type in internal/pipeline owns per-invocation orchestration state spanning load→validate→compile-loop→cache→evaluate (Result, Config, loader/linker, compiled units, cache counters, canonical root, root debug handles together); the proposed package-private per-call state struct does not duplicate any existing abstraction; the design reuses `cacheRuntime` and calls the existing per-unit helpers instead of re-wrapping them. Count note: 46 excludes the tracked `internal/pipeline/pipeline.go.backup` (4 duplicate data-carrier structs); the literal `-g '!*_test.go'` filter also matches that backup file. |
| V25 | `rg -n 'func (resolveModuleImports|typeCheckAndLowerModule|buildAndRegisterInterface|assembleModuleResult)' internal/pipeline/` and `rg -n 'func (writeCachePipelineSource|writeCacheBehaviorSources|executePipelineMain)' internal/pipeline/cache_invalidation_test.go internal/loader/loader_test.go` | Both commands rc=0. All seven named helpers exist with exact spelling: `resolveModuleImports` (pipeline_module_imports.go:41), `typeCheckAndLowerModule` (pipeline_module_compile.go:53), `buildAndRegisterInterface` (pipeline_module_compile.go:502), `assembleModuleResult` (pipeline_module_compile.go:595), `writeCachePipelineSource` (cache_invalidation_test.go:450), `writeCacheBehaviorSources` (cache_invalidation_test.go:458), `executePipelineMain` (cache_invalidation_test.go:470). Citation correction: all three test helpers live in internal/pipeline/cache_invalidation_test.go; they do NOT appear in internal/loader/loader_test.go (that file has its own separate helpers; rg there returns rc=1). |
| V26 | `go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1` (re-run at baseline d794cac257cf87763f0aafa7360ede0372272cd2 on 2026-09-07 after the R2 fix widened the regex; `go test ./internal/pipeline -run 'TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1` also passed alone, rc=0, 0.485s) | rc=0. ok internal/pipeline 0.586s; ok internal/loader 0.409s. The two Conflict Surface fixtures named by the R2 objection pass at baseline and are now inside the focused command. |

## Related Documents

- [Compile-cache artifact integrity](../../implemented/v0_35_2/m-compile-cache-unverified-artifacts.md)
  supplies the source-snapshot and cache correctness contract. This document
  preserves its landed behavior and addresses the five maintenance findings,
  without implementing its remaining unrelated work [V03, V19].
- [Cache module-ID encoding](../v0_36_0/m-cache-module-id-encoding.md) addresses cache
  directory naming and remains parked on D-57. This refactor does not change that
  function, naming scheme, or decision [V19].
- [Earlier Sonar gate cleanup](../../implemented/v0_14_1/m-sonar-gate-cleanup.md)
  discusses other rule families, coverage and gate configuration. Its location
  under implemented is not evidence that its stale status header is current;
  it is historical context, not this work's gate-state source [V19].
- Neural/SimHash matches were reviewed as described in V20–V22. The incomplete
  neural scan does not establish broad absence; repository keyword/path searches
  and the three relevant documents establish the bounded distinction.

## Future Work

Any additional historical complexity cleanup or cache policy change requires its
own scoped evidence and routing. This refactor intentionally leaves those choices
to their existing workstreams.

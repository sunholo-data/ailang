# Sprint Plan: M-CACHESRC-COGNITIVE-COMPLEXITY

**Design:** [design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md](m-cachesrc-cognitive-complexity.md) (frozen; committed at HEAD in this worktree)
**Sprint ID:** `M-CACHESRC-COGNITIVE-COMPLEXITY`
**Target release:** v0.35.2
**Planned at:** b682888c5fec82a38d6d50f9e955d725f1710412 (branch `docs/v1-iter341-cachesrc`; verified source baseline `d794cac257cf87763f0aafa7360ede0372272cd2`, std/VERSION v0.35.1)
**Planned:** 2026-09-07 (V1 mission iteration 341, planner role)
**Milestones:** 3 — one green, independently verifiable commit per milestone (built by the controller)
**Estimated duration:** 2 working days (~12 h effort including verification and review buffer; external Sonar queue time is additional elapsed time, not proof of completion)
**Risk level:** medium — behavior-preserving extraction only, but it rewrites the control flow of a complexity-103 compile-path coordinator; preservation is proven by named mutations, not by inspection
**JSON state:** `.ailang/state/sprints/sprint_M-CACHESRC-COGNITIVE-COMPLEXITY.json`

## Goal and motivation

Five surviving `go:S3776` cognitive-complexity findings from PR #1053 (29, 19, 24, 16, and 103 against a limit of 15, on `dev`) sit on the compile-cache source-snapshot work that iteration 330 landed: three nested test orchestrations in `internal/pipeline/cache_invalidation_test.go`, one in `internal/loader/loader_test.go`, and the production entry `runModuleWithCacheDependencies` in `internal/pipeline/pipeline_module.go` (complexity 103). The green quality gate does not prove anything about this debt — CI declares the Sonar scan `continue-on-error: true` and derives the project version from a release tag, so this sprint performs the frozen design's **behavior-preserving extraction**: it relocates the nested orchestration into package-private phase/cache helpers and named scenario functions, preserves every observable assertion and ordering, and then *proves* preservation with characterization tests, ten named mutation probes with required failing oracles, package/race/CI gates, and an exact-revision Sonar re-analysis confirming all five functions (and every newly introduced helper) at ≤15.

## Scope boundary

**In scope (from the design's Implementation Plan and Files to Modify/Create):**

- Phase 1 — freeze executable behavior: reconfirm the baseline and the five issue identities, add only the missing characterization assertions for moved entry-point boundaries (against pre-extraction code), extract test setup/assertion/scenario helpers in the two test packages without renaming or weakening the four named top-level entry points.
- Phase 2 — extract production responsibilities: reduce `runModuleWithCacheDependencies` to an ordered coordinator; move cache key/lookup/hydration/publication/reporting into the new cache helpers file; keep signatures, stage order, error boundaries, and diagnostic text/order byte-identical.
- Phase 3 — prove preservation and analyze: the ten named mutation probes (each isolated, each restored green), package tests under `-race`, the repository aggregate gates, and the complexity evidence chain (local report by the executor; exact-revision Sonar re-analysis by the controller).

**Explicitly out of scope (the design's Non-Goals; do not pull in):**

- No cache policy changes: no key/version/schema, artifact verification limits, directory encoding, source lifetime policy, new cache APIs, or threat-model changes; D-57 stays parked and D-55 stays closed in its existing workstream.
- No parser, type-inference, effects, code-generation, or compiler-pass semantics changes — the extraction relocates callers, not rules (`detectModulePathCollisions`, `detectModulePrefixOverlap`, `validateModulePath`, loader parsing, and the lowering passes stay untouched).
- No Sonar setting changes: no rule suppression, issue marking, threshold change, coverage exclusion, or new-code-period change.
- No historical S3776 cleanup beyond the five target findings (`TestCacheArtifacts_Migration` 16, `TestCachePipeline_WriteFailure` 27, `detectModulePrefixOverlap` 20 remain controls, outside the target set).
- Any discovered behavior **fix** must be separately designed; it must not be silently bundled. If extraction reveals that correctness requires excluded scope, stop at the current green boundary and return to design review.

## Verified baseline (controller-measured, HEAD d794cac257cf87763f0aafa7360ede0372272cd2, 2026-09-07, darwin/arm64)

- Focused baseline command (the design's widened form after the R2 carve-out fix) — **rc=0**, pipeline 0.586 s, loader 0.409 s:

```bash
go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1
```

- Both Sonar fixtures pass alone — **rc=0**, 0.485 s:

```bash
go test ./internal/pipeline -run 'TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1
```

- The five Sonar target findings were re-verified live; the issue keys/lines/complexities in the design doc's problem table are current (`AaBzgnYPBD7wArG_Hhqp`/29, `…Hhqq`/19, `…Hhqr`/24, `AaBzgqCSBD7wArG_Hhqt`/16, `AaBzgnbFBD7wArG_Hhqs`/103).
- Current file sizes at this worktree's HEAD: `pipeline_module.go` 767 lines, `cache_invalidation_test.go` 631 lines, `loader_test.go` 365 lines; the three create-targets (`pipeline_module_phases.go`, `pipeline_module_cache.go`, `pipeline_module_phases_test.go`) do not yet exist.

## Execution environment and gate ownership

- The **executor** runs on darwin/arm64 in an isolated worktree; every executor acceptance command below is runnable there. Executor gates are the **narrowest that can fail for the diff** — package-scoped `go test`/`go vet`/`gofmt` on the only two packages this sprint touches (`./internal/pipeline`, `./internal/loader`). Whole-repo aggregates (`make test`, `make lint`, `make ci`) are run once, in M3, as the design-required landing evidence.
- The executor must **NOT run git write operations**. Per-milestone snapshots (command output, characterization before/after runs, gocognit JSON, mutation evidence) go to `.snap/M<k>/` inside the worktree; **the controller builds one commit per milestone** from those snapshots.
- The **Sonar exact-revision re-analysis and the PR-level evidence are CONTROLLER-verified steps outside the executor's milestone gates**: the executor records its own local complexity evidence via the gocognit report (M2 gate 4, M3 gate 2); the design requires the controller to confirm the ≤15 finding per exact function via first-party issue APIs (`/api/issues/search` scoped by project + analysis branch/PR + `go:S3776`, pinning the analysis commit SHA, with positive component/source controls showing the changed files at the expected revision) after the PR exists. Per the design's explicit fallback: if Sonar is unavailable, local gates can complete but the exact-function analysis criterion **remains pending and must be reported to the controller** — the sprint may not self-declare Success Criterion 1 from local evidence alone.
- LOC figures are forecasting units for progress reporting (touched lines: insertions + deletions, implementation + tests), not scope or acceptance caps; the design's per-file movement estimates govern.

## Schedule and commit boundaries

| Slot | Milestone | Estimate | Dependency | Boundary |
|---|---|---:|---|---|
| Day 1, h0–3 | M1 — characterization freeze + test extraction | ~3 h; 475 LOC | none | Commit 1 (controller-built); M1 gates green after `.snap/M1/` snapshot |
| Day 1, h3–8 | M2 — production extraction | ~5 h; 910 LOC | M1 green | Commit 2; M1+M2 gates green after `.snap/M2/` snapshot |
| Day 2, h0–4 | M3 — preservation proof + analysis | ~4 h; 30 LOC evidence | M2 green | Commit 3; all cumulative gates green after `.snap/M3/` snapshot; Sonar criterion pending controller |

There is no intentionally-red boundary. If a milestone cannot meet its gate, leave its work uncommitted in the worktree for the controller and do not begin the next milestone.

## M1 — Characterization freeze and test extraction (design Phase 1)

**Description.** Freeze executable behavior before any production move. Reconfirm the focused baseline is green; author the missing characterization assertions for the moved entry-point boundaries the design names — ModeCheck does not evaluate; ModeEval returns the same first-declaration value; root lookup fallback/error text; nil/provided configuration behavior; post-load callback (`afterLoad[0]` only) position; warning order across fresh and verified warm modules; debug `CACHE` hit/miss/invalid/summary forms and counters; and the mixed verified-cached-dependency/fresh-importer case (proved read-from-artifacts, importer compiled, imported call executes with a fixed expected result) — then extract the scenario/setup/assertion helpers in both test files while retaining the four named top-level entry points, all existing subtest names where practical, every observable assertion (including non-publication = absence of `CACHE_WRITE_FAILED` plus unchanged seeded timestamp/key, not zero-writes alone), `t.Helper()` in extracted helpers, and sequential subtests (these fixtures change process working directory and environment).

**Estimated LOC:** 475 (implementation 0; tests ~200 new characterization + ~240 touched in `cache_invalidation_test.go` (+100/−140) + ~35 touched in `loader_test.go` (+20/−15)).

**Files (from the design's Files to Modify/Create):**

- **Create** `internal/pipeline/pipeline_module_phases_test.go` — focused characterization tests (~150–250 LOC per design), captured on the pre-extraction baseline.
- **Modify** `internal/pipeline/cache_invalidation_test.go` (+100/−140) — extract deleted-after-load scenario, nil/known-empty contrast (cache-IO instrumentation separated from nil-bypass assertions and the known-empty positive control), per-module embedded-key verification, cached-edit/warm and NoCache scenario functions; reuse `writeCachePipelineSource`, `writeCacheBehaviorSources`, `executePipelineMain`, manifest/stamp readers, and artifact IO hooks.
- **Modify** `internal/loader/loader_test.go` (+20/−15) — extract disk and embedded snapshot scenarios / a shared exact-content assertion; retain the forced missing stdlib path, embedded AST path, and direct equality with `std.FS` bytes.

**Dependencies:** none (first milestone).

**Acceptance criteria (executor, all runnable on darwin/arm64):**

1. Pre-extraction reconfirm, recorded to `.snap/M1/baseline.txt` — focused command exits 0:
   `go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1`
2. Characterization pass on pre-extraction production code, recorded to `.snap/M1/characterize_pre.txt`:
   `go test ./internal/pipeline ./internal/loader -count=1` exits 0 with the new characterization file present.
3. After test-helper extraction, recorded to `.snap/M1/characterize_post.txt` — the same complete-package command exits 0, and the focused command (criterion 1) exits 0 again.
4. Entry-point preservation (both exit 0):
   `test "$(grep -c '^func TestCacheSource_ExactSnapshot\|^func TestCachePipeline_EmbeddedKeys\|^func TestCachePipeline_SourceEditBehavior' internal/pipeline/cache_invalidation_test.go)" = "3"` and
   `test "$(grep -c '^func TestCacheSource_ExactSnapshot' internal/loader/loader_test.go)" = "1"`
5. No parallelism introduced into environment-mutating fixtures — the search must find nothing, i.e. exits 0:
   `! grep -q 't\.Parallel' internal/pipeline/cache_invalidation_test.go internal/loader/loader_test.go`
6. Static gates, package-scoped: `gofmt -l internal/pipeline internal/loader` prints nothing; `go vet ./internal/pipeline ./internal/loader` exits 0.

**Risks.** (a) Assertions weakened inside "cleanup" helpers — mitigated by criterion 4 plus the M3 mutation oracles that must fail on the extracted assertions. (b) The mixed cached/fresh case is *new* coverage (the design: an entirely warm graph can skip import resolution, so the fresh resolver reading interfaces from the linker needs its own oracle) — it must exist before M3's MUT-IFACE-REGISTER-OMIT; M1 commits it against the baseline where the current code already passes it. (c) Warm warning parity is specified by the design as *additional* characterization, not existing coverage (V14) — it is new here, not a moved assertion.

## M2 — Production responsibility extraction (design Phase 2)

**Description.** Replace the nested orchestration of `runModuleWithCacheDependencies` (complexity 103) with an ordered coordinator holding a small number of error checks, keeping `runModuleWithContext` and `runModuleWithCacheDependencies` signatures unchanged. Extract along the design's eight responsibility seams (environment init/span lifecycle; loader/package-resolver setup with the post-load callback at the identical boundary, ordered collision checks and topological setup; dependency-digest collection and cache eligibility/key preparation; verified-lookup accounting/debug reporting distinct from "skip compilation", then payload hydration + interface registration; fresh single-module compilation calling `resolveModuleImports` → `typeCheckAndLowerModule` → `buildAndRegisterInterface` and constructor helpers in current order; fresh artifact publication and cache summary reporting; sorted post-compilation warnings and resolver registration; root lookup/evaluator configuration/evaluation + Result assembly reusing `assembleModuleResult`). Preserve the frozen sequencing map byte-for-byte in behavior: verified-hit counting **before** the nil-payload check; nil `SourceContent` warns and bypasses both lookup and publication while pointer-to-empty stays cacheable; disabled/uninitialized cache bypasses lookup; publish requires live cache/store + nonempty computed key with artifacts before manifest authorization; the current Result is returned on each early return; guard clauses inside helpers; no new deferred child-span endings or error wrapping as incidental cleanup. A package-private per-call state struct (existing per-invocation values only — Result, Config, loader/linker, compiled units, cache counters, canonical root, root debug handles) is permitted; no globals, no new interfaces for mocking, no parallel execution.

**Estimated LOC:** 910 (implementation: `pipeline_module.go` +60/−420 = 480 touched; new `pipeline_module_phases.go` ~250; new `pipeline_module_cache.go` ~180; tests: 0 new — characterization came in M1; up to ~50 of seam-specific characterization additions permitted by the design's Deferred Decisions if a moved branch is uncovered, counted inside the M1 file).

**Files (from the design's Files to Modify/Create):**

- **Modify** `internal/pipeline/pipeline_module.go` (+60/−420) — coordinator; diagnostic rule helpers (`detectModulePathCollisions`, `detectModulePrefixOverlap`, `validateModulePath` and their call semantics) unchanged.
- **Create** `internal/pipeline/pipeline_module_phases.go` (~250) — load/validate/compile-loop/finalization/root/evaluate phase helpers and the optional per-call state.
- **Create** `internal/pipeline/pipeline_module_cache.go` (~180) — cache key/lookup/hydration/publication/reporting helpers reusing `cacheRuntime` unchanged.
- (`internal/pipeline/pipeline_module_phases_test.go` may gain at most the smallest seam characterization, per Deferred Decisions.)

**Dependencies:** M1 green commit (characterization pins exist and pass before orchestration moves).

**Acceptance criteria (executor):**

1. Focused equivalence — exits 0, output recorded to `.snap/M2/focused.txt`:
   `go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1`
2. Complete packages — `go test ./internal/pipeline ./internal/loader -count=1` exits 0 (includes the M1 characterization file against the new coordinator).
3. Package-scoped static gates — `gofmt -l internal/pipeline internal/loader` prints nothing; `go vet ./internal/pipeline ./internal/loader` exits 0; `go build ./internal/pipeline ./internal/loader` exits 0.
4. Local complexity evidence for every exact target and every new helper — exits 0 with the check passing, JSON banked to `.snap/M2/gocognit.json`:
   `go run github.com/uudashr/gocognit/cmd/gocognit@latest -json internal/pipeline/pipeline_module.go internal/pipeline/pipeline_module_phases.go internal/pipeline/pipeline_module_cache.go > .snap/M2/gocognit.json` followed by a `jq` assertion that `runModuleWithCacheDependencies` and every function declared in the two new files report cognitive complexity ≤ 15 (the two test files' four named functions are checked in M3's final report; authoritative ≤15 confirmation for all five Sonar targets is the controller's exact-revision API step, outside this executor gate).
5. Frozen file-set guard — `git status --short -- internal/pipeline internal/loader sonar-project.properties .github/workflows` lists only `pipeline_module.go` (M), the two new production files (??), and the M1 test files; any other entry (parser/type/lowering/cache-store/schema/Sonar settings) fails the milestone.
6. Coordinator shape sanity — `grep -n 'func runModuleWithCacheDependencies' internal/pipeline/pipeline_module.go` still matches (signature retained) and `test "$(awk '/^func \(.*\) runModuleWithCacheDependencies|^func runModuleWithCacheDependencies/,/^}/' internal/pipeline/pipeline_module.go | grep -cE '^\s+(if|for|switch|select)\b')" -le 15` exits 0 (few branch points remain in the coordinator body itself).

**Risks.** (a) Cache skip/validation reordering (design: high) — sequencing map frozen; warm-hit instrumentation and the imported-dependency mutation prove order in M3. (b) Narrowed helper loses partial Result/debug state on error (high) — M1 characterization pins root debug handles, timings, error/result boundaries before extraction. (c) Complexity migrates into helpers (medium) — criterion 4 analyzes every changed/new function, per the goal "complexity must not merely move". (d) `afterLoad[0]`-only callback boundary or hit-count-before-nil-check order inverted quietly — both are explicitly pinned by M1 characterization and the design's compatibility note.

## M3 — Preservation proof, race/aggregate gates, and analysis (design Phase 3)

**Description.** Prove the refactor preserved behavior, then land. The executor runs the ten named mutation probes — **each in its own isolated disposable execution worktree** (a filesystem copy of the tree under `.snap/M3/mutations/`, one directory per probe named after its mutation ID, e.g. `.snap/M3/mutations/MUT-SOURCE-EMPTY/`; per the design, each mutation is restored there and its failing assertion plus restored-green result are banked; the design's Non-Goals confine mutation probing to that disposable tree — the tracked tree is never mutated), then runs the focused packages under `-race`, the repository aggregate gates, and the final local complexity report; the controller then performs the exact-revision Sonar verification and PR-level evidence after building Commit 3 and the PR. If any mutation survives (no failing oracle), the executor strengthens the corresponding observable assertion before declaring preservation complete — a surviving mutation is a failed milestone, not a waived one. A compile error or unrelated environment failure is not a kill: restore, fix the probe, rerun.

**Estimated LOC:** 30 (implementation 0 net — mutations are transient and reverted inside the disposable copies; ~30 lines of banked evidence/deviation notes appended to the design doc's implementation-evidence section by the controller at landing, per Files to Modify/Create; test assertions strengthened only if a mutation survives, which fails the milestone until done).

**Files (from the design's Files to Modify/Create):** no tracked Go changes; `.snap/M3/` evidence tree (mutation diffs, oracle outputs, restored-green outputs, race/aggregate logs, final `gocognit.json`); `design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md` evidence section (controller-owned at landing).

**Dependencies:** M2 green commit (mutations probe the extracted code, not the 103-line original).

**Named mutation probes and required failing oracles (all ten, design table verbatim; each disposable copy is created, probed, and restored green before the next probe begins):**

| Mutation (owned change applied alone in `.snap/M3/mutations/<ID>/`) | Required failing oracle / positive control | Probe commands (oracle must FAIL with mutation, PASS restored) |
|---|---|---|
| MUT-SOURCE-EMPTY — hash empty text for every available snapshot | Deleted-after-load key comparison and embedded expected/nonempty-key comparison fail; restored controls publish real keys | `go -C .snap/M3/mutations/MUT-SOURCE-EMPTY test ./internal/pipeline -run 'TestCacheSource_ExactSnapshot\|TestCachePipeline_EmbeddedKeys' -count=1` (expect FAIL); same command in tracked tree (expect PASS) |
| MUT-SOURCE-REREAD — replace retained text with a filesystem reread | Deleted-after-load case fails at source identity or compilation; deletion is verified, ordinary source still compiles | `go -C .snap/M3/mutations/MUT-SOURCE-REREAD test ./internal/pipeline -run 'TestCacheSource_ExactSnapshot' -count=1` (expect FAIL); restored: PASS |
| MUT-NIL-PUBLISH — remove the nonempty-key publication guard | Nil-source case detects `CACHE_WRITE_FAILED`/publication attempt; seeded key/timestamp and zero-IO assertions remain | `go -C .snap/M3/mutations/MUT-NIL-PUBLISH test ./internal/pipeline -run 'TestCacheSource_ExactSnapshot' -count=1` (expect FAIL); restored: PASS |
| MUT-EMPTY-BYPASS — treat pointer-to-empty as unavailable | Known-empty case fails its real manifest-entry/key assertion; nil case still demonstrates the opposite arm | `go -C .snap/M3/mutations/MUT-EMPTY-BYPASS test ./internal/pipeline -run 'TestCacheSource_ExactSnapshot' -count=1` (expect FAIL); restored: PASS |
| MUT-WARM-OFF — force verified cached payload to take fresh compilation | Positive dependency artifact reads and zero warm encodes distinguish real reuse from merely equal output | `go -C .snap/M3/mutations/MUT-WARM-OFF test ./internal/pipeline -run 'TestCachePipeline_SourceEditBehavior' -count=1` (expect FAIL); restored: PASS |
| MUT-CACHED-CORE-OMIT — omit copying cached Core during hydration | Warm execution or required Core assertion fails after positive artifact reads | `go -C .snap/M3/mutations/MUT-CACHED-CORE-OMIT test ./internal/pipeline -run 'TestCachePipeline_SourceEditBehavior' -count=1` (expect FAIL); restored: PASS |
| MUT-IFACE-REGISTER-OMIT — skip cached interface registration | Mixed cached-dependency/fresh-importer case fails resolving or executing the imported call; positive artifact reads and fresh-importer compilation prove both branches were reached | `go -C .snap/M3/mutations/MUT-IFACE-REGISTER-OMIT test ./internal/pipeline -count=1` (expect FAIL in the mixed case; full-package run guarantees the new characterization test is included regardless of its final name); restored: PASS |
| MUT-NOCACHE-PERSIST — initialize/use cache despite NoCache | NoCache compile-directory absence fails; cached control produces a manifest | `go -C .snap/M3/mutations/MUT-NOCACHE-PERSIST test ./internal/pipeline -run 'TestCachePipeline_SourceEditBehavior' -count=1` (expect FAIL); restored: PASS |
| MUT-SNAPSHOT-LEAK — copy source into assembled runtime LoadedModule | Runtime snapshot-omission assertion fails on an actually executed module | `go -C .snap/M3/mutations/MUT-SNAPSHOT-LEAK test ./internal/pipeline -run 'TestCachePipeline_EmbeddedKeys\|TestCachePipeline_SourceEditBehavior' -count=1` (expect FAIL); restored: PASS |
| MUT-WARNING-SKIP — omit post-compile warning collection | Positive cold/warm warning fixture fails; correct-order negative control still has zero target warnings | `go -C .snap/M3/mutations/MUT-WARNING-SKIP test ./internal/pipeline -run 'TestSplitArgWarning_Integration' -count=1` (expect FAIL) and `go -C .snap/M3/mutations/MUT-WARNING-SKIP test ./internal/pipeline -count=1` (M1 warm-parity characterization must also FAIL); restored: both PASS |

MUT-SOURCE-REREAD and MUT-SNAPSHOT-LEAK may temporarily touch their original loader/result-assembly locations **solely inside the disposable mutation worktree** (design: not implementation scope expansion).

**Acceptance criteria (executor):**

1. Ten mutation evidence rows banked under `.snap/M3/mutations/<ID>/` — for every row above: the oracle command exits non-zero in the disposable copy with the named assertion failing, and exits 0 after restore (restored-green), recorded as `oracle_fail.txt` + `restored_green.txt`; a surviving mutation upgrades to "strengthen the assertion, then rerun" rather than a pass.
2. Race gate on the touched packages — `go test -race ./internal/pipeline ./internal/loader -count=1` exits 0 (design: focused packages under `-race` after all changes), recorded to `.snap/M3/race.txt`.
3. Aggregate landing evidence — `make test`, `make lint`, and `make ci` each exit 0, recorded to `.snap/M3/make_{test,lint,ci}.txt` (design: use `make ci` for the aggregate gate set rather than a hand-picked subset; if `check-changelog` or any aggregate sub-gate requires a landing artifact, report it to the controller — do not hand-edit unrelated bookkeeping).
4. Final local complexity report over all five files — `go run github.com/uudashr/gocognit/cmd/gocognit@latest -json internal/pipeline/pipeline_module.go internal/pipeline/pipeline_module_phases.go internal/pipeline/pipeline_module_cache.go internal/pipeline/cache_invalidation_test.go internal/loader/loader_test.go > .snap/M3/gocognit.json` with a `jq` assertion that the four named test functions, `runModuleWithCacheDependencies`, and every newly declared helper function are ≤ 15.
5. Cumulative focused command exits 0 one final time:
   `go test ./internal/pipeline ./internal/loader -run 'TestCacheSource_ExactSnapshot|TestCachePipeline_(EmbeddedKeys|SourceEditBehavior|WriteFailure)|TestCacheArtifacts_Migration|TestCacheKey_InvalidatesOnSourceEdit|TestContractExpressionsFullyLowered|TestSplitArgWarning_Integration' -count=1`

**Controller-verified (outside executor gates, recorded on the sprint state by the controller):** exact-revision Sonar re-analysis — `/api/issues/search` scoped by project `sunholo-data_ailang` + analysis branch/PR + `go:S3776` pinned to the analysis commit SHA — confirming each of the five old→new function mappings reports ≤15 or has no S3776 finding in a *successful, correctly scoped* analysis with confirmed file inclusion (fetch all pages or query exact components; a zero-length PR issue list without such an analysis is insufficient; the three older same-file findings may serve as same-scope rule controls; Sonar scan continuing on error in CI is not evidence).

**Risks.** (a) Mutation probes run in the tracked tree (forbidden) — the disposable-copy rule and `.snap/` layout prevent contaminated commits; the controller verifies `.snap/` is not committed. (b) A compile error or unrelated environment failure miscounted as a kill — design rule: it is not; restore and rerun. (c) Green CI masks a missing Sonar scan (medium) — Sonar is a separate controller-verified criterion, never inferred from gate status. (d) Sonar unavailable at sprint end — per design fallback, local gates complete but the exact-function criterion stays explicitly pending in the sprint state (`blocked_reason`/notes), reported to the controller. (e) External Sonar queue time inflates elapsed time — expected; it is not proof of completion either way.

## Ordered task list (each milestone lands with its own green gate run)

1. **M1.1** Reconfirm identity + focused baseline → `.snap/M1/baseline.txt` (M1 criterion 1).
2. **M1.2** Author `pipeline_module_phases_test.go` characterization (modes, root lookup, config nil/provided, callback position, warning order, debug CACHE forms/counters, mixed cached/fresh case) against the untouched production code; run complete-package gate → `.snap/M1/characterize_pre.txt`.
3. **M1.3** Extract scenario/setup/assertion helpers in `cache_invalidation_test.go` per the design's four test-extraction seams (preserving names/assertions/`t.Helper()`/sequential order).
4. **M1.4** Extract disk/embedded snapshot scenarios in `loader_test.go`.
5. **M1.5** Run M1 criteria 3–6 (post-extraction complete packages, focused command, entry-point greps, no-parallel greps, gofmt/vet) → snapshot `.snap/M1/`; hand to controller for **Commit 1**. Do not proceed if any gate is red.
6. **M2.1** Create `pipeline_module_phases.go` with the load/validate/compile-loop/finalization/root/evaluate seams and optional per-call state; rewire the coordinator body of `runModuleWithCacheDependencies` in `pipeline_module.go` to ordered phase calls (signatures unchanged).
7. **M2.2** Create `pipeline_module_cache.go` with eligibility/key, verified-lookup accounting + hydration + interface registration, publication, and reporting helpers (reusing `cacheRuntime` unchanged; hit counted before nil-payload check).
8. **M2.3** After each coherent extraction, re-run `go test ./internal/pipeline -count=1` (design: re-run affected package tests after each coherent extraction).
9. **M2.4** Run M2 criteria 1–6 (focused, complete packages, static gates, gocognit ≤15 on production files + all new functions, frozen file-set guard, coordinator-shape check) → snapshot `.snap/M2/`; hand to controller for **Commit 2**.
10. **M3.1** For each of the ten mutations in the order listed above: copy tree to `.snap/M3/mutations/<ID>/`, apply only the owned mutation, run the oracle command (expect FAIL with the named assertion), restore, rerun (expect PASS), bank both outputs.
11. **M3.2** Run the `-race` package gate → `.snap/M3/race.txt`.
12. **M3.3** Run `make test`, `make lint`, `make ci` → `.snap/M3/`.
13. **M3.4** Run the final gocognit report + ≤15 `jq` assertion over all five files → `.snap/M3/gocognit.json`; final focused command run → snapshot `.snap/M3/`; hand to controller for **Commit 3** and PR.
14. **Controller (post-PR, outside executor gates):** exact-revision Sonar API verification per M3; append implementation evidence/deviations to the design doc; record outcomes in the sprint JSON; hand off for independent evaluation.

## Success metrics (mapping to the design's Success Criteria)

1. All five target functions ≤15 **with exact-revision analysis evidence** — controller-verified Sonar API result after the PR (executor's local gocognit ≤15 is the supporting, not sufficient, evidence).
2. Every newly introduced/changed helper satisfies ≤15 (executor: M2 criterion 4 / M3 criterion 4).
3. All tests passing with target assertions, names, source identity, warm reuse and NoCache/failure behavior preserved (executor: focused + complete-package + race gates per milestone).
4. All ten named mutation probes fail their intended assertions and restored code passes (executor: M3 criterion 1).
5. Normal repository CI passes, Sonar completion verified separately (executor: `make ci`; controller: Sonar).
6. Documentation updated with implementation, tests, and analysis results (controller: design-doc evidence section).
7. Diff stays within frozen semantics and bounded file set (executor: M2 criterion 5).

## Docs to update

- The design doc itself (`design_docs/planned/v0_35_2/m-cachesrc-cognitive-complexity.md`): controller appends final implementation evidence, old→new function mappings, complexity readings, mutation results, and deviations after the downstream gates (it is the only documentation row in the design's Files to Modify/Create).
- `.ailang/state/sprints/sprint_M-CACHESRC-COGNITIVE-COMPLEXITY.json`: controller updates milestone `passes`/`started`/`completed`, `criteria_status`, and (if Sonar is unavailable at landing) `blocked_reason`.
- CHANGELOG/PR metadata: only if the aggregate `make ci` gates require it at landing; that is controller bookkeeping, not executor scope.

## Noted tensions for the controller (reported, not resolved by editing the design)

1. **Sonar Success Criterion cannot close inside executor gates.** Success Criterion 1 demands exact-revision Sonar evidence, but CI runs the scan `continue-on-error` and only the controller can perform the first-party API verification after the PR exists. The plan therefore keeps SC1 pending-with-evidence at executor completion; the design's own fallback clause authorizes exactly this.
2. **The characterization file spans two milestones.** `pipeline_module_phases_test.go` is authored in M1 against pre-extraction boundaries, but the design's Deferred Decisions also permit "the smallest additional characterization fixtures needed to cover moved branches," which can only be known during M2. The plan allows up to ~50 LOC of seam-specific additions in M2 inside the same file rather than inventing new scope.
3. **Mutation oracles depend on M1's new tests by construction.** MUT-IFACE-REGISTER-OMIT and the warm arm of MUT-WARNING-SKIP require the mixed cached/fresh case and warm warning parity, both *new* characterization (the design states an entirely warm graph cannot prove interface registration, and V14 states warm parity is not existing coverage). Their probe commands therefore use the full-package run, which includes the new tests regardless of their final agent-chosen names.
4. **LOC units.** JSON LOC figures are touched-line forecasts (insertions + deletions), consistent with the design's per-file movement estimates; the production milestone shows large deletion counts by design (−420 from the coordinator), so net-new-line velocity would mislead.

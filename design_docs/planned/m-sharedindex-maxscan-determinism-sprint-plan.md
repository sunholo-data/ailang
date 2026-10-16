# M-SHAREDINDEX-MAXSCAN-DETERMINISM sprint plan

Design: [approved design](m-sharedindex-maxscan-determinism.md)

Status: Planned; implementation awaits the coordinator plan approval handoff.
Source task: task-70869507. Planning worktree: coordinator/task-db433f56.
Target: next patch after current std v0.54.12; design target v0.54.8 is historical, no version bump in this sprint.
Duration: 1 working day, approximately 5 hours including 25% contingency.
Estimated total: 300 LOC (implementation 55, tests 160, documentation/example 85).
Risk: low correctness scope, moderate test/toolchain uncertainty.

## Goal and current state

Restore replay-exact Strict bounded similarity queries in both backends and sort shared-memory key enumeration, as approved in decisions D1–D5. Both search methods still stop within raw map iteration; sorting results cannot stabilize candidate selection. Existing tests only establish unbounded determinism. The design audits both search paths, shared-memory enumeration, sorted namespaces and interpreter/VM builtin dispatch, satisfying the systemic-analysis gate.

The worktree was clean before planning. No implementation changes are part of planning. Source sizes: sharedindex_inmemory.go 241 lines, sharedindex_test.go 452, sharedmem_inmemory.go 157, sharedmem_test.go 418, sharedindex.go 350. All are comfortably below 800 lines; put bounded-search tests in a new sharedindex_determinism_test.go to keep the existing suite small.

## Capacity and dependencies

The skill velocity script found only one recent documentation commit and no usable HEAD parent diff in this checkout. Historical changelog LOC figures are not recent throughput evidence, so no measured LOC/day claim is possible. Use bottom-up estimates: M1 90 minutes, M2 45 minutes, M3 90 minutes, plus 75 minutes of validation/contingency. Planned capacity is 300 LOC/day, an estimate rather than measured velocity.

No package, service or API dependency is added. Registry searches `ailang pkg search sharedindex` and `ailang pkg search sharedmem` returned no packages. Each milestone has a `none` reuse decision: this fixes existing runtime methods and their tests/docs; a registry dependency cannot repair them.

Go is absent on the planning host. The installed CLI reports active teaching prompt v0.16.7 and warns it may be stale; source std/VERSION is v0.54.12. The executor must use a Go-enabled environment and rebuild the CLI before accepting validation output. Do not treat the installed binary as proof of source behavior.

## Milestones

### M1: Deterministic bounded SharedIndex searches (~180 LOC)

Estimated effort: 90 minutes. Dependencies: none.

Files: internal/effects/sharedindex_inmemory.go; internal/effects/sharedindex.go; new internal/effects/sharedindex_determinism_test.go.

Write failing tests first. Add sortedNamespaceKeys and use sorted keys under the existing RLock for Strict queries in both methods. Keep the raw map path for BestEffort. Count examined entries before skipping embedding-less entries. Keep scoring, score/key result ordering, metadata, topK handling and public signatures unchanged. Document O(N log N) key preparation and O(N) storage even with small maxScan: the bound limits entries examined/scored, not total sorting cost. Run go test ./internal/effects -run SharedIndex -count=1.

Acceptance:

- [ ] Both Strict search methods score only the lexicographically first maxScan keys under the existing read lock.
- [ ] Six-key SimHash repro returns k1 with identical full results on 100 calls; an outside-prefix perfect match is excluded.
- [ ] Embedding tests prove identical bounded results and that missing embeddings consume scan budget.
- [ ] Boundary tests cover empty namespaces, topK truncation, tied scores, maxScan zero, negative and greater than namespace size; BestEffort retains raw map iteration.
- [ ] Focused SharedIndex tests pass.

Risk: Prefix membership must be asserted exactly with topK large enough to expose the entire prefix; repeated runs alone are insufficient.

Registry reuse: none; existing core behavior, searches found no package substitute.

### M2: Canonical shared-memory key enumeration (~35 LOC)

Estimated effort: 45 minutes. Dependencies: M1.

Files: internal/effects/sharedmem_inmemory.go; internal/effects/sharedmem_test.go; internal/builtins/sharedmem.go if its Keys description needs clarification.

Sort the copied Keys slice before return within the existing synchronization scheme. Update ordering comments. Test exact sorted output, empty cache and mutations. Run go test ./internal/effects -run SharedCache -count=1.

Acceptance:

- [ ] InMemorySharedCache.Keys returns ascending keys for non-sorted insertions, repeated calls and after deletion.
- [ ] Existing namespace isolation and shared-memory tests pass; no cache interface or capability changes.

Risk: Public enumeration order deliberately changes to ascending; keep cache values and namespace semantics unchanged.

Registry reuse: none; existing core behavior, searches found no package substitute.

### M3: Contract documentation and cross-engine replay validation (~85 LOC)

Estimated effort: 90 minutes. Dependencies: M1, M2.

Files: std/sharedindex.ail; std/sem.ail; new examples/runnable/sharedindex_bounded_determinism.ail; new changelogs/unreleased/2026-10-16-sharedindex-maxscan-determinism.md.

Document bounded Strict recall and examined-entry semantics. Add a compact six-key repro with fixed metadata, non-sorted insertion and printed results. Execute fresh processes ten times per engine using a rebuilt CLI: ailang run --caps IO,SharedIndex --entry main <example> and ailang run --caps IO,SharedIndex --bytecode --strict-bytecode --entry main <example>. Compare outputs against expected k1/score 1.0 and across runs/engines, using temporary output files and diff. Add a changelog fragment opening with ### .

Acceptance:

- [ ] Interface and stdlib comments explain Strict prefix selection, BestEffort unspecified subset, examined-entry budget and maxScan zero for full recall.
- [ ] A runnable six-key bounded-search example prints actual result keys and scores; ten fresh interpreter and ten strict-bytecode runs match a fixed expected result and each other.
- [ ] make test-changed, make check-file-sizes, make check-changelog, make verify-stdlib and make verify-examples pass against the rebuilt binary.
- [ ] An unreleased changelog fragment documents both observable ordering fixes and the bounded-scan recall trade-off.

Risk: Strict bytecode must really execute without fallback; compile or capability errors are failures, never parity success.

Registry reuse: none; existing core behavior, searches found no package substitute.

## Day-one sequence and verification

Morning: M1 failing regressions, fix, focused tests (90 minutes); M2 enumeration regression and fix (45 minutes). Afternoon: M3 docs, runnable fixture and parity checks (90 minutes); remaining 75 minutes for focused validation and contingency. Each milestone fits within a 60–90 minute executor session.

Run `make test-changed` (touched Go packages plus test-core), `make check-file-sizes`, `make check-changelog`, `make verify-stdlib`, and `make verify-examples`. Run the touched effects package with the race detector if supported by the executor toolchain. Full-suite coverage belongs to CI; this supersedes the design's historical `make test` acceptance line. No new scripts are required: existing CLI, make targets and shell diff suffice. Coverage target: meaningful regression assertions for every changed branch, rather than an invented repository-wide percentage.

AILANG prompt version loaded: v0.16.7 (full prompt loaded during planning). Executor must reload `ailang prompt` from its rebuilt CLI before editing any .ail file.

Showcase module: examples/runnable/sharedindex_bounded_determinism.ail.

| Requirement | Decision |
|---|---|
| Contracts | skip: effectful runtime determinism fixture; backend assertions check exact prefix and metadata |
| Effects | include: exported main() -> () ! {IO, SharedIndex} |
| Inline tests | skip: process-local SharedIndex setup and engine comparison use CLI output plus Go assertions |

Existing examples/runnable/stdlib_sharedindex.ail must continue working with unbounded queries. examples/reference/neural_semantic_search.ail remains a compatibility reference; no unrelated changes or new neural features are planned.

## Completion and handoff

Every JSON milestone begins with passes=null and is incomplete until its criteria have evidence. Preserve fixed milestone IDs and dependencies across resumed execution. No GitHub issue number was supplied or found in the design; github_issues remains empty rather than inferred from task IDs. The coordinator consumes the artifact markers and routes plan approval to sprint-executor; no duplicate executor task is sent during planning. After implementation, sprint-evaluator assesses the approved design and all criteria.

# Sprint Plan: M-ALIAS-BODY-CLOSURE

## Summary

Close interface alias bodies, exported schemes and constructor types over the defining module so an unrelated imported Item cannot change Box.items or function parameters. Implement the approved [design](m-alias-body-closure.md), scheduling the parent M-TYPE-NAME-SHADOW M2.

**Duration:** 4 engineering days (24 hours implementation/verification + 8 hours buffer).
**Estimated total:** 720 LOC: approximately 260 implementation, 380 tests, 50 examples and 30 documentation.
**Risk:** Medium; exhaustive rebuilding, recursive types, parameterized alias heads and digest finalization need care.
**Status:** Plan prepared for coordinator approval; implementation has not started. The approved design authorizes planning, not execution. Plan approval/merge triggers sprint-executor through the coordinator.
**Target:** Next unreleased patch; retain v0_52_4 artifact location for lineage. Source std/VERSION is already v0.52.5; no release bump is part of this sprint.

## Current status and velocity

- M1 local-shadow guards and M3 transitive cache dependencies from the parent design are present. Closure is absent; cacheKeyVersion is v5.
- buildAndRegisterInterface currently builds then embeds transitive aliases without closing exported types. Its caller is pipeline_module_phases.go, which must also pass the local alias environment.
- The design contains a systemic audit, three verified triggers and controls for nominal, cyclic, polymorphic, transitive and constructor cases. Its quorum log records controller PROCEED with both external seats absent; this is not an independent-review pass.
- Ran the existing analyze_velocity.sh for seven days. This shallow checkout exposes only commit 86195621 and no usable diff/LOC history; the script reports no LOC metrics. CHANGELOG.md points to changelogs/v0.32-current.md, whose latest v0.52.5 entries cover record zero values, MCP checks and media embeddings. No measured LOC/day can be inferred.
- Planning throughput is an assumption of 180 LOC/day (720/4), anchored to the design's 3–4 day estimate, with one day including verification and buffer. Re-estimate after M1 if cycle/variant handling exceeds one day.
- Go and jq are absent from PATH. No code tests or coverage baseline ran during planning. Executor must provision the existing cloud environment/toolchain, build the source binary, and capture a baseline before editing semantics. The installed ailang v0.52.5 binary warns that it may be stale.

## Registry reuse audit

Ran `ailang pkg search 'type alias'` and `ailang pkg search compiler`; both returned no packages. No candidates existed to inspect with pkg info/docs. M1–M4 each choose **none**: this is internal Go compiler interface/cache behavior and its regression verification, which cannot be supplied by an AILANG package. Reuse existing types traversal conventions, interface builders, checkModules harness and cache test helpers; add no dependency or new script.

## Milestones and daily tasks

### M1: Close alias bodies in the defining module (~330 LOC)

**Dependencies:** None
**Schedule:** Day 1; 6 hours + 2 hours buffer.

Create internal/pipeline/alias_body_closure.go and alias_body_closure_test.go; update pipeline_module_compile.go and pipeline_module_phases.go. Use elaborator.GetTypeAliases() plus the same post-shadow imports used by the checker; do not derive the environment solely from exported aliases.

First bank failing T1/T2 and four-module tests. Implement a non-mutating rebuilding walker with sorted alias roots, cycle guards and safe memoization; test self/mutual recursion and traversal-order independence. Enumerate actual Type implementors rather than relying on a hand-maintained switch alone. Recurse through TApp arguments while preserving applied parameterized alias heads; do not expand a polymorphic head without substitution. Wire alias closure before transitive embedding.

**Acceptance criteria:**

- [ ] Walker covers every concrete types.Type implementation, including TLabelled, v2 records/functions, rows, arrays and maps; nested aliases close without mutating inputs.
- [ ] Cycles terminate deterministically; TypeName is preserved or assigned; primitives, nominal types and type variables retain identity.
- [ ] T1 explicit-import and T2 both-import-order regressions pass; four-module navigation/sol/trappist regression passes.
- [ ] Closure uses local aliases over post-shadow imported aliases, including unexported local dependencies; AliasParams remain intact.

### M2: Close export schemes and constructor types (~220 LOC)

**Dependencies:** M1
**Schedule:** Day 2; 6 hours + 2 hours buffer.

Extend alias_body_closure.go/tests; update local_type_shadows_import_test.go. Inspect internal/iface/builder.go and iface.go for a minimal digest-finalization hook if needed; add digest/serialization checks in internal/iface or pipeline tests.

**File-size constraint:** internal/iface/builder.go is already 782 lines and the CI gate (`make check-file-sizes`) fails any file above 800. Any exported digest-finalize helper (re-computing iface.Digest after closure, because computeDigest runs inside the builder at builder.go:539 before closure) goes in a NEW file under internal/iface/ (e.g. internal/iface/digest_finalize.go, with its test alongside), not in builder.go. builder.go may change only by the minimal lines needed to call or expose the existing digest computation.

Bank T3, constructor and capture tests, then close schemes and constructors without altering scheme binders/constraints or nominal identity. Copy shared schemes before rewriting. The builder currently computes Digest before returning the interface (builder.go:539), so post-build mutation requires finalizing the digest after closure and embedding; do not retain a digest describing pre-closure exports. Preserve the existing digest format and aliasDigest contract.

**Acceptance criteria:**

- [ ] T3 annotation-only scheme regression passes; quantified variables, constraints and effects are preserved.
- [ ] Constructor field/result aliases close while nominal ADT results remain nominal; constructor pattern regressions pass.
- [ ] CapturedImportedAliasIsLoud becomes a positive regression; a residual nominal-shadow control still emits TC_TYPE_SHADOW_001.
- [ ] Final interface digest reflects closed schemes and constructors, is deterministic and survives serialization; alias digest coverage remains intact.
- [ ] Any digest-finalize helper lives in a new internal/iface/ file; internal/iface/builder.go stays at or below 800 lines.

### M3: Invalidate stale caches and document residuals (~100 LOC)

**Dependencies:** M2
**Schedule:** Day 3; 6 hours + 2 hours buffer.

Update internal/pipeline/cache_key.go, cache_invalidation_test.go, type_name_shadow.go, docs/LIMITATIONS.md and the parent m-type-name-shadow-and-cache.md; extend cache tests using existing helpers.

Use a synthesized v5 cache/manifest fixture rather than depending on an old executable. Assert both rejection of stale entries and stable warm hits. Keep residual nominal capture protection. Correct the parent checklist so applied-alias instantiation and import ambiguity are not accidentally claimed as delivered.

**Acceptance criteria:**

- [ ] cacheKeyVersion and migration assertions use v6; a v5 cache misses, a fresh v6 compile populates the cache, and an unchanged subsequent compile hits.
- [ ] Alias edits still invalidate transitive dependents; cached and uncached compilation give equivalent results.
- [ ] Shadow guard comments, parent M2 scope/status and docs/LIMITATIONS.md describe closure and residual parameterized/nominal/import-name behavior accurately.

### M4: Verify runtime examples and regression matrix (~70 LOC)

**Dependencies:** M3
**Schedule:** Day 4; 6 hours verification + 2 hours contingency.

Create examples/alias_body_closure/{ailang.toml,a.ail,b.ail,main.ail}; add runtime controls to alias_body_closure_test.go if existing harnesses do not cover them; update changelogs/v0.32-current.md with the delivered fix and limitations.

Read `ailang prompt` before writing any .ail files. Validate examples with the newly built source binary and existing package conventions. Run targeted tests first, then the repository checks listed below; report baseline failures separately. Do not run the full `make test` locally: the full suite runs in CI, and a full `make test` in RAM-backed /tmp has crashed executors with SIGBUS. Do not modify or rename types in the external project except in an isolated scratch copy.

**Acceptance criteria:**

- [ ] New examples/alias_body_closure package (ailang.toml, a.ail, b.ail, main.ail) checks and run with --args-json 1 returns 1 with both colliding imports.
- [ ] T1/T2/T3 runtime checks and examples/intra_package_imports pass; record-update and args-json alias decoding controls pass.
- [ ] make test-core and go test ./internal/pipeline/... ./internal/types/... ./internal/iface/... pass, then make lint check-boundaries check-file-sizes passes (full suite is left to CI).
- [ ] Stapledons workaround-removal verification is recorded if accessible; otherwise record unavailable and rely on the mandatory four-module automated reproduction.
- [ ] Sprint-evaluator receives plan, progress, test evidence and explicit remaining limitations after execution.

## Scope and risks

Applied parameterized aliases remain deferred as permitted by the design: their TApp heads remain nominal/open while safe argument closure proceeds. Add a control to prevent changing existing alias-poly behavior and document that applied heads can still resolve in importer scope. Recursive aliases retain opaque cycle references, so the universal closed-interface guarantee is limited to nonrecursive nullary aliases. Module-qualified nominal identity, imported-name ambiguity diagnostics and REPL/WASM/SMT top-level name merging remain outside this sprint.

The primary risks are wrong closure environments, context-dependent cycle memoization, mutating shared types, and stale digests. Mitigate with unexported-local/imported alias tests, self/mutual-cycle tests in multiple traversal orders, input immutability checks, and digest/serialization assertions. Keep TypeName tags, AliasParams, row tails, effect rows and scheme binders intact. If the residual TC_TYPE_SHADOW_001 control proves impossible under the specified closure, record the evidence and revise the design rather than weakening nominal guards to force a passing test.

## Validation and success metrics

All three report triggers must typecheck and run to 1; both import orders must be covered. Preserve the independent b.Item call and reject incompatible shapes, proving that accepting count(mk()) did not collapse distinct records. Require the four-module production-shape test even if the external project is unavailable.

Run `make test-core` plus `go test ./internal/pipeline/... ./internal/types/... ./internal/iface/...`, then `make lint check-boundaries check-file-sizes` after the change. Do not run the full `make test` locally (it runs in CI; a full run in RAM-backed /tmp crashed executors with SIGBUS). Existing cross_module_nonrecord_alias, cross_package_alias, alias_poly, ctor_alias_pattern, local_type_shadows_import, cache_transitive_alias and cache_alias_digest_stable suites must pass except the intentional captured-alias flip. Check the new package and existing intra_package_imports example using supported CLI entry points. Coverage goal is every walker variant, all three triggers and cold/warm cache behavior; record package coverage with targeted `go test -cover`, without inventing a baseline percentage.

## Execution handoff

Progress file: `.ailang/state/sprints/sprint_M-ALIAS-BODY-CLOSURE.json`.
All milestones start with passes/started/completed/notes null. M1 → M2 → M3 → M4 is the critical path. No parallel implementation is planned. GitHub issue: #1614. The implementation PR body uses `Fixes #1614`. The parent's #1275 is related cache work and must not be auto-closed.

Coordinator plan approval/merge is the execution gate; do not self-approve or start sprint-executor during planning. After execution, hand off the complete evidence to sprint-evaluator. No architectural decision is outstanding for this scoped plan; actual release version and external scratch-repo access are nonblocking operational details.

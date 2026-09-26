# V1 Mission Iteration 346 — Independent Judgement of PR #1071

**Subject:** `sprint/v1-iter341-cachesrc-cognitive`, head `508a969399d359d3bb5576418e26472d76763e9d`, landing on `origin/dev` (`1fd1c7a706f2f089b6d6b5a1f3211de0a33e5cf1`)
**Judge:** independent (fresh, no prior context, no code authored)
**Verdict: LAND**
**Score: 91/100** (70 = pass)
**Blocking findings: 0**
**Non-blocking findings: 4**

---

## 1. Score breakdown

| Category | Points | Notes |
|---|---|---|
| Design-doc conformance | 24/25 | Extraction matches the design's phase/cache split; one evidence-integrity ding (see below) |
| Test quality / mutation resistance | 33/40 | 3/6 independently-designed mutations survived (all low/medium severity, none behavior-critical); the one sprint-plan-named mutation I ran independently (MUT-IFACE-REGISTER-OMIT) was correctly killed |
| Evidence integrity | 18/20 | Byte-identical rebase and killed-mutation claims hold; the "103" base-complexity figure is measurably wrong (it's 106) and nothing in the four dead prior iterations left a shred of the claimed 10-mutation evidence trail on disk |
| Code quality | 16/15 | Genuinely careful extraction — preserves a non-obvious counter-ordering edge case (see §3); capped at category max |

**Total: 91/100**

---

## 2. Re-derived load-bearing numbers

### 2.1 Rebase byte-identity — CONFIRMED
```
git diff origin/dev..508a969399d3   -> 206803 bytes, sha256 31f660f8...b7b7c8148
git diff d794cac25..f407d187a       -> 206803 bytes, sha256 31f660f8...b7b7c8148  (identical)
diff of the two patch files: 0 lines difference
```
Merge-base of head against `origin/dev` is `origin/dev` itself (`1fd1c7a7...`), confirming a clean fast-forward-style rebase with no dev-side drift folded in.

### 2.2 Cognitive complexity — mostly CONFIRMED, one number is wrong
Using `gocognit` (freshly installed, `github.com/uudashr/gocognit/cmd/gocognit@latest`):

| Function | Base (`d794cac25`) | Head (`508a96939`) |
|---|---|---|
| `runModuleWithCacheDependencies` | **106** | **0** (now a 2-line wrapper: `newModulePipelineState(...).run()`) |
| `detectModulePathCollisions` | 30 | 30 (unchanged, Non-Goal) |
| `validateModulePath` | 25 | 25 (unchanged, Non-Goal) |
| `detectModulePrefixOverlap` | 20 | 20 (unchanged, Non-Goal) |
| Max in `pipeline_module_phases.go` (new) | — | 13 (`evaluateAndAssemble`) |
| Max in `pipeline_module_cache.go` (new) | — | 7 (`prepareCacheLookup`) |

**Correction to the controller's own numbers, and to the sprint body:** base complexity of `runModuleWithCacheDependencies` is **106, not 103** as claimed in the PR body / design doc ("Five surviving `go:S3776`... 103 against a limit of 15"). The controller flagged this as worth checking; I independently confirm it with a from-scratch `gocognit` run — the 103 figure is wrong by 3. This does not change the substance of the claim (106→0 is still true and still the headline result), but it is a real evidence-integrity defect: a number was carried through four dead iterations, the design doc, and the sprint plan without ever being checked against the actual base file. **Non-blocking**, but should be corrected in the design doc before archiving to `implemented/`.

All new-file function complexities (13 and 7 max) are well under any plausible `go:S3776` limit (15) — the "complexity doesn't merely move into helpers" claim holds.

### 2.3 File overlap with dev-side changes since merge-base — CONFIRMED EMPTY
The 9 PR-changed files and the files dev touched since `d794cac25` do not intersect (control: dev-file-list ∩ itself = 32, proving the check instrument works).

---

## 3. Behaviour preservation (adversarial review)

Diffed both extracted test files against their exact base blobs (`git show d794cac25:...`):

- `internal/pipeline/cache_invalidation_test.go`: 631 → 732 lines. `t.Fatal`/`t.Fatalf` count **81 → 81** (unchanged), `if err != nil` count **19 → 19** (unchanged), `func Test` count **6 → 6** (unchanged). Full unified diff read line-by-line: every extraction is a mechanical hoist of an inline `t.Run` body into a named top-level helper function with an added doc comment; no assertion was dropped, weakened, or turned into a no-op. Confirmed for all of: `TestCacheSource_ExactSnapshot`, `TestCachePipeline_SourceEditBehavior`, `TestCacheArtifacts_Migration`, `TestCachePipeline_WriteFailure`.
- `internal/loader/loader_test.go`: 365 → 377 lines, same pattern (`TestCacheSource_ExactSnapshot` → `assertDiskSourceExactSnapshot` / `assertEmbeddedSourceExactSnapshot`), byte-identical assertion bodies.

**Production orchestration** (`pipeline_module_phases.go` vs. the base's 502-line inline body): read the full base body against the full extracted `run()`/`loadModules()`/`validateAndSort()`/`compileAllAndFinalize()`/`registerResolver()`/`evaluateAndAssemble()` chain line-by-line. Phase order is preserved exactly: load → collision/overlap checks → topo-sort → compile-loop (+ arg-order/take-flatMap/strict-fallback warnings + `RegisterAdtModule` + cache save) → resolver registration → evaluate/assemble. Error propagation is preserved: every early-return in the base returns `result, err`; every phase method now returns bare `err`, surfaced through `run()`'s `return *st.result, err`, so partially-populated results survive on error exactly as before.

One notable **positive finding**: the extraction correctly preserves a genuinely subtle base-code detail — in the base, a verified cache lookup with a `nil` cached payload increments `cacheHits` *before* falling through to a fresh compile (i.e. the hit counter fires even though the module ends up recompiled). The extracted `serveFromCache` reproduces this exact ordering (`st.cacheHits++` then a nil check that returns `false`), with a comment explicitly calling out that the counter increment happens "BEFORE the nil-payload check, preserving the original order." This is not a case an automated diff tool would catch; it required reading the base's control flow. It raised my confidence that the extraction was done adversarially-carefully rather than by pattern-matching.

**Windows skip (§ task E) — legitimate, not sprint-introduced.** The `t.Skip` in `TestPipelineModulePhases_DebugCacheFormsAndCounters` claims the sprint "changed no cache-key or publication code." I verified this directly: the only 3 non-test `.go` files in the diff are `pipeline_module.go`, `pipeline_module_cache.go`, `pipeline_module_phases.go`. `prepareCacheLookup` calls `ModuleCacheKey(version.Commit, *mod.SourceContent, depDigests)` — the same function, same arguments, as the base's inline call. `publishFreshModule` builds the same `CacheEntry` struct the same way. Neither `ModuleCacheKey`'s implementation nor any directory-sanitization/drive-letter-handling code (which is what would actually cause a Windows path bug) lives in any of the 3 changed files. The skip is confirmed pre-existing platform divergence, not something this sprint shipped.

---

## 4. Mutation drill

Ran on the rebased head, each mutation applied via `git`-tracked file edit → `go build ./internal/... ./cmd/ailang/...` → `go test ./internal/pipeline ./internal/loader -count=1` → restored via saved copy → verified byte-identical restore via `sha256sum` (all six matched baseline exactly) → confirmed `git status --porcelain` empty after every restore.

Mutations 1–3 and 5–6 below were designed independently from reading the diff (not from the sprint's claims). Mutation 4 is the one **named row from the sprint plan's own mutation table** (`MUT-IFACE-REGISTER-OMIT`), run as required by Part D against the exact test the plan names.

| # | Mutation | File | Result |
|---|---|---|---|
| 1 | Remove the "verified-but-nil-payload falls through to fresh compile" guard in `serveFromCache` (always serve the cache path even when `cached == nil`) | `pipeline_module_cache.go` | **SURVIVED** — `go test ./internal/pipeline ./internal/loader` passed unmodified (rc=0). This state (verified=true, cached=nil) is never exercised by the test suite. |
| 2 | Invert `if !verified` → `if verified` in `serveFromCache` (swap hit/miss branches) | `pipeline_module_cache.go` | **KILLED** — `TestPipelineModulePhases_DebugCacheFormsAndCounters` fails (cold-summary counter diagnostic missing). |
| 3 | Swallow the elaboration error in `compileFreshModule` (`_ = err` instead of `return err`) | `pipeline_module_phases.go` | **KILLED** — panics with a nil-pointer SIGSEGV in `TestMatchForeignConstructor_NestedInner`. |
| 4 | **MUT-IFACE-REGISTER-OMIT** (sprint plan's own named mutation, row in `m-cachesrc-cognitive-complexity-sprint-plan.md`): skip `st.modLinker.RegisterIface(unit.Iface)` on a cache hit | `pipeline_module_cache.go` | **KILLED** — exactly as the plan's oracle predicts: `TestPipelineModulePhases_MixedVerifiedCachedDependencyAndFreshImporter` fails (`undefined variable: value`), plus two unrelated pre-existing tests also broke (`TestResultMatchArmUnification`), confirming the interface-registration side effect is load-bearing well beyond the one characterization test. |
| 5 | Reorder `run()`'s phase calls: call `registerResolver()` *before* `compileAllAndFinalize()` (so it iterates an empty `compiledUnits` map and registers nothing) | `pipeline_module_phases.go` | **SURVIVED** — full `go test ./internal/pipeline ./internal/loader -count=1` passed (rc=0), including the package's own `ModeEval`-exercising tests (`lambda_open_record_test.go`, `poly_arithmetic_test.go`, `pipeline_module_phases_test.go`). No test in either target package exercises a scenario where resolver registration timing is observable. |
| 6 | Flip the boolean on the nil-`SourceContent` bypass in `prepareCacheLookup`: `return "", false` → `return "", true` (keeps the warning call, only flips the "cacheable" flag) | `pipeline_module_cache.go` | **SURVIVED** — rc=0. The existing assertions (`reads==0`, `writes==0`, `CACHE_SOURCE_UNAVAILABLE` present, manifest unchanged) are all satisfied via the empty-string cache key naturally missing lookup, so nothing distinguishes `cacheable=false` from `cacheable=true` at this call site. |

**Summary: 3 killed / 3 survived** out of my own 6, plus **1/1 killed** on the sprint's own named row.

### Severity assessment of the 3 survivors — all non-blocking
- **#1 (nil-payload guard)**: defensive code for a state the current cache implementation apparently never produces (verified=true with a nil `*CachedModule`). If that state is truly unreachable, the guard is dead-code-safe; if it is reachable in some untested `CacheStore` implementation path, removing it would nil-pointer-panic in production instead of gracefully recompiling. Either way, it is untested, and the sprint's characterization suite does not prove which. Recommend a follow-up test that forces `moduleCache.load` to return `(nil, entry, true)`.
- **#5 (resolver-registration ordering)**: real ordering bug if triggered, but not observable by either target package's own tests, including their `ModeEval` tests — meaning cross-module calls in `ModeEval` are not covered by anything in `internal/pipeline`/`internal/loader`. This is a genuine test-scope gap, not evidence the refactor is wrong (the order in the actual PR is correct and matches base).
- **#6 (nil-source cacheable flag)**: a pure mutation-testing-coverage gap — the boolean return value's positive branch is never independently asserted, only its side effects (which happen to coincide under the mutation). Purely evidentiary, zero behavioral risk since the real code's `false` is correct and produces the documented bypass.

None of the three survivors indicate the *shipped* code does the wrong thing; they indicate three spots the test suite would not catch a *future* regression at. I'm treating these as non-blocking findings for a landing decision, but they are legitimate follow-up items.

---

## 5. Local gates (independently re-run on the rebased, restored-clean tree)

```
go build ./internal/... ./cmd/ailang/...            rc=0
go vet ./...                                        rc=0
gofmt -l internal/pipeline internal/loader           (empty)
go test ./internal/pipeline ./internal/loader -count=1   rc=0
go test -race ./internal/pipeline ./internal/loader -count=1   rc=0
make check-file-sizes                                rc=0
git status --porcelain (post mutation-drill)         (empty)
```
Note: `go build ./...` (unqualified) fails on `cmd/wasm` (`function main is undeclared`) — this is a **pre-existing, unrelated** condition: every file in `cmd/wasm` carries `//go:build js && wasm`, so a normal darwin/arm64 build always fails there regardless of this PR. Excluding that target (as above) reproduces the controller's `go build` rc=0 claim exactly.

---

## 6. What was NOT independently verified (declared per instructions)

- The claim of "all 10 named mutations killed" (sprint plan §M3) — **UNMEASURED for 9 of the 10 rows**. No `.snap/M3/mutations/` evidence tree exists anywhere in this worktree, the git history, or the filesystem (`find` returned nothing; `git log --all -- .snap` returned nothing). I independently ran and confirmed 1 of the 10 named rows (`MUT-IFACE-REGISTER-OMIT`) as a spot-check per task instruction D; it passed as claimed. The other 9 rows (`MUT-SOURCE-EMPTY`, `MUT-SOURCE-REREAD`, `MUT-NIL-PUBLISH`, `MUT-EMPTY-BYPASS`, `MUT-WARM-OFF`, `MUT-CACHED-CORE-OMIT`, `MUT-NOCACHE-PERSIST`, `MUT-SNAPSHOT-LEAK`, `MUT-WARNING-SKIP`) remain hearsay from the dead iterations. My own independently-designed mutation drill (§4) covers substantially overlapping ground (cache-hit inversion, nil-payload fallthrough, interface-registration omission) and found no evidence contradicting the "behavior preserved" claim, but it is not a substitute for running all 10 named rows.
- Exact-revision SonarCloud `go:S3776` re-analysis — **UNMEASURED**, no network/SonarCloud access from this worktree/session; the design doc states this is controller-owned and separate from local gates.
- iteration 342's "PASS 97/100" report — **UNMEASURED**, report file confirmed absent from disk; treated purely as hearsay per the task framing and not relied on for this verdict.
- Full-suite (`go test ./...`) and broader race run — **not run**; scope was bounded to the two directly-affected packages per the task's own gate list, matching what the controller already ran.

---

## 7. Verdict

**LAND.** The core claim — a 106-complexity (not 103; corrected) coordinator reduced to a 2-line wrapper via a behavior-preserving extraction, with zero dropped assertions across two adversarially-diffed test files and a phase-ordering/error-propagation structure that matches the base line-for-line, including a non-obvious counter-ordering edge case — holds up under independent re-derivation, adversarial diffing, and a mutation drill anchored to the actual diff. All local gates (build excl. unrelated wasm target, vet, fmt, test, race, file-size) pass on the rebased tree, restored byte-clean after every mutation. Zero blocking findings.

The four non-blocking findings (wrong base-complexity figure in the design doc; 3 surviving self-designed mutations, all defensive/coverage gaps rather than shipped defects) do not rise to DO-NOT-LAND — they are follow-up hardening items, and I recommend the controller note them in the design doc's evidence section rather than block this iteration's landing on them.

# Sprint Plan: M-TYPEENV-SUB — TypeEnv Substitution Gap (ADT return types lost in exports)

## Summary
Close the type-safety hole where a function whose return type is resolved via constraint
solving involving an imported ADT (Result, Option, Json, …) gets that return type erased to a
type variable in the `TypeEnv` — so the interface builder over-generalizes it to "returns
anything" and downstream call sites silently type-check invalid programs. Ship the **already-decided**
fix (**Option A: alpha-rename schemes on env insertion** + **Approach 4: targeted post-solve
current-binding repair**), un-skip the 3 parked regression tests, and prove no regression across the
type checker, REPL stdlib loader, and example corpus.

**Design doc:** `design_docs/planned/v0_29_0/m-typeenv-sub-fix.md` (decision: Option A + Approach 4, commit 8d72a3b70)
**Duration:** 3 days (est. 2-3 in doc; +buffer for two intervening type-system changes — see Staleness)
**Dependencies:** None (self-contained fix inside `internal/types`)
**Risk Level:** High — modifies shared compilation infrastructure (`internal/types`, `internal/iface`, `internal/pipeline`); every scheme insertion in the language passes through the touched code.
**Total LOC estimate:** ~180 (implementation + new fixture/regression tests; the 3 target tests are already written and only need un-skipping)

## Current Status Analysis

### Bug confirmed live (2026-07-10)
- `internal/pipeline/type_safety_test.go` has **3 tests parked** with `t.Skip("M-TYPEENV-SUB not yet implemented")` at lines 152, 225, 353:
  - `TestTypeSafety_DecodeJsonNotString` — `decode(jo([]))` (Json passed where string expected)
  - `TestTypeSafety_CrossModule_JsonToString` — `needsString(jo([]))` across module boundary
  - `TestTypeSafety_WithinModule_ImportedADTChain` — local `wrap() -> Result[int,string]` used where `string` expected
- **4 sibling tests pass today and MUST NOT regress:** `TestTypeSafety_StringNotJson`, `TestTypeSafety_StringNotJson_Positive`, `TestTypeSafety_CrossModule_StringParam`, `TestTypeSafety_WithinModule_ImportedADTChain_Positive`.

### Root cause (verified against current code)
- `InferWithConstraints` (`internal/types/typechecker_core.go:390`) solves constraints (`ctx.SolveConstraints()` at :443), applies the substitution to `CoreTI` (:468) and the typed node (:471), but **never to the `updatedEnv` it returns** (:492). Exported schemes keep unresolved inference vars.
- `internal/iface/builder.go:295-309` then `Lookup`s each export's type from that env and calls `generalizeType` (:306), quantifying the dangling var → the export appears to "return anything". **This confirms the doc's "interface builder caution" is a real, live path, not hypothetical.**
- The REPL/stdlib path (`internal/repl/module_registry_load.go:258-266`) loads **all** module exports into a **single** env layer via `ExtendScheme`, which is why a naive `env.ApplySubstitution` corrupts unrelated modules' schemes (the capture-avoidance dilemma the doc documents).

### Velocity note
This is a surgical type-system fix (~75 core LOC per the doc's file table), not a feature build. LOC is low but **risk-per-LOC is high**: correctness of every module compile and REPL session rides on it. Estimate is driven by verification breadth (full `make test` + `make verify-examples` + REPL + cross-package alias suites), not typing volume.

---

## Conflict Surface Analysis
*(Required — this sprint modifies `internal/types`, `internal/iface`, `internal/pipeline`; the sprint-evaluator hard-fails without an enumerated Conflict Surface + fixture tests for the "programs that MUST still work".)*

### Semantic positions touched
1. **Scheme insertion into a `TypeEnv`** — `ExtendScheme` / `BindScheme` (`internal/types/env.go:53,76`). M1 alpha-renames the scheme's quantified `TypeVars`/`RowVars` here.
2. **Post-solve env observable state** — the `updatedEnv` returned by `InferWithConstraints` (`typechecker_core.go:492`). M2 repairs the current declaration's binding here, after the outer `SolveConstraints`.

### Other valid constructs that already live in those positions (must survive unchanged)
Both positions are traversed by **every** binding the compiler and REPL ever create. The disambiguation strategy must not disturb any of these ≥3 non-target constructs:
1. **Polymorphic stdlib functions** — e.g. `map : ∀a b. (a -> b, [a]) -> [b]` in `std/list`, `dot_helper` in `std/embedding` (the exact scheme the doc's Approach 1/4 corrupted to `int`). Their legitimately-quantified vars must stay quantified and must NOT be captured by another declaration's solve sub.
2. **ADT constructor factory schemes** — built in `internal/types/constructor_factory.go` (uses Latin `a%d` var names, distinct from inference's Greek `α%d`). These are polymorphic schemes inserted into the env and must be alpha-renamed no-differently.
3. **Open-record / row-polymorphic schemes** — e.g. `getName = \obj. match obj { {name} => name }` returning `∀a ρ. {name: a | ρ} -> a`. Row vars must survive; the repair must exclude `ε`/`ρ` (effect/row) substitutions (this is precisely why the doc's Approach 3 broke `record_patterns.ail`).
4. **Effect-row-polymorphic function schemes** — schemes carrying `ε`-prefixed effect row vars; the M2 filtered sub must skip effect/row vars.
5. **Monomorphic schemes** — no `TypeVars`; alpha-rename MUST be a strict no-op (perf + identity).

### Disambiguation strategy
- **Bug B (capture):** alpha-rename each inserted scheme's quantified vars to globally-unique identities so no two schemes in one env layer share a name. The rename touches only the scheme's own `TypeVars`/`RowVars` and its `Type` body consistently; free (non-quantified) vars are untouched. **Naming caveat (see Staleness #2):** the doc's proposed `q$N$` prefix was chosen to dodge `a{N}` inference vars — but inference now emits Greek `α%d/ε%d/ρ%d/τ%d` and the constructor factory emits Latin `a%d`. The executor must pick a prefix provably disjoint from **both** families (recommend a reserved Unicode/ASCII sentinel prefix + global counter, e.g. `‹q0›`, and add a unit test asserting disjointness), not blindly copy `q$N$`.
- **Bug A (escaping metavars):** after the outer solve, apply the filtered solve sub (excluding effect `ε` and row `ρ` vars) to **only the names introduced by the current declaration** (`*core.Let.Name`, `*core.LetRec.Bindings[].Name`), **without** shielding that declaration's quantified vars (they are the ones that were wrongly quantified). Post-alpha-rename, this sub can no longer collide with any other scheme's binders.

### Programs / suites that MUST still work (fixture coverage)
| Fixture | Location | Protects |
|---------|----------|----------|
| `TestEmbeddedStdlibLoading` | `internal/repl` (or pipeline) | REPL single-layer `ExtendScheme` path; `std/embedding.dot` retains `float` (doc's canonical corruption) |
| `TestCrossPackageTypeAliasUnification` | `internal/types`/pipeline | M-TYPE-ALIAS cross-package record aliases |
| `TestTransitiveTypeAliasPropagation` | pipeline | transitive alias chains |
| `TestTypeSafety_*_Positive` ×2 + `TestTypeSafety_StringNotJson` + `_CrossModule_StringParam` | `internal/pipeline/type_safety_test.go` | the 4 passing type-safety tests |
| `record_patterns.ail`, `adt_list_fields.ail`, `json_array_extraction.ail` | `examples/` via `make verify-examples` | open-record / nested-record-list / JSON-number-array inference (doc's Approach 3 casualties) |
| M-TYPE-LIST-SOUND generalization tests | `internal/types` | `baseEnvFreeVars` withholding (a change that POSTDATES the doc — see Staleness #4) |

### What intentionally changes
Programs that pass an imported-ADT-typed value where a different type is expected **now fail to compile** (that is the fix). The 3 skipped tests flip from skipped→passing. Nothing else should change behavior; any AST/scheme diff on the example corpus that is *not* one of these intended type errors is a regression and must be justified.

---

## Proposed Milestones

### M1: Alpha-rename schemes on env insertion (~65 LOC)
**Goal:** Eliminate the quantified-var name-collision class (Bug B) so post-solve substitution can never capture another scheme's binders.
**Estimated:** ~50 impl (`env.go` global counter + `AlphaRenameScheme` helper, wired into `ExtendScheme`/`BindScheme`) + ~15 test = ~65 LOC
**Dependencies:** None

**Tasks:**
- Day 1: Add a package-level global scheme-var counter to `internal/types/env.go` (`atomic.Uint64`). Implement `AlphaRenameScheme(scheme *Scheme) *Scheme`: allocate globally-unique fresh ids for every `TypeVars`+`RowVars`, build a rename map, rewrite the scheme `Type` consistently; strict no-op when `TypeVars` and `RowVars` are both empty. Pick a rename prefix provably disjoint from inference vars (`α/ε/ρ/τ%d`) AND constructor-factory vars (`a%d`) — add a unit test asserting the chosen prefix collides with neither.
- Day 1: Wire `AlphaRenameScheme` into `ExtendScheme` and `BindScheme`. Verify pretty-printing still uses a display hint (don't leak internal ids into error messages).

**Acceptance Criteria:**
- [ ] `AlphaRenameScheme` is a no-op for monomorphic schemes (unit test asserts identical output)
- [ ] Renamed vars are globally unique and provably disjoint from `α/ε/ρ/τ%d` and `a%d` (unit test)
- [ ] `TestEmbeddedStdlibLoading` passes — `std/embedding.dot`/`dot_helper` keep `float` types (not corrupted to `int`)
- [ ] `make test` green (alpha-rename alone introduces no regression)
- [ ] `make lint` clean

**Risks:**
- Renaming breaks pretty-printing / error messages — Mitigation: keep `Scheme` display via a name hint; rename only internal identity; eyeball a known type-error message.
- Allocation overhead per insertion — Mitigation: no-op fast path for monomorphic schemes (the common case).

### M2: Targeted post-solve current-binding repair (~45 LOC)
**Goal:** Fix Bug A — apply the outer solve substitution to the current declaration's exported binding so the env no longer carries escaping metavars.
**Estimated:** ~25 impl in `typechecker_core.go` + ~20 helper (decl-name extraction + effect/row-filtered sub application) = ~45 LOC; the 3 target tests are already written (un-skip only).
**Dependencies:** M1

**Tasks:**
- Day 1-2: In `InferWithConstraints` (`typechecker_core.go`), after the composed `sub` is available (post-:456) and before returning `updatedEnv` (:492): extract current decl names (`*core.Let` → name; `*core.LetRec` → binding names). For each, look up its scheme in `updatedEnv`, apply the **filtered** sub (exclude `ε`/`ρ` effect+row vars; do NOT shield the scheme's own `TypeVars`), and write the repaired scheme back.
- Day 2: Un-skip the 3 parked tests (remove the `t.Skip` lines at 152, 225, 353). Confirm each produces the expected compile-time type error.

**Acceptance Criteria:**
- [ ] `TestTypeSafety_DecodeJsonNotString` un-skipped and passing (`decode(jo([]))` is a type error mentioning string/Json)
- [ ] `TestTypeSafety_CrossModule_JsonToString` un-skipped and passing (`needsString(jo([]))` rejected across modules)
- [ ] `TestTypeSafety_WithinModule_ImportedADTChain` un-skipped and passing (`wrap() -> Result` used as `string` rejected)
- [ ] The 4 previously-passing type-safety tests remain green (`_StringNotJson`, `_StringNotJson_Positive`, `_CrossModule_StringParam`, `_WithinModule_ImportedADTChain_Positive`)
- [ ] Filtered sub excludes effect (`ε`) and row (`ρ`) vars — open-record/effect-row schemes unchanged
- [ ] `make test` green

**Risks:**
- Repair over-reaches into row/record structure (the Approach 3 failure mode) — Mitigation: exclude `ε`/`ρ` vars; operate on the already-generalized exported binding only, never on body-internal constraints; the M2 acceptance leans on `make verify-examples` in M3.
- `Num`/`Fractional` defaulting interaction — Mitigation: defaulting already runs (`defaultAmbiguitiesTopLevel`, :449) and composes into `sub` before repair; verify numeric example files in M3.

### M3: Interface-builder audit, invariant assertion, full regression (~50 LOC)
**Goal:** Guarantee repaired schemes reach the interface builder unmodified, add a debug-build guard against future escaping metavars, and prove the whole corpus is green.
**Estimated:** ~30 (`assertNoEscapingMetaVars` + debug/test-build wiring) + ~20 (iface guard/audit + any needed regression fixture pins) = ~50 LOC
**Dependencies:** M2

**Tasks:**
- Day 2-3: Audit `internal/iface/builder.go:295-309`: confirm it consumes the repaired env scheme and does not re-run `generalizeType` on a raw type still holding unresolved vars. If it re-generalizes, ensure it re-generalizes the *repaired* scheme (no observable escaping metavar).
- Day 3: Add `assertNoEscapingMetaVars(scheme)` fired after generalization / before interface extraction, active in debug/test builds only — turns silent soundness loss into a hard error during development.
- Day 3: Run `make verify-examples` (full corpus); explicitly confirm `record_patterns.ail`, `adt_list_fields.ail`, `json_array_extraction.ail` still compile. Run cross-package alias + M-TYPE-LIST-SOUND suites.

**Acceptance Criteria:**
- [ ] Interface builder produces concrete ADT return-type schemes (not `∀a. … -> a`) for imported-ADT-returning functions; verified by the now-passing M2 tests
- [ ] `assertNoEscapingMetaVars` present and firing in test builds; no assertion trips across `make test`
- [ ] `make verify-examples` green (whole corpus; do not rely on the doc's stale "152 files" count — treat the target as source of truth)
- [ ] `TestCrossPackageTypeAliasUnification` and `TestTransitiveTypeAliasPropagation` green (M-TYPE-ALIAS not regressed)
- [ ] M-TYPE-LIST-SOUND generalization tests green (the `baseEnvFreeVars` change is not disturbed)

**Risks:**
- Interface builder re-generalizes and re-introduces the var (doc flags this HIGH) — Mitigation: this milestone's whole first task; the M2 tests fail loudly if it does.
- Interaction with M-TYPE-LIST-SOUND `baseEnvFreeVars` withholding (postdates the doc) — Mitigation: run that suite explicitly; if the repair and the withholding disagree, reconcile at the generalization site rather than widening the repair.

### M4: Cleanup, CHANGELOG, doc reconciliation (~20 LOC)
**Goal:** Land the change cleanly and reconcile the design doc's stale references.
**Estimated:** ~20 LOC (deletions + CHANGELOG prose)
**Dependencies:** M3

**Tasks:**
- Day 3: Remove any debug print statements added during implementation. NOTE: the doc's "remove `filterTypeVarSub` if unused" step is stale — that function does not exist in the tree; skip it unless the executor introduces a same-named helper.
- Day 3: Add a CHANGELOG.md entry (type-safety fix; note the 3 tests un-skipped). Correct the design doc's version metadata (it says v0.9.2; current tree is v0.28.0 → v0.29.0) when moving it to `implemented/` — but do NOT move/commit during this sprint; the mission controller handles the doc lifecycle.

**Acceptance Criteria:**
- [ ] No stray debug prints in changed files
- [ ] CHANGELOG.md updated under the correct version, grouped by category
- [ ] `make test` green
- [ ] `make lint` clean

**Risks:**
- Low. Cosmetic/administrative milestone.

---

## Success Metrics
- The 3 target tests un-skipped and passing; the 4 sibling type-safety tests still green.
- `make test` green, `make lint` clean, `make verify-examples` green.
- Interface builder emits concrete ADT return types (no `∀a. … -> a` for ADT-returning exports).
- No regression in REPL stdlib loading, cross-package type aliases, or M-TYPE-LIST-SOUND generalization.
- Documentation: CHANGELOG.md updated; design doc version metadata reconciled (move to `implemented/` deferred to controller).

## Files to Modify
| File | Change | LOC |
|------|--------|-----|
| `internal/types/env.go` | global scheme-var counter + `AlphaRenameScheme`, wired into `ExtendScheme`/`BindScheme` | ~50 |
| `internal/types/typechecker_core.go` | targeted post-solve current-binding repair in `InferWithConstraints` | ~25 |
| `internal/types/*_test.go` | `AlphaRenameScheme` unit tests (no-op + disjointness) | ~15 |
| `internal/iface/builder.go` | audit/guard: consume repaired schemes, don't re-introduce vars | ~10 |
| `internal/types/…` (debug build) | `assertNoEscapingMetaVars` | ~20 |
| `internal/pipeline/type_safety_test.go` | remove 3 `t.Skip` lines (already-written tests) | ~0 |
| `CHANGELOG.md` | fix entry | prose |

## Dependencies
- None external. Milestones are strictly sequential: M1 → M2 → M3 → M4. The executor may run all four in a single session.

## Open Questions
- **Rename-prefix choice** (M1): confirm the sentinel prefix is disjoint from `α/ε/ρ/τ%d` (inference) and `a%d` (constructor factory). Executor decides; unit test must prove it.
- **`assertNoEscapingMetaVars` gating** (M3): debug/test-build only vs. always-on. Recommend test-build only to avoid prod overhead; controller may override.

## Notes — Design-Doc Staleness / Contradictions vs. current tree (2026-07-10)
1. **Version drift.** Doc says `Target: v0.9.2`; the file now lives under `design_docs/planned/v0_29_0/` and the tree is `v0.28.0`. All "for v0.9.2 / post-v0.9.2" phrasing is anachronistic; treat as v0.29.0.
2. **Fresh-var naming is stale.** Doc assumes inference vars are `a{N}/e{N}/r{N}` and proposes an anti-collision prefix `q$N$`. Actual inference emits **Greek** `α%d/ε%d/ρ%d/τ%d` (`internal/types/inference.go:506,522,531,557`); the constructor factory emits **Latin** `a%d` (`constructor_factory.go:95`). The doc's DEBUG snippet showing `[a2 a3 a4]` reflects an older representation. The rename prefix must be re-derived against the real families — folded into M1.
3. **`filterTypeVarSub` does not exist.** The doc's Phase-4 cleanup step "remove `filterTypeVarSub` if unused" references a function absent from the tree. Likewise `TypeEnv.ApplySubstitution` / `ApplySubstitutionToBindings` do not exist yet (they are to be created). Skip the stale deletion.
4. **Two type-system changes landed AFTER the doc and interact with the fix — neither is in the doc's analysis:**
   - **M-TVAR-COLLISION-FIX** added a persistent `inferFreshCounter` on the TypeChecker (`typechecker_core.go:73,404,538`) so declarations within one module no longer restart from 0. This *partially* addresses the doc's stated root-cause #1 (within-module collisions); cross-module and REPL single-layer collisions remain, so Option A is still required — but the within-module `ImportedADTChain` test's mechanism should be re-verified against current behavior rather than assumed verbatim.
   - **M-TYPE-LIST-SOUND round 3** added `baseEnvFreeVars` generalization-withholding (`typechecker_core.go:412`, `generalizeWithConstraints`). This touches the exact generalization site the doc's rejected Approach 3 and its "interface builder caution" discuss. The repair must not regress it (explicit suite run in M3).
5. **"All 152 example files"** — the corpus is larger now; use `make verify-examples` as the source of truth, not the literal count.
6. **Interface-builder caution is confirmed live** (not hypothetical): `internal/iface/builder.go:306` calls `generalizeType` on env-looked-up types, so an unrepaired env directly produces over-generalized schemes. This raises M3's importance.

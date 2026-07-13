# Sprint Plan: M-MODULE-LET-FUNC-RESOLUTION

## Summary

Make module-level `let`/`letrec` values resolve module-scope names exactly as func
bodies do, by promoting module lets to first-class nodes of the existing SCC
call-graph machinery (delete `wrapInLets`), and stop the interim diagnostic from
citing the closed bug #327. Fixes issue [#366](https://github.com/sunholo-data/ailang/issues/366).

**Duration:** 2–3 days
**Dependencies:** None. Design landed via PR #367. Predecessors #327 (v0_29_0), M-BUG-MODULE-LET-SCOPE (v0_4_9).
**Risk Level:** Medium (one **load-bearing** go/no-go spike on the link/eval layer gates the whole approach; documented fallback exists)
**Execution model:** This sprint executes in an **isolated git worktree** by a separate Opus executor agent. All paths below are repo-relative; the executor must NOT assume the main working tree. Do not commit/push from the plan stage — the mission controller handles git.

## PLAN-STAGE REALITY CHECK (re-verified at HEAD, 2026-07-13)

Re-read the code at HEAD before planning. Findings, including **discrepancies** the executor must respect over the design doc's prose:

| Doc claim | Reality at HEAD | Verdict |
|---|---|---|
| `BuildCallGraph(funcs, …)` takes funcs only, line ~111 | `internal/elaborate/scc.go:111` — signature `BuildCallGraph(funcs []*FuncSig, symbols, imports)`; iterates funcs only, edges via `findReferences(f.Body)`. | ✅ EXACT |
| `wrapInLets` at file.go:279–302 wraps every core decl inside module lets | `internal/elaborate/file.go:279–302` elaborates lets once, then `for i, decl := range coreDecls { coreDecls[i] = e.wrapInLets(...) }`; `wrapInLets` def at **file.go:340–359**. Doc said 351 — close (emits plain `core.Let`, confirmed line 351). | ✅ confirmed (def range 340–359) |
| Second re-elaboration loop file.go:316–330 | `internal/elaborate/file.go:316–332` re-elaborates lets to wrap non-func statements. | ✅ confirmed |
| `collectModuleLets` at file.go:130 | `internal/elaborate/file.go:130` calls it; def at file.go:75; type `ModuleLet` at file.go:67. | ✅ (def at 75, doc said 130 = call site) |
| Lying hint at `internal/types/import_hint.go:50` | Exact: `import_hint.go:50` returns the `known bug #327 … bind it with let first` string; func `localResolutionHint` at :43. | ✅ EXACT |
| Hint test asserts "#327" at `local_resolution_hint_test.go:37` | Exact: `internal/types/local_resolution_hint_test.go:37`. Three assertions to update: `known bug #327` (:37), `workaround: bind it with let first` (:40), `defined in this module but not resolvable in this position` (:43). Plus the negative-case check `#327` at :61,:74. | ✅ EXACT |
| **#327 40-cell matrix at `internal/types/record_update_positions_test.go`** | ❌ **DISCREPANCY.** That path does NOT exist. The matrix is at **`internal/pipeline/record_update_positions_test.go`** (position × callee-kind audit). Executor MUST run `go test ./internal/pipeline/ -run RecordUpdate` (not `./internal/types/`). Design-doc Risk row #4 and the regression-fixtures list both name the wrong package. | ⚠ FIXED HERE |
| Regression fixtures exist | `examples/runnable/fnv1a.ail`, `examples/runnable/array_basic.ail`, `examples/deriving_eq.ail`, `examples/runnable/list_sum.ail` — all present. | ✅ |
| No `examples/module_let_helpers.ail` yet | Confirmed absent — new file. | ✅ |
| Link/eval assumes lambda-valued decls | `internal/pipeline/pipeline_module_compile.go:472,480` special-case `Value.(*core.Lambda)` but guard with `ok` (non-lambda safely skipped for param extraction). The REAL spike surface is (a) the **runtime evaluator** binding a top-level `core.Let` whose value is a non-lambda module constant (e.g. `let four = double(2)`), and (b) the core type-checker's strict forward-env threading (`typechecker_core.go CheckCoreProgram`) accepting a non-lambda named module decl. This is exactly the Phase-1 go/no-go. | ✅ spike scoped |

**Duplicate-name error code (proposed):** `MOD007`. Rationale: MOD007–MOD009 are the documented "reserved for future use" slots in `internal/errors/codes.go:82`; MOD004 = "Duplicate export" sits in the same module-namespace family, so MOD007 = "Duplicate module-scope binding (let/func same name)" is the natural adjacent allocation. (MOD014 is already taken — used inline in `pipeline_module_compile.go`/`pipeline_module.go:696`, though not present in the const block.) Executor: register MOD007 in both the const block and the registry map at `codes.go:263+`, and add a footgun fixture per existing MOD014 precedent. Human may veto in review.

## Current Status Analysis

### Velocity
- Repo-wide 7d velocity is huge (286 commits, mission-loop noise) and NOT a useful signal for this scoped fix.
- Comparable prior work: #327 (m-record-update-local-resolution) — the exhaustive-`findReferences` + 40-cell matrix — was a ~2-day elaborate/types fix. This sprint reuses that same machinery, so 2–3 days is calibrated to a real, recent analog, not aspirational.

### Remaining from Design Doc
- Phase 1: unify decl ordering (spike-gated) — ~1–1.5d
- Phase 2: semantics pinning (dup-name error, letrec decision) — ~0.5d
- Phase 3: diagnostics + fixtures + docs — ~0.5–1d

## Proposed Milestones

### Milestone 0 (GATE): Link/eval spike — non-lambda module core decl (go/no-go)
**Goal:** Prove the compile/eval/type-check layer accepts a top-level `core.Let` whose value is a **non-lambda** module constant, BEFORE deleting `wrapInLets`. This is the fallback trigger.
**Estimated:** ~30 LOC throwaway spike (no product code committed yet)
**Duration:** ≤0.5d (front-loaded on Day 1)

**Tasks:**
- Hand-construct (or force via a tiny local patch) a `core.Program` with a top-level non-lambda named decl `let four = double(2)` alongside func decls, and run it through the module compile + eval path.
- Verify: (a) `CheckCoreProgram` binds `four` in forward env for later decls; (b) the evaluator produces the value without assuming `Value` is a `*core.Lambda`; (c) `extractFuncParams` (pipeline_module_compile.go:468) safely skips it (already guarded — confirm).

**Go/No-Go:**
- **GO** → proceed to M1 (unified SCC, delete wrapInLets).
- **NO-GO** (runtime/type layer hard-requires lambda-valued module decls) → switch to the **documented fallback**: KEEP `wrapInLets`, but add module funcs into the let-value elaboration env via a pre-pass env extension (smaller, uglier, recurrence-prone). Record the no-go evidence in the sprint JSON `notes` and in CHANGELOG; the behavior-matrix success criteria stay identical either way.

**Acceptance Criteria:**
- [ ] Spike result recorded (GO or NO-GO) with the exact failing/passing evidence.
- [ ] Chosen path (unified SCC vs fallback) written into `.ailang/state/sprints/sprint_M-MODULE-LET-FUNC-RESOLUTION.json`.

**Risks:**
- Spike says GO but a corner (e.g. letrec-with-let SCC) breaks later — Mitigation: M1 keeps the matrix red→green net running continuously.

### Milestone 1: Unify decl ordering (delete wrapInLets)
**Goal:** Module lets become call-graph nodes; emitted as first-class core decls in topological order interleaved with funcs. Delete `wrapInLets` + both re-elaboration loops.
**Estimated:** ~40 LOC (scc.go) + ~+80/−60 LOC (file.go) impl
**Duration:** ~1–1.5d

**Tasks:**
- `internal/elaborate/scc.go`: extend `BuildCallGraph` to accept module lets (name + value expr) as nodes; add edges let→func, func→let, let→let via the existing exhaustive `findReferences(value)`. Preserve the DEBUG_STRICT exhaustiveness panic.
- `internal/elaborate/file.go`: for each SCC in topo order emit `core.Let` (single, non-recursive) or `core.LetRec` (recursive group) using the same `let NAME = VALUE in VAR(NAME)` shape funcs use today (file.go:200–228 pattern). Remove `wrapInLets` (340–359) and BOTH re-elaboration loops (279–302, 316–332). Route non-func statements (305+) through the same ordering.
- Ensure `DeclMeta`/contract handling (func-keyed) does NOT collide with let decl names.
- Keep the matrix green net running after EACH edit: `go test ./internal/pipeline/ -run RecordUpdate` and `go test ./internal/elaborate/...`.

**Acceptance Criteria:**
- [ ] Matrix repros v3/v4/v7(if supported)/v8 compile green; all previously-green shapes (v1/v2/v5/v6) stay green.
- [ ] `wrapInLets` and both re-elaboration loops deleted; no orphaned references (`grep -rn wrapInLets internal/` → 0).
- [ ] `go test ./internal/elaborate/... ./internal/pipeline/... -count=1` green.
- [ ] Regression fixtures compile: `fnv1a.ail` (func→let), `array_basic.ail` (let→imported func), `deriving_eq.ail` (lets + ADT ctors), `list_sum.ail` (let…in expr form untouched).

**Risks:**
- Evaluation-order change observable — Mitigation: module lets are pure-value positions (no effect rows accepted there); grep first for any effectful module-let acceptance; matrix test pins ordering.
- SCC change destabilizes #327 matrix — Mitigation: run `internal/pipeline/record_update_positions_test.go` explicitly after each edit (NOTE corrected package path).

### Milestone 2: Semantics pinning (dup-name error + letrec decision)
**Goal:** Pin the two unpinned semantics: duplicate module-scope name → compile error; letrec module-level → support or honest diagnostic.
**Estimated:** ~30 LOC impl + ~40 LOC tests
**Duration:** ~0.5d

**Tasks:**
- Duplicate-name gate: detect let-vs-func / let-vs-let same-name collisions during collection; raise **MOD007** with both positions. Register MOD007 in `internal/errors/codes.go` (const block + registry map). Add a footgun fixture (per MOD014 precedent in `internal/diag/footgun_fixtures_test.go`).
- BEFORE enabling as a hard error: `make verify-examples` + grep `examples/` and any ailang-packages for existing programs relying on let/func shadowing. If hits exist → downgrade to warning + follow-up issue (record decision).
- `letrec` decision: attempt `core.LetRec` support for self/mutual-recursive groups that include lets. Time-box to ≤0.5d — if it exceeds, emit an **honest diagnostic** (cite #366, state the `func` workaround) and file a follow-up issue. Record which path was taken.

**Acceptance Criteria:**
- [ ] Duplicate-name (v10 shape) → MOD007 compile error with both positions, test-pinned. (Or documented downgrade-to-warning if verify-examples hits.)
- [ ] `letrec` self-ref (v7 shape): either compiles (LetRec) OR emits an honest diagnostic — NEVER a silent false "undefined variable". Test-pinned.
- [ ] MOD007 registered and appears in the errors registry.

**Risks:**
- Dup-name error breaks an example silently relying on shadowing — Mitigation: verify-examples + grep before enabling; downgrade path documented.

### Milestone 3: Diagnostics, fixtures, docs
**Goal:** Stop the lying hint, ship the behavior-matrix test, new example, docs.
**Estimated:** ~150 LOC test + ~20 LOC example + docs
**Duration:** ~0.5–1d

**Tasks:**
- `internal/types/import_hint.go`: the "not resolvable in this position" path should be unreachable for this class post-fix. If provably unreachable, delete `localResolutionHint` (and its call site + `moduleFuncNames` plumbing if now dead). If a residual hint is kept for future family members, it MUST cite **#366** (not closed #327) and state the verified workaround *"declare it as a `func`"* (verified green at HEAD) — NOT "bind it with let first".
- Update/remove `internal/types/local_resolution_hint_test.go` accordingly (assertions at :37, :40, :43, :61, :74).
- New behavior-matrix test `internal/elaborate/module_let_resolution_test.go` — assert all v1–v10 shapes per the Design Freeze decisions. Must be **non-vacuous**: repros v3/v4/v8 (and v7/v10 per decisions) FAIL at pre-fix HEAD, PASS post-fix. Base-binary non-vacuity check per success criteria.
- New runnable example `examples/module_let_helpers.ail` (~20 LOC, let→func combinator shape) — must appear in `make verify-examples` as a NEW pass.
- Docs: CHANGELOG entry (widening: let→func now compiles; narrowing: duplicate module-scope name now MOD007); `internal/diag/footguns.md` #327 row → retired/#366; errors reference for MOD007.
- Offline eval replay: replay the cleaned 2026-07-13 `higher_order_functions` top-level-let solution shape through the **local ailang binary** (offline `.ail` replay, NO model, NO GPU) — confirm it now type-checks.

**Acceptance Criteria:**
- [ ] `grep -rn "known bug #327" internal/` → 0 hits.
- [ ] Behavior-matrix test green; non-vacuity demonstrated (repros fail at pre-fix HEAD).
- [ ] `examples/module_let_helpers.ail` present and passing.
- [ ] `make verify-examples` baseline unchanged EXCEPT the intended new example pass (185 pass / 5 pre-existing #341 stay; new example adds one pass).
- [ ] CHANGELOG + footguns row + MOD007 errors reference updated.

**Risks:**
- Deleting `localResolutionHint` leaves dead `moduleFuncNames` plumbing — Mitigation: grep call sites; if used elsewhere, keep the field, only retire the hint text.

## Day-by-Day Breakdown

- **Day 1 (AM):** M0 spike (go/no-go) — FIRST, load-bearing. Record decision.
- **Day 1 (PM) – Day 1.5:** M1 — unify SCC, delete wrapInLets, keep matrix net green per edit. Regression fixtures.
- **Day 2 (AM):** M2 — MOD007 dup-name gate + letrec decision (time-boxed).
- **Day 2 (PM) – Day 2.5:** M3 — hint truth pass, behavior-matrix test, new example.
- **Day 2.5–3:** M3 finish — CHANGELOG, footguns, errors reference, offline eval replay, full `go test ./internal/...` + `make verify-examples`.

## Success Metrics
- Behavior-matrix test green (v1–v10 shapes per Design Freeze decisions).
- Base-binary non-vacuity: repros FAIL at pre-fix HEAD, PASS post-fix.
- `grep -rn "known bug #327" internal/` → 0 hits.
- `make verify-examples` baseline unchanged (185 pass / 5 pre-existing #341) + 1 new example pass.
- `go test ./internal/... -count=1` green (incl. `internal/pipeline/record_update_positions_test.go` — the #327 40-cell matrix, corrected package path).
- New runnable example + CHANGELOG + footguns row + MOD007 errors reference shipped.

## Dependencies
- None blocking. Design ratified via PR #367 / issue #366.

## Open Questions / Human-Review Items
1. **MOD007** proposed for the duplicate-name error — confirm allocation (or veto).
2. Duplicate-name = hard error vs warning if verify-examples/packages reveal existing shadowing reliance — executor decides by grep, records; human may override.
3. `letrec` module-level: support (core.LetRec) vs honest diagnostic — executor decides by ≤0.5d spike cost.
4. Whether the residual not-resolvable hint survives at all post-fix, or is deleted as dead code.

## Notes / Assumptions
- **DISCREPANCY carried into this plan:** the #327 40-cell matrix test is at `internal/pipeline/record_update_positions_test.go`, NOT `internal/types/…` as the design doc's regression list and Risk row #4 state. Executor uses the pipeline path.
- The M0 spike is a genuine gate: if NO-GO, the fallback (keep wrapInLets + pre-pass env extension) is chosen and the same behavior-matrix success criteria apply.
- No GPU: the nightly-benchmark "replay" is an offline `.ail` replay through the local `ailang` binary — no model involved.
- Execution happens in an isolated worktree; the controller owns all git. The plan stage commits nothing.

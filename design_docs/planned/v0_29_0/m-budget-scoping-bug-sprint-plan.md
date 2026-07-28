# M-BUDGET-SCOPING-BUG — Sprint Plan

**Design doc**: [m-budget-scoping-bug.md](m-budget-scoping-bug.md)
**Sprint ID**: M-BUDGET-SCOPING-BUG
**Mission iteration**: 98 (2026-07-24)
**Target**: v0.27.x / v0.28.0 (planning under v0.30.0 HEAD)
**Estimated**: 1.5–2 days · ~600 LOC (impl + test migration)
**Risk**: **Medium-High** (two evaluator call-sites, non-`defer` unwind path, test-file rewrite)
**Branch**: dev (work on a feature branch)

---

## Goal

Replace the current **cumulative-across-call-chain** `@limit`/`@min` effect-budget
semantics with **hierarchical per-invocation budget frames** (push on `EnterFunction`
for annotated signatures, pop unwind-safe on every exit, bubbling charge rule). Repro
(cell 1) currently fails at `physical: 4`; must pass. Un-skip `effect_budget_demo.ail`.

---

## Confirmed / corrected edit-site line numbers (doc drift from Oct-2025)

The design doc's line refs have **drifted**. Actual HEAD (v0.30.0) locations:

| Doc claim | Actual location | Note |
|-----------|-----------------|------|
| `budget.go:293` `Merge` "limits summed" | **`budget.go:301`** (`func Merge`); comment at **:291–293** | REPLACED-for-scoping. Only test callers remain (`budget_test.go:215–248`) + unrelated `messaging.Envelope.Merge`. No production caller — safe to delete with its tests. |
| `budget.go:399` `EnterFunction` | **`budget.go:400`** (`func (br *BudgetReport) EnterFunction`) | REUSED (report/observability). **NOTE:** the *evaluator* does NOT call `EnterFunction` today — budget scoping is driven by `WithBudgetLimits`, not `EnterFunction`. See "Critical finding" below. |
| `budget.go:287` `Reset` | **`budget.go:287`** (`func Reset`) ✓ | REPLACED by push/pop. Only caller = `budget_test.go:203` (`TestBudgetContext_Reset`). Delete with its test. |
| `budget.go` "`Consume`" charge interceptor | **`budget.go:231`** `func (bc *BudgetContext) CheckAndConsume(effect, position string) error` | REPLACED — **the load-bearing edit site**. (Doc's `Consume` is named `CheckAndConsume`.) Also `ChargeSemanticOnly` **:259** feeds the cumulative caller-charge and is retired with `PopScopeAndChargeCaller`. |
| `context.go:348` shared `BudgetReport` | **`context.go:348`** (`WithBudget`, `BudgetReport: ctx.BudgetReport`) ✓ | STAYS SHARED. Enforcement state = the `Budget:` field (**:347**) + `WithBudgetLimits` (**:378**) + `PopScopeAndChargeCaller` (**:408**) is where the cumulative bug actually lives. |
| `budget_test.go:255` cumulative assertions | **`budget_test.go:251–268`** (`TestBudgetExhaustedError`, `physical: 10/15`) | MIGRATED. Plus `TestBudgetContext_Merge` :215, `TestBudgetContext_Reset` :198, `TestEffContext_PopScopeAndChargeCaller` :395, nested-scope charge test :474–494. |

### Critical finding — enforcement does NOT go through `EnterFunction`

The doc's mental model ("`EnterFunction` becomes the frame-push hook") is **half right**.
`BudgetReport.EnterFunction` (`budget.go:400`) is only ever called from `report_test.go`
— it is **report/observability only**. The **actual** budget-scoping lifecycle in the
evaluator is:

1. **Push equivalent**: `WithBudgetLimits(fn.EffectBudgets)` (`context.go:378`) →
   `WithBudget` clones a fresh `BudgetContext` and stores `CallerContext` + `DeclaredBudgets`.
   Called at **two** sites:
   - `internal/eval/eval_evaluator.go:299` (`CallFunction`)
   - `internal/eval/eval_operations.go:139` (application path)
2. **Pop equivalent**: manual `e.effContext = oldEffContext` after body eval, preceded by
   `CheckMinimums` + `PopScopeAndChargeCaller` (`ChargeSemanticOnly` → charges caller the
   callee's **declared** limit = the cumulative bug's mechanism).
   - `eval_evaluator.go:338–352`
   - `eval_operations.go:213–227`

**Frame push/pop must hook these two evaluator sites, NOT `EnterFunction`.** The interface
surface (`BudgetEnforcer`/`MinBudgetEnforcer`/`MinimumChecker`/`ScopeCharger`,
`eval_evaluator.go:50–79`) is the seam — keep it, change what the effects-side impls do.

### Unwind-safety risk (HIGH) — the pop is NOT `defer`-guarded

Both evaluator sites restore the effContext with a **manual** assignment at the *bottom*
of the function, **only reached on the normal path**. On error, `evalCore` returns an
`err` that flows to the bottom block (context IS restored there since it runs regardless
of `err`) — BUT the **early returns** on precondition failure (`eval_evaluator.go:315–318`,
`eval_operations.go:169–175`) restore `e.env`/`e.resolver` and `return nil, err`
**WITHOUT** restoring `oldEffContext`. Today that leaks the child budget context on a
precondition-failure path. The frame model makes leaks worse (stale frame → wrong
attribution on sibling calls). **The fix MUST convert scope exit to `defer` (or add
effContext restore to every early-return).** Acceptance criterion "frame pop verified on
error paths + sibling call succeeds" pins this.

---

## Milestone breakdown

### M1 — Frame stack + bubbling charge rule + `@limit` pre-op (~1 day, ~280 LOC)

**Files**: `internal/effects/budget.go`, `internal/effects/context.go`,
`internal/eval/eval_evaluator.go`, `internal/eval/eval_operations.go`

1. Add a per-execution **frame stack** to `BudgetContext` (or a new `BudgetFrameStack`
   owned by `EffContext`; prefer a stack on the shared enforcement state so it survives
   `WithBudget` shallow-copies — the frames are per-execution, the report already shares).
   Each frame = `{limits map[string]int, mins map[string]int, used map[string]int, fnName string}`.
2. `PushFrame(fnName, limits, mins)` on function entry when the signature is annotated;
   `PopFrame()` unwind-safe (**via `defer` at both evaluator sites**).
3. Rewrite `CheckAndConsume` (`budget.go:231`) → **bubbling charge rule**:
   - Physical count still always increments (global report).
   - Iterate **all active frames** constraining the effect **without mutating**.
   - Frame violates iff `used + C > limit`. If ≥1 violates → pick **first violating
     frame innermost-to-outermost**, return error naming that frame (fn + its limit +
     its own used), **increment nothing**, do not perform op.
   - Else **atomically increment `used += C` on every matching active frame**, then op.
4. Hook `WithBudgetLimits`/scope-entry → `PushFrame`; delete the `ChargeSemanticOnly`
   caller-charge in `PopScopeAndChargeCaller` (cumulative mechanism); `PopFrame` on exit.
5. Error message: `NewBudgetExhaustedError` reports the **tripped frame's** own used +
   physical, not the global running total.

**Acceptance**: cells 1, 2, 3, 4, 8 pass (repro succeeds; annotated-caller/child-bypass/
delegation/recursion/dual-violation attribution).

### M2 — `@min` frame-pop semantics + error-path suppression + attribution (~0.5 day, ~120 LOC)

**Files**: `internal/effects/budget.go`, `internal/effects/context.go`, both eval sites

1. `@min` check moves to **frame-pop on NORMAL return only**. `used < N` (own frame's
   count, incl. callee bubbling) → `BudgetMinUnmetError` naming fn/effect/N/actual.
   (Reuse/rename existing `BudgetUnderrunError`; note current `CheckMinimum` uses
   **physical** count — switch to the **frame's** semantic `used` per charging rule.)
2. **Suppress `@min` on error/exceptional exit**: if body `err != nil` (callee raised or
   `@limit` tripped), pop frame but **skip** the `@min` check — original error propagates.
   Wire in the `defer` pop: check err state before running the min-check.
3. Error attribution: callee-frame trip names callee; caller-frame trip names caller
   (cell 7a/7b), failing op not performed (pre-op).

**Acceptance**: cells 5 (+variants), 6, 7 pass.

### M3 — Test migration + demo un-skip + docs (~0.5 day, ~200 LOC test churn)

**Files**: `internal/effects/budget_test.go`, `internal/effects/report_test.go`,
`tools/verify_examples.sh`, `docs/docs/reference/effects.md`, `CHANGELOG.md`

1. **Delete** cumulative machinery + its tests: `TestBudgetContext_Merge` (:215),
   `TestBudgetContext_Reset` (:198), `TestEffContext_PopScopeAndChargeCaller` (:395),
   nested-scope cumulative charge test (:474–494); delete `Merge`/`Reset`/
   `ChargeSemanticOnly` from `budget.go` if no caller remains (coding-standard: understand
   first, then remove with pinning tests).
2. Rewrite `TestBudgetExhaustedError` (:251) to assert the **tripped frame's** own counts.
3. Add the **8 matrix-cell regression tests** (see mapping below) as the new spine —
   prefer end-to-end (`.ail` source → run → assert error/attribution) via the existing
   run harness so the evaluator wiring is exercised, plus unit tests on the frame stack.
4. Un-skip: remove `effect_budget_demo` from `run_skip_reason()` (`verify_examples.sh:40`);
   confirm `ailang run --caps IO --entry main examples/effect_budget_demo.ail` completes.
5. Docs: rewrite `effects.md` "Capability Budgets" — `@limit` = pre-op vs every active
   frame, tightest active wins (innermost-first violator); `@min` = own-frame count at
   normal-exit pop, suppressed on error; unannotated push no frame; recursion → independent
   frames. CHANGELOG entry.

**Acceptance**: all 8 cells green; demo runs; no stale `physical: N` cumulative strings;
error-path frame-pop + sibling-succeeds test green; `make test` + `make verify-examples`.

---

## Semantics-matrix → regression-test mapping (8 cells)

Each cell = one regression test asserting **both outcome and error attribution**.

| Cell | Test name (proposed) | Setup | Assert |
|------|---------------------|-------|--------|
| 1 | `TestBudgetFrame_UnannotatedCaller_AnnotatedCallee_Repro` | `main`(no anno) 3 IO → `limited @limit=3` does 2 | **Succeeds** (frame counts only limited's 2). The literal repro from the doc. |
| 2 | `TestBudgetFrame_AnnotatedCaller_AnnotatedCallee` | `outer @limit=2` 1 IO → `limited @limit=3` 2 IO | **Fails on limited's 2nd op**; charges both frames; outer 1+2=3>2; error names **outer** (tightest active). |
| 3 | `TestBudgetFrame_AnnotatedCaller_UnannotatedCallee` | `outer @limit=3` 1 IO → unannotated `helper` 3 IO | **Fails on helper's 3rd op** (outer 1+3>3); error names **outer**; delegation cannot launder. |
| 4 | `TestBudgetFrame_Recursion` | `rec @limit=3` 2 own IO → `rec(n-1)` depth 3; + control: single invocation 2 ops | **Fails during depth-1** (outermost frame 2+2=4>3 on 2nd invocation's 2nd op); independent frames + bubbling. Single invocation **succeeds**. |
| 5 | `TestBudgetFrame_MinNormalExit` (+2 variants) | `atLeast @min=3` 2 IO → return | **Fails at pop**, `BudgetMinUnmetError` used=2<min=3. Variant A: 3 ops → succeeds. Variant B: 1 own + unannotated callee 2 → succeeds (callee charges frame). |
| 6 | `TestBudgetFrame_MinErrorExit_Suppressed` | `atLeast @min=3` 1 IO → callee raises | **Callee's error propagates**; `@min` suppressed; assert surfaced error is callee's, **not** `BudgetMinUnmetError`. |
| 7 | `TestBudgetFrame_LimitAttribution_CalleeVsCaller` (a/b) | (a) callee `@limit=1` does 2; (b) cell-2/3 shape | (a) error names **callee** + its limit; (b) error names **caller**. Both: failing op **not performed** (pre-op). |
| 8 | `TestBudgetFrame_DualViolation_InnermostAttribution` | `outer @limit=1` 0 IO → `inner @limit=1` already did 1, attempts 2nd | **Fails on inner's 2nd op** (violates both frames); deterministic **innermost-first** → names **inner**; no frame incremented; op not performed. |

Plus a **frame-leak** test (acceptance criterion 6): callee error unwinds through an
annotated frame, then a **sibling** annotated call succeeds — proves `defer` pop, no stale
frame. Name: `TestBudgetFrame_ErrorUnwind_NoStaleFrame`.

---

## Risks

1. **Two evaluator call-sites, not one** (`eval_evaluator.go:292`, `eval_operations.go:132`)
   — identical buggy pattern; SYSTEMIC-FIX both, don't patch one. A shared helper
   (`pushBudgetFrame`/`deferredPopBudgetFrame`) avoids drift.
2. **Non-`defer` unwind (HIGH)** — current pop is manual + skipped on precondition
   early-returns. Must convert to `defer` or restore effContext in every early return, or
   frames leak. Directly gates acceptance criterion 6.
3. **`EnterFunction` is a decoy** — enforcement is `WithBudgetLimits`, not `EnterFunction`
   (report-only). Don't hook the wrong seam.
4. **`CheckMinimum` uses physical, not semantic** (`budget.go:190`) — the frame model needs
   the frame's own `used` (bubbled semantic) count; switching could change existing
   min-budget behavior. Migrate its tests.
5. **Shared vs per-scope state** — `WithBudget` shallow-copies (`context.go:330`) and swaps
   `Budget`. The frame stack must live where it survives that copy (per-execution) yet each
   frame is per-invocation. Decide: stack on `EffContext` (shared ref) holding
   per-invocation frames is cleanest.
6. **End-to-end tests need the run harness** — pure unit tests on `BudgetContext` won't
   exercise the evaluator wiring where the bug lives; include `.ail`-source run tests.

---

## Success metrics

- [ ] 8 matrix-cell tests + frame-leak test green
- [ ] Repro (cell 1) passes; `effect_budget_demo.ail` runs to completion, un-skipped
- [ ] `budget_test.go` migrated (no cumulative `physical: N`); `Merge`/`Reset` machinery + tests deleted
- [ ] `docs/docs/reference/effects.md` rewritten; CHANGELOG updated
- [ ] `make test` + `make verify-examples` + `make check-boundaries` green

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_29_0/m-budget-scoping-bug-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-BUDGET-SCOPING-BUG.json`

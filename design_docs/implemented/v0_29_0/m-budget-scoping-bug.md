## M-BUDGET-SCOPING-BUG: `@limit`/`@min` effect budgets are cumulative across the call chain, not per-function

**Status**: **IMPLEMENTED — iter-98 (2026-07-24), sprint-evaluator PASS 87/100 round 1.** Mark 2026-07-24 ("apply and route") ratified the QUORUM narrow-refinement carve-out's first use and directed applying the 2 reviewer-verbatim refinements. Both applied verbatim: (1) gpt5-6-sol's deterministic first-violating-frame innermost-to-outermost + check-then-atomic-increment-all rule folded into the `@limit` charging section + matrix cell 8; (2) gemini-3-1-pro's charge-interceptor bullet added to the Conflict Surface (confirmed target `CheckAndConsume` → the new frame-stack `Charge` in `budget_frame.go`). Implemented as per-invocation budget frames (`internal/effects/budget_frame.go`), defer-guarded unwind-safe pop at both evaluator sites, bubbling charge rule, `@min` suppressed on error exit. Two latent bugs fixed en route (double-charge; LetRec recursion escaping enforcement). All 8 matrix cells + frame-leak tested; demo un-skipped and runs to completion.

**Known follow-up (D1, non-blocking, from the round-1 evaluation — see [Future work](#future-work-d1)):** the budget-exhausted error message does not yet include the tripped frame's **function name** (only its effect/limit/used), which the charging rule requires for full attribution. `FnName` is stored on the frame but not threaded into `BudgetExhaustedError`/`BudgetUnderrunError`. Diagnostic-quality only — enforcement and soundness are unaffected.
**Target**: v0.27.x / v0.28.0
**Priority**: P2 (Medium — makes per-function budget annotations misleading; surfaced a shipped broken example)
**Estimated**: 1–2 days (hierarchical frame stack + `@min` exit checks + full test migration — see [Estimate rationale](#estimate-rationale))
**Dependencies**: None.

**Found during**: M-SNAKE-FEEDBACK (migrating `effect_budget_demo.ail` off `++`). Verified on v0.26.2.
**Live-repro CONFIRMED at HEAD v0.30.0** (mission iter-95, not a ghost): `limited ! {IO @limit=3}`
making only 2 own IO calls fails at `physical: 4` because `main`'s 3 prints are charged against
`limited`'s budget — the cumulative-across-chain bug is real.

---

## Verdict: the implementation contradicts the annotation's documented intent

A function annotated `! {IO @limit=3}` is documented to mean "this function may perform at
most 3 IO operations" — [docs/docs/reference/effects.md](../../../docs/docs/reference/effects.md)
and the `effect_budget_demo.ail` comments both say so. The implementation instead enforces a
**cumulative** budget over the whole dynamic call chain: a callee's `@limit` is charged the
IO already spent by its callers. A function that makes only 2 of its own IO calls fails if
the caller already made 3.

This is either a real scoping bug (most likely — the annotation is per-function in both the
docs and the user's mental model) or an undocumented design choice that needs to be documented
and the annotation re-described. Either way the current behavior is a trap.

## Minimal reproduction

```ailang
module repro
import std/io (println)

export func limited(x: int) -> () ! {IO @limit=3} {
  println("a");   -- only 2 of limited's own IO calls
  println("b")
}

export func main() -> () ! {IO} {
  println("p1"); println("p2"); println("p3");  -- 3 IO in main (no @limit)
  limited(0)                                      -- limited makes 2 → should be fine
}
```

```
$ ailang run --caps IO --entry main repro.ail
p1
p2
p3
a
Error: execution failed: effect 'IO' budget exhausted: semantic limit=3, used=3 (physical: 4)
```

`limited`'s `@limit=3` is exhausted at **physical=4** = main's 3 prints + `limited`'s first
print. The limit is being applied to the global running total, not to `limited`'s own effects.

This is exactly why `examples/effect_budget_demo.ail` cannot run to completion: its narrative
`main()` preamble (`println("=== Demo ===")` …) spends the budget before the first budgeted
helper is even entered. (Migrated off `++` in M-SNAKE-FEEDBACK; left runtime-broken and
skipped-with-reason in `verify-examples-toplevel`.)

## Investigation pointers

- Budget composition note: "[for nested scopes] limits are summed" — [internal/effects/budget.go:293](../../../internal/effects/budget.go#L293).
- `EnterFunction` attribution — [internal/effects/budget.go:399](../../../internal/effects/budget.go#L399).
- Budget context preserved across scopes — [internal/effects/context.go:348](../../../internal/effects/context.go#L348) (`BudgetReport` shared).
- Existing tests encode "physical: N" cumulative numbers — [internal/effects/budget_test.go:255](../../../internal/effects/budget_test.go) — so a fix must reconcile these (they may have been written to the buggy behavior).

## Normative semantics: hierarchical per-invocation budget frames (DECIDED)

Quorum round 1 rejected the earlier "Option A: fresh per-function frame" draft as
under-specified (it never said whether a callee op charges the caller frame, the callee frame,
both, or neither) and unsound (a *completely* fresh frame would let a child `@limit=5` bypass a
parent `@limit=2`). Both reviewers converged on the same fix; the rules below are that fix,
adopted verbatim as the normative semantics. Option B (keep cumulative, re-document) is
**dropped** — the docs, the demo comments, and the user's mental model all say per-function.

### Frame lifecycle

1. **Push on entry.** On `EnterFunction` for an invocation whose signature carries any
   `@limit`/`@min` annotation, push a new **budget frame** onto a per-execution **frame
   stack**. The frame records `{limits, mins, used}` for that invocation only. Frames are
   **per-invocation, not per-function-name**: each recursive call pushes its own independent
   frame.
2. **Pop on every exit.** The frame is popped on **all** exit paths — normal return, error
   propagation, and any exceptional unwind. Implementation must use a defer/unwind-safe
   mechanism so a callee error cannot leave a stale frame on the stack.
3. **Unannotated functions push no frame.** Their effects charge whatever annotated ancestor
   frames are currently active — budgets therefore compose through un-annotated intermediates
   (an unannotated helper called from an annotated function still spends the caller's budget).

### Charging rule (resolves the "who gets charged" ambiguity)

Every matching effect operation charges the **current (innermost) frame AND bubbles up to
charge ALL currently-active ancestor frames** that constrain that effect. Consequences:

- A callee's effects **do** count toward an annotated caller's budget — a caller cannot be
  bypassed by delegating IO to a child (gemini-3-1-pro's conflict surface).
- An annotated callee is **also** independently bounded by its own frame — the per-function
  meaning the docs promise (gpt5-6-sol's frame rule).
- An op nested under both an annotated caller and an annotated callee charges **both frames**.

### `@limit[effect]=N` — upper bound, checked before each op

Checked **before** each matching effect operation, against **every active frame** on the
stack. **Deterministic pre-op rule (gpt5-6-sol, applied verbatim iter-98):** before an effect
operation of cost C, inspect every active frame constraining that effect **without mutating any
frame**. A frame violates when `used + C > limit`. If one or more frames violate, select the
**first violating frame in innermost-to-outermost stack order**, return an error naming that
frame, **do not increment any frame**, and do not perform the operation. If none violate,
**atomically increment `used` by C in every matching active frame**, then perform the
operation. (This makes "tightest active limit wins" deterministic when one op would exceed
*multiple* active frames — frames have different `used` counts, so the smallest limit is not
necessarily the binding constraint; innermost-to-outermost first-violator selection pins the
attribution.) This is exactly why a child `@limit=5` cannot bypass a parent `@limit=2`: the
parent's frame is still active during the child's dynamic extent, is still charged, and its
check still fires. The error message must name **which frame** tripped (function name + its
limit + its own used count), not a global running total.

### `@min[effect]=N` — lower bound, checked when the frame is popped

`@min` asserts the annotated invocation performed **at least** N matching effects **within its
own frame** (own body + full dynamic extent of its callees, per the charging rule). It is
checked at frame-pop time, at these exact exit points:

- **Normal return**: check fires; `used < N` ⇒ a `BudgetMinUnmetError` naming the function,
  the effect, N, and the actual count.
- **Error / exceptional exit** (a callee raised, a `@limit` tripped, any unwind): the frame is
  still popped, but the `@min` check is **suppressed** — the original error propagates
  unchanged. Rationale: an invocation that failed mid-flight has not "completed with too few
  effects"; layering a `@min` violation over the real error would mask the root cause.
  (Suppression-on-error is itself a matrix cell with a regression test, so this choice is
  pinned deterministically, not left as an accident.)

### Semantics matrix (each cell = one regression test)

Numbers reuse the repro (`main` does 3 own `println`s; `limited` does 2) where possible.

| # | Cell | Setup | Behavior under the new semantics |
|---|------|-------|----------------------------------|
| 1 | Unannotated caller → annotated callee (**the repro**) | `main` (no annotation) does 3 IO, then calls `limited ! {IO @limit=3}` which does 2 IO | **Succeeds.** `main` pushes no frame; `limited`'s frame counts only its own 2 ops (2 ≤ 3). Today this fails at "used=3 (physical: 4)". |
| 2 | Annotated caller → annotated callee | `outer ! {IO @limit=2}` does 1 IO, then calls `limited ! {IO @limit=3}` which does 2 IO | **Fails on `limited`'s 2nd op.** That op charges both frames; `outer`'s frame would hit 3 > 2. Tightest active limit (outer's 2) wins — child `@limit=3` grants no immunity. Error names `outer`. |
| 3 | Annotated caller → unannotated callee | `outer ! {IO @limit=3}` does 1 IO, then calls unannotated `helper` which does 3 IO | **Fails on `helper`'s 3rd op** (outer's frame: 1+3 > 3). No frame for `helper`; its ops bubble to `outer`. Delegation cannot launder effects. |
| 4 | Recursion | `rec(n) ! {IO @limit=3}` does 2 own IO then calls `rec(n-1)` down to depth 3 | **Fails during the depth-1 call.** Each invocation gets an independent frame (2 ≤ 3 locally), but inner ops bubble to ancestor `rec` frames: the outermost frame accumulates 2+2 = 4 > 3 on the second invocation's 2nd op. Frames are per-invocation; bubbling still composes them. A single non-recursive `rec` invocation (2 ops) succeeds. |
| 5 | `@min` on normal exit | `atLeast ! {IO @min=3}` does 2 IO and returns | **Fails at frame pop** with `BudgetMinUnmetError` (`used=2 < min=3`). Variant: 3 ops ⇒ succeeds. Variant: 1 own op + unannotated callee doing 2 ⇒ succeeds (callee ops charge the frame). |
| 6 | `@min` on error exit | `atLeast ! {IO @min=3}` does 1 IO, then a callee raises | **Original callee error propagates**; the frame is popped; the `@min` check is suppressed (no masking). Regression test asserts the surfaced error is the callee's, not `BudgetMinUnmetError`. |
| 7 | `@limit` tripped in callee vs in caller | (a) trip inside annotated callee's own frame (callee `@limit=1`, does 2) vs (b) trip a caller frame from inside a callee (cell 2/3 shape) | **Error attribution differs and is asserted**: (a) names the callee and its limit; (b) names the caller whose frame was exceeded. In both, the failing op is **not performed** (check is pre-op). |
| 8 | Op would exceed **both** caller and callee frames (gpt5-6-sol, iter-98) | `outer ! {IO @limit=1}` does 0 IO, calls `inner ! {IO @limit=1}` which has already done 1 IO and attempts a 2nd | **Fails on `inner`'s 2nd op**, which would violate both `inner`'s frame (1+1 > 1) and `outer`'s frame (1+1 > 1). Deterministic selection picks the **first violating frame innermost-to-outermost** → attribution names **`inner`** (the innermost violator); no frame is incremented; the op is not performed. |

## Conflict surface (what the fix touches, replace vs reuse)

The mandatory inventory of existing machinery that encodes the cumulative behavior:

- **[internal/effects/budget.go:293](../../../internal/effects/budget.go#L293) — `Merge` ("limits
  are summed")**: **REPLACED for scoping purposes.** Summing limits across nested scopes is the
  root of the cumulative semantics and is incompatible with per-invocation frames — under the
  frame model, nested budgets are *never merged into one context*; each stays its own frame and
  the charging rule composes them. `Merge` may survive only for genuinely additive uses (e.g.
  combining two limit *declarations* on the same signature); if no such caller remains, delete
  it (per the coding standard: understand why it's unused first, then remove with the tests
  that pinned it).
- **[internal/effects/budget.go:399](../../../internal/effects/budget.go#L399) —
  `BudgetReport.EnterFunction`**: **REUSED but re-keyed.** Today it attributes usage to a
  `CurrentFunction` *name* with per-name maps — recursion collapses all invocations of a
  function into one row, which cannot represent per-invocation frames. `EnterFunction` becomes
  the frame-push hook; the *report* may keep aggregating per-name for human-readable output,
  but **enforcement** moves to the frame stack (report = observability, frames = semantics).
- **`BudgetContext.Reset()` ([internal/effects/budget.go:287](../../../internal/effects/budget.go#L287))**:
  **REPLACED by frame push/pop.** Its doc comment already says "entering a new function scope
  with per-invocation budget semantics" — but a mutating reset of a shared context both loses
  the parent's counts (breaking bubbling) and cannot restore them on exit. Push/pop subsumes it;
  delete `Reset` once no caller remains.
- **[internal/effects/context.go:348](../../../internal/effects/context.go#L348) — shared
  `BudgetReport` across scopes (M-DX25)**: **STAYS SHARED.** The report is diagnostics
  (`--budget-report` style totals) and sharing it across scopes is correct for whole-run
  totals. What must **stop** being shared-and-cumulative is the *enforcement* state: the
  `Budget` field's clone-and-carry usage in `WithBudgetLimits`/child contexts is where frame
  push must happen instead. The error message's "physical: N" should report the **tripped
  frame's** physical count, not the global one.
- **`BudgetContext.Consume` (or the equivalent effect-intercept/charge hook in
  [internal/effects/budget.go](../../../internal/effects/budget.go)) (gemini-3-1-pro, applied
  verbatim iter-98)**: **REPLACED.** This is the load-bearing edit site — the function that
  actually increments usage and returns the exhaustion error at op time. Currently it increments
  a flat counter and checks a single limit; it must be rewritten to iterate the active frame
  stack, apply the bubbling charging rule (increment `used` on all active ancestor frames), and
  fail pre-op per the deterministic first-violating-frame selection rule above. (The executor
  must first grep `internal/effects/budget.go` to name the exact function — the interceptor is
  where enforcement lives.)
- **[internal/effects/budget_test.go:255](../../../internal/effects/budget_test.go#L255) —
  existing assertions**: **MIGRATED, not preserved.** `TestBudgetExhaustedError` hard-codes
  cumulative `physical: 10/15` strings and other tests assert whole-chain running totals —
  these encode the buggy behavior. Per the testing policy (no backward compat), rewrite them to
  frame semantics: error strings assert the tripped frame's own counts + frame attribution;
  `Merge`/`Reset` tests are deleted alongside the machinery they pin; the semantics-matrix
  tests above become the new spine of the file.

## Acceptance Criteria

- [x] **One regression test per semantics-matrix cell** (8 cells; cells 5 and 7 include their
      listed variants; cell 8 asserts innermost-violator attribution when an op exceeds two
      frames at once), each asserting both outcome and error attribution.
      *(cmd/ailang/budget_scoping_e2e_test.go — binary-driven, all 8 cells + variants green.)*
- [x] The repro above (cell 1) passes: `limited`'s `@limit=3` bounds only `limited`'s own
      frame; `main`'s preamble spends nothing against it.
- [x] `examples/effect_budget_demo.ail` runs to completion under `--caps IO` and is **removed
      from the `verify-examples-toplevel` run-skip list**. *(tools/verify_examples.sh skip
      entry deleted; demo exits 0.)*
- [x] `internal/effects/budget_test.go` migrated per the conflict surface: no stale cumulative
      `physical: N` assertions; `Merge`-summing and `Reset` tests deleted with their machinery.
- [x] `docs/docs/reference/effects.md` "Capability Budgets" describes the hierarchical scoping
      precisely: **`@limit` = checked pre-op against every active frame, tightest active limit
      wins; `@min` = own frame's count checked at frame pop on normal exit, suppressed on error
      exit; unannotated functions push no frame; recursion pushes independent frames.**
- [x] Frame pop is verified on error paths (a test where a callee error unwinds through an
      annotated frame, then a sibling call succeeds — proving no stale frame leaked).
      *(cmd/ailang: TestBudgetFrame_ErrorUnwind_NoStaleFrame; effects:
      TestEffContext_ErrorUnwind_NoStaleFrame — direct unit proof.)*

**Status: IMPLEMENTED (sprint M-BUDGET-SCOPING-BUG, 2026-07-24).**

## Estimate rationale

The original 0.5–1 day assumed "reset a counter on function entry". The ratified semantics are
strictly more work: a per-invocation frame **stack** with unwind-safe pop, the **bubbling**
charging rule across all active frames, `@min` exit-point checks with error-path suppression,
error attribution per frame, and migration of a test file whose assertions were written to the
buggy cumulative behavior. **Honest estimate: 1–2 days** (≈1 day frames + charging + `@limit`,
≈0.5 day `@min` exit semantics + error attribution, ≈0.5 day test migration + demo un-skip +
docs).

## Quorum verification log (mission iter-95)

QUORUM-AT-PICK (doc was pre-quorum, Oct-2025). Reviewers: `gpt5-6-sol` + `gemini-3-1-pro`.

- **Round 1 — BLOCKED** (controller PASS, both reviewers reject). Converged objection: the earlier
  "Option A: fresh per-function frame" was under-specified (who gets charged for a callee op?) and
  unsound (a *completely* fresh child frame could bypass a stricter parent). Both proposed the same
  fix → **hierarchical bubbling**. Fable designer (`claude:claude-fable-5`, quota-bucket, $0
  metered, independent of both reviewers) revised the doc: normative charging rule, `@min` exit
  semantics, 7-cell semantics matrix (each → a regression test), Conflict Surface. Option B dropped.
- **Round 2 — BLOCKED** (controller PASS, both reviewers reject). **Design DIRECTION accepted by
  both** — only two NARROW, reviewer-authored refinements remain (neither disputes hierarchical
  bubbling). Per the gate (one revision + one re-quorum → still-rejected → park needs-human-review),
  the doc is PARKED. The two fixes are recorded here **ready to apply verbatim** (a ~10-minute
  unblock, then route straight to sprint-planner):

  1. **`gpt5-6-sol` — deterministic frame selection on simultaneous violation.** "Tightest active
     limit wins" is ambiguous when one op would exceed *multiple* active frames (frames have
     different `used` counts; smallest limit ≠ binding constraint). **Apply** — replace the pre-op
     rule in [`@limit`](#limiteffectn--upper-bound-checked-before-each-op) with: *"Before an effect
     operation of cost C, inspect every active frame constraining that effect without mutating any
     frame. A frame violates when `used + C > limit`. If one or more frames violate, select the
     **first violating frame in innermost-to-outermost stack order**, return an error naming that
     frame, do not increment any frame, and do not perform the operation. If none violate,
     **atomically increment `used` by C in every matching active frame**, then perform the
     operation."* Add matrix cell 8: an op that would exceed **both** caller and callee frames →
     assert attribution names the innermost violator.
  2. **`gemini-3-1-pro` — Conflict Surface omits the charge interceptor.** The inventory names
     `Merge`/`Reset`/`EnterFunction`/`BudgetReport` but not the function that actually increments
     usage and returns the exhaustion error at op time. **Apply** — add a Conflict Surface bullet:
     ***`BudgetContext.Consume` (or the equivalent effect-intercept/charge hook in
     `internal/effects/budget.go`): REPLACED.*** *Currently increments a flat counter and checks a
     single limit; must be rewritten to iterate the active frame stack, apply the bubbling charging
     rule (increment `used` on all active ancestor frames), and fail pre-op per the deterministic
     selection rule above.* (The executor must first grep `internal/effects/budget.go` to name the
     exact function — the interceptor is the load-bearing edit site.)

**Verdict**: PARK needs-human-review, NOT force-passed (Standing rule 2). Core design is sound and
ratified; the two remaining items are mechanical reviewer-verbatim applications. Mission iter-95
also flagged this as the **2nd instance** (after iter-93/`m-pure-prng`) of "re-quorum blocks solely
on a narrow, obviously-resolvable defect on an otherwise-ratified design" → a mission-control
skill-fix refining the gate to allow a bounded controller-apply-reviewer-verbatim-fix-and-reroute
for exactly this class (pending Mark ratification of first use). metered (both quorum rounds) =
$0.0636.

**Resolution (mission iter-98, 2026-07-24):** Mark ratified the narrow-refinement carve-out's
FIRST USE ("apply and route", commit `4e1348adb`). The controller applied both reviewer-verbatim
fixes into the normative sections (NOT a controller-invented resolution — each is the reviewer's
own quoted text): fix 1 → the `@limit` deterministic pre-op rule + matrix cell 8; fix 2 → the
`BudgetContext.Consume` Conflict Surface bullet. No re-quorum (per Mark + the carve-out — the
design direction was accepted by both reviewers in round 2). Routed to sprint-planner.

## Future work (D1)

**D1 — function-name attribution in budget errors (medium, non-blocking; from the iter-98
round-1 evaluation, score 87/100).** The charging rule specifies that a `@limit`/`@min` error
"must name **which frame** tripped (function name + its limit + its own used count)". The
implementation stores `FnName` on each `BudgetFrame` and selects the correct (innermost)
violating frame, but `BudgetExhaustedError`/`BudgetUnderrunError` have no `FnName` field, so the
surfaced message omits it (`effect 'IO' budget exhausted: semantic limit=2, used=2 …`). The
matrix-cell tests assert the limit/used values (which uniquely identify the frame in every test
scenario) but not the function name, so enforcement and soundness are fully correct — this is a
diagnostic-quality gap only. Fix: add `FnName string` to both error types + their constructors,
thread `f.FnName` through `Charge`/`CheckMin` (`internal/effects/budget_frame.go`), include it in
`Error()`, and extend the e2e assertions to check the name appears in stderr. Also (D2, minor):
`CallFunction` (`eval_evaluator.go`) pushes frames with an empty `fnName` for builtin→AILANG
callbacks — carry a descriptor from the `FunctionValue` so those frames are named too. Small,
self-contained; queue as a diagnostic-polish follow-up.

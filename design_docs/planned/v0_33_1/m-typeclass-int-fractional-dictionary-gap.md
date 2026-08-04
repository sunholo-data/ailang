# M-TYPECLASS-INT-FRACTIONAL-DICTIONARY-GAP: catch missing Fractional[Int] at compile time

**Status**: Planned
**Target**: v0.33.1
**Priority**: P1 — live, reproducible compiler-completeness bug that violates semantic transparency
**Estimated**: 2-4 days (root-cause investigation is Phase 1 and may extend this estimate)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to what a well-typed program computes |
| A2: Replayability | 0 | Not touched |
| A3: Effect Legibility | 0 | Not touched |
| A4: Explicit Authority | 0 | Not touched |
| A5: Bounded Verification | +1 | Moves a currently-runtime-only failure to compile time, where it belongs — directly strengthens `ailang check`'s guarantee |
| A6: Safe Concurrency | 0 | Not touched |
| A7: Machines First | +1 | Replaces an internal implementation-detail error name (`prelude::Fractional::Int::add`) a model cannot act on with a message the codebase already knows how to write (`actionableInstanceHint`) |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | Not touched |
| A10: Composability | 0 | Not touched |
| A11: Structured Failure | +1 | A type error surfaced at compile time is categorically more structured than an uncaught runtime panic-style error |
| A12: System Boundary | 0 | Not touched |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strictly improves machine-actionable diagnostics

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

## Verification Log

Claims in this doc verified against the live codebase (2026-08-04, dev @ v0.33.0):

| Claim | Method | Result |
|-------|--------|--------|
| `Fractional` has a registered instance for `Float` only, never `Int` | `grep -n "Fractional" internal/types/dictionaries.go` | Only `registerFractionalFloat()` exists ([dictionaries.go:405-459](../../../internal/types/dictionaries.go)); no `registerFractionalInt` anywhere in the file |
| A specific, actionable error message ALREADY EXISTS for exactly this class+type combination | Read [internal/types/instances.go:86-87](../../../internal/types/instances.go) (`actionableInstanceHint`) | `case "Fractional": return fmt.Sprintf("Float division (/) needs floats; %s is not Fractional. Convert ints with intToFloat, e.g. intToFloat(x) / intToFloat(y).", ts)` — written under M-AILANG-SEMANTIC-CONTEXT R1b specifically to give models a fix-oriented hint |
| That hint is NOT what fires for the observed failures | Compare `instances.go:61-66` (`Lookup` returns `MissingInstanceError` using the hint) against the actual v0.32.0 stderr (`missing dictionary method: prelude::Fractional::Int::add`) and the code path that produces it | The observed error text matches `internal/eval/eval_patterns.go:344` (`evalDictRef`, a RUNTIME dictionary-resolution path), not `instances.go`'s compile-time `Lookup`/`MissingInstanceError` path — the two are different code, and only the unhelpful one fired |
| `evalDictRef`'s missing-method branch assumes type-checking already proved the instance exists | Read [internal/eval/eval_patterns.go:325-344](../../../internal/eval/eval_patterns.go) | Comment explicitly states: *"the type checker already proved this class instance exists for the type"* — this is a safety-net path, not the primary check, and its own assumption is violated by this bug |
| **Live reproduction on current dev build**: `ailang check` passes, `ailang run` crashes | Ran `ailang check --relax-modules` then `ailang run --entry main --caps IO --relax-modules` against a minimal repro (`let weight_kg = 70; let height_squared = height_m * height_m; let bmi_raw = weight_kg / height_squared;` where `height_m = 1.75`) | `ailang check`: **"✓ No errors found!"** `ailang run`: **`Error: execution failed: missing dictionary method: prelude::Fractional::Int::add`** — reproduced live, not just in archived eval data, confirming the bug is present on dev @ v0.33.0, not something already fixed since the v0.32.0 release |
| Frequency/pattern in real data | `jq` scan of `eval_results/baselines/v0.32.0/standard/*_ailang_*.json` for `stderr` containing `"Fractional::Int::add"` | 5 occurrences, all the SAME benchmark (`explicit_dataflow_ssa`, core tier), 5 different models — consistent with one systematic gap, not 5 independent model errors |
| The likely trigger shape | Read the actual failing code (`weight_kg = 70` [Int literal] `/ height_squared` where `height_squared = height_m * height_m` and `height_m = 1.75` [Float]) | The pattern is a `let`-bound Int literal later used in a `/` expression against a Float-derived value — mixed Int/Float division through a `let` binding, not a same-line literal | 
| Duplicate design docs | `create_planned_doc.sh` auto-search (SimHash + neural) | Top matches (`gpt5-reference-code.md`, `M-GAME-B-effects-for-games.md`, `m-cloud-job-reliability.md`, `m-arch-boundaries-eval-exclusion-tighten.md`, `m-eval-data-hosting-decouple.md`, `m-decision-entropy-monitor.md`) are unrelated keyword-coincidence hits, none about typeclass dictionary resolution or numeric defaulting. Not a duplicate. |

## Problem Statement

AILANG's type system resolves arithmetic operators via typeclass dictionaries (`Num`, `Fractional`,
`Ord`, `Eq`), and the codebase already has a well-written, fix-oriented error message for exactly
the case "you divided something that isn't `Fractional`" (`actionableInstanceHint`, written under
a prior milestone specifically to help models self-correct). But at least one program shape —
a `let`-bound `Int` literal later divided against a `Float`-derived value — slips past whatever
compile-time check would normally trigger that hint, and only fails at RUNTIME, with a raw
internal dictionary-lookup error (`missing dictionary method: prelude::Fractional::Int::add`) that
names an implementation detail (a `prelude` namespace path) no model or human can act on without
reading the compiler's own source.

**Current State:**
- Live-reproduced on the current dev build: `ailang check` reports zero errors on a program that
  then crashes at `ailang run` with this exact message.
- 5 occurrences in v0.32.0 standard-mode results, all on the same benchmark
  (`explicit_dataflow_ssa`), 5 different models — a systematic gap in a specific program shape,
  not scattered model confusion.
- Violates AILANG's own semantic-transparency intent: a type error is escaping to runtime instead
  of being caught where the language's design says it should be.

**Impact:**
- Any AILANG program mixing `Int` and `Float` values through `/` via intermediate `let` bindings
  risks this failure, regardless of model — this is a language-completeness gap, not a
  model-training gap.
- Standard-mode eval results miscategorize this as `runtime_error`, when the correct instrument
  category (per `.claude/rules/eval.md`'s guidance to read `error_category` first) should really
  distinguish "the type checker missed a real type error" from "the program's logic is wrong."

## Goals

**Primary Goal:** A program that divides an `Int`-typed value by (or against) a value requiring
`Fractional` fails at compile time with the existing `actionableInstanceHint` message, never at
runtime with a raw dictionary-lookup error.

**Success Metrics:**
- The live repro from the Verification Log (`ailang check` on the minimal `weight_kg / height_squared`
  case) reports a `Fractional`-hint type error, not "No errors found."
- `ailang run` on the same program is never reached — `ailang check` rejects it first.
- All 5 real v0.32.0 `explicit_dataflow_ssa` failures, re-run through the fixed type checker,
  fail at `ailang check` with the actionable hint instead of at runtime.
- Zero new false-positive type errors on the existing `std/` and `examples/` corpus (this is a
  tightening of an existing check, not a new one — must not reject currently-valid Float/Int
  programs that use explicit `intToFloat` conversions correctly).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Root cause is NOT YET diagnosed to the exact type-inference mechanism (see Non-Goals/Phase 1) — this doc commits to investigating in Phase 1 before committing to a specific fix mechanism in Phase 2 | Fixing the wrong layer (e.g. patching the runtime error path instead of the type-checker gap) would leave the real hole open while looking closed | human (reviews Phase 1 findings before Phase 2 starts) | design → revisit after Phase 1 | high |
| Whether the fix ALSO adds a safety-net improvement to `evalDictRef`'s runtime error path (route through `actionableInstanceHint`-style formatting even when type-checking should have caught it) as defense-in-depth | Cheap, and directly prevents this exact failure MODE (unhelpful internal error) from recurring for any future type-checker gap of this shape, even one not yet discovered | human | design | low |

### Design Freeze

- [ ] Phase 1 (root-cause investigation) must complete and be reviewed before Phase 2's specific
      fix mechanism is finalized — this doc intentionally does not pre-commit to "the fix is X"
      beyond "it belongs in the type checker's handling of `/` and `let`-bound numeric literals"

## Conflict Surface

*(Required — this design touches `internal/types/`, specifically numeric-literal defaulting and
typeclass constraint resolution for `/`.)*

### Syntactic positions touched

- The `/` (division) operator's type-inference rule, wherever it currently constrains its operands
  to a `Fractional` instance (likely in the operator-resolution/elaboration pass — exact site to
  be located in Phase 1, candidates include `internal/types/defaulting.go` given its existing
  `"Fractional": TFloat` default-for-literals entry at line 30, and wherever binary-operator
  constraint generation happens in the constraint solver).
- Numeric-literal defaulting for `let`-bound values that are used in a `Fractional`-constrained
  position LATER in the same scope, not at the literal's own binding site.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| `/` between two `Float`-typed operands | Valid today | `1.75 / 2.0` — resolves to `Fractional[Float]`, works |
| `/` between two literal numbers with no other constraint | Valid today (defaults to Float per `defaulting.go:30`) | `10 / 3` where neither operand is otherwise typed — defaults to `Fractional`'s `TFloat` default |
| `/` between an explicitly-`Int`-typed value and a `Float`-typed value | **This bug** — should be a compile-time `Fractional` error (via `actionableInstanceHint`), is currently a runtime crash | `let weight_kg = 70; ... weight_kg / height_squared` where `height_squared` is Float-derived |
| Explicit conversion via `intToFloat` | Valid today, the DOCUMENTED correct pattern (the hint text itself recommends it) | `intToFloat(weight_kg) / height_squared` |

The key open question for Phase 1 is WHY the third row (an `Int`-typed `let` binding used later in
a `Fractional` position) doesn't trigger the same constraint-solving path that correctly rejects
other `Fractional[Int]` misuses elsewhere in the codebase (the existence of `actionableInstanceHint`'s
`Fractional` case proves SOME code path does catch this class correctly — the gap is why THIS
shape specifically doesn't reach it).

### Disambiguation strategy

Not yet determined — this is precisely what Phase 1 must establish: whether the constraint solver
defaults `weight_kg`'s type to `Int` at its `let`-binding site (before the later `/` usage is
analyzed) due to let-generalization/monomorphization ordering, and if so, whether the fix is to
defer defaulting until all uses in the scope are known, or to have the `/` operator's constraint
generation retroactively unify the operand's existing binding rather than accepting an
already-defaulted `Int`.

### Programs that MUST still work

- `1.75 / 2.0` (Float ÷ Float) — must keep working.
- `10 / 3` (bare literals, defaults to Float per existing `defaulting.go` behavior) — must keep
  working and keep defaulting to Float, not regress to requiring explicit annotation.
- `intToFloat(x) / y` (explicit conversion) — must keep working, this is the CORRECT pattern the
  hint text recommends; the fix must not make legitimate conversions harder.
- Any existing `std/`/`examples/` program performing division — full regression run required
  before this ships (Success Criteria).

### What deliberately changes

- A program with a `let`-bound `Int` value later used in a `Fractional`-constrained `/` expression,
  WITHOUT an explicit conversion, changes from "compiles, then crashes at runtime" to "fails at
  `ailang check` with the actionable hint." This is strictly a diagnostic-timing improvement — no
  program that previously RAN SUCCESSFULLY changes behavior (a program that hit this bug never
  produced a correct result; it crashed either way).

## Solution Design

### Overview

**Phase 1 is investigation, not implementation**: reproduce the minimal case, trace exactly where
the type checker currently decides `weight_kg`'s type and why the `/`-with-`height_squared`
constraint doesn't retroactively catch the mismatch, using the same standard the codebase already
holds design docs to (read the actual constraint-solving code, don't guess). Only once that trace
is complete does Phase 2 commit to a specific fix.

### Architecture

**Components (Phase 1 investigation targets, not yet confirmed as the fix location):**
1. **`internal/types/defaulting.go`** — owns the `"Fractional": TFloat` literal-defaulting rule;
   candidate site for why a `let`-bound literal defaults before its later `Fractional` usage is seen.
2. **Constraint generation for binary operators** (exact file TBD in Phase 1) — where `/`
   generates its `Fractional` constraint on both operands; candidate site for retroactive
   unification against an already-bound variable's type.
3. **`internal/types/instances.go`'s `Lookup`** — the compile-time path that DOES correctly use
   `actionableInstanceHint`; understanding why it's bypassed for this program shape is central to
   Phase 1.

### Implementation Plan

**Phase 1: Root-cause the type-inference gap** (~1-2 days)
- [ ] Reduce the live repro further (already minimal: 4 `let` bindings) — try variants (literal
      inline vs. `let`-bound, same-expression vs. cross-statement) to isolate exactly which
      structural feature causes the check to be skipped
- [ ] Trace the constraint-solving path for the failing variant with a debug build / verbose
      type-checker tracing (check `.claude/rules/dev-workflow.md` debug flags — `DEBUG_STRICT=1`
      may surface where this silently degrades instead of failing loudly)
- [ ] Confirm, by reading the actual constraint-generation code (not inferring from behavior),
      the exact point where `weight_kg`'s type is fixed to `Int` and why the later `/` constraint
      doesn't reopen it
- [ ] Write up findings as an addendum to this doc's Verification Log before Phase 2 starts

**Phase 2: Fix the compile-time gap** (~1-2 days, scope depends on Phase 1 findings)
- [ ] Implement the fix at the location Phase 1 identifies
- [ ] Confirm the live repro now fails at `ailang check` with the `actionableInstanceHint` message
- [ ] Run full regression: `make test`, `make verify-examples`, and specifically every division
      operation in `std/` and `examples/`

**Phase 3 (optional, cheap defense-in-depth): safety-net formatting for `evalDictRef`** (~0.5 day)
- [ ] If `evalDictRef`'s runtime missing-method path is still reachable after Phase 2 (i.e. some
      OTHER type-checker gap could theoretically hit it), route its error through
      `actionableInstanceHint`-equivalent formatting when the class/type combination matches a
      known instance hint, so any FUTURE gap of this shape degrades to a helpful message instead
      of a raw internal path

### Files to Modify/Create

**Modified files:**
- `internal/types/defaulting.go` and/or the binary-operator constraint-generation file identified
  in Phase 1 - fix location TBD, ~15-50 LOC depending on Phase 1 findings
- `internal/eval/eval_patterns.go` - optional Phase 3 safety-net formatting, ~15 LOC
- `internal/types/*_test.go` - regression tests for the Conflict Surface's "Programs that MUST
  still work" list, ~40 LOC

## Examples

### Example 1: Before/after `ailang check`

**Before:**
```
$ ailang check bmi.ail
✓ No errors found!
$ ailang run --entry main --caps IO bmi.ail
Error: execution failed: missing dictionary method: prelude::Fractional::Int::add
```

**After:**
```
$ ailang check bmi.ail
Error: type error in bmi (decl 0): Float division (/) needs floats; int is not Fractional.
Convert ints with intToFloat, e.g. intToFloat(x) / intToFloat(y).
```

## Success Criteria

- [ ] Live repro (`weight_kg`/`height_squared` minimal case) fails at `ailang check` with the
      `actionableInstanceHint` message, not "No errors found"
- [ ] All 5 real `explicit_dataflow_ssa` v0.32.0 failures, re-run, fail at compile time with the
      hint instead of at runtime
- [ ] Zero regressions across `make test`, `make verify-examples`
- [ ] Every "Programs that MUST still work" fixture from the Conflict Surface passes unchanged
- [ ] Documentation/CHANGELOG note explaining the tightened check (so a model hitting this for the
      first time post-fix understands it's a real, intentional type error, not a new bug)

## Testing Strategy

**Unit tests:**
- Minimal repro (the exact live-verified case) as a permanent regression test asserting compile-time rejection
- Each Conflict Surface "must still work" row as its own pinned test

**Integration tests:**
- Full `make test` / `make verify-examples` run — this is a tightening of an existing check, so
  the main risk is false positives on currently-valid code

**Manual testing:**
- Re-run `explicit_dataflow_ssa` through `ailang check` for at least 2 of the 5 originally-failing
  models' actual generated code (not just the synthetic minimal repro)

## Deferred Decisions

- Exact fix location and mechanism — explicitly deferred to Phase 1 findings, not decided in this
  doc (see High-Impact Decisions)
- Whether Phase 3 (safety-net formatting) ships in this same change or as a fast-follow — agent
  may choose, low risk either way

## Non-Goals

- **Committing to a specific fix mechanism before Phase 1 completes** - this doc deliberately
  stops at "investigate, then fix," per the Design Freeze, rather than guessing at the
  constraint-solver internals from the outside
- **Auditing other typeclasses (`Num`, `Ord`, `Eq`) for the same class of gap** - the Verification
  Log's evidence is specific to `Fractional`/`Int`/`/`; a broader audit is future work if this
  pattern is confirmed to generalize
- **Changing what `Fractional` means or adding a `Fractional[Int]` instance** - the existing
  design intent (per the hint text itself, "Convert ints with intToFloat") is that `Int` should
  NOT be `Fractional`; this doc fixes WHEN that's enforced, not WHETHER it should be

## Timeline

**Days 1-2**: Phase 1 (investigation)
**Days 3-4**: Phase 2 (fix + regression)
**Half day**: Phase 3 (optional)

**Total: 2-4 days (Phase 1 findings may extend this)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Phase 1 finds the gap is structurally deep (e.g. tied to monomorphization ordering) and the "real" fix is much larger than 1-2 days | Medium | Phase 1 is explicitly scoped as investigation-first with a human review gate before Phase 2 commits to an approach — if the fix is large, this doc's Design Freeze forces a re-scope conversation instead of silent scope creep |
| Tightening the check introduces false positives on currently-valid Float/Int mixed code | High if it happens | Full `make test`/`make verify-examples` regression required in Success Criteria; Conflict Surface enumerates the known-valid shapes as pinned tests before shipping |
| The fix only covers the exact repro shape and misses a sibling shape (e.g. the same gap via `*` or `-` instead of `/`) | Medium | Phase 1's repro-reduction step explicitly includes trying variant operators, not just `/`, before declaring the root cause understood |

## Related Documents

**Implemented (informs this design):**
- M-AILANG-SEMANTIC-CONTEXT (referenced in `instances.go:70` comment) — built
  `actionableInstanceHint`, the exact message this bug should be surfacing but currently isn't;
  worth locating and reading its design doc during Phase 1 for context on why the hint exists and
  what it was originally verified against

**Planned (no overlap found):**
- None of the top SimHash/neural matches (see Verification Log) discuss typeclass dictionaries,
  numeric defaulting, or division semantics.

## References

- [Design Axioms](/docs/references/axioms)
- `internal/types/instances.go`, `internal/types/dictionaries.go`, `internal/types/defaulting.go`,
  `internal/eval/eval_patterns.go`

## Future Work

- If Phase 1 reveals the gap generalizes beyond `/`+`Fractional`, a broader audit of
  typeclass-constraint enforcement for `let`-bound literals across `Num`/`Ord`/`Eq` may be
  warranted as a follow-up milestone.

---

**Document created**: 2026-08-04
**Last updated**: 2026-08-04

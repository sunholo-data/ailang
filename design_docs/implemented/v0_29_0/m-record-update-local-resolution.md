# M-RECORD-UPDATE-LOCAL-RESOLUTION: Local functions in record-update fields resolve as "undefined variable" (#327)

**Status**: Planned
**Target**: v0.29.0
**Priority**: P1 (blocks the natural `{ s | f: helper(...) }` idiom in package code; misleading diagnostic; #323-family resolution divergence)
**Estimated**: 2–3 days (root-cause + fix 1–1.5d, position audit + fixtures 1d)
**Dependencies**: None. GitHub: #327.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Restores local reasoning: an in-scope name is in scope in every expression position |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | The natural idiom fails with a FALSE diagnostic ("undefined variable" for a defined function) — models cannot recover; mid-tier burns its repair round |
| A8: Minimal Syntax | +1 | Removes a position-dependent exception to uniform scoping |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Record update composes with local helpers like every other expression |
| A11: Structured Failure | +1 | Even pre-fix, the diagnostic must stop lying (see m-diagnostic-coverage) |
| A12: System Boundary | 0 | No change |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 / A3 / A4 / A7: no violations

## Problem Statement

Found dogfooding M-DEONTIC-PKG (ailang-packages). In a package module importing
record-alias types + ADT constructors from a sibling (`import ./types (...)`), a
module-local function called **inside a record-update field** fails to type-check:

```
Error: type error in engine (decl 4): undefined variable: extendFm at engine.ail:59:44
```

for `ForceMajeure(_, d1, d2) => ({ s | dls: extendFm(s.dls, s.delivered, d1, d2) })`
where `extendFm` is a plain local `func` in the same module. Verified behavior matrix
(all live on v0.28.0, mutation-bisected):

| Formulation | Result |
|---|---|
| `{ s \| f: importedFn(...) }` | ✅ resolves |
| `{ f: localFn(...), ...all other fields }` (record LITERAL) | ✅ resolves |
| `let x = localFn(...); { s \| f: x }` (hoisted) | ✅ resolves |
| `{ s \| f: localFn(...) }` | ❌ "undefined variable" |
| — definition before or after the call site | ❌ both orders fail |
| — `pure func` vs `func`, renamed, params changed | ❌ all fail |

Fixing one site reveals the same error at the NEXT local-call site — every local
function in this position is affected in the failing module.

**Not-yet-minimal caveat (honest):** clean-room repros (single module; and a two-file
package with imported record alias + imported ADT constructors + recursive local fn +
imported fn inside the local fn) all PASS. The trigger needs something more present in
the real module — candidates: a record alias whose FIELD is itself an imported ADT
(`term: Term`), the second sibling import, or interaction with the module's other
declarations. Root-causing starts from the preserved failing artifact.

**Failing artifacts (preserved):**
- ailang-packages `packages/deontic/engine.ail` at commit `7755d25^` (pre-workaround); reproduce with `AILANG_RELAX_MODULES=1 ailang check engine.ail`.
- A ~60-line reduction (runEvents + extendFm + two-arm applyEvent, imports intact) recorded in #327.

**Systemic framing:** this is the third member of the "resolution diverges by
syntactic position" family: #323 (unresolved uppercase pattern idents silently became
catch-all variables), the nth-in-recursion misdiagnosis that led to it, and now
record-update field expressions. The elaborator evidently constructs scope/env
differently per position. The fix must be preceded by an AUDIT, not a spot patch.

## Goals

**Primary Goal:** an in-scope module-local function resolves in a record-update field
exactly as in any other expression position.

**Success Metrics:**
- The preserved deontic engine (pre-workaround shape) checks green.
- Position-audit fixture matrix green: local fn / imported fn / constructor / lambda in each of: record-literal field, record-update field, match arm body, match guard, if condition, let RHS, list element, tuple element, function argument.
- Deontic package workaround removed and package still green (byte-identical wave5 demo).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Audit-first: fixture matrix over ALL expression positions BEFORE the fix | Prevents the fourth member of this bug family | agent executes, human ratifies scope | sprint start | low |
| Interim diagnostic if fix slips a release: "defined in module but not resolvable in this position (known bug #327): hoist to let" | Stops the diagnostic lying to agents | human | design | low |

### Design Freeze
- [ ] Mark confirms audit-first sequencing (fixtures before fix)

## Conflict Surface

Touches `internal/elaborate/` (and possibly `internal/types/` env plumbing) — mandatory section:

1. **Positions extended:** record-update field expressions' name resolution. No grammar change.
2. **What already lives there:** record-update desugaring (`{ r | f: e }` → core form), the M-CROSS-MODULE record-alias TypeName preservation, #323's uppercase-pattern elaboration (same file family). The fix must not disturb: imported-fn resolution in the same position (works today), record-literal elaboration, alias expansion (M-XMOD-ALIAS/-CHAIN fixpoint logic).
3. **Disambiguation:** none grammatical — this is environment construction during elaboration of the update's field expressions.
4. **Programs that must still work (fixtures):** every existing record-update usage in stdlib/examples (`{ r | ... }` sweep via grep, spot fixtures from std/ + examples/); deontic engine post-workaround (hoisted lets must ALSO still work); the wave-1 txn benchmark reference (heavy `{ s | ... }` user).
5. **Deliberate changes:** none besides making the broken case work.

## Solution Design

1. **Fixture matrix first** (~150 LOC table-driven pipeline tests): callee kind × expression position, from the audit list above. Expect exactly one red cell (update-field × local-fn) — if MORE cells are red, the scope of the fix grows and each new red cell gets a fixture.
2. **Root-cause from the preserved artifact**: bisect the deontic module downward (not clean-room upward, which failed) — remove declarations/imports/fields until the flip is found; that delta names the mechanism.
3. **Fix in the elaborator's env construction** for update-field expressions; the mechanism determines whether it's a lookup-order, binding-group, or alias-interaction fix.
4. **Retire the workaround**: un-hoist deontic engine.ail upstream (ailang-packages PR), keep one hoisted fixture to prove both forms coexist.

### Files to Modify
| File | Change |
|---|---|
| `internal/elaborate/*.go` | env construction for record-update field elaboration (exact file per root-cause) |
| `internal/pipeline/record_update_positions_test.go` | new fixture matrix |
| ailang-packages `packages/deontic/engine.ail` | un-hoist (follow-up PR, after fix ships) |

## Success Criteria
- [ ] Pre-workaround deontic engine checks green
- [ ] Fixture matrix: all positions × all callee kinds green
- [ ] Existing record-update tests + M-XMOD-ALIAS tests unchanged
- [ ] CHANGELOG entry; #327 closed with root-cause note

## Verification Log

| Claim | Method | Result |
|---|---|---|
| Local fn in update field fails | live `ailang check`, deontic engine, v0.28.0 | "undefined variable: extendFm" |
| Imported fn in same position works | same module, aSet in Deliver arm | no error at that site |
| Record literal + hoisted forms work | mutation bisect (6 mutations run live) | both flip to next error site |
| Order-independent | definition moved above call site, re-checked | still fails |
| Clean-room repros pass | 4 standalone/2-file attempts, all live | all green (caveat recorded) |

## Non-Goals
- Redesigning record-update syntax or semantics.
- Fixing the OTHER family members here (#323 shipped; pattern-position audit was its scope) — but the fixture matrix from this doc becomes the shared regression net.

## Related Documents
- [#327](https://github.com/sunholo-data/ailang/issues/327) — the bug, with matrix + artifacts
- [m-bug-record-update-inference (implemented v0.4.8)](../../implemented/v0_4_8/m-bug-record-update-inference.md) — prior record-update bug (inference); neural 0.41, distinct: that was typing, this is name resolution
- [m-diagnostic-coverage](m-diagnostic-coverage.md) — the interim lying-diagnostic entry lives there
- changelog `#323` entry (v0.18-current [Unreleased]) — the family's first member

---
**Document created**: 2026-07-09

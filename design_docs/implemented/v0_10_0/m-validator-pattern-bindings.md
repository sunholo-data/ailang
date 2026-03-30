# M-VALIDATOR-PATTERN-BINDINGS: Validator Rejects Pattern Match Bindings

**Status**: Planned
**Target**: v0.9.13
**Priority**: P0 (High — blocks package publishing)
**Estimated**: 2 hours
**Dependencies**: M-PKG-INTERREF (parent feature, commit 9cdca384)
**Milestone ID**: M-VALIDATOR-PATTERN-BINDINGS
**Created**: 2026-03-26
**Source**: Agent message `88e6d95a` (docparse — sunholo/billing_service_api@0.3.0)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes false-negative: valid code rejected nondeterministically depending on pattern usage |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Makes `check --package` accurate — no false rejections |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | AI agents publishing packages get correct validation results |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Packages using list patterns can compose correctly |
| A11: Structured Failure | +1 | Eliminates spurious "undefined variable" errors |
| A12: System Boundary | 0 | No change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): This is a determinism FIX
- [x] A3 (Effects): No effect changes
- [x] A4 (Authority): No capability changes
- [x] A7 (Machines First): Directly improves machine reliability

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

This is a gap in the `collectPatternVars` function introduced in commit 9cdca384. The function handles 4 of 7 core pattern types completely but has one incomplete case. Full audit:

| Core Pattern Type | Binds Names? | Handled? | Status |
|------------------|-------------|----------|--------|
| `VarPattern` | Yes | ✅ Complete | `locals[p.Name] = true` |
| `LitPattern` | No | ✅ N/A | No bindings to collect |
| `WildcardPattern` | No | ✅ N/A | No bindings to collect |
| `ConstructorPattern` | Via args | ✅ Complete | Recurses into `p.Args` |
| `TuplePattern` | Via elements | ✅ Complete | Recurses into `p.Elements` |
| `RecordPattern` | Via fields | ✅ Complete | Recurses into `p.Fields` |
| **`ListPattern`** | **Via elements AND tail** | **⚠️ Incomplete** | **Recurses `p.Elements` but ignores `p.Tail`** |

**This is the only gap.** The fix is surgical — add tail handling to the existing ListPattern case.

Also audit `walkForVars` for completeness — it handles `Match` arms correctly (calls `collectPatternVars`) but should be verified for `Lambda` parameter scope.

---

## Problem Statement

### The Bug

The `check --package` validator's `collectPatternVars` function (introduced in 9cdca384) does not collect variable bindings from `ListPattern.Tail`. When a cons pattern like `x :: rest` is elaborated to Core AST, it becomes:

```go
&core.ListPattern{
    Elements: []core.CorePattern{&core.VarPattern{Name: "x"}},
    Tail:     &core.VarPattern{Name: "rest"},  // ← IGNORED
}
```

The validator adds `x` to scope but not `rest`, then reports `rest` as an undefined reference.

**Current error:**
```
ERROR: function 'collectRequired' references 'rest' which is not yet defined
```

**Current State:**
- `sunholo/billing_service_api@0.3.0` cannot be published
- Any package using cons patterns (`x :: rest =>`) or spread patterns (`[x, ...rest]`) fails validation
- Same code published fine as v0.2.2 (before inter-function ref checking was added)

**Impact:**
- Blocks dependency upgrades (billing_service_api can't be republished against firestore@0.7.0)
- Any package with list recursion patterns is affected

---

## Goals

**Primary Goal:** Fix `collectPatternVars` to handle all pattern binding sources, eliminating false "undefined reference" errors.

**Success Metrics:**
- `ailang check --package .` passes on packages with cons patterns
- `sunholo/billing_service_api@0.3.0` can be published
- No regression in actual undefined-reference detection

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add `Tail` handling to `collectPatternVars` | Only correct fix — tail bindings must be in scope | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fix approach: add `p.Tail` handling (no alternatives — this is the only correct fix)

---

## Solution Design

### Overview

Add `ListPattern.Tail` handling to `collectPatternVars` in `cmd/ailang/check_package.go`. This is a 3-line fix with a test.

### Architecture

The fix is within the existing `collectPatternVars` switch statement. No architectural changes needed.

### Implementation Plan

**Phase 1: Fix** (~30 min)
- [x] Add `p.Tail` handling to `collectPatternVars` ListPattern case
- [x] Verify `walkForVars` Match arm handling is complete (it already calls `collectPatternVars`)

**Phase 2: Test** (~1 hour)
- [x] Add Go unit tests for `collectPatternVars` and `findUnresolvedVars` with cons patterns
- [x] Run existing test suite to verify no regressions
- [x] Run `make lint` — clean

**Phase 3: Verify** (~30 min)
- [x] Update CHANGELOG.md
- [x] Ack the agent message

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/check_package.go` — Add 3 lines to `collectPatternVars` ListPattern case (~line 592)

**New files:**
- `tests/check_package/cons_pattern_pkg.ail` — Test file for validator with cons patterns

---

## Examples

### Example 1: Cons pattern in package function

**Before (fails):**
```
$ ailang check --package .
ERROR: function 'collectRequired' references 'rest' which is not yet defined
```

**After (passes):**
```
$ ailang check --package .
✓ Package check passed
```

### Example 2: The pattern that triggers the bug

```ailang
module sunholo/billing_service_api/helpers

func collectRequired(items: [Item]) -> [string] {
  match items {
    [] => [],
    x :: rest => if x.required {
      [x.name, ...collectRequired(rest)]   // 'rest' is from cons pattern
    } else {
      collectRequired(rest)
    }
  }
}
```

---

## Success Criteria

- [ ] `collectPatternVars` handles `ListPattern.Tail`
- [ ] Test file with cons pattern passes `check --package`
- [ ] `make test` passes (no regressions)
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests:**
- Add `.ail` file using `x :: rest =>` pattern in a function
- Run `ailang check --package` on it — must pass
- Run `ailang check --package` on a file with genuinely undefined refs — must still fail

**Integration tests:**
- `make test` — full test suite
- `make verify-examples` — example file verification

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Test file location and exact content — agent may choose
- Whether to add explicit `LitPattern`/`WildcardPattern` no-op cases for completeness — agent may choose

## Non-Goals

- Refactoring the entire `collectPatternVars` function
- Adding new pattern types
- Changing the inter-function reference checking architecture

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix is too narrow (other patterns affected) | Med | Systemic audit above shows ListPattern.Tail is the only gap |
| Removing a real error | Low | Test that genuinely undefined refs are still caught |

## Related Documents

- [design_docs/planned/v0_9_2/m-pkg-interref-fix.md](design_docs/planned/v0_9_2/m-pkg-interref-fix.md) — Parent feature that introduced the regression
- [design_docs/implemented/v0_3_14/list_pattern_spread.md](design_docs/implemented/v0_3_14/list_pattern_spread.md) — Original list spread pattern implementation

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Commit 9cdca384 — "Fix package inter-function references and enhance check --package"
- Agent message `88e6d95a` — Original bug report from docparse

---

**Document created**: 2026-03-26
**Last updated**: 2026-03-26

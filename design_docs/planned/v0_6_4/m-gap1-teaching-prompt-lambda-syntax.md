# M-GAP1: Fix Teaching Prompt Lambda Syntax

## Status
- **Status:** Planned
- **Target:** v0.6.4
- **Priority:** P1 (High)
- **Estimated:** 1 hour
- **Dependencies:** None

## Problem Statement

The teaching prompt (`prompts/v0.6.5.md`) contains incorrect documentation for lambda syntax with multi-parameter functions like `foldl`.

**Current teaching prompt shows:**
```ailang
-- WRONG (from teaching prompt)
let sum = foldl(\(acc, x). acc + x, 0, [1,2,3,4,5])
```

**Actual working syntax:**
```ailang
-- CORRECT (from existing examples)
let sum = foldl(\acc x. acc + x, 0, [1,2,3,4,5])
```

### Impact
- AIs using the teaching prompt generate incorrect code
- Leads to confusing "arity mismatch" errors
- Discovered during dogfooding (porting event_formatter.go to AILANG)

## Goals

**Primary Goal:** Ensure teaching prompt accurately documents lambda syntax

**Success Metrics:**
- Teaching prompt matches actual parser behavior
- AI-generated code using the prompt compiles correctly
- No syntax discrepancies between docs and implementation

## Solution Design

### Overview

Update the teaching prompt to use the correct lambda syntax: `\param1 param2. body` instead of `\(param1, param2). body`.

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `prompts/v0.6.5.md` | Fix lambda syntax examples | ~10 |
| `prompts/versions.json` | Bump to v0.6.6 if needed | ~2 |

### Implementation

1. **Search for incorrect syntax in prompt:**
   ```bash
   grep -n '\\\\(' prompts/v0.6.5.md
   ```

2. **Replace all instances of:**
   - `\(acc, x).` → `\acc x.`
   - `\(a, b).` → `\a b.`
   - Any `\(params).` pattern → `\params.` pattern

3. **Verify against working examples:**
   - Check `examples/runnable/no_loops_fold.ail`
   - Check `examples/runnable/lambdas.ail`

### Correct Lambda Syntax Reference

```ailang
-- Single parameter
let inc = \x. x + 1

-- Multiple parameters (space-separated, NOT comma-separated)
let add = \x y. x + y

-- With foldl
let sum = foldl(\acc x. acc + x, 0, [1,2,3,4,5])
let max = foldl(\acc x. if x > acc then x else acc, 0, xs)

-- Curried form (also valid)
let add = \x. \y. x + y
```

## Testing

- [ ] Run `ailang prompt` and verify examples compile
- [ ] Test AI code generation with updated prompt
- [ ] Verify all `examples/runnable/*.ail` files still pass

## Success Criteria

- [ ] No `\(param, param).` patterns in teaching prompt
- [ ] All lambda examples in prompt use `\param param.` syntax
- [ ] Teaching prompt version bumped appropriately

## Timeline

**Day 1:** Complete all changes (~1 hour)
- Search and replace incorrect syntax
- Test prompt examples
- Commit and verify

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | Correct docs improve AI code generation |
| A8: Syntax Is Liability | 0 | Documentation fix, no syntax change |

**Net Score:** +1 (Accept)

## Related Documents

- [prompts/v0.6.5.md](../../../prompts/v0.6.5.md) - Current teaching prompt
- [examples/runnable/no_loops_fold.ail](../../../examples/runnable/no_loops_fold.ail) - Working lambda examples

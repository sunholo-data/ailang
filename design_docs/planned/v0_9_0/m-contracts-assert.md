# M-CONTRACTS-ASSERT: Contract-Based Preconditions

**Status**: Planned
**Target**: v0.8.0
**Priority**: P2 (Medium) - Enables Axiom A5 compliance
**Estimated**: 4-5 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Contracts are deterministic checks |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | **Primary goal** - enables local precondition checks |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Contracts machine-verifiable |
| A8: Minimal Syntax | -1 | New `@assert` syntax (minimal impact) |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Contracts compose with functions |
| A11: Structured Failure | +1 | Contract violations are typed errors |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Contracts are pure boolean expressions
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Contracts are machine-checkable

## Problem Statement

AILANG's Axiom A5 (Bounded Verification) states: "Verification should be local, bounded, and automatable."

**Current State:**
- Type checking provides static guarantees
- No mechanism for runtime preconditions/postconditions
- AI agents cannot express domain constraints (e.g., "x > 0")
- **Axiom A5 score: 1/2 (partial)**

**Impact:**
- Domain errors caught late (at runtime) with poor diagnostics
- AI-generated code cannot express safety contracts
- Functions cannot declare invariants

## Goals

**Primary Goal:** Add `@assert` preconditions that are checked at function entry.

**Success Metrics:**
- `@assert(condition)` syntax on function parameters
- Contract violations produce typed ContractError
- Contracts visible in type signatures for AI analysis
- Static analysis can reason about contracts
- **Axiom A5 score improved to 2/2 (strong)**

## Solution Design

### Overview

Add optional `@assert(expr)` annotations to function parameters that are checked at runtime on function entry. Contracts appear in type signatures and can be analyzed statically.

### Architecture

**Components:**
1. **Contract Parser**: Parse `@assert(condition)` annotations
2. **Contract Elaborator**: Lower annotations to runtime checks
3. **Contract Checker**: Insert checks in compiled code
4. **ContractError**: New error type for violations

### Syntax

```ailang
-- Simple precondition
let divide = \x. \y @assert(y != 0). x / y

-- Multiple conditions
let sqrt = \x @assert(x >= 0). _math_sqrt(x)

-- With custom message
let index = \xs. \i @assert(i >= 0, "index must be non-negative").
  @assert(i < length(xs), "index out of bounds").
  _list_get(xs, i)
```

### Type Representation

Contracts are represented in the type system:

```
divide : int -> int@{!= 0} -> int
sqrt : float@{>= 0} -> float
```

### Implementation Plan

**Phase 1: Parser Extension** (~8 hours)
- [ ] Add `@assert` token to lexer
- [ ] Parse `@assert(condition)` after parameter
- [ ] Store contracts in AST node
- [ ] Unit tests for parsing

**Phase 2: Elaboration** (~8 hours)
- [ ] Elaborate contracts to Core representation
- [ ] Add ContractCheck Core node
- [ ] Type check contract expressions (must be bool)
- [ ] Handle contract in type inference

**Phase 3: Runtime Checks** (~8 hours)
- [ ] Insert runtime checks at function entry
- [ ] Create ContractError error type
- [ ] Include position and condition in error
- [ ] Propagate contract info through compilation

**Phase 4: Integration** (~6 hours)
- [ ] Add contracts to type pretty-printing
- [ ] Document syntax and semantics
- [ ] Create examples
- [ ] Performance testing (check overhead)

### Files to Modify/Create

**New files:**
- `internal/contract/contract.go` - Contract representation (~150 LOC)
- `internal/contract/check.go` - Runtime check insertion (~200 LOC)

**Modified files:**
- `internal/lexer/token.go` - Add ASSERT token (~5 LOC)
- `internal/lexer/lexer.go` - Scan `@assert` (~20 LOC)
- `internal/parser/parser.go` - Parse contracts (~100 LOC)
- `internal/ast/ast.go` - Add Contract field to Lambda (~20 LOC)
- `internal/elaborate/elaborate.go` - Lower contracts (~50 LOC)
- `internal/core/core.go` - Add ContractCheck node (~30 LOC)
- `internal/eval/eval.go` - Evaluate contract checks (~40 LOC)
- `internal/errors/errors.go` - Add ContractError (~30 LOC)

## Examples

### Example 1: Division by Zero Prevention

**Before:**
```ailang
let divide = \x. \y. x / y
-- Runtime panic on divide(10, 0)
```

**After:**
```ailang
let divide = \x. \y @assert(y != 0). x / y
-- ContractError: Precondition failed: y != 0 at line 1, col 23
```

### Example 2: Array Bounds Checking

```ailang
let safeGet = \xs. \i @assert(i >= 0). @assert(i < length(xs)).
  _list_get(xs, i)

-- Usage
let result = safeGet([1,2,3], 5)
-- ContractError: Precondition failed: i < length(xs) at line 1, col 37
```

### Example 3: Domain Constraints

```ailang
let factorial = \n @assert(n >= 0, "factorial requires non-negative").
  if n <= 1 then 1 else n * factorial(n - 1)

-- Type signature shows constraint
-- factorial : int@{>= 0} -> int
```

## Success Criteria

- [ ] `@assert(condition)` parses correctly
- [ ] Contract violations produce ContractError
- [ ] Contracts visible in type signatures
- [ ] Performance overhead < 5% for typical programs
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added to examples/

## Testing Strategy

**Unit tests:**
- Parser handles `@assert` syntax
- Elaborator creates correct Core nodes
- Type checker validates contract expressions

**Integration tests:**
- End-to-end contract checking
- Error messages include position and condition
- Nested contracts work correctly

**Manual testing:**
- Performance benchmark with/without contracts
- Error message clarity

## Non-Goals

**Not in this feature:**
- Postconditions (`@ensures`) - Deferred to v0.9.0
- Loop invariants - Requires different mechanism
- Static contract verification (SMT) - Separate feature
- Contract inheritance in ADTs - Future work

## Timeline

**Week 1** (16 hours):
- Phase 1: Parser extension
- Phase 2: Elaboration

**Week 2** (14 hours):
- Phase 3: Runtime checks
- Phase 4: Integration
- Documentation

**Total: ~30 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance overhead | Medium | Make contracts optional, optimize hot paths |
| Syntax conflicts | Low | `@` prefix distinguishes from other constructs |
| Contract expression complexity | Medium | Limit to pure expressions |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_18/m-dx4-coretypeinfo-completeness.md](design_docs/implemented/v0_3_18/m-dx4-coretypeinfo-completeness.md) - Type info invariant

**Planned (check for overlap):**
- [design_docs/planned/v0_7_0/m-error-propagation.md](design_docs/planned/v0_7_0/m-error-propagation.md) - Error handling (contracts produce errors)

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A5: Bounded Verification
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - KPI tracking

## Future Work

- Postconditions (`@ensures(condition)`)
- Class invariants for ADTs
- Static contract verification with SMT solver
- Contract inference from usage patterns

---

**Document created**: 2025-12-19
**Last updated**: 2025-12-19

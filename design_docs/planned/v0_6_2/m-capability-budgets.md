# M-CAPABILITY-BUDGETS: Resource-Bounded Effects

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Medium) - Enables Axiom A9 compliance
**Estimated**: 3-4 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Budgets are deterministic limits |
| A2: Replayability | +1 | Budget consumption is traced |
| A3: Effect Legibility | +1 | Budgets make effect costs explicit |
| A4: Explicit Authority | +1 | Budgets constrain capability scope |
| A5: Bounded Verification | +1 | Budget exhaustion verifiable locally |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Budgets machine-checkable |
| A8: Minimal Syntax | 0 | Minor syntax addition |
| A9: Cost Visibility | +1 | **Primary goal** - costs in type signatures |
| A10: Composability | +1 | Budgets compose (sum of parts) |
| A11: Structured Failure | +1 | Budget exhaustion is typed error |
| A12: System Boundary | +1 | Budgets enforce boundary constraints |

**Net Score: +10** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Budget checks are pure
- [x] A3 (Effects): Budget consumption explicit
- [x] A4 (Authority): Budgets constrain, don't grant authority
- [x] A7 (Machines First): Budgets machine-verifiable

## Problem Statement

AILANG's Axiom A9 (Cost Visibility) states: "Resource costs (time, memory, API calls, etc.) should be visible in types where possible."

**Current State:**
- Effects are declared but unbounded (IO can do anything)
- No way to limit resource consumption
- AI agents cannot reason about resource costs
- **Axiom A9 score: 1/2 (partial)**

**Impact:**
- Cannot express "this function makes at most N API calls"
- AI cost estimation is guesswork
- Resource-intensive operations not visible in types

## Goals

**Primary Goal:** Add capability budgets that limit effect usage at the type level.

**Success Metrics:**
- `! {IO @limit=N}` syntax for bounded effects
- Budget exhaustion produces typed BudgetExhaustedError
- Budgets visible in type signatures
- Budgets compose correctly (nested = sum of limits)
- **Axiom A9 score improved to 2/2 (strong)**

## Solution Design

### Overview

Extend the effect system to support optional budget annotations. Budgets limit how many times an effect can be used within a scope.

### Syntax

```ailang
-- Function with bounded IO
let fetchN = \n. ! {IO @limit=n}
  List.map(fetch, List.take(n, urls))

-- Type signature
-- fetchN : int -> [string] ! {IO @limit=n}

-- Compile-time known budget
let fetchThree = \(). ! {IO @limit=3}
  [fetch(url1), fetch(url2), fetch(url3)]

-- Multiple effect budgets
let process = \x. ! {IO @limit=5, FS @limit=2}
  let data = readFile("input.txt") in
  let result = compute(data) in
  writeFile("output.txt", result);
  result
```

### Budget Semantics: What Is Counted?

**Critical clarification (from design review):**

> **Rule:** Type-level budgets count **effect invocations**, not semantic cost.

| Budget Kind | Counts | Layer | Example |
|-------------|--------|-------|---------|
| `IO @limit=N` | Number of IO effect invocations | Type-level | 3 print calls = 3 uses |
| `Net @limit=N` | Number of Net effect invocations | Type-level | 2 HTTP requests = 2 uses |
| `api_calls` | Semantic API calls (may differ) | Runtime/spec | Batched API = 1 semantic call |
| `tokens` | Token consumption | Runtime/spec | Sum of input + output tokens |
| `cost_usd` | Monetary cost | Runtime/spec | Aggregated provider costs |

**Why this distinction matters:**

Type-level budgets are **syntactic guarantees** - the compiler can count effect sites statically. Runtime/spec budgets track **semantic resources** that may not map 1:1 to syntax.

```ailang
-- Type sees: 1 Net invocation
let fetchBatch = \urls. ! {Net @limit=1}
  batchFetch(urls)  -- Internally makes N HTTP calls

-- Spec sees: N api_calls (semantic)
-- envelope.api_calls: 10  -- This is the semantic limit
```

**Internal naming note:** Think `@uses` not `@limit` - it counts how many times the effect is *used*, not a rate limit.

### Budget Scope: Per-Invocation, Not Global

**Budgets apply per function invocation, not globally across the program.**

```ailang
let fetch3 = \(). ! {Net @limit=3}
  fetch(url1); fetch(url2); fetch(url3)

-- Each call to fetch3 has its own budget of 3:
fetch3()  -- Uses 3/3, OK
fetch3()  -- Fresh budget: uses 3/3, OK
fetch3()  -- Fresh budget: uses 3/3, OK
-- Total: 9 Net effects, but each invocation is bounded
```

This design choice enables:
- **Compositional reasoning** - function budgets are self-contained
- **Static verification** - budget exhaustion checkable per function
- **No hidden state** - no global counter to reason about

### Budget Composition

```ailang
-- Budget consumption adds up
let outer = \(). ! {IO @limit=10}
  inner1();  -- consumes 3
  inner2();  -- consumes 4
  inner3()   -- consumes 3
  -- Total: 10, OK

-- Budget exhaustion
let exhausted = \(). ! {IO @limit=2}
  fetch(url1);  -- 1
  fetch(url2);  -- 2
  fetch(url3)   -- ERROR: BudgetExhaustedError
```

### Type System Integration

Budgets are tracked in effect types:

```
-- Effect with budget
! {IO @limit=5}

-- Effect without budget (unlimited)
! IO

-- Multiple effects with mixed budgets
! {IO @limit=5, FS}  -- IO limited, FS unlimited
```

### Implementation Plan

**Phase 1: Syntax & Parsing** (~6 hours)
- [ ] Extend effect syntax to include `@limit=N`
- [ ] Parse budget expressions (must be int)
- [ ] Store budgets in Effect AST nodes
- [ ] Unit tests for parsing

**Phase 2: Type Checking** (~10 hours)
- [ ] Add budget field to effect types
- [ ] Implement budget composition rules
- [ ] Check budget at each effect use site
- [ ] Track remaining budget through scope

**Phase 3: Runtime Enforcement** (~8 hours)
- [ ] Add BudgetExhaustedError error type
- [ ] Track budget consumption at runtime
- [ ] Insert budget check before each effect
- [ ] Support dynamic budgets (computed at runtime)

**Phase 4: Integration** (~6 hours)
- [ ] Update effect pretty-printing
- [ ] Add to capability system
- [ ] Documentation and examples
- [ ] Performance testing

### Files to Modify/Create

**New files:**
- `internal/budget/budget.go` - Budget tracking (~150 LOC)
- `internal/budget/compose.go` - Composition rules (~100 LOC)

**Modified files:**
- `internal/types/effects.go` - Add budget field (~30 LOC)
- `internal/parser/effects.go` - Parse `@limit=N` (~40 LOC)
- `internal/types/typecheck.go` - Budget checking (~80 LOC)
- `internal/effects/effects.go` - Runtime budget tracking (~60 LOC)
- `internal/errors/errors.go` - Add BudgetExhaustedError (~20 LOC)
- `internal/eval/eval.go` - Insert budget checks (~40 LOC)

## Examples

### Example 1: Bounded API Calls

**Before:**
```ailang
-- No way to express "at most 3 API calls"
let fetchData = \urls. ! IO
  List.map(fetch, urls)  -- Could be 1000 calls!
```

**After:**
```ailang
let fetchBounded = \urls. ! {IO @limit=10}
  List.map(fetch, List.take(10, urls))
  -- Type guarantees: at most 10 IO operations
```

### Example 2: Nested Budget Composition

```ailang
let processFiles = \files. ! {FS @limit=20}
  -- Each iteration uses 2 FS ops (read + write)
  List.forEach(files, \f. ! {FS @limit=2}
    let content = readFile(f) in
    writeFile(f ++ ".bak", content)
  )
  -- With 10 files: 10 * 2 = 20 FS ops (exactly at limit)
```

### Example 3: Dynamic Budget

```ailang
let fetchWithBudget = \n. \urls. ! {IO @limit=n}
  List.map(fetch, List.take(n, urls))

-- Type: int -> [string] -> [Response] ! {IO @limit=n}
-- Budget is parameterized by n
```

### Example 4: Budget Exhaustion Error

```ailang
let overBudget = \(). ! {IO @limit=2}
  fetch("url1");
  fetch("url2");
  fetch("url3")  -- BudgetExhaustedError: IO budget (2) exhausted

-- Error includes:
-- - Effect type (IO)
-- - Budget limit (2)
-- - Current consumption (3)
-- - Position of violation
```

## Success Criteria

- [ ] `@limit=N` syntax parses correctly
- [ ] Budgets appear in type signatures
- [ ] Budget exhaustion produces typed error
- [ ] Budgets compose correctly in nested scopes
- [ ] Dynamic budgets work (parameterized limits)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Parser handles `@limit=N` syntax
- Type checker tracks budgets
- Composition rules work correctly

**Integration tests:**
- End-to-end budget enforcement
- Nested scope budget tracking
- Dynamic budget values

**Manual testing:**
- Error message clarity
- Performance overhead measurement

## Non-Goals

**Not in this feature:**
- Memory budgets - Requires different mechanism (GC integration)
- Time budgets - Need wall-clock tracking
- Automatic budget inference - Future work
- Budget transfer between effects - Out of scope

## Timeline

**Week 1** (16 hours):
- Phase 1: Syntax & Parsing
- Phase 2: Type Checking

**Week 2** (14 hours):
- Phase 3: Runtime Enforcement
- Phase 4: Integration
- Documentation

**Total: ~30 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Composition complexity | Medium | Start with simple additive composition |
| Performance overhead | Low | Budget checks are integer comparisons |
| Ergonomics | Medium | Make budgets optional (unlimited by default) |
| Dynamic budget verification | Medium | Runtime checks where static analysis fails |

## Unified Budget Architecture

**This feature is part of a larger budget system spanning v0.7.0 - v0.8.0.**

### The Two Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    UNIFIED BUDGET SYSTEM                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: TYPE-LEVEL (v0.7.0 - This Doc)                    │
│  ────────────────────────────────────────                   │
│  Syntax:   ! {IO @limit=N, Net @limit=M}                    │
│  Purpose:  Static verification, type signatures             │
│  When:     Compile time where possible                      │
│                                                              │
│                        ↓ compiles to ↓                       │
│                                                              │
│  LAYER 2: RUNTIME (v0.8.0 - D4 BudgetContext)               │
│  ────────────────────────────────────────────               │
│  Syntax:   BudgetContext{Limits, Usage, OnViolation}        │
│  Purpose:  Runtime enforcement, spec-driven limits          │
│  When:     Effect handler invocation                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### How They Work Together

| Source | Layer 1 (Type) | Layer 2 (Runtime) |
|--------|----------------|-------------------|
| **From code** | `! {IO @limit=5}` | Compiler injects budget check |
| **From spec** | N/A | `envelope.api_calls: 5` in YAML |
| **Combined** | Both apply | Runtime enforces MIN(type, spec) |

### Shared BudgetContext

Both layers use the same runtime infrastructure:

```go
// internal/effects/budget.go (shared by v0.7 + v0.8)
type BudgetContext struct {
    // Per-effect limits (from type annotations OR spec)
    EffectLimits map[string]*EffectBudget  // "IO" -> {Limit: 5, Used: 2}

    // Global limits (from spec envelope)
    GlobalLimits BudgetLimits  // api_calls, execution_ms, tokens, cost_usd

    // Current consumption
    Usage BudgetUsage

    // Policy (from spec or default)
    Policy BudgetPolicy  // strict, warn, runtime

    // Violation handler
    OnViolation func(violation BudgetViolation) error
}

type EffectBudget struct {
    Effect   string  // "IO", "Net", "AI", etc.
    Limit    int     // Max uses
    Used     int     // Current count
    Source   string  // "type" or "spec"
}
```

### Implementation Phases

**Phase A (v0.7.0): Type-Level Budgets**
1. Parse `@limit=N` syntax in effect types
2. Store budget in effect AST
3. Generate BudgetContext initialization from type annotations
4. Wire budget checks into effect handlers
5. `BudgetExhaustedError` with source location

**Phase B (v0.8.0): Spec-Driven Budgets (D4)**
1. Parse `envelope:` from design doc YAML
2. Merge spec limits with type limits (MIN wins)
3. Add `--spec` flag to inject limits at runtime
4. Tri-state verification (PROVED/RUNTIME/UNKNOWN)
5. Budget consumption in traces

### Composition Rules

When both type and spec define limits:

```
Type:  func fetch() ! {Net @limit=10}
Spec:  envelope.api_calls: 5

Result: Net budget = MIN(10, 5) = 5
Reason: Spec is more restrictive, wins
```

When nested scopes have budgets:

```ailang
let outer = \(). ! {IO @limit=10}
  let inner = \(). ! {IO @limit=3}
    io1(); io2(); io3()  -- Uses 3 of inner's 3, 3 of outer's 10
  in
  inner();  -- 3 used
  inner();  -- 6 used
  inner();  -- 9 used
  io();     -- 10 used (OK)
  io()      -- 11 used (ERROR: outer exceeded)
```

**Rule:** Inner budget is a subset of outer. When inner exhausts, outer continues. When outer exhausts, everything stops.

---

## Related Documents

**Unified Budget System:**
- [m-d4-design-doc-driven-development.md](../v0_8_0/m-d4-design-doc-driven-development.md) - D4 BudgetContext spec
- [m-sem-kernel-vision.md](../v0_8_0/m-sem-kernel-vision.md) - Budget as bounded power (Pillar 5)
- [execution-profiles.md](../v0_6_2/execution-profiles.md) - Per-profile effect budgets

**Implemented (may inform design):**
- `internal/effects/context.go` - Existing NetContext constraints (MaxBytes, Timeout)
- `internal/effects/capability.go` - Capability.Meta for future budget metadata
- `internal/eval_harness/metrics.go` - Cost tracking (informs what to budget)

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A9: Cost Visibility
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - KPI tracking

## Future Work

- Memory budgets (byte limits) - integrate with GC
- Automatic budget inference from code analysis
- Budget negotiation between caller/callee
- Budget visualization in traces
- Token/cost budgets when AI effect boundary is defined

---

**Document created**: 2025-12-19
**Last updated**: 2025-12-23

**Design Review**: Budget semantics clarified (invocation counts vs semantic cost)

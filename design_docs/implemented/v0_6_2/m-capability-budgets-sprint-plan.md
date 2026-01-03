# Sprint Plan: M-CAPABILITY-BUDGETS

## Summary

Implement type-level capability budgets that limit effect usage, enabling Axiom A9 (Cost Visibility) compliance. Budgets count effect invocations per function scope and produce typed `BudgetExhaustedError` on violation.

**Duration:** 5 days (~30 hours)
**Dependencies:** None (builds on existing effect system)
**Risk Level:** Medium (parser and type system changes)

## Current Status Analysis

### Completed Recently
- M-CODEGEN-VALUE-TYPES: ~400 LOC in 3 days (value-type record generation)
- M-DX22: ~200 LOC in 1 day (ADT constructor disambiguation)
- Effect system: Stable, 12 effects supported (IO, FS, Net, Clock, etc.)

### Velocity
- Recent average: ~100-150 LOC/day
- Estimated capacity: ~500-600 LOC for this sprint (conservative)
- Buffer: 20% for integration issues

### Existing Infrastructure to Build On
- `internal/parser/parser_effect.go` - Effect annotation parsing (110 LOC)
- `internal/types/effects.go` - Effect row types (177 LOC)
- `internal/effects/context.go` - Runtime effect context (280 LOC)
- `internal/effects/errors.go` - CapabilityError pattern (43 LOC)

## Proposed Milestones

### Milestone 1: Parser & AST (M1-PARSE)
**Goal:** Parse `@limit=N` syntax in effect annotations
**Estimated:** 80 LOC implementation + 60 LOC tests = 140 LOC
**Duration:** 1 day

**Tasks:**
- Extend effect annotation parser to recognize `@limit=` after effect name
- Parse budget value (must be int literal or identifier for dynamic budgets)
- Store budget in effect AST representation
- Add parser error for invalid budget syntax (non-integer, negative)

**Files to modify:**
- `internal/parser/parser_effect.go` - Add budget parsing (~40 LOC)
- `internal/ast/ast.go` - Add EffectBudget field (~10 LOC)
- `internal/parser/parser_effect_test.go` - Budget parsing tests (~60 LOC)

**Acceptance Criteria:**
- [ ] `! {IO @limit=5}` parses correctly
- [ ] `! {IO @limit=n}` parses (dynamic budget)
- [ ] `! {IO @limit=5, FS}` parses (mixed budgets)
- [ ] `! {IO @limit=-1}` produces error
- [ ] All existing effect tests pass
- [ ] Linting clean

**Risks:**
- Token position bugs in parser - Mitigation: Use DEBUG_PARSER=1

---

### Milestone 2: Type System (M2-TYPES)
**Goal:** Track budgets in effect types during type checking
**Estimated:** 100 LOC implementation + 60 LOC tests = 160 LOC
**Duration:** 1 day

**Tasks:**
- Add Budget field to effect Row type
- Extend ElaborateEffectRow to handle budgets
- Implement budget composition (sum of nested budgets)
- Update FormatEffectRow to display budgets

**Files to modify:**
- `internal/types/effects.go` - Add budget tracking (~50 LOC)
- `internal/types/effects_test.go` - Budget type tests (~60 LOC)
- `internal/elaborate/elaborate.go` - Wire budget through (~30 LOC)
- `internal/types/pretty.go` - Budget formatting (~20 LOC)

**Acceptance Criteria:**
- [ ] Budget field present in effect Row
- [ ] Type signatures show `! {IO @limit=5}`
- [ ] Nested budgets compose (inner + outer)
- [ ] FormatEffectRow includes budget annotations
- [ ] All type tests pass
- [ ] Linting clean

**Risks:**
- Unification complexity - Mitigation: Start with exact match, defer subtyping

---

### Milestone 3: Runtime Budget Context (M3-RUNTIME)
**Goal:** Track and enforce budgets at runtime
**Estimated:** 150 LOC implementation + 80 LOC tests = 230 LOC
**Duration:** 1.5 days

**Tasks:**
- Create BudgetContext struct with per-effect tracking
- Add BudgetExhaustedError error type
- Integrate budget checking into EffContext
- Insert budget check before each effect operation
- Support dynamic budgets (runtime-computed limits)

**New files:**
- `internal/effects/budget.go` - BudgetContext, tracking logic (~100 LOC)
- `internal/effects/budget_test.go` - Unit tests (~80 LOC)

**Files to modify:**
- `internal/effects/context.go` - Add Budget field (~20 LOC)
- `internal/effects/errors.go` - Add BudgetExhaustedError (~30 LOC)

**Acceptance Criteria:**
- [ ] BudgetContext tracks per-effect usage
- [ ] Budget checks fire before effect operations
- [ ] BudgetExhaustedError includes effect, limit, used, position
- [ ] Fresh budget per function invocation
- [ ] All effect tests pass
- [ ] Linting clean

**Risks:**
- Performance overhead - Mitigation: Budget check is O(1) map lookup

---

### Milestone 4: Integration & Docs (M4-INTEGRATE)
**Goal:** Wire everything together, add examples and documentation
**Estimated:** 80 LOC implementation + 40 LOC tests = 120 LOC
**Duration:** 1.5 days

**Tasks:**
- Wire budget from AST through type checker to runtime
- Add example files demonstrating budgets
- Update error messages with helpful hints
- Add documentation page for capability budgets
- Run full test suite, fix any regressions

**New files:**
- `examples/budget_basic.ail` - Basic budget example (~30 LOC)
- `examples/budget_composition.ail` - Nested budget example (~30 LOC)

**Files to modify:**
- `internal/pipeline/pipeline.go` - Wire budget through compilation (~30 LOC)
- `internal/eval/eval.go` - Insert budget checks (~20 LOC)
- `docs/docs/reference/effects.md` - Add budget section (~40 lines prose)

**Acceptance Criteria:**
- [ ] End-to-end budget enforcement works
- [ ] Examples compile and run correctly
- [ ] Budget exhaustion produces clear error message
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make verify-examples` passes

**Risks:**
- Integration issues between phases - Mitigation: Incremental testing

---

## Success Metrics

- Test coverage: >80% for new code
- Examples passing: budget_basic.ail, budget_composition.ail
- Documentation: effects.md updated with budget section
- All tests passing: ✅
- All linting passing: ✅
- Axiom A9 compliance: 2/2 (improved from 1/2)

## LOC Summary

| Milestone | Implementation | Tests | Total |
|-----------|----------------|-------|-------|
| M1-PARSE | 80 | 60 | 140 |
| M2-TYPES | 100 | 60 | 160 |
| M3-RUNTIME | 150 | 80 | 230 |
| M4-INTEGRATE | 80 | 40 | 120 |
| **Total** | **410** | **240** | **650** |

## Dependencies

- Effect parser (`internal/parser/parser_effect.go`) must be stable
- Effect type system (`internal/types/effects.go`) must support extension
- No external dependencies

## Open Questions

1. **Dynamic budget syntax**: Should `@limit=n` allow any expression or just identifiers?
   - Proposed: Start with int literals and identifiers only

2. **Budget policy**: Should budget exhaustion be configurable (strict/warn/runtime)?
   - Proposed: Strict by default, defer policy flag to v0.8.0 (D4 integration)

3. **Budget inheritance**: How do module-level budgets interact with function budgets?
   - Proposed: Defer to v0.8.0, start with function-level only

## Notes

- This sprint implements Phase A (v0.7.0) of the Unified Budget Architecture
- Phase B (v0.8.0) will add spec-driven budgets via D4 integration
- Budget checks are O(1) - performance impact negligible
- Per-invocation semantics chosen for compositional reasoning (see design doc)

---

**Sprint created:** 2025-12-23
**Target version:** v0.7.0

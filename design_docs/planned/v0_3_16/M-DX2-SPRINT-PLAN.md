# Sprint Plan: M-DX2 Operator Development Experience

## Summary

Reduce polymorphic operator development time from 2h to 30-60min by adding type-guided lowering, Core IR helpers, debug CLI, better error messages, and documentation. This builds on M-DX1's success and addresses the ANF opacity and type info flow issues discovered during the `++` operator implementation.

**Duration:** 8-10 hours (1-1.5 working days)
**Dependencies:** M-DX1 (completed in v0.3.10)
**Risk Level:** Low-Medium (well-scoped, incremental changes)

## Current Status Analysis

### Completed Recently (Last 14 Days)

Based on CHANGELOG and git history:

- ✅ **List concatenation operator (`++`)**: ~200 LOC in 2 hours
  - Revealed ANF opacity issues and lack of type info flow
  - Triggered M-DX2 design
- ✅ **v0.3.15 Release**: Module path unification + Net builtin fixes
  - Small changes (~100 LOC)
  - Improved pass rate from 61/88 to 64/88
- ✅ **v0.3.14 (JSON Decode + DX)**: ~860 LOC + 42 tests in ~3 days
  - JSON decoding builtin
  - Pattern matching runtime fixes
  - Type system consistency (50+ fixes)
- ✅ **M-DX1 Complete**: Builtin registry + Type DSL
  - Reduced builtin dev time from 7.5h to 2.5h
  - 52 builtins migrated and documented

**Key commits (last 14 days):**
- 6ed5837: Concat lists (the trigger for M-DX2)
- 524f48f: Restore radar charts and fix dashboard
- 06626ce: Release v0.3.15
- 48f3240: M-DX1 Day 3 design doc

**Total LOC last 14 days:** ~1,900 insertions (including auto-generated docs)

### Velocity

**Recent average:**
- v0.3.14: ~860 LOC (implementation) + ~534 LOC (tests) = 1,394 LOC in ~3 days = **~465 LOC/day**
- v0.3.15: ~100 LOC in <1 day = **~100-200 LOC/day** (bug fixes, less feature work)
- `++` operator: ~200 LOC in 2 hours = **~800 LOC/day** (small, focused task)

**Conservative estimate for M-DX2:**
- Target: ~860 LOC new + ~195 LOC modified = **1,055 LOC total**
- Conservative rate: **400-500 LOC/day** (accounting for complexity and testing)
- **Estimated duration: 2-2.5 days** (design doc says 1 day, adding 50% buffer)

### Remaining from M-DX2 Design Doc

**Status:** All work is new (nothing started yet)

- 📋 **Phase 1: Type-Guided Lowering** - ~2.5h estimated
  - TypeInfo wrapper type (~30 LOC)
  - Populate during type inference (~30 LOC)
  - Operator table with variants (~40 LOC)
  - Update lowering to use types (~50 LOC)
  - Tests (~40 LOC)

- 📋 **Phase 2: Core IR Helpers** - ~0.75h estimated
  - Core helpers (~50 LOC)
  - Tests (~120 LOC)

- 📋 **Phase 3: Debug CLI** - ~2.5h estimated
  - Debug command (~200 LOC)
  - Flag handling, formatters

- 📋 **Phase 4: Error Messages** - ~0.75h estimated
  - Error helper (~60 LOC)
  - Wrap builtin casts (~40 LOC)
  - Tests (~60 LOC)

- 📋 **Phase 5: Documentation** - ~1.5h estimated
  - ANF.md (~300 lines)
  - adding-operators.md (~400 lines)
  - Update CLAUDE.md (~10 LOC)

## Proposed Milestones

### Milestone 1: TypeInfo Plumbing & Type-Guided Lowering

**Goal:** Connect type inference to lowering pass via TypeInfo map; eliminate ANF guessing

**Estimated:**
- Implementation: ~150 LOC
- Tests: ~80 LOC
- **Total: ~230 LOC**

**Duration:** 3-4 hours (Day 1 morning + early afternoon)

**Tasks:**

**Hour 1-1.5: TypeInfo Infrastructure**
- [ ] Create `internal/types/typeinfo.go`
  - Define `TypeInfo map[NodeID]types.Type` wrapper
  - Implement `Must(id NodeID) types.Type` with friendly error
  - Add test file (~30 LOC tests)
- [ ] Check if NodeIDs exist in AST
  - If not: Add `NodeID int` field to key AST nodes in `internal/ast/ast.go`
  - Ensure NodeIDs assigned during parsing (stable across passes)
  - Add NodeID stability test (~20 LOC)

**Hour 1.5-2.5: Type Inference Integration**
- [ ] Modify `internal/types/typechecker.go`
  - Add `TypeInfo` field to typechecker
  - Populate with **principal types** (post-generalization) for each expression
  - ~30 LOC changes
- [ ] Pass TypeInfo to lowering
  - Update `internal/pipeline/lowering_context.go` to hold TypeInfo
  - Wire through pipeline (~10 LOC)

**Hour 2.5-3.5: Operator Table & Type-Guided Selection**
- [ ] Update `internal/pipeline/op_table.go`
  - Add `Variants []OperatorVariant` to each operator
  - Each variant: `{Type: types.Type, Builtin: string}`
  - Populate for existing operators (~40 LOC)
- [ ] Update `internal/pipeline/op_lowering.go`
  - Replace ANF binding-chase with TypeInfo lookup
  - Consult operator table, select variant by type match
  - Add clear compile error for unknown/ambiguous types
  - **Delete ANF binding-chase branch** (~50 LOC change, net -20 LOC)
- [ ] Add integration test
  - Test: `[1,2] ++ [3,4]` selects `_list_concat`
  - Test: `"foo" ++ "bar"` selects `_str_concat`
  - **Regression guard**: Verify `++` works with ANF lookups disabled (~60 LOC test)

**Acceptance Criteria:**
- [ ] `TypeInfo.Must()` returns type when ID exists, panics with helpful error when missing
- [ ] NodeIDs stable across parses (test: parse same file twice → identical IDs)
- [ ] `++` operator uses TypeInfo for variant selection (no ANF traversal)
- [ ] Operator table has variants for all polymorphic operators
- [ ] Clear error message when type unknown/ambiguous
- [ ] All existing tests passing
- [ ] Regression guard test passes (ANF lookups disabled for `++`)

**Risks:**
- **NodeID implementation complexity** - Mitigation: Check if NodeIDs already exist; if not, add incrementally (start with Expr nodes only)
- **Type generalization timing** - Mitigation: Store types *after* generalization step; verify with debug prints
- **Breaking existing lowering** - Mitigation: Extensive tests; verify all examples still work

---

### Milestone 2: Core IR Helpers

**Goal:** Simplify ANF traversal with utility functions (for cases where TypeInfo unavailable)

**Estimated:**
- Implementation: ~50 LOC
- Tests: ~120 LOC
- **Total: ~170 LOC**

**Duration:** 1 hour (Day 1 late afternoon)

**Tasks:**

**Hour 1: Implement and Test Helpers**
- [ ] Create `internal/core/helpers.go`
  - `ResolveValue(expr CoreExpr, binds map[string]CoreExpr) CoreExpr`
    - Follow Var bindings to terminal value
    - Handle chains (a → b → c → literal)
    - ~25 LOC
  - `IsListValue(expr CoreExpr, binds map[string]CoreExpr) bool`
    - Check if resolved value is a ListLit
    - ~15 LOC
- [ ] Create `internal/core/helpers_test.go`
  - Simple Var lookup (10 cases)
  - Nested Var chains (depth 0-5, fuzz-style test) (15 cases)
  - Var not in bindings (5 cases)
  - Non-Var expressions (5 cases)
  - ~120 LOC total
- [ ] Use in lowering where appropriate
  - Replace any remaining manual Var chasing with `ResolveValue()`
  - Simplify type checks with `IsListValue()`
  - ~10 LOC changes in `op_lowering.go`

**Acceptance Criteria:**
- [ ] `ResolveValue()` handles depth 0-5 var chains correctly
- [ ] `IsListValue()` correctly identifies list literals (including through Vars)
- [ ] Var not in bindings returns original Var (no crash)
- [ ] All 35 test cases passing
- [ ] At least 3 uses of helpers in lowering pass (replaces manual traversal)

**Risks:**
- **Minimal** - Well-scoped, pure utility functions

---

### Milestone 3: Debug CLI

**Goal:** Add `ailang debug ast` command for instant ANF/type visibility

**Estimated:**
- Implementation: ~200 LOC
- Tests: ~40 LOC
- **Total: ~240 LOC**

**Duration:** 2.5-3 hours (Day 2 morning)

**Tasks:**

**Hour 1: Command Setup & Basic Printing**
- [ ] Create `cmd/ailang/debug.go`
  - Define `debugASTCmd` with cobra
  - Add flags:
    - `--surface` (default) / `--core`
    - `--show-types`
    - `--node <id>`
    - `--compact`
    - `--limit <n>` (default 1000)
  - ~80 LOC
- [ ] Register command in `cmd/ailang/main.go` (~5 LOC)
- [ ] Implement basic AST printing
  - Re-use existing pretty-printers from pipeline
  - Handle `--surface` vs `--core` flag
  - ~60 LOC

**Hour 2: Type Annotation Formatting**
- [ ] Implement TypeInfo formatter
  - Print `NodeID: Type` annotations when `--show-types` enabled
  - One-line legend at top: "=== Type Annotations (NodeID: Type) ==="
  - ~40 LOC
- [ ] Add `--node <id>` filter
  - Find subtree matching NodeID
  - Print only that subtree
  - ~20 LOC

**Hour 3: Polish & Testing**
- [ ] Add `--limit` safeguard
  - Count nodes while printing
  - Stop at limit with "... (truncated, use --limit to show more)"
  - ~15 LOC
- [ ] Add `--compact` mode
  - No whitespace, no pretty-printing
  - Useful for logs/CI
  - ~10 LOC
- [ ] Write tests in `cmd/ailang/debug_test.go`
  - Test: `ailang debug ast examples/list_ops.ail --show-types`
  - Assert output contains "=== Surface AST ===" and "=== Type Annotations ==="
  - Test: `--compact`, `--limit` flags work
  - ~40 LOC

**Acceptance Criteria:**
- [ ] `ailang debug ast <file>` prints Surface AST by default
- [ ] `--core` flag prints Core (ANF) representation
- [ ] `--show-types` adds NodeID: Type annotations
- [ ] `--node <id>` filters to specific subtree
- [ ] `--compact` removes whitespace
- [ ] `--limit` prevents gigantic dumps (default 1000 nodes)
- [ ] One-line legend when `--show-types` enabled
- [ ] Tests pass for all flags

**Risks:**
- **Pretty-printer compatibility** - Mitigation: Re-use existing printers; extend only if needed
- **TypeInfo availability** - Mitigation: Handle case where TypeInfo not populated (show "Type: <unknown>")

---

### Milestone 4: Better Runtime Errors

**Goal:** Replace cryptic panics with actionable error messages

**Estimated:**
- Implementation: ~100 LOC
- Tests: ~60 LOC
- **Total: ~160 LOC**

**Duration:** 1 hour (Day 2 early afternoon)

**Tasks:**

**Hour 1: Error Helper & Integration**
- [ ] Create `internal/eval/builtin_errors.go`
  - Define `ArgTypeMismatch(fn string, argIdx int, want, got string, hint string) error`
  - Format error with:
    - "Builtin type mismatch for '<fn>'"
    - "Expected: <want>"
    - "Received: <got>"
    - "Likely causes: ..." (list common causes)
    - "Fix: ..." (actionable steps)
  - Add smart hint: If type-guided lowering enabled → suggest `ailang debug ast --show-types`
  - ~60 LOC
- [ ] Update `internal/builtins/string.go`
  - Wrap String builtin casts with type checks
  - Call `ArgTypeMismatch()` on mismatch
  - ~20 LOC changes
- [ ] Update `internal/builtins/list.go`
  - Wrap List builtin casts with type checks
  - ~20 LOC changes
- [ ] Write tests in `internal/eval/builtin_errors_test.go`
  - Test: Call `_str_concat` with List → expect helpful error
  - Test: Call `_list_concat` with String → expect helpful error
  - Test: Error message contains expected fields
  - ~60 LOC

**Acceptance Criteria:**
- [ ] `ArgTypeMismatch()` formats error with all required fields
- [ ] Smart hint suggests `ailang debug ast --show-types` when appropriate
- [ ] String builtins wrapped with type checks
- [ ] List builtins wrapped with type checks
- [ ] Tests pass: wrong builtin variant → helpful error (not panic)
- [ ] Error schema updated if needed

**Risks:**
- **Minimal** - Straightforward error wrapping

---

### Milestone 5: Documentation

**Goal:** Write ANF guide and operator implementation checklist to enable onboarding in <30min

**Estimated:**
- ANF.md: ~300 lines
- adding-operators.md: ~400 lines
- CLAUDE.md updates: ~10 lines
- **Total: ~710 lines** (mostly prose)

**Duration:** 1.5-2 hours (Day 2 late afternoon)

**Tasks:**

**Hour 1: ANF Guide**
- [ ] Write `docs/architecture/ANF.md`
  - **Why ANF?** (simplifies eval, explicit sequencing, deterministic order)
  - **Reading ANF:** Vars, Lets, bindings, Let chains
  - **Common patterns:**
    - Nested Lets (a = ...; b = ...; c = f(a, b))
    - Application chains (f(g(h(x))))
    - Var indirection (resolve to terminal value)
  - **Examples:** Before/after (Surface AST vs Core ANF)
  - ~300 lines

**Hour 2: Operator Guide & Integration**
- [ ] Write `docs/guides/adding-operators.md`
  - **Step-by-step checklist:**
    1. Add token to lexer (if new operator symbol)
    2. Add to parser precedence table
    3. Register in operator table (`op_table.go`) with variants
    4. Implement builtins (one per variant)
    5. Write tests (hermetic, type-driven)
  - **Where to register:**
    - `internal/lexer/token.go` (token definition)
    - `internal/parser/parser.go` (precedence)
    - `internal/pipeline/op_table.go` (operator table)
    - `internal/builtins/<module>.go` (builtin implementation)
  - **Testing patterns:**
    - Hermetic tests (MockEffContext)
    - Type checks (verify variants selected correctly)
    - Edge cases (empty lists, type mismatches)
  - **Pitfalls:**
    - Polymorphic operators need multiple variants
    - ANF resolution (use TypeInfo, not shape guessing)
    - Operator table is source of truth (no hardcoded names)
  - ~400 lines
- [ ] Update `CLAUDE.md`
  - Link to new guides in "Adding a New Language Feature" section
  - Add references in "Common Tasks"
  - ~10 lines

**Acceptance Criteria:**
- [ ] `ANF.md` covers all common ANF patterns
- [ ] `adding-operators.md` has step-by-step checklist
- [ ] Both docs include concrete examples
- [ ] CLAUDE.md links to new docs
- [ ] CONTRIBUTING.md links to new docs (if applicable)

**Risks:**
- **Documentation goes stale** - Mitigation: Link from error messages so it's used frequently; update during code reviews

---

## Success Metrics

### Code Metrics
- **Total LOC:** ~1,055 LOC (860 new + 195 modified)
- **Test coverage:** Maintain current level (~29.9%), add ~300 LOC of tests
- **Files created:** 7 new files
- **Files modified:** 9 existing files

### Functional Metrics
- [ ] Type-guided lowering works for `++` operator (no ANF traversal)
- [ ] Core helpers used in at least 3 places in lowering pass
- [ ] `ailang debug ast --show-types` prints ANF + type annotations
- [ ] Runtime builtin mismatch errors include actionable diagnostics
- [ ] `docs/architecture/ANF.md` covers all common patterns
- [ ] `docs/guides/adding-operators.md` has step-by-step checklist
- [ ] **Manual test:** New contributor can add `**` (power) operator in <1 hour using guides
- [ ] All tests passing (existing + new)
- [ ] All linting passing

### Developer Experience Metrics
- [ ] Polymorphic operator dev time reduced from 2h to 30-60min (67-75% improvement)
- [ ] ANF opacity eliminated (no more manual Var chasing)
- [ ] Helpful error messages (5 seconds to understand instead of 5 minutes)
- [ ] Documentation enables onboarding in <30 minutes

## Dependencies

**Completed:**
- ✅ M-DX1 (Builtin Registry & Type DSL) - v0.3.10

**No external blockers** - All work is self-contained

## Open Questions

1. **Do NodeIDs already exist in the AST?**
   - Action: Check `internal/ast/ast.go` for NodeID fields
   - If not: Add incrementally (start with Expr nodes)
   - If yes: Verify they're stable across passes

2. **Should we add TypeInfo to all AST nodes or just expressions?**
   - Recommendation: Start with expressions only (where operators live)
   - Expand later if needed

3. **What's the format for operator table variants?**
   - Recommendation: `{Type: types.Type, Builtin: string}`
   - Example: `{Type: T.String(), Builtin: "_str_concat"}`

4. **Should debug CLI output JSON format as well as text?**
   - Recommendation: Defer JSON output to future enhancement
   - Focus on human-readable text for M-DX2

## Notes

### Assumptions
- NodeIDs can be added to AST nodes without major refactoring
- Operator table structure can be extended without breaking existing code
- TypeInfo lifecycle (freed after lowering) is acceptable

### Velocity Considerations
- Design doc estimates **1 working day (8h)**, actual velocity suggests **1.5-2 days (10-16h)** is more realistic
- Buffer already applied in estimates (2x multiplier from 4h raw estimate)
- Largest unknowns: NodeID implementation, TypeInfo integration

### Testing Strategy
- Unit tests for each new component (~300 LOC total)
- Integration tests for type-guided lowering (~80 LOC)
- Regression guards for `++` operator (critical!)
- Manual testing: Add `**` operator using new guides (<1h)

### Success Indicators
- If Milestone 1 takes >5 hours → re-estimate remaining work
- If tests fail after Milestone 1 → all examples might need updating
- If documentation takes <1 hour → not detailed enough

---

**Total Estimated Duration:** 8-10 hours (1-1.5 working days)

**Recommended Schedule:**
- **Day 1 (6-7h):** Milestones 1-2 (TypeInfo + Core Helpers)
- **Day 2 (4-5h):** Milestones 3-5 (Debug CLI + Errors + Docs)

**Risk Mitigation:**
- Start with Milestone 1 (highest risk/complexity)
- Verify regression guards pass before proceeding
- Test each phase incrementally (don't batch changes)

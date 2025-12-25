# M-RECORD-PATTERNS: Record Pattern Matching

**Status**: Planned
**Target**: v0.6.2
**Priority**: P1 (Medium)
**Estimated**: 1-2 days (~250-350 LOC)
**Dependencies**: None (AST node already exists)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact - pattern matching is already deterministic |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | Patterns don't affect effect system |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables exhaustiveness checking for records |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Reduces code complexity - direct destructuring vs field access |
| A8: Minimal Syntax | 0 | Follows existing pattern syntax conventions |
| A9: Cost Visibility | 0 | No hidden costs |
| A10: Composability | +1 | Composes with existing patterns (nested, guards) |
| A11: Structured Failure | 0 | Follows existing pattern match failure semantics |
| A12: System Boundary | 0 | No boundary crossing |

**Net Score: +3** → **Decision: ✅ Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis by making destructuring explicit

## Problem Statement

Record pattern matching allows destructuring records directly in match expressions, extracting fields into bindings without explicit field access.

**Current State:**
- AST node `RecordPattern` exists in `internal/ast/ast_patterns.go`
- Parser function `parseRecordPattern()` returns `nil` (stub)
- Users must manually access fields: `let name = person.name; let age = person.age`
- Example files incorrectly document this as "future syntax"

**Impact:**
- Increased verbosity when working with records
- Inconsistent with ADT and list pattern matching (which work)
- Misleading comments in example files

## Goals

**Primary Goal:** Enable record destructuring in match expressions

**Success Metrics:**
- `match person { {name, age} => ... }` parses and evaluates correctly
- `match person { {name: n, age: a} => ... }` with renaming works
- Nested patterns work: `match data { {user: {name}} => ... }`
- Row polymorphism preserved (partial matches allowed)

## Solution Design

### Overview

Implement record pattern parsing, elaborate to Core patterns, and evaluate by matching record fields. Leverage existing row polymorphism for partial matching.

### Architecture

**Components:**
1. **Parser** (`parser_pattern.go`): Parse `{field, field: binding}` syntax
2. **Elaboration** (`elaborate/`): Convert AST RecordPattern to Core pattern
3. **Evaluator** (`eval_patterns.go`): Match record values against patterns

### Implementation Plan

**Phase 1: Parser** (~2 hours, ~60 LOC)
- [ ] Implement `parseRecordPattern()` in `parser_pattern.go`
- [ ] Handle shorthand: `{name}` → binds `name` to field `name`
- [ ] Handle renaming: `{name: n}` → binds `n` to field `name`
- [ ] Handle nested patterns: `{user: {name}}` → nested destructure
- [ ] Add parser tests

**Phase 2: Elaboration** (~2 hours, ~40 LOC)
- [ ] Update `elaborate/patterns.go` to handle `ast.RecordPattern`
- [ ] Convert to `core.RecordPat` (may need to add Core node)
- [ ] Ensure type checking with row polymorphism

**Phase 3: Evaluation** (~2 hours, ~60 LOC)
- [ ] Add `case *core.RecordPat:` in `evalPattern()`
- [ ] Extract fields from record value
- [ ] Bind to pattern variables
- [ ] Handle partial matches (row polymorphism)

**Phase 4: Tests & Examples** (~2 hours, ~100 LOC)
- [ ] Unit tests for parser
- [ ] Integration tests for full pipeline
- [ ] Update example file that has "future syntax" comment
- [ ] Add `examples/runnable/record_patterns.ail`

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_pattern.go` - Implement parseRecordPattern (~60 LOC)
- `internal/elaborate/patterns.go` - Handle RecordPattern elaboration (~40 LOC)
- `internal/eval/eval_patterns.go` - Evaluate record patterns (~60 LOC)
- `internal/core/core.go` - May need RecordPat Core node (~20 LOC)

**New files:**
- `examples/runnable/record_patterns.ail` - Example file (~30 LOC)

## Examples

### Example 1: Basic Record Pattern

**Before (current workaround):**
```ailang
func greet(person: {name: string, age: int}): string =
  let name = person.name in
  let age = person.age in
  name ++ " is " ++ show(age)
```

**After:**
```ailang
func greet(person: {name: string, age: int}): string =
  match person {
    {name, age} => name ++ " is " ++ show(age)
  }
```

### Example 2: Renaming Fields

```ailang
match user {
  {name: userName, email: userEmail} =>
    "User: " ++ userName ++ " <" ++ userEmail ++ ">"
}
```

### Example 3: Nested Records

```ailang
match response {
  {data: {user: {name}}} => "Found user: " ++ name
  {error: msg} => "Error: " ++ msg
}
```

### Example 4: Row Polymorphism (Partial Match)

```ailang
-- Works with any record that has 'name' field
func getName(r: {name: string | ρ}): string =
  match r { {name} => name }
```

## Success Criteria

- [ ] `{field}` shorthand parses correctly
- [ ] `{field: binding}` renaming works
- [ ] Nested patterns `{a: {b}}` work
- [ ] Type inference preserves row polymorphism
- [ ] Exhaustiveness checking includes record patterns
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Example file added and passing

## Testing Strategy

**Unit tests:**
- Parser: Various record pattern syntax forms
- Elaboration: Conversion to Core representation
- Evaluator: Pattern matching semantics

**Integration tests:**
- Full pipeline: parse → elaborate → type check → evaluate
- Type inference with record patterns
- Row polymorphism preservation

**Manual testing:**
- REPL: `match {x: 1, y: 2} { {x} => x }`
- File execution with examples

## Non-Goals

**Not in this feature:**
- Record comprehensions (`{f: v*2 for f:v in rec}`) - Future enhancement
- Record extension patterns (`{...rest}`) - Future enhancement
- Punning in record construction (`{name, age}` = `{name: name, age: age}`) - Separate feature

## Timeline

**Day 1** (4-5 hours):
- Phase 1: Parser implementation
- Phase 2: Elaboration

**Day 2** (3-4 hours):
- Phase 3: Evaluator
- Phase 4: Tests and examples

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Core AST may not have RecordPat | Low | Add node if needed (~20 LOC) |
| Row polymorphism edge cases | Medium | Follow existing record type checking patterns |
| Exhaustiveness checking complexity | Low | Start without exhaustiveness, add later if needed |

## Related Documents

**Implemented (may inform design):**
- `design_docs/archive/records/M-R5_future_enhancements.md` - Lists record patterns as future work
- `design_docs/implemented/v0_6_1/m-dx16-inline-record-match-arms.md` - Recent record work

**Planned (check for overlap):**
- None found

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/ast/ast_patterns.go:66` - Existing RecordPattern AST node
- `internal/parser/parser_pattern.go:208` - Stub to implement

## Future Work

- Record comprehensions: `{f: toUpper(v) for f:v in record}`
- Rest patterns: `{name, ...rest}` to capture remaining fields
- Record punning in construction: `{name, age}` shorthand

---

**Document created**: 2025-12-25
**Last updated**: 2025-12-25

# M-GAP4: Record Width Subtyping via Row Polymorphism

## Status
- **Status:** Planned
- **Target:** v0.6.4+
- **Priority:** P1 (High)
- **Estimated:** 2-3 days
- **Dependencies:** None (extends existing row polymorphism)

## Problem Statement

Records in AILANG currently use exact matching - there is no width subtyping. You cannot pass a record with extra fields to a function expecting fewer fields.

### Current Behavior (Exact Records)
```ailang
pure func countTurns(events: [{turnNum: int}]) -> int =
  length(events)

let events = [{turnNum: 1, streamType: "text", text: "hello"}]
countTurns(events)
-- ✗ Error: field count mismatch: 1 vs 3
```

**Note:** This is not "broken" - it's the current deliberate typing discipline. This design doc proposes a change to allow width subtyping when explicitly requested.

### Desired Behavior (with explicit open records)
```ailang
-- Open record type: accepts records with at least turnNum
pure func countTurns(events: [{turnNum: int | r}]) -> int =
  length(events)

-- Or with sugar syntax (proposed):
pure func countTurns(events: [{turnNum: int, ..}]) -> int =
  length(events)

let events = [{turnNum: 1, streamType: "text", text: "hello"}]
countTurns(events)  -- ✓ Returns 1
```

### Impact
- Cannot write generic functions that operate on record subsets
- Must repeat full record types in every function signature
- Breaks modularity and code reuse
- Discovered during dogfooding (event processing functions)

## Goals

**Primary Goal:** Enable width subtyping for records when explicitly requested via open record syntax.

**Success Metrics:**
- `{a: T | r}` unifies with `{a: T, b: U}` yielding `r := {b: U}`
- `{a: T}` remains exact and rejects extra fields (no silent semantic changes)
- Lists of open records work: `[{id: int | r}]`
- Clear diagnostics on missing/extra fields with suggestions for open records
- Principal types preserved (deterministic inference)

## Solution Design

### Overview

AILANG already has row polymorphism infrastructure (row variables `| r` in types). The issue is that the unification algorithm enforces exact field count matching instead of proper row unification that supports extension.

### Key Design Decision: Explicit Openness Only

**Do NOT change the meaning of `{a: T}`. Keep it exact.**

| Syntax | Meaning | Use Case |
|--------|---------|----------|
| `{name: string}` | Exact record - exactly these fields | Strict APIs, preventing data leakage |
| `{name: string \| r}` | Open record - at least these fields | Generic functions, pipelines |
| `{name: string, ..}` | Sugar for open record (proposed) | Concise open record syntax |

**Rationale:**
- No silent breaking semantic changes to existing code
- Clear user intent - openness is a deliberate choice
- Existing code that relies on exact matching continues to work
- Prompts/templates can default to open records for AI-generated code

**Rejected Alternative:** Implicit openness for function parameters
- Would silently widen APIs
- Could mask mistakes (extra fields flowing through unintentionally)
- Makes type errors harder to understand

### Current Type System Behavior

```ailang
-- Row polymorphism syntax exists (internal)
type HasName = {name: string | r}

-- Unification enforces exact match (current)
func greet(x: {name: string}) = "Hello, " ++ x.name
greet({name: "Alice", age: 30})  -- ✗ Fails (exact record)

-- After this change, use explicit open syntax:
func greet(x: {name: string | r}) = "Hello, " ++ x.name
greet({name: "Alice", age: 30})  -- ✓ Works (open record)
```

### Proposed Changes

**1. Proper Row Unification**

The unification algorithm needs to handle row types properly:

```
unify({a: T, b: U} , {a: T | r})  →  r := {b: U}
unify({a: T | r1} , {a: T | r2})  →  r1 ~ r2
unify({a: T}      , {a: T, b: U}) →  ERROR (exact record)
```

**2. Open Record Sugar (Optional)**

Add shorthand `..` syntax for open records:
```ailang
{name: string, ..}  ≡  {name: string | r}  -- fresh r
```

**3. Improved Error Messages**

When unification fails on records:
- List missing fields
- List extra fields (when exact record rejects them)
- Suggest: "use open record `{field: T | r}` to accept extra fields"

### Row Unification Algorithm

The current field-count-based approach is insufficient. Proper row unification requires:

#### 1. Row Normalization

Records must be treated as:
- A map of labels → types (ordering irrelevant)
- Plus an optional row tail (row variable or empty row)

```go
type NormalizedRecord struct {
    Fields map[string]Type  // label → type
    Tail   RowTail          // nil (closed), RowVar, or RowEmpty
}
```

#### 2. Row Difference / Remainder Computation

When unifying two records:
1. Unify common fields (must all succeed)
2. Compute residual fields (fields in one but not the other)
3. Push residuals into the row tail on the opposite side (if open)
4. If exact record has residuals to absorb → error

```go
// Helper functions needed:
func splitCommon(fieldsA, fieldsB map[string]Type) (
    common map[string][2]Type,  // field name → (typeA, typeB)
    onlyA  map[string]Type,
    onlyB  map[string]Type,
)

func unifyFieldTypes(common map[string][2]Type) error

func unifyRowTail(tail RowTail, residualFields map[string]Type) error
```

#### 3. Row Variable Solving

When unifying `{a: int | r}` with `{a: int, b: bool}`:
- `r` must become `{b: bool}` (or equivalent row form)
- Requires occurs-check for rows (avoid `r = {a: int | r}`)
- Substitution must apply to row tails

#### 4. Ambiguity Control

Unifying `{a: int | r1}` with `{a: int | r2}`:
- Result: `r1 ~ r2` (shared variable or equality constraint)
- Do NOT invent fields

### Implementation Plan

#### Phase 1: Row Unification (~6 hours)

**File:** `internal/types/unify.go`

Replace field-count comparison with proper row unification:

```go
func (u *Unifier) unifyRecord(rec1, rec2 *RecordType) error {
    // 1. Normalize both records (extract fields + tail)
    norm1 := normalizeRecord(rec1)
    norm2 := normalizeRecord(rec2)

    // 2. Split into common and residual fields
    common, onlyIn1, onlyIn2 := splitCommon(norm1.Fields, norm2.Fields)

    // 3. Unify common fields
    for field, types := range common {
        if err := u.unify(types[0], types[1]); err != nil {
            return fmt.Errorf("field %s: %w", field, err)
        }
    }

    // 4. Handle residuals via row tails
    if len(onlyIn1) > 0 {
        if err := u.absorbIntoTail(norm2.Tail, onlyIn1, "second"); err != nil {
            return err  // "extra fields: x, y (use open record)"
        }
    }
    if len(onlyIn2) > 0 {
        if err := u.absorbIntoTail(norm1.Tail, onlyIn2, "first"); err != nil {
            return err
        }
    }

    // 5. Unify tails if both open
    if norm1.Tail != nil && norm2.Tail != nil {
        if err := u.unifyRowTails(norm1.Tail, norm2.Tail); err != nil {
            return err
        }
    }

    return nil
}

func (u *Unifier) absorbIntoTail(tail RowTail, residuals map[string]Type, which string) error {
    if tail == nil {
        // Exact record cannot absorb extra fields
        fields := sortedKeys(residuals)
        return fmt.Errorf("extra fields in %s record: %s (use open record {... | r} to accept extra fields)",
            which, strings.Join(fields, ", "))
    }
    // Open record: unify tail with residual row
    residualRow := makeRowFromFields(residuals)
    return u.unifyRowTail(tail, residualRow)
}
```

**Additional helpers in `internal/types/types.go`:**
- `normalizeRecord(rec *RecordType) NormalizedRecord`
- `splitCommon(a, b map[string]Type) (common, onlyA, onlyB)`
- `makeRowFromFields(fields map[string]Type) RowType`
- `rowOccursCheck(varID int, row RowType) bool`

#### Phase 2: Open Record Sugar (~2 hours)

**File:** `internal/parser/parser.go`

Add parsing for `..` syntax in record types:

```ailang
{name: string, ..}  -- desugars to {name: string | r} with fresh r
```

**File:** `internal/lexer/token.go`

Add `DOTDOT` token if not present.

#### Phase 3: Error Messages (~2 hours)

**File:** `internal/errors/type_errors.go`

Structured error for record mismatch:
```go
type RecordMismatchError struct {
    MissingFields []string  // required but not present
    ExtraFields   []string  // present but not accepted (exact record)
    Suggestion    string    // "use {field: T | r} to accept extra fields"
}
```

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/types/unify.go` | Proper row unification | ~80 |
| `internal/types/types.go` | Row helper functions | ~40 |
| `internal/parser/parser.go` | `..` sugar parsing | ~20 |
| `internal/lexer/token.go` | DOTDOT token (if needed) | ~5 |
| `internal/errors/type_errors.go` | Structured record errors | ~30 |

### Design Decisions

**Q: Should width subtyping be implicit or explicit?**

A: **Explicit only.** Exact records (`{a: T}`) remain exact everywhere. Width subtyping requires open record syntax (`{a: T | r}` or `{a: T, ..}`).

**Rationale:**
- No silent semantic changes to existing code
- Users opt-in to openness deliberately
- Prevents masked mistakes (extra data flowing through)
- Prompts can default to open records without language-level implicit behavior

**Q: Does this affect record literals?**

A: No. Record literals have exact types. `{name: "Alice", age: 30}` has type `{name: string, age: int}` (exact).

**Q: What about pattern matching?**

A: Record patterns should continue to work. A pattern `{name: n}` matches records with at least a `name` field. The match compiler and runtime representation must agree on this.

**Q: Performance impact?**

A: Minimal. Row variables already exist; this changes how they unify. The algorithm is still O(n) in field count.

**Q: Principal types?**

A: Yes, preserved. Row polymorphism can have principal types if row unification is done properly. This is a non-negotiable requirement for AILANG (determinism and AI DX).

## Examples

### Exact Record (unchanged behavior)
```ailang
-- Exact record: rejects extra fields
pure func getName(x: {name: string}) -> string = x.name
getName({name: "Alice"})           -- ✓ Works
getName({name: "Alice", age: 30})  -- ✗ Error: extra field 'age'
```

### Open Record (new capability)
```ailang
-- Open record: accepts extra fields
pure func getName(x: {name: string | r}) -> string = x.name
getName({name: "Alice"})           -- ✓ Works
getName({name: "Alice", age: 30})  -- ✓ Works (age absorbed into r)

-- With sugar syntax:
pure func getName(x: {name: string, ..}) -> string = x.name
getName({name: "Bob", role: "admin"})  -- ✓ Works
```

### List of Open Records (idiomatic pattern)
```ailang
-- For pipelines processing records with varying schemas:
pure func extractIds(records: [{id: int | r}]) -> [int] =
  map(\rec. rec.id, records)

-- Or with sugar:
pure func extractIds(records: [{id: int, ..}]) -> [int] =
  map(\rec. rec.id, records)

let events = [
  {id: 1, type: "click", timestamp: 1234},
  {id: 2, type: "view", timestamp: 1235}
]
extractIds(events)  -- ✓ Returns [1, 2]
```

### Nested Open Records
```ailang
pure func getCity(x: {address: {city: string, ..}, ..}) -> string =
  x.address.city

let person = {
  name: "Alice",
  address: {city: "NYC", zip: "10001", country: "USA"}
}
getCity(person)  -- ✓ Returns "NYC"
```

## Testing

### New Test Cases
```ailang
-- test_record_width_subtyping.ail

-- Exact records remain exact
pure func exactName(x: {name: string}) -> string = x.name
let _ = exactName({name: "Alice"})           -- ✓ Works
-- exactName({name: "Alice", age: 30})       -- ✗ Should fail

-- Open records accept extras
pure func openName(x: {name: string | r}) -> string = x.name
let _ = openName({name: "Alice"})            -- ✓ Works
let _ = openName({name: "Alice", age: 30})   -- ✓ Works

-- Sugar syntax
pure func sugarName(x: {name: string, ..}) -> string = x.name
let _ = sugarName({name: "Bob", role: "admin"})  -- ✓ Works

-- List of open records
pure func countItems(xs: [{id: int, ..}]) -> int = length(xs)
let _ = countItems([{id: 1, data: "x"}, {id: 2, data: "y"}])  -- ✓ Works

-- Nested open records
pure func getZip(x: {address: {zip: string, ..}, ..}) -> string =
  x.address.zip
let _ = getZip({address: {zip: "10001", city: "NYC"}, name: "Alice"})  -- ✓ Works

-- Row variable solving
pure func passThrough[r](x: {id: int | r}) -> {id: int | r} = x
let result = passThrough({id: 1, extra: "data"})
-- result should have type {id: int, extra: string}
```

### Regression Tests
- [ ] Existing record tests still pass
- [ ] Exact records reject extra fields (explicit test)
- [ ] Type aliases remain exact
- [ ] Row polymorphism tests still pass
- [ ] Pattern matching on records still works
- [ ] Occurs check prevents cyclic row types

### Error Message Tests
```ailang
-- Should produce clear error:
pure func f(x: {a: int}) -> int = x.a
f({a: 1, b: 2})
-- Expected error: "extra field 'b' in exact record
--                  suggestion: use {a: int | r} to accept extra fields"
```

## Success Criteria

- [ ] `{a: T | r}` unifies with `{a: T, b: U}` yielding `r := {b: U}`
- [ ] `{a: T}` remains exact and rejects extra fields
- [ ] Lists of open records work: `[{id: int | r}]`
- [ ] Sugar syntax `{a: T, ..}` parses and works
- [ ] Clear diagnostics on missing/extra fields
- [ ] Error messages suggest open record syntax
- [ ] Principal types preserved (no ambiguous inference)
- [ ] Documentation updated with open record patterns

## Risks

### Risk 1: Masked Mistakes (if implicit openness - AVOIDED)

**Scenario (if we had implicit openness):**
```ailang
pure func logEvent(e: {turnNum: int}) = ...
logEvent({turnNum: 1, text: "secret"})  -- Would typecheck silently
```

If the intent was to prevent extra data flowing, implicit openness removes that guardrail.

**Mitigation:** We chose explicit openness. Users must write `{turnNum: int | r}` to accept extra fields.

### Risk 2: Inference Generalization Surprises

Row variables introduced implicitly can generalize unexpectedly, producing surprising polymorphic APIs.

**Mitigation:**
- Explicit openness means users choose when to introduce row variables
- Clear documentation on when generalization occurs

### Risk 3: Pattern Matching Confusion

If record patterns are exact today, there could be confusion about whether patterns accept extra fields.

**Mitigation:**
- Document pattern matching behavior clearly
- Patterns `{a: x}` match records with at least field `a` (consistent with most languages)
- Match compiler treats extra fields as ignored

### Risk 4: Runtime Representation Mismatch

If records are runtime structs with fixed layout, extra fields need somewhere to go.

**Mitigation:**
- Verify current runtime representation supports extra fields
- If fixed-layout structs: may need map fallback for open records

## Open Questions (Need Investigation)

Before implementation, answer these:

1. **Row tail syntax:** Is `{a: T | r}` already user-facing syntax, or internal only?
   - Check parser and examples for evidence of user-facing row syntax

2. **Record representation:** How are record types represented internally?
   - Ordered list, map, or canonical sorted list?
   - This affects the `normalizeRecord` implementation

3. **Current unification failure:** Where exactly does unification fail today for open records?
   - Is it `unifyRecord` field count check, or somewhere else?
   - Need to trace through a failing example

4. **Runtime record representation:** Are records runtime structs (fixed layout) or maps?
   - Affects pattern compilation
   - May constrain implementation approach

## Timeline

**Day 1:** Investigation + row unification core (~6 hours)
- Answer open questions above
- Implement `normalizeRecord`, `splitCommon`, row unification

**Day 2:** Testing + sugar syntax (~4 hours)
- Comprehensive test suite
- `..` sugar parsing (if desired)

**Day 3:** Error messages + documentation (~2 hours)
- Structured error types
- Documentation updates

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | More flexible API design for AI-generated code |
| A10: Composability | +1 | Functions compose better with open records |
| A5: Bounded Verification | +1 | Explicit openness preserves local reasoning |
| A3: Determinism | 0 | Principal types preserved |

**Net Score:** +3 (Accept)

## Related Documents

- [internal/types/unify.go](../../../internal/types/unify.go) - Unification implementation
- [internal/types/types.go](../../../internal/types/types.go) - Row type definitions
- PureScript records (reference implementation)
- Elm extensible records (simpler model)
- OCaml row polymorphism (academic foundation)

## References

- [Extensible Records with Scoped Labels](https://www.microsoft.com/en-us/research/publication/extensible-records-with-scoped-labels/) - Leijen
- [Row Polymorphism](https://homepages.inf.ed.ac.uk/slindley/papers/fst-draft-may2015.pdf) - Lindley & Cheney

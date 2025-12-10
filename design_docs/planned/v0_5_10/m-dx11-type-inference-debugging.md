# M-DX11: Type Inference Debugging Tools

**Status**: Partial (Phase 1 Complete)
**Target**: v0.5.10
**Priority**: P1 - Medium
**Estimated**: 3 days (24 hours)
**Dependencies**: None

## Progress Update (2025-12-10)

**Phase 1 COMPLETE:** Substitution chain tests implemented in `internal/types/substitution_chain_test.go`:
- `TestSubstitutionChainResolution` - catches α → β → float chains
- `TestSubstitutionIdempotent` - verifies idempotency property
- `TestSubstitutionLongChain` - α1 → α2 → α3 → α4 → float
- `TestSubstitutionNoChain` - direct mappings still work
- `TestSubstitutionInFunction` - chains in function param types

**Validation:** Tests caught immediate regression when chain-following code was temporarily reverted during M-FIX-RECORD-UPDATE debugging.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Reduces debug print statement clutter |
| Preserve Semantic Clarity | + | +1 | Makes type inference behavior explicit and traceable |
| Increase Determinism | 0 | 0 | Debug tools, no runtime impact |
| Lower Token Cost | + | +1 | Faster debugging = fewer context-burning iterations |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

Debugging type inference issues in AILANG is extremely difficult. The M-FIX-FLOAT-OP bug (float operators dispatching to int) took **5 hours** to fix because of poor visibility into the type inference engine.

**Current State:**
- No visibility into substitution chains (had to add 50+ lines of debug output)
- Type information fragmented across 4+ data structures
- No tests for transitive substitution (root cause went undetected)
- Error messages don't show type provenance
- Fixing a type issue required changes across 7 files

**Impact:**
- AI agents waste context on debugging iterations
- Human developers spend hours on issues that should take minutes
- Bugs in type inference engine go undetected for months
- Every type inference fix is a multi-file, high-risk change

**Concrete Example (M-FIX-FLOAT-OP):**
```
Session timeline:
T+0h: "PI()/4.0 dispatches to div_Int" - start debugging
T+1h: Check operator dispatch - wrong direction
T+2h: Check return type annotations - still broken
T+3h: Check class upgrade - still broken
T+4h: Add 50 lines of debug output, find substitution chain bug
T+5h: Fix root cause, cleanup, document
```

The root cause was a single line: substitution wasn't following chains. With proper tools, this would have been found in 30 minutes.

## Goals

**Primary Goal:** Reduce type inference debugging time from hours to minutes through better visibility tools.

**Success Metrics:**
- Substitution chain tests exist and prevent regression (immediate value)
- `typeReport(nodeID)` function shows all type info for a node (canonical primitive)
- `--debug-types` flag shows full type inference state in one place (CLI wrapper)
- Type provenance tracking available (where did this type come from?)
- Any type inference bug can be diagnosed in < 30 minutes

## Solution Design

### Overview

Add a suite of debugging tools for type inference, **ordered by immediate value and foundation-building**:

1. **Substitution chain tests** (foundation) - catch the exact class of bug we hit
2. **typeReport(nodeID)** (canonical primitive) - single entry point for all type info
3. **`--debug-types` CLI** (user-facing) - formatting/orchestration around typeReport
4. **Type provenance** (optional enhancement) - trace where types originate

**Key Architectural Principle:** Keep debug tooling **orthogonal** to the core checker.
- No `if debug { fmt.Printf(...) }` scattered all over
- Define a small `TypeDebugSink` interface, pass via context
- CLI builds "verbose tracer" only when `--debug-types` is set
- Everywhere else: no-op implementation

### Architecture

```go
// TypeDebugSink - orthogonal debug interface (no-op by default)
type TypeDebugSink interface {
    // Called at key inference points
    OnFreshTypeVar(tv TypeVarID, span SourceSpan, origin OriginKind)
    OnUnify(left, right Type, result Type)
    OnSubstitute(tv TypeVarID, resolved Type)
    OnDefault(tv TypeVarID, defaulted Type, reason string)
    OnConstraintAdd(c Constraint)
    OnConstraintResolve(c Constraint, resolved Type)
}

// NoOpDebugSink - used in production (zero overhead)
type NoOpDebugSink struct{}
func (NoOpDebugSink) OnFreshTypeVar(TypeVarID, SourceSpan, OriginKind) {}
func (NoOpDebugSink) OnUnify(Type, Type, Type)                         {}
// ... etc

// VerboseDebugSink - used when --debug-types is set
type VerboseDebugSink struct {
    provenance map[TypeVarID][]TypeOrigin
    events     []DebugEvent
}
```

**Components:**

#### 1. `typeReport(nodeID)` - The Canonical Primitive

Single call that consolidates fragmented data from 4 structures:

```go
type TypeReport struct {
    NodeID      uint64
    Raw         types.Type       // What's in CoreTI (may have TVars)
    Resolved    types.Type       // After applying full substitution closure
    Constraints []ConstraintRef  // References to constraints mentioning this type
    Origins     []TypeOrigin     // Zero or more provenance entries
}

type ConstraintRef struct {
    Constraint Constraint
    SourceSpan SourceSpan       // Where this constraint was introduced
}

type TypeOrigin struct {
    Kind   OriginKind  // Annotation / Literal / Inferred / Defaulted / FromUse / FromPattern
    NodeID uint64      // Originating node
    Span   SourceSpan  // Source location
    Note   string      // Human-readable: "parameter annotation x: float"
}

type OriginKind int
const (
    OriginAnnotation OriginKind = iota  // From explicit type annotation
    OriginLiteral                        // From literal (3.14 → float)
    OriginInferred                       // From unification
    OriginDefaulted                      // From defaulting pass (Num → Int)
    OriginFromUse                        // From call site / application
    OriginFromPattern                    // From pattern match binding
)
```

**Implementation principle:** `typeReport` is a **thin façade** over existing structures, not a new store of truth:
- Takes NodeID + InferenceContext/CoreTypeChecker
- Pulls: `coreType := coreTI[nodeID]`
- Resolves: `resolved := subst.ApplyClosure(coreType)` (full chain resolution)
- Finds: constraints where this node's type vars appear
- Looks up: provenance for those type vars (when debug mode enabled)

You do **not** want TypeReport state living anywhere persistent; always derive from live environment.

#### 2. Substitution Chain Tests

Three concrete tests that would have caught M-FIX-FLOAT-OP:

**Test 1: Idempotence**
```go
// apply(sub, t) == apply(sub, apply(sub, t))
// For all shapes: TVars, nested types, ADTs, function types
func TestSubstitutionIdempotent(t *testing.T) {
    sub := Substitution{
        "α": TFloat,
        "β": &TFunc{Param: &TVar2{Name: "α"}, Return: TInt},
    }

    for _, ty := range testTypes {
        once := ApplySubstitution(sub, ty)
        twice := ApplySubstitution(sub, once)
        assert.Equal(t, once, twice, "substitution must be idempotent")
    }
}
```

**Test 2: No Residual TVars**
```go
// The exact test that would have caught M-FIX-FLOAT-OP
func TestSubstitutionChainResolution(t *testing.T) {
    // Create chain: α → β → float
    sub := Substitution{
        "α": &TVar2{Name: "β"},
        "β": TFloat,
    }

    // Apply to α - should get float, not β
    result := ApplySubstitution(sub, &TVar2{Name: "α"})

    // This test would have FAILED before the fix
    if _, ok := result.(*TVar2); ok {
        t.Fatalf("Chain not resolved: got %v, want float", result)
    }
    assert.Equal(t, TFloat, result)
}
```

**Test 3: Cycle Detection**
```go
func TestSubstitutionCycleDetection(t *testing.T) {
    // Create cycle: α → β, β → α
    sub := Substitution{
        "α": &TVar2{Name: "β"},
        "β": &TVar2{Name: "α"},
    }

    // Either Apply panics/errors, or checker fails with "type cycle"
    // The important thing: we don't infinite loop
    assert.Panics(t, func() {
        ApplySubstitution(sub, &TVar2{Name: "α"})
    }, "cycle should be detected")
}
```

#### 3. `--debug-types` CLI Flag

Focused and queryable, but simple for v0.5.10:

```bash
# Global dump (most common use)
ailang run --debug-types test.ail

# Focus on specific node (when you know the NodeID)
ailang run --debug-types --node 44 test.ail

# Future: by source location (maps to NodeID via source map)
# ailang run --debug-types --span file.ail:12:5 test.ail
```

**Important separation of concerns:**
- Core checker exposes APIs:
  - `GetSubstitution() Substitution`
  - `GetCoreTI() map[uint64]Type`
  - `ListConstraints() []Constraint`
  - `TypeReport(nodeID) TypeReport`
- CLI layer is responsible for formatting and printing
- AI tools can call same APIs in-process without parsing strings

#### 4. Type Provenance (Optional, Debug-Only)

**Key design decision:** Provenance is a **side-map**, not fields on Type.

Why not attach to Type tree:
- Blows up memory
- Pollutes equality/hashing logic
- Easy to accidentally ship provenance into codegen

Instead:
```go
// Inside VerboseDebugSink (only active when --debug-types)
type VerboseDebugSink struct {
    provenance map[TypeVarID][]TypeOrigin
}
```

Wire provenance updates only at key events:
- `freshTypeVar()` → "inferred" + span
- `inferLambda / inferFuncDecl` → "annotation"
- `defaultNum` → "defaulted to Int/Float"
- (optionally) unifying with known type from call

Then `typeReport` can:
1. Discover set of TVars in node's type
2. Look them up in provenance map
3. Aggregate into Origins slice

**Guard behind debug mode:** Only track when `--debug-types` is set. In non-debug mode, provenance map is nil (zero overhead).

### Implementation Plan

**Phase 1: Substitution Chain Tests** (~4 hours) - ✅ COMPLETE
- [x] Add `TestSubstitutionIdempotent` - property test with various type shapes
- [x] Add `TestSubstitutionChainResolution` - the exact M-FIX-FLOAT-OP bug
- [x] Add `TestSubstitutionLongChain` - deep chain resolution
- [x] Add `TestSubstitutionNoChain` - direct mappings
- [x] Add `TestSubstitutionInFunction` - chains in function types
- [x] Verified tests caught regression during M-FIX-RECORD-UPDATE (TDD validation)

**Phase 2: TypeReport Function** (~6 hours) - CANONICAL PRIMITIVE
- [ ] Create `TypeReport`, `ConstraintRef`, `TypeOrigin` types
- [ ] Implement `typeReport(nodeID)` as thin façade
- [ ] Implement `ApplyClosure()` for full substitution chain resolution
- [ ] Consolidate info from 4 data structures:
  - `InferenceContext.substitution`
  - `CoreTypeChecker.CoreTI`
  - `CoreTypeChecker.resolvedConstraints`
  - `InferenceContext.constraints`
- [ ] Add unit tests for typeReport

**Phase 3: --debug-types CLI** (~6 hours) - USER-FACING
- [ ] Create `TypeDebugSink` interface with NoOp and Verbose implementations
- [ ] Add `--debug-types` flag to CLI
- [ ] Implement `TypeDebugDumper` that formats typeReport output
- [ ] Show substitution map with chains (both raw and resolved forms)
- [ ] Show constraints before/after defaulting
- [ ] Highlight unresolved TVars as warnings
- [ ] Add `--debug-types --node <id>` for focused output

**Phase 4: Type Provenance** (~8 hours) - OPTIONAL ENHANCEMENT
- [ ] Add `map[TypeVarID][]TypeOrigin` to VerboseDebugSink
- [ ] Wire provenance tracking at `freshTypeVar()` calls
- [ ] Wire provenance tracking when applying annotations
- [ ] Wire provenance tracking during defaulting
- [ ] Integrate into typeReport output
- [ ] Ensure zero overhead when debug flag not set

### Files to Modify/Create

**New files:**
- `internal/types/debug_sink.go` - TypeDebugSink interface (~80 LOC)
- `internal/types/debug_verbose.go` - VerboseDebugSink implementation (~150 LOC)
- `internal/types/type_report.go` - TypeReport and typeReport() (~120 LOC)
- `internal/types/substitution_chain_test.go` - Chain/idempotency tests (~200 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add `--debug-types` flag (~15 LOC)
- `internal/types/unification_core.go` - Add ApplyClosure() (~30 LOC)
- `internal/types/typechecker_core.go` - Thread TypeDebugSink (~20 LOC)
- `internal/types/inference_context.go` - Thread TypeDebugSink (~20 LOC)

## Examples

### Example 1: Debug Types Output

**Before (current state):**
```bash
$ DEBUG_BINOP=1 ./bin/ailang run test.ail
# Have to add custom debug prints to every file
# Output is scattered and incomplete
```

**After (with --debug-types):**
```bash
$ ./bin/ailang run --debug-types test.ail

=== Type Inference Debug ===

[Substitution Map]
  α3 → α7 → float (CHAIN, final: float)
  α7 → float (direct)
  α12 → int (direct)

[Active Constraints] (before defaulting)
  Num α3 at line 5
  Fractional α7 at line 6
  Eq α12 at line 8

[Active Constraints] (after defaulting)
  Num float at line 5 (was α3, defaulted via chain)
  Fractional float at line 6 (was α7)
  Eq int at line 8 (was α12)

[CoreTI Entries]
  NodeID 42: float
    Raw: α7
    Resolved: float
    Origins:
      - Annotation: return type at examples/math_trig.ail:11
      - Defaulted: via Num → Fractional → float at defaulting pass
  NodeID 43: float
    Raw: float
    Resolved: float
    Origins:
      - Literal: 3.14 at line 6
  NodeID 44: float
    Raw: α3
    Resolved: float
    Origins:
      - Inferred: via unification with NodeID 43

[Warnings]
  None - all types resolved
```

### Example 2: Type Report Function

**Usage in debugging (programmatic API):**
```go
// When debugging why a node has the wrong type
report := tc.TypeReport(nodeID)
fmt.Printf("Node %d:\n", nodeID)
fmt.Printf("  Raw: %s\n", report.Raw)
fmt.Printf("  Resolved: %s\n", report.Resolved)
fmt.Printf("  Origins:\n")
for _, o := range report.Origins {
    fmt.Printf("    - %s: %s at %s\n", o.Kind, o.Note, o.Span)
}
fmt.Printf("  Constraints: %d\n", len(report.Constraints))
```

**Output:**
```
Node 42:
  Raw: α7
  Resolved: float
  Origins:
    - Annotation: parameter annotation x: float at examples/math_trig.ail:11
    - Defaulted: Num defaulted to float via Fractional at defaulting pass
  Constraints: 2
```

### Example 3: Future Workflow with These Tools

**Debugging "why is this operator int?" (previously 5 hours, now 20-30 minutes):**

```bash
# Step 1: Run with debug output
$ ailang run --debug-types examples/math_trig.ail

# Step 2: Find the NodeID of PI() / 4.0 (or see "suspect nodes" around that span)
# Output shows:
#   NodeID 87: int  ← SUSPECT (expected float)
#     Raw: α7
#     Resolved: int
#     Origins:
#       - Annotation: PI(): float at line 3
#       - Defaulted: via Num → Int (no Fractional constraint!)

# Step 3: Immediately see the problem:
# "no Fractional constraint" - the annotation wasn't propagating!

# Step 4: Look at substitution dump:
#   α7 → α12 → int (CHAIN)
# The chain wasn't being followed → found the bug.
```

**Time breakdown:**
- T+0min: Run with --debug-types
- T+5min: Find suspect node, see "defaulted via Num → Int"
- T+10min: Check substitution chains, see incomplete resolution
- T+20min: Identify root cause in substitution logic
- T+30min: Fix and verify

## Success Criteria

- [ ] Substitution tests catch chain bugs (TDD: write test, verify it fails on pre-fix code)
- [ ] `typeReport(nodeID)` returns consolidated type info from all 4 data structures
- [ ] `--debug-types` shows substitution map with chain resolution
- [ ] `--debug-types` shows constraints before/after defaulting
- [ ] TypeDebugSink interface keeps debug code orthogonal to core checker
- [ ] NoOpDebugSink has zero overhead (benchmark)
- [ ] Type provenance shows origins for each type (when debug enabled)
- [ ] AI tools can call TypeReport API in-process (no string parsing)
- [ ] All existing tests passing
- [ ] Documentation updated (CLAUDE.md debug flags section)
- [ ] Examples added to debug guide

## Testing Strategy

**Unit tests:**
- `TestSubstitutionIdempotent` - property test with random subs and type shapes
- `TestSubstitutionChainResolution` - the exact M-FIX-FLOAT-OP bug
- `TestSubstitutionCycleDetection` - recursive types don't infinite loop
- `TestTypeReportConsolidatesInfo` - all 4 data structures included
- `TestTypeReportResolvesFully` - no TVars in Resolved when sub has answer
- `TestNoOpDebugSinkZeroOverhead` - benchmark shows no perf impact

**Integration tests:**
- `TestDebugTypesFlag` - verify CLI flag works end-to-end
- `TestDebugTypesMathTrig` - verify output on real example
- `TestDebugTypesNodeFilter` - verify --node filtering works

**TDD Validation:**
- Run substitution tests against pre-M-FIX-FLOAT-OP code
- Confirm they fail as expected
- Confirm they pass on current code

## Non-Goals

**Not in this feature:**
- Visual debugger (IDE integration) - out of scope for CLI-first
- Interactive stepping through type inference - too complex for v0.5.10
- Automatic fix suggestions - requires more infrastructure
- Performance profiling of type inference - separate feature
- Full query interface (`--around-line`, `--constraints`) - future enhancement

## Timeline

**Day 1** (8 hours):
- Phase 1: Substitution chain tests (4 hours)
- Phase 2: TypeReport function - start (4 hours)

**Day 2** (8 hours):
- Phase 2: TypeReport function - complete (2 hours)
- Phase 3: --debug-types CLI (6 hours)

**Day 3** (8 hours):
- Phase 4: Type provenance tracking (6 hours)
- Documentation and integration testing (2 hours)

**Total: ~24 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Debug output too verbose | Med | Add `--node` filter; consider levels in future |
| Performance overhead from provenance | Low | Only track when debug flag enabled; benchmark NoOp |
| Breaking existing DEBUG_* env vars | Med | Keep existing env vars, new flag is additive |
| TypeDebugSink interface too broad | Med | Start minimal, expand based on actual needs |
| Debug code leaks into hot path | High | Interface + NoOp pattern enforces separation |

## References

- [M-FIX-FLOAT-OP Summary](../../implemented/v0_5_9/m-fix-float-op-summary.md) - The debugging session that identified these issues
- [Type Checker Architecture](../../implemented/v0_3_0/typechecker.md) - Current type system design
- [CLAUDE.md Debug Flags](../../../CLAUDE.md#debug-flags) - Existing debug infrastructure

## Future Work

- **Query interface**: `--around-line <file:line:col>`, `--constraints`, `--chains`
- **Interactive type debugger**: Step through inference, set breakpoints
- **Type inference visualization**: Graph of constraints and substitutions
- **Automatic bug detection**: Lint for common type inference pitfalls
- **AI-assisted debugging**: "Why is this type int?" natural language queries

## Design Review Feedback (Incorporated)

This design incorporates feedback from review:

1. **Reordered phases**: Substitution tests first (cheap, immediately useful), then typeReport (foundation), then CLI (orchestration), then provenance (optional enhancement)

2. **Orthogonal debug architecture**: TypeDebugSink interface keeps debug concerns out of core checker. No scattered `if debug { ... }` statements.

3. **TypeReport refinements**:
   - Raw vs Resolved naming (clear semantics)
   - Multiple Origins (slice, not single string) - real types have multiple provenance sources
   - ConstraintRef with source spans
   - Thin façade design - always derived from live data, never persisted

4. **Provenance as side-map**: Not bolted onto Type AST. Separate `map[TypeVarID][]TypeOrigin` in VerboseDebugSink. Zero overhead when disabled.

5. **API-first design**: Core checker exposes structured APIs (GetSubstitution, TypeReport, etc.). CLI is just formatting. AI tools can use same APIs in-process.

---

## DX Insights from M-FIX-RECORD-UPDATE (2025-12-10)

During the same session, a NEW regression was found and fixed (record update with type aliases). This revealed additional DX gaps:

### What Would Have Helped

1. **Alias Environment Debugging** - Had to add 20+ lines of ad-hoc `DEBUG_ALIAS` prints:
   - Where is aliasEnv populated? (Elaborator? TypeChecker?)
   - Is it being passed through the pipeline?
   - What aliases are actually registered?

   **Proposed tool:** `--debug-alias` flag or integrate into `--debug-types` output showing:
   - Aliases registered at elaboration
   - Aliases passed to type checker
   - Alias lookups during unification (hit/miss)

2. **Pipeline Tracing** - Couldn't see if module vs single-file pipeline was being used:
   - Debug statement in `pipeline_single.go` never fired
   - Module pipeline was the actual code path
   - Wasted time debugging wrong code path

   **Proposed tool:** `--debug-pipeline` showing which pipeline functions are called

3. **Unification Failure Context** - Error "cannot unify row with *types.TVar2" gave no context:
   - What node caused this?
   - What types were being unified?
   - What was the unification chain leading here?

   **Proposed tool:** Unification stack trace in error messages:
   ```
   Error: cannot unify row with *types.TVar2
     at unifyRecord2 (TRecord{pos: Pos} ~ TRecordOpen{pos: ...})
       at unify 'pos' field: Pos ~ {x: int, y: int}
         at expanding alias: Pos -> TRecord{x: int, y: int}
           at row unification: Row{x: int, y: int} ~ TVar2{ρ_empty}
   ```

4. **Type/AST Relationship** - Confusion about `*ast.RecordType` vs `*ast.TypeAlias`:
   - `type NPC = { pos: Pos }` parsed as RecordType, not TypeAlias
   - Caused alias registration to be skipped

   **Proposed tool:** `ailang debug parse file.ail --show-ast` with type declarations highlighted

### Bugs Found and Fixed

1. **M-FIX-RECORD-UPDATE Bug 1: Missing alias registration in module pipeline**
   - `pipeline_single.go` had alias registration, `pipeline_module.go` didn't
   - Fix: Added alias registration to module pipeline (lines 348-353)

2. **M-FIX-RECORD-UPDATE Bug 2: RecordType not registered as alias**
   - `elaborateTypeDecl` only registered `*ast.TypeAlias`, not `*ast.RecordType`
   - Fix: Added RecordType case to register named record types as aliases

3. **M-FIX-RECORD-UPDATE Bug 3: Row/TVar2 unification failure**
   - `case *Row` in Unify only handled `*Row` and `*RowVar`, not `*TVar2`
   - Fix: Added handling for `*TVar2` with row kind

### Debug Session Timeline

```
T+0h: "Record update fails: cannot unify NPC with TRecordOpen"
T+0.5h: Added DEBUG_ALIAS env var, found aliasEnv is nil
T+1h: Found module pipeline missing alias registration
T+1.5h: Fixed pipeline, still failing with Row/TVar2 error
T+2h: Added Row/TVar2 handling, all tests pass
```

**With proposed tools, this would have been:**
- T+5min: `--debug-types` shows aliasEnv state, finds it's empty
- T+10min: Check pipeline code path, find module pipeline
- T+15min: Add registration, see new error with full context
- T+20min: Fix Row/TVar2, done

### Prioritization Update

Based on today's session, recommend adding to Phase 3:
- **Alias debugging** - Show alias registration/lookup
- **Unification stack trace** - Context for "cannot unify X with Y" errors
- **Pipeline tracing** - Which code path is active

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10
**Feedback incorporated**: 2025-12-10

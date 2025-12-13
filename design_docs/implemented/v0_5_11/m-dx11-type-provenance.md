# M-DX11-TYPE-PROVENANCE: Type Origin Tracking

**Status**: Planned
**Target**: v0.5.11+
**Priority**: P3 - Low (Optional Enhancement)
**Estimated**: 8 hours
**Dependencies**: M-DX11-DEBUG-TYPES-CLI (v0.5.11)
**Parent**: M-DX11 (Type Inference Debugging Tools)

## Overview

Track where types originate from (annotations, literals, inference, defaulting) to answer "why does this have type X?" questions.

## Prerequisites

- ✅ Phase 1: Substitution chain tests (v0.5.10)
- ⏳ Phase 2: TypeReport function (v0.5.10)
- ⏳ Phase 3: --debug-types CLI (v0.5.11)

## Goals

- Answer "why is this type int instead of float?"
- Show type provenance chain in debug output
- Zero overhead when debug flag not set

## Solution Design

### Key Design Decision: Provenance as Side-Map

**Why not attach to Type tree:**
- Blows up memory
- Pollutes equality/hashing logic
- Easy to accidentally ship provenance into codegen

**Instead:**
```go
// Inside VerboseDebugSink (only active when --debug-types)
type VerboseDebugSink struct {
    provenance map[TypeVarID][]TypeOrigin
}
```

### TypeOrigin Structure

```go
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

### Provenance Tracking Points

Wire provenance updates only at key events:
- `freshTypeVar()` → "inferred" + span
- `inferLambda / inferFuncDecl` → "annotation"
- `defaultNum` → "defaulted to Int/Float"
- unifying with known type from call

### Example Output

```
NodeID 42: float
  Raw: α7
  Resolved: float
  Origins:
    - Annotation: return type at examples/math_trig.ail:11
    - Defaulted: via Num → Fractional → float at defaulting pass
```

### Implementation Plan

**Tasks (~8 hours):**
1. Add `map[TypeVarID][]TypeOrigin` to VerboseDebugSink
2. Wire provenance tracking at `freshTypeVar()` calls
3. Wire provenance tracking when applying annotations
4. Wire provenance tracking during defaulting
5. Integrate into TypeReport output
6. Ensure zero overhead when debug flag not set

### Files to Modify

**Modified files:**
- `internal/types/debug_verbose.go` (+100 LOC) - Provenance map and tracking
- `internal/types/type_report.go` (+30 LOC) - Include origins in report
- `internal/types/inference_context.go` (+20 LOC) - Hook provenance events
- `internal/types/typechecker_core.go` (+20 LOC) - Hook provenance events

## Success Criteria

- [ ] Type provenance shows origins for each type
- [ ] Origins include: annotations, literals, inference, defaulting
- [ ] Zero overhead when debug flag not set (benchmark)
- [ ] Integrates with --debug-types output

## Non-Goals

- Interactive type debugger (stepping through inference)
- Automatic fix suggestions
- Visual/graph representation

## References

- [M-DX11 Type Inference Debugging](../v0_5_10/m-dx11-type-inference-debugging.md) - Parent design doc

---

**Document created**: 2025-12-11

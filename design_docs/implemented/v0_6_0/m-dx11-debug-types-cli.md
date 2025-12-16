# M-DX11-DEBUG-TYPES-CLI: Type Inference Debug CLI

**Status**: Planned
**Target**: v0.5.11
**Priority**: P2 - Medium
**Estimated**: 6 hours
**Dependencies**: M-DX11-TYPE-REPORT (v0.5.10)
**Parent**: M-DX11 (Type Inference Debugging Tools)

## Overview

Add `--debug-types` CLI flag that provides formatted output of type inference state, building on the TypeReport API from v0.5.10.

## Prerequisites

- ✅ Phase 1: Substitution chain tests (v0.5.10)
- ⏳ Phase 2: TypeReport function (v0.5.10) - **Must complete first**

## Goals

- `ailang run --debug-types test.ail` shows full type inference state
- `ailang run --debug-types --node 42 test.ail` shows focused output for specific node
- Output includes substitution map, constraints, and CoreTI entries
- AI tools can parse structured output

## Solution Design

### TypeDebugSink Interface

Orthogonal debug architecture - no scattered `if debug { ... }`:

```go
type TypeDebugSink interface {
    OnFreshTypeVar(tv TypeVarID, span SourceSpan, origin OriginKind)
    OnUnify(left, right Type, result Type)
    OnSubstitute(tv TypeVarID, resolved Type)
    OnDefault(tv TypeVarID, defaulted Type, reason string)
    OnConstraintAdd(c Constraint)
    OnConstraintResolve(c Constraint, resolved Type)
}

type NoOpDebugSink struct{}  // Zero overhead in production
type VerboseDebugSink struct { events []DebugEvent }  // When --debug-types
```

### CLI Output Format

```bash
$ ailang run --debug-types test.ail

=== Type Inference Debug ===

[Substitution Map]
  α3 → α7 → float (CHAIN, final: float)
  α7 → float (direct)
  α12 → int (direct)

[Active Constraints] (before defaulting)
  Num α3 at line 5
  Fractional α7 at line 6

[CoreTI Entries]
  NodeID 42: float
    Raw: α7
    Resolved: float
  NodeID 43: float
    Raw: float
    Resolved: float
```

### Implementation Plan

**Tasks (~6 hours):**
1. Create `TypeDebugSink` interface with NoOp and Verbose implementations
2. Add `--debug-types` flag to CLI
3. Implement `TypeDebugDumper` that formats TypeReport output
4. Show substitution map with chains (raw and resolved)
5. Show constraints before/after defaulting
6. Add `--node <id>` filter option

### Files to Create/Modify

**New files:**
- `internal/types/debug_sink.go` - TypeDebugSink interface (~80 LOC)
- `internal/types/debug_verbose.go` - VerboseDebugSink (~150 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add `--debug-types` flag (~15 LOC)
- `cmd/ailang/debug.go` - Format and display output (~100 LOC)
- `internal/types/typechecker_core.go` - Thread TypeDebugSink (~20 LOC)

## Success Criteria

- [ ] `--debug-types` shows substitution map with chain resolution
- [ ] `--debug-types` shows constraints before/after defaulting
- [ ] `--debug-types --node <id>` filters to specific node
- [ ] NoOpDebugSink has zero overhead (benchmark)
- [ ] Output parseable by AI tools

## References

- [M-DX11 Type Inference Debugging](../v0_5_10/m-dx11-type-inference-debugging.md) - Parent design doc
- [M-DX11-TYPE-REPORT Sprint](../v0_5_10/m-dx11-type-report-sprint-plan.md) - Prerequisite

---

**Document created**: 2025-12-11

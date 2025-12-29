# M-DX11 Phase 2 Sprint Plan: Debug Event Emission

**Status:** Completed (v0.6.2)
**Priority:** Low (DX enhancement)
**Estimated Duration:** 1 day (~5 hours)
**Design Doc:** [m-dx11-phase2-debug-events.md](m-dx11-phase2-debug-events.md)

## Summary

Wire debug event emission throughout the type checker so `--debug-types` shows non-empty Substitution Map and Constraints sections. Currently these show "(empty)" because events aren't emitted during inference.

**Current state:**
```
[Substitution Map]
  (empty)           <- No OnSubstitute events

[Constraints]
  (no constraints)  <- No OnConstraintAdd/Resolve events
```

**After this sprint:**
```
[Substitution Map]
  α1 → int (direct)
  α22 → float (direct)

[Constraints]
  Added:
    Num α1 at node 9
  Resolved:
    Num int → add at node 9
```

## Prior Art Analysis

**Phase 1 (M-DX11-PROVENANCE) established:**
- ✅ `InferenceContext.debugSink` field exists
- ✅ `InferenceContext.SetDebugSink()` wired
- ✅ `OnFreshTypeVar()` already called in `freshTypeVarWithOrigin()`
- ✅ VerboseDebugSink records provenance

**Still needed (Phase 2):**
- ❌ Unifier doesn't have debugSink
- ❌ OnSubstitute not called during unification
- ❌ OnUnify not called during unification
- ❌ OnConstraintAdd not called when constraints added
- ❌ OnConstraintResolve not called when constraints resolved

## Velocity Context

Recent velocity from M-DX11-PROVENANCE sprint:
- M1: +184 LOC (provenance infrastructure)
- M2: +26 LOC (wire to inference)
- M3: +168 LOC (TypeReport integration)
- M4: +55 LOC (benchmarks)
- **Total:** +433 LOC in ~4 hours

**Target for Phase 2:** ~200 LOC in ~5 hours (conservative, new territory)

## Milestones

### M1: Wire DebugSink to Unifier (~80 LOC, 1.5h)

**Goal:** Pass debugSink from CoreTypeChecker → InferenceContext → Unifier

**Files:**
- `internal/types/unification_core.go` - Add debugSink field to Unifier struct
- `internal/types/typechecker_core.go` - Pass sink when creating Unifier

**Implementation:**
```go
// unification_core.go
type Unifier struct {
    rowUnifier    *RowUnifier
    depth         int
    rowVarCounter int
    aliasEnv      map[string]Type
    debugSink     TypeDebugSink  // NEW
}

func NewUnifier() *Unifier {
    return &Unifier{
        rowUnifier: NewRowUnifier(),
        aliasEnv:   make(map[string]Type),
        debugSink:  NoOpDebugSink{},  // Default
    }
}

func (u *Unifier) SetDebugSink(sink TypeDebugSink) {
    u.debugSink = sink
}
```

**Acceptance Criteria:**
- [ ] Unifier has debugSink field
- [ ] SetDebugSink method exists
- [ ] debugSink passed from CoreTypeChecker
- [ ] All tests passing
- [ ] Linting clean

### M2: Emit OnSubstitute Events (~40 LOC, 1h)

**Goal:** Emit OnSubstitute when type variables are bound

**Files:**
- `internal/types/unification_core.go` - Add OnSubstitute calls
- `internal/types/debug_sink_test.go` - Add test

**Implementation:**
```go
// In Unify() or bind() function
func (u *Unifier) bind(tv *TVar2, t Type, sub Substitution) Substitution {
    newSub := sub.Extend(tv.Name, t)
    if u.debugSink != nil {
        u.debugSink.OnSubstitute(tv, t)
    }
    return newSub
}
```

**Acceptance Criteria:**
- [ ] OnSubstitute called when binding type variables
- [ ] Events appear in VerboseDebugSink.Events()
- [ ] Substitution Map section in --debug-types shows mappings
- [ ] All tests passing

### M3: Emit OnConstraintAdd/Resolve Events (~60 LOC, 1.5h)

**Goal:** Track when constraints are added and resolved

**Files:**
- `internal/types/typechecker_core.go` - Add event emission
- `internal/types/inference.go` - Track constraint addition
- `internal/types/debug_sink_test.go` - Add tests

**Implementation:**
```go
// When adding constraints (inference.go or typechecker_core.go)
func (ctx *InferenceContext) addConstraint(className string, ty Type, nodeID uint64) {
    ctx.qualifiedConstraints = append(ctx.qualifiedConstraints, ClassConstraint{
        Class: className,
        Type:  ty,
    })
    ctx.debugSink.OnConstraintAdd(className, ty, nodeID)
}

// When resolving constraints (typechecker_core.go)
func (tc *CoreTypeChecker) resolveConstraint(c ClassConstraint, method string, nodeID uint64) {
    tc.resolvedConstraints[nodeID] = &ResolvedConstraint{
        NodeID:    nodeID,
        ClassName: c.Class,
        Type:      c.Type,
        Method:    method,
    }
    tc.DebugSink.OnConstraintResolve(c.Class, c.Type, method, nodeID)
}
```

**Acceptance Criteria:**
- [ ] OnConstraintAdd called when Num/Eq/Ord constraints added
- [ ] OnConstraintResolve called when constraints resolved
- [ ] Constraints section in --debug-types shows add/resolve events
- [ ] All tests passing

### M4: Integration Test & Demo Update (~20 LOC, 0.5h)

**Goal:** Verify full integration, update demo file

**Files:**
- `examples/debug_types_demo.ail` - Update expected output in comments
- `internal/types/debug_sink_test.go` - Add integration test

**Implementation:**
```go
func TestDebugSinkFullIntegration(t *testing.T) {
    sink := NewVerboseDebugSink()
    tc := NewCoreTypeChecker()
    tc.SetDebugSink(sink)

    // Type check: let x = 1 + 2
    // Should emit:
    // - OnFreshTypeVar for x
    // - OnConstraintAdd for Num
    // - OnSubstitute when resolved
    // - OnConstraintResolve for Num → add
}
```

**Acceptance Criteria:**
- [ ] Integration test covers full event flow
- [ ] `ailang run --debug-types examples/debug_types_demo.ail` shows non-empty sections
- [ ] Demo file comments updated to match actual output
- [ ] Documentation at `ailang debug types` still accurate

## Risk Assessment

**Low Risk:** This is additive work - adding event emission doesn't change behavior.

**Potential Issues:**
1. **Unifier creation paths:** May need to wire debugSink through multiple call sites
2. **Constraint tracking:** Need to find where constraints are actually added (scattered)
3. **Node ID tracking:** Need to ensure nodeID is available at emit sites

**Mitigations:**
- M1 handles the wiring, subsequent milestones just add calls
- grep for `qualifiedConstraints` and `Constraint{` to find all addition sites
- InferenceContext already tracks currentNodeID from Phase 1

## Success Metrics

- [ ] `--debug-types` shows non-empty Substitution Map for `let x = 1 + 2`
- [ ] `--debug-types` shows Constraints Added/Resolved for numeric operations
- [ ] Zero performance regression (benchmarks unchanged)
- [ ] All existing tests pass
- [ ] `ailang debug types` help text still accurate

## Dependencies

- **M-DX11-PROVENANCE (v0.5.11):** Completed - provides infrastructure
- **No blocking dependencies for Phase 2**

## Out of Scope

- Emitting OnUnify events (adds noise, questionable value)
- Source span tracking for constraints (would require parser changes)
- --debug-types output format changes (keep current format)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Events deterministically trace inference steps |
| A2: Replayability | +1 | Event trace enables debugging session replay |
| A3: Effect Legibility | +1 | Internal inference effects made visible |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Per-node event scoping |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured sink pattern for programmatic access |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost tracking impact |
| A10: Composability | +1 | Extends Phase 1 infrastructure |
| A11: Structured Failure | +1 | Helps diagnose type errors |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Same input produces same events
- [x] A3 (Effects): Sink pattern makes observation explicit
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Events designed for machine consumption

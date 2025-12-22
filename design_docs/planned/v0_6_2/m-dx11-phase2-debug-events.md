# M-DX11 Phase 2: Type Checker Debug Event Emission

**Status:** Planned
**Priority:** Low (DX enhancement)
**Depends On:** M-DX11 (completed in v0.5.11)
**Estimated Effort:** 4-6 hours

## Problem Statement

The `--debug-types` CLI flag (M-DX11) provides infrastructure for type inference debugging, but the Substitution Map and Constraints sections are currently empty because the type checker doesn't emit events to the debug sink.

**Current state:**
```bash
$ ailang run --debug-types examples/debug_types_demo.ail
=== Type Inference Debug ===

[Substitution Map]
  (empty)           # <- No events emitted

[Constraints]
  (no constraints)  # <- No events emitted

[CoreTI Entries]
  NodeID 60: float
    Constraint: Fractional (resolved)  # <- Works! Read from CoreTI
```

The CoreTI section works because it reads directly from `tc.CoreTI`, but the substitution map and constraint events require the type checker to call sink methods.

## Proposed Solution

Add debug event emission to the type checker at key points:

### 1. Fresh Type Variable Creation

**Location:** `internal/types/typechecker_core.go` and `internal/types/unification_core.go`

```go
// When creating a fresh type variable
func (tc *CoreTypeChecker) freshTypeVar(origin OriginKind) *TVar2 {
    tv := &TVar2{Name: tc.nextVarName(), Kind: KStar{}}
    tc.DebugSink.OnFreshTypeVar(tv, tc.currentNodeID, origin)
    return tv
}
```

### 2. Unification Events

**Location:** `internal/types/unification_core.go`

```go
func (u *Unifier) Unify(t1, t2 Type, sub Substitution) (Substitution, error) {
    result, err := u.unifyInternal(t1, t2, sub)
    if err == nil && u.debugSink != nil {
        u.debugSink.OnUnify(t1, t2, result, u.currentNodeID)
    }
    return result, err
}
```

### 3. Substitution Events

**Location:** `internal/types/substitution.go` or within unification

```go
// When a type variable is bound to a type
func (u *Unifier) bindVar(tv *TVar2, t Type, sub Substitution) Substitution {
    newSub := sub.Extend(tv.Name, t)
    if u.debugSink != nil {
        u.debugSink.OnSubstitute(tv, t)
    }
    return newSub
}
```

### 4. Constraint Events

**Location:** `internal/types/typechecker_core.go`

```go
// When adding a constraint
func (tc *CoreTypeChecker) addConstraint(className string, ty Type, nodeID uint64) {
    tc.constraints = append(tc.constraints, Constraint{Class: className, Type: ty})
    tc.DebugSink.OnConstraintAdd(className, ty, nodeID)
}

// When resolving a constraint
func (tc *CoreTypeChecker) resolveConstraint(c Constraint, method string, nodeID uint64) {
    tc.resolvedConstraints[c] = method
    tc.DebugSink.OnConstraintResolve(c.Class, c.Type, method, nodeID)
}
```

## Architecture Changes

### Pass DebugSink to Unifier

The `Unifier` struct needs access to the debug sink:

```go
type Unifier struct {
    typeAliases  map[string]Type
    debugSink    TypeDebugSink  // NEW
    currentNodeID uint64        // NEW
}

func NewUnifier() *Unifier {
    return &Unifier{
        typeAliases: make(map[string]Type),
        debugSink:   NoOpDebugSink{},
    }
}

func (u *Unifier) SetDebugSink(sink TypeDebugSink) {
    u.debugSink = sink
}
```

### Track Current Node ID

To associate events with AST nodes, track the current node being processed:

```go
func (tc *CoreTypeChecker) InferWithConstraints(expr core.CoreExpr, env *TypeEnv) (...) {
    tc.currentNodeID = expr.ID()  // Set before inference
    // ... inference logic ...
}
```

## Expected Output After Implementation

```bash
$ ailang run --debug-types examples/debug_types_demo.ail
=== Type Inference Debug ===

[Substitution Map]
  α1 → int (direct)
  α2 → int (direct)
  α22 → int → int (chain: α22 → α23 → int → int)

[Constraints]
  Added:
    Num α1 at node 9
    Num α2 at node 14
    Fractional α5 at node 60
  Resolved:
    Num int → add at node 9
    Num int → mul at node 14
    Fractional float (resolved) at node 60

[CoreTI Entries]
  NodeID 9: int
    Constraint: Num → add
  NodeID 60: float
    Constraint: Fractional (resolved)
  ...
```

## Files to Modify

| File | Changes |
|------|---------|
| `internal/types/typechecker_core.go` | Add event emission for constraints, pass sink to unifier |
| `internal/types/unification_core.go` | Add DebugSink field, emit OnUnify/OnSubstitute events |
| `internal/types/unification_types.go` | Pass through debug sink in type-specific unification |
| `internal/types/substitution.go` | Optionally emit OnSubstitute when extending |

## Milestones

### M1: Unifier Debug Integration (2h)
- Add DebugSink field to Unifier
- Pass sink from CoreTypeChecker to Unifier
- Emit OnUnify events in Unify()

### M2: Substitution Events (1h)
- Emit OnSubstitute when binding type variables
- Track substitution chains for debugging

### M3: Constraint Events (1.5h)
- Emit OnConstraintAdd when constraints are added
- Emit OnConstraintResolve when constraints are resolved
- Track node IDs for constraint association

### M4: Fresh Type Variable Events (0.5h)
- Emit OnFreshTypeVar with origin information
- Track where type variables are created (annotation, literal, inferred)

## Testing

```go
func TestDebugSinkIntegration(t *testing.T) {
    sink := types.NewVerboseDebugSink()
    tc := types.NewCoreTypeChecker()
    tc.SetDebugSink(sink)

    // Infer type of: let x = 1 + 2
    // ... setup ...

    events := sink.Events()

    // Should have fresh type var events
    assert.Contains(t, eventKinds(events), types.EventFreshTypeVar)

    // Should have unify events
    assert.Contains(t, eventKinds(events), types.EventUnify)

    // Should have constraint events
    assert.Contains(t, eventKinds(events), types.EventConstraintAdd)
    assert.Contains(t, eventKinds(events), types.EventConstraintResolve)
}
```

## Alternatives Considered

### 1. Read from TypeChecker Internal State

Instead of emitting events, read substitution/constraint state directly from TypeChecker fields.

**Rejected:** The internal state is consumed/transformed during inference. Events capture the process, not just the final state.

### 2. Add Logging Instead of Sink Pattern

Use traditional `if debug { log(...) }` statements.

**Rejected:** M-DX11 specifically chose the sink pattern for zero-overhead in production. Logging would add overhead even when disabled.

## Success Criteria

- [ ] `--debug-types` shows non-empty substitution map for polymorphic code
- [ ] Constraints section shows add/resolve events
- [ ] Zero performance regression when `--debug-types` is not used
- [ ] All existing tests pass
- [ ] Demo example shows meaningful output in all sections

## Related

- **M-DX11:** Initial `--debug-types` infrastructure (v0.5.11)
- **Design Doc:** [m-dx11-debug-types-cli.md](../v0_5_11/m-dx11-debug-types-cli.md)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Debug events provide deterministic introspection into inference |
| A2: Replayability | +1 | Full event trace enables debugging session replay |
| A3: Effect Legibility | +1 | Makes type inference internal effects visible |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Events scoped to individual nodes enable local debugging |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Structured events suitable for automated analysis |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost tracking impact |
| A10: Composability | +1 | Extends existing M-DX11 infrastructure |
| A11: Structured Failure | +1 | Events help diagnose type inference failures |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Events are deterministic given same input
- [x] A3 (Effects): No hidden side effects (sink pattern is explicit)
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Events designed for machine consumption

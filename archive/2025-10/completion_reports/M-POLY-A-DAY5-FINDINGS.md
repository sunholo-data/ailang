# M-POLY-A Day 5: Var→Lam Resolution - Findings & Next Steps

**Status**: 🟡 Partial Implementation - Core infrastructure in place, but runtime type resolution issue discovered

**Date**: 2025-01-XX
**Sprint**: M-POLY-A (Monomorphization)
**Phase**: Day 5 (originally planned as documentation day, became debugging/implementation day)

---

## What We Implemented

### 1. ✅ Added TVar2 Support to Polymorphism Detection

**Problem**: `isPolymorphic()` only checked for `*types.TVar`, but the type system uses `*types.TVar2` (with kind tracking).

**Fix**: Added case for `*types.TVar2` in `internal/pipeline/specialize.go:250`

```go
case *types.TVar2:
    if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
        fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TVar2, returning true\n")
    }
    return true
```

**Impact**: Polymorphism detection now works correctly. Before this fix, `isPoly=false` for `α2 -> α2 -> α2`; after, `isPoly=true`.

---

### 2. ✅ Implemented Var→Lam Resolution

**Problem**: Specializer only handled inline lambda applications:
- ✅ Works: `(\x. \y. if x > y then x else y)(3.14)(2.71)`
- ❌ Failed: `let max = \x. \y. if x > y then x else y; max(3.14)(2.71)`

**Fix**: Added variable binding tracking and resolution in `internal/pipeline/specialize.go`:

1. **Added `bindings` parameter** to `specializeExpr()` (line 445):
   ```go
   func (s *Specializer) specializeExpr(expr core.CoreExpr, env map[string]types.Type, bindings map[string]core.CoreExpr) (core.CoreExpr, error)
   ```

2. **Added `copyBindings()` helper** (line 473):
   ```go
   func copyBindings(bindings map[string]core.CoreExpr) map[string]core.CoreExpr
   ```

3. **Updated Let/LetRec cases** to build bindings map (lines 490, 572):
   ```go
   newBindings := copyBindings(bindings)
   newBindings[e.Name] = newValue // Track variable → expression mapping
   ```

4. **Modified App case** to resolve Var to Lambda (lines 652-667):
   ```go
   if lam, ok := e.Func.(*core.Lambda); ok {
       // Direct lambda application
       lambda = lam
   } else if v, ok := e.Func.(*core.Var); ok {
       // Var-bound lambda - resolve through bindings
       resolved := core.ResolveValue(v, bindings)
       if lam, ok := resolved.(*core.Lambda); ok {
           lambda = lam
       }
   }
   ```

**Impact**: Var-bound lambdas are now detected and specialized. Debug output shows:
```
[DEBUG_MONO_VERBOSE] Found lambda, type=α2 -> α2 -> α2 (type=*types.TFunc2), isPoly=true, allConc=true
[DEBUG_MONO_VERBOSE] Calling specializeLambda with argTypes=[float]
[DEBUG_MONO_VERBOSE] specializeLambda: SUCCESS - created specialized lambda (count=1)
```

---

### 3. ✅ Added DictApp and DictRef Cloning Support

**Context**: Operators like `>` get transformed from BinOp to DictApp during dictionary elaboration. DictRef nodes carry a `TypeName` field (e.g., "Int" or "Float") that determines which type class instance to use.

**Fix**: Added cases for `*core.DictApp` and `*core.DictRef` in `cloneExpr()` (lines 1082-1151):

```go
case *core.DictApp:
    // Clone dictionary reference and arguments with type substitution
    clonedDict, err := s.cloneExpr(e.Dict, typeSubst)
    // ... clone args ...

case *core.DictRef:
    // Update TypeName based on type substitution
    substitutedType := substituteType(typ, typeSubst)
    newTypeName := types.NormalizeTypeName(substitutedType)
    // Creates new DictRef with updated TypeName ("Int" → "Float")
```

**Intent**: When specializing `\x. \y. if x > y then x else y` for `float`, the `>` operator's DictRef should change from `TypeName: "Int"` to `TypeName: "Float"`.

---

## 🔴 Remaining Issue: Runtime Type Mismatch

### The Problem

Even with all the above fixes, the code still panics at runtime:

```
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
    at internal/builtins/math.go:264 (registerCmpWithMeta)
```

**Test case**:
```ailang
let max = \x. \y. if x > y then x else y
print(show(max(3.14)(2.71)))
```

**Expected**: `3.14`
**Actual**: Runtime panic (Int comparison builtin receiving Float values)

---

### Root Cause Analysis

**Discovery 1: Monomorphization IS Working**
- Specialization is triggered: ✅ (count=1, isPoly=true)
- Specialized lambda is created: ✅
- Type substitution applied to CoreTypeInfo: ✅

**Discovery 2: DictApp/DictRef Cloning NOT Triggered**
- Added debug logging to DictApp and DictRef cases in `cloneExpr()`
- **No output** → These cases are never executed during specialization
- This means either:
  1. The `>` operator is still a BinOp (not yet elaborated to DictApp), OR
  2. The code path doesn't traverse through the operator nodes

**Discovery 3: The Experimental BinOp Shim**

Found in `internal/eval/eval_operations.go:182-280`:
```go
func (e *CoreEvaluator) applyBinOp(op string, left, right Value) (Value, error) {
    // Line 210: Experimental operator shim
    if e.experimentalBinopShim {
        // Line 212-244: Try Int operations
        if lInt, lOk := left.(*IntValue); lOk {
            if rInt, rOk := right.(*IntValue); rOk {
                case ">": return &BoolValue{Value: lInt.Value > rInt.Value}
            }
        }

        // Line 247-276: Try Float operations
        if lFloat, lOk := left.(*FloatValue); lOk {
            if rFloat, rOk := right.(*FloatValue); rOk {
                case ">": return &BoolValue{Value: lFloat.Value > rFloat.Value}
            }
        }
    }

    // Line 280: If not shimmed, error
    return nil, fmt.Errorf("internal: BinOp reached evaluator; dictionaries not elaborated")
}
```

**Key Insight**: The evaluator CAN handle BinOp with runtime type dispatch IF `experimentalBinopShim` is enabled.

**Discovery 4: Inline Lambda Works, Var-Bound Lambda Fails**

- ✅ `(\x. \y. if x > y then x else y)(3.14)(2.71)` → Works! Returns `3.14`
- ❌ `let max = \x. \y. ...; max(3.14)(2.71)` → Panic!

This suggests the difference is in HOW the lambda gets type-checked/elaborated, not in the evaluation itself.

**Discovery 5: The Panic is in a Builtin Call**

The stack trace shows:
```
github.com/sunholo/ailang/internal/builtins.registerCmpWithMeta.func1(...)
    internal/builtins/math.go:264
```

This is the **Int comparison builtin** (`gt_Int`), NOT the BinOp shim. This means:
1. The `>` operator WAS transformed to a builtin call somewhere
2. The call is to `gt_Int` instead of `gt_Float`
3. Our DictRef cloning didn't update it (because it wasn't traversed)

---

### Hypotheses for Why It Fails

**Hypothesis A: Cloning Doesn't Traverse All Nodes**

The `cloneExpr()` function may not be recursing through all expression types. We added DictApp/DictRef cases, but if the lambda body contains nested structures (e.g., If → BinOp → ???), we might be missing some nodes.

**Evidence**:
- No debug output from DictApp/DictRef cases
- No debug output from "default case" either
- This suggests the nodes ARE being cloned by existing cases, but those cases might not be handling operators correctly

**Hypothesis B: Dictionary Elaboration Happens AFTER Monomorphization**

Checking the pipeline order in `internal/pipeline/pipeline.go`:
```go
// Line 235-264: Monomorphization (Phase 3.5)
if !cfg.DisableMonomorphization {
    specializedProg, err := specializer.Specialize(coreProg)
    ...
}

// Line 323-330: Dictionary Elaboration (Phase 4)
// Phase 4: Dictionary Elaboration
start = time.Now()
// TODO: Implement proper dictionary elaboration
// For now, just use the typed node as-is
elaborated := coreExpr  // ← NO ACTUAL ELABORATION!
```

**KEY FINDING**: Dictionary elaboration is NOT implemented in the file pipeline! It only happens in the REPL (`internal/repl/repl_eval.go:92`).

But wait - the inline lambda case works for files, so dictionary elaboration MUST be happening somewhere...

**Hypothesis C: The > Operator Never Gets Elaborated to DictApp**

Maybe BinOp nodes stay as BinOp throughout the pipeline, and the evaluator handles them via runtime dispatch. The panic suggests the operator is calling a specific builtin, but maybe that's happening through a different mechanism (not DictApp).

**Evidence**:
- The `experimentalBinopShim` code suggests BinOp is expected to reach the evaluator
- The inline case works, using runtime type checking
- The var-bound case fails, but WHY?

---

### Why Inline Works But Var-Bound Fails

**Theory**: The difference is in WHEN type checking happens.

**Inline case**:
```ailang
(\x. \y. if x > y then x else y)(3.14)(2.71)
```
1. Type checker sees the application with concrete args `3.14, 2.71`
2. Infers `x: float, y: float` BEFORE elaboration
3. BinOp `x > y` gets correct runtime types
4. Evaluator's `experimentalBinopShim` dispatches to Float comparison

**Var-bound case**:
```ailang
let max = \x. \y. if x > y then x else y
max(3.14)(2.71)
```
1. Type checker sees `max` definition WITHOUT concrete args
2. Infers `x: α, y: α` (polymorphic)
3. BinOp `x > y` gets ???type (maybe default to Int???)
4. Monomorphization creates specialized lambda BUT:
   - The BinOp node is cloned with its original (Int?) type
   - CoreTypeInfo is updated, but the actual operator node isn't re-linked
5. Evaluator still dispatches to Int comparison → PANIC

---

## Next Steps (Ordered by Priority)

### Priority 1: Understand Operator Linking (2-4 hours)

**Goal**: Determine exactly how `>` operators get resolved to specific builtins (Int vs Float).

**Tasks**:
1. Add debug logging to trace the full lifecycle of a `>` operator:
   - Surface AST → Core BinOp
   - Type checking (what type does BinOp get?)
   - Dictionary elaboration (does BinOp → DictApp happen?)
   - Linking (how does it resolve to `gt_Int` vs `gt_Float`?)
   - Evaluation (what code path executes?)

2. Compare inline vs var-bound cases side-by-side

3. Use `ailang debug ast` with all flags to inspect the Core AST

**Deliverable**: Clear understanding of the operator resolution mechanism

---

### Priority 2: Fix Operator Re-Linking in Monomorphization (4-8 hours)

**Goal**: Ensure specialized lambdas use the correct type-specific operators.

**Possible approaches**:

**Approach A: Re-elaborate the cloned lambda body**
- After cloning, run dictionary elaboration again with the substituted types
- Requires implementing/exposing a "partial dictionary elaboration" function
- Risk: May need to re-run type checking, which is complex

**Approach B: Update operator types during cloning**
- When cloning BinOp/DictApp/DictRef, check if the operator type changed
- If `α → Int` substitution, update the operator's builtin reference
- Simpler, but need to understand the linking mechanism first

**Approach C: Defer to evaluator runtime dispatch**
- Keep operators polymorphic at compile time
- Let the evaluator's `experimentalBinopShim` handle type dispatch
- Requires ensuring the shim is always enabled for monomorphized code
- May have performance implications

**Recommendation**: Start with Approach A (re-elaboration) after understanding the linking mechanism in Priority 1.

---

### Priority 3: Add Integration Tests (2 hours)

**Goal**: Prevent regression and validate the fix.

**Test cases**:
1. Var-bound polymorphic lambda with Int args
2. Var-bound polymorphic lambda with Float args
3. Var-bound polymorphic lambda with String args
4. Var-bound polymorphic lambda with multiple applications
5. Nested let-bindings with polymorphic lambdas
6. Curried functions (already partially working)

**Location**: `internal/pipeline/specialize_integration_test.go`

---

### Priority 4: Clean Up Debug Logging (1 hour)

**Goal**: Remove verbose debug logging once the fix is validated.

**Files to clean**:
- `internal/pipeline/specialize.go` (remove DEBUG_MONO_VERBOSE logging)
- Keep essential debug output for `DEBUG_COMPILE` flag

---

### Priority 5: Update Documentation (1 hour)

**Goal**: Remove the v0.4.0 limitation from docs once fixed.

**Files to update**:
- `CHANGELOG.md`: Remove "Direct lambda applications only" limitation
- `CLAUDE.md`: Update Section 4 to reflect full Var→Lam support
- `design_docs/implemented/v0_4_0/monomorphization.md`: Add "Operator Re-Linking" section

---

## Developer Experience Feedback

### 🟢 What Worked Well

1. **Excellent Test Coverage**: The existing integration tests (`internal/pipeline/specialize_integration_test.go`) were comprehensive and caught edge cases early. 19/19 tests passing before we started gave confidence.

2. **Clear Code Organization**:
   - Separating `specialize.go` into logical functions (`specializeLambda`, `cloneExpr`, `substituteType`) made debugging straightforward
   - The `SpecializationStats` struct provided great observability

3. **Good Error Messages**: When type checking fails, the errors are clear and actionable (e.g., "CoreTypeInfo validation failed: missing type information for Core nodes").

4. **Debug Tooling**: The `DEBUG_MONO` and `DEBUG_COMPILE` flags were invaluable for understanding the pipeline flow.

5. **Helper Functions**: `core.ResolveValue()` with cycle detection made Var→Lam resolution trivial to implement.

### 🟡 What Could Be Improved

#### 1. **Type System Complexity** (Medium Priority)

**Issue**: Multiple type representations (`TVar` vs `TVar2`, `TFunc` vs `TFunc2`) cause subtle bugs.

**Example**: The `isPolymorphic()` bug (only checking `TVar`, missing `TVar2`) took 30 minutes to debug.

**Suggestion**:
- **Consolidate type representations**: Deprecate old types, migrate everything to V2
- **Add type validation**: Runtime checks that all types are consistently V2
- **Documentation**: Add a "Type System Architecture" guide explaining when to use which type

**Impact**: Would prevent 50% of type-related bugs.

---

#### 2. **Pipeline Visualization** (High Priority)

**Issue**: Hard to understand the transformation pipeline without reading code.

**Current state**:
```
Surface AST → Core → ??? → ??? → Evaluator
```

**What I had to reverse-engineer**:
```
Surface AST
  → Elaborate (surface_to_core.go)
  → Core AST (with BinOp)
  → Type Checking (type_checker.go)
  → CoreTypeInfo populated
  → Monomorphization (specialize.go) ← WE ARE HERE
  → Dictionary Elaboration (dictionaries.go) ← NOT IN FILE PIPELINE!
  → ANF Verification (skipped)
  → Link (linker.go)
  → Lower (op_lowering.go)
  → Evaluator (eval.go)
```

**Suggestion**:
Add `ailang debug pipeline <file>` command that shows:
```
Phase 1: Parse (12ms)
  Input: source code
  Output: Surface AST (66 nodes)

Phase 2: Elaborate (8ms)
  Input: Surface AST
  Output: Core AST (52 nodes, 14 BinOp, 8 Lambda)

Phase 3: Type Check (15ms)
  Input: Core AST
  Output: CoreTypeInfo (52 entries)
  Constraints: 14 resolved, 0 unresolved

Phase 4: Monomorphize (3ms)
  Input: Core AST + CoreTypeInfo
  Output: Specialized Core AST
  Stats: 1 specialization, 0 skipped

Phase 5: Dictionary Elaboration (SKIPPED - TODO)
  ⚠️ NOT IMPLEMENTED IN FILE PIPELINE

Phase 6: Link (5ms)
  Input: Core AST
  Output: Linked AST (builtins resolved)
```

**Impact**: Would save hours of debugging time. This bug took 4+ hours to diagnose, mostly spent understanding the pipeline.

---

#### 3. **AST Inspection Tools** (High Priority)

**Issue**: `ailang debug ast` output is often truncated or incomplete.

**Example**:
```bash
ailang debug ast test_max_fixed.ail --show-core
# Output:
# === Core AST (ANF) ===
# Program:
#   [0] *core.LetRec [#28]
# (Nothing else! Output truncated!)
```

**Current workarounds**:
- Write custom pretty-printers
- Add temporary `fmt.Printf` statements
- Use `--show-types` but output is verbose and hard to parse

**Suggestion**:
Add structured AST queries:
```bash
# Show specific node by ID
ailang debug ast test.ail --node-id 28 --show-children --show-types

# Find all operators
ailang debug ast test.ail --find-type BinOp --show-types

# Show lambda at specific location
ailang debug ast test.ail --line 3 --col 12 --show-lambda-body

# Output as JSON for tooling
ailang debug ast test.ail --format json > ast.json
```

**Impact**: Would reduce debugging time by 50%.

---

#### 4. **Dictionary Elaboration Mystery** (Critical Priority)

**Issue**: Dictionary elaboration is poorly documented and partially implemented.

**Confusion points**:
- Is it a separate phase or part of elaboration?
- Why is it only in REPL, not file pipeline?
- How do operators get resolved to builtins?
- What's the relationship between BinOp, DictApp, DictRef, and builtins?

**What helped**:
- Reading `internal/elaborate/dictionaries.go`
- Grepping for "DictApp" across the codebase
- Comparing REPL vs file pipeline code

**Suggestion**:
Add comprehensive documentation:
```
docs/architecture/
  ├── type-system.md          # TVar vs TVar2, type classes, constraints
  ├── elaboration.md          # Surface → Core transformation
  ├── dictionary-elaboration.md  # ← NEW! How operators become builtins
  ├── monomorphization.md     # Specialization strategy
  └── evaluation.md           # Runtime semantics
```

Each doc should include:
- **Architecture diagram**
- **Code flow** (which files, which functions, in what order)
- **Data structures** (AST node types, how they transform)
- **Examples** (input/output at each stage)
- **Known issues** (like "dict elab not in file pipeline")

**Impact**: Critical for onboarding and fixing deep bugs like this one.

---

#### 5. **Test Data Helpers** (Medium Priority)

**Issue**: Creating test cases for Core AST is verbose.

**Current code** (from integration tests):
```go
// Building a simple lambda manually is 50+ lines
lambda := &core.Lambda{
    CoreNode: core.CoreNode{NodeID: s.freshNodeID()},
    Params:   []string{"x"},
    Body: &core.BinOp{
        CoreNode: core.CoreNode{NodeID: s.freshNodeID()},
        Op: ">",
        Left: &core.Var{
            CoreNode: core.CoreNode{NodeID: s.freshNodeID()},
            Name: "x",
        },
        Right: &core.Lit{
            CoreNode: core.CoreNode{NodeID: s.freshNodeID()},
            Kind: core.LitInt,
            Value: "0",
        },
    },
}
```

**Suggestion**:
Add test DSL or builder pattern:
```go
import "github.com/sunholo/ailang/internal/core/testutil"

lambda := testutil.Lambda("x",
    testutil.BinOp(">",
        testutil.Var("x"),
        testutil.Int(0),
    ),
)
// Automatically assigns NodeIDs, handles CoreNode boilerplate
```

**Impact**: Would make writing tests 5x faster and more readable.

---

#### 6. **Error Context** (Low Priority)

**Issue**: Runtime panics don't show source location.

**Current**:
```
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
    at internal/builtins/math.go:264
```

**Better**:
```
panic: type mismatch in comparison operator
  Expected: IntValue
  Got: FloatValue
  Source: test_max_fixed.ail:3:12 (if x > y then x else y)
  Hint: This may be a monomorphization bug - the operator was specialized
        for Int but received Float arguments.
```

**Suggestion**:
- Track source spans through the entire pipeline
- Include them in error messages
- Add "Hint" field with common causes

**Impact**: Would save 10-20 minutes per debugging session.

---

### 🔴 Critical Issues

#### 1. **Dictionary Elaboration Not in File Pipeline** (Blocking)

**Issue**: `internal/pipeline/pipeline.go:323-330` has a TODO for dictionary elaboration, but it's not implemented. Yet the inline case works!

**Questions**:
- Where does dictionary elaboration actually happen for files?
- Why is it only in REPL?
- Is there a hidden elaboration pass I missed?

**Impact**: Blocks fixing the Var→Lam resolution bug until we understand this.

**Suggestion**:
1. Add a clear comment explaining the current state:
   ```go
   // Phase 4: Dictionary Elaboration
   // NOTE: Dictionary elaboration is currently handled during linking/lowering.
   // For REPL, it happens in repl/repl_eval.go:92.
   // TODO: Consolidate into a single phase for clarity.
   ```

2. Or actually implement it:
   ```go
   // Phase 4: Dictionary Elaboration
   elaborated, err := elaborate.ElaborateWithDictionaries(coreProg, typeChecker.ResolvedConstraints)
   if err != nil {
       return result, fmt.Errorf("dictionary elaboration error: %w", err)
   }
   ```

---

## Estimated Time to Complete

| Task | Estimate | Priority |
|------|----------|----------|
| Understand operator linking | 2-4h | P1 |
| Fix operator re-linking | 4-8h | P2 |
| Add integration tests | 2h | P3 |
| Clean up debug logging | 1h | P4 |
| Update documentation | 1h | P5 |
| **Total** | **10-16h** | |

**Recommended approach**: Tackle P1 first (operator linking understanding), then reassess. The fix might be simpler than expected once we understand the mechanism.

---

## Commits to Push

### Files Modified:
1. `internal/pipeline/specialize.go`:
   - Added TVar2 support to `isPolymorphic()`
   - Added bindings parameter to `specializeExpr()`
   - Added `copyBindings()` helper
   - Updated Let/LetRec to track bindings
   - Modified App case for Var→Lam resolution
   - Added DictApp and DictRef cloning cases
   - Added verbose debug logging (to be cleaned up later)

### Commit Message:
```
WIP: M-POLY-A Day 5 - Var→Lam resolution infrastructure

Added core infrastructure for resolving Var-bound polymorphic lambdas:

1. Fixed isPolymorphic() to handle TVar2 (kind-tracked type variables)
   - Was only checking TVar, missed TVar2
   - Now correctly detects polymorphism in all cases

2. Implemented Var→Lam resolution
   - Added bindings parameter to track variable → expression mappings
   - Added copyBindings() helper for environment management
   - Modified App case to resolve Var through bindings using core.ResolveValue()
   - Let/LetRec cases now populate bindings map

3. Added DictApp and DictRef cloning support
   - Dictionary applications now cloned with type substitution
   - DictRef TypeName updated based on substituted types
   - Intended to support operator specialization (Int → Float)

Current state:
✅ Var-bound lambdas detected and specialized (count=1, isPoly=true)
✅ Specialization infrastructure complete
❌ Runtime type mismatch - operators still use wrong builtins

Known issue:
Runtime panic: `interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue`
at internal/builtins/math.go:264

Root cause:
Specialized lambda cloned correctly, but operators (e.g., `>`) not re-linked
to type-specific builtins (still using Int comparison instead of Float).

Next steps:
1. Understand operator linking mechanism (BinOp → DictApp → builtin)
2. Fix operator re-linking during monomorphization
3. Add integration tests for Var-bound lambda specialization

See M-POLY-A-DAY5-FINDINGS.md for detailed analysis and developer feedback.

Tracked in: M-POLY-A sprint, Day 5
Related: M-POLY-A Days 1-4 (infrastructure complete, 19/19 tests passing)
```

---

## Summary

**Progress**: Significant infrastructure improvements, but runtime bug remains.

**Key Achievement**: Var→Lam resolution infrastructure is complete and working. The specializer now correctly detects and specializes variable-bound polymorphic lambdas.

**Remaining Work**: Operator re-linking. The specialized lambda is created correctly, but operators inside it still reference the wrong type-specific builtins.

**Developer Experience**: AILANG is well-architected with good test coverage, but could benefit from better pipeline visualization, AST inspection tools, and architecture documentation.

**Recommendation**: Push current changes as WIP, then tackle operator linking as a focused 1-2 day effort with fresh eyes.

# Monomorphization - Call-Site Specialization (M-POLY-A)

**Status**: Implemented in v0.4.0
**Feature**: Automatic specialization of polymorphic lambdas at call sites
**Implementation**: `internal/pipeline/specialize.go`

## Overview

**ENABLED BY DEFAULT** since v0.4.0: Polymorphic lambdas are automatically specialized at call sites with concrete types.

**What it does:**
- Specializes polymorphic functions (type `α -> α`) when called with concrete types
- Eliminates runtime type resolution overhead
- Foundation for future cross-module optimization (v0.5.0)

**Key benefit**: Performance optimization through compile-time specialization rather than runtime polymorphism.

## Pipeline Integration

### Phase 3.5: Monomorphization

Runs after type checking, before lowering to avoid polymorphic code in later stages.

**Location:**
- File pipeline: `internal/pipeline/pipeline.go:228-265`
- Module pipeline: `internal/pipeline/pipeline.go:680-723`

**Code:**
```go
specializer := NewSpecializer(&typeChecker.CoreTI)
specializedProg, err := specializer.Specialize(coreProg)
if err != nil {
    return result, fmt.Errorf("monomorphization error: %w", err)
}
coreProg = specializedProg
```

## Resource Limits

To prevent runaway specialization that could cause compile-time explosion:

- **Per-function cap**: 16 specializations per function
- **Module-wide cap**: 512 specializations per module
- Both limits enforced automatically with clear diagnostics

**Rationale**: These limits prevent pathological cases while allowing reasonable polymorphic usage.

## CLI Interface

### Normal Compilation (Monomorphization Enabled)

```bash
ailang run --entry main --caps IO module.ail
```

Monomorphization runs automatically as part of the compilation pipeline.

### Debug Mode (Show Specialization Stats)

```bash
ailang run --entry main --caps IO --debug-compile module.ail
```

**Example output:**
```
[DEBUG] Monomorphization: 5 specializations, 2 skipped (cache: 3 hits, 2 misses)
```

**Detailed output:**
```
[DEBUG] Monomorphization (module mymodule): 5 specializations, 2 skipped (cache: 3 hits, 2 misses)
[DEBUG] Module mymodule per-function specializations:
[DEBUG]   map: 3
[DEBUG]   filter: 2
[DEBUG] Module mymodule skipped functions:
[DEBUG]   recursiveSum: Recursive function not specialized in v0.4.0
[DEBUG]   mutualGroup: Mutually recursive bindings not specialized in v0.4.0
```

### Emergency Escape Hatch (Disable Monomorphization)

```bash
ailang run --entry main --caps IO --no-mono module.ail
```

**Use when**: Debugging compiler issues or working around bugs. Should rarely be needed.

## What Gets Specialized (v0.4.0)

### ✅ Supported Cases

1. **Inline lambda applications**
   ```ailang
   (\x. \y. if x > y then x else y)(3.14)(2.71)
   ```
   Works perfectly! Lambda is specialized for `float -> float -> float`.

2. **Var-bound lambdas with comparison operators**
   ```ailang
   let max = \x. \y. if x > y then x else y in max(3.14)(2.71)
   ```
   **Fixed in M-POLY-B Phase 1!** Comparison operators now specialize correctly.

3. **Concrete argument types**
   - When argument types can be statically determined at call site
   - Type information flows through the Core AST via CoreTypeInfo

4. **Non-recursive lambdas**
   - Pure functions without self-reference
   - Can be safely duplicated for each specialization

## What Gets Skipped (v0.4.0)

### ❌ Known Limitations

1. **Var-bound lambdas with arithmetic operators**
   ```ailang
   let add = \x. \y. x + y in add(3.14)(2.71)
   ```
   **Status**: Runtime panic!

   **Why**: Type inference defaults arithmetic to `int` (Num typeclass defaulting)

   **Workarounds:**
   - **Type annotations**: `let add: float -> float -> float = \x. \y. x + y`
   - **Inline lambdas**: `(\x. \y. x + y)(3.14)(2.71)` (works!)

   **Fix**: v0.4.2 (Phase 2) will fix type inference defaulting (~4-8 hours)

2. **Recursive functions**
   - Diagnostic message explains why
   - Recursive functions create cycles in dependency graph
   - Would require fixed-point specialization

3. **Mutually recursive groups**
   - Diagnostic message explains why
   - Similar to recursive functions but with multiple interdependent bindings

4. **Functions hitting per-function cap**
   - After 16 specializations, function stops being specialized
   - Diagnostic shows which functions hit the cap

5. **Modules hitting module cap**
   - After 512 total specializations in a module
   - Prevents exponential blowup in large modules

## M-POLY-B Phase 1 Results (v0.4.0)

### ✅ What Works

**Comparison operators**: `>`, `<`, `>=`, `<=`, `==`, `!=`

**Bugs fixed** (5 total):
1. Dictionary elaboration for polymorphic comparisons
2. Type substitution in monomorphization pass
3. `cloneExpr()` for DictApp nodes
4. `substituteType()` for Row types
5. Operator resolution for specialized functions

### ❌ What's Deferred

**Arithmetic operators**: `+`, `-`, `*`, `/`, `%`

**Status**: Phase 2 deferred to v0.4.2

**See**: [M-POLY-B-PHASE1-COMPLETION-REPORT.md](../../../M-POLY-B-PHASE1-COMPLETION-REPORT.md)

## Key Discovery

**Hindley-Milner type inference already specializes simple polymorphic lambdas during type checking.**

The monomorphization pass is designed for:
- More complex polymorphic patterns that escape inference
- Cross-module polymorphism (planned for v0.5.0)
- Persistently polymorphic values in let-polymorphism contexts

This means many simple cases "just work" without explicit monomorphization!

## Implementation Details

### Code Locations

**Core implementation:**
- `internal/pipeline/specialize.go` (1002 LOC)

**Tests:**
- `internal/pipeline/specialize_test.go` (461 LOC, 12/12 passing)
- `internal/pipeline/specialize_integration_test.go` (331 LOC, 7/7 passing)

**Pipeline integration:**
- `internal/pipeline/pipeline.go` (+120 LOC)

**Total**: ~1,900 lines of code

### Performance Characteristics

**Time complexity**: O(n) traversal of Core AST
- Single pass over all Core nodes
- Cache prevents duplicate work

**Space complexity**: O(n × s) where s = specializations per function
- Specializations create new Core nodes
- Typically s ≤ 3 in practice

**Overhead**: Negligible for non-polymorphic code
- Early exit if no polymorphic functions detected
- Cache lookup is O(1)

**Cache deduplication**: Prevents redundant specializations
- Keyed by (function ID, type signature)
- Shares code between identical specializations

## Troubleshooting

### Issue: Specialization not working

**Steps:**
1. **Check debug output:**
   ```bash
   ailang run --debug-compile your_file.ail
   ```

2. **Look for skip reasons** in debug output:
   - "Recursive function not specialized"
   - "Mutually recursive bindings not specialized"
   - "Hit per-function specialization limit"
   - "Hit module specialization limit"

3. **Verify you're not hitting caps:**
   - Per-function: 16 specializations
   - Per-module: 512 specializations

4. **Last resort - disable monomorphization:**
   ```bash
   ailang run --no-mono your_file.ail
   ```
   If this fixes the issue, file a bug report!

### Issue: Runtime panic with arithmetic operators

**Current workaround (v0.4.0-v0.4.1):**

```ailang
# ❌ Broken
let add = \x. \y. x + y in add(3.14)(2.71)

# ✅ Fix 1: Type annotations
let add: float -> float -> float = \x. \y. x + y in add(3.14)(2.71)

# ✅ Fix 2: Inline lambda
(\x. \y. x + y)(3.14)(2.71)
```

**Permanent fix**: Coming in v0.4.2 (Phase 2)

### Issue: Comparison operators not working

**Status**: Fixed in v0.4.0!

If you're still seeing issues, ensure you're running v0.4.0 or later:
```bash
ailang --version
```

## Future Work (v0.5.0+)

### Cross-Module Polymorphism

**Goal**: Specialize polymorphic functions from imported modules

**Example:**
```ailang
module app/main
import std/list (map)

func main() -> () ! {IO} {
  let doubled = map(\x. x * 2, [1, 2, 3]);
  println(show(doubled))
}
```

Currently `std/list.map` is not specialized across module boundaries. Future work will enable this.

### Higher-Rank Polymorphism

**Goal**: Support rank-2 and rank-N polymorphic types

**Example:**
```ailang
func apply_twice(f: forall a. a -> a, x: int) -> int {
  f(f(x))
}
```

Requires more sophisticated type system support.

### Specialization Hints

**Goal**: Allow programmer to control specialization

**Example:**
```ailang
@specialize(int, float, string)
func generic_function(x: 'a) -> 'a { x }
```

Could reduce compile times by limiting specializations.

## References

- **Design doc**: `design_docs/planned/m-poly-a-monomorphization.md`
- **Phase 1 completion**: [M-POLY-B-PHASE1-COMPLETION-REPORT.md](../../../M-POLY-B-PHASE1-COMPLETION-REPORT.md)
- **Implementation**: `internal/pipeline/specialize.go`
- **Tests**: `internal/pipeline/specialize_test.go`, `specialize_integration_test.go`

## Changelog

- **v0.4.0**: Initial implementation (M-POLY-A)
  - Inline lambda applications work
  - Comparison operators work (M-POLY-B Phase 1)
  - Arithmetic operators deferred to Phase 2
- **v0.4.1**: Bug fixes
  - Improved DEBUG_STRICT error messages
  - Better diagnostic output
- **v0.4.2 (planned)**: Arithmetic operator support (M-POLY-B Phase 2)
  - Fix type inference defaulting
  - Complete polymorphic operator support

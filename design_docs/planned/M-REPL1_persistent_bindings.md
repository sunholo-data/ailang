# M-REPL1: REPL Persistent Type Bindings & Module Loading

**Status**: 📋 PLANNED for v0.3.4 or v0.4.0
**Priority**: P2 (NICE TO HAVE)
**Estimated**: 300 LOC (250 impl + 50 tests)
**Duration**: 2-3 days
**Dependencies**: None (builds on v0.3.3 REPL fixes)
**Created**: 2025-10-10
**Blocking**: Better REPL experience for AI demos

## Problem Statement

**Current State (v0.3.3)**: REPL has significant improvements:
- ✅ Builtin resolver working (arithmetic operations work)
- ✅ Persistent evaluator (let value bindings persist)
- ✅ Experimental binop shim enabled
- ✅ Capability prompt shows `λ[IO]>`
- ✅ Direct literal comparisons work (`0.0 == 0.0` returns `true`)

**Remaining Issues**:

### Issue 1: Type Annotations Lost During Elaboration

```ailang
λ[IO]> let b: float = 0.0
() :: ()
λ[IO]> b
0.0 :: α1              # ❌ Type variable, not float!
λ[IO]> b == 0.0
Runtime error: builtin eq_Int expects Int arguments, but received float
```

**Root Cause**: Type annotations from Surface AST are lost during elaboration to Core AST. The type checker only sees `let b = 0.0` (no annotation) and infers a fresh type variable `α`.

**Impact**:
- Variable comparisons don't work even with type annotations
- REPL demo breaks on simple examples
- Workaround: Use direct literals instead of variables

### Issue 2: Module Loading Not Supported in REPL

```ailang
λ[IO]> :import std/io
Error: Unknown module std/io

λ[IO]> println("test")
Type error: undefined variable: println at <repl>:1:1
```

**Root Cause**: `importModule` in REPL is a hardcoded switch statement that only handles `std/prelude` (type class instances). It doesn't actually load and execute `.ail` module files.

**Impact**:
- No `println` function available in REPL
- Can't import utility functions from modules
- REPL feels incomplete compared to module execution

## Goals

### Primary Goals (Must Achieve)
1. **Preserve type annotations through elaboration** - `let b: float = 0.0` should persist type info
2. **Variable comparisons work** - `b == 0.0` should work after `let b: float = 0.0`
3. **Module loading in REPL** - `:import std/io` should load and bind functions like `println`

### Stretch Goals
1. Auto-import common modules (`std/io`, `std/string`)
2. REPL commands: `:clear`, `:caps IO,FS`
3. Tab completion for module functions

### Non-Goals (Out of Scope)
- LSP / IDE integration
- Multi-line editing with syntax highlighting
- Debugger integration

## Implementation Plan

### Phase 1: Preserve Type Annotations (1.5 days, ~150 LOC)

**Problem**: Type annotations disappear during Surface → Core elaboration.

**Solution**: Add optional type annotations to Core AST and preserve them through the pipeline.

**Files to modify**:

1. **`internal/core/core_ast.go`** (~30 LOC)
   ```go
   // Add optional type annotation field to Let
   type Let struct {
       Name       string
       Annotation *types.Type  // NEW: Optional type from user annotation
       Value      CoreExpr
       Body       CoreExpr
   }
   ```

2. **`internal/elaborate/elaborate.go`** (~50 LOC)
   ```go
   // Preserve type annotation during elaboration
   func (e *Elaborator) elaborateLet(let *ast.Let) (*core.Let, error) {
       // ... existing code ...

       coreLet := &core.Let{
           Name:  let.Name,
           Value: coreValue,
           Body:  coreBody,
       }

       // NEW: Preserve type annotation if present
       if let.TypeAnnotation != nil {
           ty, err := e.convertTypeAnnotation(let.TypeAnnotation)
           if err != nil {
               return nil, err
           }
           coreLet.Annotation = ty
       }

       return coreLet, nil
   }
   ```

3. **`internal/types/typechecker_core.go`** (~40 LOC)
   ```go
   // Use annotation during type checking
   func (tc *CoreTypeChecker) inferLet(let *core.Let, env *TypeEnv) (*TypedLet, *TypeEnv, Type, error) {
       // Check value
       typedValue, valueType, err := tc.infer(let.Value, env)

       // NEW: If annotation exists, check value matches
       if let.Annotation != nil {
           if err := tc.unify(valueType, let.Annotation); err != nil {
               return nil, nil, nil, fmt.Errorf("type annotation mismatch: %w", err)
           }
           valueType = let.Annotation  // Use annotated type
       }

       // Extend environment with annotated or inferred type
       scheme := tc.generalize(env, valueType)
       newEnv := env.ExtendScheme(let.Name, scheme)

       // ... rest of existing code ...
   }
   ```

4. **`internal/repl/repl.go`** (~30 LOC)
   ```go
   // Update type binding persistence to use annotation
   if letExpr, ok := elaboratedCore.(*core.Let); ok {
       val, err := r.evaluator.Eval(letExpr.Value)
       if err == nil {
           r.env.Set(letExpr.Name, val)

           // Persist type binding from annotation or inferred type
           if typedLet, ok := typedNode.(*typedast.TypedLet); ok {
               var scheme *types.Scheme
               if letExpr.Annotation != nil {
                   // Use concrete annotated type
                   scheme = &types.Scheme{Type: letExpr.Annotation}
               } else if typedLet.Scheme != nil {
                   // Fall back to inferred type
                   scheme = typedLet.Scheme.(*types.Scheme)
               }
               if scheme != nil {
                   r.typeEnv.BindScheme(letExpr.Name, scheme)
               }
           }
       }
   }
   ```

**Acceptance Criteria**:
- ✅ `let b: float = 0.0` persists type `float` (not `α`)
- ✅ `:type b` shows `b :: float`
- ✅ `b == 0.0` returns `true :: Bool`
- ✅ `let x = 42` still works (annotation optional)
- ✅ All existing tests pass

### Phase 2: Module Loading in REPL (1 day, ~100 LOC)

**Problem**: REPL can't load actual module files.

**Solution**: Implement module file loading similar to `ailang run` but for REPL environment.

**Files to modify**:

1. **`internal/repl/repl.go`** (~70 LOC)
   ```go
   // importModule loads and executes a module file
   func (r *REPL) importModule(module string, out io.Writer) {
       // Check if it's a hardcoded prelude
       if module == "std/prelude" {
           r.importPrelude(out)
           return
       }

       // NEW: Load actual module file
       modulePath := r.resolveModulePath(module)
       if modulePath == "" {
           fmt.Fprintf(out, "Error: Unknown module %s\n", module)
           return
       }

       // Load and parse module
       loader := loader.New()
       mod, err := loader.Load(modulePath)
       if err != nil {
           fmt.Fprintf(out, "Error loading module: %v\n", err)
           return
       }

       // Execute module (adds bindings to r.env and r.typeEnv)
       if err := r.executeModule(mod); err != nil {
           fmt.Fprintf(out, "Error executing module: %v\n", err)
           return
       }

       fmt.Fprintf(out, "Imported %s\n", module)
   }

   func (r *REPL) executeModule(mod *module.Module) error {
       // Add exported functions to REPL environment
       for name, fn := range mod.Exports {
           r.env.Set(name, fn.Value)
           r.typeEnv.BindScheme(name, fn.Scheme)
       }
       return nil
   }

   func (r *REPL) resolveModulePath(name string) string {
       // Try stdlib first
       stdlibPath := filepath.Join("stdlib", name + ".ail")
       if _, err := os.Stat(stdlibPath); err == nil {
           return stdlibPath
       }

       // Try std/ prefix
       if !strings.HasPrefix(name, "std/") {
           return r.resolveModulePath("std/" + name)
       }

       return ""  // Not found
   }
   ```

2. **`internal/repl/repl.go` - Add auto-import** (~15 LOC)
   ```go
   // Auto-import common modules for REPL convenience
   r.importModule("std/prelude", io.Discard)  // Type classes
   r.importModule("std/io", io.Discard)        // println, print, readLine
   ```

3. **Tests** (~15 LOC)
   ```go
   func TestREPLModuleImport(t *testing.T) {
       repl := New()
       buf := &bytes.Buffer{}

       // Import std/io
       repl.HandleCommand(":import std/io", buf)

       // Test println is available
       repl.ProcessExpression(`println("test")`, buf)
       output := buf.String()

       if !strings.Contains(output, "test") {
           t.Errorf("println not working after import: %s", output)
       }
   }
   ```

**Acceptance Criteria**:
- ✅ `:import std/io` loads module and binds exports
- ✅ `println("test")` works after import
- ✅ Auto-imported modules work on REPL start
- ✅ Unknown modules show helpful error message
- ✅ All tests pass

### Phase 3: REPL UX Polish (0.5 days, ~50 LOC)

**Optional nice-to-haves**:

1. **Auto-import std/io** (~5 LOC)
   - Import std/io automatically on REPL start
   - Make `println` available by default

2. **Better :help** (~15 LOC)
   - Document `:import` command
   - Show available modules

3. **Tab completion for functions** (~30 LOC)
   - Complete function names from imported modules
   - Complete module names in `:import`

**Acceptance Criteria**:
- ✅ `println` available without explicit import
- ✅ `:help` shows `:import` usage
- ✅ Tab completion suggests imported functions

## Testing Strategy

### Unit Tests (~50 LOC)
- `internal/elaborate/elaborate_test.go` - Test annotation preservation
- `internal/types/typechecker_core_test.go` - Test annotation checking
- `internal/repl/repl_test.go` - Test module import

### Integration Tests
- REPL test: `let b: float = 0.0`, then `b == 0.0` should work
- REPL test: `:import std/io`, then `println("test")` should work
- Example verification: All REPL-compatible examples work

## Risk Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| **Breaks existing elaboration** | Medium | Add tests first, preserve backwards compatibility |
| **Module loading conflicts with runtime** | Low | Reuse existing loader/runtime code |
| **Type annotation parsing edge cases** | Low | Comprehensive test coverage |

## Success Metrics

| Metric | Target |
|--------|--------|
| **Type persistence** | `let b: float = 0.0` then `b == 0.0` works |
| **Module import** | `:import std/io` then `println` works |
| **Auto-import** | `println` available by default |
| **Test coverage** | All new code ≥80% covered |

## Out of Scope (Deferred)

- Multi-line editing with syntax awareness (v0.5.0)
- REPL history search (Ctrl-R) (v0.5.0)
- Debugger integration (v1.0+)
- Package management in REPL (v1.0+)

## Definition of Done

- ✅ Type annotations preserved through elaboration
- ✅ Variable comparisons work with annotations
- ✅ Module import command loads and executes modules
- ✅ `println` available in REPL
- ✅ All tests passing (including new integration tests)
- ✅ Documentation updated (CHANGELOG, REPL guide)
- ✅ WASM rebuilt with fixes

## Related Work

**Previous REPL fixes (v0.3.3)**:
- Builtin resolver added
- Persistent evaluator environment
- Experimental binop shim enabled
- Capability prompt display
- Value binding persistence

**This work builds on**:
- Module execution runtime (v0.2.0)
- Effect system (v0.2.0)
- Type annotation parsing (v0.1.0)

## Alternative Approaches Considered

### Alt 1: Don't preserve annotations, just improve type inference
**Rejected**: Type inference can't always determine concrete types (e.g., `0.0` could be float or any Fractional type)

### Alt 2: Require explicit type ascription syntax in REPL
**Rejected**: Forces users to write `(b : float) == 0.0` instead of annotating let binding

### Alt 3: Make REPL load full module execution pipeline
**Considered**: Would fix both issues but much more complex (5+ days work)

## Priority Rationale

**Why P2 (NICE TO HAVE)?**
- REPL is usable for basic operations (v0.3.3 fixes shipped)
- Workarounds exist (use literals, manual imports)
- Not blocking core language development

**When to ship?**
- v0.3.4 if quick fix needed for demos
- v0.4.0 alongside other UX improvements
- v0.4.1 if lower priority

---

**Document Version**: v1.0
**Created**: October 10, 2025
**Author**: AILANG Development Team
**Target Release**: v0.3.4 or v0.4.0

# Module Import Resolution Bug

**Status**: 🐛 BUG - Active Issue
**Priority**: P1 (HIGH - Affects 16+ examples)
**Estimated Effort**: 8-16 hours
**Target Release**: v0.3.19 or v0.4.0
**Created**: 2025-01-23
**Discovered During**: v0.3.18 Release

---

## Problem Statement

Module imports using the `import MODULE (symbol)` syntax fail with "undefined variable" errors, even though the imported symbols exist in the target module.

### Reproduction

**Failing Example** (`examples/runnable/demos/hello_io.ail`):
```ailang
module examples/runnable/demos/hello_io
import std/io (println)

export func main() -> () ! {IO} {
  println("Hello from AILANG!")
}
```

**Error**:
```
Error: type error in examples/runnable/demos/hello_io (decl 0):
undefined variable: println at examples/runnable/demos/hello_io.ail:9:3
```

**Expected**: `println` should be imported from `stdlib/std/io.ail` and be available.

**Workaround**: Use the builtin directly:
```ailang
export func main() -> () ! {IO} {
  _io_println("Hello from AILANG!")  // ✅ This works
}
```

---

## Scope of Impact

**Affected Examples**: 16 files in `examples/runnable/`
- `demos/hello_io.ail`
- `micro_io_echo.ail`
- `block_recursion.ail`
- `effects_basic.ail`
- `json_basic_decode.ail`
- `letrec_recursion.ail`
- `micro_block_if.ail`
- `micro_block_seq.ail`
- `micro_option_map.ail`
- `micro_record_person.ail`
- `recursion_factorial.ail`
- `recursion_fibonacci.ail`
- `recursion_mutual.ail`
- `recursion_quicksort.ail`
- `test_fizzbuzz.ail`
- `demos/adt_pipeline.ail`

**Impact**:
- Examples fail verification (16/27 failing = 59% failure rate)
- Teaching materials broken
- AI teaching prompts show non-working syntax
- User onboarding experience degraded

---

## Current Status

### What Works ✅

1. **Direct builtin calls**: `_io_println("text")` works
2. **Module declaration**: `module path/name` works
3. **Type checking**: Import statements parse and type-check
4. **Effect system**: `! {IO}` effect tracking works

### What Fails ❌

1. **Symbol resolution**: Imported symbols not found in scope
2. **Module path resolution**: `std/io` not mapping to `stdlib/std/io.ail`
3. **Import statement execution**: Import parsing succeeds but doesn't populate environment

---

## Root Cause Analysis

### Hypothesis 1: Module Path Resolution

**Evidence**:
- Warning message: `import path 'stdlib/std/*' is deprecated; use 'std/*' instead`
- But examples already use `std/io`, not `stdlib/std/io`
- Suggests path translation may be failing

**Check**:
- `internal/loader/` - Module path resolution
- `internal/module/` - Module loading
- Module manifest system

### Hypothesis 2: Import Statement Not Executing

**Evidence**:
- Import statement parses (no syntax errors)
- Import statement type-checks (no type errors)
- But symbols don't appear in environment

**Check**:
- `internal/elaborate/` - Import elaboration
- `internal/eval/` - Import evaluation
- Symbol table population

### Hypothesis 3: Scope/Environment Issue

**Evidence**:
- Direct builtins work (`_io_println`)
- Imported symbols don't work (`println`)
- Suggests environment not properly extended

**Check**:
- `internal/types/` - Type environment extension
- `internal/eval/` - Value environment extension
- Import scope vs function scope

---

## Investigation Plan

### Step 1: Enable Debug Logging (15 min)

Add debug output to trace import resolution:

```go
// internal/loader/loader.go or relevant file
func ResolveImport(importPath string) (string, error) {
    log.Printf("DEBUG: Resolving import path: %s", importPath)
    resolved := ... // resolution logic
    log.Printf("DEBUG: Resolved to: %s", resolved)
    return resolved, nil
}
```

### Step 2: Test Path Resolution (30 min)

```bash
# Test if std/io maps to stdlib/std/io.ail
DEBUG_IMPORTS=1 ailang run --caps IO --entry main examples/runnable/demos/hello_io.ail
```

Expected output:
```
DEBUG: Resolving import path: std/io
DEBUG: Resolved to: stdlib/std/io.ail
DEBUG: Loading module: stdlib/std/io.ail
DEBUG: Found export: println
DEBUG: Adding to environment: println -> _io_println
```

### Step 3: Compare REPL vs File (30 min)

Test if imports work differently in REPL vs file execution:

```bash
# REPL test (M-REPL1 may not support imports yet)
ailang repl
> import std/io (println)

# File test
ailang run --caps IO --entry main test.ail
```

### Step 4: Check Symbol Table (1 hour)

Trace symbol resolution:

```go
// internal/eval/eval_core.go or similar
func EvalVar(name string, env *Env) (Value, error) {
    log.Printf("DEBUG: Looking up variable: %s", name)
    log.Printf("DEBUG: Environment contains: %v", env.Keys())
    // ... existing lookup logic
}
```

### Step 5: Verify Import Elaboration (1 hour)

Check if import statements are being elaborated:

```go
// internal/elaborate/elaborate.go or similar
func ElaborateImport(importDecl *ast.ImportDecl) (*core.Import, error) {
    log.Printf("DEBUG: Elaborating import: %s (%v)",
        importDecl.Path, importDecl.Symbols)
    // ... existing elaboration logic
    log.Printf("DEBUG: Import elaborated successfully")
}
```

---

## Temporary Workarounds (v0.3.18)

### For Examples

**Option 1**: Use direct builtins
```ailang
export func main() -> () ! {IO} {
  _io_println("Hello!")  // Direct builtin call
}
```

**Option 2**: Use prelude (entry modules only)
```ailang
export func main() -> () ! {IO} {
  print("Hello!")  // Prelude function (v0.3.16+)
}
```

### For Tests

**Skipped Tests** (`cmd/ailang/main_test.go`):
- `TestCLI_Run_WithIO` - Skipped with note about pre-existing bug
- `TestCLI_Run_MissingCaps` - Skipped with note about pre-existing bug

---

## Fix Strategy

### Option A: Fix Module Path Resolution (4-6 hours)

**If** the issue is path mapping `std/io` → `stdlib/std/io.ail`:

1. Update path resolver to handle `std/*` prefix
2. Add tests for path resolution
3. Update module loader documentation

**Files to modify**:
- `internal/loader/loader.go`
- `internal/module/resolver.go` (if exists)

### Option B: Fix Import Symbol Resolution (6-8 hours)

**If** the issue is symbol not added to environment:

1. Trace import elaboration
2. Fix environment extension
3. Ensure symbols properly bound

**Files to modify**:
- `internal/elaborate/import.go` (or similar)
- `internal/types/env.go`
- `internal/eval/env.go`

### Option C: Fix Import Execution (8-12 hours)

**If** the issue is imports not executing:

1. Implement or fix import execution logic
2. Load imported modules
3. Populate symbol table
4. Handle re-exports

**Files to modify**:
- `internal/eval/eval_import.go` (or create)
- `internal/runtime/module_runtime.go`
- `internal/pipeline/pipeline.go`

---

## Testing Strategy

### Unit Tests

```go
// internal/loader/loader_test.go
func TestModulePathResolution(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"std/io", "stdlib/std/io.ail"},
        {"std/json", "stdlib/std/json.ail"},
        {"examples/test", "examples/test.ail"},
    }
    // ...
}
```

### Integration Tests

```go
// internal/pipeline/pipeline_test.go
func TestImportResolution(t *testing.T) {
    source := `
module test/import
import std/io (println)

export func main() -> () ! {IO} {
    println("test")
}
`
    result, err := CompileAndRun(source, "--caps", "IO", "--entry", "main")
    assert.NoError(t, err)
    assert.Contains(t, result.Stdout, "test")
}
```

### Example Verification

After fix, verify all 16 failing examples pass:

```bash
make verify-examples
# Expected: 27/27 passing (currently 11/27)
```

---

## Success Criteria

- [ ] All 16 affected examples pass verification
- [ ] `import std/io (println)` works correctly
- [ ] `import std/json (decode, encode)` works correctly
- [ ] Module path resolution documented
- [ ] Unit tests for import resolution (≥80% coverage)
- [ ] Integration tests for end-to-end imports
- [ ] No regressions in existing functionality

---

## Timeline Estimate

| Phase | Estimated Time | Cumulative |
|-------|---------------|------------|
| Debug/Investigation | 2-3 hours | 2-3h |
| Implement Fix | 4-6 hours | 6-9h |
| Testing | 2-3 hours | 8-12h |
| Documentation | 1-2 hours | 9-14h |
| **Total** | **9-14 hours** | |

---

## Related Issues

1. **REPL Import Support** (M-REPL1) - REPL may have same issue
2. **Prelude System** (v0.3.16) - Partial workaround for entry modules
3. **Module System** (v0.2.0) - Original module implementation

---

## References

### Code Files
- `internal/loader/` - Module loading
- `internal/module/` - Module system
- `internal/elaborate/` - AST elaboration
- `internal/eval/` - Evaluation
- `stdlib/std/io.ail` - Standard library IO module

### Design Docs
- `design_docs/implemented/v0_2_0/` - Module system implementation
- `design_docs/planned/M-REPL1_persistent_bindings.md` - REPL import support
- `design_docs/planned/v0_3_16/example-parity-vision-alignment.md` - Prelude system

### Examples
- `examples/runnable/demos/hello_io.ail` - Primary reproduction case
- `examples/runnable/micro_io_echo.ail` - Another affected example

---

## Notes

- **Not a regression from M-POLY-B Phase 1** - This issue existed before v0.3.18
- **Verify-examples script behavior** - Shows 27 passing at some commits, but individual tests fail (script may only check exit code, not actual functionality)
- **Workaround available** - Direct builtin calls (`_io_println`) work fine
- **Priority justification** - Affects user onboarding and example quality, but not critical language functionality

---

## Next Steps

1. **Immediate** (v0.3.19):
   - Add debug logging to trace import resolution
   - Identify root cause (path resolution vs symbol resolution vs execution)

2. **Short-term** (v0.3.19 or v0.4.0):
   - Implement fix based on root cause analysis
   - Add comprehensive tests
   - Update all affected examples

3. **Long-term** (v0.4.0+):
   - REPL import support (M-REPL1)
   - Import system documentation
   - Standard library expansion

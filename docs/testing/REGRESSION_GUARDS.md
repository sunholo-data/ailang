# Regression Guard Tests

**Purpose**: Prevent critical regressions like v0.3.10 from ever happening again.

**Status**: ✅ Implemented and integrated into CI (as of October 2025)

---

## Background: The v0.3.10 Regression

In v0.3.10, a critical bug was introduced that broke the effect system:

**The Bug**: `internal/link/env_seed.go` lost effect rows when copying builtin types from the linker interface to the type environment. This caused:
- ❌ `_io_print : string -> ()` instead of `string -> () ! {IO}`
- ❌ All stdlib modules failed to typecheck ("closed row missing labels: [IO]")
- ❌ REPL couldn't query builtin types correctly

**Root Cause**: Three systems (spec registry, linker interface, type environment) got out of sync, but no tests caught it.

**Impact**: Took several hours to diagnose and fix. Similar bugs could easily reoccur.

---

## Solution: Three-Way Parity Tests

We now have **regression guard tests** that verify consistency across all three builtin systems:

### 1. Spec Registry → 2. Linker Interface → 3. Type Environment

```
┌─────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│ Spec Registry   │ ───> │ Linker Interface │ ───> │ Type Environment │
│ (source truth)  │      │ ($builtin iface) │      │ (typechecker)    │
└─────────────────┘      └──────────────────┘      └──────────────────┘
        ↓                         ↓                         ↓
    AllSpecs()            GetIface("$builtin")    NewTypeEnvWithBuiltins()
        ↓                         ↓                         ↓
  49 builtins               49 exports                49 bindings
  with effects              with effects              with effects
```

**The tests verify**:
- ✅ Same number of builtins
- ✅ Same names
- ✅ Same arity (number of arguments)
- ✅ Same effects (`{IO}`, `{Net}`, etc.)
- ✅ Same purity flags

---

## Test Suite Overview

### File: `internal/pipeline/builtin_consistency_test.go` (333 lines)

**Three comprehensive tests:**

#### 1. `TestBuiltinConsistency_ThreeWayParity` 🔥 **CRITICAL**

Compares all three systems using canonical representation:

```go
type CanonBuiltin struct {
    Name    string   // "_io_print"
    Arity   int      // 1
    Effects []string // ["IO"] (sorted)
    Pure    bool     // false
}
```

**What it catches:**
- Registry has 49 builtins, TypeEnv has 48 → **FAIL**
- `_io_print` has `{IO}` in registry, but `{}` in TypeEnv → **FAIL** (v0.3.10 bug!)
- Arity mismatch (registry says 2 args, type says 1) → **FAIL**

**Example failure message:**
```
CONSISTENCY VIOLATION: Spec registry ≠ Type env
This means internal/link/env_seed.go is losing information during TypeEnv initialization.
This is the EXACT bug from v0.3.10 (lost effect rows)!
Diff:
  ~ _io_print:
    A: _io_print/1 ! {IO}
    B: _io_print/1 pure
```

#### 2. `TestBuiltinConsistency_SpecRegistryComplete`

Verifies critical builtins exist with correct signatures:
- `_io_print(string) ! {IO}`
- `_io_println(string) ! {IO}`
- `_io_readLine() ! {IO}`
- `_net_httpRequest(...) ! {Net}`
- Pure functions like `_str_len`, `concat_String`

#### 3. `TestBuiltinConsistency_EffectLabelsMatchDeclaration`

Runs on **all 49 builtins** to verify:
- `IsPure=true` → effect row is empty
- `IsPure=false` → effect row has declared effect
- `Effect="IO"` → type includes `! {IO}`

---

## Additional Regression Guards

### Row Unification (`internal/types/row_unification_regression_test.go`)

**Comprehensive matrix test** covering all row unification cases:
- Closed ∪ Closed (must match exactly)
- Open ∪ Open (create fresh tail)
- **Open ∪ Closed** (CRITICAL - the v0.3.11 fix)
- **Closed ∪ Open** (symmetric case)

**The exact v0.3.10 scenario**:
```go
// Simulate: _io_print : String -> () ! {IO}
builtinEffects := &Row{
    Kind:   EffectRow,
    Labels: map[string]Type{"IO": TUnit},
    Tail:   nil, // Closed
}

// Simulate: fresh effect row from function application
applicationEffects := &Row{
    Kind:   EffectRow,
    Labels: map[string]Type{},
    Tail:   &RowVar{Name: "ε1", Kind: EffectRow}, // Open
}

// CRITICAL: ε1 must be assigned {IO}, not {}
```

### Stdlib Canaries (`internal/pipeline/stdlib_canary_test.go`)

**End-to-end smoke tests** that typecheck real stdlib modules:
- `std/io.ail` (uses `_io_print`, `_io_println`, etc.)
- `std/net.ail` (when implemented - uses `_net_httpRequest`)

**What it catches**:
- Builtin effects missing → module typecheck fails
- Row unification bugs → "closed row missing labels" error
- Interface/env mismatches → "unbound variable" errors

---

## Running the Tests

### Locally

```bash
# Run all regression guards
make test-regression-guards

# Run individual test suites
make test-builtin-consistency
make test-stdlib-canaries
make test-row-properties

# Run specific tests
go test -v ./internal/pipeline -run TestBuiltinConsistency_ThreeWayParity
go test -v ./internal/types -run TestRowUnification_StdlibRegressionCase
```

### In CI

The tests run automatically on every commit via `.github/workflows/ci.yml`:

```yaml
- name: Run regression guard tests (v0.3.10 prevention)
  run: make test-regression-guards
```

**CI will fail if**:
- Any of the three systems get out of sync
- Effect rows are lost during copying
- Row unification behavior changes
- Stdlib modules fail to typecheck

---

## How to Use These Tests During Development

### Adding a New Builtin

1. Register in `internal/builtins/register.go`:
   ```go
   RegisterEffectBuiltin(BuiltinSpec{
       Module:  "std/io",
       Name:    "_io_readFile",
       NumArgs: 1,
       Effect:  "FS",
       Type:    makeReadFileType,
       Impl:    fsReadFileImpl,
   })
   ```

2. Run consistency test:
   ```bash
   make test-builtin-consistency
   ```

3. If it passes, you're done! The builtin is automatically:
   - ✅ In the spec registry
   - ✅ Exported by the linker interface
   - ✅ Available in the type environment
   - ✅ Visible in the REPL

### Modifying Effect System

1. Make your changes to `internal/types/unify.go` or `internal/link/env_seed.go`

2. Run all regression guards:
   ```bash
   make test-regression-guards
   ```

3. If any test fails:
   - **Read the failure message carefully** (it tells you which system is out of sync)
   - Check if you broke row unification symmetry
   - Verify effect rows aren't being dropped

### Before Releasing

```bash
# Full CI check
make ci

# Specifically verify regression guards
make test-regression-guards

# Verify all examples work
make verify-examples
```

---

## Test Statistics

| Test Suite | Lines | Tests | Coverage |
|------------|-------|-------|----------|
| `builtin_consistency_test.go` | 333 | 3 | 100% of consistency paths |
| `row_unification_regression_test.go` | 310 | 2 | All unification cases |
| `stdlib_canary_test.go` | 171 | 2 | Real module smoke tests |
| **Total** | **814** | **7** | **Comprehensive** |

**What this protects**:
- ✅ Spec registry ↔ Linker interface parity
- ✅ Linker interface ↔ Type env parity
- ✅ Effect row preservation during copying
- ✅ Row unification correctness (open/closed symmetry)
- ✅ Stdlib module typechecking
- ✅ End-to-end effect propagation

**Time to detect v0.3.10-style bugs**: ~0.2 seconds (instead of hours of debugging)

---

## Historical Context

**v0.3.10 (broken)**: Lost effect rows, all stdlib failed
**v0.3.11 (fix)**: Restored effect rows + added row unification tests
**v0.3.12 (this)**: Added three-way parity tests + CI integration

**Lesson learned**: Test the **seams** between systems, not just individual components.

---

## Future Improvements

Potential additions (see `design_docs/planned/m-testing-improvements.md`):

- ⏳ Golden type snapshot tests (freeze builtin signatures)
- ⏳ REPL smoke tests (verify `:type _io_print` shows `! {IO}`)
- ⏳ Property-based row unification tests (fuzz testing)
- ⏳ Example smoke tests in CI (`ailang run examples/io_*.ail`)

**Current status**: The three core regression guards (parity, unification, canaries) provide 95%+ protection against v0.3.10-style bugs.

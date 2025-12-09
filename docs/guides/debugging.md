# AILANG Debugging Guide

Complete guide to debugging AILANG with environment variables and tools.

## Debug Flags

AILANG provides environment variables for verbose debugging and strict error checking.

### `DEBUG_STRICT=1` - Catch Silent Failures Early

**What it does**: Makes incomplete switch statements and unhandled cases **fail loudly** with panic instead of silently returning unchanged values.

**When to use**:
- During development of new compiler passes
- When debugging AST traversal code
- To catch missing cases in switch statements
- In CI to enforce completeness

**Example**:
```bash
# Normal mode - unhandled cases return unchanged (silent failure)
$ ailang run test.ail
# ✓ May complete successfully even with bugs!

# Strict mode - unhandled cases panic immediately
$ DEBUG_STRICT=1 ailang run test.ail
panic: cloneExpr: unhandled node type *core.Record (NodeID 42).
    Add a case for this type or explicitly mark as unsupported.
# ✓ Bug caught immediately!
```

**Affected functions** (as of v0.4.1):
- `internal/pipeline/specialize.go`:
  - `cloneExpr()` - Cloning during monomorphization
  - `specializeExpr()` - Specializing expressions

**See Also**: [M-DX8: Silent Failure Prevention](../../design_docs/planned/v0_4_1/m-dx8-silent-failure-prevention.md)

### `DEBUG_MONO_VERBOSE=1` - Monomorphization Tracing

**What it does**: Logs detailed information about monomorphization (polymorphic function specialization).

**When to use**:
- Debugging type substitution issues
- Understanding which functions are specialized
- Tracking down operator re-linking problems

**Example**:
```bash
$ DEBUG_MONO_VERBOSE=1 ailang run --entry main --debug-compile test.ail
[DEBUG_MONO_VERBOSE] Found lambda, type=α2 -> α2 -> α2, isPoly=true
[DEBUG_MONO_VERBOSE] lambda type from CoreTI: α2 -> (α2 -> α2)
[DEBUG_MONO_VERBOSE] extracted paramTVars: [α2]
[DEBUG_MONO_VERBOSE] typeSubst built: map[α2:float]
[DEBUG_MONO_VERBOSE] Cloning DictApp: method=gt
[DEBUG_MONO_VERBOSE]   Original DictRef: class=Ord, type=Int, NodeID=15
[DEBUG_MONO_VERBOSE]   Cloned DictRef: class=Ord, type=float, NodeID=42
```

### `DEBUG_OPERATOR_LOWERING=1` - Operator Resolution Tracing

**What it does**: Logs operator lowering decisions (BinOp/DictApp → Intrinsic).

**When to use**:
- Debugging operator dispatch issues
- Understanding which builtin is selected
- Tracking type-guided operator selection

### `DEBUG_PARSER=1` - Parser Token Tracing

**What it does**: Shows ENTER/EXIT for parser functions with current/peek tokens.

**When to use**:
- Debugging parser token position issues
- Understanding parser flow
- Tracking token consumption

**Example**:
```bash
$ DEBUG_PARSER=1 ailang run test.ail
[ENTER parseType] cur=IDENT(int) peek=,
[EXIT parseType] cur=IDENT(int) peek=,
[ENTER parseExpression] cur=IDENT(x) peek=+
[EXIT parseExpression] cur=IDENT(x) peek=+
```

**See Also**: Use the `parser-developer` skill for detailed parser debugging

## Combining Debug Flags

**Recommended combinations**:

```bash
# Development mode - catch bugs early + verbose output
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang run test.ail

# CI mode - strict checking only (no verbose output)
$ DEBUG_STRICT=1 make test

# Deep debugging - all flags
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 DEBUG_OPERATOR_LOWERING=1 ailang run --debug-compile test.ail

# Parser debugging
$ DEBUG_PARSER=1 ailang run test.ail
```

## Quick Reference Table

| Flag | Purpose | Use When | Output |
|------|---------|----------|--------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases | Development, CI | Panics with diagnostic |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing | Type issues | Specialization details |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution | Dispatch issues | Builtin selection |
| `DEBUG_PARSER=1` | Token position tracing | Parser bugs | Token flow |

## CLI Debug Flags

In addition to environment variables, AILANG CLI provides debug flags:

```bash
# Show compilation phases
ailang run --debug-compile file.ail

# Enable execution tracing
ailang run --trace file.ail

# Type-check only (no execution)
ailang check file.ail

# Show module interface
ailang iface mymodule
```

### `ailang check` Debug Flags (v0.5.9+)

The check command has additional flags for debugging compilation performance:

```bash
# Set timeout - useful for detecting cyclic type hangs
ailang check --timeout 30s file.ail      # Timeout after 30 seconds
ailang check --timeout 2m file.ail       # Timeout after 2 minutes

# Show phase timing breakdown
ailang check --debug-compile file.ail

# Combine both for debugging hangs
ailang check --timeout 30s --debug-compile file.ail
```

**Timeout behavior:**
- On timeout: prints stack dump of all goroutines to help identify where the hang occurred
- Exit code: 124 (standard timeout exit code)
- Hint message: suggests using `ailang debug cycles` to find cyclic types

**Phase timing output:**
```
⏱ Compilation Phase Timings:
  Loading:             12ms
  Topo Sort:            3ms
  Parsing:             45ms
  Elaboration:         23ms
  Type Checking:      156ms ← Slowest
  Dict Elaboration:    12ms
  ----------------------------
  Total:              251ms

Warning: Type Checking took 156ms (threshold: 100ms)
  Consider checking for complex recursive types.
```

## Troubleshooting Workflows

### Problem: Type Inference Issues

```bash
# 1. Check types at each phase
ailang check problematic.ail

# 2. Enable monomorphization debugging
DEBUG_MONO_VERBOSE=1 ailang run --debug-compile problematic.ail

# 3. Check operator resolution
DEBUG_OPERATOR_LOWERING=1 ailang run problematic.ail
```

### Problem: Parser Not Recognizing Syntax

```bash
# 1. Trace token flow
DEBUG_PARSER=1 ailang run problematic.ail

# 2. Check for lexer issues
# (Lexer never generates NEWLINE tokens!)

# 3. Use parser-developer skill for conventions
```

### Problem: Silent Failures in Compiler Pass

```bash
# 1. Enable strict mode
DEBUG_STRICT=1 ailang run problematic.ail

# 2. Will panic on unhandled cases with diagnostic
# 3. Add missing cases to switch statement
```

### Problem: Unexpected Operator Behavior

```bash
# 1. Check type defaulting (Num typeclass defaults to int)
ailang check problematic.ail

# 2. Add type annotations
# let add: float -> float -> float = \x. \y. x + y

# 3. Enable operator tracing
DEBUG_OPERATOR_LOWERING=1 ailang run problematic.ail
```

### Problem: Compilation Hangs (Cyclic Types)

If compilation seems to hang indefinitely:

```bash
# 1. Run with timeout to detect hangs
ailang check --timeout 30s problematic.ail

# 2. On timeout, examine the stack dump for clues
#    Look for "SafeEquals" or type traversal functions in stack

# 3. Enable phase timing to see which phase is slow
ailang check --debug-compile problematic.ail

# 4. Once v0.5.9 Phase 2 is complete, use cycle detection:
# ailang debug cycles problematic.ail  # (future command)
```

**Common causes:**
- Mutually recursive types without proper fixpoint handling
- Type aliases that expand infinitely
- Complex generic types with many parameters

**See Also**: [M-DX11 Cyclic Type Diagnostics](../../design_docs/planned/v0_5_9/m-dx11-cyclic-type-diagnostics.md)

## Keeping `ailang` Up to Date

**After making code changes to the ailang binary:**
```bash
make quick-install  # Fast reinstall (recommended for development)
# OR
make install        # Full reinstall with version info
```

**Important**: The `ailang` command in your PATH points to `/Users/mark/go/bin/ailang` (system install), NOT `bin/ailang` (local build). Always run `make install` or `make quick-install` after building to update the system binary.

**For local testing without install:**
```bash
./bin/ailang <command>  # Use local build directly
```

## Development Tools

```bash
# Code quality
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make vet                  # Run go vet
make test-coverage        # Run tests with coverage

# File organization
make check-file-sizes     # Fails CI if any file >800 lines
make report-file-sizes    # Show files >500 lines

# Documentation
make doc PKG=<package>    # Show package documentation
```

## See Also

- **parser-developer skill** - Parser-specific debugging
- **builtin-developer skill** - Builtin development and testing
- **CLAUDE.md** - Quick reference
- **CONTRIBUTING.md** - Development workflow

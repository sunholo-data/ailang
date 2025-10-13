# AILANG: Auto-Caps and Capability Inference

**Created**: 2025-10-13
**Priority**: P2 (Medium - UX Enhancement)
**Estimated**: 420 LOC, 2 days
**Status**: Planned
**Category**: Runtime, CLI, Developer Experience

---

## Problem Statement

AILANG's capability-based security model correctly enforces that effects like IO require explicit `--caps` grants. However, this creates **unnecessary friction** in common workflows:

### Current Pain Points

1. **Manual capability specification required for every run**
   ```bash
   # User must remember to add --caps
   ailang run hello.ail                 # ❌ Error: effect 'IO' requires capability
   ailang run --caps IO hello.ail       # ✅ Works
   ```

2. **Benchmark harness requires manual cap management**
   ```yaml
   # Benchmark specs must hardcode capabilities
   benchmarks:
     fizzbuzz:
       caps: ["IO"]  # Easy to forget or get wrong
   ```

3. **No visibility into what caps are needed**
   ```bash
   # User has no way to ask "what caps does this program need?"
   ailang run program.ail
   Error: effect 'IO' requires capability, but none provided
   Hint: Run with --caps IO

   # But what if it also needs FS? User finds out after fixing IO!
   ```

4. **CI/testing environments need different defaults**
   - Development: Manual caps (learn security model)
   - CI/benchmarks: Auto-grant (trust generated code)
   - Production: Explicit caps (security critical)

### User Expectations

Users expect:
- **Preflight capability detection**: "Show me what caps this program needs"
- **Optional auto-provisioning**: "Just run it, I trust this code" (CI/dev mode)
- **Secure defaults**: Manual caps by default (security-first)
- **Clear error messages**: "You need IO and FS, run with --caps IO,FS"

---

## Evidence from AI Eval

**Affected Benchmarks**: fizzbuzz, adt_option
**Models Affected**: All models
**Failure Pattern**: 35+ failures with "effect 'IO' requires capability, but none provided"

### Example Failure

```
Error: execution failed: effect 'IO' requires capability, but none provided
Hint: Run with --caps IO
```

**Generated Code** (correct AILANG, but fails due to missing caps):
```ailang
module benchmark/solution

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, World!")
}
```

**Current workaround**: Manually add `--caps IO` to benchmark harness config.

**Problem**: This doesn't scale. New benchmarks forget to add caps. Multi-effect programs require trial-and-error to discover all needed caps.

---

## Proposed Solution

Implement **two-tier capability system**:

### 1. Entrypoint Effect Preflight (Always On)

Before running a program, **statically analyze** the entrypoint's effect signature and report required capabilities:

```bash
# Preflight automatically runs before execution
$ ailang run hello.ail

⚠️  Capability Check:
    Entrypoint 'main' requires: IO

    Run with: ailang run --caps IO hello.ail
    Or use:   ailang run --auto-caps hello.ail

Error: Missing required capabilities
Exit code: 2  # Distinct from runtime errors (1)
```

**Multi-effect example**:
```bash
$ ailang run file_processor.ail

⚠️  Capability Check:
    Entrypoint 'main' requires: IO, FS

    Run with: ailang run --caps IO,FS file_processor.ail
    Or use:   ailang run --auto-caps file_processor.ail

Error: Missing required capabilities
```

**Benefits**:
- ✅ User sees **all** required caps upfront
- ✅ No trial-and-error discovery
- ✅ Clear, actionable error message
- ✅ Distinct exit code (2) for automation

### 2. Auto-Caps Flag (Opt-In)

Add `--auto-caps` flag to **automatically grant only the inferred capabilities**:

```bash
# Development/CI mode: Just run it
$ ailang run --auto-caps hello.ail
✓ Auto-granted capabilities: IO
Hello, World!

# Explicit mode: Security-conscious
$ ailang run --caps IO hello.ail
Hello, World!

# No caps: Fails with preflight
$ ailang run hello.ail
⚠️  Capability Check: Entrypoint requires: IO
Error: Missing required capabilities
```

**Environment variable** for CI:
```bash
export AILANG_AUTO_CAPS=1
ailang run hello.ail  # Auto-grants IO
```

**Logging**:
```bash
$ ailang run --auto-caps --verbose program.ail
→ Analyzing entrypoint 'main'...
→ Required effects: {IO, FS}
→ Auto-granting capabilities: IO, FS
✓ Running with capabilities: [IO, FS]
```

### 3. Benchmark Harness Integration

Update eval harness to use auto-caps:

```go
// internal/eval_harness/runner.go
func (r *Runner) Execute(benchmark Benchmark) Result {
    // Always use auto-caps for benchmarks
    cmd := exec.Command("ailang", "run", "--auto-caps",
                        "--entry", benchmark.Entrypoint,
                        benchmark.File)
    // ...
}
```

**Benefits**:
- ✅ No manual cap management in benchmark specs
- ✅ Auto-adapts to new effects (e.g., Net in future)
- ✅ Maintains security defaults elsewhere

---

## Technical Design

### Architecture

```
┌─────────────────────────────────────────────────────┐
│  User runs: ailang run [--auto-caps] program.ail   │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │  1. Load Module       │
         │  2. Type Check        │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │  3. Extract Effects   │───► module.GetEntrypoint("main")
         │     from Entrypoint   │     → Type: () -> () ! {IO, FS}
         └───────────┬───────────┘     → Extract: {IO, FS}
                     │
                     ▼
         ┌───────────────────────┐
         │  4. Capability        │
         │     Preflight         │
         └───────────┬───────────┘
                     │
        ┌────────────┴──────────────┐
        │                           │
        ▼                           ▼
┌──────────────┐          ┌─────────────────┐
│ Auto-Caps?   │          │ Manual Caps?    │
│ YES          │          │ YES             │
└──────┬───────┘          └─────┬───────────┘
       │                         │
       │                         ▼
       │                  ┌──────────────┐
       │                  │ Caps Match?  │
       │                  └──────┬───────┘
       │                         │
       │                  ┌──────┴──────┐
       │                  │             │
       │                  ▼             ▼
       │            ┌─────────┐   ┌─────────┐
       │            │ YES     │   │ NO      │
       │            └────┬────┘   └────┬────┘
       │                 │             │
       │                 │             ▼
       │                 │      ┌─────────────────┐
       │                 │      │ Error: Missing  │
       │                 │      │ caps (exit 2)   │
       │                 │      └─────────────────┘
       │                 │
       ▼                 ▼
┌──────────────────────────┐
│  5. Grant Capabilities   │
│     to Runtime Context   │
└─────────┬────────────────┘
          │
          ▼
┌──────────────────────────┐
│  6. Execute Program      │
└──────────────────────────┘
```

### Implementation Components

#### 1. Effect Extraction (`internal/effects/analysis.go`, ~80 LOC)

```go
// ExtractRequiredCapabilities analyzes an entrypoint's type signature
// and returns the set of required capabilities.
func ExtractRequiredCapabilities(mod *module.Module, entrypoint string) ([]string, error) {
    entry := mod.GetExport(entrypoint)
    if entry == nil {
        return nil, fmt.Errorf("entrypoint '%s' not found", entrypoint)
    }

    // Extract effect row from type: () -> T ! {IO, FS}
    effectRow := entry.Type.Effects
    caps := make([]string, 0, len(effectRow))

    for _, eff := range effectRow {
        caps = append(caps, eff.Name)
    }

    return caps, nil
}
```

**Tests**:
- Pure functions → empty caps
- IO effects → ["IO"]
- Multiple effects → ["IO", "FS"]
- Polymorphic effects → concrete caps only

#### 2. Preflight Check (`internal/runtime/preflight.go`, ~120 LOC)

```go
// PreflightCheck validates capabilities before execution.
// Returns: (granted_caps, missing_caps, error)
func PreflightCheck(required []string, provided []string, autoCaps bool) ([]string, []string, error) {
    if autoCaps {
        // Auto-caps: grant all required
        return required, nil, nil
    }

    // Manual mode: check provided caps
    missing := setDifference(required, provided)
    if len(missing) > 0 {
        return provided, missing, &MissingCapsError{
            Required: required,
            Provided: provided,
            Missing:  missing,
        }
    }

    return provided, nil, nil
}

type MissingCapsError struct {
    Required []string
    Provided []string
    Missing  []string
}

func (e *MissingCapsError) Error() string {
    return fmt.Sprintf(
        "\n⚠️  Capability Check:\n" +
        "    Entrypoint requires: %s\n" +
        "    \n" +
        "    Run with: ailang run --caps %s <file>\n" +
        "    Or use:   ailang run --auto-caps <file>\n",
        strings.Join(e.Required, ", "),
        strings.Join(e.Required, ","),
    )
}

func (e *MissingCapsError) ExitCode() int {
    return 2  // Distinct from runtime errors (1)
}
```

#### 3. CLI Integration (`cmd/ailang/main.go`, ~50 LOC)

```go
// Add --auto-caps flag
var (
    autoCaps = flag.Bool("auto-caps", false, "Automatically grant required capabilities")
    capsFlag = flag.String("caps", "", "Manually specify capabilities (comma-separated)")
)

// In runCommand():
func runCommand(args []string) error {
    // Parse flags...

    // Check environment variable
    if os.Getenv("AILANG_AUTO_CAPS") == "1" {
        *autoCaps = true
    }

    // Load module and extract required caps
    mod, err := loader.Load(filename)
    required, err := effects.ExtractRequiredCapabilities(mod, entrypoint)

    // Preflight check
    provided := parseCapsList(*capsFlag)
    granted, missing, err := runtime.PreflightCheck(required, provided, *autoCaps)

    if err != nil {
        if mcErr, ok := err.(*runtime.MissingCapsError); ok {
            fmt.Fprintln(os.Stderr, mcErr)
            os.Exit(2)  // Distinct exit code
        }
        return err
    }

    // Log granted caps if verbose
    if *verbose && *autoCaps {
        fmt.Printf("→ Auto-granted capabilities: %s\n", strings.Join(granted, ", "))
    }

    // Execute with granted capabilities
    ctx := effects.NewContext(granted)
    result, err := runtime.Execute(mod, entrypoint, ctx)
    // ...
}
```

#### 4. Benchmark Harness (`internal/eval_harness/runner.go`, ~40 LOC)

```go
// Always use auto-caps for benchmarks
func (r *Runner) buildCommand(b *Benchmark) *exec.Cmd {
    args := []string{
        "run",
        "--auto-caps",  // ← New: auto-grant caps
        "--entry", b.Entrypoint,
    }

    if b.ArgsJSON != "" {
        args = append(args, "--args-json", b.ArgsJSON)
    }

    args = append(args, b.OutputFile)
    return exec.Command("ailang", args...)
}
```

#### 5. Capability Manifest (`internal/effects/manifest.go`, ~80 LOC)

Optional JSON output for tooling:

```bash
$ ailang inspect --caps program.ail
{
  "entrypoint": "main",
  "required_capabilities": ["IO", "FS"],
  "effect_signature": "() -> () ! {IO, FS}"
}
```

---

## Implementation Plan

### Phase 1: Core Infrastructure (Day 1, ~200 LOC)

**Tasks**:
1. ✅ Create `internal/effects/analysis.go` - Effect extraction (~80 LOC)
2. ✅ Create `internal/runtime/preflight.go` - Preflight check (~120 LOC)
3. ✅ Add tests for both packages (~100 LOC)

**Tests**:
- Extract caps from pure functions (empty)
- Extract caps from IO functions (["IO"])
- Extract caps from multi-effect functions (["IO", "FS"])
- Preflight with auto-caps (grants all)
- Preflight with matching manual caps (success)
- Preflight with missing caps (error with exit code 2)

### Phase 2: CLI Integration (Day 1-2, ~100 LOC)

**Tasks**:
1. ✅ Add `--auto-caps` flag to `cmd/ailang/main.go` (~30 LOC)
2. ✅ Add `AILANG_AUTO_CAPS` env var support (~10 LOC)
3. ✅ Wire preflight into run command (~40 LOC)
4. ✅ Add verbose logging for granted caps (~20 LOC)

**Tests**:
- CLI: `--auto-caps` grants required caps
- CLI: `AILANG_AUTO_CAPS=1` grants caps
- CLI: Missing caps shows preflight error
- CLI: Exit code 2 for missing caps

### Phase 3: Benchmark Harness (Day 2, ~50 LOC)

**Tasks**:
1. ✅ Update `internal/eval_harness/runner.go` - Add `--auto-caps` (~40 LOC)
2. ✅ Remove manual cap specs from benchmark YAMLs (~10 LOC)
3. ✅ Test benchmarks run without manual caps

**Tests**:
- Benchmark with IO runs successfully
- Benchmark with no effects runs without caps
- Benchmark with multiple effects gets all caps

### Phase 4: Documentation & Examples (Day 2, ~70 LOC)

**Tasks**:
1. ✅ Update README.md - Document `--auto-caps` flag
2. ✅ Update docs/guides/capabilities.md - Preflight guide
3. ✅ Add example: `examples/auto_caps_demo.ail`
4. ✅ Update CHANGELOG.md

---

## Testing Strategy

### Unit Tests

**`internal/effects/analysis_test.go`**:
- `TestExtractCaps_Pure` - Pure function returns empty caps
- `TestExtractCaps_IO` - IO function returns ["IO"]
- `TestExtractCaps_Multi` - Multi-effect returns all caps
- `TestExtractCaps_Missing` - Invalid entrypoint returns error

**`internal/runtime/preflight_test.go`**:
- `TestPreflight_AutoCaps` - Auto-caps grants all required
- `TestPreflight_ManualMatch` - Manual caps match required
- `TestPreflight_ManualMissing` - Manual caps missing some
- `TestPreflight_ExitCode` - MissingCapsError has exit code 2

### Integration Tests

**End-to-end CLI tests**:
```bash
# Test auto-caps
ailang run --auto-caps examples/hello_io.ail
# Expected: Runs successfully, prints "Hello, World!"

# Test preflight error
ailang run examples/hello_io.ail
# Expected: Exit 2, shows preflight message

# Test manual caps
ailang run --caps IO examples/hello_io.ail
# Expected: Runs successfully

# Test env var
AILANG_AUTO_CAPS=1 ailang run examples/hello_io.ail
# Expected: Runs successfully
```

**Benchmark tests**:
```bash
# Test benchmark harness
make eval-suite
# Expected: All benchmarks run with auto-granted caps
```

---

## Success Criteria

- [ ] Preflight shows required caps before execution
- [ ] `--auto-caps` flag auto-grants only required caps
- [ ] `AILANG_AUTO_CAPS=1` env var works in CI
- [ ] Missing caps exit with code 2 (distinct from runtime error 1)
- [ ] Benchmark harness uses auto-caps, no manual specs needed
- [ ] Verbose mode logs granted caps
- [ ] All unit tests pass (90%+ coverage)
- [ ] All integration tests pass
- [ ] Documentation updated

---

## Security Considerations

### Secure by Default

**Default behavior** (no `--auto-caps`):
- ❌ Fails with preflight error
- ✅ User must explicitly grant caps
- ✅ Security-conscious workflows unaffected

**Opt-in behavior** (`--auto-caps`):
- ✅ Only grants **required** caps (not all caps)
- ✅ Logs granted caps in verbose mode
- ✅ Appropriate for:
  - Development/testing (trusted code)
  - CI/benchmarks (generated code)
  - Scripts (convenience)
- ❌ NOT for:
  - Production untrusted code
  - Security-critical deployments

### Capability Restrictions

Even with `--auto-caps`, certain capabilities may be restricted:

```go
// Future: Capability whitelisting
var SafeAutoCaps = []string{"IO", "FS"}

func PreflightCheck(...) {
    if autoCaps {
        // Only auto-grant safe caps
        for _, cap := range required {
            if !contains(SafeAutoCaps, cap) {
                return nil, nil, fmt.Errorf(
                    "capability '%s' requires explicit --caps (not auto-grantable)",
                    cap,
                )
            }
        }
    }
}
```

**Example**: Future `Net` capability might require explicit grant even with `--auto-caps`.

---

## Alternatives Considered

### 1. Always Auto-Grant (Rejected)

**Pros**: Maximum convenience
**Cons**: Security hole, violates principle of least privilege

**Decision**: Rejected. Explicit security by default is core to AILANG's design.

### 2. Infer Caps at Compile-Time Only (Rejected)

**Pros**: No runtime overhead
**Cons**: Requires separate "inspect" command, can't enforce at runtime

**Decision**: Rejected. Preflight should be automatic and seamless.

### 3. Per-Benchmark Cap Specs (Current State)

**Pros**: Explicit control
**Cons**: Manual maintenance, easy to forget, doesn't scale

**Decision**: Replace with auto-caps for benchmarks.

---

## References

### Related Docs
- [Effect System](../implemented/v0_2/effects_system.md) - Current capability implementation
- [M-EVAL Runtime Errors](./20251006_runtime_error_ailang_runtime_errors.md) - Original discovery
- [CLI Documentation](../../docs/cli.md) - Command-line interface

### Related Issues
- Float Equality Bug (FIXED in v0.3.3) - Same design doc originally
- M-EVAL Benchmark Failures - 35 failures due to missing caps

### Similar Systems
- **Deno**: `--allow-net`, `--allow-read`, `--allow-write` (explicit)
- **Docker**: `--cap-add` (explicit)
- **SELinux**: Policy-based capability management

AILANG's approach: **Explicit by default, auto-infer for trusted contexts**.

---

## Estimated Impact

### Before Implementation
- ❌ Every IO program requires manual `--caps IO`
- ❌ Benchmark failures due to forgotten caps
- ❌ Trial-and-error to discover multi-effect requirements
- ❌ Poor developer experience for simple scripts

### After Implementation
- ✅ Preflight shows all required caps upfront
- ✅ `--auto-caps` for dev/CI workflows
- ✅ Benchmarks auto-adapt to new effects
- ✅ Clear, actionable error messages
- ✅ Distinct exit codes for automation

**Projected benchmark improvement**:
- **Before**: 35 failures (missing caps)
- **After**: 0 failures (auto-granted)
- **Impact**: +15-20% AI success rate (projected)

---

## Migration Plan

### Backward Compatibility

**No breaking changes**:
- `--caps` flag still works exactly as before
- Default behavior unchanged (explicit caps required)
- `--auto-caps` is opt-in

### Benchmark Harness Migration

**Step 1**: Add `--auto-caps` to runner (Day 2)
**Step 2**: Remove manual `caps:` from YAML specs (optional cleanup)
**Step 3**: Monitor benchmark results (expect 0 cap-related failures)

### Documentation Updates

**Update**:
- README.md - Add `--auto-caps` to CLI reference
- docs/guides/capabilities.md - Explain preflight and auto-caps
- examples/ - Add auto_caps_demo.ail
- CHANGELOG.md - Document new flag

---

*Created: 2025-10-13*
*Priority: P2 (Medium - UX Enhancement)*
*Estimated: 420 LOC, 2 days*

# M-R2: Minimal Effect Runtime

**Milestone**: M-R2 (Effect System Runtime)
**Version**: v0.2.0
**Timeline**: 1–1.5 weeks
**Estimated LOC**: ~700–900 lines
**Priority**: HIGH (depends on M-R1)

---

## Executive Summary

M-R2 implements a **minimal, safe effect runtime** that brings type-level effect tracking (already complete in v0.1.0) into the runtime world. Effects like `IO` and `FS` will execute with capability-based security: no capability = runtime error.

**Philosophy**: Secure by default, simple to use, easy to audit.

**Current State**: Effect syntax parses ✅, type-checks ✅, but has no runtime behavior ❌
**Target State**: Effects execute with capability tokens; unauthorized access denied ✅

---

## Problem Statement

### The Effect Gap (v0.1.0)

In v0.1.0, effects exist only at the type level:

```ailang
-- This type-checks perfectly
func readConfig() -> String ! {FS} {
  readFile("config.txt")
}
```

**What happens at runtime**:
- Type checker validates `! {FS}` annotation ✅
- Effect row tracked in type ✅
- But `readFile()` call... does nothing ❌

**Why**: No runtime effect handlers exist. The `readFile` function is just a stub.

### What Users Need

Users need effects to **actually do something**:

```ailang
module examples/config

import std/fs (readFile, FS)

export func loadConfig() -> String ! {FS} {
  readFile("config.yaml")
}
```

```bash
$ ailang run examples/config.ail --caps FS
# Should actually read config.yaml
```

**Without `--caps FS`**:
```bash
$ ailang run examples/config.ail
Error: EFFECT_CAP_MISSING
  Effect 'FS' requires capability, but none provided
  Hint: Run with --caps FS
```

---

## Goals & Non-Goals

### Goals

1. **Capability-Based Security**: Effects require explicit capability grants
2. **IO Effect**: `print`, `println`, `readLine` work
3. **FS Effect**: `readFile`, `writeFile`, `exists` work (sandboxed in v0.2.0)
4. **Clear Errors**: Missing capabilities produce helpful error messages
5. **Deterministic Testing**: Effects respect `AILANG_SEED`, `TZ`, `LANG` env vars

### Non-Goals (Deferred)

- ❌ Effect combinators/composition DSL (v0.3+)
- ❌ Custom effect handlers (`handle ... with ...` syntax) (v0.3+)
- ❌ Effect budgets (rate limiting, quotas) (v0.3+)
- ❌ Async effects (v0.3+)
- ❌ Net, Clock effects (add in v0.2.1 if time allows)
- ❌ Effect polymorphism (`forall e. ... ! e`) (v0.3+)

---

## Design

### Architecture Overview

```
┌─────────────────┐
│ User Code       │
│ println("Hi!")  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Effect Call     │  Check capability in EffContext
│ (runtime check) │
└────────┬────────┘
         │
    ✓ Has Cap     ✗ No Cap
         │             │
         ▼             ▼
┌─────────────┐  ┌──────────────┐
│ Execute Op  │  │ Throw Error  │
│ (ioPrintln) │  │ CAP_MISSING  │
└─────────────┘  └──────────────┘
```

### Key Data Structures

#### 1. Capability

**File**: `internal/effects/capability.go`
**Size**: ~50 LOC

```go
// Capability represents a granted runtime capability
type Capability struct {
    Name string // "IO", "FS", "Net", etc.

    // Meta holds optional metadata for future use
    // Examples: budgets, tracing context, sandboxing rules
    Meta map[string]any
}

// NewCapability creates a basic capability
func NewCapability(name string) Capability {
    return Capability{
        Name: name,
        Meta: make(map[string]any),
    }
}
```

#### 2. EffContext

**File**: `internal/effects/context.go`
**Size**: ~100 LOC

```go
// EffContext holds runtime capability grants
type EffContext struct {
    Caps map[string]Capability // Effect name → Capability

    // Env holds environment configuration for deterministic effects
    Env EffEnv
}

// EffEnv provides deterministic effect execution
type EffEnv struct {
    Seed    int64  // AILANG_SEED for reproducible randomness
    TZ      string // TZ for deterministic time
    Locale  string // LANG for deterministic string operations
    Sandbox string // Root directory for FS operations (if set)
}

// NewEffContext creates a new effect context
func NewEffContext() *EffContext {
    return &EffContext{
        Caps: make(map[string]Capability),
        Env:  loadEffEnv(),
    }
}

// Grant adds a capability to the context
func (ctx *EffContext) Grant(cap Capability) {
    ctx.Caps[cap.Name] = cap
}

// HasCap checks if a capability is granted
func (ctx *EffContext) HasCap(name string) bool {
    _, ok := ctx.Caps[name]
    return ok
}

// RequireCap checks for a capability; errors if missing
func (ctx *EffContext) RequireCap(name string) error {
    if !ctx.HasCap(name) {
        return NewCapabilityError(name)
    }
    return nil
}

// loadEffEnv loads effect environment from OS environment
func loadEffEnv() EffEnv {
    seed := int64(0)
    if seedStr := os.Getenv("AILANG_SEED"); seedStr != "" {
        if s, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
            seed = s
        }
    }

    return EffEnv{
        Seed:    seed,
        TZ:      getEnv("TZ", "UTC"),
        Locale:  getEnv("LANG", "C"),
        Sandbox: os.Getenv("AILANG_FS_SANDBOX"),
    }
}
```

#### 3. Effect Operations

**File**: `internal/effects/ops.go`
**Size**: ~150 LOC

```go
// EffOp is a function that implements an effect operation
type EffOp func(ctx *EffContext, args []eval.Value) (eval.Value, error)

// Registry holds all effect operations
var Registry = map[string]map[string]EffOp{
    "IO": {
        "print":    ioPrint,
        "println":  ioPrintln,
        "readLine": ioReadLine,
    },
    "FS": {
        "readFile":  fsReadFile,
        "writeFile": fsWriteFile,
        "exists":    fsExists,
    },
}

// Call invokes an effect operation
func Call(ctx *EffContext, effectName, opName string, args []eval.Value) (eval.Value, error) {
    // Check capability
    if err := ctx.RequireCap(effectName); err != nil {
        return nil, err
    }

    // Lookup operation
    effectOps, ok := Registry[effectName]
    if !ok {
        return nil, fmt.Errorf("unknown effect: %s", effectName)
    }

    op, ok := effectOps[opName]
    if !ok {
        return nil, fmt.Errorf("unknown operation %s in effect %s", opName, effectName)
    }

    // Execute operation
    return op(ctx, args)
}
```

### Built-in Effects

#### IO Effect Operations

**File**: `internal/effects/io.go`
**Size**: ~150 LOC

```go
// ioPrint implements IO.print(s: String) -> ()
func ioPrint(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("print: expected 1 argument, got %d", len(args))
    }

    str, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("print: expected String, got %T", args[0])
    }

    fmt.Print(str.Value)
    return &eval.UnitValue{}, nil
}

// ioPrintln implements IO.println(s: String) -> ()
func ioPrintln(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("println: expected 1 argument, got %d", len(args))
    }

    str, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("println: expected String, got %T", args[0])
    }

    fmt.Println(str.Value)
    return &eval.UnitValue{}, nil
}

// ioReadLine implements IO.readLine() -> String
func ioReadLine(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 0 {
        return nil, fmt.Errorf("readLine: expected 0 arguments, got %d", len(args))
    }

    reader := bufio.NewReader(os.Stdin)
    line, err := reader.ReadString('\n')
    if err != nil {
        if err == io.EOF {
            return &eval.StringValue{Value: ""}, nil
        }
        return nil, fmt.Errorf("readLine: %w", err)
    }

    // Trim trailing newline
    line = strings.TrimSuffix(line, "\n")
    return &eval.StringValue{Value: line}, nil
}
```

#### FS Effect Operations

**File**: `internal/effects/fs.go`
**Size**: ~200 LOC

```go
// fsReadFile implements FS.readFile(path: String) -> String
func fsReadFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("readFile: expected 1 argument, got %d", len(args))
    }

    pathVal, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("readFile: expected String, got %T", args[0])
    }

    path := pathVal.Value

    // Apply sandbox if configured
    if ctx.Env.Sandbox != "" {
        path = filepath.Join(ctx.Env.Sandbox, path)
    }

    // Read file
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("readFile: %w", err)
    }

    return &eval.StringValue{Value: string(content)}, nil
}

// fsWriteFile implements FS.writeFile(path: String, content: String) -> ()
func fsWriteFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("writeFile: expected 2 arguments, got %d", len(args))
    }

    pathVal, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("writeFile: expected String for path, got %T", args[0])
    }

    contentVal, ok := args[1].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("writeFile: expected String for content, got %T", args[1])
    }

    path := pathVal.Value
    content := contentVal.Value

    // Apply sandbox
    if ctx.Env.Sandbox != "" {
        path = filepath.Join(ctx.Env.Sandbox, path)
    }

    // Write file (0644 permissions)
    err := os.WriteFile(path, []byte(content), 0644)
    if err != nil {
        return nil, fmt.Errorf("writeFile: %w", err)
    }

    return &eval.UnitValue{}, nil
}

// fsExists implements FS.exists(path: String) -> Bool
func fsExists(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("exists: expected 1 argument, got %d", len(args))
    }

    pathVal, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("exists: expected String, got %T", args[0])
    }

    path := pathVal.Value

    // Apply sandbox
    if ctx.Env.Sandbox != "" {
        path = filepath.Join(ctx.Env.Sandbox, path)
    }

    _, err := os.Stat(path)
    exists := err == nil

    return &eval.BoolValue{Value: exists}, nil
}
```

### Integration with Evaluator

**File**: `internal/eval/eval_core.go` (modifications)
**Changes**: ~100 LOC

```go
// Add EffContext to CoreEvaluator
type CoreEvaluator struct {
    env         *Environment
    registry    *types.DictionaryRegistry
    resolver    GlobalResolver
    effContext  *effects.EffContext // NEW: Effect context

    experimentalBinopShim bool
}

// SetEffContext sets the effect context for this evaluator
func (e *CoreEvaluator) SetEffContext(ctx *effects.EffContext) {
    e.effContext = ctx
}

// Handle effect operations during evaluation
func (e *CoreEvaluator) evalEffectOp(effectName, opName string, args []Value) (Value, error) {
    if e.effContext == nil {
        return nil, fmt.Errorf("effect operation %s.%s called without effect context", effectName, opName)
    }

    return effects.Call(e.effContext, effectName, opName, args)
}
```

### CLI Integration

**File**: `cmd/ailang/main.go` (modifications)
**Changes**: ~50 LOC

```go
// Add --caps flag
var capsFlag string
flag.StringVar(&capsFlag, "caps", "", "Enable capabilities (comma-separated: IO,FS,Net)")

// Parse capabilities
func parseCaps(capsStr string) []effects.Capability {
    if capsStr == "" {
        return nil
    }

    var caps []effects.Capability
    for _, name := range strings.Split(capsStr, ",") {
        name = strings.TrimSpace(name)
        if name != "" {
            caps = append(caps, effects.NewCapability(name))
        }
    }
    return caps
}

// In run command:
// Create effect context
effCtx := effects.NewEffContext()
for _, cap := range parseCaps(capsFlag) {
    effCtx.Grant(cap)
}

// Pass to evaluator
evaluator.SetEffContext(effCtx)
```

---

## Implementation Plan

### Phase 1: Core Infrastructure (Days 1-2)

**Goal**: Set up effect context and capability system

**Tasks**:
1. Create `internal/effects/` package
2. Implement `Capability` and `EffContext` (~150 LOC)
3. Implement effect operation registry (~100 LOC)
4. Write unit tests (~200 LOC)

**Deliverable**: Compiling effect infrastructure

### Phase 2: IO Effect (Days 3-4)

**Goal**: Implement IO operations

**Tasks**:
1. Implement `ioPrint`, `ioPrintln`, `ioReadLine` (~150 LOC)
2. Wire to evaluator (~50 LOC)
3. Add CLI `--caps` flag (~50 LOC)
4. Write IO tests (~250 LOC)

**Deliverable**: IO effects working

### Phase 3: FS Effect (Days 5-6)

**Goal**: Implement FS operations with sandboxing

**Tasks**:
1. Implement `fsReadFile`, `fsWriteFile`, `fsExists` (~200 LOC)
2. Add sandbox support (~50 LOC)
3. Write FS tests with temp dirs (~300 LOC)

**Deliverable**: FS effects working safely

### Phase 4: Integration & Testing (Days 7-8)

**Goal**: End-to-end integration

**Tasks**:
1. Update stdlib wrappers to call effect ops
2. Test with real examples
3. Error message polish
4. Documentation

**Deliverable**: Production-ready M-R2

---

## Testing Strategy

### Unit Tests (~750 LOC)

**Test Files**:
- `internal/effects/capability_test.go` (~100 LOC)
- `internal/effects/context_test.go` (~150 LOC)
- `internal/effects/io_test.go` (~250 LOC)
- `internal/effects/fs_test.go` (~250 LOC)

**Test Cases**:

1. **Capability Management**
   - Grant capability
   - Check has capability
   - Require capability (success/failure)

2. **IO Operations**
   - Print/println output capture
   - ReadLine from string input
   - Missing IO capability → error

3. **FS Operations**
   - Read existing file
   - Write new file
   - File existence check
   - Missing FS capability → error
   - Sandbox enforcement

4. **Effect Context**
   - Multiple capabilities
   - Environment loading (AILANG_SEED, etc.)
   - Deterministic behavior

### Integration Tests (~300 LOC)

**Test Files**:
- `tests/integration/effects_test.go` (~300 LOC)

**Test Cases**:

1. **IO Example**
   ```ailang
   module test/io_demo
   import std/io (println, IO)
   export func main() -> () ! {IO} {
     println("Hello from effects!")
   }
   ```
   Expected: Prints "Hello from effects!"

2. **FS Example**
   ```ailang
   module test/fs_demo
   import std/fs (readFile, FS)
   export func main() -> String ! {FS} {
     readFile("config.txt")
   }
   ```
   Expected: Returns file contents

3. **Denied Capability**
   ```bash
   $ ailang run test.ail  # No --caps flag
   Error: EFFECT_CAP_MISSING
   ```

### Example Verification

**Target**: All `examples/effects_*.ail` work

**Priority Examples**:
- `examples/effects_pure.ail` - Already works (no runtime effects)
- `examples/demos/hello_io.ail` - Should print with `--caps IO`
- `examples/effects_basic.ail` (create) - Basic IO demo

---

## Error Handling

### Error Types

**File**: `internal/effects/errors.go`
**Size**: ~50 LOC

```go
// CapabilityError represents a missing capability error
type CapabilityError struct {
    Effect string
}

func (e *CapabilityError) Error() string {
    return fmt.Sprintf("effect '%s' requires capability, but none provided", e.Effect)
}

func NewCapabilityError(effect string) *CapabilityError {
    return &CapabilityError{Effect: effect}
}
```

### Error Messages

1. **Missing Capability**
   ```
   Error: EFFECT_CAP_MISSING
     Effect 'IO' requires capability, but none provided

     This function has effect signature: () -> () ! {IO}

     Fix: Run with --caps IO
     Example: ailang run file.ail --caps IO
   ```

2. **Invalid Capability Name**
   ```
   Error: Unknown capability 'FOO'
     Valid capabilities: IO, FS, Net, Clock
   ```

3. **Effect Operation Error**
   ```
   Error: IO operation 'println' failed
     Cause: broken pipe

     This may indicate a problem with stdout redirection.
   ```

---

## Security Considerations

### Sandbox Mode

**Environment Variable**: `AILANG_FS_SANDBOX`

```bash
# Restrict FS operations to /tmp/sandbox
export AILANG_FS_SANDBOX=/tmp/sandbox
ailang run app.ail --caps FS
```

All FS paths are joined with sandbox root:
- `readFile("config.txt")` → `/tmp/sandbox/config.txt`
- `readFile("/etc/passwd")` → `/tmp/sandbox/etc/passwd` (safe!)

### Capability Deny by Default

No capabilities granted unless explicitly requested:

```bash
# Safe: No file access
ailang run app.ail

# Dangerous: Full FS access (user must opt-in)
ailang run app.ail --caps FS
```

### Future: Capability Budgets (v0.3+)

```go
type Capability struct {
    Name string
    Meta map[string]any {
        "io.max_writes": 100,     // Limit writes
        "fs.max_bytes": 1048576,  // 1MB limit
        "fs.allowed_dirs": []string{"/tmp"},
    }
}
```

---

## Performance Considerations

### Effect Call Overhead

**Target**: <100ns per capability check

**Strategy**:
- Capability map lookup is O(1)
- No reflection or dynamic dispatch
- Direct function calls for operations

**Benchmark** (to be written):
```go
func BenchmarkEffectCall(b *testing.B) {
    ctx := effects.NewEffContext()
    ctx.Grant(effects.NewCapability("IO"))
    args := []eval.Value{&eval.StringValue{Value: "test"}}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        effects.Call(ctx, "IO", "println", args)
    }
}
```

### IO Buffering

- `print` calls are unbuffered (immediate stdout write)
- For performance-critical code, batch prints or use buffered IO in future

---

## Stdlib Integration

### Update std/io

**File**: `stdlib/std/io.ail` (modifications)

```ailang
module stdlib/std/io

-- Effect marker (already exists)
effect IO

-- Builtin wrappers (already exist in equation form)
export func print(s: String) -> () ! {IO} = _io_print(s)
export func println(s: String) -> () ! {IO} = _io_println(s)
export func readLine() -> String ! {IO} = _io_readLine()

-- NEW: Document capabilities
-- @requires IO capability
-- @deterministic No (uses real stdout)
```

### Create std/fs

**File**: `stdlib/std/fs.ail` (NEW)

```ailang
module stdlib/std/fs

-- Effect marker
effect FS

-- File operations
export func readFile(path: String) -> String ! {FS} = _fs_readFile(path)
export func writeFile(path: String, content: String) -> () ! {FS} = _fs_writeFile(path, content)
export func exists(path: String) -> Bool ! {FS} = _fs_exists(path)

-- @requires FS capability
-- @sandbox Respects AILANG_FS_SANDBOX
```

---

## Future Extensions (Post-v0.2.0)

### v0.2.1: Additional Effects
- **Net**: `httpGet`, `httpPost`
- **Clock**: `now`, `sleep`

### v0.3.0: Effect Handlers
- Custom handlers: `handle { ... } with { case Op => ... }`
- Effect composition
- Resumption

### v0.3.0: Effect Budgets
- Rate limiting per capability
- Resource quotas (bytes read/written, requests/sec)

---

## Acceptance Criteria

### Minimum Success

- ✅ IO effect: print, println, readLine work
- ✅ FS effect: readFile, writeFile, exists work
- ✅ Capability checking works (deny if missing)
- ✅ `--caps` flag functional
- ✅ Sandbox mode works (`AILANG_FS_SANDBOX`)
- ✅ 5+ examples pass with effects

### Stretch Goals

- ✅ Net effect (httpGet basic)
- ✅ 10+ examples pass
- ✅ Comprehensive error messages
- ✅ Performance <100ns/check

---

## Dependencies

### Upstream (Must Exist Before M-R2)

- ✅ M-R1 (Module Execution) - Need runtime infrastructure
- ✅ Effect syntax parsing (complete)
- ✅ Effect type-checking (complete)

### Downstream (Depend on M-R2)

- ✅ Effect examples in `examples/`
- ✅ Stdlib effects (`std/io`, `std/fs`)

---

## Risks & Mitigation

| Risk | Impact | Mitigation | Fallback |
|------|--------|-----------|----------|
| Effect overhead too high | Med | Optimize capability checks; benchmark | Add `--no-caps-check` for trusted code |
| Sandbox bypass | High | Test with path traversal; review security | Document limitations; improve in v0.2.1 |
| stdlib not callable | High | Test all wrappers; fix equation-form calls | Inline effect ops if needed |

---

## Status

**Status**: Design complete, ready for implementation
**Depends On**: M-R1 (module execution runtime)
**Blocks**: Example upgrades, effect documentation

---

**Document Version**: v1.0
**Created**: 2025-10-02
**Last Updated**: 2025-10-02
**Author**: AILANG Development Team

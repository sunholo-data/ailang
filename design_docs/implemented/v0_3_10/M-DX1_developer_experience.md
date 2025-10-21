# M-DX1: Developer Experience Improvements

**Status:** 📋 Planned
**Target:** v0.3.9
**Priority:** P0 (High ROI - 70% time reduction)
**Estimated:** 3 days (parallelizable)
**Dependencies:** None

## Problem Statement

Adding a new builtin function to AILANG currently takes **7.5 hours** due to:
- Scattered registration across 4 files (2+ hours debugging)
- Verbose type construction with 35 lines of nested structs (1+ hour trial-and-error)
- Poor error messages that don't indicate WHERE or HOW to fix issues (1+ hour debugging)
- Manual testing workflow without validation tools
- Lack of documentation for new contributors

**Impact:** This friction slows down language evolution and makes it hard for new contributors to add features.

## Friction Log Summary

From implementing `_net_httpRequest` (real data):

```
Time breakdown:
├── Implementation:              4.0 hours (actual code)
├── Debugging registration:      2.0 hours ← PAIN POINT #1
├── Debugging types:             1.0 hour  ← PAIN POINT #2
├── Debugging syntax:            0.5 hours
└── Total:                       7.5 hours

Files touched per builtin:
├── internal/effects/net.go         (676 LOC) - Implementation
├── internal/runtime/builtins.go    (229 LOC) - Runtime wrapper
├── internal/builtins/registry.go   (134 LOC) - Metadata
└── internal/link/builtin_module.go (487 LOC) - Type sigs + exports
    Total: 4 files, ~1,526 LOC to navigate
```

**Root Causes:**
1. No centralized registration → updates in 4 places → easy to forget one
2. Verbose type construction → TVar vs TApp confusion → trial-and-error
3. Poor error messages → "undefined global variable" doesn't say WHERE to fix
4. No validation tooling → errors found at runtime, not compile/startup
5. Manual testing → rebuild → run → check by eye (slow iteration)

## Goals

**Primary Goal:** Reduce builtin development time from 7.5 hours to ~2.5 hours (70% improvement)

**Success Metrics:**
- Files touched: 4 → 1 (-75%)
- Type construction LOC: 35 → 10 (-71%)
- Registration errors caught at compile/startup (not runtime)
- New contributor can add builtin in <3 hours (vs 7.5 hours)

## Solution: Fast Wins Strategy

Ship 7 improvements in v0.3.9 (3 days, parallelizable):

### Day 1: Core Infrastructure
1. ✅ **Central Builtin Registration** (M-DX1.1)
2. ✅ **Type Builder DSL** (M-DX1.2)

### Day 2: Validation & Testing
3. ✅ **Doctor Command + Rich Errors** (M-DX1.3)
4. ✅ **Builtin Test Harness** (M-DX1.4)

### Day 3: Developer Tools
5. ✅ **REPL Type Queries** (M-DX1.5)
6. ✅ **Tailored Error Diagnostics** (M-DX1.6)
7. ✅ **Documentation** (M-DX1.7)

---

## M-DX1.1: Central Builtin Registration

**Goal:** Add a builtin once, in one place.

**Estimated:** ~400 LOC implementation + ~150 LOC tests = **~550 LOC**
**Duration:** 4-6 hours (half day)

### Current State (Scattered)

```
Adding _net_httpRequest requires updates in 4 files:

1. internal/effects/net.go
   func netHTTPRequest(...) { ... }                    // Implementation

2. internal/runtime/builtins.go
   case "_net_httpRequest":
     result, err := effects.netHTTPRequest(...)        // Runtime wrapper

3. internal/builtins/registry.go
   "_net_httpRequest": {NumArgs: 4, IsPure: false},    // Metadata

4. internal/link/builtin_module.go
   "_net_httpRequest": <35 lines of type construction> // Type signature
   exports = append(exports, "_net_httpRequest")       // Export list
```

**Problem:** Easy to forget step 3 or 4 → runtime error with no guidance.

### Proposed State (Centralized)

```go
// internal/builtins/registry.go - SINGLE POINT OF REGISTRATION

type BuiltinSpec struct {
    Name    string
    NumArgs int
    IsPure  bool
    Effect  string // "", "Net", "IO", "FS"
    Type    func() types.Type
    Impl    func(*effects.EffContext, []eval.Value) (eval.Value, error)
}

func RegisterEffectBuiltin(spec BuiltinSpec) error {
    // 1) Validate (name clash, missing impl, missing type)
    // 2) Wire to runtime/builtins.go (add to dispatch table)
    // 3) Register metadata (arity, purity)
    // 4) Add to link/builtin_module.go (type + export)
    return nil
}

// Usage: Register _net_httpRequest in ONE place
func init() {
    RegisterEffectBuiltin(BuiltinSpec{
        Name:    "_net_httpRequest",
        NumArgs: 4,
        IsPure:  false,
        Effect:  "Net",
        Type:    makeHTTPRequestType,  // See M-DX1.2 for readable builder
        Impl:    effects.netHTTPRequest,
    })
}
```

### Implementation Plan

**Files to create/modify:**
- `internal/builtins/registry.go` (~150 LOC new)
  - `BuiltinSpec` struct
  - `RegisterEffectBuiltin()` function
  - Validation logic (duplicates, nil checks)
  - Registry storage (map[string]BuiltinSpec)

- `internal/builtins/registry_test.go` (~100 LOC new)
  - Test registration validates names
  - Test duplicate detection
  - Test missing impl/type errors

- `internal/runtime/builtins.go` (~50 LOC modified)
  - Replace hardcoded dispatch with registry lookup
  - Keep backward compatibility during migration

- `internal/link/builtin_module.go` (~50 LOC modified)
  - Generate interface from registry
  - Generate exports from registry

- `CLAUDE.md` (~50 lines modified)
  - Update "Adding a New Language Feature" section
  - Document new registration pattern

**Migration Strategy:**
1. Implement registry infrastructure (day 1 morning)
2. Migrate 1-2 Net builtins as proof of concept (day 1 afternoon)
3. Verify all tests pass
4. Mark old scattered approach as deprecated
5. Complete migration in v0.3.10 (incremental)

### Acceptance Criteria

- [x] Adding new builtin requires editing only 1 file (`internal/builtins/registry.go`)
- [x] Registration includes: impl, type, metadata, export (all in BuiltinSpec)
- [x] Compile error if `Impl` function signature doesn't match
- [x] Startup error if type function returns nil
- [x] 1-2 existing builtins migrated and working
- [x] All existing tests pass (no regressions)

### Example Usage

```go
// Before: 4 files, 50+ lines scattered
// After: 1 file, ~15 lines centralized

RegisterEffectBuiltin(BuiltinSpec{
    Name:    "_net_httpRequest",
    NumArgs: 4,
    IsPure:  false,
    Effect:  "Net",
    Type:    makeHTTPRequestType,
    Impl:    effects.netHTTPRequest,
})

func makeHTTPRequestType() types.Type {
    // See M-DX1.2 for readable builder syntax
    T := types.NewBuilder()
    return T.Func(
        T.String(), T.String(),
        T.List(T.Record(Field("name", T.String()), Field("value", T.String()))),
        T.String(),
    ).Returns(
        T.App("Result", T.Record(...), T.Con("NetError")),
    ).Effects("Net")
}
```

---

## M-DX1.2: Type Builder DSL

**Goal:** Define complex types in ~10 readable lines (not 35 lines of nested structs).

**Estimated:** ~250 LOC implementation + ~100 LOC tests = **~350 LOC**
**Duration:** 4-6 hours (half day)

### Current State (Nested Structs)

```go
// internal/link/builtin_module.go - httpRequest type (35 lines, unreadable)
"_net_httpRequest": &types.TFunc2{
    Args: []types.Type{
        &types.TCon{Name: "String"},
        &types.TCon{Name: "String"},
        &types.TApp{
            Con: &types.TCon{Name: "List"},
            Arg: &types.TRecord{
                Fields: map[string]types.Type{
                    "name":  &types.TCon{Name: "String"},
                    "value": &types.TCon{Name: "String"},
                },
                Tail: nil,
            },
        },
        &types.TCon{Name: "String"},
    },
    Ret: &types.TApp{
        Con: &types.TCon{Name: "Result"},
        Arg: &types.TRecord{
            Fields: map[string]types.Type{
                "status":  &types.TCon{Name: "Int"},
                "headers": &types.TApp{ /* nested list */ },
                "body":    &types.TCon{Name: "String"},
                "ok":      &types.TCon{Name: "Bool"},
            },
            Tail: nil,
        },
    },
    EffectRow: &types.TEffectRow{
        Effects: map[string]types.Type{
            "Net": &types.TCon{Name: "Net"},
        },
        Tail: nil,
    },
}
```

**Problems:**
- Hard to read (what's the actual signature?)
- Easy mistakes: `TVar` vs `TApp`, forgetting `Tail: nil`
- Trial-and-error to get it right
- Copy-paste errors when cloning similar types

### Proposed State (Fluent Builder)

```go
// internal/types/builder.go - Readable, compile-time safe

func makeHTTPRequestType() types.Type {
    T := types.NewBuilder()

    headerType := T.Record(
        Field("name", T.String()),
        Field("value", T.String()),
    )

    responseType := T.Record(
        Field("status", T.Int()),
        Field("headers", T.List(headerType)),
        Field("body", T.String()),
        Field("ok", T.Bool()),
    )

    return T.Func(
        T.String(),        // method
        T.String(),        // url
        T.List(headerType), // headers
        T.String(),        // body
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net")
}
```

**Benefits:**
- ✅ Self-documenting (clear structure)
- ✅ Compile-time safety (wrong field name = compile error)
- ✅ Reusable (extract `headerType` once)
- ✅ 10 lines vs 35 lines (-71%)

### Implementation Plan

**Files to create:**
- `internal/types/builder.go` (~250 LOC new)
  - `Builder` struct
  - Primitive constructors: `String()`, `Int()`, `Bool()`, `Float()`
  - Complex constructors: `List()`, `Con()`, `App()`, `Record()`
  - `Func()` with fluent `.Returns()` and `.Effects()`
  - Helper: `Field(name, type)` for record fields

- `internal/types/builder_test.go` (~100 LOC new)
  - Test basic types match expected structs
  - Test complex nested types
  - Test httpRequest example matches current implementation
  - Test compile-time safety (wrong usage = compile error)

**API Design:**

```go
package types

type Builder struct{}

func NewBuilder() *Builder {
    return &Builder{}
}

// Primitives
func (b *Builder) String() Type { return &TCon{Name: "String"} }
func (b *Builder) Int() Type    { return &TCon{Name: "Int"} }
func (b *Builder) Bool() Type   { return &TCon{Name: "Bool"} }
func (b *Builder) Float() Type  { return &TCon{Name: "Float"} }

// Constructors
func (b *Builder) Con(name string) Type {
    return &TCon{Name: name}
}

func (b *Builder) App(con string, args ...Type) Type {
    // Handle Result<T, E>, List<T>, etc.
    result := &TCon{Name: con}
    for _, arg := range args {
        result = &TApp{Con: result, Arg: arg}
    }
    return result
}

func (b *Builder) List(elem Type) Type {
    return &TApp{Con: &TCon{Name: "List"}, Arg: elem}
}

func (b *Builder) Record(fields ...FieldSpec) Type {
    m := make(map[string]Type)
    for _, f := range fields {
        m[f.Name] = f.Type
    }
    return &TRecord{Fields: m, Tail: nil}
}

// Function types (fluent API)
func (b *Builder) Func(args ...Type) *FuncBuilder {
    return &FuncBuilder{builder: b, args: args}
}

type FuncBuilder struct {
    builder *Builder
    args    []Type
    ret     Type
    effects []string
}

func (fb *FuncBuilder) Returns(ret Type) *FuncBuilder {
    fb.ret = ret
    return fb
}

func (fb *FuncBuilder) Effects(eff ...string) Type {
    row := &TEffectRow{Effects: make(map[string]Type), Tail: nil}
    for _, e := range eff {
        row.Effects[e] = &TCon{Name: e}
    }
    return &TFunc2{Args: fb.args, Ret: fb.ret, EffectRow: row}
}

// Helper for record fields
type FieldSpec struct {
    Name string
    Type Type
}

func Field(name string, typ Type) FieldSpec {
    return FieldSpec{Name: name, Type: typ}
}
```

### Acceptance Criteria

- [x] Builder API implemented with fluent interface
- [x] `httpRequest` type built with builder matches current implementation
- [x] Complex types reduced from 35 lines to ~10 lines
- [x] Compile-time validation (wrong type name = compile error)
- [x] 2-3 existing type signatures migrated as examples
- [x] All tests pass (unit tests + existing integration tests)

### Migration Plan

1. Create `internal/types/builder.go` with full API
2. Write comprehensive unit tests
3. Migrate `httpRequest` type as proof of concept
4. Migrate 1-2 other complex types (e.g., `readFile`, `writeFile`)
5. Document in `docs/ADDING_BUILTINS.md`
6. Mark old nested struct approach as deprecated

---

## M-DX1.3: Doctor Command + Rich Errors

**Goal:** Fail fast with actionable guidance (WHERE and HOW to fix).

**Estimated:** ~200 LOC implementation + ~80 LOC tests = **~280 LOC**
**Duration:** 4-6 hours (half day)

### Current State (Cryptic Errors)

```
Error: undefined global variable '_net_httpRequest'

← No hint WHERE to fix (which file?)
← No hint WHAT is wrong (missing registration? typo? wrong module?)
← Developer has to:
  1. Binary search through 4 files
  2. Compare with working builtin
  3. Guess which step was forgotten
```

### Proposed State (Helpful Errors)

```bash
# 1. Startup validation (optional, dev mode only)
$ ailang run examples/ai_call.ail
Error: Builtin validation failed (3 issues)

  ✗ '_net_httpRequest' exported but not registered
    → Fix: Add to internal/builtins/registry.go:
       RegisterEffectBuiltin(BuiltinSpec{
         Name: "_net_httpRequest",
         ...
       })

  ✗ '_io_debug' registered but missing type signature
    → Fix: Add 'Type: makeDebugType' to BuiltinSpec

  ✗ '_fs_readFile' has arity mismatch (registered: 2, impl: 1)
    → Fix: Update NumArgs in BuiltinSpec or impl signature

# 2. Doctor command (explicit validation)
$ ailang doctor builtins
Checking builtin consistency...

✓ 45 builtins registered
✓ 45 have type signatures
✓ 45 have implementations
✗ 3 issues found (see above)

Registered builtins by effect:
  Net: _net_httpGet, _net_httpPost, _net_httpRequest
  IO:  _io_print, _io_println, _io_readLine
  FS:  _fs_readFile, _fs_writeFile, _fs_listDir
  ...

# 3. List command
$ ailang builtins list
Net builtins (3):
  _net_httpGet      : (String) -> Result<String, NetError> ! {Net}
  _net_httpPost     : (String, String) -> Result<String, NetError> ! {Net}
  _net_httpRequest  : (String, String, List<Header>, String) -> Result<Response, NetError> ! {Net}

IO builtins (5):
  _io_print         : (String) -> () ! {IO}
  _io_println       : (String) -> () ! {IO}
  ...

# 4. Better runtime error
$ ailang run examples/broken.ail
Error: Builtin '_net_htpRequest' not found in $builtin module.

Did you mean: _net_httpRequest?

Hint: If this is a new builtin, add to internal/builtins/registry.go:
  RegisterEffectBuiltin(BuiltinSpec{
    Name: "_net_htpRequest",
    ...
  })

Registered Net builtins: _net_httpGet, _net_httpPost, _net_httpRequest
```

### Implementation Plan

**Files to create/modify:**
- `internal/builtins/validator.go` (~150 LOC new)
  - `ValidateBuiltins() []ValidationError`
  - Cross-check: registered vs typed vs exported
  - Arity validation (NumArgs vs impl signature)
  - Name clash detection
  - Missing impl/type detection

- `internal/builtins/validator_test.go` (~50 LOC new)
  - Test each validation rule
  - Test error messages include fixes

- `cmd/ailang/doctor.go` (~50 LOC new)
  - `ailang doctor builtins` subcommand
  - Pretty-print validation errors
  - Exit code 1 if errors found

- `cmd/ailang/builtins.go` (~30 LOC new)
  - `ailang builtins list` subcommand
  - Group by effect, show signatures

- `internal/errors/builtin_errors.go` (~50 LOC new)
  - Enhanced `BuiltinNotFoundError(name)`
  - Similarity matching (Levenshtein distance)
  - Hints with file locations and example code

- `cmd/ailang/main.go` (~20 LOC modified)
  - Optional: startup validation in dev mode
  - Flag: `--validate-builtins` (default: false)

**Validation Rules:**

```go
type ValidationError struct {
    Builtin  string
    Message  string
    Fix      string
    Location string // "internal/builtins/registry.go:42"
}

func ValidateBuiltins() []ValidationError {
    errors := []ValidationError{}

    // 1. Check: All registered builtins have implementations
    for name, spec := range Registry {
        if spec.Impl == nil {
            errors = append(errors, ValidationError{
                Builtin:  name,
                Message:  "Missing implementation",
                Fix:      "Add 'Impl: effects.funcName' to BuiltinSpec",
                Location: "internal/builtins/registry.go",
            })
        }
    }

    // 2. Check: All registered builtins have type signatures
    for name, spec := range Registry {
        if spec.Type == nil {
            errors = append(errors, ValidationError{
                Builtin:  name,
                Message:  "Missing type signature",
                Fix:      "Add 'Type: makeTypeFunc' to BuiltinSpec",
                Location: "internal/builtins/registry.go",
            })
        }
    }

    // 3. Check: All exported builtins are registered
    exported := link.GetBuiltinInterface().Exports
    for _, exp := range exported {
        if _, ok := Registry[exp]; !ok {
            errors = append(errors, ValidationError{
                Builtin:  exp,
                Message:  "Exported but not registered",
                Fix:      fmt.Sprintf("Add RegisterEffectBuiltin(\"%s\", ...) or remove from exports", exp),
                Location: "internal/link/builtin_module.go",
            })
        }
    }

    // 4. Check: No name clashes
    seen := make(map[string]bool)
    for name := range Registry {
        if seen[name] {
            errors = append(errors, ValidationError{
                Builtin:  name,
                Message:  "Duplicate registration",
                Fix:      "Remove duplicate RegisterEffectBuiltin() call",
                Location: "internal/builtins/registry.go",
            })
        }
        seen[name] = true
    }

    return errors
}
```

**Similarity Matching:**

```go
func findSimilar(target string, candidates []string) []string {
    type match struct {
        name     string
        distance int
    }
    matches := []match{}

    for _, cand := range candidates {
        dist := levenshtein(target, cand)
        if dist <= 3 { // Max 3 edits
            matches = append(matches, match{cand, dist})
        }
    }

    // Sort by distance
    sort.Slice(matches, func(i, j int) bool {
        return matches[i].distance < matches[j].distance
    })

    // Return top 3
    result := []string{}
    for i := 0; i < len(matches) && i < 3; i++ {
        result = append(result, matches[i].name)
    }
    return result
}
```

### Acceptance Criteria

- [x] `ailang doctor builtins` shows all registration issues
- [x] Errors show WHERE to fix (file + line if possible)
- [x] Errors show HOW to fix (example code snippet)
- [x] `ailang builtins list` shows all registered builtins by effect
- [x] Runtime error for missing builtin includes similarity suggestions
- [x] Startup validation catches common mistakes (if enabled)
- [x] All tests pass

---

## M-DX1.4: Builtin Test Harness

**Goal:** Test effect builtins without compiling a full program.

**Estimated:** ~200 LOC implementation + ~150 LOC tests = **~350 LOC**
**Duration:** 4-6 hours (half day)

### Current State (Manual Testing)

```
To test _net_httpRequest:
1. Write .ail file with httpRequest call
2. Run: ailang run --caps Net --net-allow=api.example.com test.ail
3. Check output by eye
4. Modify code
5. Repeat...

Problems:
- Slow (rebuild + parse + typecheck + eval each time)
- Manual (no automated assertions)
- Limited coverage (hard to test error cases)
- No mocking (real network calls)
```

### Proposed State (Unit Tests)

```go
// internal/effects/net_test.go
func TestNetHTTPRequest_Success(t *testing.T) {
    ctx := testctx.NewMockEffContext("Net")

    // Setup: Mock HTTP client
    ctx.SetHTTPClient(mockHTTPClient(200, "OK", []byte("response body")))

    // Input: method, url, headers, body
    method := eval.TaggedValue{Tag: "String", Value: "GET"}
    url := eval.TaggedValue{Tag: "String", Value: "https://api.example.com"}
    headers := eval.MakeList()
    body := eval.TaggedValue{Tag: "String", Value: ""}

    // Execute
    result, err := netHTTPRequest(ctx, []eval.Value{method, url, headers, body})

    // Assert
    require.NoError(t, err)
    okVariant := result.(eval.TaggedValue)
    assert.Equal(t, "Ok", okVariant.Tag)

    response := okVariant.Value.(eval.RecordValue)
    assert.Equal(t, 200, response.Get("status"))
    assert.Equal(t, "response body", response.Get("body"))
    assert.Equal(t, true, response.Get("ok"))
}

func TestNetHTTPRequest_Timeout(t *testing.T) {
    ctx := testctx.NewMockEffContext("Net")
    ctx.SetNetTimeout(1 * time.Millisecond)
    ctx.SetHTTPClient(mockSlowClient(5 * time.Second))

    // ... execute ...

    // Assert: Err(Transport("timeout"))
    errVariant := result.(eval.TaggedValue)
    assert.Equal(t, "Err", errVariant.Tag)

    netErr := errVariant.Value.(eval.TaggedValue)
    assert.Equal(t, "Transport", netErr.Tag)
    assert.Contains(t, netErr.Value.(string), "timeout")
}

func TestNetHTTPRequest_DisallowedHost(t *testing.T) {
    ctx := testctx.NewMockEffContext("Net")
    ctx.SetAllowedHosts([]string{"api.example.com"}) // api.evil.com not allowed

    url := eval.TaggedValue{Tag: "String", Value: "https://api.evil.com"}
    // ... execute ...

    // Assert: Err(DisallowedHost("api.evil.com"))
    errVariant := result.(eval.TaggedValue)
    assert.Equal(t, "Err", errVariant.Tag)
    assert.Equal(t, "DisallowedHost", errVariant.Value.(eval.TaggedValue).Tag)
}
```

### Implementation Plan

**Files to create:**
- `internal/effects/testctx/mock_context.go` (~150 LOC new)
  - `MockEffContext` struct
  - `NewMockEffContext(caps ...string) *MockEffContext`
  - Mock HTTP client setup
  - Mock FS setup (for future FS tests)
  - Helper methods for common scenarios

- `internal/effects/testctx/helpers.go` (~50 LOC new)
  - `MakeString(s string) eval.Value`
  - `MakeInt(i int) eval.Value`
  - `MakeBool(b bool) eval.Value`
  - `MakeList(items ...eval.Value) eval.Value`
  - `MakeRecord(fields map[string]eval.Value) eval.Value`
  - Extractors: `GetString(v)`, `GetInt(v)`, `GetBool(v)`, etc.

**Files to modify:**
- `internal/effects/net_test.go` (~150 LOC new tests)
  - `TestNetHTTPRequest_Success`
  - `TestNetHTTPRequest_4xx`
  - `TestNetHTTPRequest_5xx`
  - `TestNetHTTPRequest_Timeout`
  - `TestNetHTTPRequest_InvalidHeader`
  - `TestNetHTTPRequest_DisallowedHost`
  - `TestNetHTTPRequest_OriginChange` (redirects)

**Mock Context API:**

```go
package testctx

import (
    "net/http"
    "time"
    "github.com/yourusername/ailang/internal/effects"
    "github.com/yourusername/ailang/internal/eval"
)

type MockEffContext struct {
    capabilities map[string]bool
    httpClient   *http.Client
    allowedHosts []string
    timeout      time.Duration
    // Future: fsRoot, mockFiles, etc.
}

func NewMockEffContext(caps ...string) *MockEffContext {
    ctx := &MockEffContext{
        capabilities: make(map[string]bool),
        allowedHosts: []string{}, // Allow all by default in tests
        timeout:      30 * time.Second,
    }
    for _, cap := range caps {
        ctx.capabilities[cap] = true
    }
    return ctx
}

func (m *MockEffContext) HasCapability(cap string) bool {
    return m.capabilities[cap]
}

func (m *MockEffContext) SetHTTPClient(client *http.Client) {
    m.httpClient = client
}

func (m *MockEffContext) SetAllowedHosts(hosts []string) {
    m.allowedHosts = hosts
}

func (m *MockEffContext) SetNetTimeout(timeout time.Duration) {
    m.timeout = timeout
}

func (m *MockEffContext) GetHTTPClient() *http.Client {
    if m.httpClient == nil {
        return http.DefaultClient
    }
    return m.httpClient
}

// ... implement effects.EffContext interface ...
```

**Helper Functions:**

```go
package testctx

import "github.com/yourusername/ailang/internal/eval"

func MakeString(s string) eval.Value {
    return eval.TaggedValue{Tag: "String", Value: s}
}

func MakeInt(i int) eval.Value {
    return eval.TaggedValue{Tag: "Int", Value: i}
}

func MakeBool(b bool) eval.Value {
    return eval.TaggedValue{Tag: "Bool", Value: b}
}

func MakeList(items ...eval.Value) eval.Value {
    return eval.ListValue{Items: items}
}

func MakeRecord(fields map[string]eval.Value) eval.Value {
    return eval.RecordValue{Fields: fields}
}

func GetString(v eval.Value) string {
    return v.(eval.TaggedValue).Value.(string)
}

func GetInt(v eval.Value) int {
    return v.(eval.TaggedValue).Value.(int)
}

// ... more helpers ...
```

### Acceptance Criteria

- [x] `MockEffContext` implements `effects.EffContext` interface
- [x] 6-10 tests for `netHTTPRequest` covering:
  - [x] Success (200 OK)
  - [x] Client error (4xx)
  - [x] Server error (5xx)
  - [x] Timeout
  - [x] Invalid header
  - [x] Disallowed host
  - [x] Optional: Origin change (redirects)
- [x] Helper functions for constructing/inspecting values
- [x] Tests run fast (<1s total) without real network calls
- [x] All tests pass

---

## M-DX1.5: REPL Type Queries

**Goal:** "What's the type of this?" without binary-searching code.

**Estimated:** ~150 LOC implementation + ~50 LOC tests = **~200 LOC**
**Duration:** 3-4 hours (half day)

### Current State (No Introspection)

```
REPL> concat_String("hello", "world")
"helloworld" : String

← What's the type of concat_String itself?
← No way to ask!
← Have to read source code or docs
```

### Proposed State (Type Queries)

```
# 1. REPL :type command
REPL> :type concat_String
concat_String : (String, String) -> String

REPL> :type concat_String("hello", "world")
"helloworld" : String

REPL> :type httpRequest
httpRequest : (String, String, List<{name: String, value: String}>, String)
              -> Result<{status: Int, headers: List<{name: String, value: String}>, body: String, ok: Bool}, NetError>
              ! {Net}

# 2. REPL :explain command (optional, nice-to-have)
REPL> let x = concat_String(42, "world")
Type error: Expected String, got Int
  In argument 1 of concat_String
  At line 1, column 25

REPL> :explain
Type inference trail:
  1. concat_String : (String, String) -> String
  2. First argument: 42 : Int
  3. Expected: String
  4. Got: Int
  5. Unification failed: Int ≠ String

# 3. CLI type check (optional, nice-to-have)
$ ailang check --line examples/ai_call.ail:58
Line 58, column 10-20: DisallowedHost(host)
  Type: NetError
  Context: Pattern match in Err branch
```

### Implementation Plan

**Files to modify:**
- `internal/repl/repl.go` (~100 LOC modified)
  - Add `:type <expr>` command handler
  - Parse expression, run typechecker, print result
  - Handle both expressions and identifiers
  - Pretty-print types (use existing type formatter)

- `internal/repl/repl.go` (~50 LOC modified, optional)
  - Add `:explain` command handler
  - Store last type error with full inference trail
  - Pretty-print inference steps

- `cmd/ailang/check.go` (~50 LOC new, optional)
  - `ailang check --line <file:line>` subcommand
  - Parse file, find expression at line
  - Run typechecker, print type
  - Exit code 1 if type error

**REPL Type Command Implementation:**

```go
// internal/repl/repl.go

func (r *REPL) handleTypeCommand(expr string) {
    // Parse expression
    p := parser.New(lexer.New(expr))
    ast, err := p.ParseExpression()
    if err != nil {
        fmt.Printf("Parse error: %v\n", err)
        return
    }

    // Typecheck (use existing pipeline)
    typed, err := r.pipeline.TypeCheck(ast)
    if err != nil {
        fmt.Printf("Type error: %v\n", err)
        return
    }

    // Pretty-print type
    fmt.Printf("%s : %s\n", expr, types.Format(typed.Type))
}

func (r *REPL) handleCommand(line string) bool {
    switch {
    case strings.HasPrefix(line, ":type "):
        expr := strings.TrimPrefix(line, ":type ")
        r.handleTypeCommand(expr)
        return true

    case line == ":explain":
        r.handleExplainCommand()
        return true

    case line == ":quit" || line == ":q":
        return false

    case line == ":help" || line == ":?":
        r.printHelp()
        return true

    default:
        return false // Not a command, evaluate as expression
    }
}

func (r *REPL) printHelp() {
    fmt.Println("REPL Commands:")
    fmt.Println("  :type <expr>  Show the type of an expression")
    fmt.Println("  :explain      Show detailed type error from last failure")
    fmt.Println("  :help         Show this help")
    fmt.Println("  :quit         Exit REPL")
}
```

**Type Inference Trail (Optional):**

```go
// internal/types/inference.go

type InferenceStep struct {
    Description string
    Type        Type
    Location    ast.Position
}

type InferenceTrail struct {
    Steps []InferenceStep
    Error error
}

func (tc *TypeChecker) RecordStep(desc string, typ Type, loc ast.Position) {
    if tc.trail != nil {
        tc.trail.Steps = append(tc.trail.Steps, InferenceStep{
            Description: desc,
            Type:        typ,
            Location:    loc,
        })
    }
}

// Usage in typechecker:
tc.RecordStep(fmt.Sprintf("Inferred argument %d", i), argType, arg.Position())
```

### Acceptance Criteria

- [x] `:type <expr>` shows type of expression in REPL
- [x] `:type <ident>` shows type of variable/function
- [x] Works for builtins, user functions, and complex expressions
- [x] Pretty-prints complex types (multi-line if needed)
- [x] `:help` documents all REPL commands
- [x] Optional: `:explain` shows type error details
- [x] Optional: `ailang check --line` shows type at cursor
- [x] All tests pass

---

## M-DX1.6: Tailored Error Diagnostics

**Goal:** Save 30-60 minutes per confusion with specific hints.

**Estimated:** ~150 LOC implementation + ~50 LOC tests = **~200 LOC**
**Duration:** 3-4 hours (half day)

### Current State (Generic Errors)

```
# 1. Function call in if condition
if httpRequest("GET", url, [], "") { ... }
Error: Expected Bool, got Result<Response, NetError>

← Unhelpful: Doesn't explain WHY or HOW to fix

# 2. Nullary constructor
match x {
  Foo() => ...  // Error: Foo is not a function
}

← Confusing: ADT constructors look like functions but aren't

# 3. Block returning ()
if cond then {
  println("yes");
  42
} else {
  println("no");
  99
}
Error: Type mismatch in if branches (Int vs ())

← Unclear: Where did () come from?
```

### Proposed State (Helpful Hints)

```ailang
# 1. Function call in if condition
if httpRequest("GET", url, [], "") { ... }
Error: Type mismatch in if condition
  Expected: Bool
  Got: Result<Response, NetError>

Hint: AILANG doesn't allow function calls directly in if conditions.
      Extract the call to a let binding first:

        let result = httpRequest("GET", url, [], "");
        if isOk(result) then { ... }

      See: docs/LIMITATIONS.md#no-effectful-conditionals

# 2. Nullary constructor
match x {
  Foo() => ...
}
Error: Type error in pattern
  Foo is a nullary constructor, not a function.

Hint: Nullary constructors don't use parentheses in patterns:

        match x {
          Foo => ...    -- Correct
          Foo() => ...  -- Wrong (treated as function call)
        }

      See: docs/guides/pattern-matching.md#nullary-constructors

# 3. Block returning ()
if cond then {
  println("yes");
  42
} else {
  println("no");
  99
}
Error: Type mismatch in if branches
  Expected: Int (from then branch)
  Got: () (from else branch)

Hint: Block expressions return the value of their last expression.
      The println in the else branch returns (), not 99.

      Fix: Add 99 as the final expression:

        if cond then {
          println("yes");
          42
        } else {
          println("no");
          99          -- This is now the return value
        }

      See: docs/syntax.md#block-expressions
```

### Implementation Plan

**Files to modify:**
- `internal/errors/diagnostics.go` (~150 LOC new)
  - Pattern matching for common error scenarios
  - Generate tailored hints with example code
  - Links to documentation

- `internal/types/typechecker.go` (~50 LOC modified)
  - Detect specific error patterns
  - Attach diagnostic context to type errors
  - Use enhanced error formatting

**Error Pattern Detection:**

```go
// internal/errors/diagnostics.go

type Diagnostic struct {
    Message     string
    Hint        string
    Example     string
    DocLink     string
}

func EnhanceTypeError(err error, context *ErrorContext) error {
    // 1. Function call in if condition
    if isEffectfulCondition(err, context) {
        return &EnhancedError{
            Original: err,
            Diagnostic: Diagnostic{
                Message: "AILANG doesn't allow function calls directly in if conditions.",
                Hint:    "Extract the call to a let binding first",
                Example: `
    let result = httpRequest("GET", url, [], "");
    if isOk(result) then { ... }`,
                DocLink: "docs/LIMITATIONS.md#no-effectful-conditionals",
            },
        }
    }

    // 2. Nullary constructor with parentheses
    if isNullaryConstructorCall(err, context) {
        return &EnhancedError{
            Original: err,
            Diagnostic: Diagnostic{
                Message: "Nullary constructors don't use parentheses in patterns.",
                Hint:    "Remove the () from the pattern",
                Example: `
    match x {
      Foo => ...    -- Correct
      Foo() => ...  -- Wrong
    }`,
                DocLink: "docs/guides/pattern-matching.md#nullary-constructors",
            },
        }
    }

    // 3. Block returning ()
    if isBlockReturningUnit(err, context) {
        return &EnhancedError{
            Original: err,
            Diagnostic: Diagnostic{
                Message: "Block expressions return the value of their last expression.",
                Hint:    "Ensure the last expression in each branch has the expected type",
                Example: `
    if cond then {
      println("yes");
      42          -- This is the return value
    } else {
      println("no");
      99          -- This is the return value
    }`,
                DocLink: "docs/syntax.md#block-expressions",
            },
        }
    }

    // No pattern matched, return original error
    return err
}

func isEffectfulCondition(err error, ctx *ErrorContext) bool {
    // Check if error is in if condition position
    // Check if expected type is Bool
    // Check if actual type is an effect type (has ! in signature)
    return ctx.IsIfCondition && ctx.ExpectedType == "Bool" && ctx.ActualType.HasEffects()
}

// ... similar pattern matchers ...
```

**Error Context Tracking:**

```go
// internal/types/typechecker.go

type ErrorContext struct {
    IsIfCondition   bool
    IsPatternMatch  bool
    IsBlockExpr     bool
    ExpectedType    types.Type
    ActualType      types.Type
    ASTNode         ast.Node
}

func (tc *TypeChecker) checkIfExpr(expr *ast.IfExpr) (types.Type, error) {
    // Set context for better error messages
    ctx := &ErrorContext{IsIfCondition: true}

    condType, err := tc.checkExpr(expr.Condition)
    if err != nil {
        return nil, errors.EnhanceTypeError(err, ctx)
    }

    ctx.ExpectedType = types.BoolType
    ctx.ActualType = condType

    if !types.Unify(condType, types.BoolType) {
        err := fmt.Errorf("Type mismatch in if condition\n  Expected: %s\n  Got: %s",
            types.Format(types.BoolType), types.Format(condType))
        return nil, errors.EnhanceTypeError(err, ctx)
    }

    // ... rest of if checking ...
}
```

### Acceptance Criteria

- [x] Function call in if condition → Extract to let binding hint
- [x] Nullary constructor with () → Remove () hint
- [x] Block returning () → Add final expression hint
- [x] Each hint includes:
  - [x] Explanation of what's wrong
  - [x] How to fix it
  - [x] Example code
  - [x] Link to docs (if applicable)
- [x] All tests pass
- [x] 3+ test cases for each diagnostic pattern

---

## M-DX1.7: Documentation (ADDING_BUILTINS.md)

**Goal:** Onboard a new contributor in minutes (not hours).

**Estimated:** **~250 lines of markdown**
**Duration:** 2-3 hours (half day)

### Content Outline

```markdown
# Adding a Builtin Function to AILANG

This guide shows how to add a new builtin function to AILANG in ~3 steps using the centralized registry (v0.3.9+).

## Quick Start (10 minutes)

### Step 1: Implement the function

Create your implementation in `internal/effects/<effect>.go`:

\`\`\`go
// internal/effects/string.go
func strLen(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    s := args[0].(eval.TaggedValue).Value.(string)
    return eval.TaggedValue{Tag: "Int", Value: len(s)}, nil
}
\`\`\`

### Step 2: Register it

Add one `RegisterEffectBuiltin()` call in `internal/builtins/registry.go`:

\`\`\`go
// internal/builtins/registry.go
func init() {
    RegisterEffectBuiltin(BuiltinSpec{
        Name:    "_str_len",
        NumArgs: 1,
        IsPure:  true,  // No side effects
        Effect:  "",    // Pure function
        Type:    makeStrLenType,
        Impl:    effects.strLen,
    })
}

func makeStrLenType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.Int())
}
\`\`\`

### Step 3: Test it

\`\`\`bash
$ ailang doctor builtins   # Validate registration
✓ 46 builtins registered
✓ All validations passed

$ ailang repl
REPL> :type _str_len
_str_len : (String) -> Int

REPL> _str_len("hello")
5 : Int
\`\`\`

Done! Your builtin is now available. 🎉

---

## Full Guide

### 1. Choose a Name

Builtin names follow the pattern: `_<effect>_<operation>`

Examples:
- Pure functions: `_str_len`, `_math_sqrt`, `_list_reverse`
- Effect functions: `_io_println`, `_net_httpGet`, `_fs_readFile`

**Why the underscore prefix?** It prevents name clashes with user-defined functions and makes builtins visually distinct.

### 2. Implement the Function

Create your implementation in the appropriate file:
- `internal/effects/io.go` - IO operations (print, readLine, etc.)
- `internal/effects/net.go` - Network operations (httpGet, httpPost, etc.)
- `internal/effects/fs.go` - Filesystem operations (readFile, writeFile, etc.)
- `internal/effects/string.go` - String utilities (new file)
- `internal/effects/math.go` - Math utilities (new file)

**Function Signature:**
\`\`\`go
func yourFunction(ctx *EffContext, args []eval.Value) (eval.Value, error)
\`\`\`

**Example (pure function):**
\`\`\`go
func strLen(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // Extract argument
    s := args[0].(eval.TaggedValue).Value.(string)

    // Compute result
    length := len(s)

    // Return as AILANG value
    return eval.TaggedValue{Tag: "Int", Value: length}, nil
}
\`\`\`

**Example (effect function):**
\`\`\`go
func ioPrintln(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // Check capability
    if !ctx.HasCapability("IO") {
        return nil, fmt.Errorf("IO capability required")
    }

    // Extract argument
    s := args[0].(eval.TaggedValue).Value.(string)

    // Perform effect
    fmt.Println(s)

    // Return unit
    return eval.UnitValue, nil
}
\`\`\`

### 3. Define the Type Signature

Use the **Type Builder API** for readable, compile-time-safe type construction:

**Simple types:**
\`\`\`go
func makeStrLenType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.Int())
}
// Type: (String) -> Int
\`\`\`

**Multiple arguments:**
\`\`\`go
func makeConcatType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String(), T.String()).Returns(T.String())
}
// Type: (String, String) -> String
\`\`\`

**Effect functions:**
\`\`\`go
func makePrintlnType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.Unit()).Effects("IO")
}
// Type: (String) -> () ! {IO}
\`\`\`

**Complex types (lists, records, ADTs):**
\`\`\`go
func makeHTTPRequestType() types.Type {
    T := types.NewBuilder()

    // Define header type once, reuse it
    headerType := T.Record(
        Field("name", T.String()),
        Field("value", T.String()),
    )

    responseType := T.Record(
        Field("status", T.Int()),
        Field("headers", T.List(headerType)),
        Field("body", T.String()),
        Field("ok", T.Bool()),
    )

    return T.Func(
        T.String(),         // method
        T.String(),         // url
        T.List(headerType), // headers
        T.String(),         // body
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net")
}
// Type: (String, String, List<{name: String, value: String}>, String)
//       -> Result<{status: Int, headers: List<...>, body: String, ok: Bool}, NetError>
//       ! {Net}
\`\`\`

**Builder API Reference:**
- Primitives: `T.String()`, `T.Int()`, `T.Bool()`, `T.Float()`, `T.Unit()`
- Constructors: `T.Con("Name")`, `T.App("Con", arg1, arg2, ...)`
- Collections: `T.List(elemType)`, `T.Record(Field("name", type), ...)`
- Functions: `T.Func(arg1, arg2, ...).Returns(retType).Effects("Eff1", "Eff2")`

### 4. Register the Builtin

Add **one registration** in `internal/builtins/registry.go`:

\`\`\`go
// internal/builtins/registry.go
func init() {
    // ... existing registrations ...

    RegisterEffectBuiltin(BuiltinSpec{
        Name:    "_str_len",      // Builtin name (with _ prefix)
        NumArgs: 1,               // Number of arguments
        IsPure:  true,            // true = no side effects, false = has effects
        Effect:  "",              // "" for pure, "IO"/"Net"/"FS" for effects
        Type:    makeStrLenType,  // Type signature function
        Impl:    effects.strLen,  // Implementation function
    })
}
\`\`\`

**BuiltinSpec Fields:**
- `Name`: Builtin name with `_` prefix (e.g., `_str_len`)
- `NumArgs`: Number of arguments (used for arity checking)
- `IsPure`: `true` if no side effects, `false` if has effects
- `Effect`: `""` for pure functions, `"IO"`/`"Net"`/`"FS"` for effect functions
- `Type`: Function that constructs the type signature
- `Impl`: Implementation function (must match signature above)

### 5. Test It

**Automated validation:**
\`\`\`bash
$ ailang doctor builtins
✓ 46 builtins registered
✓ 46 have type signatures
✓ 46 have implementations
✓ All validations passed
\`\`\`

**REPL testing:**
\`\`\`bash
$ ailang repl
REPL> :type _str_len
_str_len : (String) -> Int

REPL> _str_len("hello")
5 : Int

REPL> _str_len("世界")
6 : Int  -- UTF-8 byte count, not character count
\`\`\`

**Unit tests (recommended):**
\`\`\`go
// internal/effects/string_test.go
func TestStrLen(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    result, err := strLen(ctx, []eval.Value{
        testctx.MakeString("hello"),
    })

    require.NoError(t, err)
    assert.Equal(t, 5, testctx.GetInt(result))
}
\`\`\`

### 6. Add Examples (Optional)

Create an example file in `examples/`:

\`\`\`ailang
-- examples/string_length.ail
module examples/string_length

import std/io (println)

func main() -> () ! {IO} {
  let s = "hello";
  let len = _str_len(s);
  _io_println(concat_String("Length: ", show(len)))
}
\`\`\`

Test it:
\`\`\`bash
$ ailang run --caps IO --entry main examples/string_length.ail
Length: 5
\`\`\`

---

## Common Patterns

### Pure Functions (No Effects)

\`\`\`go
RegisterEffectBuiltin(BuiltinSpec{
    Name:    "_str_len",
    NumArgs: 1,
    IsPure:  true,   // ← Pure
    Effect:  "",     // ← No effect
    Type:    makeStrLenType,
    Impl:    effects.strLen,
})
\`\`\`

### Effect Functions

\`\`\`go
RegisterEffectBuiltin(BuiltinSpec{
    Name:    "_io_println",
    NumArgs: 1,
    IsPure:  false,  // ← Has effects
    Effect:  "IO",   // ← IO effect
    Type:    makePrintlnType,
    Impl:    effects.ioPrintln,
})
\`\`\`

### Functions Returning Result<T, E>

\`\`\`go
func makeReadFileType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(
        T.App("Result", T.String(), T.Con("FSError")),
    ).Effects("FS")
}
// Type: (String) -> Result<String, FSError> ! {FS}
\`\`\`

Implementation:
\`\`\`go
func fsReadFile(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    path := args[0].(eval.TaggedValue).Value.(string)

    content, err := os.ReadFile(path)
    if err != nil {
        // Return Err(FSError(...))
        return eval.TaggedValue{
            Tag: "Err",
            Value: eval.TaggedValue{
                Tag:   "ReadFailed",
                Value: err.Error(),
            },
        }, nil
    }

    // Return Ok(content)
    return eval.TaggedValue{
        Tag:   "Ok",
        Value: eval.TaggedValue{Tag: "String", Value: string(content)},
    }, nil
}
\`\`\`

---

## Checklist

Before committing your builtin:

- [ ] Implementation in `internal/effects/<effect>.go`
- [ ] Registration in `internal/builtins/registry.go` (one `RegisterEffectBuiltin()` call)
- [ ] Type signature using Type Builder API
- [ ] `ailang doctor builtins` passes
- [ ] REPL `:type` shows correct signature
- [ ] Unit tests in `internal/effects/<effect>_test.go` (recommended)
- [ ] Example file in `examples/` (optional)
- [ ] Update CHANGELOG.md with new builtin

---

## Troubleshooting

### "Builtin not found" error

\`\`\`
Error: Builtin '_str_len' not found in $builtin module.
\`\`\`

**Fix:** Make sure you called `RegisterEffectBuiltin()` in `internal/builtins/registry.go`.

Run `ailang doctor builtins` to check registration status.

### "Type error: Expected X, got Y"

**Fix:** Your type signature doesn't match your implementation.

Check:
1. Number of arguments (`NumArgs` vs actual args)
2. Argument types (String vs Int vs Bool, etc.)
3. Return type (what your impl returns vs what type signature says)

Use `:type` in REPL to inspect the inferred type.

### "Missing capability" error

\`\`\`
Error: IO capability required
\`\`\`

**Fix:** Your effect function requires a capability that wasn't granted.

Run with: `ailang run --caps IO,Net,FS ...`

Check `ctx.HasCapability("IO")` in your implementation.

### Arity mismatch

\`\`\`
✗ '_str_len' has arity mismatch (registered: 2, impl: 1)
\`\`\`

**Fix:** `NumArgs` in `BuiltinSpec` doesn't match the number of arguments your implementation expects.

Update `NumArgs` to match your `Type` signature and `Impl` function.

---

## Advanced Topics

### Variable-Arity Builtins

Currently not supported. All builtins must have fixed arity.

**Workaround:** Create multiple builtins (e.g., `_str_concat2`, `_str_concat3`) or accept a list.

### Polymorphic Builtins

Currently limited. Type variables are supported but require manual type construction.

**Example (identity function):**
\`\`\`go
func makeIdentityType() types.Type {
    a := &types.TVar{Name: "a"}
    return &types.TFunc2{
        Args: []types.Type{a},
        Ret:  a,
        EffectRow: &types.TEffectRow{Effects: map[string]types.Type{}, Tail: nil},
    }
}
// Type: forall a. (a) -> a
\`\`\`

### Async/Concurrent Builtins

Not yet supported. Builtins are synchronous.

**Future:** v0.4.0+ will add channels and async effects.

---

## Resources

- [CLAUDE.md](../CLAUDE.md) - Development workflow
- [CONTRIBUTING.md](../docs/CONTRIBUTING.md) - Contribution guidelines
- [prompts/v0.3.8.md](../prompts/v0.3.8.md) - AILANG syntax guide
- [docs/LIMITATIONS.md](../docs/LIMITATIONS.md) - Known limitations

---

## Questions?

- File an issue: https://github.com/yourrepo/ailang/issues
- Check existing builtins for examples: `internal/effects/*.go`
- Run `ailang doctor builtins` for validation help
\`\`\`

---

## Implementation Notes

**File location:** `docs/ADDING_BUILTINS.md`

**Cross-references to add:**
- Update `CLAUDE.md` to link to this guide
- Update `docs/CONTRIBUTING.md` to reference this guide
- Add to README.md under "Developer Documentation"

**Estimated time:** 2-3 hours to write, review, and integrate.

---

## Acceptance Criteria

- [x] Complete guide with Quick Start (10 min) section
- [x] Step-by-step instructions with code examples
- [x] Common patterns (pure, effect, Result types)
- [x] Troubleshooting section with common errors
- [x] Checklist for contributors
- [x] Links to related documentation
- [x] Examples from real builtins (strLen, httpRequest)
- [x] Cross-referenced from CLAUDE.md and CONTRIBUTING.md

---

## Timeline & Dependencies

### Day 1: Core Infrastructure (Parallel)
**Morning (4 hours):**
- [ ] M-DX1.1: Central Builtin Registration (~4-6 hours)
  - Implement `BuiltinSpec` and `RegisterEffectBuiltin()`
  - Wire to runtime/builtins.go and link/builtin_module.go
  - Migrate 1-2 builtins as proof of concept
  - Write unit tests

**Afternoon (4 hours):**
- [ ] M-DX1.2: Type Builder DSL (~4-6 hours)
  - Implement fluent Builder API
  - Write unit tests
  - Migrate 2-3 complex type signatures

**Can be done in parallel** - no dependencies between M-DX1.1 and M-DX1.2

### Day 2: Validation & Testing (Parallel)
**Morning (4 hours):**
- [ ] M-DX1.3: Doctor Command + Rich Errors (~4-6 hours)
  - Implement validation logic
  - Add `ailang doctor builtins` command
  - Add `ailang builtins list` command
  - Enhance error messages with hints

**Afternoon (4 hours):**
- [ ] M-DX1.4: Builtin Test Harness (~4-6 hours)
  - Implement `MockEffContext`
  - Add helper functions for value construction
  - Write 6-10 tests for netHTTPRequest

**Can be done in parallel** - M-DX1.3 depends on M-DX1.1, M-DX1.4 is independent

### Day 3: Developer Tools (Parallel)
**Morning (3 hours):**
- [ ] M-DX1.5: REPL Type Queries (~3-4 hours)
  - Add `:type` command to REPL
  - Optional: Add `:explain` command
  - Optional: Add `ailang check --line` CLI

**Afternoon (3 hours):**
- [ ] M-DX1.6: Tailored Error Diagnostics (~3-4 hours)
  - Implement pattern matching for common errors
  - Add hints with example code
  - Write tests for each diagnostic

**Late Afternoon (2 hours):**
- [ ] M-DX1.7: Documentation (~2-3 hours)
  - Write docs/ADDING_BUILTINS.md
  - Cross-reference from CLAUDE.md and CONTRIBUTING.md

**Can be done in parallel** - M-DX1.5, M-DX1.6, M-DX1.7 are independent

---

## Total Estimates

**LOC:**
- Implementation: ~1,450 LOC
- Tests: ~580 LOC
- Docs: ~250 lines markdown
- **Total: ~2,280 LOC**

**Time:**
- Day 1: 8 hours (2 people × 4 hours OR 1 person × 8 hours)
- Day 2: 8 hours (2 people × 4 hours OR 1 person × 8 hours)
- Day 3: 8 hours (3 people × 2-3 hours OR 1 person × 8 hours)
- **Total: 3 days (24 hours) with 1 person, faster with parallelization**

**Dependencies:**
- M-DX1.3 depends on M-DX1.1 (needs registry for validation)
- All others are independent and can be parallelized

---

## Success Metrics

### Quantitative (Measured)
- **Time reduction:** 7.5 hours → 2.5 hours per builtin (target: -67%)
- **Files touched:** 4 → 1 (target: -75%)
- **Type construction LOC:** 35 → 10 (target: -71%)
- **Test coverage:** 28.2% → 30%+ (new code well-tested)
- **Validation coverage:** 100% of registration issues caught by `ailang doctor builtins`

### Qualitative (Observed)
- [ ] Can add new builtin in <3 hours (vs 7.5 hours)
- [ ] Registration errors caught at startup/validation (not runtime)
- [ ] Type construction is readable and self-documenting
- [ ] `ailang doctor builtins` becomes part of dev workflow
- [ ] New contributors can add builtins following docs/ADDING_BUILTINS.md
- [ ] Error messages are actionable (WHERE + HOW to fix)

---

## Deferred to Future Versions

### Out of Scope (v0.3.9)
1. **Parser relaxations** (v0.4.0+)
   - Allow function calls in if conditions (requires language design decision)
   - Auto-infer `()` at end of blocks (needs broader discussion)
   - Estimated: 2-3 days

2. **Builtin test coverage metrics** (v0.4.0+)
   - Track which builtins have tests vs don't
   - Generate coverage report
   - Estimated: 1 day

3. **Auto-generation from //go:builtin annotation** (v0.4.0+)
   - Reflection/codegen for automatic registration
   - Requires significant complexity, defer until proven need
   - Estimated: 3-5 days

### Why Defer
These have lower immediate ROI for the specific pain point (reducing builtin development time from 7.5h to 2.5h). They're valuable but not critical for v0.3.9 goals.

---

## Risks & Mitigation

### Technical Risks

1. **Breaking existing builtins during migration**
   - **Impact:** High (breaks AILANG runtime)
   - **Likelihood:** Medium
   - **Mitigation:**
     - Incremental migration (1-2 builtins at a time)
     - Run full test suite after each migration
     - Keep old scattered approach as deprecated fallback
     - Feature flag for new registry system

2. **Type Builder API too complex/awkward**
   - **Impact:** Medium (doesn't achieve readability goal)
   - **Likelihood:** Low
   - **Mitigation:**
     - Prototype with 2-3 examples first
     - Get feedback before full implementation
     - Iterate on API design based on real usage
     - Keep old nested struct approach as fallback

3. **Validation false positives slow down development**
   - **Impact:** Medium (frustrating developer experience)
   - **Likelihood:** Low
   - **Mitigation:**
     - Make validation opt-in (`ailang doctor`) not automatic
     - Extensive testing with edge cases
     - Allow validation to be disabled with flag

4. **REPL type queries break existing REPL**
   - **Impact:** High (breaks interactive development)
   - **Likelihood:** Low
   - **Mitigation:**
     - Add new commands without modifying existing behavior
     - Comprehensive REPL tests before/after
     - Feature flag for new commands

### Schedule Risks

1. **Scope creep (trying to solve all 7 issues at once)**
   - **Impact:** High (delays v0.3.9 release)
   - **Likelihood:** Medium
   - **Mitigation:**
     - Strict prioritization (M-DX1.1 → M-DX1.2 → M-DX1.3)
     - Can ship partial improvements (e.g., registry without builder)
     - Set clear cut line: ship high-priority items, defer rest

2. **Underestimated complexity**
   - **Impact:** Medium (delays release)
   - **Likelihood:** Medium
   - **Mitigation:**
     - Conservative estimates (added 20-30% buffer)
     - Daily progress tracking
     - Can defer M-DX1.5-1.7 if needed

---

## Implementation Report (Post-Completion)

**To be filled after implementation:**

### Actual Metrics
- LOC implemented: ___
- LOC tests: ___
- Time spent: ___ days
- Test coverage: ___%
- Builtins migrated: ___

### Lessons Learned
- What went well:
- What was harder than expected:
- What would we do differently:

### Known Issues
- Issue 1:
- Issue 2:

### Future Work
- Enhancement 1:
- Enhancement 2:

---

## Approval & Sign-off

**Reviewed by:** ___
**Approved by:** ___
**Date:** ___

**Ready to implement:** [ ] Yes [ ] No

**Notes:**

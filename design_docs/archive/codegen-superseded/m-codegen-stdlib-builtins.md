# M-CODEGEN-STDLIB-BUILTINS: Go Codegen Runtime Implementations for Stdlib Functions

**Status**: Planned
**Target**: v0.9.3
**Priority**: P1 (Blocking — prevents `go build` of any project using stdlib)
**Estimated**: 1-2 days (~12h: 4h audit + 6h implementation + 2h testing)
**Dependencies**: M-CODEGEN-MULTIMODULE-BUGS (v0.9.2, complete)
**Reporter**: DocParse `go build` — `undefined: Trim, Split, Substring, Map`
**Created**: 2026-03-17

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Generated Go functions are deterministic — same semantics as interpreter |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | +1 | Pure builtins map to pure Go functions; effectful builtins require handler interfaces |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Go compiler verifies all function references at build time |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Enables machine compilation of AILANG → Go for all stdlib usage |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Go implementations have predictable, visible cost |
| A10: Composability | +1 | Stdlib functions compose in generated Go as they do in interpreter |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

---

## Problem Statement

When AILANG code imports stdlib functions (e.g., `import std/string (trim, split)`), the Go codegen emits references to Go functions that don't exist. The VarGlobal fallback at [codegen_expr_simple.go:156](internal/gen/golang/codegen_expr_simple.go#L156) converts `trim` → `Trim` via `ToPascalCase`, but no Go function `Trim` is defined in the generated code.

**Current state of the VarGlobal resolution chain** (codegen_expr_simple.go:60-157):

```
VarGlobal(name)
  → check topLevelFuncs map (user-defined functions)      ✅ works
  → check ADT constructors (LookupADTConstructor)         ✅ works
  → check effect builtins (mapEffectBuiltinToHandler)      ✅ works
  → check math builtins (mapPureMathBuiltin)               ✅ works (21 mappings)
  → check list builtins (mapPureListBuiltin)               ⚠️ partial (only concat, length)
  → check _impl for cross-module refs                      ✅ works
  → FALLBACK: ToPascalCase(name)                           ❌ generates undefined Go names
```

**Scope — DocParse audit:**

| Module | Total exports | Missing from codegen | Examples |
|--------|---------------|---------------------|----------|
| `std/string` | 22 | ~18 | Trim, Split, Substring, ToUpper, ToLower, Find, Contains, ... |
| `std/list` | 35 | ~30 | Map, Filter, Foldl, Reverse, Take, Drop, Any, SortBy, ... |
| `std/json` | 37 | ~35 | Decode, Encode, Get, Has, AsString, AsNumber, Keys, ... |
| `std/xml` | 14 | ~12 | Parse, FindAll, FindFirst, GetText, GetAttr, GetChildren, ... |
| `std/option` | 7 | ~5 | Map, FlatMap, GetOrElse, IsNone, IsSome |
| `std/result` | 7 | ~5 | Map, FlatMap, GetOrElse, IsOk, IsErr |
| `std/math` | 21 | 0 | All mapped via mapPureMathBuiltin ✅ |

**Total:** ~105 stdlib function exports need Go codegen support.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| 3-tier strategy (inline mapping / runtime helpers / effect handlers) | Determines the entire architecture for ~105 stdlib functions; wrong categorization means re-implementing functions later | human | design | high |
| Runtime helpers use `interface{}` (untyped) vs typed specialization | Affects performance of all generated Go code using stdlib; switching from `interface{}` to typed later requires rewriting all ~50 helpers and call sites | human | design | high |
| Self-contained runtime vs importing AILANG runtime package (`go.mod` dependency) | Self-contained means code duplication but zero dependency; import means single source of truth but couples generated code to AILANG version | human | design | high |
| JSON decode/encode as effect handler (Tier 3) vs pure runtime helper (Tier 2) | Determines whether JSON operations require user-supplied handler implementations or work out of the box | human | design | med |
| Argument order handling at `App` expression level vs wrapper functions | Wrapper is simpler but adds indirection; App-level fix is cleaner but touches hot codegen path | agent | compile | med |
| Conditional emission (only emit referenced helpers) vs emit-all | Emit-all is simpler but bloats output; conditional requires reference tracking | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Self-contained runtime (re-implement helpers) vs `go.mod` dependency on AILANG runtime — affects all generated code
- [ ] Whether JSON/XML operations are Tier 2 (pure runtime) or Tier 3 (effect handler) — determines if users must implement handler interfaces
- [ ] Whether `interface{}` is acceptable for all runtime helpers or some need typed specialization now
- [ ] How `applyFunc` will call closure values from generated code — must work with both lambdas and named function references
- [ ] Whether to emit all runtime helpers or only those referenced by the compiled code

---

## Solution Design

### The Established Pattern

The codegen already has **3 builtin mapping mechanisms** established across v0.5.7-v0.7.0:

1. **Runtime helpers** (`runtime.go`) — Go functions emitted with every generated package. Currently handles: `Cons`, `ListHead`, `ListTail`, `ListLen`, `Length`, `ConcatList`, `FieldGet`, `RecordUpdate`, `Concat`, converters.

2. **mapPureBuiltin tables** (`codegen_expr_simple.go`) — Name → Go expression mappings checked at VarGlobal resolution. Currently handles: 21 math builtins, 2 list builtins.

3. **Effect handler interfaces** (`handlers.go`) — Effect builtins map to interface method calls. Currently handles: FS, IO, AI, Env effects.

### Proposed Architecture: 3-Tier Builtin Strategy

**Tier 1: Inline Go stdlib mapping** (for builtins with direct Go equivalents)
- Pattern: `mapPureStringBuiltin`, `mapPureJsonBuiltin`, etc.
- Go equivalent exists → map directly
- Example: `trim` → `strings.TrimSpace`, `split` → `strings.Split`
- **This is the mapPureMathBuiltin pattern extended to all modules**

**Tier 2: Runtime helper functions** (for builtins needing Go implementations)
- Pattern: emit Go function in `runtime.go`
- No direct Go equivalent → write a Go function
- Example: `map` → `Map(f, xs)` runtime helper (iterate + apply function), `filter` → `Filter(p, xs)`
- These work with `interface{}` to maintain runtime polymorphism

**Tier 3: Effect handler interfaces** (for effectful builtins)
- Already implemented for FS, IO, AI, Env
- JSON decode/encode need handler methods (they involve parsing, not pure computation)
- XML parse needs handler method (involves Go's `encoding/xml`)

### Implementation Details

#### Tier 1: Direct Go Mappings (~30 functions)

New function: `mapPureStringBuiltin(name) string` — same pattern as `mapPureMathBuiltin`.

```go
// std/string → Go strings package
"trim" / "_str_trim":               → strings.TrimSpace   // needs strings import
"toUpper" / "_str_upper":           → strings.ToUpper
"toLower" / "_str_lower":           → strings.ToLower
"contains":                         → strings.Contains     // 2-arg
"startsWith" / "_str_startsWith":   → strings.HasPrefix
"endsWith" / "_str_endsWith":       → strings.HasSuffix
"split" / "_str_split":             → strings.Split        // 2-arg → []string
"find" / "_str_find":               → strings.Index        // returns -1 not Option
"join" / "_str_join":               → strings.Join         // note: AILANG arg order differs
"intToStr" / "_string_intToStr":    → strconv.FormatInt    // needs strconv import
"floatToStr":                       → strconv.FormatFloat
```

**Complication — argument order:** Some AILANG functions have different arg order from Go stdlib. `join(delimiter, xs)` vs `strings.Join(xs, delimiter)`. The codegen needs to handle this at the `App` expression level, not just the name level.

**Complication — return types:** AILANG `find` returns `int` (-1 for not found), but AILANG `stringToInt` returns `Option[int]`. The codegen needs to wrap Go results in the appropriate AILANG types.

#### Tier 2: Runtime Helper Functions (~50 functions)

These are functions where no direct Go equivalent exists, or the AILANG semantics differ:

```go
// std/list — higher-order functions need runtime helpers
func Map(f, xs interface{}) interface{} {
    list := toSlice(xs)
    result := make([]interface{}, len(list))
    for i, x := range list {
        result[i] = applyFunc(f, x)
    }
    return result
}

func Filter(p, xs interface{}) interface{} { ... }
func Foldl(f, acc, xs interface{}) interface{} { ... }
func Reverse(xs interface{}) interface{} { ... }
func Take(n, xs interface{}) interface{} { ... }
func Drop(n, xs interface{}) interface{} { ... }
func Any(p, xs interface{}) interface{} { ... }
func SortBy(cmp, xs interface{}) interface{} { ... }

// std/string — compound operations
func Substring(s interface{}, start, end interface{}) interface{} { ... }
func Repeat(s interface{}, n interface{}) interface{} { ... }
func Words(s interface{}) interface{} { ... }
func Chars(s interface{}) interface{} { ... }

// std/option — ADT operations
func OptionMap(f, opt interface{}) interface{} { ... }
func OptionGetOrElse(opt, default interface{}) interface{} { ... }

// std/json — pure accessors (decode/encode are effectful)
func JsonGet(obj, key interface{}) interface{} { ... }
func JsonHas(obj, key interface{}) interface{} { ... }
func JsonAsString(j interface{}) interface{} { ... }
```

**Key insight:** Many of these are already implemented as Go builtins in `internal/builtins/*.go`. The codegen needs to **re-emit equivalent implementations** in the generated runtime, since the generated code doesn't import the AILANG runtime.

**Alternative approach:** Instead of re-implementing, consider generating a `go.mod` that imports the AILANG runtime package and calls the existing Go builtin implementations. This would reduce code duplication but adds a dependency.

#### Tier 3: Effect Handler Extensions (~25 functions)

JSON and XML operations that involve I/O or complex parsing:

```go
// Already handled by effect handler interfaces:
//   FS: ReadFile, WriteFile, Exists, etc.
//   IO: println, readLine
//   AI: complete, chat
//   Env: getEnv, getArgs

// NEW handlers needed:
type JsonHandler interface {
    Decode(s string) (interface{}, error)
    Encode(obj interface{}) (string, error)
}

type XmlHandler interface {
    Parse(xml string) (interface{}, error)
    Serialize(node interface{}) string
}
```

### Implementation Plan

**Phase 1: Audit & Tier 1 — Direct mappings** (~4h)
- [ ] Audit all stdlib exports referenced by DocParse
- [ ] Add `mapPureStringBuiltin` with Go stdlib mappings
- [ ] Handle argument order differences (join, find)
- [ ] Track `needsStringsImport` flag (like `needsMathImport`)
- [ ] Test with DocParse compilation
- [ ] ~80 LOC

**Phase 2: Tier 2 — Runtime helpers for HOFs** (~4h)
- [ ] Add `Map`, `Filter`, `Foldl`, `Reverse`, `Take`, `Drop`, `Any`, `SortBy` to runtime
- [ ] Add `Substring`, `Repeat`, `Words`, `Chars` string helpers
- [ ] Add `applyFunc` helper for calling function values from generated code
- [ ] Test each helper with unit tests
- [ ] ~200 LOC runtime + ~100 LOC tests

**Phase 3: Tier 2 — JSON/Option/Result/XML helpers** (~3h)
- [ ] Add JSON accessor helpers (Get, Has, AsString, AsNumber, Keys, etc.)
- [ ] Add Option helpers (Map, FlatMap, GetOrElse, IsNone, IsSome)
- [ ] Add Result helpers (Map, FlatMap, GetOrElse, IsOk, IsErr)
- [ ] Add XML helpers (FindAll, FindFirst, GetText, GetAttr, GetChildren, GetTag)
- [ ] ~200 LOC runtime + ~80 LOC tests

**Phase 4: Integration** (~1h)
- [ ] Full DocParse compilation + `go build`
- [ ] Update CHANGELOG
- [ ] Update design doc

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_expr_simple.go` — Add `mapPureStringBuiltin`, extend `mapPureListBuiltin` (~80 LOC)
- `internal/gen/golang/codegen.go` — Add `needsStringsImport` tracking (~10 LOC)

**New files:**
- `internal/gen/golang/codegen_runtime_stdlib.go` — All stdlib runtime helpers (~300 LOC)
- `internal/gen/golang/codegen_runtime_stdlib_test.go` — Tests (~180 LOC)

**Total:** ~570 LOC implementation + ~180 LOC tests = ~750 LOC

---

## Examples

### Example 1: std/string mapping (Tier 1)

**AILANG:**
```ailang
import std/string (trim, toUpper, split)
export pure func process(text: string) -> [string] =
  split(trim(toUpper(text)), ",")
```

**Generated Go (before — broken):**
```go
func process__process_impl(text interface{}) interface{} {
    return Split(Trim(ToUpper(text)))  // ❌ undefined: Split, Trim, ToUpper
}
```

**Generated Go (after — working):**
```go
import "strings"

func process__process_impl(text interface{}) interface{} {
    return strings.Split(strings.TrimSpace(strings.ToUpper(text.(string))), ",")
}
```

### Example 2: std/list mapping (Tier 2)

**AILANG:**
```ailang
import std/list (map, filter)
export pure func evens(xs: [int]) -> [int] =
  filter(\x. x % 2 == 0, xs)
```

**Generated Go (after):**
```go
// In runtime.go:
func Filter(pred, xs interface{}) interface{} {
    list := toSlice(xs)
    var result []interface{}
    for _, x := range list {
        if applyFunc(pred, x).(bool) {
            result = append(result, x)
        }
    }
    return result
}
```

---

## Success Criteria

- [ ] DocParse 22-module compilation → `go build` succeeds with zero undefined symbols
- [ ] All stdlib string functions generate valid Go
- [ ] All stdlib list HOFs (map, filter, foldl, reverse, etc.) work in generated code
- [ ] JSON/XML accessor functions work in generated code
- [ ] All existing codegen tests pass
- [ ] New tests for each runtime helper function
- [ ] CHANGELOG updated

## Testing Strategy

**Unit tests:**
- Each runtime helper function tested in isolation
- Argument order verification for mismatched functions (join, etc.)
- Higher-order function helpers tested with closure values

**Integration tests:**
- DocParse 22-module `go build` succeeds
- Compare output of generated Go binary vs AILANG interpreter for same inputs

## Deferred Decisions

The following are intentionally left open for the implementer:

- Which stdlib functions go into Tier 1 (inline) vs Tier 2 (runtime helper) when the mapping is ambiguous (e.g., `substring` could be `s[start:end]` inline or a helper) — [agent may resolve]
- Exact `applyFunc` implementation strategy (reflect-based vs type-switch vs generated dispatch) — [agent may resolve]
- Whether to split runtime helpers into multiple files (`codegen_runtime_string.go`, `codegen_runtime_list.go`, etc.) or keep one file — [agent may resolve]
- Stdlib functions beyond DocParse's usage — will be added incrementally as projects need them — [agent may resolve per project]
- How to handle AILANG functions with different argument order from Go equivalents (per-function wrapper vs generic arg-reorder at App level) — [agent may resolve]

## Non-Goals

- **Typed specialization** — All runtime helpers use `interface{}`. Typed versions (e.g., `MapInt64` for `[int] -> [int]`) are a future optimization (tracked in M-CODEGEN-IR-STRATEGY).
- **Eliminating interpreter dependency** — Some complex builtins (JSON decode, XML parse) may still need effect handler implementations from the user. We don't auto-generate those.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Argument order mismatches | Medium | Document each mapping; add integration tests comparing interpreter vs Go output |
| `applyFunc` for HOFs needs closures | Medium | Follow existing `FnCaller` pattern in runtime; test with lambdas and named functions |
| Large runtime.go bloat | Low | New file `codegen_runtime_stdlib.go`; only emit helpers that are actually referenced |
| Semantic differences (e.g., AILANG `find` vs Go `strings.Index`) | Medium | Wrapper functions in runtime to normalize semantics |

## Related Documents

**Implemented (informs pattern):**
- [m-codegen-stdlib-math](../../implemented/v0_5_9/m-codegen-stdlib-math.md) — Math builtin mapping pattern (the template for this work)
- [m-codegen-list-type-definition](../../implemented/v0_7_0/m-codegen-list-type-definition.md) — List type alias for codegen
- [m-codegen-multimodule-bugs](../v0_9_2/m-codegen-multimodule-bugs.md) — Multi-module fixes (prerequisite)
- [m-codegen-unified-slice-converters](../../implemented/v0_6_0/m-codegen-unified-slice-converters.md) — Systemic slice converter pattern

**Planned:**
- [m-codegen-ir-strategy](../v0_10_0/m-codegen-ir-strategy.md) — Future IR refactoring (would restructure this)
- [m-codegen-api-server](../v0_10_0/m-codegen-api-server.md) — Compiled API server (depends on this)

## Future Work

- **Conditional emission** — Only emit runtime helpers for builtins actually referenced by the compiled code (tree-shaking)
- **Typed specialization** — Generate type-specific helpers (e.g., `MapString` for `(string -> string, [string]) -> [string]`) for better performance
- **Import AILANG runtime** — Alternative to re-implementing: generate `go.mod` that imports `github.com/sunholo/ailang/runtime` and delegates to existing Go builtin implementations

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17

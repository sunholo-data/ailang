# M-SERVE-API-ZERO-VALUE-PADDING: Zero-Value Padding for Missing Named Parameters

**Status**: Implemented
**Target**: v0.9.5
**Priority**: P1 (High — blocks AI-first error handling in serve-api)
**Estimated**: 1 day (~4-5 hours implementation + tests + docs)
**Dependencies**: M-SERVE-API-AGENT-ENHANCEMENTS (implemented in v0.9.2)
**Milestone ID**: M-SERVE-API-ZERO-VALUE-PADDING
**Created**: 2026-03-26
**Source**: DocParse agent message `08f8af96`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | HTTP layer is already outside the deterministic core |
| A2: Replayability | 0 | No change to replay semantics |
| A3: Effect Legibility | +1 | Missing params become predictable zero-values instead of opaque unit |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Functions can validate params with standard comparisons (`== ""`) |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents get typed zero-values they can reason about, not unit traps |
| A8: Minimal Syntax | 0 | No new language syntax — changes are in the serve-api runtime only |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Zero-value padding composes with named binding, @route, @nowrap |
| A11: Structured Failure | +1 | Functions can return typed error responses instead of crashing on unit |
| A12: System Boundary | +1 | HTTP→AILANG boundary gains type-safe translation for missing params |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — zero-values are deterministic per type
- [x] A3 (Effects): No hidden side effects — HTTP serving is already an effect
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Zero-value padding is specifically FOR machine callers (agents)

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

This is a direct follow-on to M-SERVE-API-AGENT-ENHANCEMENTS. That sprint added named parameter binding but left a gap: when a named param is omitted from the JSON body, it becomes `nil` in Go, which converts to `eval.UnitValue{}` via `internal/embed/convert.go:34-36`. Unit values crash when used with type-specific operations (e.g., `length(apiKey)` where `apiKey` is unit instead of `""`).

**What already exists (do NOT re-implement):**
- **Named parameter binding** — `parseNamedArgs()` in `handler.go:128-155` maps JSON keys to function params by name with exact + snake_case matching.
- **Positional `{"args": [...]}` format** — Still supported as highest-priority fallback.
- **Zero-arg unit workaround** — `handler.go:256-262` retries with unit arg for zero-arg functions.
- **Type string in ExportInfo** — `ExportInfo.Type` has human-readable signature (e.g., `"string -> string -> int"`).

**What's genuinely new:**
1. **Per-parameter type extraction from AST** — `ast.Param` has a `Type` field but `extractParamNames()` only extracts `Name`. Need to also extract types.
2. **Zero-value substitution in `parseNamedArgs()`** — Replace `nil` slots with type-appropriate zero-values (`""` for string, `0` for int, `false` for bool, `[]` for list).
3. **Positional arg padding** — When fewer positional args than parameters, pad remaining with zero-values instead of returning a framework error.

---

## Problem Statement

### Current Behavior

When a named parameter is missing from a JSON request body, the serve-api pipeline is:

```
JSON body missing "apiKey" → parseNamedArgs returns nil for that slot
→ embed.fromGoInternal(nil) → eval.UnitValue{}
→ AILANG function receives unit for apiKey parameter
→ length(apiKey) crashes: "cannot take length of unit"
```

The function author has **no way** to check for missing params and return a typed error because:
1. There is no `== unit` comparison in AILANG (unit is not a value you test against)
2. The crash happens inside builtins, not at a point where the function can intercept it

Similarly, positional args with fewer than expected params:
```
POST {"args": ["file.docx"]}  (function expects 3 params)
→ evaluator returns: "function expects 3 arguments, got 1"
→ 500 error with framework message — never reaches user code
```

### Desired Behavior

```
JSON body missing "apiKey" → parseNamedArgs returns "" for string param
→ embed.fromGoInternal("") → eval.StringValue{Value: ""}
→ AILANG function receives "" for apiKey parameter
→ if apiKey == "" then Error("apiKey is required") else ...
→ 400 response with typed error: {"error": "apiKey is required"}
```

**Impact:** Every serve-api function that accepts optional parameters is currently broken for AI agents. Agents cannot send partial JSON and get meaningful error responses.

---

## Goals

**Primary goal:** Missing named parameters receive type-appropriate zero-values so AILANG functions can validate inputs and return structured errors.

**Success metrics:**
- `POST /api/v1/parse {"filepath": "x"}` with missing `apiKey: string` param → function receives `""`, not unit
- `POST /api/v1/compute {"args": [1]}` with 3-param function → pads with `0, 0` for int params
- No breaking changes to existing named or positional binding
- OpenAPI spec marks padded params as optional with documented defaults

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| Use AST param types (not string parsing) | Avoids fragile regex on `ExportInfo.Type` string | Design doc | Pre-implementation | Medium — would need rewrite |
| Pad positional args too (not just named) | Consistency — both paths should behave the same | Design doc | Pre-implementation | Low |
| Zero-value (not null/Option) for missing | Keeps AILANG simple — no Option type needed yet | Design doc | Pre-implementation | High — would need ADT |

### Design Freeze Checklist

- [x] Zero-value per type: `string → ""`, `int → 0`, `float → 0.0`, `bool → false`, `list → []`, `record → {}`
- [x] Unknown/complex types: pad with `nil` (becomes unit) — no change from current behavior
- [x] `ast.Param.Type` is available at extraction time (confirmed: `ast_expr.go:149-153`)

---

## Solution Design

### Overview

```
                   ┌─────────────────────┐
  JSON body ──────►│ parseNamedArgs()    │
                   │ + paramTypes []string│──► args with zero-values
                   └─────────────────────┘
                              ▲
                              │
               ┌──────────────┴──────────────┐
               │ extractParamNames() →        │
               │ extractParamInfo()           │
               │ (names + types from AST)     │
               └──────────────────────────────┘
```

### Phase 1: Extract Per-Parameter Types from AST

**File: `internal/apiserver/routes.go`**

Rename `extractParamNames()` → `extractParamInfo()` and also extract type strings:

```go
// ExportInfo gains a new field:
ParamTypes []string `json:"param_types,omitempty"` // e.g. ["string", "string", "int"]

func extractParamInfo(modInfo *ModuleInfo, file *ast.File) {
    for _, fn := range file.Funcs {
        if !fn.IsExport {
            continue
        }
        names := make([]string, len(fn.Params))
        types := make([]string, len(fn.Params))
        for i, p := range fn.Params {
            names[i] = p.Name
            types[i] = typeToString(p.Type) // "string", "int", "bool", "list", etc.
        }
        for i := range modInfo.Exports {
            if modInfo.Exports[i].Name == fn.Name {
                modInfo.Exports[i].ParamNames = names
                modInfo.Exports[i].ParamTypes = types
                break
            }
        }
    }
}

func typeToString(t ast.Type) string {
    switch v := t.(type) {
    case *ast.SimpleType:
        return v.Name // "string", "int", "bool", "float"
    case *ast.ListType:
        return "list"
    case *ast.ArrayType:
        return "array"
    case *ast.RecordType:
        return "record"
    default:
        return "unknown"
    }
}
```

### Phase 2: Zero-Value Lookup

**File: `internal/apiserver/handler.go`**

```go
func zeroValueForType(typeName string) interface{} {
    switch typeName {
    case "string":
        return ""
    case "int":
        return 0
    case "float":
        return 0.0
    case "bool":
        return false
    case "list", "array":
        return []interface{}{}
    case "record":
        return map[string]interface{}{}
    default:
        return nil // unknown types remain nil → unit (current behavior)
    }
}
```

### Phase 3: Pad Missing Named Params

**File: `internal/apiserver/handler.go`**

Modify `parseNamedArgs()` to accept and use type info:

```go
func parseNamedArgs(body map[string]interface{}, paramNames []string, paramTypes []string) []interface{} {
    if len(paramNames) == 0 {
        return nil
    }
    args := make([]interface{}, len(paramNames))
    matched := 0
    for i, name := range paramNames {
        if val, ok := body[name]; ok {
            args[i] = val
            matched++
            continue
        }
        snake := camelToSnake(name)
        if snake != name {
            if val, ok := body[snake]; ok {
                args[i] = val
                matched++
                continue
            }
        }
        // Zero-value padding for unmatched params
        if i < len(paramTypes) {
            args[i] = zeroValueForType(paramTypes[i])
        }
    }
    if matched == 0 {
        return nil
    }
    return args
}
```

### Phase 4: Pad Positional Args

**File: `internal/apiserver/handler.go`**

In `parseArgsWithNames()`, when `{"args": [...]}` has fewer elements than parameters:

```go
// After extracting positional args from {"args": [...]}:
if len(posArgs) < len(paramNames) && len(paramTypes) > 0 {
    padded := make([]interface{}, len(paramNames))
    copy(padded, posArgs)
    for i := len(posArgs); i < len(paramNames); i++ {
        if i < len(paramTypes) {
            padded[i] = zeroValueForType(paramTypes[i])
        }
    }
    posArgs = padded
}
```

### Phase 5: OpenAPI Schema Updates

**File: `internal/apiserver/openapi.go`**

- Mark params with known zero-values as `"required": false` in the schema
- Add `"default": <zero-value>` to each property with a known type
- Add `x-ailang-param-types` alongside existing `x-ailang-param-names`

---

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/apiserver/server.go` | Add `ParamTypes []string` to `ExportInfo` | ~2 |
| `internal/apiserver/routes.go` | Rename `extractParamNames` → `extractParamInfo`, add `typeToString()` | ~30 |
| `internal/apiserver/handler.go` | Add `zeroValueForType()`, update `parseNamedArgs()` + `parseArgsWithNames()` signatures | ~40 |
| `internal/apiserver/openapi.go` | Add `x-ailang-param-types`, mark optional params, add defaults | ~20 |
| `internal/apiserver/named_args_test.go` | Update tests: partial match expects `""` not `nil`, add positional padding tests | ~40 |
| `changelogs/v0.9-current.md` | Document the DX fix | ~5 |
| `docs/docs/guides/serve-api.md` | Document zero-value padding behavior | ~15 |

**Total: ~150 LOC**

---

## Testing Strategy

### Unit Tests

1. **Named binding — partial match pads zero-values:**
   ```
   params: [path: string, outputFormat: string, maxSize: int]
   body: {"path": "file.docx"}
   expected: ["file.docx", "", 0]
   ```

2. **Named binding — all params present (no padding):**
   ```
   params: [path: string, outputFormat: string]
   body: {"path": "f.docx", "outputFormat": "blocks"}
   expected: ["f.docx", "blocks"]
   ```

3. **Named binding — unknown type gets nil (unit):**
   ```
   params: [callback: MyCustomType]
   body: {}  (one key matched elsewhere to trigger binding)
   expected: [nil]  // unknown type, no zero-value
   ```

4. **Positional padding:**
   ```
   params: [a: string, b: int, c: bool]
   body: {"args": ["hello"]}
   expected: ["hello", 0, false]
   ```

5. **Zero-arg function unchanged:**
   ```
   params: []
   body: {}
   expected: nil (unchanged behavior)
   ```

### Integration Tests

6. **End-to-end: AILANG function validates missing param:**
   ```ailang
   module api

   @route("POST", "/parse")
   export func parse(filepath: string, apiKey: string) -> string =
     if apiKey == "" then "error: apiKey is required"
     else "parsed: " ++ filepath
   ```
   ```
   POST /api/v1/parse {"filepath": "test.pdf"}
   → 200 {"result": "error: apiKey is required"}
   ```

### Regression Tests

7. **Existing named binding tests still pass** (no breaking changes)
8. **Existing positional `{"args": [...]}` tests still pass**
9. **`@nowrap` + named binding + partial params work together**

---

## Success Criteria

- [ ] `parseNamedArgs()` returns `""` for missing string params (not nil)
- [ ] `parseNamedArgs()` returns `0` for missing int params
- [ ] `parseNamedArgs()` returns `false` for missing bool params
- [ ] Positional `{"args": [...]}` with fewer args pads with zero-values
- [ ] Unknown/complex types still become unit (backward compatible)
- [ ] OpenAPI spec shows optional params with defaults
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Changelog updated
- [ ] serve-api docs updated

---

## Non-Goals

- **Option/Maybe type** — Full optional type support is a separate language feature; zero-values are the pragmatic HTTP-boundary solution
- **Validation annotations** — `@required`, `@optional`, `@default(value)` annotations are future work
- **Type coercion** — If JSON sends `"123"` for an int param, this doc does NOT address coercion; that's a separate concern
- **Deeply nested type resolution** — `Result<string, Error>` or `List<int>` param types get `"unknown"` — only simple types are padded

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Zero-values mask genuinely required params | Medium — function may silently accept empty string | Document that functions MUST validate; OpenAPI marks as optional |
| `ast.Param.Type` is nil for inferred params | Low — type annotation may be omitted | Fall back to `"unknown"` → nil → unit (current behavior) |
| Breaking change to `parseNamedArgs` signature | Low — internal function | All callers are in `handler.go`, update in same PR |

---

## Future Work

- **`@required` / `@optional` annotations** — Let function authors declare which params are truly required vs optional at the HTTP boundary
- **`@default(value)` annotation** — Custom default values instead of type zero-values
- **Type coercion** — Automatic `"123"` → `123` for int params from JSON strings
- **Option type** — `Option<string>` params could receive `None` instead of `""` for a more principled approach

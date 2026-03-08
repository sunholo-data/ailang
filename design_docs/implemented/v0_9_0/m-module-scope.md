# M-MODULE-SCOPE: Non-exported function name collision across modules

**Status:** IMPLEMENTED (2026-03-08)
**Priority:** P0 — violates module encapsulation, breaks docparse demo
**Source:** docparse-benchmark agent (msg cb18cb5a, msg 6d76e85a)
**Affects:** Any two modules with same-named internal functions loaded via ModuleRuntime (CLI pipeline)
**Regression test:** `internal/runtime/scope_collision_test.go` (3 tests — all PASS after fix)

## Problem

Non-exported (internal) functions in different modules collide when both modules
are loaded. The last-loaded module's version shadows all previous definitions.
This violates module encapsulation and is silent — no error, just wrong behavior.

### Minimal Reproduction

```ailang
-- module_a.ail
module a
export pure func greetA(name: string) -> string = format(name)
pure func format(name: string) -> string = "Hello from A: " ++ name

-- module_b.ail
module b
export pure func greetB(name: string) -> string = format(name)
pure func format(name: string) -> string = "Hello from B: " ++ name

-- main.ail: imports both
-- Expected: greetA("World") → "Hello from A: World"
-- Actual:   greetA("World") → "Hello from B: World"  ← BUG
```

Confirmed via:
- Native CLI: `ailang run --entry main --caps IO` with `/tmp/scope_bug/bug_repro/`
- Go test: `TestModuleScopeCollision_Runtime` in `internal/runtime/scope_collision_test.go`

### Impact on DocParse (the original bug report)

Both `docx_parser.ail` and `pptx_parser.ail` define internal `joinParagraphTexts`:

| Module | `joinParagraphTexts` calls | Handles |
|--------|---------------------------|---------|
| `docx_parser` | `extractParagraphText` | `w:r`/`w:t` (WordprocessingML) |
| `pptx_parser` | `extractDrawingMLText` | `a:r`/`a:t` (DrawingML) |

When `main.ail` imports both, pptx's version shadows docx's. DOCX table cell
extraction then looks for DrawingML tags in WordprocessingML XML → returns empty
strings for all cells. This is the #1 gap vs Unstructured (0% text overlap on
table-heavy files).

Additionally: `docx_parser.parseTableRows` and `direct_ai_parser.parseTableRows`
collide (different parameter types: `[XmlNode]` vs `[Json]`).

## Root Cause

The ModuleRuntime (`internal/runtime/runtime.go`) uses a **single shared evaluator**
with a **flat namespace** for all module bindings, including non-exported functions.

### The shared evaluator mechanism

```go
// runtime.go:227 — evaluateModule uses shared rt.evaluator for ALL modules
func (rt *ModuleRuntime) evaluateModule(inst *ModuleInstance) error {
    resolver := newModuleGlobalResolver(inst, rt)
    rt.evaluator.SetGlobalResolver(resolver)  // same evaluator!
    ...
}

// runtime.go:309,335 — extractBindings adds ALL names to shared env
rt.evaluator.Env().Set(name, val)  // BOTH exported AND internal functions
```

### How names are resolved

| Function type | Elaboration | Storage | Lookup |
|--------------|-------------|---------|--------|
| **Exported** | `VarGlobal{Module: "mod", Name: "fn"}` | `ModuleInstance.Exports["fn"]` | Module-qualified via `moduleGlobalResolver` |
| **Internal** | `Var{Name: "fn"}` | `rt.evaluator.env["fn"]` | **Bare name in shared flat map** |

### The collision sequence

```
1. Load scope_a (via dependency resolution):
   - Elaborate: format → Var{Name: "format"}
   - extractBindings: rt.evaluator.Env().Set("format", closure_a)  ← line 335
   - Export: transformA captured with env reference

2. Load scope_b:
   - Elaborate: format → Var{Name: "format"}
   - extractBindings: rt.evaluator.Env().Set("format", closure_b)  ← OVERWRITES
   - Export: transformB captured with env reference

3. Runtime: scope_a.transformA(1) calls format(1)
   - evalCoreVar(Var{"format"})
   - rt.evaluator.env.Get("format") → closure_b  ← WRONG!
   - Returns 1 + 200 = 201 instead of 1 + 100 = 101
```

### Key files

| File | Line | Role |
|------|------|------|
| `internal/runtime/runtime.go` | 227 | `evaluateModule` — shared `rt.evaluator` for all modules |
| `internal/runtime/runtime.go` | 309, 335 | `extractBindings` — adds ALL bindings to shared env |
| `internal/elaborate/expressions.go` | 52-63 | Elaborates non-exported as `Var` (bare name) |
| `internal/eval/env.go` | 26-38 | Flat `map[string]Value` environment |
| `internal/eval/eval_expressions.go` | 85-99 | `evalCoreVar` does bare-name lookup |
| `internal/eval/eval_expressions.go` | 164 | Closures capture `Env: e.env` by REFERENCE |

### Why ModuleRegistry (WASM) is NOT affected

The WASM/REPL code path uses `ModuleRegistry` (`internal/repl/module_registry.go:534`)
which creates a **fresh evaluator per LoadModule call**. This provides natural isolation.
The bug is specific to the native CLI pipeline's `ModuleRuntime`.

## Investigation Summary

| Test | Result | Why |
|------|--------|-----|
| Go builtins (`TestXmlParse_OOXMLTable`) | PASS | Tests Go code directly, no modules |
| Native CLI single module (`/tmp/table_test.ail`) | PASS | No conflicting imports |
| ModuleRegistry (`TestWasmTableXmlExtraction`) | PASS | Fresh evaluator per module |
| ModuleRegistry collision (`TestModuleScopeCollision`) | PASS | Fresh evaluator per module |
| **ModuleRuntime collision (`TestModuleScopeCollision_Runtime`)** | **FAIL** | **Shared evaluator — the bug** |
| **Native CLI multi-module** | **FAIL** | **Same root cause** |

## Proposed Fix

### Option A: Per-module environment isolation (Recommended)

Save and restore the evaluator environment around each module's evaluation.
Each module gets a fresh scope for its internal bindings; closures capture
that scope. Cross-module references go through the resolver (VarGlobal path).

```go
// runtime.go evaluateModule — isolate per-module internals
func (rt *ModuleRuntime) evaluateModule(inst *ModuleInstance) error {
    // Save parent env, create fresh scope for this module's internals
    savedEnv := rt.evaluator.Env().Clone()
    defer rt.evaluator.SetEnv(savedEnv)  // Restore after module eval

    // Module evaluates in fresh scope — internal names don't leak
    ...
}
```

**Pros:** Clean isolation, minimal API change, matches ModuleRegistry behavior
**Cons:** Requires `SetEnv` method on evaluator

### Option B: Module-qualify internal names during elaboration

Convert all non-exported declarations to `VarGlobal` with module prefix:

```go
// elaborate/expressions.go — treat ALL module-level decls as VarGlobal
if e.currentModule != "" {
    return &core.VarGlobal{
        Module: e.currentModule,
        Name:   ex.Name,
    }
}
```

**Pros:** Explicit, debuggable, matches how exports already work
**Cons:** Larger elaboration change, needs matching resolver changes

### Option C: Namespace-qualify environment keys

Keep flat env but key by `"module/name.funcName"` instead of bare `"funcName"`:

```go
rt.evaluator.Env().Set(inst.Path + "." + name, val)
```

**Pros:** Simplest change
**Cons:** Requires threading module name through evaluator

### Recommended: Option A

Per-module isolation is the cleanest fix. It matches how ModuleRegistry already
works (fresh evaluator per module = natural isolation) and doesn't require
changes to elaboration or the Core AST.

## Workaround (Immediate)

Rename internal functions to be globally unique:

```ailang
-- docx_parser.ail
pure func docxJoinParagraphTexts(ps: [XmlNode]) -> string = ...

-- pptx_parser.ail
pure func pptxJoinParagraphTexts(ps: [XmlNode]) -> string = ...
```

This unblocks the docparse demo while the proper fix is implemented.

## Estimate

- **Option A (per-module isolation):** 4-6 hours
  - Modify `evaluateModule` to save/restore env per module (2h)
  - Ensure cross-module imports still resolve via resolver (1h)
  - Verify regression test passes (30min)
  - Test docparse demo end-to-end (1h)
- **Workaround (rename):** 30 minutes
  - Rename colliding functions in docparse modules
  - Verify tables extract correctly

## Regression Tests

### Existing (proves the bug)

- `internal/runtime/scope_collision_test.go:TestModuleScopeCollision_Runtime`
  - scope_a: `format(x) = x + 100`, scope_b: `format(x) = x + 200`
  - Asserts `testA(1) == 101` — currently FAILS (returns 201)

### Additional tests needed after fix

1. **Closure capture** — verify closures reference their own module's internals after other modules load
2. **DocParse pattern** — load docx_parser + pptx_parser, verify `joinParagraphTexts` dispatches correctly
3. **Three-way collision** — docx_parser + pptx_parser + direct_ai_parser all with `parseTableRows`

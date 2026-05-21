# M2 Diagnosis: M-WASM-TYPECHECK-FLOAT-DIVERGENCE

## Conclusion

The bug is **not** the typeEnv injection (Suspect A) or the constructor double-registration (Suspect B) hypothesised pre-sprint. The actual root cause is:

> The WASM module-loading path (`internal/repl/module_registry_load.go`) does **not propagate type aliases from imported modules**. The CLI pipeline (`internal/pipeline/pipeline_module_compile.go:201-205`) does. As a result, declared parameter types like `st: CommonsState` and `a: Point` are not resolved when type-checking imported-module references — the type-checker falls back to inferring open record types from usage, producing schemes with un-quantified free type variables (e.g. `α88`). These free variables leak across modules through the type environment chain. Adding a new constraint (e.g. `gap_force(g: float) -> string` introducing `g > 0.5` which adds a `Fractional` constraint) is enough to push the constraint solver into unifying these leaked vars with the wrong concrete type — `string` in this test, `int` in the browser.

## Evidence

### Test result

`internal/repl/wasm_float_divergence_bisect_test.go` shows the bug only triggers when:

- `gap_force` has BOTH a `float` parameter AND a float-literal in body (`if g > 0.5 then "high" else "low"`)
- NOT triggered by: float-return-only, int-param, string-only, missing helper

### Scheme dump (the smoking gun)

`internal/repl/wasm_float_divergence_diag_test.go` dumps the inferred schemes of each module's exports.

**`consensus.ail` (in both no-helper AND with-helper cases) — note no `∀α88`:**
```
currentLeader :: (Ord[α88], Ord[α88], ...) => { sentiment: { x: α88, | r }, | r } -> Persona
distance      :: ∀α55. (Num[α55], ...) => ({ x: α55, | r }, { x: α55, | r }) -> α55
```

But `currentLeader` is declared:
```ailang
export pure func currentLeader(st: CommonsState) -> Persona
```

where `CommonsState = { sentiment: Point, ... }` and `Point = { x: float, y: float }`. The scheme **should** be:
```
currentLeader :: { sentiment: { x: float, y: float }, ... } -> Persona
```

with `α88` resolved to `float`. Instead, `α88` is free and the scheme has redundant Ord-constraints (10 copies of `Ord[α88]`).

### Why the bug only surfaces with `gap_force`

When `gap_force(g: float) -> string` with body `if g > 0.5 then "high" else "low"` is added:

- citizen.ail's `compose_user_prompt` scheme flips from `{ sentiment: { x: α128, | r }, | r }` (no_helper) to `{ sentiment: { x: string, | r }, | r }` (with_helper) — **α128 concretized to `string`**
- Same for `compose`, `speak`
- This poisons commons_browser.ail's later type-checking of `let sx = match score_result { Ok(s) => s.x, Err(_) => 0.0 }`, where `s.x` ends up `string` instead of `float`
- `jnum(sx)` at `commons_browser.ail:201:43` fails with `cannot unify type constructors: float vs string`

### Why this only happens on WASM

`internal/pipeline/pipeline_module_compile.go:201-205` (CLI) registers imported type aliases:

```go
for name, target := range imports.ImportedTypeAliases {
    typeChecker.RegisterTypeAlias(name, target)
}
```

`internal/repl/module_registry_load.go:273-276` (WASM) only registers the **current module's** type aliases:

```go
elabAliases := elaborator.GetTypeAliases()
for name, target := range elabAliases {
    typeChecker.RegisterTypeAlias(name, target)
}
```

There is **no equivalent loop for `imports.ImportedTypeAliases`**. The `RegisteredModule` struct does not even store type aliases — only `Exports`.

Similarly, parameter/return type annotations (`SetParamTypeAnnotations`, `SetReturnTypeAnnotations`) are wired in the CLI (`pipeline_module_compile.go:207-220`) but never called in the WASM path.

## Fix

Two changes in `internal/repl/module_registry.go` + `internal/repl/module_registry_load.go`:

1. **Store type aliases on `RegisteredModule`**: extend the struct with `TypeAliases map[string]types.Type`; populate it from `elaborator.GetTypeAliases()` when registering the module.

2. **Propagate imported aliases on LoadModule**: before type-checking the current module, iterate over `mr.modules`, collect each module's `TypeAliases`, and register them on BOTH the elaborator (so signatures like `st: CommonsState` resolve in the AST) and the type checker (for unification expansion).

This mirrors the CLI's behaviour in `pipeline_module_compile.go:194-205`. Parameter/return annotations may need similar treatment, but cross-module alias propagation alone should be enough to fix the divergence — let's verify after M3 ships.

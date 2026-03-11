---
paths:
  - "internal/types/**"
  - "internal/elaborate/**"
  - "internal/iface/**"
  - "internal/pipeline/**"
  - "internal/core/**"
---

# Type System Rules

## CoreTypeInfo Invariant

Every Core node must have an entry in CoreTypeInfo before lowering. Validation enforces this in all paths (file pipeline, module pipeline, REPL).

```go
if err := ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI); err != nil {
    return result, fmt.Errorf("CoreTypeInfo validation failed: %w", err)
}
```

Validated nodes may carry polymorphic types (accepted as long as a type exists). Validator: `internal/pipeline/validate_coretypeinfo.go`.

## Safe Type Traversal - Cycle Protection

Every function of shape `func(types.Type) T` MUST document cycle-safety. Either use `traverse.Walk` or add a `visited` parameter.

Package: `internal/types/traverse/` — provides `Walk()`, `CollectFreeVars()`, `HasCycles()`.

## Monomorphization

Enabled by default (v0.4.0). Polymorphic lambdas specialized at call sites.

```bash
ailang run --entry main --caps IO module.ail  # Normal (mono enabled)
ailang run --debug-compile module.ail          # Show specialization stats
ailang run --no-mono module.ail                # Disable (escape hatch)
```

Resource limits: 16 per-function, 512 per-module. See `design_docs/implemented/v0_4_0/monomorphization.md`.

## ast.Type Switch Exhaustiveness

**ALL 8 variants must be handled.** Silent `default` cases corrupt imported polymorphic types.

| Variant | Example |
|---------|---------|
| `*ast.SimpleType` | `int`, `string`, `Result` |
| `*ast.TypeVar` | `a`, `e` (type parameters) |
| `*ast.FuncType` | `(int) -> string` |
| `*ast.ListType` | `[int]` |
| `*ast.ArrayType` | `Array[int]` |
| `*ast.TupleType` | `(int, string)` |
| `*ast.TypeApp` | `Result[a, e]` |
| `*ast.RecordType` | `{name: string}` |

**Rules:** `default:` in ast.Type switches MUST `panic()`, never return fake data. When adding a new variant, grep for ALL type switches: `grep -rn "case \*ast\." internal/`

# AILANG Error Codes Reference

All AILANG diagnostic errors follow the format `<PREFIX><NUMBER>` (e.g. `MOD013`, `PAR001`).

## Error code prefixes

| Prefix | Phase | Description |
|--------|-------|-------------|
| `PAR` | Parser | Syntax errors during parsing |
| `MOD` | Module | Module system violations |
| `LDR` | Loader | Module loading and resolution errors |
| `IMP` | Import | Import statement errors |
| `DSG` | Desugar | Desugaring transformation errors |
| `TC` | Type checker | Type checking failures |
| `ELB` | Elaboration | Core AST elaboration errors |
| `LNK` | Linker | Linking and dictionary resolution errors |
| `EVA` | Evaluator | Runtime evaluation errors |
| `RT` | Runtime | Low-level runtime errors |

## Machine-readable registry

Every release publishes `error_codes.json` as a release asset. This file follows schema-v1 and contains one record per error code with `code`, `category`, `summary`, and `fix_hint` fields.

### Downloading

```bash
# Latest release
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/error_codes.json \
  -o error_codes.json

# Specific version
curl -L https://github.com/sunholo-data/ailang/releases/download/v0.17.0/error_codes.json \
  -o error_codes.json
```

### Schema (v1)

```json
{
  "schema_version": "v1",
  "records": [
    {
      "code": "MOD013",
      "category": "package",
      "summary": "Shared module_prefix between root and dependency",
      "fix_hint": "Remove the dependency, change one side's module_prefix, or use explicit pkg/ imports"
    }
  ]
}
```

### Consuming in CI

```bash
# Check if a specific error code exists
jq '.records[] | select(.code == "MOD013")' error_codes.json

# List all module errors
jq '.records[] | select(.code | startswith("MOD"))' error_codes.json

# Extract all fix hints
jq '.records[] | {code, fix_hint}' error_codes.json
```

### Consuming from Go

```go
import "encoding/json"

type ErrorRecord struct {
    Code     string `json:"code"`
    Category string `json:"category"`
    Summary  string `json:"summary"`
    FixHint  string `json:"fix_hint"`
}

type ErrorCodesOutput struct {
    SchemaVersion string        `json:"schema_version"`
    Records       []ErrorRecord `json:"records"`
}
```

## Type-checker error-quality codes

Some type-checker failures carry an inline `TC_*` code plus a `Suggestion:` line so a model (or human) can look up the failure class and get an actionable fix. Examples: `TC_REC_001`–`TC_REC_004` (record errors) and `TC_ARITY_001`.

**`TC_ARITY_001` — function arity mismatch.** Emitted when a function is called with the wrong number of arguments. AILANG has **strict arity and no partial application**, so the diagnostic is directional and style-aware: it states how many arguments the function expects vs how many were provided, and the `Suggestion:` line names the fix. Under-supply (e.g. `add(1)` on an arity-2 `add`, or a partial application `let inc = add(1)`) reminds you that AILANG has no partial application — call with all N arguments, or wrap in a lambda. Over-supply (e.g. `add(1, 2, 3)`) tells you to remove the extra argument(s). This is a diagnostic-only improvement; arity semantics are unchanged.

```
TC_ARITY_001: function expects 2 argument(s), but 1 provided
  Suggestion: AILANG has no partial application — call with all 2 arguments, or wrap in a lambda `\a b. f(a, b)`.
```

## Individual error code pages

- [MOD013 — Shared module_prefix](mod013.md)
- [Effect row mismatch](typ_effect_row_mismatch.md)

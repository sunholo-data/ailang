---
reviewBy: 2027-03-08
---

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
| `BC` | Bytecode | Strict-VM coverage (`--strict-bytecode`) |

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

**`TC_MATCH_001` — unknown constructor in a pattern.** Emitted when a match arm names a constructor that no local declaration, import, or loaded module defines: a typo, or an alias that was never written in an import (`import std/option (None as Nada)`). The runtime matches constructors by name, so such an arm used to compile and never match. The `Suggestion:` line lists the scrutinee's constructors when its type is known.

```
TC_MATCH_001: constructor pattern 'Bogus' does not name any constructor in scope (no local declaration, import, or loaded module defines it), so this arm could never match
  Suggestion: Option's constructors are: None, Some
```

**`TC_MATCH_005` — `Json` match with a `JNumber` arm and no `JInt` arm (warning).** Transitional for v0.58.x. `std/json.decode` gives `JInt(n)` for integer literals and `JNumber(x)` for other numbers, so a match written before `JInt` existed, `match j { JNumber(n) => …, _ => … }`, compiles but sends every integer to the catch-all. The warning is keyed on std/json's own `JNumber` (aliased imports included; a module's own `JNumber` constructor is never flagged), looks at every pattern depth (`Some(JNumber(n))` counts), and is reported on a warm compile cache too. It never blocks compilation. Fix: add a `JInt(i) => …` arm (`intToFloat(i)` gives a float), or read numbers with `asNumber`/`getNumber`/`getInt`, which accept both.

```
TC_MATCH_005 at app.ail:6:45
  this match over std/json.Json has a JNumber arm but no JInt arm; decode now gives JInt(n) for integer literals such as 42, so integers skip the JNumber arm.
  Fix: add a `JInt(i) => …` arm (intToFloat(i) gives a float), or read numbers with asNumber/getNumber/getInt, which accept both
```

**`TC_DEPRECATED_001` — use of a `@deprecated` function (warning).** Emitted once per use site — a call, a function passed as a value, a module-qualified `L.f` or an aliased import — of a function declared `@deprecated("replacement")` in another module. The message names the replacement and the suggestion carries the annotation's optional note (for example, that the arguments swap). A name imported by name and never used warns once, at that name in the import list (the import itself fails once the alias is removed). Defining it and uses inside the defining module do not warn; nor do uses inside an installed dependency package (`pkg/<other>/...`), which that package's own `check` reports. If the replacement is itself deprecated, the message names the first function along the chain that is not (`use headOr`, not `use head_or`); when the chain cycles or reaches a name that does not exist, it names the annotation's own replacement. It is reported on a warm compile cache too, and never blocks compilation. The stdlib renames before 1.0 (`head_or` → `headOr`, `rand_int` → `randInt`, …) use it; see [Deprecated stdlib names](/docs/reference/stability#deprecated-stdlib-names).

```
TC_DEPRECATED_001 at app.ail:4:31
  std/list.head_or is deprecated; use headOr.
  Fix: replace head_or with headOr (import it from std/list)
```

**`TC_DEPRECATED_002` — `@deprecated` names no replacement (error).** The replacement in `@deprecated("x")` must exist: an export of the same module (`"headOr"`), or `"mod/path.name"` from a module this one imports. It is an error, at the annotation, when that name is not exported (with a did-you-mean, spelled the way the annotation must spell it) or is the function itself. Nothing else: a replacement that is itself deprecated, or a chain that cycles, is legal — whether a name in another module is deprecated is not part of this module's compile-cache key, so rejecting it would give one verdict cold and another warm. A std function deprecated in a later release therefore gives an unchanged library warnings only, never an error, and those warnings are not reported inside an installed dependency. Fix: point the annotation at an existing export.

```
Error TC_DEPRECATED_002 at lib.ail:6:1: @deprecated("headOR") on head_or names no replacement: lib does not export headOR
  Fix: did you mean @deprecated("headOr")?
```

**`PAR_DEPRECATED_ARG` / `PAR_EXPERIMENTAL_ARG` — malformed annotation (parse error).** `@deprecated` takes the replacement's name and an optional note, as one or two non-empty string literals: `@deprecated("headOr")`, `@deprecated("contains", "arguments swap")`. `@experimental` takes no arguments; `@experimental("x")` is `PAR_EXPERIMENTAL_ARG: @experimental takes no arguments`.

These codes are inline diagnostic codes, like `TC_MATCH_005`; they are not entries of the generated `error_codes.json` registry, which covers the `PREFIX###` codes in `internal/errors/codes.go`.

## Individual error code pages

- [BC001 — Function is evaluator-only under --strict-bytecode](bc001.md)
- [MOD007 — Duplicate module-scope binding (let/func same name)](mod007.md)
- [MOD013 — Shared module_prefix](mod013.md)
- [MOD015 — Imported name collides with a module-level definition](mod015.md)
- [RT001 — Integer division or modulo by zero](rt001.md)
- [Effect row mismatch](typ_effect_row_mismatch.md)

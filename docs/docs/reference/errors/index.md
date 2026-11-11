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

**`TC_MATCH_002` — a `match` that misses a case.** Every `match` must cover every value of the scrutinee's type with its unguarded arms: every constructor of an ADT (all of them, from the type's own declaration, not only the ones a module imports), `true` and `false`, `[]` and a non-empty list, every combination inside tuples, records and nested patterns. `int`, `float` and `string` have infinitely many values, so a match on them needs a catch-all (`_` or a variable). Arms with an `if` guard never count, because the guard may be false. The error names concrete missing patterns, in syntax you can paste as an arm (`Blue`, `Some(Err(_))`, `[]`, `(true, false)`, `{a: false, ...}`), and appears in human, `--json` and `--format agent` output with the match's file:line:col. Every way of compiling a program (`check`, `run`, `run --bytecode`, `run --strict-bytecode`, `compile --emit-go`, `test`) gives it before anything runs. Fix: add the arm, end with `_ => …`, or — for a case that cannot happen — write `C => unreachable("why")`. Through v0.60.0 such a match passed `check` and failed at run time.

```text
Error: p1.ail:3:38: TC_MATCH_002: non-exhaustive match on Color — no arm matches: Blue
  Suggestion: add the missing arm: Blue => ...
  If this case cannot happen, say so: Blue => unreachable("<why it cannot happen>")
```

The same code, with "this match cannot be checked for exhaustiveness", is reported when the checker cannot place a constructor in a type (a compiler bug if it ever fires): it fails closed, and an unguarded `_ =>` arm resolves it.

**`TC_MATCH_003` — an arm no value can reach (warning).** Every value the arm matches is already matched by an unguarded arm above it (`Running(n) => …, other => …, Idle => …` warns on `Idle`). It never blocks compilation; `check --json` lists it under `warnings` and `--format agent` prints a `warn:` line, also when the module is served from the compile cache.

**`TC_MATCH_004` — a match too large to check.** The check stops after 10^6 steps per match rather than guess. No program in the measured corpora comes near it. Fix: add a final `_ =>` arm.

**`TC_MATCH_006` — a constructor pattern with the wrong number of arguments.** `C(n)` against `type U = C(int, int) | E`. Such a pattern never matched on the evaluator while the bytecode VM bound the first field (#2243), so its coverage was not one thing; it is now an error located at the pattern, with the right shape as the fix (`write C(_, _)`).

**`TC_MATCH_005` — `Json` match with a `JNumber` arm and no `JInt` arm (warning).** Transitional for v0.58.x. `std/json.decode` gives `JInt(n)` for integer literals and `JNumber(x)` for other numbers, so a match written before `JInt` existed, `match j { JNumber(n) => …, _ => … }`, compiles but sends every integer to the catch-all. The warning is keyed on std/json's own `JNumber` (aliased imports included; a module's own `JNumber` constructor is never flagged), looks at every pattern depth (`Some(JNumber(n))` counts), and is reported on a warm compile cache too. It never blocks compilation. Fix: add a `JInt(i) => …` arm (`intToFloat(i)` gives a float), or read numbers with `asNumber`/`getNumber`/`getInt`, which accept both.

```
TC_MATCH_005 at app.ail:6:45
  this match over std/json.Json has a JNumber arm but no JInt arm; decode now gives JInt(n) for integer literals such as 42, so integers skip the JNumber arm.
  Fix: add a `JInt(i) => …` arm (intToFloat(i) gives a float), or read numbers with asNumber/getNumber/getInt, which accept both
```

**`TC_NOMINAL_001` — two different types that share a name.** A nominal type (an ADT) is its defining module plus its name, so `q/a`'s `type T` and `q/b`'s `type T` are different types. Emitted where a value of one is used where the other is expected; both modules are named. Until v0.60.0 such programs type-checked and failed at run time. See [tc_nominal_001.md](tc_nominal_001.md).

```
TC_NOMINAL_001: cannot unify T (defined in q/a) with T (defined in q/b): they are different types that share a name.
  Suggestion: use the T from one module on both sides (import it by name from the module whose T you mean, e.g. import q/a (T)), or convert between them explicitly.
```

**`TC_AMBIGUOUS_TYPE_001` — a type name two modules both declare.** A bare type name that is not declared in the module, not imported by name, not a built-in or prelude name, and that two or more imported modules export — or, when no imported module exports it, that two or more modules reached only indirectly (marked "(indirect)") declare. A module an import reaches indirectly never competes with a name an imported module exports. The candidates are listed in module order; the fix is to import the one you mean by name. It used to mean whichever module happened to come first.

**`TC_AMBIGUOUS_CTOR_001` — a constructor two imported modules both offer.** The constructor auto-import brings every constructor of a selectively imported module into scope; when two modules offer the same name and neither is imported by name, a pattern is resolved by the scrutinee's type once inference has decided it (a deferred choice, independent of the order of the code; the pattern's own number of arguments narrows the candidates first). The error is reported, at the pattern or expression, for an expression; for a pattern whose scrutinee's type nothing in the enclosing top-level declaration decides (fix: import the constructor by name, or annotate the value matched); for a pattern whose argument count fits another candidate than the type matched (fix: the argument count); and as the cause of a conflict it creates — a let-bound function matching the constructor stays monomorphic while the choice is open, so using it at two types (`Box[int]` and `Box[string]`) fails; the message names the function and the conflict. Fix for expressions and conflicts: `import M (C)` from the module you mean.

## Individual error code pages

- [BC001 — Function is evaluator-only under --strict-bytecode](bc001.md)
- [MOD007 — Duplicate module-scope binding (let/func same name)](mod007.md)
- [MOD013 — Shared module_prefix](mod013.md)
- [MOD015 — Imported name collides with a module-level definition](mod015.md)
- [RT001 — Integer division or modulo by zero](rt001.md)
- RT002 — no arm of a `match` matched, with the match's position (defence in depth: a checked program cannot reach it, except through a checker bug)
- RT010 — `unreachable(msg)` was evaluated: `RT010: unreachable code reached: <msg>`
- [TC_NOMINAL_001 — Two different types that share a name](tc_nominal_001.md)
- [Effect row mismatch](typ_effect_row_mismatch.md)

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

**`TC_MATCH_002` — a `match` that misses a case.** Every `match` must cover every value of the scrutinee's type with its unguarded arms: every constructor of an ADT (all of them, from the type's own declaration, not only the ones a module imports), `true` and `false`, `[]` and a non-empty list, every combination inside tuples, records and nested patterns. `int`, `float` and `string` have infinitely many values, so a match on them needs a catch-all (`_` or a variable). Arms with an `if` guard never count, because the guard may be false. The error names concrete missing patterns, in syntax you can paste as an arm (`Blue`, `Some(Err(_))`, `[]`, `(true, false)`, `{a: false, ...}`; at most five, then "and more"), and appears in human, `--json` and `--format agent` output with the match's file:line:col. Every way of compiling a program (`check`, `run`, `run --bytecode`, `run --strict-bytecode`, `compile --emit-go`, `test`) gives it before anything runs. Fix: add the arm, end with `_ => …`, or — for a case that cannot happen — write `C => unreachable("why")`. Through v0.60.0 such a match passed `check` and failed at run time.

```text
Error: p1.ail:3:38: TC_MATCH_002: non-exhaustive match on Color — no arm matches: Blue
  Suggestion: add the missing arm: Blue => ...
  If this case cannot happen, say so: Blue => unreachable("<why it cannot happen>")
```

The same code, with "this match cannot be checked for exhaustiveness", is reported in two cases:

- **A constructor matched by name against a value nothing types.** When neither a constructor nor its type is imported, the pattern names it by name and constrains nothing, so a value of any type can reach the match (`let h = \e. match e { Msg(_) => 1, Done => 2 }` where `Msg` is declared only in a module you import something else from). Fix: import the constructor's type, `import bn/a (Ev)`; its constructors come into scope and fix the value's type. Annotating the value matched also works.
- **The checker cannot place a constructor in a type** (a compiler bug if it ever fires). It fails closed; an unguarded `_ =>` arm resolves it.

**`TC_MATCH_003` — an arm no value can reach (warning).** Every value the arm matches is already matched by an unguarded arm above it (`Running(n) => …, other => …, Idle => …` warns on `Idle`). It never blocks compilation; `check --json` lists it under `warnings` and `--format agent` prints a `warn:` line, also when the module is served from the compile cache.

**`TC_MATCH_004` — a match too large to check.** The check stops after 10^6 steps per match rather than guess. No program in the measured corpora comes near it. Fix: add a final `_ =>` arm.

**`TC_MATCH_006` — a constructor pattern with the wrong number of arguments.** `C(n)` against `type U = C(int, int) | E`. Such a pattern never matched on the evaluator while the bytecode VM bound the first field (#2243), so its coverage was not one thing; it is now an error located at the pattern, with the right shape as the fix (`write C(_, _)`), whatever the other arms are — a catch-all does not hide it. A constructor matched by name against a value whose type nothing fixes, when two loaded types declare that name with different arities (`ar/x`'s `Msg(string)` and `ar/y`'s `Msg(string, int)`), is the same error whatever its own count: the engines would disagree on which values it matches. The message names every candidate; the fix is to import the type you mean.

**`TC_BUILTIN_SHADOW_001` — a module-level definition or an import named `unreachable`.** A builtin wins over a module-level name, so a program defining its own `unreachable`, or importing something under that name (`import std/string (trim as unreachable)`), would silently call the builtin (`RT010`) instead. Rename the definition or the import; a local `let unreachable = …` is unaffected.

**`TC_CONTRACT_001` — a contract clause does not type-check.** A `requires`/`ensures` clause is type-checked like a function body: it must be a `bool` over the function's parameters (and `result`, in `ensures` only), with every name in scope and every `==` on a type with `Eq` (an enum compared in an `ensures` needs `deriving (Eq)`, exactly as in a body; the message names the type and where it is declared). A bounded `forall i: lo..hi => p` needs `int` bounds and a `bool` body. Before #2248 the clause's inference error was swallowed, and a clause containing `forall` was never checked at all, so `requires { forall i: 0..2 => i + "x" }` passed `check`.

```text
TC_CONTRACT_001: requires clause of total at app.ail:3:12 does not type-check (a contract is a bool predicate over the parameters): type unification failed at [arithmetic at app.ail:3:32]: cannot unify type constructors: int vs string
```

**`TC_TEST_001` — a test body does not type-check.** Named `test "…" { }` blocks, `property "…" { }` blocks, inline `properties [ … ]` and inline `tests [ … ]` rows of the module being checked or run are type-checked in the form `ailang test` compiles them: a test or property body is checked pure; a property, and a test body with no `assert` and no earlier bare check, must end in a `bool` (the runner fails anything else with "expected bool result"); and a row `(inputs, expected)` must type as a call of the function whose result has the expected value's type (stricter than the runner, which compares row values without type-checking them). A dependency's test blocks and rows are not checked when it is imported: they are not part of the importer's program; `check` of the dependency itself (or `check --package`) reports them. Before #2232, `check` never looked at them, so a test that could not compile passed `check`. Fix the body, add the missing import, or move effectful checks into an exported entry run with `ailang run --caps …`.

```text
TC_TEST_001: test "illtyped" at app.ail:10:1 does not type-check: type unification failed at [function application of absInt at app.ail:11:16]: failed to unify parameter 0: cannot unify type constructors: int vs string
```

**`TC_FORALL_001` — `forall` outside a contract.** The bounded quantifier `forall i: lo..hi => p` exists for `requires`/`ensures` clauses, where the verifier encodes it; nothing evaluates it at run time. In a function body, a `let` or a test body it is this error (before, it was an uncoded `type inference not implemented for *core.Forall` error). Compute the condition with recursion or a `std/list` function such as `any`, or move it into a contract.

```text
TC_FORALL_001: `forall i: lo..hi => p` at app.ail:3:43 is a contract quantifier: it is allowed only in a requires/ensures clause
```

**`TC_MATCH_005` — `Json` match with a `JNumber` arm and no `JInt` arm (warning).** Transitional for v0.58.x. `std/json.decode` gives `JInt(n)` for integer literals and `JNumber(x)` for other numbers, so a match written before `JInt` existed, `match j { JNumber(n) => …, _ => … }`, compiles but sends every integer to the catch-all. The warning is keyed on std/json's own `JNumber` (aliased imports included; a module's own `JNumber` constructor is never flagged), looks at every pattern depth (`Some(JNumber(n))` counts), and is reported on a warm compile cache too. It never blocks compilation. Fix: add a `JInt(i) => …` arm (`intToFloat(i)` gives a float), or read numbers with `asNumber`/`getNumber`/`getInt`, which accept both.

```
TC_MATCH_005 at app.ail:6:45
  this match over std/json.Json has a JNumber arm but no JInt arm; decode now gives JInt(n) for integer literals such as 42, so integers skip the JNumber arm.
  Fix: add a `JInt(i) => …` arm (intToFloat(i) gives a float), or read numbers with asNumber/getNumber/getInt, which accept both
```

**`TC_RIGID_001` — a signature promises more generality than the body keeps.** A type variable written in a top-level `func`'s signature — listed in `[a]` or used implicitly (`func f(x: elem) -> elem`) — means *any type*, and the body must work for every one. Emitted when the body makes it a specific type (`func toInt[a](x: a) -> int = x`), makes two of them the same type (`func pick[a, b](x: a, y: b) -> a = y`), or uses an integer literal at it (`func bump[a](x: a) -> a = x + 1` — a literal has no value of an arbitrary type, and the runtime would give it an `int`). Class constraints are still inferred and attached (`func add[a](x: a, y: a) -> a = x + y` is fine): AILANG has no syntax for writing one. When the variable is probably a typo for a type in scope (`state` for `State`, `strng` for `string`) or the name of a parameter (`-> (stack, int)`), the message says so. Fix: write the concrete type in the signature, use one variable where the body needs one type, or make the body generic. Through v0.60.x such signatures were silently narrowed to whatever the body needed.

```
TC_RIGID_001: the signature of `step` says `state` is any type, but the body requires `state` to be `{n: int | r}` (field access). Did you mean the type `State`? A lowercase name in a signature that is not a declared or imported type is a type variable
  Suggestion: write `State` instead of `state`
```

**`TC_NOMINAL_001` — two different types that share a name.** A nominal type (an ADT) is its defining module plus its name, so `q/a`'s `type T` and `q/b`'s `type T` are different types. Emitted where a value of one is used where the other is expected; both modules are named. Until v0.60.0 such programs type-checked and failed at run time. See [tc_nominal_001.md](tc_nominal_001.md).

```
TC_NOMINAL_001: cannot unify T (defined in q/a) with T (defined in q/b): they are different types that share a name.
  Suggestion: use the T from one module on both sides (import it by name from the module whose T you mean, e.g. import q/a (T)), or convert between them explicitly.
```

**`TC_REC_001` — a record has no such field.** A field access, a record update, an open pattern (`{name, ...}`) or an open annotation (`{name: string | r}`) requires a field the record does not have. The record's own fields are listed. Before v0.60.x (#2210) the check was skipped when the record reached the access through a lambda parameter, so such programs type-checked and failed at run time with `record has no field`.

```
TC_REC_001: field 'bogus' not found in record {label, score}
```

**`TC_REC_004` — a field with two types.** A row variable extends two records (`func f(p: {y: int | r}, q: {x: int | r})`), and a call binds it to a field one of them already has (`f({y: 1, x: "s"}, {x: 2})` makes `r = {x: string}`, so `q` would have `x` as both `int` and `string`). The two copies are one field and must agree. Fix: give the two records separate row variables, or make the types agree.

```
TC_REC_004: field 'x' would have two types, int and string: a row variable shared by two records was bound to a field one of them already has
```

**`TC004` / `TC_REC_003` — an infinite type.** A type variable (`TC004`) or a record row (`TC_REC_003`) would have to contain itself, as in `\x. {self: x} == x` or `letrec f = \x. {inner: f(x)}`. Before #2221 a record in the cycle made `check` run forever.

```
TC004: occurs check failed: α1 occurs in { self: α1 } (an infinite type)
```

**`TC_NEWTYPE_001` — a single-constructor type is not its record.** `type W = W({ x: int })` is an ADT, not the record `{ x: int }` (that would be the record alias `type W = { x: int }`). Field access `w.x` works on a `W`; anywhere else a `W` meeting a record type — a bare record passed as a `W`, a `W` passed as a record, a record update on a `W` — is this error. Until v0.60.2 the two converted silently, so a record was accepted as a `W` and two such types over the same record could reach each other. See [tc_newtype_001.md](tc_newtype_001.md).

**`TC_AMBIGUOUS_TYPE_001` — a type name two modules both declare.** A bare type name that is not declared in the module, not imported by name, not a built-in or prelude name, and that two or more imported modules export — or, when no imported module exports it, that two or more modules reached only indirectly (marked "(indirect)") declare. A module an import reaches indirectly never competes with a name an imported module exports. The candidates are listed in module order; the fix is to import the one you mean by name. It used to mean whichever module happened to come first.

**`TC_AMBIGUOUS_CTOR_001` — a constructor two imported modules both offer.** The constructor auto-import brings every constructor of a selectively imported module into scope; when two modules offer the same name and neither is imported by name, a pattern is resolved by the scrutinee's type once inference has decided it (a deferred choice, independent of the order of the code; the pattern's own number of arguments narrows the candidates first). The error is reported, at the pattern or expression, for an expression; for a pattern whose scrutinee's type nothing in the enclosing top-level declaration decides (fix: import the constructor by name, or annotate the value matched); for a pattern whose argument count fits another candidate than the type matched (fix: the argument count); and as the cause of a conflict it creates — a let-bound function matching the constructor stays monomorphic while the choice is open, so using it at two types (`Box[int]` and `Box[string]`) fails; the message names the function and the conflict. Fix for expressions and conflicts: `import M (C)` from the module you mean.

**`TC_UNBOUND_TYPE_001` — a lowercase name applied to type arguments that names no type.** Whether a name in type position is a type variable is decided by scope: a name an enclosing `[T]` list binds is a variable (any case), a lowercase name the module declares as a type or imports by name is that type, and any other lowercase name is an implicitly quantified variable. A lowercase name written with arguments (`zz[int]`) has no variable reading (type variables cannot be applied), so when no type of that name is declared, imported by name, or exported by an import, it is this error. When a type is spelled the same up to case (`option[int]`, `map[string, int]`), the suggestion names it (`Option`, `Map`). Uppercase undeclared names are not reported yet (#1485).

```
TC_UNBOUND_TYPE_001: type 'zz' is not in scope: no type constructor 'zz' is declared here, imported by name, or exported by an import.
```

**`TC_UNBOUND_TYPE_003` — a type declaration whose body names a type variable that is not its parameter.** A function signature quantifies an undeclared lowercase name implicitly (the caller supplies it); a type declaration cannot, so in the body of a `type` declaration — alias, constructor fields, record fields — a lowercase name must be one of the declaration's own `[..]` parameters or a type in scope. Otherwise the type would silently stand for any type. Fix: add the parameter, or write the type you meant; the message suggests the built-in a name from another language means (`str` → `string`, `number`/`double` → `float`, `integer` → `int`, `boolean` → `bool`), a type differing only by case, or one edit away (two when both names have six letters or more), or the import when an imported module exports the type but it is not imported by name; with nothing close there is no suggestion.

```
TC_UNBOUND_TYPE_003: type variable 'integer' is not a parameter of Count: write `type Count[integer] = …` to make it a parameter, or did you mean the type `int`?
```

**`TC_TYPE_ALIAS_CYCLE_001` — a type alias that refers to itself.** An alias (`type X = <type expression>`, not a constructor list or a record) stands for its body, so an alias that mentions itself, directly (`type L[a] = [L[a]]`) or through other aliases (`type A = B`, `type B = A`), has no finite meaning. Recursion needs a nominal type: make it an ADT with a constructor. A cycle through an ADT or a record declaration (`type Node = {label: string, kids: [Node]}`) is allowed.

```
TC_TYPE_ALIAS_CYCLE_001: type alias 'T2' refers to itself; a recursive type must be an ADT: write `type T2[a] = T2((T2[a], T2[a]))` (a constructor) instead
```

**`TC_UNBOUND_TYPE_002` — a lowercase name read as a type variable that may have meant a type (warning).** Two readings keep a lowercase name a type variable, as they were before #2211, and warn: an implicit variable named like a type that an imported module exports but this module does not import by name (`import std/sem (validate_namespace)` then `n: namespace`; fix: `import std/sem (namespace)` if the type was meant, otherwise rename the variable), and a name imported by name that is a value, not a type (`import std/list (map)` then `(map) -> map`). One warning per declaration and name; reported on a warm compile cache too. It never blocks compilation: an error would let a library break every importer just by exporting a type named like a common type variable.

```
TC_UNBOUND_TYPE_002 at app.ail:3:18
  'namespace' is treated as a type variable here; std/sem exports a type 'namespace', which is not imported by name.
  Fix: import it by name, import std/sem (namespace), if you meant the type; otherwise rename the type variable
```

**`PAR_TYPE_DECL_BUILTIN` — a type declaration named like a built-in type (parse error).** A built-in name always means the built-in, so `type float = float` or `type string = int` declares a type nothing can refer to (the second used to be silently ignored at every use). Fix: give the type its own name (`type Meters = float`, `type Name = string`) and use that name.

**`PAR_TYPE_PARAM_BUILTIN` — a type parameter named like a built-in type (parse error).** A built-in (`int`, `string`, `float`, `bool`, `unit`, `char`, `bytes`) is never a type variable, so `type Result[int] = Ok(int) | Err(string)` or `func f[string](x: string)` declares a parameter that nothing can refer to. Fix: rename the parameter (`[a]`, `[T]`) and use that name in the declaration.

**`PAR_TYPE_PARAM_APPLIED` — a type parameter applied to type arguments (parse error).** There are no higher-kinded type parameters, so `func same[m](x: m[int])` cannot mean "any container of int". Before #2240 the parameter list was ignored and `m[int]` silently named an opaque type that no value could have. Fix: use a concrete type constructor (`Option[int]`, `[int]`).

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
- RT002 — no arm of a `match` matched, with the match's position (defence in depth: a checked program cannot reach it, except through a checker bug)
- RT010 — `unreachable(msg)` was evaluated: `RT010: unreachable code reached: <msg>`
- [TC_NEWTYPE_001 — A single-constructor type is not its record](tc_newtype_001.md)
- [TC_NOMINAL_001 — Two different types that share a name](tc_nominal_001.md)
- [Effect row mismatch](typ_effect_row_mismatch.md)

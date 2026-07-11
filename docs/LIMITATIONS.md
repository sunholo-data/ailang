# AILANG Known Limitations

> **Canonical page:** the maintained, live-verified limitations list is the published reference —
> **[docs/docs/reference/limitations.md](docs/docs/reference/limitations.md)** (rendered at
> <https://ailang.sunholo.com/docs/reference/limitations>). This root file is a pointer + short
> summary; the website copy is authoritative and carries per-entry repro transcripts + verified-at
> dates.

**All entries below were live-verified at AILANG `v0.28.0-141-g379990ad5` on 2026-07-10.** Per the
[M-V1-STABILITY-PROMISE](design_docs/planned/v1_0_0/m-v1-stability-promise.md) entry policy, every
open limitation is a reproducible artifact with a verified-at date, and fixed items move to a
dated "Resolved" list — this file is no longer allowed to freeze at a past version.

## Open limitations (summary)

See the [canonical page](docs/docs/reference/limitations.md) for repros, transcripts, and
workarounds. Verified open at v0.28.0 (2026-07-10):

| Limitation | Kind | Verified-at repro |
|---|---|---|
| **Y-combinator / recursive lambdas** | Design constraint (Hindley-Milner occurs-check) | `let Y = \f. (\x. f(x(x)))(\x. f(x(x)))` → `occurs check failed`. Use named `func` recursion. |
| **If-else multi-statement branches need braces** | Design constraint (no layout-sensitive parsing) | bare `let` in an `else` → "if-else branches require explicit braces". Wrap in `{ … }`. |
| **Duplicate record types with identical fields** | Go-codegen only (interpreter unaffected) | `--emit-go` may pick the first structurally-matching struct. `ailang run` returns the correct value. |
| **WASM type-checker depth limit** | WASM-host only (CLI unaffected) | deeply-recursive type structure exceeds the JS-engine stack; structured `depth budget exceeded` error. |
| **`?` error-propagation operator** | Not yet implemented (planned) | `r?` → `PAR_NO_PREFIX_PARSE: unexpected token in expression: ?`. Use explicit `match` on `Result`. |
| **Typed quasiquotes** | Not yet implemented (planned) | quasiquote syntax not accepted. Use `"${expr}"` interpolation / `concat([..])`. |
| **CSP concurrency (channels / session types)** | Deferred | no channel/session-type surface in the parser. |
| **Raw-mode single keypress / mid-call `std/ai.step()` abort** | Narrow input gaps | line input (`readLine`, `asyncReadStdinLines`) works; raw keypress is out-of-core by design. |

### If-Else Branches Require Explicit Braces {#if-else-branches-require-explicit-braces}

AILANG is not layout-sensitive, so a multi-statement `if`/`else` branch must be wrapped in braces.
Without them, only the first `let` is parsed as the branch and you get
"if-else branches require explicit braces when using let bindings":

```ailang
-- ❌ fails
if x > maxX then [] else
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest

-- ✅ works
if x > maxX then [] else {
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest
}
```

Single-expression branches don't need braces. See the
[canonical page](docs/docs/reference/limitations.md) for the full entry.

## Resolved (were documented as broken; re-verified working at v0.28.0)

- **Polymorphic arithmetic lambdas** — `let add = \x. \y. x + y in add(3.14)(2.71)` → `5.85`
  (fixed v0.7.0, [m-poly-arithmetic-fix](design_docs/implemented/v0_7_0/m-poly-arithmetic-fix.md)).
- **`match` inside block-body lambdas in HOF arguments** —
  `map(\item. { let s = match item { 0 => "zero", _ => "ok" }; s }, [0,1,2])` → `[zero, ok, ok]`
  ([design doc archived](design_docs/archive/v0_13_0_m-dx-match-in-hof-block-lambda.md)). The old
  ML/Haskell `match … with | …` form is **retired** — it now emits `PAR019`; use brace-form
  `match x { pat => expr }`.
- **Multi-statement block expressions** — `{ e1; e2; e3 }` sequencing works fully; the old
  `let _ = … in` workaround is no longer needed.
- **Forgiving statement separators (v0.29+, M-SYNTAX-AI-FORGIVING)** — a `;`-separated
  sequence is now accepted directly in a `=`-body (`func f() = s1; s2; e`), and a
  **newline** is a soft statement separator inside `{ }` blocks (`{ let x = e\n rest }`).
  Both parse to the same AST as the `;`-in-braces form; a `;` is only *required* to
  separate two statements on the **same line**. Records still use commas.
- **String interpolation** — `"Value: ${x}"` → `Value: 42` (v0.12.1); `++` is now list-only
  (string `++` is a type error, v0.13.0).
- **Pattern guards** — `match x { n if n > 100 => …, n if n > 0 => …, _ => … }` (v0.6.2).

## Reporting New Limitations

File an issue at <https://github.com/sunholo-data/ailang/issues> with: AILANG version
(`ailang --version`), a minimal repro, expected vs actual behavior, and whether it's a bug or a
design constraint.

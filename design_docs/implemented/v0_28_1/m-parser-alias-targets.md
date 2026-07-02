## M-PARSER-ALIAS-TARGETS: parse tuple- and function-type aliases (`type Pair = (a, b)`, `type Handler = (Req) -> Resp`)

**Status**: IMPLEMENTED (on `dev`, 2026-07-02; first release: v0.28.1 tbd)
**Target**: v0.28.1 (patch-level)
**Priority**: P3 (Low–Medium — function-type aliases (`type Handler = (Req) -> Resp`) are an idiomatic readability win; workaround is to inline the type in each signature. Tuple aliases similar.)
**Estimated**: ~0.5 day (one parser dispatch case + tests)
**Dependencies**: None.
**Related**: extends the original alias feature [v0_4_9 m-feat-type-alias](../archive/v0_4_9_m-feat-type-alias.md) (which shipped `type Foo = Bar` / `type Names = [x]` but never tuple/function targets); surfaced by [M-XMOD-ALIAS](implemented/v0_28_1/m-cross-module-type-alias.md).

**Discovered by**: M-XMOD-ALIAS — its `NonRecordTargetKinds` fixture had to drop tuple/function targets because they don't parse.
**Verified against**: v0.28.0 — both fail to **parse** within a single module (below), so this is purely a parser gap; the type checker/interface layer already handle these type kinds fine in function signatures.

---

## Problem

The parser rejects tuple- and function-typed alias targets:

```ailang
type Pair = (int, string)      -- PAR_TYPE_BODY_EXPECTED at `(`, then PAR_NO_PREFIX_PARSE at `,`
type Pred = (int) -> bool      -- PAR_NO_PREFIX_PARSE at `->`
```

Both `(int, string)` and `(int) -> bool` are valid types *in function-parameter position* — only the
`type X = …` alias-target position rejects them.

## Root cause

[`internal/parser/parser_type_decl.go:79`](../../internal/parser/parser_type_decl.go#L79)
(`parseTypeDeclBody`) dispatches on the first token of the alias target:

| First token | Handled? | Result |
|---|---|---|
| `{` | yes (L93) | record type |
| `[` | yes (L98) | list alias via `parseType()` |
| IDENT | yes (L111) | named alias / sum type |
| **`(`** | **no** | falls through to `PAR_TYPE_BODY_EXPECTED` (L221) |

There is simply no `LPAREN` case, so a target beginning with `(` (every tuple and parenthesized function
type) hits the catch-all error.

## Proposed fix

Add an `LPAREN` case that defers to `parseType()` (which already parses tuples and function types in
signature position), mirroring the existing `LBRACKET` list case:

```go
// Tuple or function type alias: type Pair = (int, string) / type Pred = (int) -> bool
if p.curTokenIs(lexer.LPAREN) {
    typeExpr := p.parseType()
    return &ast.TypeAlias{Target: typeExpr, Pos: p.curPos()}
}
```

Place it alongside the `LBRACKET` branch (before the IDENT disambiguation).

## Conflict Surface

*(Touches `internal/parser/` — required.)*

**Position extended:** the first token of a `type X = <target>` body — adding `(` to the recognized set.

**What else lives in that position?** `{` (record), `[` (list), IDENT (named/sum, incl. `Ctor(int)` sum
constructors), and a leading `|` (Haskell-style sum). `(` is currently **only** the error path — nothing
valid begins an alias target with `(` today — so adding the case cannot shadow an existing construct.

**Disambiguation:** dispatch is on the literal first token; `(` is disjoint from `{`, `[`, `|`, and IDENT.
Note the sum-constructor case `type Shape = Circle(int)` starts with an **IDENT** (`Circle`), not `(`, so it
is untouched.

**Programs that MUST still work (fixtures):**
1. `type Foo = Bar` (named alias) — unchanged.
2. `type Names = [string]` (list) — unchanged.
3. `type Point = { x: int }` (record) — unchanged.
4. `type Color = Red | Green` and `type Box = Box(int)` (sum / newtype) — unchanged (IDENT path).
5. **New:** `type Pair = (int, string)` parses to `TypeAlias{Target: TupleType}`; `type Pred = (int) -> bool`
   parses to `TypeAlias{Target: FuncType}`; both usable cross-module (composes with M-XMOD-ALIAS).

**Intentional change:** tuple/function alias targets now parse. No previously-valid program changes meaning.

## Test plan

- Parser unit tests: the two new forms produce `*ast.TypeAlias` with `*ast.TupleType` / `*ast.FuncType`
  targets; the 4 pre-existing forms still parse to their existing nodes (regression-surface fixtures).
- Pipeline: extend `TestXModAlias_NonRecordTargetKinds` to re-add the tuple + function cases now that they parse.
- `make test` + `make verify-examples` green.

## Out of scope

- Parameterized aliases (`type Pair[a] = (a, a)`) — separate `M-XMOD-ALIAS-POLY` concern.

---

## Implementation Report (2026-07-02, on `dev`)

Added the `LPAREN → parseType()` branch to `parseTypeDeclBody`
([internal/parser/parser_type_decl.go](../../internal/parser/parser_type_decl.go)), exactly as proposed
(~8 LOC). Tuple/function targets lower via the existing `astTypeToInternalType` `TupleType`/`FuncType`
cases and cross module boundaries for free through M-XMOD-ALIAS — verified end-to-end.

**Tests:** `TestParseTupleAndFuncTypeAliases` (parser, pins AST node shape + guards named/list forms) and
`TestXModAlias_TupleAndFuncAliasTargets` (pipeline, cross-module use). Both red before the fix
(`PAR_TYPE_BODY_EXPECTED`), green after. Full parser package + type-system sweep green.

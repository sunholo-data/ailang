---
title: Reserved Keywords
---

# Reserved Keywords

The lexer keyword map contains **41 entries**. Most cannot be used as names.
The four contextual testing words remain legal identifiers where a testing marker
is not expected; their reservation is contextual, not global. `case` is an ordinary
identifier, not a keyword. No words are released by this diagnostic change.

A reserved word in a parameter name produces `PAR_RESERVED_KEYWORD` at that word.
The invalid parameter is dropped for parser recovery; the source still fails checking.
Record-type fields retain `PAR_FIELD_NAME_EXPECTED` with a reservation explanation.
Rename the parameter and all of its uses before checking again.

## Complete keyword map

| Word | Current use or reservation, owner and trigger |
| --- | --- |
| `func` | Function declaration or parenthesized function literal. |
| `pure` | Pure function marker. |
| `let` | Immutable binding. |
| `letrec` | Recursive binding. |
| `in` | Binding scope. |
| `if` | Conditional or match guard. |
| `then` | Conditional true branch. |
| `else` | Conditional false branch. |
| `match` | Pattern matching with braces (no `with`). |
| `with` | PAR019 dialect detection and planned effect-handler syntax. Handler lane: consume or release at syntax freeze. Alternatives: `using`, `given`, `w`. |
| `type` | Type declaration. |
| `class` | Type-class declaration syntax. |
| `instance` | Type-class instance syntax. |
| `module` | Module declaration. |
| `import` | Import declaration. |
| `export` | Export marker. |
| `extern` | External declaration. |
| `forall` | Universal type quantification. |
| `exists` | Reserved for existential type quantification; the type-system lane must consume or release it at syntax freeze. |
| `test` | Contextual named test marker; also a legal identifier. |
| `tests` | Contextual test block marker; also a legal identifier. |
| `property` | Contextual property marker; also a legal identifier. |
| `properties` | Contextual property block marker; also a legal identifier. |
| `assert` | Reserved for planned assertion builtin; assertion/testing lane must consume or release at syntax freeze. Alternatives: `check`, `verify`. |
| `spawn` | Reserved for planned concurrency lane (CSP/session types). Consume or release at concurrency syntax freeze. Alternatives: `chan`, `fork`, `concurrent`, `pick`. |
| `parallel` | Reserved for planned concurrency lane (CSP/session types). Consume or release at concurrency syntax freeze. Alternatives: `chan`, `fork`, `concurrent`, `pick`. |
| `select` | Reserved for planned concurrency lane (CSP/session types). Consume or release at concurrency syntax freeze. Alternatives: `chan`, `fork`, `concurrent`, `pick`. |
| `channel` | Reserved for planned concurrency lane (CSP/session types). Consume or release at concurrency syntax freeze. Alternatives: `chan`, `fork`, `concurrent`, `pick`. |
| `send` | Reserved for planned CSP/session-types lane. CSP lane: consume or release at syntax freeze. Alternatives: `tx`, `rx`, `deadline`, `expires`. |
| `recv` | Reserved for planned CSP/session-types lane. CSP lane: consume or release at syntax freeze. Alternatives: `tx`, `rx`, `deadline`, `expires`. |
| `timeout` | Reserved for planned CSP/session-types lane. CSP lane: consume or release at syntax freeze. Alternatives: `tx`, `rx`, `deadline`, `expires`. |
| `as` | Import alias. |
| `deriving` | Type-class derivation. |
| `requires` | Precondition contract. |
| `ensures` | Postcondition contract. |
| `invariant` | Invariant contract. |
| `true` | Boolean literal. |
| `false` | Boolean literal. |
| `not` | Boolean negation. |
| `and` | Boolean conjunction. |
| `or` | Boolean disjunction. |

## Reservation decisions

The [effect-handler lane](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_1_0/m-effect-handlers.md)
owns the proposed `handle ... with` syntax. Its syntax freeze must explicitly
consume `with` as syntax or release it. This is separate from its current PAR019
role: `match x with { ... }` is rejected with the existing dialect diagnostic;
AILANG uses `match x { ... }` and guards use `if`.

The [CSP/session-types lane](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_1_0/m-csp-session-types.md)
owns channel communication and concurrency reservations. Its examples are
unfrozen sketches, including function-form receive/send; they do not establish
that those words need keyword status. At syntax freeze the lane must explicitly
consume or release each listed word. The assertion/testing lane has the same
obligation for its planned assertion builtin (see the
[named-test assertion plan](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_33_1/m-named-test-assert-lowering-sprint-plan.md)).
No syntax-freeze decision is made here. Until its owner rules, each reservation
remains. A release must update the keyword map, token enum, token strings,
`IsKeyword`, and this reference together.

## Working rename example

```ailang
module examples/runnable/reserved_keyword_names

pure func combine(a: int, rx: int, b: int) -> int = a + rx + b
pure func usingValue(using: int) -> int = using
export pure func main() -> int = combine(1, 2, 3) + usingValue(4)
```

Use `ailang check examples/runnable/reserved_keyword_names.ail` to check the
corrected three-parameter call. Recovery of an invalid parameter is not a
successful type check and does not preserve a three-argument call's arity.

See [Language Syntax](./language-syntax.md), or run `ailang prompt` for the
current teaching prompt and `ailang check file.ail` for diagnostics.

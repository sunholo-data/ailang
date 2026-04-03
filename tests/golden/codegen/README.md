# Golden Test Corpus for Go Codegen

These files test the AILANG → Go compilation pipeline end-to-end.

## Structure

Each test has:
- `<name>.ail` — AILANG source input
- `<name>.go.golden` — Baseline Go output captured from the old codegen (reference only)

## Test Coverage

| File | Features Tested |
|------|----------------|
| `literals.ail` | int, float, bool, string, unit literals |
| `arithmetic.ail` | binary ops (+, *, >, &&), int/float arithmetic |
| `functions.ail` | identity, higher-order, recursion, let-chain |
| `adt_simple.ail` | nullary constructors, single-arg ADT, match |
| `adt_multiarg.ail` | multi-arg constructors, match with field extraction |
| `records.ail` | record type, literal, field access, record update |
| `lists.ail` | empty list, cons (::), list pattern match |
| `if_else.ail` | if/else, chained if-else, multi-arg functions |
| `match_patterns.ail` | recursive ADT, nested match, literal patterns |
| `let_bindings.ail` | nested lets, variable shadowing |
| `string_ops.ail` | string concatenation (++), string equality |
| `tuples.ail` | tuple creation, tuple pattern matching, wildcard |

## Usage

The golden files are **reference baselines** from the old codegen. The new pipeline does NOT
need to produce identical output — it needs to produce Go that:
1. Compiles with `go build`
2. Passes `go vet`
3. Has the same semantics (same function names, correct types)

The golden files document what the old codegen produced for comparison/debugging.

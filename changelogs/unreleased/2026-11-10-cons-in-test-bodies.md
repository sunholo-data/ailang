### Fixed — `::` in named-test bodies (#1914), and four more re-printing defects found by a round-trip test

`test "c" { 1 :: [2] == [1, 2] }` failed with `PAR_NO_PREFIX_PARSE: unexpected token in
expression: ::` although `ailang check` accepted the file. Named-test bodies are re-printed with
`PrintAILANGSource` and re-parsed; the parser stores `x :: xs` as a call of `::`, and the printer
wrote it back as `::(x, xs)`, a spelling that parses only as a pattern. It now prints `(x :: xs)`.

A new property test (`internal/testing/printer_roundtrip_test.go`) checks that
`parse(print(e))` is AST-equal to `e` for all 2,134 function bodies in `examples/` and `std/`
and for a corpus that must contain every `ast.Expr` node type. It found four more defects of the
same seam, each fixed:
- the minimum int (`0x8000000000000000`) printed as a decimal that does not parse, and other
  negative hex literals came back as a unary minus; they now print as hex;
- `func(...) -> T ! {IO} { … }` lost its effect row;
- an `if`, `let`, `match` or lambda used as an operand or a callee lost its grouping
  (`(if c then a else b) ++ ys` came back as `if c then a else (b ++ ys)`);
- a plain string containing `${` came back as an interpolation; `$` is now printed as `\$`.

`--strict-syntax` accepts `x :: xs`. It used to reject it and suggest `::(x, xs)`, which does not
parse as an expression, so a cons could not be written under strict syntax at all. `::(x, rest)`
remains valid as a pattern.

### Added — bounded fixed-decimal formatting (2026-10-23)

`std/string.formatFixed(x, decimals)` formats float64 values with precision
0..1000. Interpreter, strict bytecode and generated Go use the same strconv
semantics, including nearest-even ties, signed zero and nonfinite strings.
Invalid precision fails ordinarily in every engine. `floatToStr` is unchanged.

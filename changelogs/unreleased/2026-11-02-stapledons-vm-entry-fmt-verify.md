### Fixed — `run --bytecode` ran the wrong `main` (#2052)

`--entry main` on the VM took the first compiled function whose name ended in `.main` or `_main`,
from any loaded module. Modules compile in sorted id order, and the entry module's id is its file
path (`sim/tools/exoplanets_verify`), so an imported module's `main` (`pkg/stapledons/...`) won
whenever its id sorted first. The `_main` rule meant a local `run_main` declared before `main` won
too, with no imports at all. When the wrong function compiled natively it ran silently and exited
0; stapledons only saw it because the two mains' argument records clashed (`GET_FIELD: index 4
exceeds field count 2`). Entry lookup is now exact, `<entry module>.<name>`, on `ailang run` and in
the bytecode named-test engine. Small repros that "didn't reproduce" were package dirs that sorted
ahead of `pkg/`.

### Fixed — `ailang fmt` on a `test` block with a `let` statement

`test "t" { let p = {life: 1.0}; p.life == 1.0 }` passed `ailang check` but `fmt` failed with
`nil expression node`: test bodies were printed expression by expression, and a `let` statement is
a Let without a body. They now print as a statement sequence, with the same `;` rule as blocks.

### Fixed — `ailang verify` float literals and helpers calling `sqrt`

- Float literals reached Z3 in Go notation: `1e+09` became the unknown constant `e+09`, and a
  whole number like `1.0` became the Int numeral `1`, a sort mismatch in a Real definition. Floats
  are now positional decimals with a decimal point; NaN and Inf are refused.
- A local helper calling `std/math.sqrt` was inlined with a raw `(sqrt s)` body and Z3 errored
  with `unknown constant sqrt`. The guard that skips unencodable calls was set up after helper
  bodies were encoded; it now runs first, so the caller is SKIPPED as `unsupported_encoding`.
  Helpers with encodable bodies still verify.

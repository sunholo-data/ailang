### Changed — BREAKING: `show` is total and unambiguous; `std/string.showPreview` (2026-11-11)

`show` no longer elides. Containers wider than 80 columns used to lose their middle
(`show(range(0, 50))` was `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]`), values nested more
than three deep became `...`, and `bytes` longer than 32 were cut. Every element and every level
is now rendered. Strings **inside** lists, arrays, tuples, records, maps and constructors are
quoted and escaped: `show(["a", "b, c"])` is `["a", "b, c"]` (was `[a, b, c]`, the same text as
`show(["a", "b", "c"])`), `show(Some("x"))` is `Some("x")`, `show(["", "x"])` is `["", "x"]`
(was `[, x]`). A **top-level** string is still returned unchanged, so `"${name}"` and
`println(show(s))` print exactly as before.

- Escapes, the same set `ailang fmt` writes for literals (one shared escaper): `\"` `\\` `\n` `\t`
  `\r`, `\$` before `{`, `\u{X}` for other control and non-printing characters, `\xHH` for a
  byte that is not valid UTF-8. Printable non-ASCII text is written as is. The old byte-level
  cut could return invalid UTF-8; that is fixed.
- `bytes` render as `<bytes:HEX>` in full; an upload's filename and MIME type are no longer shown.
- Ints, floats, bools, unit, record field order (by name), map order (encoded-key order) and
  `<function>` are unchanged. `show` of a value nested a million deep works (the renderer no
  longer recurses) and costs time and memory linear in the output.
- The same spelling is now used everywhere a value is displayed: `ailang run` printing an
  entry's return value (floats there change from `1e+21` to `1000000000000000000000.0`), the
  REPL echo, `ailang test` failure and counterexample messages, trace argument/result
  rendering (credential redaction unchanged), the WASM playground (record fields are now in
  name order, not Go map order), and `ailang compile` output. The evaluator and the bytecode VM
  print byte-identical text for every shape.
- New: `std/string.showPreview(x: a, maxChars: int) -> string`. It renders `x` as `show`
  does, except that a string is quoted even at the top level (`showPreview("abc", 10)` is
  `"abc"` with the quotes), so a cut preview can never be mistaken for a complete one. If the
  rendering has at most `maxChars` code points it is returned whole; otherwise the first
  `maxChars` code points are followed by `…(+N chars)`, N being exactly how many were left out.
  It never builds the full rendering. `maxChars <= 0` keeps only the marker. Additive to the
  Stable `std/string` interface (golden refrozen).
- A quoted string reads back as the same string through the loader, which NFC-normalises
  source: a combining mark or other character that normalisation would change is written
  `\u{X}` when normalisation might change the string (it is not NFC, or has a run of more than 30
  combining marks or similar characters such as U+FF9E, where normalisation inserts U+034F) or the
  character follows an escape. NFC text in any script is written as is. The check is one linear pass
  (quoting a million marks takes well under a second). `ailang fmt` uses the
  same escaper; this also fixes fmt changing the value of literals such as `"e\u{301}"` and
  turning `"\n\u{303}"` into an invalid escape. Every tracked `.ail` file formats identically.
- A constructor used as a value (`map(Some, xs)`, `let f = Ok`) is a function on the bytecode VM
  and in `ailang compile` output too: it printed as `Some` on the VM (and `<function>` on the
  evaluator), and calling it failed on the VM and panicked in compiled programs. Compiled programs
  also quote nested strings with the same NFC rule as the interpreters (the rule and its Unicode
  tables are copied into the emitted runtime: about 95 KB of source and 110 KB of binary in every
  compiled program).
- Teaching prompt v0.16.15 (active) documents the quoting and `showPreview`.
- Benchmark `run_length_encode` is retired (moved to `benchmarks/retired/`): its expected output
  was the old lossy rendering of a `[(int, string)]`. It is replaced by `run_length_encode_v2`,
  which states its output format (`3a 2b 4c 1a`) explicitly. The new id keeps ELO and banked rows
  separable.

**Migrate.** Search for `show(` applied to a container that holds strings, and for interpolation
of such values (`"${xs}"`). If the old bare form was what you wanted, render it explicitly
(`join(", ", xs)`). If the output is parsed, hashed or compared, the new form is the one you
wanted: it is injective per type. Where a test asserts on shown text, update the expected string
(`"[a, b]"` becomes `"[\"a\", \"b\"]"`). For bounded logging use `showPreview(v, 200)`. Design:
design_docs/planned/v1_0_0/m-v1-show-total.md.

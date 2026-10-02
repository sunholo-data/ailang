### Fixed — `std/json` numbers round-trip exactly (#1460)

- `encode(jnum(-0.0))` now emits `-0.0` (was `0`), and `decode("-0")` keeps the sign bit.
- Large and tiny floats use exponent notation outside `1e-6 <= |x| < 1e21`, the same window as Go
  `encoding/json` and `JSON.stringify`: `1e21` → `1e+21` (was 22 digits), `1e308` → `1e+308` (was 309
  digits), `5e-324` → `5e-324` (was 324 digits), `1e-7` → `1e-7`. Mid-range whole floats are unchanged
  (`jnum(100.0)` is still `100`).
- Whole floats at or beyond 2^63 no longer go through an `int64` conversion. On arm64 that conversion
  saturated, so `encode(jnum(9223372036854775808.0))` printed `9223372036854775807`; output no longer
  depends on the host CPU.
- Integer literals beyond ±2^63 decode exactly instead of saturating: `10000000000000000000` is `1e19`,
  not `9223372036854775807`.
- Number literals outside float64 range (`1e400`) now decode to `Err("json: number 1e400 out of float64
  range")` instead of silently becoming Infinity (which then re-encoded as `null`). Underflow such as
  `1e-400` still decodes to `0`.
- The interpreter, the legacy evaluator encoder and the bytecode VM now share one formatter
  (`eval.FormatJSONNumber`) and one parser (`eval.ParseJSONNumber`). The legacy encoder used `%g`, so
  its output changes too: `1e10` is now `10000000000` (was `1e+10`), and NaN/Infinity are now `null`
  (it used to emit `NaN`/`+Inf`, which is not valid JSON).
- Consumers diffing number text produced by v0.50.x see different bytes, but only in the ranges listed
  above. The stapledons-godot `num` workaround is no longer needed once both sides upgrade.

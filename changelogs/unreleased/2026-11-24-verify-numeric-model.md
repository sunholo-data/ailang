### Changed — `ailang verify` reports int64 overflow and checks NaN-producing float operations (M-VERIFY-NUMERIC-MODEL)

The verifier proves `int` as an unbounded integer and `float` as a real number. It now checks the
operations where that differs from the runtime (decisions D2 a′ and D4 b′, ruled 2026-11-24):

- **int64 overflow.** Every int `+ - *` and negation in a verified body must keep in-range operands
  in range. Where it may not, the result stays `✓ VERIFIED` and carries a note (JSON
  `caveats: [{kind: "int_overflow", model}]`) with an input that wraps, e.g.
  `inc(x) = x + 1` at `x = 9223372036854775807`. The note changes no verdict or exit code; bound the
  input in `requires` to remove it. `ailang verify --int-model=int64` (also with `--package`) makes it
  a VIOLATION with reason code `int_overflow`.
- **NaN and ±Infinity.** Float `/` and `%` by a possible zero, `sqrt` of a possible negative and `log`
  of a possible non-positive are VIOLATIONS with reason code `float_nan_or_inf`. Float inputs are
  assumed to be ordinary numbers; verified float results carry caveat `float_inputs_assumed_non_nan`.
- `numeric_model` gains `int_overflow` (`reported` | `checked`) and `float_nan`; the `Scope:` line says so.
- A call-site or numeric obligation is now downgraded to inconclusive only when its own clauses
  mention an uninterpreted builtin (`sqrt(x)` with `x` possibly negative is a violation, not inconclusive).
- `make verify-gap-map` counts caveats and accepts `--int-model` (via the binary's flag).

Not checked yet: arithmetic inside HOF lambdas, recursion, helper bodies and contracts; int `/` and
`%` by zero (a runtime error, outside the partial-correctness model); `pow`; float overflow to ±Inf.

Gap map (8 corpora, 765 results), default mode: no verdict changed; 56 verified results carry an
`int_overflow` note, 5 `int_overflow_undecided`, 12 `float_inputs_assumed_non_nan`. Under
`--int-model=int64`, 61 would change (56 violations, 4 unknown, 1 skipped) — the list to bound before
the default flips. Also freezes teaching prompt v0.16.17 (superseded by v0.16.18; two mutable
versions had turned `TestRealRegistry_PostMigrationSplitCounts` red).

### Changed — Z3 verification docs and teaching prompt now match what the verifier does (2026-11-23)

Every claim about `ailang verify` in the contracts guide, the limitations pages, the teaching
prompt, the devtools prompt and the `use-ailang` skill was re-checked against v0.60.6 with one
probe file per claim. Corrected:

- **Teaching prompt v0.16.17** (new active version). v0.16.16 said floats are not verifiable
  (they verify, as reals), that recursive functions are skipped (they are unrolled, and are
  INCONCLUSIVE unless `requires` bounds the depth), and that bitwise operators are verifiable
  (they are skipped as unencodable). It now also says that `requires` is never checked at call
  sites, that `int` is proved unbounded (int64 overflow is not caught), and that callers of
  `map`/`filter`/`foldl` are skipped (#1375).
- **Contracts guide.** The states table said INCONCLUSIVE exits 0; it exits 1 (`--allow-inconclusive`
  exits 0). It no longer claims literal-lambda `map`/`filter`/`foldl` verify (#1375). The string
  table listed `a ++ b`, which is a type error on strings (use interpolation or `concat`), and
  omitted `contains`. Record fields may be lists. The "all possible inputs" tip now names what
  the encoded model leaves out. The status note lists the measured open gaps.
- **Known limitations.** New canonical entry "Static verification: what a VERIFIED result does not
  cover", with transcripts for call-site preconditions, int64 overflow, NaN, recursion, HOFs,
  bitwise operators and unannotated generic parameters (#2400). New summary rows in
  `docs/LIMITATIONS.md`.
- **Devtools prompt v0.8.1, `use-ailang` skill, stability and implementation-status pages, IFC
  guide.** The verdict list gained INCONCLUSIVE. Nested patterns (`Some(Some(x))`) and imported
  function calls do verify (the IFC guide said they could not). `--strict` wording fixed.
- **`examples/runnable/contracts/basic.ail`.** `safeDivide` and `increment` carried contracts Z3
  correctly refuted (a negative dividend; `max <= x`) in a file presented as the basic example;
  their `requires` are now complete and all four contracts verify.

### Fixed — `ailang verify` encoded float `/` and `%` as integer division (#2062, 2026-11-03)

The verifier picked an operator's SMT-LIB encoding by name only. Float `/` reached the encoder as
`DictApp{Fractional Float, div}` and got the truncating **Int** encoding (`div`, `abs`, `(>= n 0)`)
on Real terms. Float `%` mapped to Int-only `mod`, and string `<`/`>` in a dictionary application
mapped to Int `<`. Z3 either rejected the script (`sort mismatch`, `unknown constant result`) or
silently coerced it with `to_int` and **refuted true contracts**. `1.0 / (2.0*r*r*r) * …` under
`requires { r > 1.5 }` came back `VIOLATION` at `r = 7/4`, which tripped PUB006 and blocked
publishing `sunholo/relativity`.

Division, remainder and ordering now dispatch on the operand sort: the dictionary's instance type,
then literal kind or Core type info.

- Float `/` → Real `(/ n d)`.
- Float `%` → exact Go `math.Mod`: `n - d·trunc(n/d)`, sign of the dividend.
- String ordering → `str.<` / `str.<=`.
- Int encodings are byte-identical.

A sort-dependent method on an unknown instance now skips the function rather than defaulting to
Int. Float division by zero stays solver-unspecified, the same stance as Int. Every one of the 333
verdicts across the repo's 72 contract files is unchanged. Design:
`design_docs/implemented/v0_57_1/m-smt-float-div-mod-sort-dispatch.md`.

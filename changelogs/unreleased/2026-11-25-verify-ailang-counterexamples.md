### Improved — AILANG-syntax Z3 counterexamples in `ailang verify`

- `ailang verify` now formats concrete Z3 counterexample and overflow-caveat models into canonical AILANG syntax (`xs = [-5, 9223372036854775807]`, `{ age: 36, name: "Ada" }`, `Some(42)`, `None`, `-1.5`) instead of raw SMT-LIB (`(seq.unit ...)`, `(mk_Record_...)`).
- Strips internal `$p_` parameter prefixes and omits internal solver `(Array ...)` stub tables and un-evaluated symbolic `result` definitions so agents can copy counterexamples straight into `test` assertions.

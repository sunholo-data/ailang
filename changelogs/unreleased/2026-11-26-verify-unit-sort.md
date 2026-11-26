### Added — `ailang verify` encodes unit, `()`

Unit is now the SMT datatype `Tuple0`, the 0-tuple, whose one value is `()`. Before, any function whose signature mentioned `()` was skipped ("unit type cannot be encoded"), along with every caller of such a helper. That included every caller of a `Result[(), E]` helper, the usual shape of a check that succeeds with no value. A unit value is shown as `()` in a counterexample and in a witness-replay verdict, not as the AST printer's `<nil>`. Witness replay passes it to the real function.

A module that declares its own type named `Tuple0` keeps it. A function that needs unit is then skipped, and the skip names the clash.

### Fixed — bounded unrolling of a zero-argument recursive function

A zero-argument function is `f(_: ())`. When it calls itself, bounded unrolling now drops the unit argument from the recursive call, as an ordinary call does, and refers to the unrolled function by its bare name. Both were Z3 errors (`unknown constant f_0 (Tuple0)`, then `(f_2 )`). Neither could happen before, because the unit literal itself failed to encode and the function was skipped.

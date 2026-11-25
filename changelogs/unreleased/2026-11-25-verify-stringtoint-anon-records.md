### Added — `ailang verify` encodes `stringToInt` and `charCode`

- `std/string.stringToInt` is an uninterpreted function with the facts Go's `strconv.ParseInt` guarantees. Unsigned digits within int64 give `Some(n)`. No sign and not all digits (or `""`) give `None`. A signed string is left open, so a model through it is inconclusive.
- `std/string.charCode` is an uninterpreted `Int` in 0 … 0x10FFFF for a single rune. It is not Z3's `str.to_code`: Z3's characters stop at 0x2FFFF, so that would prove `charCode(c) < 200000`, which is false.
- Downstream gap map: 3 contracts newly verified, 2 move from skipped to inconclusive, no regressions.

### Fixed — anonymous records in parameters

- A record inside a parameter's list or type argument (`xs: [{ name: string, q: Question }]`) is declared. It never was, and every such function was skipped.
- A record whose field is an ADT is declared after that ADT. It was declared before it, which also caused a skip.
- Two inline records with the same field names but different field types (`[{key, desc: string}]` and `[{key, desc: Json}]`) share one SMT sort name. A function that needs that sort is now a skip that names the collision. It was a Z3 sort-mismatch ERROR.

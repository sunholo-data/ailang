# std/json number-accessor naming, and the unwritten Result-vs-Option rule

- **Date**: 2026-09-28
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `asNumber|asFloat` over design_docs/ (only v0_6_0 m-json-accessors-enabled, v0_7_4 m-stdlib-gaps, both historical); `Result.{0,20}Option` over design_docs/planned (no ruling); `naming consisten|stdlib naming|accessor naming` (none); the stdlib exports themselves (`grep '^export .*-> (Option|Result)\[' std/*.ail`: 52 Option, 69 Result); the v0.16.6 prompt; `m-v1-stability-promise.md` for rename rules

The email-parse dogfood agent (2026-09-27, the same report as `inbox_1790498369833_8034dce8`)
flagged this as minor. Two separate things are behind it. **Naming:** there is no `asFloat`. `std/json` uses JSON's
word for the type, `asNumber`/`getNumber`/`getNumberArray` returning `float`, next to
`getInt`, so a caller who thinks in AILANG types (`float`, `int`) has to guess the JSON term.
Fixing that means adding `asFloat`/`getFloat` aliases, renaming, or only teaching the mapping.
Any rename changes a public surface under the stability promise
(`design_docs/implemented/v0_29_0/m-v1-stability-promise.md`: deprecations live ≥1 minor, and
no deprecation mechanism exists yet). **Result vs Option:** the stdlib already follows a
consistent but unwritten rule. A parse or decode that can fail *with a reason* returns
`Result[_, string]`: `std/json.decode`, `std/yaml.decode`, `std/jwt.decodeJWT`,
`std/array` and `std/embedding` `decodeF64LE`/`decodeF32LE`, `std/json.decodeFloatArray`. A lookup
that can simply find *nothing* returns `Option`: every `as*`/`get*`, `std/bytes.byteAt`. The only
exported outlier found is `std/sem.decode_frame -> Option[sem_frame]`, which is also snake_case.
The internal `_embedding_decode -> Option` builtin is not exported. The prompt teaches `Option`
and `Result` separately (v0.16.6 §Option, §Result) but never states the rule, so the split reads
as arbitrary. Several acceptable remedies exist (aliases vs renames vs teach-only; fix or
grandfather `decode_frame`), and some touch public surface, so rubric rows 3 and 4 apply. The
cheap core of a doc: state the rule in the prompt and in `docs/docs/reference/stdlib.md`, decide
aliases for the number accessors, and resolve `decode_frame`. Low priority: nothing is broken.

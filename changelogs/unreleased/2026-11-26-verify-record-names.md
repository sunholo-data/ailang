### Changed — `ailang verify` tells apart inline records that share field names

An inline record was named by its field names alone. So `decisions`' `Choice(string, [{ key: string, desc: string }])` and `ChoiceR(Json, [{ key: string, desc: Json }])` both named the sort `Record_desc_key`, and every function needing it was skipped. When two records share field names but not field sorts, each is now named by its field sorts as well (`Record_desc_key__String__String`, `Record_desc_key__Json__String`). A record literal takes the sort of its checked type. Every other record keeps its name.

If two names still coincide once the field sorts are added, which takes a type named like a mangled sort (`{ a: [int] }` and `{ a: Seq_Int }`), the function is skipped and the skip names the clash.

### Fixed — two record sort bugs the collision skip hid

- An inline record in a named alias's field (`type Decision = { answers: [{ name: string, answer: Answer }], … }`) was never declared. The per-function filter walked the alias's fields but never kept the record itself, so a function using the alias was skipped with "sort … is not declared".
- A `[]` passed to a parametric constructor (`Ok([])` building a `Result[[{ name: string, score: int }], string]`) was typed `(Seq Int)`, a Z3 `! ERROR`. It now takes the constructor's field sort at the constructed type.

Downstream gap map: see the M-VERIFY-SOUNDNESS-COVERAGE program doc (item 4e).

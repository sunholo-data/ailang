### Fixed — `ailang verify` inlines helpers that take or return `Option`/`Result`

A verified function that called a helper with `Option`, `Result` or another
parametric ADT in its signature was `⚠ SKIPPED` ("calls `parsePos` whose
signature uses an unencodable type `Option[int]`"), although the same types
already verified inside a function's own body. Such a helper is now inlined
when the ADT is applied to encodable types (`Option[int]`,
`Result[float, MyErr]`, a user `Box[int]`), so the caller verifies, or gets a
counterexample, including a `requires_violation` when it passes `None` to a
helper whose `requires` rules it out.

A helper whose signature has a type variable (`Option[a]`), a tuple
(`Result[(), string]`), a function type or `bytes` still makes the caller
SKIPPED.

A helper with neither `pure` nor an effect row (`func get(…) -> Option[Json]`)
is still refused, as before; its skip hint used to describe list patterns and
now says to mark the helper `pure` or declare `! {}`.

Across the eight downstream corpora (`make verify-gap-map`), ailang-parse's
`pdfAiPagesCountFromInfo` moved from skipped to verified and nothing regressed.
The example `examples/runnable/contracts/unencodable_callee_skip.ail`, which
showed the skip with an `Option[float]` callee, now uses a tuple callee.

### Fixed — `--policy` can admit `Declassify` and `Rand` (#1557)

`allowed_caps` validated names against the effect op registry rather than the language's
canonical effect set. `Declassify` and `Rand` have no entries in that registry, so both were
rejected as `unknown capability` in every security mode, and a program declaring either could
never run under `--policy`, `policy-tool` or the `ailang_only` lane. This blocked the
`prompt_injection` benchmark for AILANG World. Unknown-capability checks now use the canonical
set (`types.KnownEffectNames`). Restricted mode admits `Declassify`, which has no runtime
operations, alongside the `Rand` it already listed. Effects without a confined adapter, such
as `DB`, `Async` and `SharedMem`, are still refused in restricted mode with the `trusted_host`
migration.

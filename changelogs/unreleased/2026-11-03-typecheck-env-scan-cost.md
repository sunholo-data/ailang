### Changed — type checking is 2–3× cheaper on large packages

`inferApp` snapshot the variables owned by the enclosing scope on every function
application by walking the whole type environment — every enclosing frame plus the
~930-binding builtin/prelude frame — three times over. Environment frames now memoise
their free type, row and effect-row variable sets (`internal/types/env_cache.go`),
invalidated by the only in-place writers (`BindScheme`/`BindType`); callers read the
shared sets. One extra application now visits ~70 environment bindings instead of
~2,810, independent of module size.

Cold `ailang check --package .` (darwin/arm64, instructions retired):
`sunholo/ailang_parse` 1.59×10¹² → 0.52×10¹² (−67%, peak RSS 234 → 204 MB);
`sunholo/crew_core` 0.6.0 1.34×10¹² → 0.70×10¹² (−48%, 228 → 178 MB). This is what
keeps large packages under the registry validator's 200 s check budget.

Equivalence: every query is recomputed uncached and compared in all `internal/types`
and `internal/pipeline` tests (`types.SetEnvVarCrossCheck`), and inferred interfaces
are unchanged across `std/`, `examples/` and both packages. New guard:
`TestEnvScan_WorkPerApplication` fails if per-application environment work exceeds
500 bindings or grows with declarations or imports.

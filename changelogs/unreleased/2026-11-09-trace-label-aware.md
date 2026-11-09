### Fixed — security: IFC-labelled values no longer reach execution traces (M-TRACE-LABEL-AWARE) (2026-11-09)

`std/secret` documented that a `<secret>` value is kept out of traces; the runtime did not
honour it. The tracer runs below the type system, so labels never reached it: a program that
printed 6 characters of a `string<secret>` wrote the full value **4 times** into a `deep`
trace, and an effect argument (a `writeFile` body) carried it verbatim at the **default
`standard`** tier. A 30-copy reproduction of the reporter's OAuth shape (labelled token
through `postFormBody` → `json.encode/decode` → `getString` → `substring`, a tail-recursive
loop, a closure and `map`) now records **0** copies at every tier.

How: the static IFC pass reports every call that receives, captures or returns a labelled
value (`types.CheckModuleIFCWithTraceSites`); the pipeline stamps the label on the final Core
`App` (`core.App.TraceRedact`, compile-cache key `v8` → `v9`); the evaluator opens a label
scope on the trace collector for the call's dynamic extent, inside which function args/results,
effect args/results and OTEL effect-span enrichment record `<redacted:LABEL>` — values are not
rendered at all. Names, arity and the span tree are kept. Redacting the whole extent closes the
higher-order and polymorphic cases (a label-transparent helper runs inside its caller's scope)
at the cost of over-redacting unlabelled values computed inside a labelled call.

Not covered yet, and documented in `std/secret`: a labelled value returned by a function in
another *user* module (the per-module IFC gap, `m-ifc-cross-module-labels`). The bytecode VM
does not carry the marks, so a traced labelled program falls back from `--bytecode` to the
evaluator and `--strict-bytecode` refuses it. Canary tests:
`internal/runner/trace_label_redaction_test.go`. Example:
`examples/runnable/secrets/trace_redaction.ail`.

package elaborate

// Warning is the common interface for all non-blocking compile-time warnings
// surfaced through the pipeline (pipeline.Result.Warnings).
//
// Historically the pipeline only carried exhaustiveness warnings
// (*ExhaustivenessWarning). This interface generalizes the warning surface so
// additional diagnostic passes (e.g. the split-argument-order DX warning in
// internal/pipeline/warn_split_args.go) can contribute warnings without the
// pipeline needing to know their concrete types. Render sites only rely on
// String(), so they remain unchanged.
//
// *ExhaustivenessWarning already satisfies this interface via its String()
// method.
type Warning interface {
	String() string
}

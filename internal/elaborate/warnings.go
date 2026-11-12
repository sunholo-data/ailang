package elaborate

// Warning is the common interface for all non-blocking compile-time warnings
// surfaced through the pipeline (pipeline.Result.Warnings): the split-argument
// order warning, TC_MATCH_003 (types.MatchWarning), TC_MATCH_005 and the
// other diagnostic passes. Render sites rely only on String(); machine output
// reads the optional Code/Position/WarningFile/Suggestion/Message methods
// (cmd/ailang/check_warnings.go).
type Warning interface {
	String() string
}

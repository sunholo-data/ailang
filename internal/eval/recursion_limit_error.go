package eval

import "fmt"

// RecursionLimitError is RT_REC_003: the AILANG call depth passed
// --max-recursion-depth.
//
// It is a type, not a formatted string, so that code which wraps callback
// errors can recognise it and pass it through unchanged. Recursion that re-enters
// the evaluator through a builtin callback (map, foldl, sortBy...) fails at the
// innermost call; wrapping it once per level buried this one line under 50,000
// "callback error at index 0:" prefixes (1.9 MB of stderr, 21 s to build) (#1317).
type RecursionLimitError struct {
	Limit int
}

func (e *RecursionLimitError) Error() string {
	return fmt.Sprintf("RT_REC_003: max recursion depth %d exceeded. Try a smaller input, an iterative std/list helper such as foldl or map instead of hand-rolled recursion, or raise the ceiling with --max-recursion-depth", e.Limit)
}

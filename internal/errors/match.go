package errors

import "strconv"

// Run-time match failure (M-V1-MATCH-EXHAUSTIVENESS M0).
//
// A `match` none of whose arms matched the scrutinee. Once the type checker
// rejects non-exhaustive matches this is defence in depth: it catches checker
// bugs and paths that bypass the check. Both engines raise this one type, so
// the evaluator and the bytecode VM report the same code and text, as RT001
// does for division by zero (arith.go).

// MatchFailureError is RT002: no arm of a match matched its scrutinee.
//
// It describes the scrutinee by SHAPE only — a constructor tag, "empty list",
// "list of 3 elements", a type name — and never by payload: a scrutinee can be
// an IFC-labelled secret, and this text reaches logs and traces.
type MatchFailureError struct {
	// Pos is "file:line:col" where the match starts; the VM gives
	// "file:line:col" too when the lowering knew the column. Empty if unknown.
	Pos string
	// Ctor is the scrutinee's constructor tag when it is an ADT value.
	Ctor string
	// Shape describes a non-ADT scrutinee ("empty list", "int"); ignored
	// when Ctor is set.
	Shape string
}

func (e *MatchFailureError) Error() string {
	msg := RT002 + ": no arm matched in match"
	if e.Pos != "" {
		msg += " at " + e.Pos
	}
	switch {
	case e.Ctor != "":
		msg += " (scrutinee constructor: " + e.Ctor + ")"
	case e.Shape != "":
		msg += " (scrutinee: " + e.Shape + ")"
	}
	return msg
}

// Code returns the registry code, RT002.
func (e *MatchFailureError) Code() string { return RT002 }

// ListShape describes a list scrutinee of n elements without its contents.
func ListShape(n int) string {
	switch n {
	case 0:
		return "empty list"
	case 1:
		return "list of 1 element"
	}
	return "list of " + strconv.Itoa(n) + " elements"
}

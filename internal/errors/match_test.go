package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestMatchFailureErrorText(t *testing.T) {
	cases := []struct {
		err  *MatchFailureError
		want string
	}{
		{&MatchFailureError{Pos: "p1.ail:3:38", Ctor: "Blue"}, "RT002: no arm matched in match at p1.ail:3:38 (scrutinee constructor: Blue)"},
		{&MatchFailureError{Pos: "p1.ail:3", Shape: ListShape(0)}, "RT002: no arm matched in match at p1.ail:3 (scrutinee: empty list)"},
		{&MatchFailureError{Shape: ListShape(1)}, "RT002: no arm matched in match (scrutinee: list of 1 element)"},
		{&MatchFailureError{Shape: ListShape(3)}, "RT002: no arm matched in match (scrutinee: list of 3 elements)"},
		// Ctor wins over Shape; neither carries a payload.
		{&MatchFailureError{Ctor: "Some", Shape: "int"}, "RT002: no arm matched in match (scrutinee constructor: Some)"},
		{&MatchFailureError{}, "RT002: no arm matched in match"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
		if c.err.Code() != RT002 {
			t.Errorf("Code() = %q, want RT002", c.err.Code())
		}
	}
	// Reachable through wrapping, as the VM frame and evaluator wrap it.
	var mf *MatchFailureError
	if !errors.As(fmt.Errorf("execution failed: %w", cases[0].err), &mf) || mf.Ctor != "Blue" {
		t.Fatal("errors.As did not find *MatchFailureError through %w")
	}
}

func TestUnreachableError(t *testing.T) {
	e := &UnreachableError{Msg: "callers pass a non-empty list"}
	if e.Code() != RT010 || e.Error() != "RT010: unreachable code reached: callers pass a non-empty list" {
		t.Fatalf("got %q (%s)", e.Error(), e.Code())
	}
	if _, ok := GetErrorInfo(RT010); !ok {
		t.Fatal("RT010 is not registered")
	}
}

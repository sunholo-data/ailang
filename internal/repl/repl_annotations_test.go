package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// #2238 review: the prompt's expression path passes the elaborator's
// annotations to the checker, as both pipelines do. It used to pass none, so
// `let f: int -> int = \x. 0 in f("s")` printed `0 :: Int`.
func TestREPLEnforcesAnnotations(t *testing.T) {
	r := New()
	r.initBuiltins()
	r.importModule("std/prelude", io.Discard)
	for _, c := range []struct{ src, want string }{
		{`let f: int -> int = \x. 0 in f("s")`, "Type error"},
		{`let f = func(x: int) -> int { 0 } in f("s")`, "Type error"},
		{`let f = func(x: int) -> int { "s" } in f(1)`, "Type error"},
		{`let f = func(x: int) -> int { x + 1 } in f(41)`, "42 ::"},
		{`let f: int -> int = \x. x * 2 in f(21)`, "42 ::"},
	} {
		var out bytes.Buffer
		r.ProcessExpression(c.src, &out)
		if !strings.Contains(out.String(), c.want) {
			t.Errorf("%s printed %q, want it to contain %q", c.src, out.String(), c.want)
		}
	}
}

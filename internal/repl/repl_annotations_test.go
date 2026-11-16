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

// #2238 second review: a literal's `! {}` and its effect-contract parameter
// annotations hold at the prompt as in a file. The `! {}` literal applied on
// the spot used to print "e" and `1 :: Int` (probes/z/r_imm.ail is rejected in
// a file); the a02 shape needs the literal marks (SetLiteralLambdas), the
// `! {}` rows need the effect annotations (SetEffectAnnotationsFull) and the
// declared-lambda lookup passed to ValidateEffects.
func TestREPLEnforcesLiteralEffectAnnotations(t *testing.T) {
	r := New()
	r.initBuiltins()
	r.importModule("std/prelude", io.Discard)
	r.importModule("std/io", io.Discard)
	for _, c := range []struct{ src, want, never string }{
		{`(func(x: int) -> int ! {} { println("e"); x })(1)`, "uses effects not declared in its ! {} annotation", "1 ::"},
		{`let f = func(x: int) -> int ! {} { println("e"); x } in f(1)`, "Effect checking error", "1 ::"},
		{`(func(x: int) -> int ! {} { x + 1 })(1)`, "2 ::", "rror"},
		{`let ap = \g. g(41, \n. ()) in ap(func(x: int, k: int -> () ! {IO}) -> int { x + 1 })`, "Type error", "42 ::"},
		// #2295's shape: a literal hands its `! {IO}`-contract callback to an
		// effect-polymorphic callee. Only the literal path (marks → equality)
		// keeps the annotation's compatibility tail from leaking into the row.
		// The outer `! {IO}` literal is what gets charged (a prompt has no
		// enclosing signature; a prompt-level `let` of an effectful literal is
		// checked as a declaration without one, so it is rejected anyway).
		{`let poly = func(n: int, cb: int -> () ! {e}) -> int ! {e} { cb(n); n } in (func(u: int) -> int ! {IO} { let step = func(n: int, cb: int -> () ! {IO}) -> int ! {IO} { poly(n, cb) }; step(1, \x. ()) })(0)`, "1 ::", "rror"},
	} {
		var out bytes.Buffer
		r.ProcessExpression(c.src, &out)
		got := out.String()
		if !strings.Contains(got, c.want) || strings.Contains(got, c.never) {
			t.Errorf("%s printed %q, want it to contain %q and not %q", c.src, got, c.want, c.never)
		}
	}
}

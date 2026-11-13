package types_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2273 review round 1: the cases an independent review found. Each row pins
// one of: what is rejected, where, and that the message can be acted on as
// written (no inference variables, no internal names, no contradicting hint).
func TestRigidSignature_ReviewCases(t *testing.T) {
	for _, c := range []struct {
		name, decls string
		kind        string   // "" = must be accepted
		line        int      // 0 = any
		want, not   []string // substrings the message must / must not contain
	}{
		{name: "a literal inside an inner let is not defaulted away",
			decls: "pure func succ[a](x: a) -> a {\n  let s = show(x + 1);\n  x\n}",
			kind:  "literal", line: 4},
		{name: "arithmetic and show inside inner lets keep `a` general",
			decls: "pure func f[a](x: a) -> a {\n  let y = x + x;\n  let s = show(y);\n  y\n}"},
		{name: "a type an import exports, not imported by name, is still a variable",
			decls: "import std/sem (has_embedding)\npure func f[b](x: b, y: namespace) -> b = y",
			kind:  "equal", want: []string{"`b` and `namespace` are independent"}},
		{name: "a typo hint that contradicts the body is not offered",
			decls: "import std/option (Option, Some)\npure func parse(s: string) -> Option[resutl] = Some(toI(s))\npure func toI(s: string) -> int = 3",
			kind:  "concrete", want: []string{"requires `resutl` to be `int`"}, not: []string{"Did you mean", "Result"}},
		{name: "an open record is suggested with one named rest",
			decls: "pure func step(s: state) -> state = {s | n: s.n + 1}",
			kind:  "concrete", want: []string{"write `{n: int | r}` instead of `state`"}, not: []string{"..."}},
		{name: "a type the body fixes only in shape is not offered for pasting",
			decls: "pure func compose[a, b](f: a, g: b) -> b = \\x. g(f(x))",
			kind:  "concrete", want: []string{"the form `(_) -> _`"}, not: []string{"α", "ρ"}},
		{name: "an ANF temporary is not named",
			decls: "pure func useInt(n: int) -> int = n * 2\npure func f1[a](x: a, xs: [a]) -> [a] {\n  let g = \\y. \\z. useInt(z);\n  let k = g(1)(x);\n  xs\n}",
			kind:  "concrete", want: []string{"(function application)"}, not: []string{"$tmp"}},
		{name: "parameter names reused as type variables get no parameter-name note",
			decls: "pure func cmp(a: a, b: b) -> int = if a < b then 1 else 0",
			kind:  "equal", not: []string{"name of a parameter"}},
		{name: "a call within a recursive group is named, with its location",
			decls: "pure func f[a](x: a, n: int) -> a = if n <= 0 then x else { let _k = g(n - 1); x }\npure func g(n: int) -> int = f(n, n)",
			kind:  "concrete", line: 4, want: []string{"recursive group (`f`, `g`)", "function application of f"}},
		{name: "a builtin's variables are its own (#2305)",
			decls: "pure func both[a, b](xs: [a], ys: [b]) -> int = _list_length(xs) + _list_length(ys)"},
		{name: "a builtin use does not pin another declaration's variable",
			decls: "pure func f[a](x: a, ys) -> a {\n  let n = _list_length(ys);\n  x\n}\npure func g[b](x: b, ys) -> (b, int) = (x, _list_length(ys))\npure func use() -> int = f(1, [\"s\"])"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkRigid(t, c.decls)
			if c.kind == "" {
				if err != nil {
					t.Fatalf("rejected a correct program: %v", err)
				}
				return
			}
			var re *types.RigidSigError
			if !errors.As(err, &re) {
				t.Fatalf("want TC_RIGID_001, got %v", err)
			}
			if re.Kind() != c.kind {
				t.Errorf("kind = %q, want %q (%v)", re.Kind(), c.kind, err)
			}
			if c.line != 0 && re.Pos.Line != c.line {
				t.Errorf("line = %d, want %d (%v)", re.Pos.Line, c.line, err)
			}
			text := err.Error()
			for _, w := range c.want {
				if !strings.Contains(text, w) {
					t.Errorf("message lacks %q:\n%v", w, err)
				}
			}
			for _, w := range c.not {
				if strings.Contains(text, w) {
					t.Errorf("message contains %q:\n%v", w, err)
				}
			}
		})
	}
}

// A constructor choice decided late (#2228's deferred ctorChoice, resolved
// inside SolveConstraints) is part of the substitution the rule reads: a
// signature variable the declaration pins through it is still TC_RIGID_001.
// When nothing in the declaration decides it, a rigid scrutinee leaves the
// choice open, which is TC_AMBIGUOUS_CTOR_001 (also a rejection).
func TestRigidSignature_DeferredConstructorChoice(t *testing.T) {
	err := checkModules(t, withMain("pure func h(y: UA) -> int = 0\npure func g[a](x: a) -> int = match x { C(n) => n, _ => 0 } + h(x)\nexport pure func run() -> int = g(mkb())\n"))
	var re *types.RigidSigError
	if !errors.As(err, &re) {
		t.Fatalf("decided in the declaration: want TC_RIGID_001, got %v", err)
	}
	err = checkModules(t, withMain("pure func g[a](x: a) -> int = match x { C(n) => n, _ => 0 }\nexport pure func run() -> int = g(mkb())\n"))
	if err == nil || !strings.Contains(err.Error(), types.TC_AMBIGUOUS_CTOR_001) {
		t.Fatalf("undecided: want TC_AMBIGUOUS_CTOR_001, got %v", err)
	}
}

// No diagnostic shows a rigid variable's internal name (`a@{f}`), whichever
// check reports it.
func TestRigidSignature_NoInternalNamesInErrors(t *testing.T) {
	for _, decls := range []string{
		"pure func f[a](x: a) -> [a] = [x, [x]]",
		"pure func ren(p: {name: string | r}, q: {name: string | s}) -> {name: string | r} = q",
		"pure func g[a](x: a) -> a = x ++ [1]",
	} {
		err := checkRigid(t, decls)
		if err == nil {
			t.Fatalf("accepted: %s", decls)
		}
		if strings.Contains(err.Error(), "@{") {
			t.Errorf("internal name in: %v", err)
		}
	}
}

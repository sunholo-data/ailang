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
		// #2286: a lowercase name is a type by scope. What is still a variable is rigid.
		{name: "g3: a type an import exports, not imported by name, is a rigid variable",
			decls: "import std/sem (has_embedding)\npure func f[b](x: b, y: namespace) -> b = y",
			kind:  "equal", want: []string{"std/sem exports a type 'namespace', which is not imported by name"}},
		{name: "g6: the same name used at a concrete type",
			decls: "import std/sem (has_embedding)\npure func g(x: namespace) -> int = x + 1",
			kind:  "concrete", want: []string{"requires `namespace` to be `int`", "not imported by name"}},
		{name: "g8: a value imported by name, used as a type, is a rigid variable",
			decls: "import std/list (map)\npure func f(x: map) -> int = x",
			kind:  "concrete", want: []string{"'map' is treated as a type variable here; it is imported by name from std/list, but as a value"}},
		{name: "g9: ... and is independent of the signature's other variables",
			decls: "import std/list (map)\npure func pick[a](x: a, y: map) -> a = y",
			kind:  "equal", want: []string{"`a` and `map` are independent", "imported by name from std/list"}},
		{name: "a declared lowercase type is a type, not a variable",
			decls: "import std/string (length)\ntype namespace = string\npure func v(ns: namespace) -> int = length(ns)"},
		// #2289: one rule for annotations inside a declaration.
		{name: "a literal's annotation naming the enclosing variable is that rigid variable",
			decls: "pure func fa[a](x: a) -> int {\n  let g = func(y: a) -> int { y + 0 };\n  g(x)\n}",
			kind:  "concrete", line: 4},
		{name: "a literal's own new variable is flexible (an equality, like a let annotation)",
			decls: "pure func fb(n: int) -> int {\n  let g = func(y: t) -> t { y + 1 };\n  g(n)\n}\npure func fc() -> string {\n  let id = func(y: t) -> t { y };\n  \"${show(id(1))}${id(\"s\")}\"\n}"},
		{name: "a literal keeping the enclosing variable general",
			decls: "pure func fd[a](x: a) -> a {\n  let g = func(y: a) -> a { y };\n  g(x)\n}"},
		// #2288: record rows are bound by unification now, so `| r` is enforced.
		{name: "a field the signature's open record does not list",
			decls: "pure func nick(p: {name: string | r}) -> string = p.nick",
			kind:  "concrete", want: []string{"uses the field(s) `nick`", "add `nick` to the record type"}},
		{name: "a closed record returned for an open one",
			decls: "pure func reset(p: {name: string | r}) -> {name: string | r} = {name: \"x\"}",
			kind:  "concrete", want: []string{"no further fields"}},
		{name: "two open rows made one",
			decls: "pure func ren(p: {name: string | r}, q: {name: string | s}) -> {name: string | r} = q",
			kind:  "equal", want: []string{"`| r` and `| s`"}},
		{name: "an open record passed through keeps its row",
			decls: "pure func pass[a](p: {v: a | r}) -> {v: a | r} = p"},
		// Review round 2: an anonymous `...` is a flexible wildcard, as on dev;
		// only a named row is rigid.
		{name: "g22: anonymous rows in parameter and result are one flexible row",
			decls: "pure func f(r: {x: int, ...}) -> {x: int, ...} = {r | x: r.x + 1}\npure func use() -> string = f({x: 1, y: \"a\"}).y"},
		{name: "g22b: the named twin keeps its row",
			decls: "pure func f(r: {x: int | s}) -> {x: int | s} = {r | x: r.x + 1}\npure func use() -> string = f({x: 1, y: \"a\"}).y"},
		{name: "c08: an anonymous row nested in a named one",
			decls: "pure func f(p: {y: int | r}, q: {x: {k: int, ...} | r}) -> {x: {k: int, ...} | r} = q\npure func use() -> string = f({y: 1, x: {k: 1, m: \"a\"}}, {x: {k: 2, m: \"b\"}}).x.m"},
		// #2284: field access is a deferred constraint, settled at the same
		// boundary where the rule runs.
		{name: "a rigid variable constrained only by a deferred field access",
			decls: "pure func getN[a](p: a) -> int = p.n",
			kind:  "concrete", want: []string{"requires `a` to be `{n: int | r}`"}},
		{name: "a rigid variable a match pins to a newtype",
			decls: "type W = W({n: int, m: string})\npure func viaW[a](p: a) -> int {\n  let k = p.n;\n  match p { W(r) => r.n + k }\n}",
			kind:  "concrete", want: []string{"requires `a` to be `W`"}},
		{name: "a rigid row read through a let-bound copy",
			decls: "pure func carried(p: {n: int | r}) -> string {\n  let q = p;\n  q.m\n}",
			kind:  "concrete", want: []string{"uses the field(s) `m`"}},
		// Message: a literal reusing the enclosing variable's name is told so.
		{name: "a literal's annotation reusing the enclosing variable is pointed at",
			decls: "pure func f[a](x: a) -> int { let g = func(y: a) -> a { y + 1 }; g(3) }",
			kind:  "concrete", want: []string{"is also written in an annotation inside `f`"}},
		// Message: #2286's note about a declared lowercase type is not attached
		// to a TC_RIGID_001 about other names.
		{name: "no unrelated declared-type note",
			decls: "import std/list (map)\ntype point = {x: int, y: int}\npure func mixbad(p: point, e: elem, m: map) -> elem = m",
			kind:  "equal", not: []string{"'point'", "_r0", "_r1"}},
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

// #2283's contract check binds the parameters at the declaration's rigid
// types (InferPredicateRigid), so a clause cannot specialise them; clauses that
// keep them general still check.
func TestRigidSignature_Contracts(t *testing.T) {
	for _, c := range []struct {
		name, decls string
		reject      bool
	}{
		{"requires uses a literal at a", "pure func ident2[a](x: a) -> a\n  requires { x + 1 > 0 }\n{ x }", true},
		{"ensures compares a with a string", "pure func ident3[a](x: a) -> a\n  ensures { x == \"s\" || true }\n{ x }", true},
		{"ensures x == x where the body needs Eq a", "pure func same[a](x: a, y: a) -> bool\n  ensures { x == x }\n{ x == y }", false},
		// A clause that needs only a class on `a` (no concrete type) is accepted,
		// as on dev: there is no syntax to declare the constraint, and callers
		// are not held to it (the gap is tracked in #2337).
		{"ensures result == x where the body needs nothing", "pure func ident[a](x: a) -> a\n  ensures { result == x }\n{ x }", false},
		{"ensures with Ord the body does not need", "pure func pick[a](x: a, y: a) -> a\n  ensures { result <= x || result >= x }\n{ y }", false},
		{"ensures == on the body's numeric result", "pure func add[a](x: a, y: a) -> a\n  ensures { result == x + y }\n{ x + y }", false},
		{"requires makes a equal to b", "pure func two[a, b](x: a, y: b) -> a\n  requires { x == y }\n{ x }", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkRigid(t, c.decls)
			if !c.reject {
				if err != nil {
					t.Fatalf("rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "TC_CONTRACT_001") || !strings.Contains(err.Error(), types.TC_RIGID_001) {
				t.Fatalf("want TC_CONTRACT_001 carrying TC_RIGID_001, got %v", err)
			}
			if !strings.Contains(err.Error(), "the clause") {
				t.Errorf("the message should blame the clause: %v", err)
			}
		})
	}
}

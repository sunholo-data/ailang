package pipeline

import (
	"strings"
	"testing"
)

// PR #2209 review: `let` must not generalise a variable that an enclosing
// parameter still owns. Generalisation compares "variables of the type being
// generalised" with "variables free in the environment", and the two used to be
// separate walks. The environment walk had no case for records (closed or
// open), record rows or type applications, and it read the environment without
// the current substitution. Each program below is ill-typed; each type-checked
// on the PR head and crashed at run time ("missing dictionary method:
// prelude::Num::String::add", "_str_join: expected string, got int").
// The open-record forms were newly accepted by the PR; the closed-record,
// Option and lambda-parameter forms were already accepted on dev.
var letGeneralisationUnsound = []struct {
	name, src, want string
}{
	{"open record, `...`", `module m

export func f(p: {name: a, ...}) -> int {
  let q = p;
  q.name + 1
}

export func main() -> () ! {IO} = println(show(f({name: "s", z: 1})))
`, "cannot unify"},
	{"open record, `| r`", `module m

export func f(p: {name: a | r}) -> int {
  let q = p;
  q.name + 1
}

export func main() -> () ! {IO} = println(show(f({name: "s"})))
`, "cannot unify"},
	{"open record, update in a lambda", `module m

export func f(p: {name: a | r}) -> int {
  let g = \u. {p | name: p.name};
  g(1).name + 1
}

export func main() -> () ! {IO} = println(show(f({name: "s"})))
`, "cannot unify"},
	{"unannotated parameter, record update", `module m
import std/string (concat)

export func main() -> () ! {IO} {
  let f = \p. let g = \u. {p | name: p.name} in concat([show(g(1).name + 1), g(2).name]);
  println(f({name: 5}))
}
`, "cannot unify"},
	{"closed record", `module m

export func f(p: {name: a}) -> int {
  let q = p;
  q.name + 1
}

export func main() -> () ! {IO} = println(show(f({name: "s"})))
`, "cannot unify"},
	{"closed record, update in a lambda", `module m

export func f(p: {name: a, age: int}) -> int {
  let g = \u. {p | age: 2};
  g(1).name + 1
}

export func main() -> () ! {IO} = println(show(f({name: "s", age: 1})))
`, "cannot unify"},
	{"type application", `module m

export func f(p: Option[a]) -> int {
  let q = p;
  match q { Some(x) => x + 1, None => 0 }
}

export func main() -> () ! {IO} = println(show(f(Some("s"))))
`, "cannot unify"},
	{"unannotated parameter, closed record built from its field", `module m
import std/string (concat)

export func main() -> () ! {IO} {
  let f = \p. let g = \u. {name: p.name} in concat([show(g(1).name + 1), g(2).name]);
  println(f({name: 5}))
}
`, "string"}, // #2239: g carries the access to p.name, so each use re-links its
	// field type to p's — the conflict is reported at the field ("string vs
	// int") rather than as Num[string]; either way the program is rejected.
}

func TestLetGeneralisation_EnclosingParameterVariablesStayMonomorphic(t *testing.T) {
	for _, tc := range letGeneralisationUnsound {
		t.Run(tc.name, func(t *testing.T) {
			_, err := writeAndCheck(t, map[string]string{"m.ail": tc.src}, "m.ail")
			if err == nil {
				t.Fatal("ill-typed program type-checked: an inner let generalised a variable owned by the enclosing parameter")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("rejected for the wrong reason (want %q): %v", tc.want, err)
			}
		})
	}
}

// The withholding must not cost legitimate let-polymorphism. Inside functions
// whose parameter is an open record: a let-bound function over open records is
// used at two field types, a let-bound function is used at two argument types
// while capturing the parameter, and a let-bound record of functions (a value,
// so generalisable) is used at two types.
func TestLetGeneralisation_LegitimatePolymorphismOverRecordsSurvives(t *testing.T) {
	cases := map[string]string{
		"function over open records at two field types": `module m
import std/string (concat)

export func h(p: {name: string, ...}) -> string {
  let getN = \r. r.name;
  let n = getN({name: 5, y: true});
  concat([getN({name: "a", x: 1}), show(n + 1), p.name])
}

export func k(q) -> string {
  let getN = \r. r.name;
  concat([getN({name: "a", x: 1}), show(getN({name: 5}) + 1), q.title])
}
`,
		"function capturing the parameter, two argument types": `module m
import std/string (concat)

export func f(p: {name: string, ...}) -> string {
  let wrap = \x. {value: x, tag: p.name};
  let a = wrap(1);
  let b = wrap("s");
  concat([show(a.value + 1), b.value, a.tag])
}
`,
		"record value of functions at two types": `module m
import std/string (concat)

export func g(p: {name: string, ...}) -> string {
  let ops = {id: \x. x, label: \u. p.name};
  concat([show(ops.id(1) + 1), ops.id("s"), ops.label(0)])
}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := writeAndCheck(t, map[string]string{"m.ail": src}, "m.ail"); err != nil {
				t.Fatalf("valid let-polymorphism rejected: %v", err)
			}
		})
	}
}

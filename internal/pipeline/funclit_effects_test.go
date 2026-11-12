package pipeline

import (
	"strings"
	"testing"
)

// #2238 review, effect side, in the package that validates effects: a
// literal's own row variables are charged where it is called (not to the
// function around it), a func declaration's are checked against its
// signature, and a literal's explicit `! {}` is checked wherever the literal
// sits (records, lists, tuples, updates, operands, match arms).
func TestFuncLitEffects_Validation(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"literal declaring ! {e}", "export func main() -> () ! {IO} {\n  let run = func(g: int -> () ! {e}) -> () ! {e} { g(1) };\n  run(\\n. println(show(n)))\n}\n", ""},
		{"literal taking ! {e} with no annotation of its own", "export func main() -> () ! {IO} {\n  let run = func(g: int -> () ! {e}, n: int) -> () { g(n) };\n  run(\\n. println(show(n)), 7)\n}\n", ""},
		{"module-level literal", "let run = func(g: int -> () ! {e}) -> () ! {e} { g(1) }\nexport func main() -> () ! {IO} = run(\\n. println(show(n)))\n", ""},
		{"a pure func calling an ! {e} callback is still rejected", "func runTwice(f: () -> int ! {e}) -> int = f() + f()\nexport func main() -> () ! {IO} = println(show(runTwice(\\u. 1)))\n", "runTwice"},
		// A row the literal does not own (outer's e) is still charged to the
		// enclosing function, even when the literal is only returned. Whether
		// building a closure should charge at all is #2322; this pins today's
		// rule (the top-level one) so dropping every tail cannot slip in.
		{"a returned literal's foreign row is charged to the enclosing function", "func outer(cb: int -> () ! {e}) -> (int -> () ! {e}) {\n  \\n. cb(n)\n}\nexport func main() -> () ! {IO} = outer(\\n. println(show(n)))(1)\n", "Unresolved effect tail: e"},
		{"the same with the row declared", "func outer(cb: int -> () ! {e}) -> (int -> () ! {e}) ! {e} {\n  \\n. cb(n)\n}\nexport func main() -> () ! {IO} = outer(\\n. println(show(n)))(1)\n", ""},
		{"! {} literal in a record", "type W = { f: (int) -> int }\nexport func main() -> () ! {IO} {\n  let w: W = { f: func(n: int) -> int ! {} { println(\"x\"); n } };\n  println(show(w.f(1)))\n}\n", "declared in its ! {} annotation"},
		{"! {} literal in a list", "export func main() -> () ! {IO} {\n  let fs = [func(n: int) -> int ! {} { println(\"x\"); n }];\n  println(show(length(fs)))\n}\n", "uses effects not declared"},
		{"! {} literal in a tuple", "export func main() -> () ! {IO} {\n  let p = (1, func(n: int) -> int ! {} { println(\"x\"); n });\n  println(\"ok\")\n}\n", "uses effects not declared"},
		{"! {} literal in a record update", "type W = { f: (int) -> int, k: int }\nexport func main() -> () ! {IO} {\n  let w: W = { f: \\n. n, k: 1 };\n  let w2 = { w | f: func(n: int) -> int ! {} { println(\"x\"); n } };\n  println(show(w2.k))\n}\n", "uses effects not declared"},
		{"! {} literal in a match arm", "export func main() -> () ! {IO} {\n  let f = match 1 { 1 => func(n: int) -> int ! {} { println(\"x\"); n }, _ => \\n. n };\n  println(show(f(1)))\n}\n", "uses effects not declared"},
		{"! {} literal that is pure", "export func main() -> () ! {IO} {\n  let w = { f: func(n: int) -> int ! {} { n + 1 } };\n  println(show(w.f(1)))\n}\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkSource(t, "m", "module m\nimport std/list (length)\n"+c.src)
			if c.want == "" {
				if err != nil {
					t.Fatalf("must check: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
		})
	}
}

package types_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2273: a type variable written in a signature is rigid in the body. Seen
// through the module pipeline (the elaborator decides which variables are
// rigid; the type checker enforces them at the binding's generalization).

func checkRigid(t *testing.T, decls string) error {
	t.Helper()
	return checkModules(t, map[string]string{
		"m/main.ail": "module m/main\n\n" + decls + "\n\nexport func main() -> () ! {IO} = println(\"ok\")\n",
	})
}

func TestRigidSignature_Rejects(t *testing.T) {
	for _, c := range []struct {
		name, decls string
		kind        string   // the violation's kind (report data)
		want        []string // substrings of the message
	}{
		{"issue: toInt[a] returns its argument as int",
			"pure func toInt[a](x: a) -> int = x", "concrete",
			[]string{"`toInt` says `a` is any type", "requires `a` to be `int`", "return type annotation"}},
		{"issue: f[a] returns a literal",
			"pure func f[a](x: a) -> a = 1", "literal", []string{"integer literal"}},
		{"issue: bump[a] adds a literal",
			"pure func bump[a](x: a) -> a = x + 1", "literal", []string{"`bump`", "`int` or `float`"}},
		{"issue: implicit variable elem",
			"pure func g(x: elem) -> elem = x + 1", "literal", []string{"`elem`"}},
		{"two variables made one",
			"pure func pick[a, b](x: a, y: b) -> a = y", "equal", []string{"`a` and `b` are independent"}},
		{"miscased record type name",
			"type State = { n: int }\npure func step(s: state) -> int = s.n + 1", "concrete",
			[]string{"Did you mean the type `State`?", "field access"}},
		{"miscased ADT name",
			"type Shape = Circle(int) | Square(int)\npure func mk(x: int) -> shape = Circle(x)", "concrete",
			[]string{"Did you mean the type `Shape`?"}},
		{"misspelled primitive",
			"pure func twice(x: strng) -> strng = \"${x}${x}\"", "concrete",
			[]string{"Did you mean the type `string`?"}},
		{"a parameter name used as a type",
			"pure func loop(n: int, stack: [int]) -> (stack, int) = (stack, n)", "concrete",
			[]string{"`stack` is the name of a parameter of `loop`, not a type", "`[int]`"}},
		{"parameter names as types, compared",
			"pure func cmp(a: a, b: b) -> int = if a < b then 1 else 0", "equal",
			[]string{"`a` and `b` are independent", "comparison"}},
		{"curried callback with swapped variables",
			"pure func fold(f: a -> b -> b, acc: b, xs: [a]) -> b = match xs { [] => acc, x :: rest => fold(f, f(acc, x), rest) }",
			"equal", []string{"`a` and `b`"}},
		{"element used as a number",
			"pure func firstOr[a](xs: [a], d: a) -> a = match xs { [] => d, h :: _ => h + 0 }", "literal", nil},
		{"comparison with a literal",
			"pure func pos[a](x: a) -> bool = x > 0", "literal", nil},
		{"let annotation in the body pins the variable",
			"pure func local[a](x: a) -> a { let y: int = x; x }", "concrete", []string{"let annotation y"}},
		{"call of a monomorphic function",
			"pure func id2(y: int) -> int = y\npure func g2[a](x: a) -> int = id2(x)", "concrete", []string{"function application of id2"}},
		{"a type parameter called as a function",
			"pure func app[f](g: f, x: int) -> int = g(x)", "concrete",
			[]string{"write `(int) -> int` instead of `f`"}},
		{"recursion at another type",
			"pure func rec[a](x: a, n: int) -> int = if n == 0 then 0 else rec(n, n - 1)", "concrete", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkRigid(t, c.decls)
			if err == nil {
				t.Fatalf("accepted; want TC_RIGID_001")
			}
			var re *types.RigidSigError
			if !errors.As(err, &re) {
				t.Fatalf("want *types.RigidSigError, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), types.TC_RIGID_001) {
				t.Errorf("error does not carry %s: %v", types.TC_RIGID_001, err)
			}
			if got := re.Kind(); got != c.kind {
				t.Errorf("kind = %q, want %q (%v)", got, c.kind, err)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("message lacks %q:\n%v", w, err)
				}
			}
			if re.Pos.Line == 0 {
				t.Errorf("no source position: %v", err)
			}
			if re.Suggestion == "" {
				t.Errorf("no suggestion: %v", err)
			}
		})
	}
}

// Negative controls: correct signatures, and the forms that are not rigid.
func TestRigidSignature_Accepts(t *testing.T) {
	for _, c := range []struct{ name, decls string }{
		{"identity", "pure func ident[a](x: a) -> a = x"},
		{"constraint inferred: +", "pure func add[a](x: a, y: a) -> a = x + y"},
		{"constraint inferred: show", "pure func sh[a](x: a) -> string = show(x)"},
		{"constraint inferred: <", "pure func lt[a](x: a, y: a) -> bool = x < y"},
		{"constraint inferred: ==", "pure func eqq[a](x: a, y: a) -> bool = x == y"},
		{"float literal: Fractional is inferred", "pure func half[a](x: a) -> a = x / 2.0"},
		{"recursion at the same type", "pure func len[a](xs: [a]) -> int = match xs { [] => 0, _ :: t => 1 + len(t) }"},
		{"higher-order", "pure func mapp[a, b](f: (a) -> b, xs: [a]) -> [b] = match xs { [] => [], h :: t => f(h) :: mapp(f, t) }"},
		{"tuples", "pure func swap[a, b](p: (a, b)) -> (b, a) = match p { (x, y) => (y, x) }"},
		{"implicit variables", "pure func k(x: acc, y: elem) -> acc = x"},
		{"open record", "pure func name(p: {name: string | r}) -> string = p.name"},
		{"let annotation names the signature variable", "pure func loc[a](x: a) -> a { let y: a = x; y }"},
		{"partially annotated", "pure func partial[a](x: a, y) -> a = x"},
		{"unannotated declaration is inferred as before", "pure func noann(x, y) = x + y + 1"},
		{"local lambda", "pure func usesLocal[a](x: a) -> a { let f = \\y. y; f(x) }"},
		{"function literal annotated with the signature variable", "pure func lam[a](x: a) -> a = (func(y: a) -> a { y })(x)"},
		{"mutual recursion at the same types", "pure func isEv[a](n: int, t: a) -> a = if n == 0 then t else isOd(n - 1, t)\npure func isOd[b](n: int, t: b) -> b = if n == 0 then t else isEv(n - 1, t)"},
		{"explicit parameter used only as an effect row", "func app[a, e](f: (a) -> a ! {e}, x: a) -> a ! {e} = f(x)"},
		{"lowercase declared type is not a variable (std/sem's namespace)", "import std/string (length)\ntype namespace = string\npure func v(ns: namespace) -> int = length(ns)"},
		{"used at two types by another declaration", "pure func ident[a](x: a) -> a = x\npure func both() -> string = \"${show(ident(1))}${ident(\"s\")}\""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := checkRigid(t, c.decls); err != nil {
				t.Fatalf("rejected a correct program: %v", err)
			}
		})
	}
}

// The exported scheme is the written one, under the written names.
func TestRigidSignature_ImporterSeesDeclaredScheme(t *testing.T) {
	err := checkModules(t, map[string]string{
		"m/lib.ail": "module m/lib\nexport pure func pair[a, b](x: a, y: b) -> (a, b) = (x, y)\n",
		"m/main.ail": "module m/main\nimport m/lib (pair)\n" +
			"export func main() -> () ! {IO} = match pair(1, \"s\") { (n, s) => println(\"${show(n + 1)}${s}\") }\n",
	})
	if err != nil {
		t.Fatalf("importer rejected: %v", err)
	}
}

func TestSuggestTypeForVar(t *testing.T) {
	cands := []string{"State", "Result", "string", "int", "Option", "TodoItem"}
	dist := func(a, b string) int { // test-local Levenshtein
		prev := make([]int, len(b)+1)
		for j := range prev {
			prev[j] = j
		}
		for i := 1; i <= len(a); i++ {
			cur := make([]int, len(b)+1)
			cur[0] = i
			for j := 1; j <= len(b); j++ {
				cost := 1
				if a[i-1] == b[j-1] {
					cost = 0
				}
				cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			}
			prev = cur
		}
		return prev[len(b)]
	}
	for name, want := range map[string]string{
		"state": "State", "result": "Result", "todo_item": "TodoItem", "strng": "string",
		"optoin": "Option", "a": "", "acc": "", "elem": "", "zzzz": "",
	} {
		if got := types.SuggestTypeForVar(name, cands, dist); got != want {
			t.Errorf("SuggestTypeForVar(%q) = %q, want %q", name, got, want)
		}
	}
}

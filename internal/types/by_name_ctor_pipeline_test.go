package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2303/#2353: a constructor matched by name — neither it nor its type is
// imported (m/w loads Color and Light, the main module imports neither) — is
// typed by the value matched, seen through the pipeline so the deferred
// by-name choice (by_name_ctor.go, ambiguous_ctor.go) runs in this package.
func TestByNameCtorChoice_Pipeline(t *testing.T) {
	files := func(body string) map[string]string {
		return map[string]string{
			"m/x.ail": "module m/x\nexport type Color = Red | Green | Blue | Shade(int)\n" +
				"export pure func mkc(i: int) -> Color = if i == 0 then Red else if i == 1 then Green else Blue\n",
			"m/y.ail": "module m/y\nexport type Light = Green | Red\nexport pure func mkl(i: int) -> Light = if i == 0 then Green else Red\n",
			"m/w.ail": "module m/w\nimport m/x (mkc)\nimport m/y (mkl)\nimport std/option (Option, Some, None)\nimport std/result (Result, Ok, Err)\n" +
				"export pure func cc(i: int) = mkc(i)\nexport pure func ll(i: int) = mkl(i)\n" +
				"export pure func oc(i: int) -> Option[Color] = if i < 0 then None else Some(mkc(i))\n" +
				"export pure func oi(i: int) -> Option[int] = if i > 0 then Some(i) else None\n" +
				"export pure func os(i: int) -> Option[string] = if i > 0 then Some(\"s\") else None\n" +
				"export pure func ro(i: int) -> Result[int, string] = if i > 0 then Ok(i) else Err(\"neg\")\n",
			"m/main.ail": "module m/main\nimport std/list (nth)\nimport m/w (cc, ll, oc, oi, os, ro)\n" + body,
		}
	}
	for _, c := range []struct {
		name, body string
		code, want string // "" = must check
	}{
		{"by-name on a typed value", "export pure func f(i: int) -> int = match cc(i) { Red => 1, _ => 0 }\n", "", ""},
		{"nested by-name constructors on a typed value (#2353)",
			"export pure func f(i: int) -> int = match oc(i) { Some(Red) => 1, Some(Shade(n)) => n, Some(_) => 2, None => 3 }\n", "", ""},
		{"a bound variable has its real type (#2353)",
			"export pure func f(i: int) -> int = match oc(i) { Some(Shade(n)) => n + 1, _ => 0 }\n", "", ""},
		{"a bound variable used at the wrong type (#2353)",
			"export pure func f(xs: [int]) -> string = match nth(xs, 0) { Some(v) => v, None => \"\" }\n", "", "cannot unify"},
		{"a let-bound lambda decided by its one use",
			"export pure func f(i: int) -> int {\n  let k = \\v. match v { Green => 10, _ => 20 };\n  k(cc(i))\n}\n", "", ""},
		{"a let-bound lambda used at two types",
			"export pure func f(i: int) -> int {\n  let k = \\v. match v { Green => 10, _ => 20 };\n  k(cc(i)) + k(ll(i))\n}\n",
			types.TC_AMBIGUOUS_CTOR_001, "cannot be generic"},
		// A by-name constructor stays a deferred choice even with ONE
		// candidate type: deciding it from the candidates would make the
		// verdict depend on what else the import closure declares (#2362
		// review). So a helper over it is monomorphic (known limitation for
		// by-name Option/Result in library modules; the message never says
		// "which one" for a single candidate) ...
		{"a one-candidate helper at two type arguments",
			"export pure func f(i: int) -> bool {\n  let isSome = \\o. match o { Some(_) => true, _ => false };\n  isSome(oi(i)) && isSome(os(i))\n}\n",
			types.TC_AMBIGUOUS_CTOR_001, "(std/option's Option) is matched by name and not imported, so the type of the value it matches is decided by the uses of `isSome`"},
		{"a one-candidate helper at one type", "export pure func f(i: int) -> bool {\n  let isSome = \\o. match o { Some(_) => true, _ => false };\n  isSome(oi(i)) && isSome(oi(i + 1))\n}\n", "", ""},
		// ... and a contract clause's let is an inner let, decided by its use.
		{"a let-bound helper in a requires clause",
			"pure func f(r: Result[int, string]) -> int\nrequires { let g = \\q. match q { Ok(_) => true, _ => false } in g(r) }\n{ 1 }\nexport pure func h() -> int = f(ro(1))\n", "", ""},
		{"a let-bound helper in an ensures clause",
			"pure func f(n: int) -> Result[int, string]\nensures { let g = \\q. match q { Ok(v) => v > 0, _ => true } in g(result) }\n{ ro(n) }\n", "", ""},
		{"a two-candidate helper in a contract used at two types",
			"pure func f(n: int) -> int\nrequires { let k = \\v. match v { Green => true, _ => false } in k(cc(n)) || k(ll(n)) }\n{ 1 }\n",
			types.TC_AMBIGUOUS_CTOR_001, "cannot be generic"},
		{"a record value", "export pure func f(r: {a: int}) -> int = match r { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "against a value of type { a: int }, which is not built from constructors here"},
		{"a list value", "export pure func f(r: [int]) -> int = match r { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "against a value of type [int], which is not built from constructors here"},
		{"a function value", "export pure func f(r: (int) -> int) -> int = match r { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "against a value of type int -> int, which is not built from constructors here"},
		{"a signature type variable", "export pure func f[a](x: a) -> int = match x { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "a is a type variable of the signature"},
		{"an unannotated top-level parameter", "export pure func f(x) -> int = match x { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "cannot tell which type Red belongs to"},
		// Green is declared by two loaded types, so the early foreign-
		// constructor check (#1478) cannot name one; the value is an Option.
		{"a typed value without that constructor", "export pure func f(i: int) -> int = match oc(i) { Green => 1, _ => 0 }\n",
			types.TC_MATCH_007, "which has no constructor Green"},
		{"a typed value without that constructor (one candidate)", "export pure func f(i: int) -> int = match ll(i) { Blue => 1, _ => 0 }\n",
			"", "belongs to ADT 'Color', not 'Light'"},
		{"a value no constructor builds", "export pure func f(i: int) -> int = match i { Red => 1, _ => 0 }\n",
			types.TC_MATCH_007, "is not built from constructors"},
		{"wrong arity against the type matched", "export pure func f(i: int) -> int = match cc(i) { Shade(a, b) => a, _ => 0 }\n",
			types.TC_MATCH_006, "Shade has 2 argument(s), but Shade takes 1"},
		{"an arity no candidate has, decided by a use",
			"export pure func f(i: int) -> int {\n  let k = \\v. match v { Red(a) => a, _ => 0 };\n  k(cc(i))\n}\n",
			types.TC_MATCH_006, "Red has 1 argument(s), but Red takes 0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkModules(t, files(c.body))
			switch {
			case c.want == "" && err != nil:
				t.Fatalf("must check: %v", err)
			case c.want != "" && err == nil:
				t.Fatalf("must fail with %s %q", c.code, c.want)
			case c.want != "" && (!strings.Contains(err.Error(), c.code) || !strings.Contains(err.Error(), c.want)):
				t.Fatalf("want %s %q, got %v", c.code, c.want, err)
			}
		})
	}
}

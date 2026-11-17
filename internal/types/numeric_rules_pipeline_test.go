package types_test

import (
	"strings"
	"testing"
)

// The #2378 rules seen through the pipeline (numeric_variants.go,
// numeric_diagnostics.go, literal_restriction.go), so the type checker's
// decisions and messages are exercised in this package.

func wantNumericCheck(t *testing.T, name string, files map[string]string, want, notWant []string) {
	t.Helper()
	err := checkModules(t, files)
	if len(want) == 0 {
		if err != nil {
			t.Errorf("%s: must check, got %v", name, err)
		}
		return
	}
	if err == nil {
		t.Errorf("%s: must fail with %q, but checks", name, want)
		return
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("%s: error lacks %q:\n%v", name, w, err)
		}
	}
	for _, w := range notWant {
		if strings.Contains(err.Error(), w) {
			t.Errorf("%s: error must not say %q:\n%v", name, w, err)
		}
	}
}

// A variable is defaulted to int only when it carries an integer literal AND
// occurs in no parameter position of any function type in the binding's type.
// Literal-free variables and variables a function in the result takes stay
// generic, as before the rule (#2393 review R1).
func TestNumericRules_WhatIsGeneric(t *testing.T) {
	lib := "module m/l\n" +
		"export func add(a, b) = a + b\n" +
		"export func mul(a, b) = a * b\n" +
		"export func getOps() = (add, mul)\n" +
		"export func pick(b) = if b then Some(add) else None\n" +
		"export func mk() = \\x. x + 1\n" +
		"export func K() = 3\n"
	main := func(body string) map[string]string {
		return map[string]string{"m/l.ail": lib, "m/main.ail": "module m/main\nimport m/l (add, mul, getOps, pick, mk, K)\n" +
			"let ops = (add, mul)\nlet fns = [add, mul]\nlet rec0 = {plus: add, times: mul}\n" + body}
	}
	wantNumericCheck(t, "literal-free containers and a lambda in the result stay generic", main(
		"export func main() -> string {\n"+
			"  let a = match ops { (f, g) => f(1.5, 2.0) + intToFloat(g(2, 3)) };\n"+
			"  let b = match getOps() { (f, g) => f(1.5, 2.0) + intToFloat(g(2, 3)) };\n"+
			"  let c = match pick(true) { Some(f) => f(1.5, 2.0), None => 0.0 };\n"+
			"  let d = rec0.plus(1.5, 2.0) + intToFloat(rec0.times(2, 3));\n"+
			"  show(a + b + c + d + mk()(2.5) + intToFloat(mk()(2) + K()))\n}\n"), nil, nil)
	wantNumericCheck(t, "a constant's literal is an int, and the error names it", map[string]string{
		"m/main.ail": "module m/main\nlet N = 3\nexport func main() -> string = show(N * 0.5)\n"},
		[]string{"`N` is an int: its integer literal (at m/main.ail:2:9)", "write that literal as a float (3.0)"}, nil)
	wantNumericCheck(t, "a result-only literal is an int", map[string]string{
		"m/main.ail": "module m/main\nfunc one(x) = 1\nexport func main() -> string = show(one(2.5) * 0.5)\n"},
		[]string{"`one` is an int: its integer literal (at m/main.ail:2:15)"}, nil)
	wantNumericCheck(t, "no literal, no literal note", map[string]string{
		"m/main.ail": "module m/main\nfunc k(n: int) -> int = n\nexport func main() -> string = show(k(2) * 0.5)\n"},
		[]string{"Fractional[int]"}, []string{"integer literal (at", "single value"})
}

// A let-bound function is copied per use like a top-level one, its own
// literal included; an expansive value is one value, and two calls of it at
// int and float are reported as exactly that (#2393 review R3).
func TestNumericRules_LetBoundBindings(t *testing.T) {
	lib := "module m/l\nexport func add(a, b) = a + b\nexport func inc(x) = x + 1\n"
	for _, c := range []struct {
		name, body    string
		want, notWant []string
	}{
		{"own literal, copied per use", "export func main() -> string {\n  let f = \\x. x + 1;\n  let n: int = 2;\n  \"${show(f(n))} ${show(f(2.5))}\"\n}\n", nil, nil},
		{"block ending in a lambda", "export func main() -> string {\n  let g = { let k = 1; \\u. inc(u) };\n  let n: int = 2;\n  \"${show(g(n))} ${show(g(2.5))}\"\n}\n", nil, nil},
		{"recursive, own literal", "export func main() -> string {\n  letrec s = \\n. if n <= 0 then 0 else n + s(n - 1);\n  let n: int = 3;\n  \"${show(s(n))} ${show(s(2.5))}\"\n}\n", nil, nil},
		{"imported literal-free callee: generic", "export func main() -> string {\n  let h = if true then \\u. add(u, u) else \\u. u;\n  \"${show(h(2))} ${show(h(2.5))}\"\n}\n", nil, nil},
		{"expansive value at int and float", "export func main() -> string {\n  let h = if true then \\u. inc(u) else \\u. u;\n  let n: int = 2;\n  \"${show(h(n))} ${show(h(2.5))}\"\n}\n",
			[]string{"`h` is a single value, not a function", "m/main.ail:4:3", "need different ones (int and float)"}, nil},
		{"expansive value, one call wrong by itself", "export func main() -> string {\n  let h = if true then \\u. inc(u) else \\u. u;\n  let n: int = 2;\n  show(h(n) * 0.5)\n}\n",
			[]string{"Fractional[int]"}, []string{"single value"}},
	} {
		wantNumericCheck(t, c.name, map[string]string{"m/l.ail": lib, "m/main.ail": "module m/main\nimport m/l (add, inc)\n" + c.body}, c.want, c.notWant)
	}
}

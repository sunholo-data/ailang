package main

import "testing"

// A packed Array[float] (built by make/fromList/literal) must read exactly like
// the boxed store it replaced: show, ==, toList, get, set, append.
func TestStdArrayPackedFloatParity(t *testing.T) {
	const imports = `import std/array (make, fromList, toList, get, set, append, length)`
	cases := []struct{ name, body, want string }{
		{"show fromList", `show(fromList([1.0, 2.5, -3.0]))`, "#[1.0, 2.5, -3.0]"},
		{"show make", `show(make(3, 0.5))`, "#[0.5, 0.5, 0.5]"},
		{"toList", `show(toList(fromList([1.0, 2.0])))`, "[1.0, 2.0]"},
		{"get", `show(get(make(4, 1.5), 3))`, "1.5"},
		{"set", `show(set(make(3, 0.0), 1, 7.0))`, "#[0.0, 7.0, 0.0]"},
		{"set leaves original", `{ let a = make(2, 1.0); let b = set(a, 0, 2.0); show(a) }`, "#[1.0, 1.0]"},
		{"append", `show(append(fromList([1.0]), 2.0))`, "#[1.0, 2.0]"},
		{"eq make vs fromList", `show(toList(make(2, 1.0)) == toList(fromList([1.0, 1.0])))`, "true"},
		{"neq", `show(toList(make(2, 1.0)) == [1.0, 2.0])`, "false"},
		{"length 100k", `show(length(make(100000, 2.0)))`, "100000"},
		{"ints unaffected", `show(set(fromList([1, 2]), 0, 5))`, "#[5, 2]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for backend, got := range runArrayCase(t, arrayProgram(imports, c.body), false) {
				if got != c.want {
					t.Errorf("%s: got %q, want %q", backend, got, c.want)
				}
			}
		})
	}
}

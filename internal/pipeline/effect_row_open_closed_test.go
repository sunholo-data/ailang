package pipeline

import (
	"strings"
	"testing"
)

func TestEffectRowOpenClosedAnnotatedCallbacks(t *testing.T) {
	for _, tt := range []struct {
		name, src string
		reject    bool
	}{
		{"map", `pure func via(g: (int) -> int ! {IO}, xs: [int]) -> [int] = map(g, xs)`, true},
		{"filter", `pure func via(g: (int) -> bool ! {IO}, xs: [int]) -> [int] = filter(g, xs)`, true},
		{"any", `pure func via(g: (int) -> bool ! {IO}, xs: [int]) -> bool = any(g, xs)`, true},
		{"foldl", `pure func via(g: (int, int) -> int ! {IO}, xs: [int]) -> int = foldl(g, 0, xs)`, true},
		{"record field", `type Hook = { run: (int) -> int ! {IO} }
pure func via(h: Hook, xs: [int]) -> [int] = map(h.run, xs)`, true},
		{"mixed row", `pure func via[e](g: (int) -> int ! {IO, e}, xs: [int]) -> [int] = map(g, xs)`, true},
		{"direct call", `pure func via(g: (int) -> int ! {IO}, x: int) -> int = g(x)`, true},
		{"named effectful callback", `func loud(x: int) -> int ! {IO} { println("leak"); x }
pure func via(xs: [int]) -> [int] = map(loud, xs)`, true},
		{"option map", `pure func via(g: (int) -> int ! {IO}, x: Option[int]) -> Option[int] = O.map(g, x)`, true},
		{"pure mapE rejected", `pure func via(xs: [int]) -> [int] = mapE(\x. { println(show(x)); x }, xs)`, true},
		{"any IO lambda in IO caller", `func via(xs: [int]) -> bool ! {IO} = any(\x. { println(show(x)); true }, xs)`, false},
		{"pure map", `pure func via(xs: [int]) -> [int] = map(\x. x * 2, xs)`, false},
		{"mapE lambda", `func via(xs: [int]) -> [int] ! {IO} = mapE(\x. { println(show(x)); x }, xs)`, false},
		{"mapE rowpoly", `func via[e](g: (int) -> int ! {e}, xs: [int]) -> [int] ! {e} = mapE(g, xs)`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := "module row_guard\nimport std/list (map, filter, any, foldl, mapE)\nimport std/io (println)\nimport std/option as O\nimport std/option (Option)\n" + tt.src + "\n"
			err := checkModuleSource(t, "row_guard", src)
			if !tt.reject {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("must reject hidden IO: %v", err)
			}
			if strings.Contains(err.Error(), "internal invariant") {
				t.Fatalf("must give an effect diagnostic: %v", err)
			}
		})
	}
}

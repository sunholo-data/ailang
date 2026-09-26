package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1318: std/list.sortBy, take, zip, concat and flatMap were `[x] ++ recursive-call`
// bodies — one AILANG frame per element — so they failed RT_REC_003 at the DEFAULT
// recursion limit (10,000) on ordinary inputs, the very error whose message tells
// users to switch to a std/list helper. Each row runs over 50,001 elements at the
// default limit, on both backends; a regression to a per-element recursive body
// fails here with RT_REC_003. The list is built with split so the test program
// itself never recurses.
//
// The stability row pins that sortBy keeps equal elements in input order: the merge
// sort it replaced did (its merge took the left element on cmp <= 0), and the Go
// codegen helper used unstable sort.Slice until the same change. Its comparator is
// a two-parameter lambda: a curried `\p. \q.` one cannot be over-applied by the VM
// (CallClosure requires exact arity) and falls back to the evaluator, before and
// after this change.
func TestStdListCombinatorsAtDefaultDepth(t *testing.T) {
	stablePairsLiteral, stablePairsWant := stablePairs(60)
	cases := []struct {
		name, body, want string
	}{
		{"sortBy", `{ let s = sortBy(compare, xs()) in "${show(length(s))}:${head_or(s, "?")}" }`, "50001:"},
		{"take", `show(length(take(40000, xs())))`, "40000"},
		{"zip", `show(length(zip(xs(), xs())))`, "50001"},
		{"concat", `show(length(concat(xs(), xs())))`, "100002"},
		{"flatMap", `show(length(flatMap(\x. [x, x], xs())))`, "100002"},
		{"sortBy stable", `join(",", map(\p. match p { (_, v) => show(v) }, sortBy(\p q. match p { (a, _) => match q { (b, _) => a - b } }, pairs())))`, stablePairsWant},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "listdepth.ail")
			prog := fmt.Sprintf(`module test/listdepth

import std/list (sortBy, take, zip, concat, flatMap, length, head_or, map)
import std/string (split, repeat, compare, join)

func xs() -> [string] = split(repeat("b,a,", 25000), ",")
func pairs() -> [(int, int)] = %s

export func main() -> string = %s
`, stablePairsLiteral, c.body)
			if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AILANG_NO_CACHE", "1")
			for _, backend := range []struct {
				name string
				args []string
			}{
				{"evaluator", []string{"run", "--relax-modules", src}},
				{"vm", []string{"run", "--bytecode", "--relax-modules", src}},
			} {
				stdout, stderr, code := runCLI(t, backend.args...)
				if code != 0 {
					t.Fatalf("%s: exit %d\nstderr=%s", backend.name, code, stderr)
				}
				// A VM run that silently fell back to the evaluator would pass on
				// the evaluator's answer: the callback builtins need VM-native
				// HOF entries, and the bridge rejects closures.
				if backend.name == "vm" && (!strings.Contains(stderr, "via bytecode VM") || strings.Contains(stderr, "falling back to evaluator")) {
					t.Fatalf("vm: program did not run on the VM\nstderr=%s", stderr)
				}
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				if got := lines[len(lines)-1]; got != c.want {
					t.Errorf("%s: %s = %q, want %q", backend.name, c.name, got, c.want)
				}
			}
		})
	}
}

// stablePairs returns an AILANG literal of n (key, index) pairs with keys i%3 in
// input order, and the index order a stable sort by key must produce. n is kept
// well above 12: below that Go's sorts fall back to insertion sort, which is stable
// anyway, so a small fixture cannot tell sort.SliceStable from sort.Slice.
func stablePairs(n int) (literal, want string) {
	items := make([]string, n)
	var order []string
	for i := 0; i < n; i++ {
		items[i] = fmt.Sprintf("(%d, %d)", (n-i)%3, i)
	}
	for key := 0; key < 3; key++ {
		for i := 0; i < n; i++ {
			if (n-i)%3 == key {
				order = append(order, fmt.Sprint(i))
			}
		}
	}
	return "[" + strings.Join(items, ", ") + "]", strings.Join(order, ",")
}

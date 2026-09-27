package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-NUMERICS-QUICK: range, the std/embedding vector functions and the std/json
// array extractors each run over 50,000 elements at the DEFAULT recursion limit,
// on the evaluator and the bytecode VM. Before the change there was no range, and
// std/embedding.dot/add_vectors/euclidean_distance and std/json.filterNumbers/
// filterStrings/allNumbers/allStrings recursed once per element (RT_REC_003 at
// 10,000); add/subtract and the json builders were also quadratic.
//
// The small rows pin the semantics the AILANG bodies had, including how
// mismatched lengths behave, because the rewrite promised callers only a speed
// change. axpy is new and strict on length.
func TestStdNumericAtDefaultDepth(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"range", `show(length(range(0, 50000)))`, "50000"},
		{"range values", `show(range(2, 6))`, "[2, 3, 4, 5]"},
		{"range empty", `show(length(range(5, 5)) + length(range(5, 1)))`, "0"},
		{"dot", `show(dot(ones(), ones()))`, "50000.0"},
		{"dot common prefix", `show(dot([1.0, 2.0, 3.0], [4.0, 5.0]))`, "14.0"},
		{"add_vectors", `show(length(add_vectors(ones(), ones())))`, "50000"},
		{"add keeps longer tail", `show(add_vectors([1.0], [10.0, 20.0]))`, "[11.0, 20.0]"},
		{"euclidean_distance", `show(euclidean_distance(ones(), scale(0.0, ones())))`, "223.60679774997897"},
		{"subtract drops b's tail", `show(euclidean_distance([3.0], [0.0, 4.0]))`, "3.0"},
		{"axpy", `show(axpy(0.5, [2.0, 4.0], [1.0, 1.0]))`, "[2.0, 3.0]"},
		{"allNumbers", `match allNumbers(map(\x. JNumber(x), ones())) { Some(ns) => show(length(ns)), None => "none" }`, "50000"},
		{"allNumbers strict", `match allNumbers([JNumber(1.0), JString("x")]) { Some(_) => "some", None => "none" }`, "none"},
		{"filterStrings", `show(length(filterStrings(map(\x. JString("s"), ones()))))`, "50000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "numeric.ail")
			prog := fmt.Sprintf(`module test/numeric

import std/list (range, length, map)
import std/embedding (dot, add_vectors, scale, axpy, euclidean_distance)
import std/json (Json, JNumber, JString, allNumbers, filterStrings)

func ones() -> [float] = map(\i. 1.0, range(0, 50000))

export func main() -> string = %s
`, c.body)
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
					t.Fatalf("%s: exit %d\nstderr=%.800s", backend.name, code, stderr)
				}
				if backend.name == "vm" && (!strings.Contains(stderr, "via bytecode VM") || strings.Contains(stderr, "falling back to evaluator")) {
					t.Fatalf("vm: program did not run on the VM\nstderr=%.800s", stderr)
				}
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				if got := lines[len(lines)-1]; got != c.want {
					t.Errorf("%s: %s = %q, want %q", backend.name, c.name, got, c.want)
				}
			}
		})
	}
}

// axpy refuses mismatched lengths instead of truncating a gradient update.
func TestStdNumericAxpyRejectsMismatchedLengths(t *testing.T) {
	src := filepath.Join(t.TempDir(), "axpy.ail")
	prog := `module test/axpy

import std/embedding (axpy)

export func main() -> string = show(axpy(1.0, [1.0, 2.0], [1.0]))
`
	if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_NO_CACHE", "1")
	_, stderr, code := runCLI(t, "run", "--relax-modules", src)
	if code == 0 || !strings.Contains(stderr, "axpy: vector lengths differ (x has 2, y has 1)") {
		t.Fatalf("want a length-mismatch error, got exit %d\nstderr=%.600s", code, stderr)
	}
}

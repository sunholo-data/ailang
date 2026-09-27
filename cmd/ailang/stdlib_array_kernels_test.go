package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// arrayKernelImports is the std/array surface M3 (float kernels) and M5 (bulk
// updates) added, plus the constructors the cases need.
const arrayKernelImports = `import std/array (make, fromList, toList, get, length, dot, axpy, scale, add, sub, mul, sum, argmax, updateMany, scatterAdd)`

// arrayKernelCases is shared with the VM no-fallback test (M4).
var arrayKernelCases = []struct{ name, body, want string }{
	{"dot", `show(dot(fromList([1.0, 2.0, 3.0]), fromList([4.0, 5.0, 6.0])))`, "32.0"},
	{"dot 100k", `show(dot(make(100000, 1.0), make(100000, 2.0)))`, "200000.0"},
	{"axpy", `show(axpy(0.5, fromList([2.0, 4.0]), fromList([1.0, 1.0])))`, "#[2.0, 3.0]"},
	{"scale", `show(scale(2.0, fromList([1.5, -1.0])))`, "#[3.0, -2.0]"},
	{"add", `show(add(fromList([1.0, 2.0]), fromList([10.0, 20.0])))`, "#[11.0, 22.0]"},
	{"sub", `show(sub(fromList([1.0, 2.0]), fromList([10.0, 20.0])))`, "#[-9.0, -18.0]"},
	{"mul", `show(mul(fromList([1.0, 2.0]), fromList([10.0, 20.0])))`, "#[10.0, 40.0]"},
	{"sum", `show(sum(fromList([0.5, 0.25, 0.25])))`, "1.0"},
	{"sum empty", `show(sum(fromList([])))`, "0.0"},
	{"argmax", `show(argmax(fromList([1.0, 7.0, 3.0, 7.0])))`, "1"},
	{"argmax skips NaN", `show(argmax(fromList([0.0 / 0.0, 2.0, 1.0])))`, "1"},
	{"no input mutation", `{ let x = fromList([1.0, 2.0]); let y = axpy(1.0, x, x); show(x) }`, "#[1.0, 2.0]"},
	{"updateMany", `show(updateMany(make(3, 0.0), [(0, 1.0), (2, 5.0)]))`, "#[1.0, 0.0, 5.0]"},
	{"updateMany later wins", `show(updateMany(make(2, 0.0), [(1, 1.0), (1, 2.0)]))`, "#[0.0, 2.0]"},
	{"updateMany any type", `show(updateMany(fromList(["a", "b"]), [(1, "z")]))`, `#[a, z]`},
	{"scatterAdd", `show(scatterAdd(make(3, 0.0), [0, 0, 2], [1.0, 2.0, 5.0]))`, "#[3.0, 0.0, 5.0]"},
}

func TestStdArrayFloatKernels(t *testing.T) {
	for _, c := range arrayKernelCases {
		t.Run(c.name, func(t *testing.T) {
			for backend, got := range runArrayCase(t, arrayProgram(arrayKernelImports, c.body), false) {
				if got != c.want {
					t.Errorf("%s: got %q, want %q", backend, got, c.want)
				}
			}
		})
	}
}

// Strictness: every two-array kernel refuses a length mismatch, argmax refuses
// an empty array, and the bulk updates refuse an out-of-bounds index.
func TestStdArrayKernelErrors(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"dot lengths", `show(dot(make(2, 1.0), make(3, 1.0)))`, "dot: array lengths differ (2 and 3)"},
		{"add lengths", `show(add(make(2, 1.0), make(1, 1.0)))`, "add: array lengths differ (2 and 1)"},
		{"axpy lengths", `show(axpy(1.0, make(2, 1.0), make(1, 1.0)))`, "axpy: array lengths differ (2 and 1)"},
		{"argmax empty", `show(argmax(fromList([])))`, "argmax: empty array has no largest element"},
		{"updateMany oob", `show(updateMany(make(2, 0.0), [(2, 1.0)]))`, "updateMany: index 2 out of bounds (array length: 2)"},
		{"scatterAdd oob", `show(scatterAdd(make(2, 0.0), [-1], [1.0]))`, "scatterAdd: index -1 out of bounds (array length: 2)"},
		{"scatterAdd lengths", `show(scatterAdd(make(2, 0.0), [0, 1], [1.0]))`, "scatterAdd: 2 indices but 1 values"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "err.ail")
			if err := os.WriteFile(src, []byte(arrayProgram(arrayKernelImports, c.body)), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AILANG_NO_CACHE", "1")
			_, stderr, code := runCLI(t, "run", "--relax-modules", src)
			if code == 0 || !strings.Contains(stderr, c.want) {
				t.Fatalf("want error %q, got exit %d\nstderr=%.600s", c.want, code, stderr)
			}
		})
	}
}

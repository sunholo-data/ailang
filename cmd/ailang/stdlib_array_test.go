package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-NUMERICS-VEC-ARRAY-INGEST: std/array and the float codecs, end to end on
// the evaluator and the bytecode VM.

// arrayProgram wraps body as main() in a module importing the array, embedding
// and bytes surfaces the tests use.
func arrayProgram(imports, body string) string {
	return "module test/arr\n\n" + imports + "\n\nexport func main() -> string = " + body + "\n"
}

// runArrayCase runs src on both backends and returns the last stdout line of
// each. When vmNative is set, the VM must run the program itself (no fallback
// to the evaluator).
func runArrayCase(t *testing.T, prog string, vmNative bool) map[string]string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "arr.ail")
	if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_NO_CACHE", "1")
	out := map[string]string{}
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
		if vmNative && backend.name == "vm" && (!strings.Contains(stderr, "via bytecode VM") || strings.Contains(stderr, "falling back to evaluator")) {
			t.Fatalf("vm: program did not run on the VM\nstderr=%.800s", stderr)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		out[backend.name] = lines[len(lines)-1]
	}
	return out
}

// An out-of-bounds set used to return the array unchanged (D5).
func TestStdArraySetOutOfBoundsFails(t *testing.T) {
	src := filepath.Join(t.TempDir(), "oob.ail")
	prog := arrayProgram("import std/array (fromList, set, get)",
		`show(get(set(fromList([1.0, 2.0, 3.0]), 3, 9.0), 0))`)
	if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, args := range [][]string{
		{"run", "--relax-modules", src},
		{"run", "--bytecode", "--relax-modules", src},
	} {
		_, stderr, code := runCLI(t, args...)
		if code == 0 || !strings.Contains(stderr, "array_set: index 3 out of bounds (array length: 3)") {
			t.Fatalf("%v: want an out-of-bounds error, got exit %d\nstderr=%.600s", args, code, stderr)
		}
	}
}

// The width is in the codec's name: F64 is exact, F32 rounds to float32, and a
// byte count that is not a whole number of floats is an Err naming the count.
func TestStdEmbeddingFloatCodecs(t *testing.T) {
	const imports = `import std/embedding (encodeF32LE, decodeF32LE, encodeF64LE, decodeF64LE)
import std/bytes (fromString)

func showR(r: Result[[float], string]) -> string = match r { Ok(xs) => show(xs), Err(e) => "Err: ${e}" }`
	cases := []struct{ name, body, want string }{
		{"f64 exact", `showR(decodeF64LE(encodeF64LE([0.1, 2.5, -3.0])))`, "[0.1, 2.5, -3.0]"},
		{"f32 lossy", `showR(decodeF32LE(encodeF32LE([0.1, 1.5])))`, "[0.10000000149011612, 1.5]"},
		{"f64 empty", `showR(decodeF64LE(encodeF64LE([])))`, "[]"},
		{"f64 bad length", `showR(decodeF64LE(fromString("abc")))`, "Err: decodeF64LE: 3 bytes is not a whole number of 8-byte floats"},
		{"f32 bad length", `showR(decodeF32LE(fromString("abcde")))`, "Err: decodeF32LE: 5 bytes is not a whole number of 4-byte floats"},
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

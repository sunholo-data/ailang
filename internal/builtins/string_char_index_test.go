package builtins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ailang#688: charAt/charCode materialised []rune(s) on every call, so an
// index-shaped primitive cost O(len(s)) time and allocated 4*len(s)+16 bytes
// regardless of the index. These tests pin BOTH halves of the fix:
//
//   - semantics are byte-for-byte what []rune(s)[i] produced (it was a cost
//     change, not a behaviour change), and
//   - the cost does not scale with len(s) (the arm that fails if anyone
//     reverts to []rune).
//
// The second is the load-bearing one: every correctness test below passes
// just as well against the O(n) implementation.

// charAtCorpus covers ASCII, multi-byte, astral-plane, invalid UTF-8 and the
// empty string. Invalid UTF-8 matters because []rune() maps each bad byte to
// U+FFFD, and any decode-based rewrite has to reproduce that exactly.
var charAtCorpus = []string{
	"",
	"a",
	"hello",
	"héllo",
	"日本語テキスト",
	"a\U0001F600b",
	"\xff",
	"a\xffb",
	"\xff\xfe\xfd",
	strings.Repeat("abcdefghij", 20),
}

// refCharAt is the pre-fix implementation, kept verbatim as the oracle.
func refCharAt(s string, idx int) (string, bool) {
	runes := []rune(s)
	if idx < 0 || idx >= len(runes) {
		return "", false
	}
	return string(runes[idx]), true
}

func TestCharAtMatchesRuneSliceSemantics(t *testing.T) {
	for _, s := range charAtCorpus {
		// Probe past both ends, so out-of-bounds is covered on every input.
		for idx := -2; idx <= len([]rune(s))+2; idx++ {
			wantVal, wantOK := refCharAt(s, idx)
			got, err := strCharAtImpl(nil, []eval.Value{
				&eval.StringValue{Value: s}, &eval.IntValue{Value: idx},
			})
			if wantOK {
				if err != nil {
					t.Errorf("charAt(%q, %d): got error %v, want %q", s, idx, err, wantVal)
					continue
				}
				sv, ok := got.(*eval.StringValue)
				if !ok {
					t.Errorf("charAt(%q, %d): got %T, want *eval.StringValue", s, idx, got)
					continue
				}
				if sv.Value != wantVal {
					t.Errorf("charAt(%q, %d) = %q, want %q", s, idx, sv.Value, wantVal)
				}
				continue
			}
			if err == nil {
				t.Errorf("charAt(%q, %d): got %v, want out-of-bounds error", s, idx, got)
				continue
			}
			// The message quotes the RUNE length; a byte length here would be
			// the same class of bug as the codegen tier's.
			want := fmt.Sprintf("index %d out of bounds for string of length %d", idx, len([]rune(s)))
			if !strings.Contains(err.Error(), want) {
				t.Errorf("charAt(%q, %d) error = %q, want it to contain %q", s, idx, err.Error(), want)
			}
		}
	}
}

func TestCharCodeMatchesRuneSliceSemantics(t *testing.T) {
	for _, s := range charAtCorpus {
		runes := []rune(s)
		got, err := strCharCodeImpl(nil, []eval.Value{&eval.StringValue{Value: s}})
		if len(runes) == 1 {
			if err != nil {
				t.Errorf("charCode(%q): got error %v, want %d", s, err, runes[0])
				continue
			}
			iv, ok := got.(*eval.IntValue)
			if !ok {
				t.Errorf("charCode(%q): got %T, want *eval.IntValue", s, got)
				continue
			}
			if iv.Value != int(runes[0]) {
				t.Errorf("charCode(%q) = %d, want %d", s, iv.Value, int(runes[0]))
			}
			continue
		}
		if err == nil {
			t.Errorf("charCode(%q): got %v, want error (%d characters)", s, got, len(runes))
			continue
		}
		want := fmt.Sprintf("got %d characters", len(runes))
		if !strings.Contains(err.Error(), want) {
			t.Errorf("charCode(%q) error = %q, want it to contain %q", s, err.Error(), want)
		}
	}
}

// TestCharAtCostDoesNotScaleWithLength is the arm that fails on a revert to
// []rune(s)[idx]. It reads allocated BYTES rather than wall-clock or alloc
// COUNT, because bytes are the mechanism: the old body allocated exactly
// 4*len(s)+16 per call (measured on #688's repro, at every index), while the
// count stayed flat at 3. A count-based assertion would therefore have passed
// against the defect.
//
// The two residual allocations per call are the boxed *eval.StringValue and
// its one-rune payload. Those are the evaluator's value representation, not
// #688, and they do not grow with the input.
func TestCharAtCostDoesNotScaleWithLength(t *testing.T) {
	bytesPerOp := func(n int) uint64 {
		s := strings.Repeat("abcdefghij", n/10)
		args := []eval.Value{&eval.StringValue{Value: s}, &eval.IntValue{Value: 0}}
		const iters = 2000
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		for i := 0; i < iters; i++ {
			if _, err := strCharAtImpl(nil, args); err != nil {
				t.Fatal(err)
			}
		}
		runtime.ReadMemStats(&m1)
		return (m1.TotalAlloc - m0.TotalAlloc) / iters
	}
	const long = 80000
	small, large := bytesPerOp(80), bytesPerOp(long)
	t.Logf("bytes/op: len=80 -> %d, len=%d -> %d (pre-fix would be ~%d)", small, long, large, 4*long+16)

	// Control: the measurement must be able to see a large allocation at all,
	// otherwise "small" below is vacuous. []rune(s) is exactly what was removed.
	var sink []rune
	ref := func() uint64 {
		s := strings.Repeat("abcdefghij", long/10)
		const iters = 200
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		for i := 0; i < iters; i++ {
			sink = []rune(s)
		}
		runtime.ReadMemStats(&m1)
		return (m1.TotalAlloc - m0.TotalAlloc) / iters
	}()
	_ = sink
	if ref < uint64(long) {
		t.Fatalf("instrument failure: []rune control measured %d bytes/op for a %d-rune string, want >= %d", ref, long, long)
	}
	t.Logf("control ([]rune materialisation): %d bytes/op", ref)

	if large > small+64 {
		t.Errorf("charAt allocation scales with string length (%d bytes/op at len=80 vs %d at len=%d): "+
			"the []rune materialisation is back (ailang#688)", small, large, long)
	}
	if large > 256 {
		t.Errorf("charAt allocates %d bytes per call at len=%d, want a small constant", large, long)
	}
}

// TestCodegenCharAtIsRuneIndexed runs the ACTUAL helper body the emit-go
// backend ships, rather than re-deriving what it ought to do.
//
// Before ailang#688 that body indexed by BYTE: charAt("héllo", 1) returned
// "Ã" compiled and "é" interpreted, and charAt("héllo", 5) returned "o"
// compiled where the interpreter raises out-of-bounds. A compiled program
// silently disagreed with the same source under the interpreter.
func TestCodegenCharAtIsRuneIndexed(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles and runs a Go program")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
	spec := GetCodegenSpec("_str_charAt")
	if spec == nil || spec.Helper == nil {
		t.Fatal("instrument failure: no codegen spec/helper for _str_charAt")
	}
	// charCode had no codegen spec at all, so compiled programs using it failed
	// with "undefined: CharCode". Emitted alongside charAt because they share
	// the import constraint and the rune-vs-byte hazard.
	ccSpec := GetCodegenSpec("_str_charCode")
	if ccSpec == nil || ccSpec.Helper == nil {
		t.Fatal("no codegen spec/helper for _str_charCode: compiled programs using charCode will not build")
	}
	for _, imp := range ccSpec.Imports {
		if !slices.Contains([]string{"fmt", "reflect", "strings"}, imp) {
			t.Errorf("charCode spec declares import %q, which runtime.go cannot emit for a Helper spec", imp)
		}
	}

	// runtimeImports is what codegen.go actually emits at the top of
	// runtime.go: a CLOSED allowlist. GoCodegenSpec.Imports is explicitly
	// skipped for Helper specs (codegen_registry.go, "Don't track imports
	// here"), so a helper body that reaches for anything outside this set
	// emits Go that does not compile.
	//
	// The first draft of this test built its import block from spec.Imports
	// and passed while the real pipeline failed with "undefined: utf8" — a
	// test that verified its own arithmetic instead of the artifact. Rendering
	// against the real allowlist is what makes it an artifact test.
	runtimeImports := []string{"fmt", "reflect", "strings"}
	for _, imp := range spec.Imports {
		if !slices.Contains(runtimeImports, imp) {
			t.Errorf("spec declares import %q, which runtime.go cannot emit for a Helper spec; "+
				"the generated code will not compile", imp)
		}
	}
	var imports strings.Builder
	for _, imp := range runtimeImports {
		fmt.Fprintf(&imports, "\t%q\n", imp)
	}
	prog := fmt.Sprintf(`package main

import (
%s)

var _ = reflect.TypeOf
var _ = strings.TrimSpace

func toInt64(v interface{}) int64 { return v.(int64) }

%s {
%s
}

%s {
%s
}

func main() {
	defer func() {
		if r := recover(); r != nil { fmt.Println("PANIC") }
	}()
	fmt.Println(CharAt("h\u00e9llo", int64(1)))
	fmt.Println(CharAt("h\u00e9llo", int64(4)))
	fmt.Println(CharCode("\u00e9"))
	fmt.Println(CharCode("\U0001F600"))
	fmt.Println(CharAt("h\u00e9llo", int64(5)))
}
`, imports.String(), spec.Helper.Signature, spec.Helper.Body, ccSpec.Helper.Signature, ccSpec.Helper.Body)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(prog), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cgcharat\n\ngo 1.21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated CharAt helper did not build/run: %v\n%s\n--- program ---\n%s", err, out, prog)
	}
	got := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	// "héllo" is 5 runes; index 4 is the last, index 5 is past the end.
	// 233 = U+00E9, 128512 = U+1F600: both are RUNE code points, which a byte-
	// or UTF-16-based implementation would not produce.
	want := []string{"\u00e9", "o", "233", "128512", "PANIC"}
	if len(got) != len(want) {
		t.Fatalf("generated CharAt printed %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("generated CharAt line %d = %q, want %q (byte indexing returns %q for line 0)",
				i, got[i], want[i], "\u00c3")
		}
	}
}

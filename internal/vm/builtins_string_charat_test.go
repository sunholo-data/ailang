package vm

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// ailang#688, VM tier. internal/vm carries its own copy of every string
// builtin and this file's header states the invariant those copies live under:
// "Each function below matches the semantics of its evaluator counterpart in
// internal/builtins/string*.go". Nothing tested that, which is how the three
// tiers came to disagree in the first place (the emit-go tier byte-indexed).
//
// These tests pin the VM copy against the SAME []rune oracle the
// internal/builtins tests use, so the two tiers agree by construction rather
// than by inspection.

var vmCharAtCorpus = []string{
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

func TestVMCharAtMatchesRuneSliceSemantics(t *testing.T) {
	for _, s := range vmCharAtCorpus {
		runes := []rune(s)
		for idx := -2; idx <= len(runes)+2; idx++ {
			got, err := builtinStrCharAt([]bytecode.Value{
				bytecode.NewString(s), bytecode.NewInt(int64(idx)),
			})
			if idx >= 0 && idx < len(runes) {
				if err != nil {
					t.Errorf("charAt(%q, %d): got error %v, want %q", s, idx, err, string(runes[idx]))
					continue
				}
				if got.AsString() != string(runes[idx]) {
					t.Errorf("charAt(%q, %d) = %q, want %q", s, idx, got.AsString(), string(runes[idx]))
				}
				continue
			}
			if err == nil {
				t.Errorf("charAt(%q, %d) = %q, want out-of-bounds error", s, idx, got.AsString())
				continue
			}
			want := fmt.Sprintf("index %d out of bounds for string of length %d", idx, len(runes))
			if !strings.Contains(err.Error(), want) {
				t.Errorf("charAt(%q, %d) error = %q, want it to contain %q", s, idx, err.Error(), want)
			}
		}
	}
}

func TestVMCharCodeMatchesRuneSliceSemantics(t *testing.T) {
	for _, s := range vmCharAtCorpus {
		runes := []rune(s)
		got, err := builtinStrCharCode([]bytecode.Value{bytecode.NewString(s)})
		if len(runes) == 1 {
			if err != nil {
				t.Errorf("charCode(%q): got error %v, want %d", s, err, runes[0])
				continue
			}
			if got.Int != int64(runes[0]) {
				t.Errorf("charCode(%q) = %d, want %d", s, got.Int, int64(runes[0]))
			}
			continue
		}
		if err == nil {
			t.Errorf("charCode(%q) = %d, want error (%d characters)", s, got.Int, len(runes))
			continue
		}
		want := fmt.Sprintf("got %d characters", len(runes))
		if !strings.Contains(err.Error(), want) {
			t.Errorf("charCode(%q) error = %q, want it to contain %q", s, err.Error(), want)
		}
	}
}

// See the internal/builtins twin for why this measures allocated BYTES and not
// alloc count: the pre-fix body's count was flat at 3 while its byte volume was
// 4*len(s)+16, so only bytes discriminate the defect.
func TestVMCharAtCostDoesNotScaleWithLength(t *testing.T) {
	bytesPerOp := func(n int) uint64 {
		s := strings.Repeat("abcdefghij", n/10)
		args := []bytecode.Value{bytecode.NewString(s), bytecode.NewInt(0)}
		const iters = 2000
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		for i := 0; i < iters; i++ {
			if _, err := builtinStrCharAt(args); err != nil {
				t.Fatal(err)
			}
		}
		runtime.ReadMemStats(&m1)
		return (m1.TotalAlloc - m0.TotalAlloc) / iters
	}
	const long = 80000
	small, large := bytesPerOp(80), bytesPerOp(long)
	t.Logf("bytes/op: len=80 -> %d, len=%d -> %d (pre-fix would be ~%d)", small, long, large, 4*long+16)

	var sink []rune
	s := strings.Repeat("abcdefghij", long/10)
	const iters = 200
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	for i := 0; i < iters; i++ {
		sink = []rune(s)
	}
	runtime.ReadMemStats(&m1)
	ref := (m1.TotalAlloc - m0.TotalAlloc) / iters
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

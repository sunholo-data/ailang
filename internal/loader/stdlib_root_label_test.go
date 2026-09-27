package loader

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/stdlibroot"
)

// The "module not found" error names where the one root came from, so a user
// with a stray ./std or a stale AILANG_STDLIB_PATH can see why it was chosen.
func TestErrWithSearchTrace_NamesTheRootSource(t *testing.T) {
	cases := map[string]string{
		"flag":     "--stdlib-path",
		"env":      "AILANG_STDLIB_PATH",
		"cwd":      "./std in the working directory",
		"binary":   "the std/ next to the ailang binary",
		"user":     "the user data directory",
		"system":   "a system directory",
		"whatever": "whatever",
	}
	r := NewStdlibResolver("", false, false)
	for source, want := range cases {
		msg := r.errWithSearchTrace("zz", stdlibroot.Root{Dir: "/x/std", Source: source}).Error()
		if !strings.Contains(msg, "stdlib root: /x/std (from "+want+")") {
			t.Errorf("source %q: error lacks %q:\n%s", source, want, msg)
		}
		if !strings.Contains(msg, "remove a partial ./std") {
			t.Errorf("source %q: on-disk root error lacks the partial-std tip", source)
		}
	}
	emb := r.errWithSearchTrace("zz", stdlibroot.Root{Source: "embedded"}).Error()
	if !strings.Contains(emb, "the copy built into this binary") || strings.Contains(emb, "partial ./std") {
		t.Errorf("embedded root error = %q", emb)
	}
}

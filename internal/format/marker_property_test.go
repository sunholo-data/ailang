package format

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// marker_property_test.go is the M3 marker property gate. Each input comment
// carries a UNIQUE marker token; the formatted output must contain each marker
// EXACTLY once, and output must be idempotent (fmt(fmt(x)) == fmt(x)). Unique
// markers avoid the substring-collision problem of raw comment text (a bare `--`
// is a substring of every `-- foo`).

// markeredCorpus generates a set of commented programs whose every comment is a
// unique marker, exercising rules 1–5 across the multi-line sites.
func markeredPrograms() []string {
	return []string{
		// File header + leading + trailing on decls.
		"-- MK0\nmodule m\n\n-- MK1\npure func f() -> int = 1  -- MK2\n\n-- MK3\npure func g() -> int = 2\n",
		// Block: leading, trailing, floating (statements are identifier-starters so
		// no `;` gluing occurs across the blank line).
		"module m\n\nexport func main() -> () ! {IO} {\n  -- MK4\n  let s = 1  -- MK5\n  let t = s\n\n  -- MK6\n  println(t)\n}\n",
		// Match arms.
		"module m\n\npure func h(x: int) -> int =\n  match x {\n    -- MK7\n    0 => 0,  -- MK8\n    _ => 1\n  }\n",
		// Consecutive comments (rule 5).
		"-- MK9\n-- MK10\nmodule m\n\npure func k() -> int = 3\n",
	}
}

func TestMarkerProperty_UniquePreservation(t *testing.T) {
	for i, src := range markeredPrograms() {
		t.Run(fmt.Sprintf("prog%d", i), func(t *testing.T) {
			out := fmtWithComments(t, src)
			// Every MK marker present in the input must appear exactly once.
			comments, _ := lexer.CollectComments([]byte(src))
			for _, c := range comments {
				marker := extractMarker(c.Text)
				if marker == "" {
					continue
				}
				if n := strings.Count(out, marker); n != 1 {
					t.Errorf("marker %s appears %d times (want 1):\n%s", marker, n, out)
				}
			}
			// Idempotence.
			assertIdempotent(t, out)
		})
	}
}

// extractMarker pulls the MKn token from a comment body, or "" if absent.
func extractMarker(text string) string {
	for _, f := range strings.Fields(text) {
		if strings.HasPrefix(f, "MK") {
			return f
		}
	}
	return ""
}

// TestMarkerProperty_CorpusIdempotence formats every parse-valid commented corpus
// file that the formatter accepts, then formats the output again, requiring
// byte-identity (fmt(fmt(x)) == fmt(x)).
func TestMarkerProperty_CorpusIdempotence(t *testing.T) {
	var checked, idempotent int
	walkAilExamples(t, func(path string, data []byte) {
		p := parser.New(lexer.New(string(data), path))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			return
		}
		out1, err := SourceWithComments(prog, data, Options{})
		if err != nil {
			return // fail-closed files are excluded (measured in the corpus gate)
		}
		// Only assert idempotence on files whose formatted output re-parses cleanly
		// (the pre-existing Phase-1 printer bugs are excluded — they fail closed at
		// the CLI via the round-trip check).
		rp := parser.New(lexer.New(string(out1), path))
		reprog := rp.Parse()
		if len(rp.Errors()) > 0 || reprog == nil || reprog.File == nil {
			return
		}
		checked++
		out2, err := SourceWithComments(reprog, out1, Options{})
		if err != nil {
			t.Errorf("%s: second format failed: %v", path, err)
			return
		}
		if string(out1) == string(out2) {
			idempotent++
		} else {
			t.Errorf("%s: not idempotent (fmt(fmt(x)) != fmt(x))", path)
		}
	})
	if checked == 0 {
		t.Skip("no corpus files checked for idempotence")
	}
	t.Logf("marker/idempotence: %d/%d corpus files idempotent", idempotent, checked)
}

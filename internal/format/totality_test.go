package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// totality_test.go verifies the attacher's totality guarantee (every comment gets
// exactly one attachment or the file fails closed) over the hard-left-wall cases
// and the M0-inventory single-line-list families (whose interior comments float
// to the nearest enclosing multi-line list, never trapped in the first child).

// tryFmt parses and formats src, returning the output and any fail-closed error.
func tryFmt(t *testing.T, src string) (string, error) {
	t.Helper()
	p := parser.New(lexer.New(src, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse:\n%s\nerror: %v", src, errs[0])
	}
	out, err := SourceWithComments(prog, []byte(src), Options{})
	return string(out), err
}

// TestTotality_HardLeftWall_ListElement covers a comment BEFORE the first element
// of a single-line list literal that is the inline body of a declaration
// (`func f() = <comment> [1,2,3]`). The comment is interior to a value the printer
// emits inline, so there is no stable boundary — the attacher FAILS CLOSED rather
// than relocate it non-idempotently. The contract is: never a silent drop. This
// exercises the hard-left-wall property (the comment is never trapped inside the
// element `1`); the fail-closed outcome is the safe, enumerated behavior.
func TestTotality_HardLeftWall_ListElement(t *testing.T) {
	src := "module m\n\npure func f() -> [int] =\n  -- head\n  [1, 2, 3]\n"
	out, err := tryFmt(t, src)
	if err != nil {
		// Fail-closed: acceptable and expected here (interior-to-inline-value).
		ee, ok := err.(*EnvelopeError)
		if !ok || ee.Kind != "comment-unattached" {
			t.Fatalf("expected fail-closed comment-unattached, got: %v", err)
		}
		return
	}
	// If it DID attach, the comment must still survive exactly once (no silent drop
	// or duplication) — the hard-left-wall guarantee.
	if n := strings.Count(out, "-- head"); n != 1 {
		t.Errorf("comment appears %d times (want 1):\n%s", n, out)
	}
}

// TestTotality_NoSilentDrop_FailClosed proves that when a comment cannot be
// attached, the formatter FAILS CLOSED (returns an error) rather than silently
// dropping or relocating it. We construct a case the attacher does not cover
// (a comment mid single-line call args) and require either a clean attachment OR
// a fail-closed error — never silent loss.
func TestTotality_NoSilentDrop(t *testing.T) {
	cases := []string{
		"module m\n\npure func f() -> int =\n  g(1,\n    -- mid arg\n    2)\n",
		"module m\n\n-- before\npure func f() -> int = 1\n",
		"module m\n\npure func f() -> int = 1\n-- after last decl\n",
	}
	for _, src := range cases {
		out, err := tryFmt(t, src)
		comments, _ := lexer.CollectComments([]byte(src))
		if err != nil {
			// Fail-closed: acceptable, no silent loss.
			continue
		}
		for _, c := range comments {
			txt := strings.TrimRight(c.Text, " \t")
			if n := strings.Count(out, txt); n != 1 {
				t.Errorf("comment %q appears %d times (want exactly 1 — silent drop/dup is a totality violation):\nsrc:\n%s\nout:\n%s", txt, n, src, out)
			}
		}
	}
}

// TestTotality_MatchArms covers comments at match-case boundaries (a multi-line
// M0 site).
func TestTotality_MatchArms(t *testing.T) {
	src := "module m\n\npure func f(x: int) -> int =\n  match x {\n    -- zero case\n    0 => 0,\n    _ => 1\n  }\n"
	out, err := tryFmt(t, src)
	if err != nil {
		t.Fatalf("match-arm comment failed closed: %v", err)
	}
	if n := strings.Count(out, "-- zero case"); n != 1 {
		t.Errorf("match-arm comment appears %d times (want 1):\n%s", n, out)
	}
	// Fixed point.
	p := parser.New(lexer.New(out, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("match output did not re-parse:\n%s\n%v", out, errs[0])
	}
	out2, err := SourceWithComments(prog, []byte(out), Options{})
	if err != nil {
		t.Fatalf("match second format: %v", err)
	}
	if string(out2) != out {
		t.Errorf("match not idempotent:\n--- 1 ---\n%s\n--- 2 ---\n%s", out, string(out2))
	}
}

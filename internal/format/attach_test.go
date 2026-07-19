package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// attach_test.go covers deterministic attachment (rules 1–5), emission
// interleaving, totality, and per-rule fixed-point (idempotence) tests.

// fmtWithComments parses src and formats it with comments re-attached. It fails
// the test on any error (attachment must be total for these fixtures).
func fmtWithComments(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse:\n%s\nerror: %v", src, errs[0])
	}
	out, err := SourceWithComments(prog, []byte(src), Options{})
	if err != nil {
		t.Fatalf("SourceWithComments:\n%s\nerror: %v", src, err)
	}
	return string(out)
}

// assertMarkerPreserved checks that every comment in src appears exactly once in
// out (marker preservation at the fixture level).
func assertMarkerPreserved(t *testing.T, src, out string) {
	t.Helper()
	comments, _ := lexer.CollectComments([]byte(src))
	for _, c := range comments {
		txt := strings.TrimRight(c.Text, " \t")
		if n := strings.Count(out, txt); n != 1 {
			t.Errorf("comment %q appears %d times in output (want 1):\n%s", txt, n, out)
		}
	}
}

// assertIdempotent formats the output a second time and requires byte-identity.
func assertIdempotent(t *testing.T, out string) {
	t.Helper()
	p := parser.New(lexer.New(out, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("formatted output did not re-parse:\n%s\nerror: %v", out, errs[0])
	}
	out2b, err := SourceWithComments(prog, []byte(out), Options{})
	if err != nil {
		t.Fatalf("second format failed:\n%s\nerror: %v", out, err)
	}
	if string(out2b) != out {
		t.Errorf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, string(out2b))
	}
}

func TestAttach_Rule2_Leading(t *testing.T) {
	// A comment directly above a decl with no blank line → leading.
	src := "module m\n\n-- doc\npure func f() -> int = 1\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	if !strings.Contains(out, "-- doc\npure func f") {
		t.Errorf("leading comment not emitted directly above decl:\n%s", out)
	}
	assertIdempotent(t, out)
}

func TestAttach_Rule1_Trailing(t *testing.T) {
	// A comment on the same line as a decl → trailing.
	src := "module m\n\npure func f() -> int = 1  -- note\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	// The comment must be on the SAME line as its owner (rule 1 fixed point).
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "-- note") {
			if !strings.Contains(line, "func f") {
				t.Errorf("trailing comment moved off its owner's line: %q", line)
			}
		}
	}
	assertIdempotent(t, out)
}

func TestAttach_Rule4_FileHeader(t *testing.T) {
	// A comment before the module → file boundary 0.
	src := "-- header\nmodule m\n\npure func f() -> int = 1\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	if !strings.HasPrefix(out, "-- header\n") {
		t.Errorf("file-header comment not at top:\n%s", out)
	}
	assertIdempotent(t, out)
}

func TestAttach_Rule3_FloatingInBlock(t *testing.T) {
	// A comment separated by a blank line inside a block → floating.
	src := "module m\n\nexport func main() -> () ! {IO} {\n  let s = 1;\n\n  -- floating\n  ()\n}\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	if !strings.Contains(out, "-- floating") {
		t.Errorf("floating comment lost:\n%s", out)
	}
	assertIdempotent(t, out)
}

func TestAttach_Rule1_TrailingInBlock(t *testing.T) {
	src := "module m\n\nexport func main() -> () ! {IO} {\n  let s = 1  -- sum\n  println(s)\n  ()\n}\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "-- sum") && !strings.Contains(line, "let s") {
			t.Errorf("trailing block comment moved off its statement: %q", line)
		}
	}
	assertIdempotent(t, out)
}

func TestAttach_ZeroCommentByteIdentity(t *testing.T) {
	// Comment-free input: SourceWithComments must equal Source byte-for-byte.
	src := "module m\n\npure func add(x: int, y: int) -> int = x + y\n\nexport func main() -> () ! {IO} {\n  let s = add(1, 2)\n  ()\n}\n"
	p := parser.New(lexer.New(src, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs[0])
	}
	withC, err := SourceWithComments(prog, []byte(src), Options{})
	if err != nil {
		t.Fatalf("SourceWithComments: %v", err)
	}
	plain, err := Source(prog, Options{})
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	if string(withC) != string(plain) {
		t.Errorf("comment-free output diverged from Phase-1 Source:\n--- withComments ---\n%s\n--- Source ---\n%s", withC, plain)
	}
}

func TestAttach_ExampleFromDesignDoc(t *testing.T) {
	// The design doc's worked example (Examples section).
	src := `-- File header: formatting demo.
module examples/fmt_demo

import std/io (println)

-- Adds two integers.
pure func add(x: int, y: int) -> int = x + y  -- trailing note

export func main() -> () ! {IO} {
  let s = add(1, 2)  -- compute the sum

  -- floating comment before the final expression
  println(show(s))
}
`
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	assertIdempotent(t, out)
	// Structural spot checks.
	if !strings.HasPrefix(out, "-- File header: formatting demo.\nmodule examples/fmt_demo") {
		t.Errorf("header/module ordering wrong:\n%s", out)
	}
	if !strings.Contains(out, "-- Adds two integers.\npure func add") {
		t.Errorf("leading comment on add lost:\n%s", out)
	}
}

func TestAttach_Rule5_ConsecutiveComments(t *testing.T) {
	// Consecutive comments preserve order and grouping.
	src := "-- line one\n-- line two\nmodule m\n\npure func f() -> int = 1\n"
	out := fmtWithComments(t, src)
	assertMarkerPreserved(t, src, out)
	i1 := strings.Index(out, "-- line one")
	i2 := strings.Index(out, "-- line two")
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Errorf("consecutive comment order not preserved:\n%s", out)
	}
	assertIdempotent(t, out)
}

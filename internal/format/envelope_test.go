package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// envelope_test.go covers the token-anchored envelope: anchor conversion,
// literal clamping, bracket matching, child-boundary resolution with the hard
// left wall, and the fail-closed interpolation-comment carve-out.

func mustEnvelope(t *testing.T, src string) *Envelope {
	t.Helper()
	env, err := NewEnvelope([]byte(src))
	if err != nil {
		t.Fatalf("NewEnvelope(%q): %v", src, err)
	}
	return env
}

func parseExpr(t *testing.T, src string) ast.Node {
	t.Helper()
	p := parser.New(lexer.New(src, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse %q: %v", src, errs[0])
	}
	if prog == nil || prog.File == nil || len(prog.File.Decls) == 0 {
		t.Fatalf("no decls parsed from %q", src)
	}
	return prog.File.Decls[0]
}

func TestEnvelope_OffsetConversion(t *testing.T) {
	// Multi-byte: "héπ" occupies rune columns; the `;` after must land on '['.
	src := "let x = [1, 2]\n"
	env := mustEnvelope(t, src)
	// Column 9 (1-based) on line 1 is the '[' after "let x = ".
	off, err := env.offsetOf(1, 9)
	if err != nil {
		t.Fatalf("offsetOf: %v", err)
	}
	if env.src[off] != '[' {
		t.Errorf("offset %d = %q, want '['", off, string(env.src[off]))
	}
}

func TestEnvelope_UnicodeOffset(t *testing.T) {
	src := `let x = "héπ😀"` + "\n"
	env := mustEnvelope(t, src)
	// The opening quote is at rune column 9.
	off, err := env.offsetOf(1, 9)
	if err != nil {
		t.Fatalf("offsetOf: %v", err)
	}
	if env.src[off] != '"' {
		t.Errorf("offset %d = %q, want '\"'", off, string(env.src[off]))
	}
}

func TestEnvelope_MinAnchorRecoversLeftmostToken(t *testing.T) {
	// `x + 42` — the BinaryOp is positioned at `+`; MinAnchor must recover `x`.
	decl := parseExpr(t, "let r = x + 42\n")
	env := mustEnvelope(t, "let r = x + 42\n")
	// Drill to the value expression.
	let, ok := decl.(*ast.Let)
	if !ok {
		t.Fatalf("expected Let, got %T", decl)
	}
	min, err := env.MinAnchor(let.Value)
	if err != nil {
		t.Fatalf("MinAnchor: %v", err)
	}
	if env.src[min] != 'x' {
		t.Errorf("MinAnchor landed on %q at %d, want 'x'", string(env.src[min]), min)
	}
}

func TestEnvelope_BracketMatching(t *testing.T) {
	src := "let x = [1, [2, 3], 4]\n"
	env := mustEnvelope(t, src)
	open := strings.IndexByte(env.src, '[')
	closeIdx := env.matchBracket(open)
	// The matching ] is the final one before the newline.
	wantClose := strings.LastIndexByte(env.src, ']')
	if closeIdx != wantClose {
		t.Errorf("matchBracket(%d) = %d, want %d", open, closeIdx, wantClose)
	}
}

func TestEnvelope_HardLeftWall_FirstChildDoesNotConsumeParenOpen(t *testing.T) {
	// [ x ] — x's widening must STOP at the list's `[` (parentOpen), not consume it.
	src := "let l = [ x ]\n"
	env := mustEnvelope(t, src)
	parentOpen := strings.IndexByte(env.src, '[')
	xPos := strings.IndexByte(env.src, 'x')
	widened := env.WidenLeft(xPos, parentOpen)
	if widened <= parentOpen {
		t.Errorf("WidenLeft crossed the parent's `[`: widened=%d, parentOpen=%d (hard left wall breached)", widened, parentOpen)
	}
	if widened != xPos {
		t.Errorf("WidenLeft should not move over the `[`: got %d, want %d (x)", widened, xPos)
	}
}

func TestEnvelope_HardLeftWall_NestedList(t *testing.T) {
	// [ [ y ] ] — the inner list's widening stops at the outer `[`.
	src := "let l = [ [ y ] ]\n"
	env := mustEnvelope(t, src)
	outerOpen := strings.IndexByte(env.src, '[')
	innerOpen := strings.IndexByte(env.src[outerOpen+1:], '[') + outerOpen + 1
	// Widen the inner list's own open with the outer list as parent: it may absorb
	// its own `[` but must not cross the outer `[`.
	widened := env.WidenLeft(innerOpen, outerOpen)
	if widened <= outerOpen {
		t.Errorf("inner widening crossed outer `[`: widened=%d, outerOpen=%d", widened, outerOpen)
	}
}

func TestEnvelope_WidenOverModifiers(t *testing.T) {
	// A func decl's min-anchor is at `func`; widening (file level, parentOpen=-1)
	// must absorb the leading `export pure` modifiers.
	src := "export pure func f(x: int) -> int = x\n"
	env := mustEnvelope(t, src)
	funcPos := strings.Index(env.src, "func")
	widened := env.WidenLeft(funcPos, -1)
	if !strings.HasPrefix(env.src[widened:], "export") {
		t.Errorf("WidenLeft did not absorb modifiers: got %q...", safeSlice(env.src, widened, 20))
	}
}

func TestEnvelope_InterpolationCommentRefused(t *testing.T) {
	// A comment inside a ${...} hole must make NewEnvelope fail closed.
	src := "let s = \"pre${ x -- oops\n }post\"\n"
	_, err := NewEnvelope([]byte(src))
	if err == nil {
		t.Fatal("expected fail-closed envelope error for interpolation comment, got nil")
	}
	ee, ok := err.(*EnvelopeError)
	if !ok {
		t.Fatalf("expected *EnvelopeError, got %T: %v", err, err)
	}
	if ee.Kind != "interp-comment" {
		t.Errorf("error Kind = %q, want interp-comment", ee.Kind)
	}
}

func TestEnvelope_NoFalseInterpolationRefusal(t *testing.T) {
	// An ordinary interpolation with no comment must NOT be refused.
	src := "let s = \"hello ${name}!\"\n"
	if _, err := NewEnvelope([]byte(src)); err != nil {
		t.Errorf("unexpected refusal for comment-free interpolation: %v", err)
	}
}

func TestEnvelope_AnchorClampedToLiteral(t *testing.T) {
	// An anchor pointing inside a string literal clamps to the region start.
	src := `let s = "abc"` + "\n"
	env := mustEnvelope(t, src)
	quote := strings.IndexByte(env.src, '"')
	// Fabricate a Pos pointing at the 'b' inside the literal (rune col of 'b').
	// Convert via AnchorOf and expect clamping to the opening quote.
	// 'b' is 2 runes into the string, i.e. quote byte + 2.
	insideCol := quote + 3 // 1-based column of 'b' on line 1 (all ASCII here)
	off, err := env.AnchorOf(ast.Pos{Line: 1, Column: insideCol})
	if err != nil {
		t.Fatalf("AnchorOf: %v", err)
	}
	if off != quote {
		t.Errorf("anchor inside literal = %d, want clamped to region start %d", off, quote)
	}
}

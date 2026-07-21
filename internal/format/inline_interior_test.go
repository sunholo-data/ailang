package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// inline_interior_test.go covers M1 (let-chain discovery + attachment model) and M2
// (conditional multi-line emission) for M-AILANG-FMT-INLINE-INTERIOR.
//
// M1 tests exercise the ATTACHMENT layer directly (AttachComments), asserting that a
// comment at a let-chain boundary now resolves to the root *ast.Let owner instead of
// hitting the strict-interior fail-closed guard — while an UNMODELLED binding-value
// interior comment STILL refuses (Decision 2: the guard is unchanged).

// attachAll parses src, builds the envelope, and returns the parsed file plus the
// attachment set (or the fail-closed error). Returning the SAME parsed file is
// essential: attachment owners are AST pointers, so ownership checks must use the
// exact file the attacher walked. It is the M1 attachment-layer probe (M2 covers
// emission).
func attachAll(t *testing.T, src string) (*ast.File, []Attachment, error) {
	t.Helper()
	p := parser.New(lexer.New(src, "test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse:\n%s\nerror: %v", src, errs[0])
	}
	env, err := NewEnvelope([]byte(src))
	if err != nil {
		t.Fatalf("envelope:\n%s\nerror: %v", src, err)
	}
	atts, aerr := AttachComments(env, prog.File)
	return prog.File, atts, aerr
}

// findRootLet returns the first *ast.Let with a non-nil Body reachable from the
// file's decls (the chain root the attacher keys on).
func findRootLet(f *ast.File) *ast.Let {
	var found *ast.Let
	var visit func(n ast.Node)
	visitExpr := func(e ast.Expr) { visit(e) }
	visit = func(n ast.Node) {
		if found != nil || n == nil {
			return
		}
		switch v := n.(type) {
		case *ast.FuncDecl:
			if v.Body != nil {
				visit(v.Body)
			}
		case *ast.Let:
			if v.Body != nil {
				found = v
				return
			}
			if v.Value != nil {
				visit(v.Value)
			}
		case *ast.Block:
			for _, e := range v.Exprs {
				visitExpr(e)
			}
		case *ast.If:
			visit(v.Condition)
			visit(v.Then)
			if v.Else != nil {
				visit(v.Else)
			}
		}
	}
	for _, d := range f.Decls {
		visit(d)
	}
	return found
}

// ownsAny reports whether any attachment in atts is keyed on owner.
func ownsAny(atts []Attachment, owner any) bool {
	for _, a := range atts {
		if a.Owner == owner {
			return true
		}
	}
	return false
}

func TestInlineInterior_AttachBeforeSecondBinding_Leading(t *testing.T) {
	// Comment directly above the 2nd binding, no blank line → LEADING at boundary 1.
	src := "module m\n\nexport func main() -> int =\n  let x = 1 in\n  -- KEEP\n  let y = 2 in\n  x + y\n"
	f, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed (should succeed at a let-chain boundary): %v", err)
	}
	root := findRootLet(f)
	if root == nil {
		t.Fatal("no root let found")
	}
	if !ownsAny(atts, root) {
		t.Errorf("comment not attached to the root let owner; attachments=%+v", atts)
	}
}

func TestInlineInterior_AttachFloatingBetweenBindings(t *testing.T) {
	// A blank line separates the comment from the NEXT binding → FLOATING at boundary 1
	// (rule 3). The comment hugs the previous binding, then a blank line, then `let y`.
	src := "module m\n\nexport func main() -> int =\n  let x = 1 in\n  -- FLOAT\n\n  let y = 2 in\n  x + y\n"
	f, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed: %v", err)
	}
	root := findRootLet(f)
	if !ownsAny(atts, root) {
		t.Errorf("floating comment not attached to root let owner; attachments=%+v", atts)
	}
	// It must be a floating place (blank line above the next binding).
	var gotFloating bool
	for _, a := range atts {
		if a.Owner == root && a.Place == PlaceFloating {
			gotFloating = true
		}
	}
	if !gotFloating {
		t.Errorf("expected a PlaceFloating attachment at the let-chain boundary; got %+v", atts)
	}
}

func TestInlineInterior_AttachTrailingAfterIn(t *testing.T) {
	// Same-line comment after `in` → TRAILING on the preceding binding.
	src := "module m\n\nexport func main() -> int =\n  let x = 1 in  -- TRAIL\n  let y = 2 in\n  x + y\n"
	_, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed: %v", err)
	}
	var gotTrailing bool
	for _, a := range atts {
		if a.Place == PlaceTrailing {
			gotTrailing = true
		}
	}
	if !gotTrailing {
		t.Errorf("expected a PlaceTrailing attachment for the same-line comment; got %+v", atts)
	}
}

func TestInlineInterior_AttachBeforeTail(t *testing.T) {
	// Comment above the terminal tail → LEADING at the tail boundary.
	src := "module m\n\nexport func main() -> int =\n  let x = 1 in\n  let y = 2 in\n  -- TAIL\n  x + y\n"
	f, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed: %v", err)
	}
	root := findRootLet(f)
	if !ownsAny(atts, root) {
		t.Errorf("tail comment not attached to root let owner; attachments=%+v", atts)
	}
}

func TestInlineInterior_AttachConsecutiveComments(t *testing.T) {
	// Two consecutive comments between bindings → both attach, source order kept.
	src := "module m\n\nexport func main() -> int =\n  let x = 1 in\n  -- ONE\n  -- TWO\n  let y = 2 in\n  x + y\n"
	_, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed: %v", err)
	}
	// Both comments present in the attachment set.
	var n int
	for range atts {
		n++
	}
	if n < 2 {
		t.Errorf("expected >=2 attachments for two consecutive comments, got %d: %+v", n, atts)
	}
}

func TestInlineInterior_AttachBeforeFirstBinding_Boundary0(t *testing.T) {
	// Comment directly after `=` above the FIRST binding → boundary 0. This is the
	// single-binding let…in shape M0 found in array_adt.ail / std_deflate_pdf_objstm.ail.
	src := "module m\n\nexport func main() -> int =\n  -- HEAD\n  let x = 1 in\n  x + 1\n"
	f, atts, err := attachAll(t, src)
	if err != nil {
		t.Fatalf("attachment failed at boundary 0: %v", err)
	}
	root := findRootLet(f)
	if !ownsAny(atts, root) {
		t.Errorf("boundary-0 comment not attached to root let owner; attachments=%+v", atts)
	}
}

func TestInlineInterior_UnmodelledBindingValueInterior_StillRefuses(t *testing.T) {
	// A comment strictly INSIDE a multi-line binding VALUE that has NO registered child
	// list (an if/then/else split across lines) must STILL refuse (Decision 2 — the
	// strict-interior guard is unchanged). The comment sits between `then …` and
	// `else …`, interior to the if-expression, which the formatter does not decompose.
	src := "module m\n\nexport func main() -> int =\n  let x =\n    if true then\n      1\n    -- INTERIOR\n    else\n      2\n  in\n  x + 1\n"
	_, _, err := attachAll(t, src)
	if err == nil {
		t.Fatalf("expected fail-closed refusal for an unmodelled binding-value interior comment, but attachment succeeded")
	}
	ee, ok := err.(*EnvelopeError)
	if !ok || ee.Kind != "comment-unattached" {
		t.Fatalf("expected comment-unattached refusal, got %v", err)
	}
}

// inlineInteriorFooterCarriers are target files whose let-chain comment attaches, but
// which carry an ADDITIONAL comment in a still-deferred class (e.g. a top-level footer
// comment after the last expression). They legitimately remain refused after M1/M2 —
// their residual refusal is NOT let-chain-interior class (M3 verifies the sub-class is
// 0). records.ail has a trailing `-- Output: 25` footer after `distance(point)`.
var inlineInteriorFooterCarriers = map[string]string{
	"examples/runnable/records.ail": "footer comment `-- Output: 25` after the last top-level expression (no-enclosing-list/footer class, deferred)",
}

func TestInlineInterior_AllTargetsAttach(t *testing.T) {
	// M1 acceptance METRIC: the M0-verified N=28 let-chain files no longer return
	// comment-unattached FOR THEIR LET-CHAIN COMMENTS at the attachment layer. Files
	// carrying an additional deferred-class comment (footer, etc.) stay refused for
	// THAT comment — enumerated in inlineInteriorFooterCarriers. (Emission stays inline
	// until M2, so this asserts ATTACHMENT resolution only, not lossless formatting.)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	var attached int
	var unexpectedRefusals []string
	for _, rel := range inlineInteriorLetChainTargets {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		p := parser.New(lexer.New(string(data), rel))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			t.Fatalf("%s not parse-valid", rel)
		}
		env, err := NewEnvelope(data)
		if err != nil {
			t.Fatalf("%s envelope: %v", rel, err)
		}
		_, aerr := AttachComments(env, prog.File)
		if aerr != nil {
			if ee, ok := aerr.(*EnvelopeError); ok && ee.Kind == "comment-unattached" {
				if _, known := inlineInteriorFooterCarriers[rel]; known {
					continue // known deferred-class residual; not a let-chain refusal
				}
				unexpectedRefusals = append(unexpectedRefusals, rel)
				continue
			}
			t.Fatalf("%s unexpected attach error: %v", rel, aerr)
		}
		attached++
	}
	fully := len(inlineInteriorLetChainTargets) - len(inlineInteriorFooterCarriers)
	t.Logf("M1 ATTACHMENT: %d/%d target let-chain files attach cleanly (%d carry a deferred-class footer/other comment: %v); was 0/%d at baseline",
		attached, len(inlineInteriorLetChainTargets), len(inlineInteriorFooterCarriers), keysOf(inlineInteriorFooterCarriers), len(inlineInteriorLetChainTargets))
	if len(unexpectedRefusals) != 0 {
		t.Fatalf("M1: %d target files still refuse comment-unattached with NO known deferred-class cause: %v",
			len(unexpectedRefusals), unexpectedRefusals)
	}
	if attached != fully {
		t.Fatalf("M1: expected %d clean-attaching targets, got %d", fully, attached)
	}
}

func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// TestInlineInterior_LetChainPreservedAndIdempotent is the M2 acceptance gate (design
// §Testing Strategy). It runs the Example-2 source through the full 7-assertion chain:
// parse OK → SourceWithComments OK → KEEP_INTERIOR exactly once → exact canonical
// multi-line golden → structural reparse equality (ignore Pos/Span) → SourceWithComments
// on the output → pass-two bytes == pass-one bytes. It FAILS if the comment is floated
// to a declaration/block boundary instead of the chain boundary.
func TestInlineInterior_LetChainPreservedAndIdempotent(t *testing.T) {
	src := "module demo\n\nexport func main() -> int =\n  let x = 1 in\n  -- KEEP_INTERIOR\n  let y = 2 in\n  x + y\n"

	// (1) Parse OK.
	p := parser.New(lexer.New(src, "demo"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("(1) parse: %v", errs[0])
	}

	// (2) SourceWithComments OK.
	out, err := SourceWithComments(prog, []byte(src), Options{})
	if err != nil {
		t.Fatalf("(2) SourceWithComments: %v", err)
	}
	outS := string(out)

	// (3) KEEP_INTERIOR exactly once.
	if n := strings.Count(outS, "-- KEEP_INTERIOR"); n != 1 {
		t.Fatalf("(3) KEEP_INTERIOR appears %d times (want 1):\n%s", n, outS)
	}

	// (4) Exact canonical multi-line golden (the design's Example 2). The comment MUST
	// sit between the two bindings at the chain boundary — NOT floated to the decl/block.
	golden := "module demo\n\nexport func main() -> int =\n  let x = 1 in\n  -- KEEP_INTERIOR\n  let y = 2 in\n  x + y\n"
	if outS != golden {
		t.Fatalf("(4) canonical golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", outS, golden)
	}
	// Guard against a silent regression that stops refusing but floats the comment to a
	// boundary: the comment line must be immediately BETWEEN the two `let` lines.
	lines := strings.Split(outS, "\n")
	var ci = -1
	for i, l := range lines {
		if strings.Contains(l, "-- KEEP_INTERIOR") {
			ci = i
		}
	}
	if ci < 1 || ci+1 >= len(lines) ||
		!strings.Contains(lines[ci-1], "let x = 1 in") ||
		!strings.Contains(lines[ci+1], "let y = 2 in") {
		t.Fatalf("(4b) comment not at the chain boundary (floated?):\n%s", outS)
	}

	// (5) Reparse output; structural AST equality with the original (ignore Pos/Span).
	rp := parser.New(lexer.New(outS, "demo"))
	reprog := rp.Parse()
	if errs := rp.Errors(); len(errs) > 0 {
		t.Fatalf("(5) reparse of output failed: %v", errs[0])
	}
	if diff := cmp.Diff(prog.File, reprog.File, ignorePosSpan); diff != "" {
		t.Fatalf("(5) structural round-trip differs (-orig +reparsed):\n%s", diff)
	}

	// (6)+(7) SourceWithComments on the output; pass-two bytes == pass-one bytes.
	out2, err := SourceWithComments(reprog, out, Options{})
	if err != nil {
		t.Fatalf("(6) second SourceWithComments: %v", err)
	}
	if string(out2) != outS {
		t.Fatalf("(7) not idempotent:\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s", outS, string(out2))
	}
}

// TestInlineInterior_TargetsFormatLosslessly is the M2 METRIC probe: every clean
// target file (all but the deferred-footer carriers) now formats without a refusal and
// preserves its comment count exactly. This is the CLI-equivalent "refusals for the
// target set drop to 0" check at the library layer.
func TestInlineInterior_TargetsFormatLosslessly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	var ok int
	for _, rel := range inlineInteriorLetChainTargets {
		if _, footer := inlineInteriorFooterCarriers[rel]; footer {
			continue // deferred-class residual comment keeps it refused (M3 verifies)
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		p := parser.New(lexer.New(string(data), rel))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			t.Fatalf("%s not parse-valid", rel)
		}
		out, ferr := SourceWithComments(prog, data, Options{})
		if ferr != nil {
			t.Errorf("%s: expected lossless format, got refusal: %v", rel, ferr)
			continue
		}
		in, _ := lexer.CollectComments(data)
		got, _ := lexer.CollectComments(out)
		if len(got) != len(in) {
			t.Errorf("%s: comment count changed %d -> %d (marker loss)", rel, len(in), len(got))
			continue
		}
		// Structural round-trip must hold.
		rp := parser.New(lexer.New(string(out), rel))
		reprog := rp.Parse()
		if len(rp.Errors()) > 0 || reprog == nil || reprog.File == nil {
			t.Errorf("%s: formatted output did not reparse", rel)
			continue
		}
		if diff := cmp.Diff(prog.File, reprog.File, ignorePosSpan); diff != "" {
			t.Errorf("%s: structural round-trip broke after formatting", rel)
			continue
		}
		ok++
	}
	want := len(inlineInteriorLetChainTargets) - len(inlineInteriorFooterCarriers)
	t.Logf("M2 LOSSLESS: %d/%d clean target files format losslessly (comment count preserved + round-trip)", ok, want)
	if ok != want {
		t.Fatalf("M2: expected %d clean targets to format losslessly, got %d", want, ok)
	}
}

// TestInlineInterior_DeferredFilesStillRefuse is the M3 fail-closed probe (library
// layer): a deferred non-let refusal file still returns comment-unattached (no partial
// output), proving the strict-interior guard remains intact for out-of-scope classes.
// The CLI equivalent (exit 2 + unchanged SHA-256 under `fmt --write`) rides on this
// same SourceWithComments refusal — cmd/ailang/fmt.go writes nothing when it errors.
func TestInlineInterior_DeferredFilesStillRefuse(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	// Enumerated deferred refusals across distinct classes: a non-let equation body, an
	// inline tests list, and a footer/no-enclosing-list case.
	deferred := []string{
		"examples/docs/records_person.ail",     // non-let single-expression equation body
		"examples/inline_tests_arithmetic.ail", // inline tests[...] list
		"examples/runnable/cli_args_demo.ail",  // footer / no-enclosing-list
	}
	for _, rel := range deferred {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		p := parser.New(lexer.New(string(data), rel))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			t.Fatalf("%s not parse-valid", rel)
		}
		out, ferr := SourceWithComments(prog, data, Options{})
		if ferr == nil {
			t.Errorf("%s: expected fail-closed refusal (deferred class), but it formatted:\n%s", rel, out)
			continue
		}
		if ee, ok := ferr.(*EnvelopeError); !ok || ee.Kind != "comment-unattached" {
			t.Errorf("%s: expected comment-unattached refusal, got %v", rel, ferr)
		}
		if out != nil {
			t.Errorf("%s: fail-closed must return NO partial output, got %d bytes", rel, len(out))
		}
	}
}

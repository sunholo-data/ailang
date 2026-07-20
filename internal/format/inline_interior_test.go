package format

import (
	"os"
	"path/filepath"
	"testing"

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

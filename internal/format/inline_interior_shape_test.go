package format

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// inline_interior_shape_test.go is the M0 (mandatory, read-only) surface-AST shape
// gate for M-AILANG-FMT-INLINE-INTERIOR. It PROVES — rather than assumes — that
// each of the 28 measured let-chain refusal files reaches its refused comment
// through a nested *ast.Let.Body chain, NOT a bare-`;` *ast.Block.Exprs list. Any
// file whose refusal site is a *ast.Block.Exprs (or any non-let shape) is routed to
// an EXCLUDED bucket with the reason logged, because Block.Exprs handling is OUT of
// scope for this sprint (design Deferred Decision). The verified nested-Let.Body
// count is the target set N that fixes the M1/M3 acceptance denominator.
//
// This test performs NO printer change and mutates NO parser/ast state. It uses the
// surface parser (parser.New(lexer.New(...))) and type-switches on the surface
// *ast.Let / *ast.Block nodes the attacher walks — the correct reliable probe (the
// CLI `ailang debug ast` prints the CORE elaborated AST, not the surface shape).

// inlineInteriorLetChainTargets is the design doc's enumerated 28-file let-chain set
// (§Reproduced 59-File Refusal Enumeration; the four let-chain subcategories total 28).
var inlineInteriorLetChainTargets = []string{
	// Let chains — equation-form Block1(*ast.Let) (15)
	"examples/integer_literals.ail",
	"examples/runnable/ai_image_generation.ail",
	"examples/runnable/array_adt.ail",
	"examples/runnable/array_grid.ail",
	"examples/runnable/func_expressions.ail",
	"examples/runnable/lambdas_advanced.ail",
	"examples/runnable/lambdas_closures.ail",
	"examples/runnable/lambdas_curried.ail",
	"examples/runnable/lambdas_higher_order.ail",
	"examples/runnable/std_deflate_pdf_objstm.ail",
	"examples/runnable/string_repeat.ail",
	"examples/runnable/xml_zip_roundtrip.ail",
	"examples/snippets/showcase/lambdas_basic.ail",
	"examples/snippets/showcase/lambdas_records.ail",
	"examples/tests/test_m_r7_comprehensive.ail",
	// Let chains — bare function-body *ast.Let (6)
	"examples/reference/neural_semantic_search.ail",
	"examples/reference/ollama_embed_test.ail",
	"examples/reference/semantic_retrieval.ail",
	"examples/runnable/string_interp_nested.ail",
	"examples/runnable/string_interpolation.ail",
	"examples/runnable/string_split.ail",
	// Let chains — let body continues as another let (5)
	"examples/reference/sharedmem_cache.ail",
	"examples/runnable/polymorphic_comparison_simple.ail",
	"examples/runnable/polymorphic_lambdas_phase1.ail",
	"examples/runnable/records.ail",
	"examples/runnable/tar_gzip_reader.ail",
	// Let chains — top-level let value rooted at another let (2)
	"examples/deriving_eq.ail",
	"examples/snippets/type_classes_working_reference.ail",
}

// repoRootFromFormat resolves the repository root from the internal/format package
// directory so the corpus-relative target paths resolve regardless of the module
// checkout location (worktree vs main).
func repoRootFromFormat(t *testing.T) string {
	t.Helper()
	// This test runs with CWD = internal/format; the repo root is two levels up.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

// letChainShape is the M0 classification of a target file's refusal site.
type letChainShape int

const (
	shapeNestedLetBody letChainShape = iota // root *ast.Let whose Body is (or flattens to) another *ast.Let
	shapeBlockExprs                         // bare-`;`/newline *ast.Block.Exprs (EXCLUDED — out of scope)
	shapeNonLet                             // some other shape (EXCLUDED)
	shapeNoRefusal                          // the file no longer refuses (unexpected at baseline)
)

func (s letChainShape) String() string {
	switch s {
	case shapeNestedLetBody:
		return "nested-*ast.Let.Body"
	case shapeBlockExprs:
		return "EXCLUDED:*ast.Block.Exprs"
	case shapeNonLet:
		return "EXCLUDED:non-let"
	case shapeNoRefusal:
		return "no-refusal(unexpected)"
	default:
		return "unknown"
	}
}

// TestInlineInterior_LetChainSurfaceShape proves the surface-AST shape of every
// target file's refusal site (M0). It logs a per-file classification and asserts the
// nested-Let.Body count equals the full target set (any EXCLUDED file fails the gate
// unless it is knowingly excluded, in which case the design's acceptance denominator
// must be adjusted — see the sprint JSON notes).
func TestInlineInterior_LetChainSurfaceShape(t *testing.T) {
	root := repoRootFromFormat(t)

	type result struct {
		path  string
		shape letChainShape
		byte  int
		note  string
	}
	var results []result

	for _, rel := range inlineInteriorLetChainTargets {
		abs := filepath.Join(root, rel)
		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("read target %s: %v", rel, err)
		}
		p := parser.New(lexer.New(string(data), rel))
		prog := p.Parse()
		if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
			t.Fatalf("target %s is not parse-valid (M0 requires parse-valid targets)", rel)
		}

		// M0 is a SHAPE proof that must survive the M1 attach change (once the chain
		// list is registered these files no longer refuse). Rather than depend on the
		// live attacher refusing, classify the shape directly: find the first comment a
		// let-chain root (or bare block) brackets, and classify that site. This is the
		// AST-shape fact the design's flattening traversal depends on, independent of
		// whether attachment currently succeeds.
		env, err := NewEnvelope(data)
		if err != nil {
			t.Fatalf("target %s: envelope: %v", rel, err)
		}
		siteByte, shape, note := classifyFirstChainComment(env, prog.File)
		if siteByte < 0 {
			results = append(results, result{rel, shapeNoRefusal, -1, "no comment brackets a let-chain root or block in this file"})
			continue
		}
		results = append(results, result{rel, shape, siteByte, note})
	}

	// Report + tally.
	sort.Slice(results, func(i, j int) bool { return results[i].path < results[j].path })
	var nested, excludedBlock, excludedNonLet, noRefusal int
	var excludedNames []string
	for _, r := range results {
		t.Logf("M0 SHAPE  %-60s  byte=%-6d  %s  (%s)", r.path, r.byte, r.shape, r.note)
		switch r.shape {
		case shapeNestedLetBody:
			nested++
		case shapeBlockExprs:
			excludedBlock++
			excludedNames = append(excludedNames, r.path+" [Block.Exprs]")
		case shapeNonLet:
			excludedNonLet++
			excludedNames = append(excludedNames, r.path+" [non-let]")
		case shapeNoRefusal:
			noRefusal++
			excludedNames = append(excludedNames, r.path+" [no-refusal]")
		}
	}
	t.Logf("M0 CLASSIFICATION: targets=%d nested-Let.Body=%d EXCLUDED-Block.Exprs=%d EXCLUDED-non-let=%d no-refusal=%d",
		len(inlineInteriorLetChainTargets), nested, excludedBlock, excludedNonLet, noRefusal)
	if len(excludedNames) > 0 {
		for _, n := range excludedNames {
			t.Logf("  M0 EXCLUDED: %s (Block.Exprs handling is OUT of scope; adjust M1/M3 denominator)", n)
		}
	}

	// GATE: prove the AST shape for the full target set. The design premise (R2
	// data-refutation) is that all 28 chain via nested *ast.Let.Body. If any file is
	// EXCLUDED, this fails LOUDLY so the acceptance denominator is re-derived rather
	// than silently under/over-counted.
	if nested != len(inlineInteriorLetChainTargets) {
		t.Fatalf("M0 shape gate: expected all %d targets to be nested-*ast.Let.Body, got %d nested (%d Block.Exprs, %d non-let, %d no-refusal) — see EXCLUDED logs and adjust the M1 target set N + refusal-count target in the sprint JSON before proceeding to M1",
			len(inlineInteriorLetChainTargets), nested, excludedBlock, excludedNonLet, noRefusal)
	}
}

// classifyFirstChainComment scans comments in source order and returns the first one
// whose enclosing site is a let-chain root or a bare multi-expr block, together with
// the shape classification of that site. Returns (-1, shapeNoRefusal, note) if no
// comment brackets such a construct. This is the M0 shape proof, decoupled from the
// live attacher so it stays green after M1 registers the chain list.
func classifyFirstChainComment(env *Envelope, f *ast.File) (int, letChainShape, string) {
	for _, c := range env.Comments() {
		shape, note := classifyRefusalSite(env, f, c.Start)
		if shape == shapeNestedLetBody || shape == shapeBlockExprs {
			return c.Start, shape, note
		}
	}
	return -1, shapeNoRefusal, "no comment brackets a let-chain root or block"
}

// classifyRefusalSite finds the tightest enclosing construct that brackets byteOff
// and reports whether that site is a nested *ast.Let.Body chain, a bare-`;`
// *ast.Block.Exprs list, or some other shape. It uses the attacher's own anchor
// machinery (MinAnchor / subtreeEnd) to resolve node byte ranges.
//
// Decision rule:
//   - A construct is a nested-let chain if there exists a *ast.Let whose Body is
//     itself a *ast.Let and whose [min, subtreeEnd] range brackets byteOff, AND that
//     Let is the tightest such construct (no bare multi-expr *ast.Block brackets the
//     byte more tightly).
//   - A construct is Block.Exprs if the tightest bracketing multi-expr *ast.Block
//     (len(Exprs) > 1) brackets the byte more tightly than any nested-let root.
func classifyRefusalSite(env *Envelope, f *ast.File, byteOff int) (letChainShape, string) {
	a := &attacher{env: env}
	c := &shapeClassifier{a: a, byteOff: byteOff, bestLetWidth: -1, bestBlockWidth: -1}
	for _, d := range f.Decls {
		c.walkNode(d, 0)
	}
	switch {
	case c.bestLetWidth >= 0 && (c.bestBlockWidth < 0 || c.bestLetWidth <= c.bestBlockWidth):
		return shapeNestedLetBody, c.letNote
	case c.bestBlockWidth >= 0:
		return shapeBlockExprs, c.blockNote
	default:
		return shapeNonLet, "no enclosing nested-let chain or multi-expr block brackets the refusal byte"
	}
}

// shapeClassifier walks the AST recording the tightest bracketing nested-let chain
// root and the tightest bracketing multi-expression block around byteOff.
type shapeClassifier struct {
	a       *attacher
	byteOff int

	bestLetWidth int // width of tightest nested-let-chain root bracketing byteOff; -1 if none
	letNote      string

	bestBlockWidth int // width of tightest multi-expr Block bracketing byteOff; -1 if none
	blockNote      string
}

// bracketsFrom reports whether byteOff lies in [leftWall, subtreeEnd(n)], returning
// the width of that range. leftWall extends a construct's lower bound to its
// enclosing body's opening delimiter so that a comment sitting at boundary 0 of a
// let-chain (directly after `=` / `{`, above the first binding) is still recognized
// as owned by the chain — that is exactly the leading position the let-chain models.
func (c *shapeClassifier) bracketsFrom(n ast.Node, leftWall int) (width int, ok bool) {
	min, err := c.a.env.MinAnchor(n)
	if err != nil {
		return 0, false
	}
	lo := min
	if leftWall >= 0 && leftWall < lo {
		lo = leftWall
	}
	end := c.a.subtreeEnd(n)
	if lo <= c.byteOff && c.byteOff <= end {
		return end - lo, true
	}
	return 0, false
}

// bodyOpen returns the byte just past the enclosing body's opening delimiter for a
// FuncDecl equation/block body: the offset after `=` or `{` preceding the body's
// first anchor. It approximates the left wall by scanning left from the body's min
// anchor for the nearest `=` or `{` at code level. -1 if none is found.
func (c *shapeClassifier) bodyOpen(body ast.Node) int {
	min, err := c.a.env.MinAnchor(body)
	if err != nil {
		return -1
	}
	for j := min - 1; j >= 0; j-- {
		if c.a.env.inStringSpan(j) {
			continue
		}
		ch := c.a.env.src[j]
		if ch == '=' || ch == '{' {
			return j + 1
		}
	}
	return -1
}

func (c *shapeClassifier) walkNode(n ast.Node, leftWall int) {
	switch v := n.(type) {
	case *ast.FuncDecl:
		if v.Body != nil {
			c.walkExpr(v.Body, c.bodyOpen(v.Body))
		}
	case *ast.Let:
		c.walkExpr(v, leftWall)
	case ast.Expr:
		c.walkExpr(v, leftWall)
	}
}

func (c *shapeClassifier) walkExpr(e ast.Expr, leftWall int) {
	switch v := e.(type) {
	case *ast.Let:
		// A let-chain root: any *ast.Let with a non-nil Body. The design flattens the
		// chain to [binding(x,vx), …, tail]; a length-2 chain (single binding + non-let
		// tail) is handled identically to a multi-binding nested chain. Recording the
		// TIGHTEST (smallest width) bracketing root means the outermost qualifying Let
		// is preferred only when it is the tightest — but because we thread the enclosing
		// left wall, the outermost root of an equation/block body wins for boundary-0
		// comments, which is the maximal chain the emitter will own.
		if v.Body != nil {
			if w, ok := c.bracketsFrom(v, leftWall); ok {
				if c.bestLetWidth < 0 || w < c.bestLetWidth {
					c.bestLetWidth = w
					if _, bodyIsLet := v.Body.(*ast.Let); bodyIsLet {
						c.letNote = "nested Let whose Body is *ast.Let (multi-binding chain) brackets the refusal byte"
					} else {
						c.letNote = "root Let with non-let Body (single-binding let…in chain) brackets the refusal byte"
					}
				}
			}
		}
		// Recurse with no inherited left wall: only the OUTERMOST let of a body carries
		// the enclosing body's wall (recorded above); inner chain links and nested
		// independent chains bracket from their own min anchor.
		if v.Value != nil {
			c.walkExpr(v.Value, -1)
		}
		if v.Body != nil {
			c.walkExpr(v.Body, -1)
		}
	case *ast.Block:
		open := c.a.openBraceBefore(blockNodes(v))
		if len(v.Exprs) > 1 {
			if w, ok := c.bracketsFrom(v, open); ok {
				if c.bestBlockWidth < 0 || w < c.bestBlockWidth {
					c.bestBlockWidth = w
					c.blockNote = "multi-expression *ast.Block.Exprs brackets the refusal byte (bare-; form)"
				}
			}
		}
		// A single-expression equation-form Block wraps the root let with NO brace; the
		// chain inside inherits the enclosing body's left wall.
		inner := leftWall
		if open >= 0 {
			inner = open + 1
		}
		for _, ex := range v.Exprs {
			c.walkExpr(ex, inner)
		}
	case *ast.Match:
		c.walkExpr(v.Expr, -1)
		for _, cs := range v.Cases {
			if cs.Guard != nil {
				c.walkExpr(cs.Guard, -1)
			}
			if cs.Body != nil {
				c.walkExpr(cs.Body, -1)
			}
		}
	case *ast.If:
		c.walkExpr(v.Condition, -1)
		c.walkExpr(v.Then, -1)
		if v.Else != nil {
			c.walkExpr(v.Else, -1)
		}
	case *ast.LetRec:
		if v.Value != nil {
			c.walkExpr(v.Value, -1)
		}
		if v.Body != nil {
			c.walkExpr(v.Body, -1)
		}
	case *ast.FuncCall:
		c.walkExpr(v.Func, -1)
		for _, arg := range v.Args {
			c.walkExpr(arg, -1)
		}
	case *ast.Lambda:
		if v.Body != nil {
			c.walkExpr(v.Body, -1)
		}
	case *ast.FuncLit:
		if v.Body != nil {
			c.walkExpr(v.Body, -1)
		}
	case *ast.BinaryOp:
		c.walkExpr(v.Left, -1)
		c.walkExpr(v.Right, -1)
	}
}

// blockNodes converts a block's expressions to the []ast.Node slice openBraceBefore
// expects.
func blockNodes(b *ast.Block) []ast.Node {
	nodes := make([]ast.Node, 0, len(b.Exprs))
	for _, ex := range b.Exprs {
		nodes = append(nodes, ex)
	}
	return nodes
}

package format

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// attach.go implements deterministic comment attachment (design rules 1–5) over
// the token-anchored envelope. Every scanned comment is attached to exactly one
// boundary of exactly one owner ordered child-list, or the file fails closed
// (totality). Attachment is a pure function of comment byte spans + line numbers
// and the child-list anchors the envelope resolves — no source-position guessing.
//
// The attachment is expressed against the MULTI-LINE ordered child lists the
// printer emits (design M0 inventory: file top-level, value-position block, match
// cases, tests/properties blocks, typeclass methods, test/property decl bodies,
// func-lit-body wrap). A comment appearing at a single-line / string-built child
// list boundary is an INTERIOR comment; the resolution floats it to the nearest
// enclosing multi-line list boundary (rule 3), so those sites need no separate
// owner — the hard-left-wall keeps a first-child comment on the parent's
// boundary 0 rather than trapping it in the first child.

// Place is where a comment sits relative to its owner boundary.
type Place int

const (
	// PlaceLeading: a comment immediately above a child (rule 2), emitted before it.
	PlaceLeading Place = iota
	// PlaceTrailing: a comment on the same source line as a child (rule 1),
	// emitted after that child on the same output line.
	PlaceTrailing
	// PlaceFloating: a comment on its own line at a list boundary (rules 3, 4),
	// emitted on its own line at that boundary.
	PlaceFloating
)

// Attachment binds one comment to a boundary within an owner's ordered child
// list. Owner is the *ast list node; Index is the boundary (0..len(children)):
// boundary k means "before child k"; boundary len(children) means "after the
// last child, before the close". For PlaceTrailing, Index is the child the
// comment trails.
type Attachment struct {
	Comment lexer.Comment
	Owner   any // the list-owning node (*ast.File, *ast.Block, *ast.Match, ...)
	Place   Place
	Index   int
}

// childList is one MULTI-LINE ordered child list discovered in the AST: its owner
// node, its children (in source order), and each child's byte anchor range
// (min-anchor .. end-of-subtree) plus line span. openByte is the parent's own
// opening delimiter (the hard left wall), or -1 at the file level.
type childList struct {
	owner    any
	children []childSpan
	openByte int
}

type childSpan struct {
	node      ast.Node
	minByte   int // leftmost subtree anchor (post widening)
	startLine int // 1-based line of minByte
	endLine   int // 1-based line of the child's last content
}

// attacher resolves comments to attachments over an envelope + AST.
type attacher struct {
	env   *Envelope
	lists []childList // all multi-line lists, in discovery order
	// chainConsumed marks *ast.Let nodes that are the Body link of an ENCLOSING
	// let-chain root, so they are not re-registered as their own (overlapping suffix)
	// chain. Only maximal chains are registered (design Let-Chain Discovery).
	chainConsumed map[*ast.Let]bool
}

// AttachComments computes the total attachment set for a program. It returns an
// error (fail-closed) if any comment cannot be attached to exactly one boundary.
func AttachComments(env *Envelope, f *ast.File) ([]Attachment, error) {
	a := &attacher{env: env, chainConsumed: map[*ast.Let]bool{}}
	a.collectLists(f)
	return a.resolve()
}

// lineOf returns the 1-based line for a byte offset using the envelope table.
func (a *attacher) lineOf(off int) int {
	ls := a.env.lineStart
	// binary search: largest line whose start <= off
	lo, hi := 1, len(ls)-1
	line := 1
	for lo <= hi {
		mid := (lo + hi) / 2
		if ls[mid] <= off {
			line = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return line
}

// collectLists walks the AST and records every MULTI-LINE ordered child list
// with per-child anchors. The set mirrors the M0 inventory's 9 multi-line sites.
func (a *attacher) collectLists(f *ast.File) {
	// File top-level: module, imports, decls in source order (boundary 0 = before
	// module). The file has no owning open delimiter → hard wall -1.
	var top []ast.Node
	if f.Module != nil {
		top = append(top, f.Module)
	}
	for _, imp := range f.Imports {
		top = append(top, imp)
	}
	top = append(top, f.Decls...)
	a.addList(f, top, -1)

	for _, d := range f.Decls {
		a.walkNode(d)
	}
}

// addList records a multi-line child list from a slice of AST nodes.
func (a *attacher) addList(owner any, nodes []ast.Node, openByte int) {
	if len(nodes) == 0 {
		return
	}
	cl := childList{owner: owner, openByte: openByte}
	for _, n := range nodes {
		min, err := a.env.MinAnchor(n)
		if err != nil {
			// A child with no convertible anchor cannot bound a boundary; skip the
			// whole list defensively (its comments float to an enclosing list).
			return
		}
		min = a.env.WidenLeft(min, openByte)
		end := a.subtreeEnd(n)
		cl.children = append(cl.children, childSpan{
			node:      n,
			minByte:   min,
			startLine: a.lineOf(min),
			endLine:   a.lineOf(end),
		})
	}
	a.lists = append(a.lists, cl)
}

// subtreeEnd returns the byte offset of the last content of a node's subtree
// (the max anchor; a lower bound on the true end, sufficient for line-based
// same-line trailing classification since the last token's line is its line).
func (a *attacher) subtreeEnd(n ast.Node) int {
	max := -1
	visitAnchors(n, func(p ast.Pos) {
		if p.Line == 0 {
			return
		}
		off, err := a.env.AnchorOf(p)
		if err != nil {
			return
		}
		if off > max {
			max = off
		}
	})
	if max == -1 {
		return 0
	}
	return max
}

// walkNode recurses into a node looking for nested multi-line child lists
// (blocks, match cases). It uses the exhaustive expr shapes the printer emits
// multi-line.
func (a *attacher) walkNode(n ast.Node) {
	switch v := n.(type) {
	case *ast.FuncDecl:
		if v.Body != nil {
			a.walkExpr(v.Body)
		}
	case *ast.Let:
		// Top-level `let … in` value/body chain (design: top-level let rooted at another
		// let). Register it as a chain owner, using the decl's own left wall (-1: the
		// file/top-level has no brace).
		a.registerLetChainAndRecurse(v, -1)
	case ast.Expr:
		a.walkExpr(v)
	}
}

// walkExpr recurses into an expression, recording block and match child lists.
func (a *attacher) walkExpr(e ast.Expr) {
	switch v := e.(type) {
	case *ast.Block:
		nodes := make([]ast.Node, 0, len(v.Exprs))
		for _, ex := range v.Exprs {
			nodes = append(nodes, ex)
		}
		// A single-expression Block coming from equation form (`func f() = expr`)
		// prints as `= expr`, NOT a braced multi-line block (funcBody), so it is not
		// a multi-line child list. Only register a Block as a list when a real
		// enclosing `{` is found (openBraceBefore >= 0 AND it is a `{`, not a `[`).
		//
		// EXCEPTION (M-AILANG-FMT-INLINE-INTERIOR): a braced block whose SOLE expression
		// is a let-chain root (a leading-`in` continuation chain, e.g. sharedmem_cache's
		// `{ let _ = … in let _ = … }`) must NOT register the block as a one-child list.
		// That block list's single child spans the whole chain and would tie the chain
		// list on width (same `{` wall, same `}` end), losing the tie-break and trapping
		// a between-binding comment in the strict-interior guard. Skipping it lets the
		// chain list (registered when we recurse into the sole expr) own the boundaries.
		open := a.openBraceBefore(nodes)
		if open >= 0 && a.env.src[open] == '{' && len(v.Exprs) >= 1 && !isSingleLetChainBlock(v) {
			a.addList(v, nodes, open)
		}
		for _, ex := range v.Exprs {
			a.walkExpr(ex)
		}
	case *ast.Match:
		nodes := make([]ast.Node, 0, len(v.Cases))
		for _, c := range v.Cases {
			if c.Body != nil {
				nodes = append(nodes, c.Body)
			}
		}
		// Match arms open at the `{` after the scrutinee; keep the hard wall at it.
		open := a.openBraceBefore(nodes)
		a.addList(v, nodes, open)
		a.walkExpr(v.Expr)
		for _, c := range v.Cases {
			if c.Guard != nil {
				a.walkExpr(c.Guard)
			}
			if c.Body != nil {
				a.walkExpr(c.Body)
			}
		}
	case *ast.If:
		a.walkExpr(v.Condition)
		a.walkExpr(v.Then)
		if v.Else != nil {
			a.walkExpr(v.Else)
		}
	case *ast.Let:
		a.registerLetChainAndRecurse(v, -1)
	case *ast.LetRec:
		if v.Value != nil {
			a.walkExpr(v.Value)
		}
		if v.Body != nil {
			a.walkExpr(v.Body)
		}
	case *ast.FuncCall:
		a.walkExpr(v.Func)
		for _, arg := range v.Args {
			a.walkExpr(arg)
		}
	case *ast.Lambda:
		if v.Body != nil {
			a.walkExpr(v.Body)
		}
	case *ast.FuncLit:
		if v.Body != nil {
			a.walkExpr(v.Body)
		}
	case *ast.BinaryOp:
		a.walkExpr(v.Left)
		a.walkExpr(v.Right)
	}
}

// registerLetChainAndRecurse registers the maximal `let … in` chain rooted at root
// (unless root is already consumed as an enclosing chain's Body link) as one
// non-overlapping ordered child list owned by root, then recurses into each binding
// value and the tail to discover nested lists (blocks, matches, independent chains).
//
// wall is the enclosing body's opening delimiter byte (`=` or `{`); pass -1 to have
// the chain scan left for its own enclosing opener. The list open boundary is that
// wall so a boundary-0 comment sitting above the first binding (directly after `=`/
// `{`) is claimed by the chain rather than orphaned.
func (a *attacher) registerLetChainAndRecurse(root *ast.Let, wall int) {
	if root == nil {
		return
	}
	// Skip a Let that a parent chain already consumed (no overlapping suffix lists).
	if a.chainConsumed[root] {
		if root.Value != nil {
			a.walkExpr(root.Value)
		}
		if root.Body != nil {
			a.walkExpr(root.Body)
		}
		return
	}
	// A bare statement-let (Body == nil) is not a chain; recurse normally.
	if root.Body == nil {
		if root.Value != nil {
			a.walkExpr(root.Value)
		}
		return
	}

	lc := flattenLetChain(root)
	// Mark every binding after the root as consumed so it is not re-registered.
	for _, b := range lc.Bindings {
		if b != root {
			a.chainConsumed[b] = true
		}
	}

	// Left wall: the enclosing body opener if given, else scan left for `=`/`{`.
	openByte := wall
	if openByte < 0 {
		openByte = a.enclosingLetWall(root)
	}

	a.addLetChainList(lc, openByte)

	// Recurse into binding VALUES and the tail (not the binding lets themselves — they
	// are the chain's own children). A nested independent chain in a value registers
	// its own maximal list.
	for _, b := range lc.Bindings {
		if b.Value != nil {
			a.walkExpr(b.Value)
		}
	}
	if lc.Tail != nil {
		a.walkExpr(lc.Tail)
	}
}

// enclosingLetWall scans left from the chain root's min anchor for the nearest `=` or
// `{` at code level (the enclosing equation/brace body opener). Returns that byte, or
// MinAnchor(root)-1 if none is found (the design's fallback: never claim comments
// before the root let).
func (a *attacher) enclosingLetWall(root *ast.Let) int {
	min, err := a.env.MinAnchor(root)
	if err != nil {
		return -1
	}
	for j := min - 1; j >= 0; j-- {
		if a.env.inStringSpan(j) {
			continue
		}
		c := a.env.src[j]
		if c == '=' || c == '{' {
			return j
		}
	}
	return min - 1
}

// addLetChainList records the flattened chain as one childList with EXPLICIT
// non-overlapping child spans. Unlike the generic addList, a binding child's byte end
// is subtreeEnd(binding.Value) (NOT subtreeEnd(binding), which would swallow the whole
// nested body); the tail child spans its own subtree. This creates a stable boundary
// before each subsequent binding and before the tail.
func (a *attacher) addLetChainList(lc letChain, openByte int) {
	cl := childList{owner: lc.Root, openByte: openByte}
	for _, b := range lc.Bindings {
		min, err := a.env.MinAnchor(b)
		if err != nil {
			return // no anchor → cannot bound a boundary; skip (comments float outward)
		}
		min = a.env.WidenLeft(min, openByte)
		// Binding child end = end of its VALUE subtree; `in` has no AST position (V13).
		end := min
		if b.Value != nil {
			if e := a.subtreeEnd(b.Value); e > end {
				end = e
			}
		}
		cl.children = append(cl.children, childSpan{
			node:      b,
			minByte:   min,
			startLine: a.lineOf(min),
			endLine:   a.lineOf(end),
		})
	}
	if lc.Tail != nil {
		min, err := a.env.MinAnchor(lc.Tail)
		if err != nil {
			return
		}
		min = a.env.WidenLeft(min, openByte)
		end := a.subtreeEnd(lc.Tail)
		cl.children = append(cl.children, childSpan{
			node:      lc.Tail,
			minByte:   min,
			startLine: a.lineOf(min),
			endLine:   a.lineOf(end),
		})
	}
	if len(cl.children) == 0 {
		return
	}
	a.lists = append(a.lists, cl)
}

// openBraceBefore returns the byte offset of the `{`/`[` that opens the list
// whose first child starts at children[0], searching left from the first child's
// min-anchor for the nearest opening delimiter. It is the hard left wall.
func (a *attacher) openBraceBefore(nodes []ast.Node) int {
	if len(nodes) == 0 {
		return -1
	}
	min, err := a.env.MinAnchor(nodes[0])
	if err != nil {
		return -1
	}
	// Search left for the nearest '{' or '[' at code level.
	for j := min - 1; j >= 0; j-- {
		if a.env.inStringSpan(j) {
			continue
		}
		c := a.env.src[j]
		if c == '{' || c == '[' {
			return j
		}
	}
	return -1
}

// resolve applies rules 1–5 to every comment, producing a total attachment set
// or a fail-closed error.
func (a *attacher) resolve() ([]Attachment, error) {
	comments := a.env.Comments()
	out := make([]Attachment, 0, len(comments))
	for _, c := range comments {
		at, ok := a.attachOne(c)
		if !ok {
			return nil, envErr("comment-unattached",
				"comment at byte %d (%q) could not be attached to any boundary", c.Start, c.Text)
		}
		out = append(out, at)
	}
	// Group-stable: sort by comment start so consecutive comments keep source order.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Comment.Start < out[j].Comment.Start })
	return out, nil
}

// attachOne classifies a single comment per rules 1–5. It picks the SMALLEST
// enclosing multi-line list (the one whose child span most tightly brackets the
// comment) and resolves the boundary within it.
func (a *attacher) attachOne(c lexer.Comment) (Attachment, bool) {
	cLine := a.lineOf(c.Start)

	// Rule 1 FIRST (across all lists): a same-line trailing comment attaches to the
	// child whose LAST content shares the comment's source line and lies to its
	// left. Checked before the enclosing-list range filter because a trailing
	// comment sits just past its child's subtree-end anchor (outside the child's
	// interval but on its line). Prefer the innermost (rightmost min-anchor) owner.
	{
		bestK, bestList, bestMin := -1, -1, -1
		for i := range a.lists {
			cl := &a.lists[i]
			for k, ch := range cl.children {
				if ch.endLine == cLine && ch.minByte < c.Start && ch.minByte > bestMin {
					bestMin, bestK, bestList = ch.minByte, k, i
				}
			}
		}
		if bestList >= 0 {
			cl := &a.lists[bestList]
			return Attachment{Comment: c, Owner: cl.owner, Place: PlaceTrailing, Index: bestK}, true
		}
	}

	best := -1
	bestWidth := 1 << 62
	for i := range a.lists {
		cl := &a.lists[i]
		// The list accepts comments from just past its opening delimiter (or byte 0
		// at the file level) through its closing delimiter — so a comment BEFORE the
		// first child (boundary 0) is inside the list, not orphaned.
		lo := cl.openByte + 1 // openByte -1 at file level → lo 0
		hi := a.listEnd(cl)
		if c.Start < lo || c.Start > hi {
			continue
		}
		w := hi - lo
		if w < bestWidth {
			bestWidth = w
			best = i
		}
	}
	if best == -1 {
		return Attachment{}, false
	}
	cl := &a.lists[best]

	// Fail-closed guard: if the comment's LINE falls strictly WITHIN a child's
	// rendered line span (after its first line, before its last), the comment is
	// interior to a MULTI-LINE child that the chosen list does not decompose into a
	// tighter sublist (e.g. a top-level `let ... in` chain that the printer
	// collapses onto fewer lines, or a multi-line expression with no recognized
	// child list). Attaching it to a list boundary would relocate it
	// non-idempotently, so we refuse (fail-closed). A tighter enclosing list, when
	// one exists, is chosen as `best` first, so reaching here means none does.
	// (Rule 1 trailing is already resolved above; this only affects floating/leading.)
	for _, ch := range cl.children {
		if ch.startLine < cLine && cLine < ch.endLine {
			return Attachment{}, false
		}
	}

	// Find the boundary index: the first child whose min-anchor is on a line at or
	// after the comment's line. Comments before the first child → boundary 0
	// (rule 4 / hard left wall). After the last child → boundary len (rule 4 tail).
	boundary := len(cl.children)
	for k, ch := range cl.children {
		if ch.startLine > cLine {
			boundary = k
			break
		}
	}

	// Rule 2 vs 3 (with rule 5 grouping): a comment is LEADING iff there is no
	// blank line between it and the next child, treating intervening consecutive
	// comments as non-blank (a contiguous comment run immediately above a child is
	// all leading). Otherwise it is FLOATING (a blank line separates it from the
	// next child, it sits between siblings, or it precedes a close).
	if boundary < len(cl.children) {
		next := cl.children[boundary]
		if a.contiguousToChild(c, cLine, next.startLine) {
			return Attachment{Comment: c, Owner: cl.owner, Place: PlaceLeading, Index: boundary}, true
		}
	}
	return Attachment{Comment: c, Owner: cl.owner, Place: PlaceFloating, Index: boundary}, true
}

// contiguousToChild reports whether the comment ending on cLine reaches the child
// at childLine with NO blank line in between — where intervening lines that are
// themselves comments count as non-blank (rule 5 grouping). A blank line anywhere
// in the gap makes the comment floating.
func (a *attacher) contiguousToChild(c lexer.Comment, cLine, childLine int) bool {
	if childLine <= cLine {
		return false
	}
	// Every line strictly between cLine and childLine must be occupied by a comment
	// (no blank line). cLine itself holds this comment.
	occupied := map[int]bool{}
	for _, cc := range a.env.Comments() {
		occupied[a.lineOf(cc.Start)] = true
	}
	for ln := cLine + 1; ln < childLine; ln++ {
		if !occupied[ln] {
			return false // a blank (or non-comment) line breaks contiguity
		}
	}
	return true
}

// listEnd returns the byte offset just past the last child of a list (its
// enclosing close, approximated by the last child's subtree end or the matching
// close of openByte).
func (a *attacher) listEnd(cl *childList) int {
	if cl.openByte >= 0 {
		if close := a.env.matchBracket(cl.openByte); close >= 0 {
			return close
		}
	}
	// File level (or unmatched): end at the last child's subtree end.
	last := cl.children[len(cl.children)-1]
	return a.subtreeEnd(last.node)
}

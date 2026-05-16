package lsp

import (
	"sort"
	"sync"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// PositionIndex answers "what AILANG identifier is at this cursor position?"
// for an LSP hover/definition/references handler. It's built from a parsed
// *ast.File and walks every Identifier node, recording each one's start
// position and computing its end as start + len(name) (since AILANG idents
// don't span lines).
//
// Lookup is O(log N) via binary search over (line, startCol) sorted entries.
//
// MVP scope: only Identifiers are indexed. Function/Block/Let nodes carry
// only Pos (start), not Span (end), so we can't reliably bound them.
// Tracking those is M-PARSER-POSITION-FIDELITY territory; for hover and
// definition the leaf-Identifier resolution is enough.
type PositionIndex struct {
	entries []indexEntry
}

type indexEntry struct {
	line     int // 1-indexed
	startCol int // 1-indexed
	endCol   int // 1-indexed, exclusive
	ident    *ast.Identifier
}

// BuildPositionIndex parses src and returns an index of every identifier
// in the file. Returns nil if parsing fails outright (the file might still
// have a partial AST; we surface diagnostics through the M2 path instead).
func BuildPositionIndex(filename string, src string) *PositionIndex {
	l := lexer.New(src, filename)
	p := parser.New(l)
	file := p.ParseFile()
	if file == nil {
		return nil
	}
	return BuildPositionIndexFromFile(file)
}

// BuildPositionIndexFromFile builds the index from an already-parsed file.
// Used by tests and by the server when it already has the AST in hand.
func BuildPositionIndexFromFile(file *ast.File) *PositionIndex {
	idx := &PositionIndex{}
	idx.collect(file)
	idx.sort()
	return idx
}

// Lookup returns the Identifier whose extent contains (line, col), or nil
// if no Identifier is at that cursor. Both line and col are 1-indexed
// (matching ast.Pos), but LSP positions are 0-indexed — callers must
// translate.
func (px *PositionIndex) Lookup(line, col int) *ast.Identifier {
	if px == nil || len(px.entries) == 0 {
		return nil
	}
	// Binary search for the first entry whose line >= target. Then linear-
	// scan the (typically few) entries on that line for the one containing
	// col. The predicate must monotonically flip from false to true; entries
	// on the target line are the first "true" group.
	i := sort.Search(len(px.entries), func(i int) bool {
		return px.entries[i].line >= line
	})
	for i < len(px.entries) && px.entries[i].line == line {
		e := px.entries[i]
		if col >= e.startCol && col < e.endCol {
			return e.ident
		}
		i++
	}
	return nil
}

// AllIdents returns every indexed identifier in source-order. Used by M4
// references walks; declared-and-exported here so M4 doesn't have to
// re-walk the AST.
func (px *PositionIndex) AllIdents() []*ast.Identifier {
	if px == nil {
		return nil
	}
	out := make([]*ast.Identifier, 0, len(px.entries))
	for _, e := range px.entries {
		out = append(out, e.ident)
	}
	return out
}

func (px *PositionIndex) collect(file *ast.File) {
	if file == nil {
		return
	}
	// f.Decls is deprecated and overlaps with f.Funcs/f.Statements (per
	// the comment in ast.File). Walk only the canonical fields to avoid
	// double-counting.
	for _, fd := range file.Funcs {
		if fd != nil {
			px.walkExpr(fd.Body)
		}
	}
	for _, n := range file.Statements {
		if e, ok := n.(ast.Expr); ok {
			px.walkExpr(e)
		}
	}
}

func (px *PositionIndex) walkExpr(e ast.Expr) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *ast.Identifier:
		px.add(n)
	case *ast.Literal:
		// leaf, no idents
	case *ast.BinaryOp:
		px.walkExpr(n.Left)
		px.walkExpr(n.Right)
	case *ast.UnaryOp:
		px.walkExpr(n.Expr)
	case *ast.Lambda:
		px.walkExpr(n.Body)
	case *ast.FuncLit:
		px.walkExpr(n.Body)
	case *ast.FuncCall:
		px.walkExpr(n.Func)
		for _, a := range n.Args {
			px.walkExpr(a)
		}
	case *ast.Let:
		px.walkExpr(n.Value)
		px.walkExpr(n.Body)
	case *ast.LetRec:
		px.walkExpr(n.Body)
	case *ast.Block:
		for _, child := range n.Exprs {
			px.walkExpr(child)
		}
	case *ast.If:
		px.walkExpr(n.Condition)
		px.walkExpr(n.Then)
		px.walkExpr(n.Else)
	case *ast.Match:
		px.walkExpr(n.Expr)
		for _, c := range n.Cases {
			px.walkExpr(c.Body)
		}
	case *ast.List:
		for _, child := range n.Elements {
			px.walkExpr(child)
		}
	case *ast.Tuple:
		for _, child := range n.Elements {
			px.walkExpr(child)
		}
	case *ast.Record:
		for _, f := range n.Fields {
			px.walkExpr(f.Value)
		}
	case *ast.RecordAccess:
		px.walkExpr(n.Record)
	case *ast.RecordUpdate:
		px.walkExpr(n.Base)
		for _, f := range n.Fields {
			px.walkExpr(f.Value)
		}
	}
}

func (px *PositionIndex) add(id *ast.Identifier) {
	if id == nil || id.Pos.Line <= 0 || id.Pos.Column <= 0 {
		return
	}
	startCol := id.Pos.Column
	endCol := startCol + len(id.Name)
	if endCol == startCol {
		endCol = startCol + 1
	}
	px.entries = append(px.entries, indexEntry{
		line:     id.Pos.Line,
		startCol: startCol,
		endCol:   endCol,
		ident:    id,
	})
}

func (px *PositionIndex) sort() {
	sort.Slice(px.entries, func(i, j int) bool {
		if px.entries[i].line != px.entries[j].line {
			return px.entries[i].line < px.entries[j].line
		}
		return px.entries[i].startCol < px.entries[j].startCol
	})
}

// indexCache stores the most recent PositionIndex per URI so hover/
// definition/references handlers don't re-parse the file every request.
// Server holds one of these via sync.Map.
type indexCache struct {
	mu sync.RWMutex
	by map[string]*PositionIndex
}

func newIndexCache() *indexCache {
	return &indexCache{by: map[string]*PositionIndex{}}
}

func (c *indexCache) put(path string, idx *PositionIndex) {
	c.mu.Lock()
	c.by[path] = idx
	c.mu.Unlock()
}

func (c *indexCache) get(path string) *PositionIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.by[path]
}

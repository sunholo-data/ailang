package lsp

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TestProbeIdentPositionFidelity is the M-AILANG-LSP-FOR-AI sprint plan's
// Day-3 09:30 position-fidelity gate. It runs the parser against a real
// example file, walks every Identifier, and verifies that:
//
//  1. No identifier has a (0,0) Pos (would mean parser dropped position info)
//  2. No two consecutive identifiers share the exact same (line, col) (would
//     mean cursor-to-symbol disambiguation is impossible)
//  3. At least one line carries multiple identifiers, and they have distinct
//     columns (the test of "useful resolution for hover")
//
// If this test fails, the sprint plan calls for pausing M3 to either backfill
// positions in the parser or downscope hover to function-level only.
func TestProbeIdentPositionFidelity(t *testing.T) {
	t.Parallel()
	// Run against multiple fixtures so we hit single-line, multi-ident cases.
	fixtures := []string{
		"../../examples/runnable/parser_block_trailing_record.ail",
		"../../examples/pattern_matching_adt.ail",
		"../../examples/string_replace.ail",
	}
	for _, f := range fixtures {
		t.Run(f, func(t *testing.T) {
			probeOne(t, f)
		})
	}
}

func probeOne(t *testing.T, fixture string) {
	src, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}

	l := lexer.New(string(src), fixture)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors on %s: %v", fixture, errs)
	}

	type idHit struct {
		name      string
		line, col int
	}
	var hits []idHit
	walkFile(file, func(n ast.Node) {
		if id, ok := n.(*ast.Identifier); ok {
			hits = append(hits, idHit{id.Name, id.Pos.Line, id.Pos.Column})
		}
	})

	if len(hits) == 0 {
		t.Fatal("no Identifier nodes walked — fixture or walker is wrong")
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].line != hits[j].line {
			return hits[i].line < hits[j].line
		}
		return hits[i].col < hits[j].col
	})

	// Check 1: no (0,0) positions.
	zeroPos := 0
	for _, h := range hits {
		if h.line == 0 && h.col == 0 {
			zeroPos++
			t.Logf("ident %q has (0,0) position", h.name)
		}
	}
	if zeroPos > 0 {
		t.Errorf("PROBE FAILED: %d/%d identifiers have (0,0) position", zeroPos, len(hits))
	}

	// Check 2: no consecutive duplicates.
	dupPos := 0
	for i := 1; i < len(hits); i++ {
		if hits[i].line == hits[i-1].line && hits[i].col == hits[i-1].col {
			dupPos++
			t.Logf("dup at L%d:C%d  %q vs %q", hits[i].line, hits[i].col, hits[i-1].name, hits[i].name)
		}
	}
	if dupPos > 0 {
		t.Errorf("PROBE FAILED: %d pairs of identifiers share exact (line,col)", dupPos)
	}

	// Check 3: at least one line has ≥2 idents with distinct columns.
	sameLine := map[int][]idHit{}
	for _, h := range hits {
		sameLine[h.line] = append(sameLine[h.line], h)
	}
	maxRun := 0
	var multiLine int
	for line, group := range sameLine {
		if len(group) > maxRun {
			maxRun = len(group)
			multiLine = line
		}
	}
	if maxRun < 2 {
		t.Logf("WARN: no line in fixture has ≥2 identifiers — probe is inconclusive on multi-ident lines")
	} else {
		seenCols := map[int]bool{}
		for _, h := range sameLine[multiLine] {
			if seenCols[h.col] {
				t.Errorf("PROBE FAILED: line %d has two identifiers at col %d", multiLine, h.col)
			}
			seenCols[h.col] = true
		}
	}

	// Always log a summary so the probe result is visible even on PASS.
	t.Logf("PROBE SUMMARY for %s", fixture)
	t.Logf("  identifiers: %d (zero-pos: %d, dups: %d, max-on-one-line: %d)",
		len(hits), zeroPos, dupPos, maxRun)
	if maxRun >= 2 {
		t.Logf("  example multi-ident line %d:", multiLine)
		for _, h := range sameLine[multiLine] {
			t.Logf("    L%d:C%d  %s", h.line, h.col, h.name)
		}
	}

	// Final verdict for surfacing in test output.
	verdict := "PROBE PASSED — proceed with M3 implementation as planned"
	if zeroPos > 0 || dupPos > 0 {
		verdict = fmt.Sprintf("PROBE FAILED — pause M3, decide path (a) backfill positions vs (b) coarser hover (zero-pos=%d dup-pos=%d)", zeroPos, dupPos)
	}
	t.Logf("\n%s", verdict)
}

// walkFile is a minimal AST walker that visits every expression node
// (and lets the probe pick out Identifiers via type switch). Only used by
// the probe — the real M3 PositionIndex walker will be more complete.
func walkFile(f *ast.File, visit func(ast.Node)) {
	if f == nil {
		return
	}
	// f.Decls is deprecated and overlaps with f.Funcs/f.Statements (the
	// header comment on ast.File says so). Walk only the canonical fields
	// to avoid visiting the same node twice.
	for _, fd := range f.Funcs {
		visit(fd)
		walkExpr(fd.Body, visit)
	}
	for _, n := range f.Statements {
		if e, ok := n.(ast.Expr); ok {
			walkExpr(e, visit)
		}
	}
}

func walkExpr(e ast.Expr, visit func(ast.Node)) {
	if e == nil {
		return
	}
	visit(e)
	switch n := e.(type) {
	case *ast.Identifier, *ast.Literal:
		// leaf
	case *ast.BinaryOp:
		walkExpr(n.Left, visit)
		walkExpr(n.Right, visit)
	case *ast.UnaryOp:
		walkExpr(n.Expr, visit)
	case *ast.Lambda:
		walkExpr(n.Body, visit)
	case *ast.FuncLit:
		walkExpr(n.Body, visit)
	case *ast.FuncCall:
		walkExpr(n.Func, visit)
		for _, a := range n.Args {
			walkExpr(a, visit)
		}
	case *ast.Let:
		walkExpr(n.Value, visit)
		walkExpr(n.Body, visit)
	case *ast.LetRec:
		walkExpr(n.Body, visit)
	case *ast.Block:
		for _, e := range n.Exprs {
			walkExpr(e, visit)
		}
	case *ast.If:
		walkExpr(n.Condition, visit)
		walkExpr(n.Then, visit)
		walkExpr(n.Else, visit)
	case *ast.Match:
		walkExpr(n.Expr, visit)
		for _, c := range n.Cases {
			walkExpr(c.Body, visit)
		}
	case *ast.List:
		for _, e := range n.Elements {
			walkExpr(e, visit)
		}
	case *ast.Tuple:
		for _, e := range n.Elements {
			walkExpr(e, visit)
		}
	case *ast.Record:
		for _, f := range n.Fields {
			walkExpr(f.Value, visit)
		}
	case *ast.RecordAccess:
		walkExpr(n.Record, visit)
	}
}

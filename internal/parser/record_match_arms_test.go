package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseExpressionForTest parses an expression in a function context
// Returns the match expression from inside a wrapper function
func parseExpressionForTest(t *testing.T, exprInput string) ast.Expr {
	t.Helper()

	// Wrap expression in a minimal function to make it parseable
	input := `module test
func test() -> int = ` + exprInput

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("parse error: %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(program.File.Funcs))
	}

	// Function bodies using = syntax are wrapped in a Block for uniform handling
	// Unwrap to get the actual expression
	body := program.File.Funcs[0].Body
	if block, ok := body.(*ast.Block); ok && len(block.Exprs) == 1 {
		return block.Exprs[0]
	}
	return body
}

// TestRecordLiteralInMatchArm verifies that record literals parse correctly
// in match arms (not as blocks).
//
// Bug: Prior to fix, `match x { A => {f: 1} }` was parsed as a Block
// because parseCase() special-cased LBRACE to use parseBlockOrExpression().
//
// Fix: Apply same lookahead logic from parseRecordLiteral() to distinguish
// record literals ({field: value}) from blocks ({ expr; expr }).
func TestRecordLiteralInMatchArm(t *testing.T) {
	expr := parseExpressionForTest(t, `match x { A => {name: "test", value: 42} }`)

	// Verify it's a Match expression
	matchExpr, ok := expr.(*ast.Match)
	if !ok {
		t.Fatalf("expected *ast.Match, got %T", expr)
	}

	// Verify the case body is a Record (not a Block)
	if len(matchExpr.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(matchExpr.Cases))
	}

	caseBody := matchExpr.Cases[0].Body
	record, ok := caseBody.(*ast.Record)
	if !ok {
		t.Fatalf("expected case body to be *ast.Record, got %T (this is the bug - it was parsed as a Block)", caseBody)
	}

	// Verify record has expected fields
	if len(record.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(record.Fields))
	}

	if record.Fields[0].Name != "name" {
		t.Errorf("expected first field name 'name', got %q", record.Fields[0].Name)
	}
	if record.Fields[1].Name != "value" {
		t.Errorf("expected second field name 'value', got %q", record.Fields[1].Name)
	}
}

// TestNestedRecordInMatchArm verifies nested record literals work in match arms.
func TestNestedRecordInMatchArm(t *testing.T) {
	expr := parseExpressionForTest(t, `match x { A => {pos: {x: 1.0, y: 2.0}} }`)

	matchExpr, ok := expr.(*ast.Match)
	if !ok {
		t.Fatalf("expected *ast.Match, got %T", expr)
	}

	if len(matchExpr.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(matchExpr.Cases))
	}

	// Outer record
	outerRecord, ok := matchExpr.Cases[0].Body.(*ast.Record)
	if !ok {
		t.Fatalf("expected case body to be *ast.Record, got %T", matchExpr.Cases[0].Body)
	}

	if len(outerRecord.Fields) != 1 {
		t.Fatalf("expected 1 field in outer record, got %d", len(outerRecord.Fields))
	}

	// Inner record (nested)
	innerRecord, ok := outerRecord.Fields[0].Value.(*ast.Record)
	if !ok {
		t.Fatalf("expected nested field value to be *ast.Record, got %T", outerRecord.Fields[0].Value)
	}

	if len(innerRecord.Fields) != 2 {
		t.Fatalf("expected 2 fields in inner record, got %d", len(innerRecord.Fields))
	}
}

// TestBlockInMatchArmStillWorks verifies that blocks with semicolons still parse as blocks.
// This is the disambiguation: semicolons mean it's a block, not a record.
func TestBlockInMatchArmStillWorks(t *testing.T) {
	expr := parseExpressionForTest(t, `match x { A => { foo(); bar() } }`)

	matchExpr, ok := expr.(*ast.Match)
	if !ok {
		t.Fatalf("expected *ast.Match, got %T", expr)
	}

	if len(matchExpr.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(matchExpr.Cases))
	}

	// Should be a Block (has semicolon separator)
	block, ok := matchExpr.Cases[0].Body.(*ast.Block)
	if !ok {
		t.Fatalf("expected case body to be *ast.Block, got %T", matchExpr.Cases[0].Body)
	}

	if len(block.Exprs) != 2 {
		t.Fatalf("expected 2 expressions in block, got %d", len(block.Exprs))
	}
}

// TestEmptyBraceInMatchArmIsBlock verifies that {} in match arm is empty block (existing behavior).
func TestEmptyBraceInMatchArmIsBlock(t *testing.T) {
	expr := parseExpressionForTest(t, `match x { A => {} }`)

	matchExpr, ok := expr.(*ast.Match)
	if !ok {
		t.Fatalf("expected *ast.Match, got %T", expr)
	}

	if len(matchExpr.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(matchExpr.Cases))
	}

	// {} returns empty block (existing behavior - Block has 0 exprs)
	// OR it might return a Unit literal - check what parseRecordLiteral does
	body := matchExpr.Cases[0].Body

	// parseRecordLiteral returns Block{} for empty braces
	switch v := body.(type) {
	case *ast.Block:
		// Expected - empty block
		if len(v.Exprs) != 0 {
			t.Errorf("expected empty block, got %d expressions", len(v.Exprs))
		}
	case *ast.Literal:
		// Also acceptable if it's unit literal
		if v.Kind != ast.UnitLit {
			t.Errorf("expected UnitLit, got %v", v.Kind)
		}
	default:
		t.Fatalf("expected *ast.Block or *ast.Literal(Unit), got %T", body)
	}
}

// TestRecordUpdateInMatchArm verifies record update syntax works in match arms.
func TestRecordUpdateInMatchArm(t *testing.T) {
	expr := parseExpressionForTest(t, `match x { A => {base | field: 42} }`)

	matchExpr, ok := expr.(*ast.Match)
	if !ok {
		t.Fatalf("expected *ast.Match, got %T", expr)
	}

	if len(matchExpr.Cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(matchExpr.Cases))
	}

	// Should be a RecordUpdate
	update, ok := matchExpr.Cases[0].Body.(*ast.RecordUpdate)
	if !ok {
		t.Fatalf("expected case body to be *ast.RecordUpdate, got %T", matchExpr.Cases[0].Body)
	}

	if len(update.Fields) != 1 {
		t.Fatalf("expected 1 updated field, got %d", len(update.Fields))
	}
}

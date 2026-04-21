package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestGAP2_NewlineLPAREN tests that LPAREN on a new line is NOT parsed as a function call.
// Bug: `expr\n(next)` was incorrectly parsed as `expr(next)`.
// Fix: The parser now breaks the infix loop when LPAREN appears on a new line.
func TestGAP2_NewlineLPAREN(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedStmts int
		desc          string
	}{
		{
			name: "bare_identifier",
			code: `module test
let x = 1
x`,
			expectedStmts: 2, // Let + Identifier
			desc:          "bare identifier on new line should be separate statement",
		},
		{
			name: "parenthesized_identifier",
			code: `module test
let x = 1
(x)`,
			expectedStmts: 2, // Let + Identifier (inside parens)
			desc:          "parenthesized identifier on new line should be separate statement",
		},
		{
			name: "tuple_after_foldl",
			code: `module test
let sum = foldl(\acc x. acc + x, 0, [1, 2, 3])
(sum, 42)`,
			expectedStmts: 2, // Let + Tuple
			desc:          "tuple on new line should not be absorbed as foldl argument",
		},
		{
			name: "same_line_is_call",
			code: `module test
let f = \x. x
f(42)`,
			expectedStmts: 2, // Let + FuncCall (f(42) on same expression level)
			desc:          "function call syntax f(x) should still work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := lexer.New(tt.code, tt.name+".ail")
			p := New(lex)
			file := p.ParseFile()

			if len(p.Errors()) > 0 {
				for _, e := range p.Errors() {
					t.Errorf("Parse error: %v", e)
				}
				t.Fatalf("Parser errors in test case: %s", tt.desc)
			}

			if len(file.Statements) != tt.expectedStmts {
				t.Errorf("Expected %d statements, got %d", tt.expectedStmts, len(file.Statements))
				for i, stmt := range file.Statements {
					t.Logf("  [%d] %T", i, stmt)
				}
			}
		})
	}
}

// TestGAP2_MultiParamLambda tests that \x y. body creates a multi-param lambda (NOT curried).
// According to the teaching prompt:
// - `\x y. body` should be multi-param (call as f(x, y))
// - `\x. \y. body` should be curried (call as f(x)(y))
func TestGAP2_MultiParamLambda(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		numParams int
		isCurried bool
		desc      string
	}{
		{
			name:      "single_param",
			code:      `\x. x + 1`,
			numParams: 1,
			isCurried: false,
			desc:      "single param lambda",
		},
		{
			name:      "multi_param_space",
			code:      `\x y. x + y`,
			numParams: 2,
			isCurried: false,
			desc:      "multi-param lambda with space (should NOT be curried)",
		},
		{
			name:      "multi_param_three",
			code:      `\x y z. x + y + z`,
			numParams: 3,
			isCurried: false,
			desc:      "three-param lambda",
		},
		{
			name:      "explicit_curried",
			code:      `\x. \y. x + y`,
			numParams: 1, // outer lambda has 1 param
			isCurried: true,
			desc:      "explicitly curried lambda should have nested structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := lexer.New(tt.code, "lambda.ail")
			p := New(lex)
			expr := p.parseExpression(LOWEST)

			if len(p.Errors()) > 0 {
				for _, e := range p.Errors() {
					t.Errorf("Parse error: %v", e)
				}
				t.Fatalf("Parser errors: %s", tt.desc)
			}

			lambda, ok := expr.(*ast.Lambda)
			if !ok {
				t.Fatalf("Expected Lambda, got %T", expr)
			}

			if len(lambda.Params) != tt.numParams {
				t.Errorf("Expected %d params, got %d", tt.numParams, len(lambda.Params))
			}

			// Check if body is a nested lambda (curried)
			_, bodyIsLambda := lambda.Body.(*ast.Lambda)
			if tt.isCurried != bodyIsLambda {
				t.Errorf("Expected curried=%v, got body is lambda=%v", tt.isCurried, bodyIsLambda)
			}
		})
	}
}

// TestGAP2_BareVsParenIdentical verifies that bare and parenthesized final expressions
// produce identical AST structures (the original bug caused different ASTs).
func TestGAP2_BareVsParenIdentical(t *testing.T) {
	bareCode := `module test
let x = foldl(\acc n. acc + n, 0, [1, 2])
x`

	parenCode := `module test
let x = foldl(\acc n. acc + n, 0, [1, 2])
(x)`

	bareLex := lexer.New(bareCode, "bare.ail")
	bareParse := New(bareLex)
	bareFile := bareParse.ParseFile()
	if len(bareParse.Errors()) > 0 {
		t.Fatal("Parse errors in bare code")
	}

	parenLex := lexer.New(parenCode, "paren.ail")
	parenParse := New(parenLex)
	parenFile := parenParse.ParseFile()
	if len(parenParse.Errors()) > 0 {
		t.Fatal("Parse errors in paren code")
	}

	// Both should have same number of statements
	if len(bareFile.Statements) != len(parenFile.Statements) {
		t.Errorf("Statement count mismatch: bare=%d, paren=%d",
			len(bareFile.Statements), len(parenFile.Statements))
	}

	// Specifically, both should have 2 statements (Let + final expression)
	if len(bareFile.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(bareFile.Statements))
	}

	// The Let value should have the same structure (FuncCall, not chained)
	if len(bareFile.Statements) >= 1 {
		bareLet, ok := bareFile.Statements[0].(*ast.Let)
		if !ok {
			t.Fatal("Expected Let statement")
		}
		bareFC, ok := bareLet.Value.(*ast.FuncCall)
		if !ok {
			t.Fatal("Expected FuncCall in Let value")
		}
		// The FuncCall.Func should be an Identifier (foldl), not a chained FuncCall
		if _, isChained := bareFC.Func.(*ast.FuncCall); isChained {
			t.Error("Bare code has chained FuncCall (bug not fixed)")
		}
	}

	if len(parenFile.Statements) >= 1 {
		parenLet, ok := parenFile.Statements[0].(*ast.Let)
		if !ok {
			t.Fatal("Expected Let statement")
		}
		parenFC, ok := parenLet.Value.(*ast.FuncCall)
		if !ok {
			t.Fatal("Expected FuncCall in Let value")
		}
		// The FuncCall.Func should be an Identifier (foldl), not a chained FuncCall
		if _, isChained := parenFC.Func.(*ast.FuncCall); isChained {
			t.Error("Paren code has chained FuncCall (bug not fixed)")
		}
	}
}

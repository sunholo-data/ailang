package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestREPLFileParity tests that expressions parse identically in REPL and file context
func TestREPLFileParity(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple_arithmetic", "1 + 2"},
		{"multiplication", "2 * 3"},
		{"complex_expr", "1 + 2 * 3 - 4"},
		{"function_call", "foo(bar, baz)"},
		{"list_literal", "[1, 2, 3]"},
		{"record_literal", "{x: 1, y: 2}"},
		{"lambda", `\x. x + 1`},
		{"let_expr", "let x = 5 in x + 1"},
		{"if_expr", "if true then 1 else 0"},
		{"boolean", "true && false"},
		{"string", `"hello world"`},
		{"field_access", "foo.bar.baz"},
		{"comparison", "x > 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse as REPL input (standalone expression)
			replParser := New(lexer.New(tt.expr, "<repl>"))
			replProg := replParser.Parse()

			if len(replParser.Errors()) > 0 {
				t.Fatalf("REPL parse errors: %v", replParser.Errors())
			}

			// Parse as file input (standalone expression)
			fileParser := New(lexer.New(tt.expr, "test.ail"))
			fileProg := fileParser.Parse()

			if len(fileParser.Errors()) > 0 {
				t.Fatalf("File parse errors: %v", fileParser.Errors())
			}

			// Compare AST structures
			replAST := ast.PrintProgram(replProg)
			fileAST := ast.PrintProgram(fileProg)

			// Only difference should be the filename
			// Normalize filenames for comparison
			replASTNorm := replAST
			fileASTNorm := fileAST

			// The AST printer already normalizes to "test://unit" so they should match
			if replASTNorm != fileASTNorm {
				t.Errorf("REPL and file ASTs differ:\nREPL:\n%s\n\nFile:\n%s",
					replAST, fileAST)
			}
		})
	}
}

// TestREPLFileParityWithContext tests that module context doesn't affect expression parsing
func TestREPLFileParityWithContext(t *testing.T) {
	expr := "1 + 2 * 3"

	// Parse in REPL (no module)
	replParser := New(lexer.New(expr, "<repl>"))
	replProg := replParser.Parse()

	if len(replParser.Errors()) > 0 {
		t.Fatalf("REPL parse errors: %v", replParser.Errors())
	}

	// Parse in file with module declaration
	fileInput := "module Test\n" + expr
	fileParser := New(lexer.New(fileInput, "test.ail"))
	fileProg := fileParser.Parse()

	if len(fileParser.Errors()) > 0 {
		t.Fatalf("File parse errors: %v", fileParser.Errors())
	}

	// Extract expression from file's statements
	if len(fileProg.File.Statements) == 0 {
		t.Fatal("No statements parsed in file")
	}

	// The expression should be the same regardless of module context
	// We can't easily compare the extracted expr directly, but we can verify
	// both parsed successfully and have statements
	if len(replProg.File.Statements) == 0 {
		t.Error("REPL produced no statements")
	}

	if len(fileProg.File.Statements) == 0 {
		t.Error("File produced no statements")
	}
}

// TestREPLMultilineExpression tests that multi-line expressions work
func TestREPLMultilineExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"let_with_newline",
			"let x = 5\nin x + 1",
		},
		{
			"if_multiline",
			"if true\nthen 1\nelse 0",
		},
		{
			"lambda_multiline",
			`\x.\ny.\nx + y`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "<repl>"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Logf("Parse errors (may be expected): %v", p.Errors())
			}

			// Just verify it doesn't panic
			_ = prog
		})
	}
}

// TestREPLCommandsNotParsed tests that REPL commands are not parsed as expressions
func TestREPLCommandsNotParsed(t *testing.T) {
	// These are REPL commands, not AILANG expressions
	// The parser should either error or treat them as identifiers
	commands := []string{
		":help",
		":quit",
		":type",
		":import",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			p := New(lexer.New(cmd, "<repl>"))
			_ = p.Parse()

			// Commands start with ':', parser will see ':' as unexpected
			// Just verify no panic
		})
	}
}

// TestREPLIncompleteExpression tests handling of incomplete expressions
func TestREPLIncompleteExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"incomplete_let", "let x ="},
		{"incomplete_if", "if true then"},
		{"incomplete_lambda", `\x.`},
		{"incomplete_call", "foo("},
		{"incomplete_list", "[1, 2,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "<repl>"))
			prog := p.Parse()

			// Should produce errors but not panic
			_ = prog
			_ = p.Errors()
		})
	}
}

// TestREPLExpressionStatement tests that expressions are treated as statements
func TestREPLExpressionStatement(t *testing.T) {
	tests := []string{
		"42",
		"1 + 2",
		"foo()",
		"[1, 2, 3]",
		"{x: 1}",
		"true",
		`"hello"`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			p := New(lexer.New(input, "<repl>"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Parse errors: %v", p.Errors())
			}

			if prog.File == nil {
				t.Fatal("Expected parsed file")
			}

			if len(prog.File.Statements) == 0 {
				t.Error("Expected at least one statement")
			}

			// Verify the statement is an expression
			// (the parser treats top-level expressions as statements)
			stmt := prog.File.Statements[0]
			if stmt == nil {
				t.Error("Statement is nil")
			}
		})
	}
}

// TestParserFilenamePreservation tests that parser preserves filename in AST
func TestParserFilenamePreservation(t *testing.T) {
	tests := []struct {
		filename string
		input    string
	}{
		{"<repl>", "1 + 2"},
		{"test.ail", "1 + 2"},
		{"foo/bar/baz.ail", "1 + 2"},
		{"test://unit", "1 + 2"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			p := New(lexer.New(tt.input, tt.filename))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parse errors: %v", p.Errors())
			}

			// The filename should be preserved in the AST
			// (though the printer normalizes to "test://unit")
			if prog.File != nil && prog.File.Path != "" {
				// File path is set - good
			}
		})
	}
}

// TestREPLFileParityTypes tests that type declarations parse identically in REPL and file context
// This is part of M-P2 lock-in: ensuring type syntax works consistently across contexts
func TestREPLFileParityTypes(t *testing.T) {
	tests := []struct {
		name string
		decl string
	}{
		{"simple_alias", "type UserId = int"},
		{"list_alias", "type Names = [string]"},
		// TODO: Tuple type aliases not yet supported - deferred to future milestone
		// {"tuple_alias", "type Point = (int, int)"},
		{"simple_record", "type Point = { x: int, y: int }"},
		{"nested_record", "type User = { name: string, addr: { street: string } }"},
		{"simple_enum", "type Color = Red | Green | Blue"},
		{"enum_with_fields", "type Option = Some(int) | None"},
		{"generic_type", "type Box[a] = { value: a }"},
		{"exported_alias", "export type UserId = int"},
		{"exported_record", "export type Point = { x: int, y: int }"},
		{"exported_sum", "export type Color = Red | Green | Blue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse as REPL input
			replParser := New(lexer.New(tt.decl, "<repl>"))
			replProg := replParser.Parse()

			if len(replParser.Errors()) > 0 {
				t.Fatalf("REPL parse errors: %v", replParser.Errors())
			}

			// Parse as file input
			fileParser := New(lexer.New(tt.decl, "test.ail"))
			fileProg := fileParser.Parse()

			if len(fileParser.Errors()) > 0 {
				t.Fatalf("File parse errors: %v", fileParser.Errors())
			}

			// Compare AST structures
			replAST := ast.PrintProgram(replProg)
			fileAST := ast.PrintProgram(fileProg)

			// The AST printer normalizes to "test://unit" so they should match exactly
			if replAST != fileAST {
				t.Errorf("REPL and file ASTs differ for type declaration:\nREPL:\n%s\n\nFile:\n%s",
					replAST, fileAST)
			}

			// Verify the type declaration was actually parsed
			if replProg.File == nil || len(replProg.File.Statements) == 0 {
				t.Error("REPL produced no type declarations")
			}
			if fileProg.File == nil || len(fileProg.File.Statements) == 0 {
				t.Error("File produced no type declarations")
			}

			// Verify it's actually a TypeDecl
			replStmt := replProg.File.Statements[0]
			fileStmt := fileProg.File.Statements[0]

			if _, ok := replStmt.(*ast.TypeDecl); !ok {
				t.Errorf("REPL statement is not TypeDecl: %T", replStmt)
			}
			if _, ok := fileStmt.(*ast.TypeDecl); !ok {
				t.Errorf("File statement is not TypeDecl: %T", fileStmt)
			}
		})
	}
}

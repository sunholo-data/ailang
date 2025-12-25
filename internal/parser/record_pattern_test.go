package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

func TestParseRecordPattern(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFields int
		wantRest   bool
		wantErr    bool
	}{
		{
			name:       "empty record pattern",
			input:      "match r { {} => 1 }",
			wantFields: 0,
			wantRest:   false,
		},
		{
			name:       "single field shorthand",
			input:      "match r { {name} => name }",
			wantFields: 1,
			wantRest:   false,
		},
		{
			name:       "single field with renaming",
			input:      "match r { {name: n} => n }",
			wantFields: 1,
			wantRest:   false,
		},
		{
			name:       "multiple fields shorthand",
			input:      "match r { {name, age} => name }",
			wantFields: 2,
			wantRest:   false,
		},
		{
			name:       "multiple fields with renaming",
			input:      "match r { {name: n, age: a} => n }",
			wantFields: 2,
			wantRest:   false,
		},
		{
			name:       "mixed shorthand and renaming",
			input:      "match r { {name, age: a} => name }",
			wantFields: 2,
			wantRest:   false,
		},
		{
			name:       "nested record pattern",
			input:      "match r { {user: {name}} => name }",
			wantFields: 1,
			wantRest:   false,
		},
		{
			name:       "rest pattern",
			input:      "match r { {name, ...} => name }",
			wantFields: 1,
			wantRest:   true,
		},
		{
			name:       "trailing comma",
			input:      "match r { {name, age,} => name }",
			wantFields: 2,
			wantRest:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test://record_pattern")
			p := New(l)
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				if !tt.wantErr {
					t.Errorf("unexpected parse errors: %v", p.Errors())
				}
				return
			}

			if tt.wantErr {
				t.Error("expected parse error but got none")
				return
			}

			// Find the match expression
			if prog.File == nil || len(prog.File.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(prog.File.Statements))
			}

			match, ok := prog.File.Statements[0].(*ast.Match)
			if !ok {
				t.Fatalf("expected Match, got %T", prog.File.Statements[0])
			}

			if len(match.Cases) != 1 {
				t.Fatalf("expected 1 case, got %d", len(match.Cases))
			}

			recPat, ok := match.Cases[0].Pattern.(*ast.RecordPattern)
			if !ok {
				t.Fatalf("expected RecordPattern, got %T", match.Cases[0].Pattern)
			}

			if len(recPat.Fields) != tt.wantFields {
				t.Errorf("expected %d fields, got %d", tt.wantFields, len(recPat.Fields))
			}

			if recPat.Rest != tt.wantRest {
				t.Errorf("expected Rest=%v, got %v", tt.wantRest, recPat.Rest)
			}
		})
	}
}

func TestRecordPatternFieldDetails(t *testing.T) {
	input := "match r { {name: n, age} => n }"
	l := lexer.New(input, "test://record_pattern")
	p := New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	match := prog.File.Statements[0].(*ast.Match)
	recPat := match.Cases[0].Pattern.(*ast.RecordPattern)

	// Check first field (renamed)
	if recPat.Fields[0].Name != "name" {
		t.Errorf("expected first field name 'name', got %q", recPat.Fields[0].Name)
	}
	ident, ok := recPat.Fields[0].Pattern.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier pattern for first field, got %T", recPat.Fields[0].Pattern)
	}
	if ident.Name != "n" {
		t.Errorf("expected binding 'n', got %q", ident.Name)
	}

	// Check second field (shorthand)
	if recPat.Fields[1].Name != "age" {
		t.Errorf("expected second field name 'age', got %q", recPat.Fields[1].Name)
	}
	ident2, ok := recPat.Fields[1].Pattern.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier pattern for second field, got %T", recPat.Fields[1].Pattern)
	}
	if ident2.Name != "age" {
		t.Errorf("expected binding 'age', got %q", ident2.Name)
	}
}

func TestNestedRecordPattern(t *testing.T) {
	input := "match r { {user: {name, email}} => name }"
	l := lexer.New(input, "test://record_pattern")
	p := New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	match := prog.File.Statements[0].(*ast.Match)
	outerPat := match.Cases[0].Pattern.(*ast.RecordPattern)

	if len(outerPat.Fields) != 1 {
		t.Fatalf("expected 1 outer field, got %d", len(outerPat.Fields))
	}

	if outerPat.Fields[0].Name != "user" {
		t.Errorf("expected outer field 'user', got %q", outerPat.Fields[0].Name)
	}

	innerPat, ok := outerPat.Fields[0].Pattern.(*ast.RecordPattern)
	if !ok {
		t.Fatalf("expected nested RecordPattern, got %T", outerPat.Fields[0].Pattern)
	}

	if len(innerPat.Fields) != 2 {
		t.Errorf("expected 2 inner fields, got %d", len(innerPat.Fields))
	}
}

package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// parseFuncFromModule is a helper to parse the first function declaration.
func parseFuncFromModule(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	l := lexer.New(src, "test.ail")
	p := New(l)
	prog := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors for %q: %v", src, p.Errors())
	}
	if prog == nil || len(prog.File.Funcs) == 0 {
		t.Fatalf("no function declarations parsed from %q", src)
	}
	return prog.File.Funcs[0]
}

// getLabelName returns the label name from a LabelledType, or "".
func getLabelName(typ ast.Type) string {
	if lt, ok := typ.(*ast.LabelledType); ok && lt.Label != nil {
		return lt.Label.Name
	}
	return ""
}

// getRefinementLabel returns the {not IDENT} label from a LabelledType, or "".
func getRefinementLabel(typ ast.Type) string {
	if lt, ok := typ.(*ast.LabelledType); ok && lt.Refinement != nil {
		return lt.Refinement.NotLabel
	}
	return ""
}

// getBaseType unwraps LabelledType to get the underlying base type.
func getBaseType(typ ast.Type) ast.Type {
	if lt, ok := typ.(*ast.LabelledType); ok {
		return lt.Base
	}
	return typ
}

// TestLabelSyntaxParamAndReturn: func f(x: string<email>) -> string<user>
func TestLabelSyntaxParamAndReturn(t *testing.T) {
	src := "module test\nexport func f(x: string<email>) -> string<user> { x }\n"
	fn := parseFuncFromModule(t, src)

	if len(fn.Params) == 0 {
		t.Fatal("expected at least one param")
	}
	paramType := fn.Params[0].Type
	if getLabelName(paramType) != "email" {
		t.Errorf("param label: got %q, want \"email\"", getLabelName(paramType))
	}
	base := getBaseType(paramType)
	if st, ok := base.(*ast.SimpleType); !ok || st.Name != "string" {
		t.Errorf("param base type: got %T %v, want SimpleType{string}", base, base)
	}

	retType := fn.ReturnType
	if getLabelName(retType) != "user" {
		t.Errorf("return label: got %q, want \"user\"", getLabelName(retType))
	}
	retBase := getBaseType(retType)
	if st, ok := retBase.(*ast.SimpleType); !ok || st.Name != "string" {
		t.Errorf("return base type: got %T %v, want SimpleType{string}", retBase, retBase)
	}
}

// TestRefinementSyntaxParam: func g(x: string{not email}) -> string
func TestRefinementSyntaxParam(t *testing.T) {
	src := "module test\nexport func g(x: string{not email}) -> string { x }\n"
	fn := parseFuncFromModule(t, src)

	if len(fn.Params) == 0 {
		t.Fatal("expected at least one param")
	}
	paramType := fn.Params[0].Type
	ref := getRefinementLabel(paramType)
	if ref != "email" {
		t.Errorf("refinement label: got %q, want \"email\"", ref)
	}
	base := getBaseType(paramType)
	if st, ok := base.(*ast.SimpleType); !ok || st.Name != "string" {
		t.Errorf("base type: got %T %v, want SimpleType{string}", base, base)
	}
}

// TestLabelSyntaxErrors: unsupported refinement forms produce parse errors with hints
func TestLabelSyntaxErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // substring expected in error message, code, or fix
	}{
		{
			name:    "bang not supported",
			input:   "module test\nexport func h(x: string{!email}) -> string { x }\n",
			wantErr: "not",
		},
		{
			name:    "conjunction not supported",
			input:   "module test\nexport func h(x: string{not email && not user}) -> string { x }\n",
			wantErr: "MVP",
		},
		{
			// {label = email} is indistinguishable from a function body at 2-token lookahead;
			// any parse error suffices — the parser will fail on the malformed body
			name:    "label equals form not supported",
			input:   "module test\nexport func h(x: string{label = email}) -> string { x }\n",
			wantErr: "PAR_", // any parser error code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			p.Parse()
			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("expected parse error for %q but got none", tt.input)
			}
			found := false
			for _, e := range errs {
				full := e.Error()
				if strings.Contains(full, tt.wantErr) {
					found = true
					break
				}
				if perr, ok := e.(*ParserError); ok {
					if strings.Contains(perr.Message, tt.wantErr) ||
						strings.Contains(perr.Fix, tt.wantErr) ||
						strings.Contains(perr.Code, tt.wantErr) {
						found = true
						break
					}
					for _, s := range perr.Suggestions {
						if strings.Contains(s, tt.wantErr) {
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, errs)
			}
		})
	}
}

// TestLabelPrettyPrinterRoundTrip: pretty-printer emits canonical T<label> and T{not LABEL} form
func TestLabelPrettyPrinterRoundTrip(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
	}{
		{input: "string<email>", wantType: "string<email>"},
		{input: "string{not email}", wantType: "string{not email}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			src := "module test\nexport func f(x: " + tt.input + ") -> string { x }\n"
			l := lexer.New(src, "test.ail")
			p := New(l)
			prog := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}
			fn := prog.File.Funcs[0]
			got := fn.Params[0].Type.String()
			if got != tt.wantType {
				t.Errorf("pretty-print: got %q, want %q", got, tt.wantType)
			}
		})
	}
}

// TestExistingTypeSyntaxUnchanged: plain unlabelled types still parse correctly
func TestExistingTypeSyntaxUnchanged(t *testing.T) {
	cases := []string{
		"module test\nexport func f(x: int) -> bool { true }\n",
		"module test\nexport func f(x: string) -> string { x }\n",
		"module test\nexport func f(x: [int]) -> [int] { x }\n",
	}
	for _, src := range cases {
		l := lexer.New(src, "test.ail")
		p := New(l)
		p.Parse()
		if len(p.Errors()) > 0 {
			t.Errorf("regression in unlabelled syntax for %q: %v", src, p.Errors())
		}
	}
}

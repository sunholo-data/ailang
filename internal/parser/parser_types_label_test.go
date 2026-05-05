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
			// `{not email && not user}` doesn't match the strict refinement
			// shape `{not IDENT}` (peek4 is `&&`, not `}`), so the parser
			// declines to enter refinement parsing and the malformed input
			// fails at the surrounding parameter-list parser. Any parse error
			// suffices — the user sees that the form is unsupported.
			// Tightened in M-PARSER-REFINEMENT-LOOKAHEAD (v0.15.2): peek3+peek4
			// disambiguation takes precedence over the dedicated MVP error.
			name:    "conjunction not supported",
			input:   "module test\nexport func h(x: string{not email && not user}) -> string { x }\n",
			wantErr: "PAR_", // any parser error code
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

// TestRefinementVsFunctionBodyDisambiguation guards against the regression
// introduced by M-TAINT-TYPES (v0.14.3) where the refinement-type parser
// path mis-claimed a function body opening `{ not <call>(...) }` as a
// refinement, emitting PAR_REFINE_MVP instead of letting the function body
// parse normally. Caught when migrating motoko_agent off its AILANG fork
// (the fork was based on v0.13.0, predates M-TAINT-TYPES, so the syntax
// was idiomatic in motoko-agent's bool-returning helpers).
//
// Fixed in M-PARSER-REFINEMENT-LOOKAHEAD (v0.15.2) by extending the parser
// to 4-token lookahead and tightening the refinement-entry guard to require
// peek3=IDENT && peek4=RBRACE (the exact `{not LABEL}` shape).
//
// Each case below MUST parse without errors. The function uses `not <call>`
// as a top-level expression in a function body whose return type is `bool`
// — historically the most ambiguous position with the refinement syntax.
func TestRefinementVsFunctionBodyDisambiguation(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "not <call> as function body",
			src: "module test\n" +
				"pure func is_pos(x: int) -> bool { x > 0 }\n" +
				"func is_neg(x: int) -> bool {\n" +
				"  not is_pos(x)\n" +
				"}\n",
		},
		{
			name: "not <call> with field access",
			src: "module test\n" +
				"type T = { v: int }\n" +
				"pure func is_pos(t: T) -> bool { t.v > 0 }\n" +
				"func is_neg(t: T) -> bool {\n" +
				"  not is_pos(t)\n" +
				"}\n",
		},
		{
			name: "not <call> in let body returning bool",
			src: "module test\n" +
				"pure func has(x: int) -> bool { x > 0 }\n" +
				"export func main() -> () ! {IO} {\n" +
				"  let result = not has(5) in\n" +
				"  if result then _io_println(\"no\") else _io_println(\"yes\")\n" +
				"}\n",
		},
		{
			// The motoko_agent rpc.ail pattern verbatim (modulo names).
			name: "motoko_agent is_extension_tool_call shape",
			src: "module test\n" +
				"type Tool = { tool: string }\n" +
				"pure func is_native_tool_name(name: string) -> bool {\n" +
				"  name == \"WriteFile\"\n" +
				"}\n" +
				"func is_extension_tool_call(call_req: Tool) -> bool {\n" +
				"  not is_native_tool_name(call_req.tool)\n" +
				"}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.src, "test.ail")
			p := New(l)
			p.Parse()
			if len(p.Errors()) > 0 {
				t.Errorf("expected clean parse, got errors: %v\nsource:\n%s", p.Errors(), tc.src)
			}
		})
	}
}

// TestRefinementStillParsesAfterDisambiguation: the well-formed
// `T{not LABEL}` refinement syntax must still work after the lookahead
// tightening. Belt-and-suspenders coverage alongside the existing
// TestRefinementSyntaxParam.
func TestRefinementStillParsesAfterDisambiguation(t *testing.T) {
	src := "module test\nexport func h(x: string{not email}) -> string { x }\n"
	l := lexer.New(src, "test.ail")
	p := New(l)
	p.Parse()
	if len(p.Errors()) > 0 {
		t.Errorf("refinement parsing regression — `string{not email}` should parse: %v", p.Errors())
	}
}

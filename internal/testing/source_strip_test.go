package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func parseStripFixture(t *testing.T, name string) (string, string, *ast.File) {
	t.Helper()
	path := filepath.Join("testdata", "strip", name)
	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(sourceBytes)
	p := parser.New(lexer.New(source, path))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse %s: %v", path, errs)
	}
	return path, source, file
}

func TestStripNonPureFunctions_DeclarationRanges(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		absent  []string
		present []string
	}{
		{
			name:    "multiline body",
			fixture: "named_test_multiline.ail",
			absent:  []string{"export func big", "x > 10\n}"},
			present: []string{"module named_test_multiline"},
		},
		{
			name:    "contracts",
			fixture: "named_test_contract.ail",
			absent:  []string{"export func guarded", "requires {", "ensures {"},
			present: []string{"module named_test_contract"},
		},
		{
			name:    "leading annotation",
			fixture: "named_test_annotated.ail",
			absent:  []string{"@verify", "export func shout", "println(s)"},
			present: []string{"export pure func big"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, source, file := parseStripFixture(t, tt.fixture)
			// Exercise declaration-range deletion independently of the M2 policy:
			// force the target declaration to be classified as effectful.
			if tt.fixture != "named_test_annotated.ail" {
				file.Funcs[0].Effects = []ast.EffectAnnotation{{Name: "IO"}}
			}
			got := new(Executor).stripNonPureFunctions(source, file)
			for _, needle := range tt.absent {
				if strings.Contains(got, needle) {
					t.Errorf("stripped source unexpectedly contains %q:\n%s", needle, got)
				}
			}
			for _, needle := range tt.present {
				if !strings.Contains(got, needle) {
					t.Errorf("stripped source does not contain control %q:\n%s", needle, got)
				}
			}
		})
	}
}

// Both disjuncts of the invalid-span guard (`endLine == 0 || endLine < startLine`)
// must be exercised. Covering only the zero-value End lets the `< startLine`
// disjunct be deleted with every test still green — a mutation that survived
// the iteration-127 review and is the reason this is a table.
func TestStripNonPureFunctions_InvalidSpanFallsBackToPosition(t *testing.T) {
	source := "module fallback\nfunc remove() -> int { 1 }\nlet control = 2\n"

	tests := []struct {
		name string
		span ast.Span
	}{
		{
			// End is the zero value: hand-built ASTs, extern decls.
			name: "unset end",
			span: ast.Span{Start: ast.Pos{Line: 2}},
		},
		{
			// End precedes Start. Without the `< startLine` disjunct this
			// produces the inverted range {start:2,end:1}, which matches no
			// line at all, so the declaration would survive uncut.
			name: "end before start",
			span: ast.Span{Start: ast.Pos{Line: 2}, End: ast.Pos{Line: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &ast.File{Funcs: []*ast.FuncDecl{{
				Name: "remove",
				Pos:  ast.Pos{Line: 2},
				Span: tt.span,
			}}}
			got := new(Executor).stripNonPureFunctions(source, file)
			if strings.Contains(got, "func remove") {
				t.Fatalf("fallback did not remove declaration line:\n%s", got)
			}
			if !strings.Contains(got, "let control = 2") {
				t.Fatalf("fallback removed known-positive control:\n%s", got)
			}
		})
	}
}

func TestStripNonPureFunctions_EffectivelyPureAndKeepSet(t *testing.T) {
	source := strings.Join([]string{
		"module policy",
		"func implicit() -> int { 1 }",
		"func explicit() -> int ! {} { 2 }",
		"func effectful() -> int ! {IO} { 3 }",
		"pure func keyword() -> int { 4 }",
	}, "\n")
	p := parser.New(lexer.New(source, "policy.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse policy source: %v", errs)
	}
	got := new(Executor).stripNonPureFunctions(source, file, "effectful")
	for _, kept := range []string{"func explicit", "func effectful", "pure func keyword"} {
		if !strings.Contains(got, kept) {
			t.Errorf("expected %q to survive:\n%s", kept, got)
		}
	}
	if strings.Contains(got, "func implicit") {
		t.Errorf("implicit-effects function unexpectedly survived:\n%s", got)
	}
}

func TestStripNonPureFunctions_ExtractFunctionBindingModuleless(t *testing.T) {
	path, _, file := parseStripFixture(t, "moduleless_contract.ail")
	executor := NewExecutor(path)
	binding, err := executor.ExtractFunctionBinding("big", file)
	if err != nil {
		t.Fatalf("ExtractFunctionBinding(big): %v", err)
	}
	if binding == nil || binding.Name != "big" {
		t.Fatalf("binding = %#v, want non-nil binding named big", binding)
	}
	if _, err := executor.ExtractFunctionBinding("nonexistent", file); err == nil || !strings.Contains(err.Error(), "function 'nonexistent' not found") {
		t.Fatalf("negative control error = %v, want function-not-found", err)
	}
}

func TestStripNonPureFunctions_ParseErrorDetectorHasPositiveControl(t *testing.T) {
	tests := []struct {
		fixture       string
		strip         bool
		wantParseCode bool
	}{
		{fixture: "named_test_multiline.ail", strip: true},
		{fixture: "named_test_effectful.ail", strip: true},
		{fixture: "named_test_annotated.ail", strip: true},
		{fixture: "named_test_contract.ail", strip: true},
		// Known-positive control: the detector must observe a genuine parse error.
		{fixture: "malformed_control.ail", wantParseCode: true},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			path := filepath.Join("testdata", "strip", tt.fixture)
			sourceBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			source := string(sourceBytes)
			p := parser.New(lexer.New(source, path))
			file := p.ParseFile()
			if tt.strip {
				if errs := p.Errors(); len(errs) > 0 {
					t.Fatalf("parse fixture before strip: %v", errs)
				}
				source = new(Executor).stripNonPureFunctions(source, file)
				p = parser.New(lexer.New(source, path))
				p.ParseFile()
			}
			hasParseCode := strings.Contains(fmt.Sprint(p.Errors()), "PAR_NO_PREFIX_PARSE")
			if hasParseCode != tt.wantParseCode {
				t.Fatalf("PAR_NO_PREFIX_PARSE present = %v, want %v; errors: %v", hasParseCode, tt.wantParseCode, p.Errors())
			}
		})
	}
}

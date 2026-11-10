package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TestIFCSummaryErrorNamesPrivateFunctionsAsFunctions (#2156): every function
// gets an IFC summary, exported or not. A summary error on a private function
// used to read "export f: missing IFC parameter name", which sent the reporter
// looking for an export that did not exist. The parser no longer produces a
// nameless parameter (PAR_PARAM_NAME), so this AST is built by hand; the
// Validate check stays as the fail-closed invariant.
// Red mutation: always format "export %s" — the private case fails.
func TestIFCSummaryErrorNamesPrivateFunctionsAsFunctions(t *testing.T) {
	nameless := func(name string, export bool) *ast.File {
		return &ast.File{Funcs: []*ast.FuncDecl{{
			Name:       name,
			IsExport:   export,
			Params:     []*ast.Param{{Name: "", Type: &ast.SimpleType{Name: "int"}}},
			ReturnType: &ast.SimpleType{Name: "int"},
			Body:       &ast.Literal{Kind: ast.IntLit, Value: int64(1)},
		}}}
	}
	for _, tc := range []struct {
		export bool
		want   string
	}{
		{false, "function f: missing IFC parameter name"},
		{true, "export f: missing IFC parameter name"},
	} {
		_, err := AnalyzeModuleIFC(nameless("f", tc.export), nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("export=%v: err = %v, want it to contain %q", tc.export, err, tc.want)
		}
	}
}

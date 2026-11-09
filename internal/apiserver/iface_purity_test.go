package apiserver

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Metadata extraction must not replace an interface decision with AST spelling.
func TestMCPMeta_PreservesIfacePurity(t *testing.T) {
	for _, pure := range []bool{false, true} {
		mod := &ModuleInfo{Exports: []ExportInfo{{Name: "probe", Pure: pure}}}
		file := &ast.File{Funcs: []*ast.FuncDecl{{Name: "probe", IsPure: !pure}}}
		extractMCPToolMetaAnnotations(mod, file)
		if mod.Exports[0].Pure != pure {
			t.Fatalf("iface purity %v overwritten with %v", pure, mod.Exports[0].Pure)
		}
	}
}

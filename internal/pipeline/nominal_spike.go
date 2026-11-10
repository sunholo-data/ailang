package pipeline

import "github.com/sunholo-data/ailang/internal/ast"

// spikeTypeHomes builds the M0 spike's type-name -> defining-module table:
// explicit selective type imports, then this module's own declarations (local
// wins, matching shadowLocalTypeNames). Auto-import channels, the prelude and
// builtins are deliberately NOT resolved here: M0 measures how much of the
// program still reaches the checker unqualified without them.
func spikeTypeHomes(imports *moduleImports, file *ast.File, modID string) map[string]string {
	homes := make(map[string]string, len(imports.TypeHomes))
	for name, mod := range imports.TypeHomes {
		homes[name] = mod
	}
	if file != nil {
		for _, name := range localTypeNames(file) {
			homes[name] = modID
		}
	}
	return homes
}

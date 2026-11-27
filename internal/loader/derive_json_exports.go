package loader

import (
	"slices"

	"github.com/sunholo-data/ailang/internal/ast"
)

// registerDerivedJsonExports adds the codecs `deriving (Json)` synthesizes —
// encode<T>(v: T) -> Json and decode<T>(j: Json) -> Result[T, string] — to the
// export table of a module whose exported type T derives Json (DERIVE-JSON).
//
// The elaborator synthesizes their bodies (elaborate.synthesizeDerivedJsonDecls)
// and replaces these entries with the real declarations when it elaborates the
// defining module. Until then an importer's `import ./types (Item, encodeItem,
// decodeItem)` is resolved by GetImportExport against this table, which
// buildExports fills from the parsed file.Funcs alone — so without these
// signature-only entries the names were not exported (IMP010) whenever the
// importer was resolved first.
//
// A function the module writes itself under the same name wins, exactly as
// the elaborator lets it.
func registerDerivedJsonExports(file *ast.File, loaded *LoadedModule) {
	if file == nil || loaded == nil {
		return
	}
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, node := range list {
			td, ok := node.(*ast.TypeDecl)
			if !ok || !td.Exported || len(td.TypeParams) > 0 || !slices.Contains(td.Deriving, ast.DeriveJson) {
				continue
			}
			if loaded.Exports == nil {
				loaded.Exports = make(map[string]*ast.FuncDecl)
			}
			for _, fn := range derivedJsonStubs(td) {
				if _, written := loaded.Exports[fn.Name]; !written {
					loaded.Exports[fn.Name] = fn
				}
			}
		}
	}
}

// derivedJsonStubs returns the signature-only declarations of td's codecs.
func derivedJsonStubs(td *ast.TypeDecl) []*ast.FuncDecl {
	self := func() ast.Type { return &ast.SimpleType{Name: td.Name, Pos: td.Pos} }
	jsonType := func() ast.Type { return &ast.SimpleType{Name: "Json", Pos: td.Pos} }
	return []*ast.FuncDecl{
		{
			Name:       "encode" + td.Name,
			Params:     []*ast.Param{{Name: "v", Type: self(), Pos: td.Pos}},
			ReturnType: jsonType(),
			IsPure:     true,
			IsExport:   true,
			Pos:        td.Pos,
			Origin:     "derive_json",
		},
		{
			Name:   "decode" + td.Name,
			Params: []*ast.Param{{Name: "j", Type: jsonType(), Pos: td.Pos}},
			ReturnType: &ast.TypeApp{
				Constructor: "Result",
				Args:        []ast.Type{self(), &ast.SimpleType{Name: "string", Pos: td.Pos}},
				Pos:         td.Pos,
			},
			IsPure:   true,
			IsExport: true,
			Pos:      td.Pos,
			Origin:   "derive_json",
		},
	}
}

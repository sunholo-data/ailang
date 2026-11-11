package elaborate

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// replModuleID is the module identity of types declared without a file: REPL
// inputs and hand-built elaborations. All of a session's inputs share it.
const replModuleID = "<repl>"

// SetTypeScope installs the type-name resolver of the module being elaborated
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3). The scope's Module is this module's
// canonical ID; its names already include the module's own declarations.
func (e *Elaborator) SetTypeScope(s *types.TypeScope) { e.typeScope = s }

// TypeScope returns the installed (or file-derived) type scope.
func (e *Elaborator) TypeScope() *types.TypeScope { return e.typeScope }

// ModuleID is the identity stamped on the types and constructors this module
// declares: the type scope's module when one is installed, else the path the
// elaborator was created with, else the REPL identity.
func (e *Elaborator) ModuleID() string {
	if e.typeScope != nil && e.typeScope.Module != "" {
		return e.typeScope.Module
	}
	if e.filePath != "" {
		return e.filePath
	}
	return replModuleID
}

// ensureTypeScope gives an elaborator that was not handed a scope one built
// from the file's own type declarations, so every path (single-file, REPL,
// runtime fallback) stamps the types it declares. Names it does not declare
// resolve through the builtin home table, else to this module.
func (e *Elaborator) ensureTypeScope(file *ast.File) {
	if e.typeScope != nil {
		return
	}
	s := types.NewTypeScope(e.ModuleID())
	if file != nil {
		for _, name := range LocalTypeNames(file) {
			s.Bind(name, types.TypeRef{Module: s.Module, Name: name})
		}
	}
	e.typeScope = s
}

// LocalTypeNames returns the names of the type declarations of a file, in
// source order (Decls, then Statements).
func LocalTypeNames(file *ast.File) []string {
	var out []string
	if file == nil {
		return out
	}
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, d := range list {
			if td, ok := d.(*ast.TypeDecl); ok {
				out = append(out, td.Name)
			}
		}
	}
	return out
}

// typeCon mints the TCon a bare type name denotes in this module.
func (e *Elaborator) typeCon(name string) *types.TCon {
	if e.typeScope == nil {
		e.ensureTypeScope(nil)
	}
	return e.typeScope.Con(name)
}

// GetNewtypeADTs returns the local single-constructor ADTs wrapping a record
// that GetTypeAliases lists for field access only.
func (e *Elaborator) GetNewtypeADTs() map[string]bool { return e.newtypeADTs }

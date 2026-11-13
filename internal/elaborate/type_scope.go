package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// replModuleID is the module identity of types declared without a file: REPL
// inputs and hand-built elaborations. All of a session's inputs share it.
const replModuleID = "<repl>"

// SetTypeScope installs the type-name resolver of the module being elaborated
// (M-V1-NOMINAL-TYPE-IDENTITY §4.3). The scope's Module is this module's
// canonical ID; its names already include the module's own declarations.
func (e *Elaborator) SetTypeScope(s *types.TypeScope) { e.typeScope, e.typeScopeImports = s, s != nil }

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
			s.BindLocal(name)
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

// typeCon mints the TCon a bare type name denotes in this module. An
// ambiguous name (TC_AMBIGUOUS_TYPE_001) is latched as the elaboration error,
// located at the annotation that wrote it.
//
// A lowercase name that denotes no type is TC_UNBOUND_TYPE_001 (#2211). Only
// an applied name (`m[int]`, no such type declared, imported or exported by
// an import) reaches here undeclared: a bare one is a type variable
// (simpleType), and an applied one has no type-variable reading, since type
// variables cannot be applied. An undeclared uppercase name is still an
// opaque type of this module (#1485).
func (e *Elaborator) typeCon(name string, pos ast.Pos) *types.TCon {
	if e.typeScope == nil {
		e.ensureTypeScope(nil)
	}
	r, err := e.typeScope.Lookup(name)
	// Only an installed scope knows the imports; a file-derived one cannot
	// tell an imported type from a value, so it reports nothing.
	if err == nil && e.typeScopeImports && isLowerName(name) && !e.typeScope.Declares(name) {
		err = types.UnboundLowerTypeError(name)
	}
	e.latchTypeErr(pos, err)
	return &types.TCon{Name: r.Name, Module: r.Module}
}

// simpleType mints what a bare SimpleType denotes. The parser makes a bare
// lowercase name a SimpleType when the file declares it or imports it by name
// (#2211). One imported by name that is a value, not a type, keeps the
// meaning it had before #2211: an implicitly quantified type variable
// (pipeline/warn_type_names.go warns TC_UNBOUND_TYPE_002). So does any
// lowercase name a file-derived scope cannot resolve. The decision is
// types.TypeScope.ReadsAsTypeVar, shared with the interface builder; its
// comment states the contract for passes that collect type variables.
func (e *Elaborator) simpleType(name string, pos ast.Pos) types.Type {
	if e.typeScope == nil {
		e.ensureTypeScope(nil)
	}
	if e.typeScope.ReadsAsTypeVar(name) {
		return &types.TVar2{Name: name, Kind: types.Star}
	}
	return e.typeCon(name, pos)
}

// latchTypeErr records the first annotation error, located at pos.
func (e *Elaborator) latchTypeErr(pos ast.Pos, err error) {
	if err != nil && e.typeAnnotationErr == nil {
		e.typeAnnotationErr = fmt.Errorf("at %s: %w", e.sourcePos(pos), err)
	}
}

func isLowerName(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z'
}

// GetNewtypeADTs returns the local single-constructor ADTs wrapping a record
// that GetTypeAliases lists for field access only.
func (e *Elaborator) GetNewtypeADTs() map[string]bool { return e.newtypeADTs }

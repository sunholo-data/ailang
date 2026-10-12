package smt

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// smtAdmissible is deliberately distinct from interface/runtime purity:
// Declassify changes IFC labels but does not prevent encoding a contract.
// Callers without surface provenance retain keyword-only admission.
func smtAdmissible(meta *core.DeclMeta, decl ...*ast.FuncDecl) bool {
	if meta == nil {
		return false
	}
	if len(decl) > 0 && decl[0] != nil {
		fd := decl[0]
		return fd.IsPure || rowAdmissible(fd.Effects)
	}
	return meta.IsPure
}

// rowAdmissible requires an explicit closed row. Matching the AST atom name
// admits both bare and scoped Declassify; row variables and other effects fail.
func rowAdmissible(effects []ast.EffectAnnotation) bool {
	if effects == nil {
		return false
	}
	for _, effect := range effects {
		if effect.Name != "Declassify" {
			return false
		}
	}
	return true
}

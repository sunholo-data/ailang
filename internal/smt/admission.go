package smt

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// inadmissibleCalleeError: a callee reached from a verified function is outside
// the effect rule below (no `pure`, no admissible declared row).
type inadmissibleCalleeError struct{ callee string }

func (e inadmissibleCalleeError) Error() string {
	return fmt.Sprintf("%v: callee %q has an inadmissible effect row", ErrUnsupportedConstruct, e.callee)
}

func (e inadmissibleCalleeError) Unwrap() error { return ErrUnsupportedConstruct }

// unsupportedConstructHint is the hint for an encoding the fragment refuses.
func unsupportedConstructHint(err error) string {
	var callee inadmissibleCalleeError
	if errors.As(err, &callee) {
		return fmt.Sprintf("Mark %q `pure` or declare `! {}` (or only `! {Declassify}`): a callee is admitted only by its declaration, so an unannotated func is refused even when it performs no effects.", callee.callee)
	}
	return "Match on list patterns (x :: rest), string patterns, and similar unsupported constructs are outside the decodable fragment. Rewrite using length/element access on lists, or verify a version without the unsupported construct."
}

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

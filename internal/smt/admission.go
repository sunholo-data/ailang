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
		return fmt.Sprintf("%q declares effects, or has no `!` clause and takes a function-typed parameter (its effects could come from the callback). Give it `! {}` (or only `! {Declassify}`) if it is pure, or inline its logic into the caller.", callee.callee)
	}
	return "The construct named in the reason is outside the decodable fragment. Rewrite it with constructs the encoder supports (e.g. length/element access instead of a list pattern nested in a record pattern), or verify a version without it."
}

// smtAdmissible is deliberately distinct from interface/runtime purity:
// Declassify changes IFC labels but does not prevent encoding a contract.
// Callers without surface provenance retain keyword-only admission.
//
// A function with no `!` clause and no keyword is admitted when no parameter
// type mentions a function type. Effect checking rejects every undeclared
// effect in such a body (IO, Clock, ...; Debug is ambient, as it is for
// `pure` and `! {}`), so it has exactly the guarantees of `! {}`. A function
// parameter could carry a callback's row into the function's own (a latent
// row), so it keeps the rowless function out.
func smtAdmissible(meta *core.DeclMeta, decl ...*ast.FuncDecl) bool {
	if meta == nil {
		return false
	}
	if len(decl) > 0 && decl[0] != nil {
		fd := decl[0]
		return fd.IsPure || rowAdmissible(fd.Effects) || (fd.Effects == nil && !hasFunctionParam(fd))
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

// hasFunctionParam reports whether any parameter type of fd mentions a
// function type, at any depth of the written type.
func hasFunctionParam(fd *ast.FuncDecl) bool {
	for _, p := range fd.Params {
		if p != nil && mentionsFunctionType(p.Type) {
			return true
		}
	}
	return false
}

// mentionsFunctionType walks a written (AST) type. Cycle-safe: AST types are
// finite trees; named types are not expanded.
func mentionsFunctionType(t ast.Type) bool {
	switch ty := t.(type) {
	case nil:
		return false
	case *ast.FuncType:
		return true
	case *ast.SimpleType, *ast.TypeVar:
		return false
	case *ast.ListType:
		return mentionsFunctionType(ty.Element)
	case *ast.ArrayType:
		return mentionsFunctionType(ty.Element)
	case *ast.TupleType:
		return anyMentionsFunctionType(ty.Elements)
	case *ast.TypeApp:
		return anyMentionsFunctionType(ty.Args)
	case *ast.RecordType:
		for _, f := range ty.Fields {
			if f != nil && mentionsFunctionType(f.Type) {
				return true
			}
		}
		return false
	case *ast.LabelledType:
		return mentionsFunctionType(ty.Base)
	default:
		panic(fmt.Sprintf("mentionsFunctionType: unhandled ast.Type %T", t))
	}
}

func anyMentionsFunctionType(ts []ast.Type) bool {
	for _, t := range ts {
		if mentionsFunctionType(t) {
			return true
		}
	}
	return false
}

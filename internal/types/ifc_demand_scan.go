package types

import (
	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_demand_scan.go — the breakage bound of #1719, made into a fast path.
//
// Every callable diagnostic needs a demand, and every demand starts at a
// written `{not ℓ}` or positive `<ℓ>` on a parameter: of a module function, of
// a lambda or function literal, or inside a written function type. A module
// with none of these cannot get a new diagnostic from the callable domain, so
// it keeps the plain label walk and pays nothing for the summaries. (Imported
// sinks are not visible to this single-module pass at all until #1134.)

func (c *ifcChecker) moduleHasDemands(file *ast.File) bool {
	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		sig := c.sigs[fn.Name]
		for _, p := range sig.params {
			if p.hasNot || !LabelEqual(p.label, LabelBottom()) || c.typeHasDemands(p.typ) {
				return true
			}
		}
		if c.typeHasDemands(fn.ReturnType) || c.exprHasDemands(fn.Body) {
			return true
		}
	}
	for _, td := range c.types.decls {
		for _, t := range declBodyTypes(td) {
			if c.typeHasDemands(t) {
				return true
			}
		}
	}
	return false
}

// typeHasDemands reports a function type with a labelled or refined
// parameter anywhere inside t.
func (c *ifcChecker) typeHasDemands(t ast.Type) bool {
	switch tt := t.(type) {
	case nil, *ast.SimpleType, *ast.TypeVar:
		return false
	case *ast.FuncType:
		for _, p := range tt.Params {
			if len(demandsOfType("", "", p)) > 0 || c.typeHasDemands(p) {
				return true
			}
		}
		return c.typeHasDemands(tt.Return)
	case *ast.LabelledType:
		return c.typeHasDemands(tt.Base)
	case *ast.RecordType:
		for _, f := range tt.Fields {
			if f != nil && c.typeHasDemands(f.Type) {
				return true
			}
		}
		return false
	case *ast.ListType:
		return c.typeHasDemands(tt.Element)
	case *ast.ArrayType:
		return c.typeHasDemands(tt.Element)
	case *ast.TupleType:
		for _, el := range tt.Elements {
			if c.typeHasDemands(el) {
				return true
			}
		}
		return false
	case *ast.TypeApp:
		for _, a := range tt.Args {
			if c.typeHasDemands(a) {
				return true
			}
		}
		return false
	default:
		return true // unknown type form: do not take the fast path
	}
}

func (c *ifcChecker) paramsHaveDemands(params []*ast.Param) bool {
	for _, p := range params {
		if p != nil && p.Type != nil && (len(demandsOfType("", "", p.Type)) > 0 || c.typeHasDemands(p.Type)) {
			return true
		}
	}
	return false
}

// exprHasDemands reports a demand written anywhere inside e (a lambda or
// function-literal parameter, or a let annotation).
func (c *ifcChecker) exprHasDemands(e ast.Expr) bool {
	any := func(es ...ast.Expr) bool {
		for _, x := range es {
			if c.exprHasDemands(x) {
				return true
			}
		}
		return false
	}
	switch x := e.(type) {
	case nil, *ast.Identifier, *ast.Literal:
		return false
	case *ast.BinaryOp:
		return any(x.Left, x.Right)
	case *ast.UnaryOp:
		return any(x.Expr)
	case *ast.FuncCall:
		return any(x.Func) || any(x.Args...)
	case *ast.Let:
		return c.typeHasDemands(x.Type) || any(x.Value, x.Body)
	case *ast.LetRec:
		return c.typeHasDemands(x.Type) || any(x.Value, x.Body)
	case *ast.Block:
		return any(x.Exprs...)
	case *ast.If:
		return any(x.Condition, x.Then, x.Else)
	case *ast.Match:
		if any(x.Expr) {
			return true
		}
		for _, cs := range x.Cases {
			if any(cs.Guard, cs.Body) {
				return true
			}
		}
		return false
	case *ast.Record:
		for _, f := range x.Fields {
			if any(f.Value) {
				return true
			}
		}
		return false
	case *ast.RecordUpdate:
		if any(x.Base) {
			return true
		}
		for _, f := range x.Fields {
			if any(f.Value) {
				return true
			}
		}
		return false
	case *ast.RecordAccess:
		return any(x.Record)
	case *ast.List:
		return any(x.Elements...)
	case *ast.Array:
		return any(x.Elements...)
	case *ast.Tuple:
		return any(x.Elements...)
	case *ast.Lambda:
		return c.paramsHaveDemands(x.Params) || any(x.Body)
	case *ast.FuncLit:
		return c.paramsHaveDemands(x.Params) || c.typeHasDemands(x.ReturnType) || any(x.Body)
	default:
		return true // a form this scan does not model: keep the full analysis
	}
}

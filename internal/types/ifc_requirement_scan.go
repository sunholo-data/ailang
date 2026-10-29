package types

import "github.com/sunholo-data/ailang/internal/ast"

// Callable summaries can add errors only when the module contains a written
// label/refinement. Keep the original concrete label/Check B walk in all cases,
// but do not build symbolic higher-order graphs for ordinary unannotated code.
// Unknown AST forms conservatively enable analysis; they never hide a demand.
func moduleHasIFCRequirements(file *ast.File) bool {
	var exprs []ast.Expr
	var types []ast.Type
	params := func(ps []*ast.Param) {
		for _, p := range ps {
			if p != nil {
				types = append(types, p.Type)
			}
		}
	}
	for _, fn := range file.Funcs {
		if fn != nil {
			params(fn.Params)
			types = append(types, fn.ReturnType)
			exprs = append(exprs, fn.Body)
		}
	}
	for _, decl := range file.Decls {
		if td, ok := decl.(*ast.TypeDecl); ok {
			types = append(types, declBodyTypes(td)...)
		}
	}
	for len(exprs) > 0 {
		e := exprs[len(exprs)-1]
		exprs = exprs[:len(exprs)-1]
		switch x := e.(type) {
		case nil, *ast.Identifier, *ast.Literal, *ast.Error:
		case *ast.BinaryOp:
			exprs = append(exprs, x.Left, x.Right)
		case *ast.UnaryOp:
			exprs = append(exprs, x.Expr)
		case *ast.FuncCall:
			exprs = append(exprs, x.Func)
			exprs = append(exprs, x.Args...)
		case *ast.Let:
			types = append(types, x.Type)
			exprs = append(exprs, x.Value, x.Body)
		case *ast.LetRec:
			types = append(types, x.Type)
			exprs = append(exprs, x.Value, x.Body)
		case *ast.Block:
			exprs = append(exprs, x.Exprs...)
		case *ast.If:
			exprs = append(exprs, x.Condition, x.Then, x.Else)
		case *ast.Match:
			exprs = append(exprs, x.Expr)
			for _, arm := range x.Cases {
				exprs = append(exprs, arm.Guard, arm.Body)
			}
		case *ast.Record:
			for _, f := range x.Fields {
				exprs = append(exprs, f.Value)
			}
		case *ast.RecordUpdate:
			exprs = append(exprs, x.Base)
			for _, f := range x.Fields {
				exprs = append(exprs, f.Value)
			}
		case *ast.RecordAccess:
			exprs = append(exprs, x.Record)
		case *ast.List:
			exprs = append(exprs, x.Elements...)
		case *ast.Array:
			exprs = append(exprs, x.Elements...)
		case *ast.Tuple:
			exprs = append(exprs, x.Elements...)
		case *ast.Lambda:
			params(x.Params)
			exprs = append(exprs, x.Body)
		case *ast.FuncLit:
			params(x.Params)
			types = append(types, x.ReturnType)
			exprs = append(exprs, x.Body)
		default:
			return true
		}
	}
	seen := map[ast.Type]bool{}
	for len(types) > 0 {
		typ := types[len(types)-1]
		types = types[:len(types)-1]
		if typ == nil || seen[typ] {
			continue
		}
		seen[typ] = true
		switch x := typ.(type) {
		case *ast.LabelledType:
			return true
		case *ast.SimpleType, *ast.TypeVar:
		case *ast.FuncType:
			types = append(types, x.Return)
			types = append(types, x.Params...)
		case *ast.RecordType:
			for _, f := range x.Fields {
				if f != nil {
					types = append(types, f.Type)
				}
			}
		case *ast.ListType:
			types = append(types, x.Element)
		case *ast.ArrayType:
			types = append(types, x.Element)
		case *ast.TupleType:
			types = append(types, x.Elements...)
		case *ast.TypeApp:
			types = append(types, x.Args...)
		default:
			return true
		}
	}
	return false
}

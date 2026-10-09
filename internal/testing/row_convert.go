package testing

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// astExprToCore is the lightweight harness converter. Contracts can use binary
// operators here; inline rows use the stricter ast.RowExprSupported gate.
func astExprToCore(expr ast.Expr) (core.CoreExpr, error) {
	node := core.CoreNode{NodeID: nextNodeID()}
	children := func(exprs []ast.Expr) ([]core.CoreExpr, error) {
		out := make([]core.CoreExpr, len(exprs))
		for i, child := range exprs {
			c, err := astExprToCore(child)
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	}
	switch e := expr.(type) {
	case *ast.Literal:
		kind, err := astLitKindToCore(e.Kind)
		if err != nil {
			return nil, err
		}
		return &core.Lit{CoreNode: node, Kind: kind, Value: e.Value}, nil
	case *ast.Identifier:
		return &core.Var{CoreNode: node, Name: e.Name}, nil
	case *ast.Tuple:
		elems, err := children(e.Elements)
		if err != nil {
			return nil, err
		}
		return &core.Tuple{CoreNode: node, Elements: elems}, nil
	case *ast.List:
		elems, err := children(e.Elements)
		if err != nil {
			return nil, err
		}
		return &core.List{CoreNode: node, Elements: elems}, nil
	case *ast.Record:
		fields := make(map[string]core.CoreExpr, len(e.Fields))
		for _, field := range e.Fields {
			val, err := astExprToCore(field.Value)
			if err != nil {
				return nil, err
			}
			fields[field.Name] = val
		}
		return &core.Record{CoreNode: node, Fields: fields}, nil
	case *ast.FuncCall:
		fn, err := astExprToCore(e.Func)
		if err != nil {
			return nil, err
		}
		args, err := children(e.Args)
		if err != nil {
			return nil, err
		}
		return &core.App{CoreNode: node, Func: fn, Args: args}, nil
	case *ast.UnaryOp:
		operand, err := astExprToCore(e.Expr)
		if err != nil {
			return nil, err
		}
		return &core.UnOp{CoreNode: node, Op: e.Op, Operand: operand}, nil
	case *ast.BinaryOp:
		left, err := astExprToCore(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := astExprToCore(e.Right)
		if err != nil {
			return nil, err
		}
		return &core.BinOp{CoreNode: node, Op: e.Op, Left: left, Right: right}, nil
	default:
		return nil, fmt.Errorf("unsupported AST expression type in test harness: %T", expr)
	}
}

func astLitKindToCore(kind ast.LiteralKind) (core.LitKind, error) {
	switch kind {
	case ast.IntLit:
		return core.IntLit, nil
	case ast.FloatLit:
		return core.FloatLit, nil
	case ast.BoolLit:
		return core.BoolLit, nil
	case ast.StringLit:
		return core.StringLit, nil
	case ast.UnitLit:
		return core.UnitLit, nil
	default:
		return 0, fmt.Errorf("unsupported literal kind: %v", kind)
	}
}

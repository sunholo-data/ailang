package lower

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// FlattenBlock converts a Core expression (typically a let-chain) into a
// sequence of statements plus a final return expression.
//
// Core IR uses nested Let/LetRec for sequential computation:
//
//	Let("a", e1,
//	  Let("b", e2,
//	    body))
//
// This flattens to:
//
//	var a = e1
//	var b = e2
//	return body
//
// This is the heart of the Statement IR lowering — it bridges the gap
// between functional let-chains and imperative statement sequences.
func FlattenBlock(e core.CoreExpr, cti types.CoreTypeInfo) ([]stmt.Stmt, stmt.Expr) {
	if e == nil {
		return nil, stmt.LitUnit{}
	}

	var stmts []stmt.Stmt
	cur := e

	for {
		switch c := cur.(type) {
		case *core.Let:
			// The value may itself be a let-chain (e.g., Let _tmp = ... in expr).
			// We need to flatten those inner lets as statements first, then assign
			// the final expression as the value of the outer VarDecl.
			innerStmts, innerExpr := flattenValue(c.Value, cti)
			stmts = append(stmts, innerStmts...)

			varType := resolveVarType(c, cti)
			_, line := spanOf(c)
			stmts = append(stmts, stmt.VarDecl{
				Name:  c.Name,
				Type:  varType,
				Value: innerExpr,
				Line:  line,
			})
			cur = c.Body
			continue

		case *core.LetRec:
			// Flatten recursive bindings.
			_, line := spanOf(c)
			for _, b := range c.Bindings {
				varType := resolveBindingType(b, cti)
				value := lowerExpr(b.Value, cti)
				bLine := line
				if _, l := spanOf(b.Value); l > 0 {
					bLine = l
				}
				stmts = append(stmts, stmt.VarDecl{
					Name:  b.Name,
					Type:  varType,
					Value: value,
					Line:  bLine,
				})
			}
			cur = c.Body
			continue

		case *core.If:
			// If the If is the tail expression AND both branches are simple,
			// lower as IfExpr. Otherwise, lower as IfStmt with returns.
			if isSimpleExpr(c.Then) && isSimpleExpr(c.Else) {
				retExpr := stmt.IfExpr{
					Cond: lowerExpr(c.Cond, cti),
					Then: lowerExpr(c.Then, cti),
					Else: lowerExpr(c.Else, cti),
				}
				return stmts, retExpr
			}

			// Complex if — lower branches as sub-blocks.
			thenStmts, thenRet := FlattenBlock(c.Then, cti)
			elseStmts, elseRet := FlattenBlock(c.Else, cti)

			_, line := spanOf(c)
			// Append return statements to each branch.
			if thenRet != nil {
				thenStmts = append(thenStmts, stmt.ReturnStmt{Value: thenRet, Line: line})
			}
			if elseRet != nil {
				elseStmts = append(elseStmts, stmt.ReturnStmt{Value: elseRet, Line: line})
			}

			stmts = append(stmts, stmt.IfStmt{
				Cond: lowerExpr(c.Cond, cti),
				Then: thenStmts,
				Else: elseStmts,
				Line: line,
			})
			// After an if-with-returns, the return expression is nil
			// (both branches return).
			return stmts, nil

		case *core.Match:
			// Lower match as a switch statement.
			switchStmt := LowerMatchStmt(c, cti)
			stmts = append(stmts, switchStmt)
			return stmts, nil

		default:
			// Terminal expression — this is the return value.
			return stmts, lowerExpr(cur, cti)
		}
	}
}

// resolveVarType gets the type of a Let binding from CoreTypeInfo.
func resolveVarType(let *core.Let, cti types.CoreTypeInfo) stmt.ResolvedType {
	// The value's type is the variable's type.
	if let.Value != nil {
		if t, ok := cti[let.Value.ID()]; ok {
			return ProjectType(t)
		}
	}
	return nil // let type inference handle it
}

// resolveBindingType gets the type of a recursive binding.
func resolveBindingType(b core.RecBinding, cti types.CoreTypeInfo) stmt.ResolvedType {
	if b.Value != nil {
		if t, ok := cti[b.Value.ID()]; ok {
			return ProjectType(t)
		}
	}
	return nil
}

// flattenValue flattens a Core expression that appears as the RHS of a Let binding.
// If the value is itself a let-chain, we extract its inner statements and return
// only the final expression as the value. This ensures variables are declared
// in the correct order.
func flattenValue(e core.CoreExpr, cti types.CoreTypeInfo) ([]stmt.Stmt, stmt.Expr) {
	if e == nil {
		return nil, stmt.LitUnit{}
	}
	switch c := e.(type) {
	case *core.Let:
		var stmts []stmt.Stmt
		cur := e
		for {
			let, ok := cur.(*core.Let)
			if !ok {
				break
			}
			// Recursively flatten the inner value too.
			innerStmts, innerExpr := flattenValue(let.Value, cti)
			stmts = append(stmts, innerStmts...)
			varType := resolveVarType(let, cti)
			stmts = append(stmts, stmt.VarDecl{
				Name:  let.Name,
				Type:  varType,
				Value: innerExpr,
			})
			cur = let.Body
		}
		// The final expression is the body of the innermost Let.
		return stmts, lowerExpr(cur, cti)
	case *core.LetRec:
		var stmts []stmt.Stmt
		for _, b := range c.Bindings {
			varType := resolveBindingType(b, cti)
			value := lowerExpr(b.Value, cti)
			stmts = append(stmts, stmt.VarDecl{
				Name:  b.Name,
				Type:  varType,
				Value: value,
			})
		}
		bodyExpr := lowerExpr(c.Body, cti)
		return stmts, bodyExpr
	default:
		return nil, lowerExpr(e, cti)
	}
}

// isSimpleExpr checks if a Core expression can be lowered to a single
// Statement IR expression (no statements needed).
func isSimpleExpr(e core.CoreExpr) bool {
	switch e.(type) {
	case *core.Var, *core.VarGlobal, *core.Lit:
		return true
	case *core.BinOp, *core.UnOp, *core.Intrinsic:
		return true
	case *core.App:
		return true
	case *core.Record, *core.RecordAccess:
		return true
	case *core.List, *core.Array, *core.Tuple:
		return true
	case *core.DictApp, *core.DictRef:
		return true
	case *core.Lambda:
		return true
	default:
		return false
	}
}

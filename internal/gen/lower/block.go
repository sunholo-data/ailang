package lower

import (
	"fmt"

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
// only the final expression as the value. If the value is an If or Match whose
// branches contain Let chains, we hoist those branches into IfStmt/SwitchStmt
// that assign a fresh temporary; the returned expression is a reference to the
// temporary. This ensures inner bindings are declared in the correct order and
// not silently dropped by lowerLetExpr — see M4_LOWER_TMP_SCOPE regression.
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
		// The final expression is the body of the innermost Let. Recurse
		// through flattenValue so If/Match tails are handled too.
		tailStmts, tailExpr := flattenValue(cur, cti)
		stmts = append(stmts, tailStmts...)
		return stmts, tailExpr

	case *core.LetRec:
		var stmts []stmt.Stmt
		for _, b := range c.Bindings {
			varType := resolveBindingType(b, cti)
			// LetRec binding values are typically lambdas, which lowerExpr
			// handles fine; if they contain Let-chains, lowerLambda runs
			// FlattenBlock internally so bindings are preserved.
			value := lowerExpr(b.Value, cti)
			stmts = append(stmts, stmt.VarDecl{
				Name:  b.Name,
				Type:  varType,
				Value: value,
			})
		}
		tailStmts, tailExpr := flattenValue(c.Body, cti)
		stmts = append(stmts, tailStmts...)
		return stmts, tailExpr

	case *core.If:
		// Simple branches can be lowered as a pure IfExpr — no temps needed.
		if isSimpleExpr(c.Then) && isSimpleExpr(c.Else) {
			return nil, stmt.IfExpr{
				Cond: lowerExpr(c.Cond, cti),
				Then: lowerExpr(c.Then, cti),
				Else: lowerExpr(c.Else, cti),
			}
		}
		// Complex branches may contain their own Let chains. Hoist the
		// entire if into a temp-assignment IfStmt so inner bindings become
		// real VarDecls in the surrounding block.
		tempName := fmt.Sprintf("$valTmp%d", c.ID())
		var resolved stmt.ResolvedType
		if t, ok := cti[c.ID()]; ok {
			resolved = ProjectType(t)
		}
		_, line := spanOf(c)

		thenStmts, thenExpr := flattenValue(c.Then, cti)
		if thenExpr != nil {
			thenStmts = append(thenStmts, stmt.AssignStmt{
				Name: tempName, Value: thenExpr, Line: line,
			})
		}
		elseStmts, elseExpr := flattenValue(c.Else, cti)
		if elseExpr != nil {
			elseStmts = append(elseStmts, stmt.AssignStmt{
				Name: tempName, Value: elseExpr, Line: line,
			})
		}

		stmts := []stmt.Stmt{
			stmt.VarDecl{
				Name:  tempName,
				Type:  resolved,
				Value: stmt.LitUnit{},
				Line:  line,
			},
			stmt.IfStmt{
				Cond: lowerExpr(c.Cond, cti),
				Then: thenStmts,
				Else: elseStmts,
				Line: line,
			},
		}
		return stmts, stmt.VarRef{Name: tempName}

	case *core.Match:
		// Matches as let-values always need to be hoisted: LowerMatchExpr
		// only accepts a trivial 2-arm lit-pattern shape and panics on
		// anything else. We reuse LowerMatchStmt to produce the switch or
		// if-chain, then rewrite its trailing ReturnStmts into AssignStmts
		// so the value flows into a fresh temp instead of returning from
		// the enclosing function.
		tempName := fmt.Sprintf("$valTmp%d", c.ID())
		var resolved stmt.ResolvedType
		if t, ok := cti[c.ID()]; ok {
			resolved = ProjectType(t)
		}
		_, line := spanOf(c)

		matchStmt := LowerMatchStmt(c, cti)
		rewritten := rewriteReturnsToAssign(matchStmt, tempName)

		stmts := []stmt.Stmt{
			stmt.VarDecl{
				Name:  tempName,
				Type:  resolved,
				Value: stmt.LitUnit{},
				Line:  line,
			},
			rewritten,
		}
		return stmts, stmt.VarRef{Name: tempName}

	default:
		return nil, lowerExpr(e, cti)
	}
}

// rewriteReturnsToAssign walks a statement tree and replaces every
// trailing ReturnStmt with an AssignStmt to tempName. This is used when
// a Match/If is hoisted into a temp-assignment pattern inside
// flattenValue: LowerMatchStmt emits `return <value>` at the end of each
// arm body, but we want the value to flow into the enclosing temp
// instead of unwinding the stack.
func rewriteReturnsToAssign(s stmt.Stmt, tempName string) stmt.Stmt {
	switch s := s.(type) {
	case stmt.ReturnStmt:
		return stmt.AssignStmt{
			Name:  tempName,
			Value: s.Value,
			Line:  s.Line,
		}
	case stmt.IfStmt:
		return stmt.IfStmt{
			Cond: s.Cond,
			Then: rewriteReturnsToAssignList(s.Then, tempName),
			Else: rewriteReturnsToAssignList(s.Else, tempName),
			Line: s.Line,
		}
	case stmt.SwitchStmt:
		cases := make([]stmt.SwitchCase, len(s.Cases))
		for i, c := range s.Cases {
			cases[i] = stmt.SwitchCase{
				Tag:      c.Tag,
				Bindings: c.Bindings,
				Body:     rewriteReturnsToAssignList(c.Body, tempName),
			}
		}
		return stmt.SwitchStmt{
			Scrutinee: s.Scrutinee,
			ADTName:   s.ADTName,
			Cases:     cases,
			Default:   rewriteReturnsToAssignList(s.Default, tempName),
			Line:      s.Line,
		}
	default:
		return s
	}
}

func rewriteReturnsToAssignList(ss []stmt.Stmt, tempName string) []stmt.Stmt {
	if len(ss) == 0 {
		return ss
	}
	out := make([]stmt.Stmt, len(ss))
	for i, s := range ss {
		out[i] = rewriteReturnsToAssign(s, tempName)
	}
	return out
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

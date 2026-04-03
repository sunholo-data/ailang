package lower

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// LowerMatchStmt converts a Core Match into a Statement IR SwitchStmt.
// This handles all 7 pattern types by dispatching to the appropriate
// Statement IR construct:
//
//   - ConstructorPattern → SwitchStmt with tag-based cases
//   - LitPattern → SwitchStmt or if-chain
//   - VarPattern/WildcardPattern → default case
//   - ListPattern, TuplePattern, RecordPattern → if-chain with destructuring
func LowerMatchStmt(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	if len(m.Arms) == 0 {
		return stmt.ExprStmt{Value: stmt.LitUnit{}}
	}

	// Classify the match: are all arms constructor patterns?
	if allConstructorPatterns(m.Arms) {
		return lowerConstructorMatch(m, cti)
	}

	// Mixed or non-constructor patterns — lower as if-chain.
	return lowerIfChainMatch(m, cti)
}

// LowerMatchExpr converts a Core Match into a Statement IR expression.
// For simple cases, returns an IfExpr. For complex cases, this is
// harder — the caller should use FlattenBlock which handles Match
// as a statement with returns in each branch.
func LowerMatchExpr(m *core.Match, cti types.CoreTypeInfo) stmt.Expr {
	// For 2-arm matches with simple patterns, try to produce IfExpr.
	if len(m.Arms) == 2 {
		first := m.Arms[0]
		if _, ok := first.Pattern.(*core.LitPattern); ok {
			if isSimpleExpr(first.Body) && isSimpleExpr(m.Arms[1].Body) {
				cond := lowerLitPatternCond(lowerExpr(m.Scrutinee, cti), first.Pattern.(*core.LitPattern))
				return stmt.IfExpr{
					Cond: cond,
					Then: lowerExpr(first.Body, cti),
					Else: lowerExpr(m.Arms[1].Body, cti),
				}
			}
		}
	}

	// Complex match as expression — lower the first arm's body as default.
	// This is a lossy approximation; the block lowerer handles it properly.
	if len(m.Arms) > 0 {
		return lowerExpr(m.Arms[0].Body, cti)
	}
	return stmt.LitUnit{}
}

func allConstructorPatterns(arms []core.MatchArm) bool {
	for _, arm := range arms {
		switch arm.Pattern.(type) {
		case *core.ConstructorPattern:
			continue
		case *core.WildcardPattern, *core.VarPattern:
			continue // default/variable arms are fine in constructor matches
		default:
			return false
		}
	}
	return true
}

// lowerConstructorMatch produces a SwitchStmt for ADT pattern matching.
func lowerConstructorMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	scrutinee := lowerExpr(m.Scrutinee, cti)

	// Try to determine the ADT type name from the scrutinee's type.
	adtName := ""
	if m.Scrutinee != nil {
		if t, ok := cti[m.Scrutinee.ID()]; ok {
			adtName = extractADTName(t)
		}
	}

	var cases []stmt.SwitchCase
	var defaultBody []stmt.Stmt

	for _, arm := range m.Arms {
		switch pat := arm.Pattern.(type) {
		case *core.ConstructorPattern:
			bindings := extractBindings(pat, cti)
			bodyStmts, retExpr := FlattenBlock(arm.Body, cti)
			if retExpr != nil {
				bodyStmts = append(bodyStmts, stmt.ReturnStmt{Value: retExpr})
			}

			// If there's a guard, wrap the body in an if.
			if arm.Guard != nil {
				guard := lowerExpr(arm.Guard, cti)
				bodyStmts = []stmt.Stmt{
					stmt.IfStmt{
						Cond: guard,
						Then: bodyStmts,
					},
				}
			}

			cases = append(cases, stmt.SwitchCase{
				Tag:      pat.Name,
				Bindings: bindings,
				Body:     bodyStmts,
			})

		case *core.WildcardPattern, *core.VarPattern:
			// Default case.
			bodyStmts, retExpr := FlattenBlock(arm.Body, cti)
			if retExpr != nil {
				bodyStmts = append(bodyStmts, stmt.ReturnStmt{Value: retExpr})
			}
			defaultBody = bodyStmts
		}
	}

	return stmt.SwitchStmt{
		Scrutinee: scrutinee,
		ADTName:   adtName,
		Cases:     cases,
		Default:   defaultBody,
	}
}

// extractADTName gets the ADT type name from a types.Type.
func extractADTName(t types.Type) string {
	switch t := t.(type) {
	case *types.TCon:
		// Skip primitives.
		switch t.Name {
		case "int", "float", "bool", "string", "()", "unit", "bytes":
			return ""
		}
		return t.Name
	case *types.TApp:
		if con, ok := t.Constructor.(*types.TCon); ok {
			return con.Name
		}
	}
	return ""
}

// extractBindings creates Binding entries for constructor pattern fields.
func extractBindings(pat *core.ConstructorPattern, cti types.CoreTypeInfo) []stmt.Binding {
	var bindings []stmt.Binding
	for i, arg := range pat.Args {
		switch a := arg.(type) {
		case *core.VarPattern:
			if a.Name == "_" {
				continue
			}
			bindings = append(bindings, stmt.Binding{
				Name:       a.Name,
				FieldIndex: i,
				Type:       stmt.InterfaceType{}, // resolved later if needed
			})
		case *core.WildcardPattern:
			continue
		default:
			// Nested patterns (e.g., Some(Some(x))) — bind to a temp.
			tmpName := fmt.Sprintf("_pat_%d", i)
			bindings = append(bindings, stmt.Binding{
				Name:       tmpName,
				FieldIndex: i,
				Type:       stmt.InterfaceType{},
			})
		}
	}
	return bindings
}

// lowerIfChainMatch handles non-constructor patterns as an if-else chain.
func lowerIfChainMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	scrutinee := lowerExpr(m.Scrutinee, cti)

	// Build if-else chain from bottom up.
	// Last arm is the else branch (or default).
	if len(m.Arms) == 0 {
		return stmt.ExprStmt{Value: stmt.LitUnit{}}
	}

	// Process arms in reverse to build nested if-else.
	var result stmt.Stmt

	// Last arm is the default/else.
	lastArm := m.Arms[len(m.Arms)-1]
	lastStmts, lastRet := FlattenBlock(lastArm.Body, cti)
	if lastRet != nil {
		lastStmts = append(lastStmts, stmt.ReturnStmt{Value: lastRet})
	}

	// If only one arm, it's just the body.
	if len(m.Arms) == 1 {
		if len(lastStmts) == 1 {
			return lastStmts[0]
		}
		// Wrap in if-true for consistency.
		return stmt.IfStmt{
			Cond: stmt.LitBool{Value: true},
			Then: lastStmts,
		}
	}

	// Build from the bottom.
	result = stmt.IfStmt{
		Cond: stmt.LitBool{Value: true}, // default
		Then: lastStmts,
	}

	// Process remaining arms in reverse order.
	for i := len(m.Arms) - 2; i >= 0; i-- {
		arm := m.Arms[i]
		cond := lowerPatternCond(scrutinee, arm.Pattern)

		armStmts, armRet := FlattenBlock(arm.Body, cti)
		if armRet != nil {
			armStmts = append(armStmts, stmt.ReturnStmt{Value: armRet})
		}

		// Add bindings for variable patterns.
		bindStmts := lowerPatternBindings(scrutinee, arm.Pattern, cti)
		armStmts = append(bindStmts, armStmts...)

		// Add guard if present.
		if arm.Guard != nil {
			guard := lowerExpr(arm.Guard, cti)
			cond = stmt.BinOp{Op: stmt.OpAnd, Left: cond, Right: guard}
		}

		result = stmt.IfStmt{
			Cond: cond,
			Then: armStmts,
			Else: []stmt.Stmt{result},
		}
	}

	return result
}

// lowerPatternCond produces a boolean condition for matching a pattern.
func lowerPatternCond(scrutinee stmt.Expr, pat core.CorePattern) stmt.Expr {
	switch p := pat.(type) {
	case *core.LitPattern:
		return lowerLitPatternCond(scrutinee, p)

	case *core.ConstructorPattern:
		// Tag check — compare scrutinee's tag.
		return stmt.BinOp{
			Op:    stmt.OpEq,
			Left:  stmt.FieldAccess{Record: scrutinee, Field: "Tag"},
			Right: stmt.LitString{Value: p.Name},
		}

	case *core.VarPattern, *core.WildcardPattern:
		return stmt.LitBool{Value: true} // always matches

	case *core.TuplePattern:
		// Tuple patterns match by structure — always true if types match.
		return stmt.LitBool{Value: true}

	case *core.ListPattern:
		// List pattern — check length.
		if len(p.Elements) == 0 && p.Tail == nil {
			// Match empty list.
			return stmt.BinOp{
				Op:    stmt.OpEq,
				Left:  stmt.BuiltinCall{Name: "_len", Args: []stmt.Expr{scrutinee}},
				Right: stmt.LitInt{Value: 0},
			}
		}
		return stmt.BinOp{
			Op:    stmt.OpGte,
			Left:  stmt.BuiltinCall{Name: "_len", Args: []stmt.Expr{scrutinee}},
			Right: stmt.LitInt{Value: int64(len(p.Elements))},
		}

	case *core.RecordPattern:
		// Record patterns — always match (fields are accessed by name).
		return stmt.LitBool{Value: true}

	default:
		return stmt.LitBool{Value: true}
	}
}

func lowerLitPatternCond(scrutinee stmt.Expr, p *core.LitPattern) stmt.Expr {
	var litExpr stmt.Expr
	switch v := p.Value.(type) {
	case int64:
		litExpr = stmt.LitInt{Value: v}
	case int:
		litExpr = stmt.LitInt{Value: int64(v)}
	case float64:
		litExpr = stmt.LitFloat{Value: v}
	case bool:
		litExpr = stmt.LitBool{Value: v}
	case string:
		litExpr = stmt.LitString{Value: v}
	default:
		litExpr = stmt.LitUnit{}
	}
	return stmt.BinOp{
		Op:    stmt.OpEq,
		Left:  scrutinee,
		Right: litExpr,
	}
}

// lowerPatternBindings produces VarDecl statements for pattern variable bindings.
func lowerPatternBindings(scrutinee stmt.Expr, pat core.CorePattern, cti types.CoreTypeInfo) []stmt.Stmt {
	switch p := pat.(type) {
	case *core.VarPattern:
		if p.Name == "_" {
			return nil
		}
		return []stmt.Stmt{
			stmt.VarDecl{Name: p.Name, Value: scrutinee},
		}

	case *core.TuplePattern:
		var stmts []stmt.Stmt
		for i, elem := range p.Elements {
			if vp, ok := elem.(*core.VarPattern); ok && vp.Name != "_" {
				stmts = append(stmts, stmt.VarDecl{
					Name:  vp.Name,
					Value: stmt.FieldAccess{Record: scrutinee, Field: fmt.Sprintf("_%d", i)},
				})
			}
		}
		return stmts

	case *core.ListPattern:
		var stmts []stmt.Stmt
		for i, elem := range p.Elements {
			if vp, ok := elem.(*core.VarPattern); ok && vp.Name != "_" {
				stmts = append(stmts, stmt.VarDecl{
					Name: vp.Name,
					Value: stmt.BuiltinCall{
						Name: "_list_get",
						Args: []stmt.Expr{scrutinee, stmt.LitInt{Value: int64(i)}},
					},
				})
			}
		}
		if p.Tail != nil {
			if vp, ok := (*p.Tail).(*core.VarPattern); ok && vp.Name != "_" {
				stmts = append(stmts, stmt.VarDecl{
					Name: vp.Name,
					Value: stmt.BuiltinCall{
						Name: "_list_tail",
						Args: []stmt.Expr{scrutinee, stmt.LitInt{Value: int64(len(p.Elements))}},
					},
				})
			}
		}
		return stmts

	case *core.RecordPattern:
		var stmts []stmt.Stmt
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(p.Fields))
		for k := range p.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fieldPat := p.Fields[k]
			if vp, ok := fieldPat.(*core.VarPattern); ok && vp.Name != "_" {
				stmts = append(stmts, stmt.VarDecl{
					Name:  vp.Name,
					Value: stmt.FieldAccess{Record: scrutinee, Field: k},
				})
			}
		}
		return stmts

	default:
		return nil
	}
}

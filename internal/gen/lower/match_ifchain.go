package lower

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// lowerIfChainMatch lowers any match as the evaluator runs it: try each arm
// in source order; an arm fires when its pattern matches AND its guard —
// evaluated with the pattern's bindings in scope — holds; otherwise control
// moves to the next arm.
//
// Each arm becomes
//
//	if <patternCond> { <patternBindings>; <body> } else { <later arms> }
//
// with an unguarded irrefutable arm ending the chain (later arms are
// unreachable). A guard that cannot see any binding is ANDed into the
// condition. A guard on a pattern that binds variables must run AFTER the
// bindings, and on failure fall through to the later arms; duplicating the
// later arms into both else-branches would grow exponentially with the
// number of guarded arms, so a per-match flag carries the fall-through
// instead:
//
//	var $matchedN = false
//	if <patternCond> { <bindings>; if <guard> { $matchedN = true; <body> } }
//	if !$matchedN { <later arms> }
//
// In tail position every body ends in a return, so the flag is only read
// on the fall-through path; in value position (flattenValue rewrites the
// returns into assignments) the flag is what keeps later arms from running.
func lowerIfChainMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	var pre []stmt.Stmt
	scrutinee := lowerExpr(m.Scrutinee, cti)
	if !isAtomicExpr(scrutinee) {
		// Every arm reads the scrutinee; evaluate it once.
		name := fmt.Sprintf("$scrut%d", m.ID())
		pre = append(pre, stmt.VarDecl{Name: name, Value: scrutinee})
		scrutinee = stmt.VarRef{Name: name}
	}
	var scrutType types.Type
	if m.Scrutinee != nil {
		scrutType = cti[m.Scrutinee.ID()]
	}

	flag := fmt.Sprintf("$matched%d", m.ID())
	needFlag := false

	var rest []stmt.Stmt // statements implementing arms[i+1:]
	for i := len(m.Arms) - 1; i >= 0; i-- {
		arm := m.Arms[i]
		cond := patternCond(scrutinee, scrutType, arm.Pattern)
		binds := patternBindings(scrutinee, arm.Pattern)
		body := armBodyStmts(arm, cti)

		var guardStmts []stmt.Stmt
		var guardExpr stmt.Expr
		if arm.Guard != nil {
			guardStmts, guardExpr = flattenValue(arm.Guard, cti)
		}

		if arm.Guard != nil && (len(binds) > 0 || len(guardStmts) > 0) {
			// Bind, then guard, then fall through on failure. A guard
			// that needs statements (ANF lets) also takes this form so
			// they run only once the pattern has matched.
			needFlag = true
			guarded := stmt.IfStmt{
				Cond: guardExpr,
				Then: append([]stmt.Stmt{stmt.AssignStmt{Name: flag, Value: stmt.LitBool{Value: true}}}, body...),
			}
			if cond == nil {
				cond = stmt.LitBool{Value: true}
			}
			armThen := append(append(binds, guardStmts...), guarded)
			armStmt := stmt.IfStmt{Cond: cond, Then: armThen}
			next := []stmt.Stmt{armStmt}
			if len(rest) > 0 {
				next = append(next, stmt.IfStmt{
					Cond: stmt.UnOp{Op: stmt.OpNot, Operand: stmt.VarRef{Name: flag}},
					Then: rest,
				})
			}
			rest = next
			continue
		}

		if guardExpr != nil {
			cond = andCond(cond, guardExpr)
		}
		then := append(binds, body...)
		if cond == nil {
			// Unguarded irrefutable arm: always fires, later arms are dead.
			rest = then
			continue
		}
		rest = []stmt.Stmt{stmt.IfStmt{Cond: cond, Then: then, Else: rest}}
	}

	stmts := pre
	if needFlag {
		stmts = append(stmts, stmt.VarDecl{Name: flag, Type: stmt.PrimitiveType{Kind: stmt.PrimBool}, Value: stmt.LitBool{Value: false}})
	}
	stmts = append(stmts, rest...)
	if len(stmts) == 1 {
		return stmts[0]
	}
	// Wrap in a block-shaped If so the match stays one statement and its
	// bindings stay scoped to it.
	return stmt.IfStmt{Cond: stmt.LitBool{Value: true}, Then: stmts}
}

// isAtomicExpr reports whether re-evaluating e is free and pure.
func isAtomicExpr(e stmt.Expr) bool {
	switch e.(type) {
	case stmt.VarRef, stmt.GlobalRef, stmt.LitInt, stmt.LitFloat, stmt.LitBool, stmt.LitString, stmt.LitUnit:
		return true
	}
	return false
}

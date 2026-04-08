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
//
// In practice this is only reachable for non-tail-position matches
// (e.g. `1 + match x { ... }`). Tail-position matches are intercepted
// by FlattenBlock and routed to LowerMatchStmt, which produces a
// proper SwitchStmt or if-chain. The Phase 2C corpus does not exercise
// non-tail-position matches.
//
// For the one supported shape — a 2-arm match where the first arm is a
// LitPattern and both bodies are simple expressions — we emit an IfExpr.
// Anything else PANICS rather than silently producing a lossy approximation
// (the previous behavior was to return `lowerExpr(arms[0].Body)`, which
// produced a wrong-but-running result for any match it didn't recognize).
//
// If you hit this panic, the right fix is usually to (a) refactor the
// AILANG source so the match is in tail position, or (b) extend
// FlattenBlock to bind the match's result into a temp via a hoisted
// SwitchStmt and then reference that temp.
func LowerMatchExpr(m *core.Match, cti types.CoreTypeInfo) stmt.Expr {
	// Supported shape: 2-arm lit-pattern match → IfExpr.
	if len(m.Arms) == 2 {
		first := m.Arms[0]
		if litPat, ok := first.Pattern.(*core.LitPattern); ok {
			if isSimpleExpr(first.Body) && isSimpleExpr(m.Arms[1].Body) {
				cond := lowerLitPatternCond(lowerExpr(m.Scrutinee, cti), litPat)
				return stmt.IfExpr{
					Cond: cond,
					Then: lowerExpr(first.Body, cti),
					Else: lowerExpr(m.Arms[1].Body, cti),
				}
			}
		}
	}

	panic(fmt.Sprintf(
		"lower: LowerMatchExpr called on a non-tail-position match shape "+
			"that has no IfExpr lowering (arms=%d). The previous lossy "+
			"fallback (returning the first arm's body) was removed by "+
			"M-LOWER-FIX follow-up. Refactor the source to put the match "+
			"in tail position, or extend FlattenBlock to hoist it.",
		len(m.Arms)))
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
//
// Bug A.2 fix (M-LOWER-FIX follow-up): when a constructor arm has literal
// sub-patterns (e.g., `Num(0) => true`), `extractBindings` previously
// silently dropped the literal value, so the switch matched any `Num(_)`.
// We now bind each literal field to a temp and synthesize a guard
// `_lit_<i> == <literal>`. If the guard fails inside the matched case, the
// case body falls through to the *default* body inlined into the else
// branch — the SwitchStmt itself has no per-case fall-through, so we
// duplicate the default body. This is fine for the corpus (default bodies
// are short like `return false`), and exhaustiveness checking guarantees a
// default exists when literal sub-patterns are present.
func lowerConstructorMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	scrutinee := lowerExpr(m.Scrutinee, cti)

	// Try to determine the ADT type name from the scrutinee's type.
	adtName := ""
	if m.Scrutinee != nil {
		if t, ok := cti[m.Scrutinee.ID()]; ok {
			adtName = extractADTName(t)
		}
	}

	// First pass: collect the default body, if any. We need it before
	// processing constructor arms with literal-sub-pattern guards.
	var defaultBody []stmt.Stmt
	for _, arm := range m.Arms {
		switch arm.Pattern.(type) {
		case *core.WildcardPattern, *core.VarPattern:
			bodyStmts, retExpr := FlattenBlock(arm.Body, cti)
			if retExpr != nil {
				bodyStmts = append(bodyStmts, stmt.ReturnStmt{Value: retExpr})
			}
			defaultBody = bodyStmts
		}
	}

	var cases []stmt.SwitchCase

	for _, arm := range m.Arms {
		pat, ok := arm.Pattern.(*core.ConstructorPattern)
		if !ok {
			continue // var/wildcard already captured above
		}
		bindings, litGuards := extractBindingsAndGuards(pat)
		bodyStmts, retExpr := FlattenBlock(arm.Body, cti)
		if retExpr != nil {
			bodyStmts = append(bodyStmts, stmt.ReturnStmt{Value: retExpr})
		}

		// Combine literal-sub-pattern guards with the explicit `if` guard.
		var combinedGuard stmt.Expr
		for _, g := range litGuards {
			if combinedGuard == nil {
				combinedGuard = g
			} else {
				combinedGuard = stmt.BinOp{Op: stmt.OpAnd, Left: combinedGuard, Right: g}
			}
		}
		if arm.Guard != nil {
			armGuard := lowerExpr(arm.Guard, cti)
			if combinedGuard == nil {
				combinedGuard = armGuard
			} else {
				combinedGuard = stmt.BinOp{Op: stmt.OpAnd, Left: combinedGuard, Right: armGuard}
			}
		}

		if combinedGuard != nil {
			// Wrap the body in an `if guard then body else <default>`.
			// The else branch is the duplicated default body so the switch
			// case still produces a value when the literal/explicit guard
			// fails. Without the duplication the case would silently exit
			// the switch and skip the default — this was Bug A.2.
			ifStmt := stmt.IfStmt{
				Cond: combinedGuard,
				Then: bodyStmts,
				Else: defaultBody, // may be nil if no default; see below
			}
			if len(litGuards) > 0 && len(defaultBody) == 0 {
				panic(fmt.Sprintf(
					"lower: constructor pattern %s has literal sub-arguments "+
						"but no wildcard/var default arm — match is non-exhaustive "+
						"and exhaustiveness checking should have rejected it",
					pat.Name))
			}
			bodyStmts = []stmt.Stmt{ifStmt}
		}

		cases = append(cases, stmt.SwitchCase{
			Tag:      pat.Name,
			Bindings: bindings,
			Body:     bodyStmts,
		})
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

// extractBindingsAndGuards creates Binding entries for constructor pattern
// fields and synthesizes guard expressions for any literal sub-patterns
// (Bug A.2 fix). Returns (bindings, extra guards).
//
// For each field of the constructor:
//
//   - Var pattern (`Foo(x)`): bind `x` to the field. No guard.
//   - Wildcard (`Foo(_)`): skip — no binding, no guard.
//   - Literal (`Foo(0)`): bind to a temp `_lit_<i>` AND emit a guard
//     `_lit_<i> == <literal>`. The guard is AND-combined into the case
//     body's outer `if` so the case fires only on a value match. This
//     fixes the silent-success bug where `Num(0) => true` matched
//     `Num(_)`.
//   - Nested pattern (`Foo(Some(x))`): bind to a temp `_pat_<i>` for
//     downstream destructuring (currently unsupported beyond binding).
func extractBindingsAndGuards(pat *core.ConstructorPattern) ([]stmt.Binding, []stmt.Expr) {
	var bindings []stmt.Binding
	var guards []stmt.Expr
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
		case *core.LitPattern:
			tmpName := fmt.Sprintf("_lit_%d", i)
			bindings = append(bindings, stmt.Binding{
				Name:       tmpName,
				FieldIndex: i,
				Type:       stmt.InterfaceType{},
			})
			guards = append(guards, stmt.BinOp{
				Op:    stmt.OpEq,
				Left:  stmt.VarRef{Name: tmpName},
				Right: lowerLitPatternToExpr(a),
			})
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
	return bindings, guards
}

// lowerLitPatternToExpr converts a LitPattern's value to the matching
// stmt.Expr literal node. Used by extractBindingsAndGuards to build
// equality guards for literal sub-patterns.
func lowerLitPatternToExpr(p *core.LitPattern) stmt.Expr {
	switch v := p.Value.(type) {
	case int64:
		return stmt.LitInt{Value: v}
	case int:
		return stmt.LitInt{Value: int64(v)}
	case float64:
		return stmt.LitFloat{Value: v}
	case bool:
		return stmt.LitBool{Value: v}
	case string:
		return stmt.LitString{Value: v}
	default:
		return stmt.LitUnit{}
	}
}

// lowerIfChainMatch handles non-constructor patterns as an if-else chain.
//
// Bug A fix (M-LOWER-FIX): every arm — including the single-arm case and the
// "last arm" of a multi-arm chain — must have its pattern bindings computed
// and prepended. The previous version only ran lowerPatternBindings for
// non-last arms, which caused single-arm tuple matches (`swap`, `fst`) and
// last-arm list-cons matches (`sumList`) to reference unbound variables.
func lowerIfChainMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	scrutinee := lowerExpr(m.Scrutinee, cti)

	if len(m.Arms) == 0 {
		return stmt.ExprStmt{Value: stmt.LitUnit{}}
	}

	// Build the body for one arm: bindings + (optionally) guard wrap + arm body.
	armBody := func(arm core.MatchArm) []stmt.Stmt {
		armStmts, armRet := FlattenBlock(arm.Body, cti)
		if armRet != nil {
			armStmts = append(armStmts, stmt.ReturnStmt{Value: armRet})
		}
		bindStmts := lowerPatternBindings(scrutinee, arm.Pattern, cti)
		return append(bindStmts, armStmts...)
	}

	// Single-arm: emit bindings + body, no surrounding If.
	if len(m.Arms) == 1 {
		body := armBody(m.Arms[0])
		if len(body) == 1 {
			return body[0]
		}
		return stmt.IfStmt{
			Cond: stmt.LitBool{Value: true},
			Then: body,
		}
	}

	// Multi-arm: build the chain from the last arm upward. Even the last arm
	// gets its bindings computed; if its pattern is irrefutable (var/wildcard)
	// it becomes a true default, otherwise it becomes a guarded If with the
	// pattern's condition.
	lastArm := m.Arms[len(m.Arms)-1]
	lastStmts := armBody(lastArm)

	var result stmt.Stmt
	if isIrrefutablePattern(lastArm.Pattern) {
		// True default — wrap in If{true} for shape consistency.
		result = stmt.IfStmt{
			Cond: stmt.LitBool{Value: true},
			Then: lastStmts,
		}
	} else {
		// Refutable last arm — must check the pattern condition. If it
		// fails, fall through to a unit ExprStmt (the match was inexhaustive
		// at the source level; type checker should have flagged it).
		cond := lowerPatternCond(scrutinee, lastArm.Pattern)
		if lastArm.Guard != nil {
			cond = stmt.BinOp{
				Op:    stmt.OpAnd,
				Left:  cond,
				Right: lowerExpr(lastArm.Guard, cti),
			}
		}
		result = stmt.IfStmt{
			Cond: cond,
			Then: lastStmts,
		}
	}

	// Process remaining arms in reverse order.
	for i := len(m.Arms) - 2; i >= 0; i-- {
		arm := m.Arms[i]
		cond := lowerPatternCond(scrutinee, arm.Pattern)

		armStmts := armBody(arm)

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

// isIrrefutablePattern reports whether a pattern always matches and so can
// safely serve as the default/else arm of an if-chain without a condition.
func isIrrefutablePattern(p core.CorePattern) bool {
	switch p.(type) {
	case *core.VarPattern, *core.WildcardPattern:
		return true
	}
	return false
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

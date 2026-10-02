package lower

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// LowerMatchStmt converts a Core Match into Statement IR.
//
// Two shapes are produced (m-vm-match-lowering):
//
//   - SwitchStmt — the fast path, for a match the switch can express
//     exactly: every arm a constructor pattern with distinct tags and only
//     variable/wildcard arguments, optionally followed by ONE final,
//     unguarded catch-all. See switchEligible.
//   - an if-chain — everything else: any pattern kind at any depth, guards
//     that reference the arm's bindings, catch-alls in any position. It
//     mirrors the evaluator's linear arm loop: match → bind → guard → next
//     arm on failure. See lowerIfChainMatch.
func LowerMatchStmt(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	if len(m.Arms) == 0 {
		return stmt.ExprStmt{Value: stmt.LitUnit{}}
	}
	if switchEligible(m.Arms) {
		return lowerConstructorMatch(m, cti)
	}
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

// switchEligible reports whether a SwitchStmt computes exactly what the
// evaluator's first-match-wins arm loop computes. A switch dispatches on
// the tag alone, binds fields positionally, and runs Default only when no
// case tag matched, so it is exact only when:
//
//   - every arm but the last is a constructor pattern, and tags are
//     distinct (a repeated tag would make the second arm unreachable);
//   - constructor arguments are variables or wildcards (a literal or nested
//     argument is a refutable condition the tag test does not check);
//   - a variable/wildcard arm, if any, is the FINAL arm and unguarded
//     (anywhere else it would lose its source position, #1473);
//   - at least one constructor arm exists (an all-catch-all match has no
//     ADT to switch on).
//
// Guards on constructor arms are fine: they run after the case bindings,
// and on failure the only arm that can still match is the catch-all.
func switchEligible(arms []core.MatchArm) bool {
	seen := make(map[string]bool, len(arms))
	for i, arm := range arms {
		switch p := arm.Pattern.(type) {
		case *core.ConstructorPattern:
			if seen[p.Name] {
				return false
			}
			seen[p.Name] = true
			for _, a := range p.Args {
				switch a.(type) {
				case *core.VarPattern, *core.WildcardPattern:
				default:
					return false
				}
			}
		case *core.VarPattern, *core.WildcardPattern:
			if i != len(arms)-1 || arm.Guard != nil {
				return false
			}
		default:
			return false
		}
	}
	return len(seen) > 0
}

// lowerConstructorMatch produces a SwitchStmt for a match switchEligible
// accepted. Constructor arguments become case Bindings (positional
// GET_FIELD); a guard wraps the case body and falls back to the catch-all
// body on failure; a variable catch-all binds the scrutinee in Default.
func lowerConstructorMatch(m *core.Match, cti types.CoreTypeInfo) stmt.Stmt {
	scrutinee := lowerExpr(m.Scrutinee, cti)

	adtName := ""
	if m.Scrutinee != nil {
		if t, ok := cti[m.Scrutinee.ID()]; ok {
			adtName = extractADTName(t)
		}
	}

	// The catch-all, if present, is the final arm (switchEligible).
	var defaultBody []stmt.Stmt
	last := m.Arms[len(m.Arms)-1]
	if _, isCtor := last.Pattern.(*core.ConstructorPattern); !isCtor {
		defaultBody = append(patternBindings(scrutinee, last.Pattern), armBodyStmts(last, cti)...)
	}

	var cases []stmt.SwitchCase
	for _, arm := range m.Arms {
		pat, ok := arm.Pattern.(*core.ConstructorPattern)
		if !ok {
			continue
		}
		var bindings []stmt.Binding
		for i, arg := range pat.Args {
			if vp, ok := arg.(*core.VarPattern); ok && vp.Name != "_" {
				bindings = append(bindings, stmt.Binding{
					Name:       vp.Name,
					FieldIndex: i,
					Type:       stmt.InterfaceType{},
				})
			}
		}
		body := armBodyStmts(arm, cti)
		if arm.Guard != nil {
			// The guard runs after the case bindings; its own ANF lets
			// (flattenValue) run only inside the matched case.
			guardStmts, guardExpr := flattenValue(arm.Guard, cti)
			body = append(guardStmts, stmt.IfStmt{
				Cond: guardExpr,
				Then: body,
				Else: defaultBody,
			})
		}
		cases = append(cases, stmt.SwitchCase{Tag: pat.Name, Bindings: bindings, Body: body})
	}

	return stmt.SwitchStmt{
		Scrutinee: scrutinee,
		ADTName:   adtName,
		Cases:     cases,
		Default:   defaultBody,
	}
}

// armBodyStmts flattens an arm body into statements ending in a return.
func armBodyStmts(arm core.MatchArm, cti types.CoreTypeInfo) []stmt.Stmt {
	body, ret := FlattenBlock(arm.Body, cti)
	if ret != nil {
		body = append(body, stmt.ReturnStmt{Value: ret})
	}
	return body
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

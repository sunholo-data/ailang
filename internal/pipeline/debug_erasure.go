// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// DebugEraser removes the Debug ghost effect from a Core program.
// In release mode, Debug calls become unit literals and Debug is removed
// from all effect rows, making Debug-only functions pure.
type DebugEraser struct{}

// Erase removes all Debug effects and calls from a Core program.
// Returns a new program with Debug erased.
func (e *DebugEraser) Erase(prog *core.Program) *core.Program {
	erased := &core.Program{
		Decls: make([]core.CoreExpr, len(prog.Decls)),
		Meta:  prog.Meta,
	}
	for i, decl := range prog.Decls {
		erased.Decls[i] = e.eraseExpr(decl)
	}
	return erased
}

// isDebugBuiltin returns true if the name refers to a Debug builtin function.
func isDebugBuiltin(name string) bool {
	return name == "_debug_log" || name == "_debug_check"
}

// isDebugCall returns true if an App node calls a Debug builtin.
func isDebugCall(app *core.App) bool {
	switch fn := app.Func.(type) {
	case *core.Var:
		return isDebugBuiltin(fn.Name)
	case *core.VarGlobal:
		// "$builtin" is what DebugLocationInjector rewrites user-level
		// std/debug.log/check calls into; before that pass existed, user
		// calls reached the wrapper by name and were never erased.
		return (fn.Ref.Module == "std/debug" || fn.Ref.Module == "$builtin") && isDebugBuiltin(fn.Ref.Name)
	}
	return false
}

// unitLit returns a unit literal expression.
func unitLit() core.CoreExpr {
	return &core.Lit{Kind: core.UnitLit, Value: nil}
}

// eraseExpr recursively transforms a Core expression, replacing Debug calls
// with unit literals. Traversal is the shared mapCoreChildren walker.
func (e *DebugEraser) eraseExpr(expr core.CoreExpr) core.CoreExpr {
	if app, ok := expr.(*core.App); ok && isDebugCall(app) {
		return unitLit()
	}
	return mapCoreChildren(expr, e.eraseExpr)
}

// mapCoreChildren rebuilds expr with f applied to each direct child. It is
// the one structural walker shared by the Core-to-Core passes in this
// package (DebugEraser, DebugLocationInjector); f decides what to do at a
// node, this function only knows the node shapes. Leaves are returned as-is.
func mapCoreChildren(expr core.CoreExpr, f func(core.CoreExpr) core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch n := expr.(type) {
	case *core.App:
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = f(arg)
		}
		return &core.App{
			CoreNode: n.CoreNode,
			Func:     f(n.Func),
			Args:     newArgs,
		}

	case *core.Let:
		return &core.Let{
			CoreNode: n.CoreNode,
			Name:     n.Name,
			Value:    f(n.Value),
			Body:     f(n.Body),
		}

	case *core.LetRec:
		newBindings := make([]core.RecBinding, len(n.Bindings))
		for i, b := range n.Bindings {
			newBindings[i] = core.RecBinding{
				Name:  b.Name,
				Value: f(b.Value),
			}
		}
		return &core.LetRec{
			CoreNode: n.CoreNode,
			Bindings: newBindings,
			Body:     f(n.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: n.CoreNode,
			Params:   n.Params,
			Body:     f(n.Body),
		}

	case *core.If:
		return &core.If{
			CoreNode: n.CoreNode,
			Cond:     f(n.Cond),
			Then:     f(n.Then),
			Else:     f(n.Else),
		}

	case *core.Match:
		newArms := make([]core.MatchArm, len(n.Arms))
		for i, arm := range n.Arms {
			newArms[i] = core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   f(arm.Guard),
				Body:    f(arm.Body),
			}
		}
		return &core.Match{
			CoreNode:   n.CoreNode,
			Scrutinee:  f(n.Scrutinee),
			Arms:       newArms,
			Exhaustive: n.Exhaustive,
		}

	case *core.Record:
		newFields := make(map[string]core.CoreExpr, len(n.Fields))
		for k, v := range n.Fields {
			newFields[k] = f(v)
		}
		return &core.Record{
			CoreNode: n.CoreNode,
			Fields:   newFields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: n.CoreNode,
			Record:   f(n.Record),
			Field:    n.Field,
		}

	case *core.RecordUpdate:
		newUpdates := make(map[string]core.CoreExpr, len(n.Updates))
		for k, v := range n.Updates {
			newUpdates[k] = f(v)
		}
		return &core.RecordUpdate{
			CoreNode: n.CoreNode,
			Base:     f(n.Base),
			Updates:  newUpdates,
		}

	case *core.List:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = f(elem)
		}
		return &core.List{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.Array:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = f(elem)
		}
		return &core.Array{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.Tuple:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = f(elem)
		}
		return &core.Tuple{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.BinOp:
		return &core.BinOp{
			CoreNode: n.CoreNode,
			Op:       n.Op,
			Left:     f(n.Left),
			Right:    f(n.Right),
		}

	case *core.UnOp:
		return &core.UnOp{
			CoreNode: n.CoreNode,
			Op:       n.Op,
			Operand:  f(n.Operand),
		}

	case *core.Intrinsic:
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = f(arg)
		}
		return &core.Intrinsic{
			CoreNode: n.CoreNode,
			Op:       n.Op,
			Args:     newArgs,
		}

	case *core.Forall:
		return &core.Forall{
			CoreNode: n.CoreNode,
			Var:      n.Var,
			Lo:       f(n.Lo),
			Hi:       f(n.Hi),
			Body:     f(n.Body),
		}

	case *core.DictAbs:
		return &core.DictAbs{
			CoreNode: n.CoreNode,
			Params:   n.Params,
			Body:     f(n.Body),
		}

	case *core.DictApp:
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = f(arg)
		}
		return &core.DictApp{
			CoreNode: n.CoreNode,
			Dict:     f(n.Dict),
			Method:   n.Method,
			Args:     newArgs,
		}

	// Atomic/leaf nodes — no children to recurse into
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef:
		return expr

	default:
		// Unknown node type — return as-is (safe default)
		return expr
	}
}

// EraseDebugFromEffectRow removes the "Debug" label from an effect row.
// If the row becomes empty (no labels, no tail), returns nil (pure).
// If the row was nil (pure), returns nil.
// Delegates to the generic EraseEffectFromRow.
func EraseDebugFromEffectRow(row *types.Row) *types.Row {
	return EraseEffectFromRow(row, "Debug")
}

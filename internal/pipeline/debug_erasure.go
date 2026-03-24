// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
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
		return fn.Ref.Module == "std/debug" && isDebugBuiltin(fn.Ref.Name)
	}
	return false
}

// unitLit returns a unit literal expression.
func unitLit() core.CoreExpr {
	return &core.Lit{Kind: core.UnitLit, Value: nil}
}

// eraseExpr recursively transforms a Core expression, replacing Debug calls
// with unit literals and erasing Debug from effect rows.
func (e *DebugEraser) eraseExpr(expr core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch n := expr.(type) {
	case *core.App:
		if isDebugCall(n) {
			return unitLit()
		}
		// Recurse into function and args
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = e.eraseExpr(arg)
		}
		return &core.App{
			CoreNode: n.CoreNode,
			Func:     e.eraseExpr(n.Func),
			Args:     newArgs,
		}

	case *core.Let:
		return &core.Let{
			CoreNode: n.CoreNode,
			Name:     n.Name,
			Value:    e.eraseExpr(n.Value),
			Body:     e.eraseExpr(n.Body),
		}

	case *core.LetRec:
		newBindings := make([]core.RecBinding, len(n.Bindings))
		for i, b := range n.Bindings {
			newBindings[i] = core.RecBinding{
				Name:  b.Name,
				Value: e.eraseExpr(b.Value),
			}
		}
		return &core.LetRec{
			CoreNode: n.CoreNode,
			Bindings: newBindings,
			Body:     e.eraseExpr(n.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: n.CoreNode,
			Params:   n.Params,
			Body:     e.eraseExpr(n.Body),
		}

	case *core.If:
		return &core.If{
			CoreNode: n.CoreNode,
			Cond:     e.eraseExpr(n.Cond),
			Then:     e.eraseExpr(n.Then),
			Else:     e.eraseExpr(n.Else),
		}

	case *core.Match:
		newArms := make([]core.MatchArm, len(n.Arms))
		for i, arm := range n.Arms {
			newArms[i] = core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   e.eraseExpr(arm.Guard),
				Body:    e.eraseExpr(arm.Body),
			}
		}
		return &core.Match{
			CoreNode:   n.CoreNode,
			Scrutinee:  e.eraseExpr(n.Scrutinee),
			Arms:       newArms,
			Exhaustive: n.Exhaustive,
		}

	case *core.Record:
		newFields := make(map[string]core.CoreExpr, len(n.Fields))
		for k, v := range n.Fields {
			newFields[k] = e.eraseExpr(v)
		}
		return &core.Record{
			CoreNode: n.CoreNode,
			Fields:   newFields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: n.CoreNode,
			Record:   e.eraseExpr(n.Record),
			Field:    n.Field,
		}

	case *core.RecordUpdate:
		newUpdates := make(map[string]core.CoreExpr, len(n.Updates))
		for k, v := range n.Updates {
			newUpdates[k] = e.eraseExpr(v)
		}
		return &core.RecordUpdate{
			CoreNode: n.CoreNode,
			Base:     e.eraseExpr(n.Base),
			Updates:  newUpdates,
		}

	case *core.List:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = e.eraseExpr(elem)
		}
		return &core.List{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.Array:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = e.eraseExpr(elem)
		}
		return &core.Array{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.Tuple:
		newElems := make([]core.CoreExpr, len(n.Elements))
		for i, elem := range n.Elements {
			newElems[i] = e.eraseExpr(elem)
		}
		return &core.Tuple{
			CoreNode: n.CoreNode,
			Elements: newElems,
		}

	case *core.BinOp:
		return &core.BinOp{
			CoreNode: n.CoreNode,
			Op:       n.Op,
			Left:     e.eraseExpr(n.Left),
			Right:    e.eraseExpr(n.Right),
		}

	case *core.UnOp:
		return &core.UnOp{
			CoreNode: n.CoreNode,
			Op:       n.Op,
			Operand:  e.eraseExpr(n.Operand),
		}

	case *core.Intrinsic:
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = e.eraseExpr(arg)
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
			Lo:       e.eraseExpr(n.Lo),
			Hi:       e.eraseExpr(n.Hi),
			Body:     e.eraseExpr(n.Body),
		}

	case *core.DictAbs:
		return &core.DictAbs{
			CoreNode: n.CoreNode,
			Params:   n.Params,
			Body:     e.eraseExpr(n.Body),
		}

	case *core.DictApp:
		newArgs := make([]core.CoreExpr, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = e.eraseExpr(arg)
		}
		return &core.DictApp{
			CoreNode: n.CoreNode,
			Dict:     e.eraseExpr(n.Dict),
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
func EraseDebugFromEffectRow(row *types.Row) *types.Row {
	if row == nil {
		return nil
	}

	// Check if Debug is present
	if _, hasDebug := row.Labels["Debug"]; !hasDebug {
		return row // No Debug to erase
	}

	// Build new labels without Debug
	newLabels := make(map[string]types.Type, len(row.Labels)-1)
	for k, v := range row.Labels {
		if k != "Debug" {
			newLabels[k] = v
		}
	}

	// Build new budgets without Debug
	var newBudgets map[string]*int
	if row.Budgets != nil {
		newBudgets = make(map[string]*int, len(row.Budgets))
		for k, v := range row.Budgets {
			if k != "Debug" {
				newBudgets[k] = v
			}
		}
		if len(newBudgets) == 0 {
			newBudgets = nil
		}
	}

	// Build new min budgets without Debug
	var newMinBudgets map[string]*int
	if row.MinBudgets != nil {
		newMinBudgets = make(map[string]*int, len(row.MinBudgets))
		for k, v := range row.MinBudgets {
			if k != "Debug" {
				newMinBudgets[k] = v
			}
		}
		if len(newMinBudgets) == 0 {
			newMinBudgets = nil
		}
	}

	// If no labels remain and no row variable tail, this is pure
	if len(newLabels) == 0 && row.Tail == nil {
		return nil
	}

	return &types.Row{
		Kind:       row.Kind,
		Labels:     newLabels,
		Tail:       row.Tail,
		Budgets:    newBudgets,
		MinBudgets: newMinBudgets,
	}
}

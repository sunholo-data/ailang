package elaborate

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ElaborateWithDictionaries transforms operators to dictionary calls
// This is the second pass after type checking
func ElaborateWithDictionaries(prog *core.Program, resolved map[uint64]*types.ResolvedConstraint) (*core.Program, error) {
	elaborator := &DictElaborator{
		resolved:    resolved,
		freshVarNum: 0,
	}

	// Transform each declaration
	var newDecls []core.CoreExpr
	for _, decl := range prog.Decls {
		transformed := elaborator.transformExpr(decl)
		newDecls = append(newDecls, transformed)
	}

	// Preserve program metadata and flags (CRITICAL for module exports!)
	return &core.Program{
		Decls: newDecls,
		Meta:  prog.Meta,  // Preserve export metadata
		Flags: prog.Flags, // Preserve compilation state
	}, nil
}

// DictElaborator handles dictionary transformation
type DictElaborator struct {
	resolved    map[uint64]*types.ResolvedConstraint
	freshVarNum int
}

// transformExpr recursively transforms Core expressions
func (de *DictElaborator) transformExpr(expr core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.BinOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.BinOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Left:     de.transformExpr(e.Left),
					Right:    de.transformExpr(e.Right),
				}
			}

			// Transform to dictionary application
			// First transform the operands
			left := de.transformExpr(e.Left)
			right := de.transformExpr(e.Right)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build the ANF structure properly:
			// For now, just use DictApp directly with DictRef as the dictionary
			// This is valid ANF since DictRef is atomic
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{left, right},
			}
		}

		// No dictionary transformation needed, just recurse
		return &core.BinOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Left:     de.transformExpr(e.Left),
			Right:    de.transformExpr(e.Right),
		}

	case *core.UnOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.UnOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Operand:  de.transformExpr(e.Operand),
				}
			}

			// Transform to dictionary application
			operand := de.transformExpr(e.Operand)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build ANF structure properly with DictRef directly in DictApp
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{operand},
			}
		}

		// No transformation needed
		return &core.UnOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Operand:  de.transformExpr(e.Operand),
		}

	case *core.Intrinsic:
		// M-DX19: Check if this intrinsic has a resolved constraint (for == and != on ADT types)
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				args := make([]core.CoreExpr, len(e.Args))
				for i, arg := range e.Args {
					args[i] = de.transformExpr(arg)
				}
				return &core.Intrinsic{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Args:     args,
				}
			}

			// Transform to dictionary application
			// Transform the operands
			args := make([]core.CoreExpr, len(e.Args))
			for i, arg := range e.Args {
				args[i] = de.transformExpr(arg)
			}

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     args,
			}
		}

		// No dictionary transformation needed - pass through for OpLowering
		args := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			args[i] = de.transformExpr(arg)
		}
		return &core.Intrinsic{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Args:     args,
		}

	case *core.Let:
		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    de.transformExpr(e.Value),
			Body:     de.transformExpr(e.Body),
		}

	case *core.LetRec:
		var newBindings []core.RecBinding
		for _, binding := range e.Bindings {
			newBindings = append(newBindings, core.RecBinding{
				Name:  binding.Name,
				Value: de.transformExpr(binding.Value),
			})
		}
		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: newBindings,
			Body:     de.transformExpr(e.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     de.transformExpr(e.Body),
		}

	case *core.App:
		var newArgs []core.CoreExpr
		for _, arg := range e.Args {
			newArgs = append(newArgs, de.transformExpr(arg))
		}
		return &core.App{
			CoreNode: e.CoreNode,
			Func:     de.transformExpr(e.Func),
			Args:     newArgs,
		}

	case *core.If:
		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     de.transformExpr(e.Cond),
			Then:     de.transformExpr(e.Then),
			Else:     de.transformExpr(e.Else),
		}

	case *core.Match:
		var newArms []core.MatchArm
		for _, arm := range e.Arms {
			var guard core.CoreExpr
			if arm.Guard != nil {
				guard = de.transformExpr(arm.Guard)
			}
			newArms = append(newArms, core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   guard,
				Body:    de.transformExpr(arm.Body),
			})
		}
		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  de.transformExpr(e.Scrutinee),
			Arms:       newArms,
			Exhaustive: e.Exhaustive,
		}

	case *core.Record:
		newFields := make(map[string]core.CoreExpr)
		for k, v := range e.Fields {
			newFields[k] = de.transformExpr(v)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   newFields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   de.transformExpr(e.Record),
			Field:    e.Field,
		}

	case *core.List:
		var newElements []core.CoreExpr
		for _, elem := range e.Elements {
			newElements = append(newElements, de.transformExpr(elem))
		}
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: newElements,
		}

	// Atomic expressions - return as is
	case *core.Var, *core.Lit, *core.DictRef:
		return expr

	// Already dictionary nodes - preserve
	case *core.DictAbs, *core.DictApp:
		return expr

	default:
		// Unknown type - return as is
		return expr
	}
}

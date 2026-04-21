package pipeline

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// ResolveVarTypes propagates monomorphic types from bindings to Var occurrences (M-DX4).
//
// Problem: After type inference and ApplySubstitution, Var nodes may still have type
// variables (α4, etc.) in CoreTI because the substitution doesn't always capture bindings.
//
// Solution: For each Var with a TVar in CoreTI, look up its binding (Let value or lambda param)
// and if the binding has a concrete type (non-TVar head), copy it to the Var.
//
// Rules:
//   - Only propagate from monomorphic bindings (concrete heads: Int, Float, String, Bool, List)
//   - Never invent types; only copy existing concrete types
//   - Preserve polymorphism: don't over-constrain lambda params or polymorphic lets
//   - Idempotent: running twice has no effect
//
// This is a WORKAROUND for M-DX4. The principled fix (M-POLY-B) will re-elaborate specialized
// bodies after monomorphization, which will naturally resolve all operator types.
type VarResolver struct {
	coreTI   types.CoreTypeInfo
	bindings map[string]types.Type // Var name → monomorphic type from binding
}

// NewVarResolver creates a new Var type resolver
func NewVarResolver(coreTI types.CoreTypeInfo) *VarResolver {
	return &VarResolver{
		coreTI:   coreTI,
		bindings: make(map[string]types.Type),
	}
}

// Resolve walks the Core AST and resolves Var types from bindings
func (r *VarResolver) Resolve(prog *core.Program) {
	for _, decl := range prog.Decls {
		r.resolveExpr(decl)
	}
}

// resolveExpr recursively resolves Var types
func (r *VarResolver) resolveExpr(expr core.CoreExpr) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *core.Var:
		// Check if this Var has a TVar in CoreTI
		if varType, ok := r.coreTI.Get(e.ID()); ok {
			head := types.Head(varType)
			if head == types.HeadUnknown {
				// TVar! Try to resolve from binding
				if bindingType, found := r.bindings[e.Name]; found {
					// Copy monomorphic type from binding
					r.coreTI.Set(e.ID(), bindingType)
				}
			}
		}

	case *core.Let:
		// First, resolve the value
		r.resolveExpr(e.Value)

		// Check if value has a monomorphic type
		if valueType, ok := r.coreTI.Get(e.Value.ID()); ok {
			head := types.Head(valueType)
			// Only record if monomorphic (concrete head)
			if isMonomorphicHead(head) {
				r.bindings[e.Name] = valueType
			}
		}

		// Then resolve the body (where the Var may be used)
		r.resolveExpr(e.Body)

		// Clean up binding after body
		delete(r.bindings, e.Name)

	case *core.LetRec:
		// LetRec is complex - for now, don't propagate (preserve polymorphism)
		// Each binding might reference others, so we can't safely propagate
		for _, binding := range e.Bindings {
			r.resolveExpr(binding.Value)
		}
		r.resolveExpr(e.Body)

	case *core.Lambda:
		// Lambda params: don't propagate unless call site is specialized (M-POLY-B)
		// Just recurse into body
		r.resolveExpr(e.Body)

	case *core.App:
		r.resolveExpr(e.Func)
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *core.If:
		r.resolveExpr(e.Cond)
		r.resolveExpr(e.Then)
		r.resolveExpr(e.Else)

	case *core.Match:
		r.resolveExpr(e.Scrutinee)
		for _, arm := range e.Arms {
			r.resolveExpr(arm.Body)
		}

	case *core.Intrinsic:
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *core.BinOp:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)

	case *core.UnOp:
		r.resolveExpr(e.Operand)

	case *core.Record:
		for _, fieldVal := range e.Fields {
			r.resolveExpr(fieldVal)
		}

	case *core.RecordAccess:
		r.resolveExpr(e.Record)

	case *core.RecordUpdate:
		r.resolveExpr(e.Base)
		for _, updateVal := range e.Updates {
			r.resolveExpr(updateVal)
		}

	case *core.List:
		for _, elem := range e.Elements {
			r.resolveExpr(elem)
		}

	case *core.Tuple:
		for _, elem := range e.Elements {
			r.resolveExpr(elem)
		}

	case *core.DictAbs:
		r.resolveExpr(e.Body)

	case *core.DictApp:
		r.resolveExpr(e.Dict)
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *core.Lit, *core.VarGlobal:
		// Leaf nodes - no children

	default:
		// Unknown type - skip
	}
}

// isMonomorphicHead checks if a type head is concrete (not a type variable)
func isMonomorphicHead(head types.TypeHead) bool {
	switch head {
	case types.HeadInt, types.HeadFloat, types.HeadString, types.HeadBool, types.HeadList:
		return true
	default:
		return false
	}
}

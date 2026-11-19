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

// shadow hides the entries of names until the returned func restores them.
func (r *VarResolver) shadow(names ...string) func() {
	type saved struct {
		name string
		typ  types.Type
		had  bool
	}
	prev := make([]saved, 0, len(names))
	for _, n := range names {
		t, had := r.bindings[n]
		prev = append(prev, saved{n, t, had})
		delete(r.bindings, n)
	}
	return func() {
		for i := len(prev) - 1; i >= 0; i-- {
			if prev[i].had {
				r.bindings[prev[i].name] = prev[i].typ
			} else {
				delete(r.bindings, prev[i].name)
			}
		}
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

		// The binding shadows any outer entry of its name for the body, and
		// the outer entry comes back afterwards. It used to be deleted, and
		// a lambda parameter or pattern variable did not shadow it at all, so
		// `let x = 1; match o { Some(x) => x == k }` gave the pattern's x the
		// type int and lowered == to eq_Int on strings.
		restore := r.shadow(e.Name)
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
		restore()

	case *core.LetRec:
		// LetRec is complex - for now, don't propagate (preserve polymorphism)
		// Each binding might reference others, so we can't safely propagate.
		// The group's names still shadow outer entries.
		restore := r.shadow(recNames(e.Bindings)...)
		for _, binding := range e.Bindings {
			r.resolveExpr(binding.Value)
		}
		r.resolveExpr(e.Body)
		restore()

	case *core.Lambda:
		// Lambda params: don't propagate unless call site is specialized (M-POLY-B)
		// They shadow outer entries in the body.
		restore := r.shadow(e.Params...)
		r.resolveExpr(e.Body)
		restore()

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
			// Pattern variables shadow outer entries in the guard and body.
			restore := r.shadow(patternBoundVars(arm.Pattern)...)
			r.resolveExpr(arm.Guard)
			r.resolveExpr(arm.Body)
			restore()
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

	case *core.Array:
		for _, elem := range e.Elements {
			r.resolveExpr(elem)
		}

	case *core.Forall:
		r.resolveExpr(e.Lo)
		r.resolveExpr(e.Hi)
		restore := r.shadow(e.Var)
		r.resolveExpr(e.Body)
		restore()

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

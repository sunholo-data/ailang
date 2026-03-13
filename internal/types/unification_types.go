package types

import (
	"fmt"
)

// flattenFunc collects all parameters from a potentially curried function type.
// E.g., TFunc2{[a], Return: TFunc2{[b], Return: c}} → params=[a,b], return=c
// A non-curried TFunc2{[a,b], Return: c} → params=[a,b], return=c (unchanged).
func flattenFunc(f *TFunc2) (params []Type, ret Type) {
	params = append(params, f.Params...)
	ret = f.Return
	for {
		inner, ok := ret.(*TFunc2)
		if !ok || len(inner.Params) == 0 {
			break
		}
		params = append(params, inner.Params...)
		ret = inner.Return
	}
	return
}

// unifyFunctions unifies two function types
func (u *Unifier) unifyFunctions(t1 *TFunc2, t2 Type, sub Substitution) (Substitution, error) {
	if t2Func, ok := t2.(*TFunc2); ok {
		p1, r1 := t1.Params, t1.Return
		p2, r2 := t2Func.Params, t2Func.Return

		// M-DOCPARSE-DX M1: If arity mismatches, try flattening curried forms.
		// This allows (a -> (b -> c)) to unify with ((a, b) -> c).
		if len(p1) != len(p2) {
			fp1, fr1 := flattenFunc(t1)
			fp2, fr2 := flattenFunc(t2Func)
			if len(fp1) == len(fp2) {
				p1, r1 = fp1, fr1
				p2, r2 = fp2, fr2
			} else {
				return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(fp1), len(fp2))
			}
		}

		// Unify parameters
		for i := range p1 {
			var err error
			sub, err = u.Unify(p1[i], p2[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify parameter %d: %w", i, err)
			}
		}

		// Unify effect rows
		if t1.EffectRow != nil || t2Func.EffectRow != nil {
			eff1 := t1.EffectRow
			if eff1 == nil {
				eff1 = EmptyEffectRow()
			}
			eff2 := t2Func.EffectRow
			if eff2 == nil {
				eff2 = EmptyEffectRow()
			}
			var err error
			sub, err = u.Unify(eff1, eff2, sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify effect rows: %w", err)
			}
		}

		// Unify return type
		return u.Unify(r1, r2, sub)
	}
	if t2Old, ok := t2.(*TFunc); ok {
		// Cross-unify TFunc2 ↔ TFunc: unify params and return, elide effect rows
		// (symmetric with unifyTFunc's TFunc2 case)
		fp1, fr1 := flattenFunc(t1)
		p2, r2 := t2Old.Params, t2Old.Return
		if len(fp1) != len(p2) {
			return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(fp1), len(p2))
		}
		for i := range fp1 {
			var err error
			sub, err = u.Unify(fp1[i], p2[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify parameter %d: %w", i, err)
			}
		}
		return u.Unify(fr1, r2, sub)
	}
	if t2Var, ok := t2.(*TVar2); ok {
		// Swap and retry
		return u.Unify(t2Var, t1, sub)
	}
	return nil, fmt.Errorf("cannot unify function type with %T", t2)
}

// unifyTFunc unifies old-style TFunc types
// M-FIX-FLOAT-OP: Added to handle TFunc types that appear after substitution chain resolution
func (u *Unifier) unifyTFunc(t1 *TFunc, t2 Type, sub Substitution) (Substitution, error) {
	switch t2 := t2.(type) {
	case *TFunc:
		// Both are old-style TFunc
		if len(t1.Params) != len(t2.Params) {
			return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(t1.Params), len(t2.Params))
		}

		// Unify parameters
		for i := range t1.Params {
			var err error
			sub, err = u.Unify(t1.Params[i], t2.Params[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify parameter %d: %w", i, err)
			}
		}

		// Unify return types
		return u.Unify(t1.Return, t2.Return, sub)

	case *TFunc2:
		// Convert TFunc to TFunc2 semantics: unify params and return
		// M-DOCPARSE-DX M1: flatten curried TFunc2 before comparing with TFunc
		fp2, fr2 := flattenFunc(t2)
		p1, r1 := t1.Params, t1.Return
		if len(p1) != len(fp2) {
			return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(p1), len(fp2))
		}

		// Unify parameters
		for i := range p1 {
			var err error
			sub, err = u.Unify(p1[i], fp2[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify parameter %d: %w", i, err)
			}
		}

		// Unify return types
		return u.Unify(r1, fr2, sub)

	case *TVar2:
		// Swap and retry
		return u.Unify(t2, t1, sub)

	default:
		return nil, fmt.Errorf("cannot unify function type with %T", t2)
	}
}

// unifyLists unifies two list types
// DX-17: Handles both TList and TApp("list", ...) representations
func (u *Unifier) unifyLists(t1 *TList, t2 Type, sub Substitution) (Substitution, error) {
	// Check if t2 is also a list (TList or TApp("list", ...))
	if elem, ok := AsList(t2); ok {
		return u.Unify(t1.Element, elem, sub)
	}
	if t2Var, ok := t2.(*TVar2); ok {
		// Swap and retry
		return u.Unify(t2Var, t1, sub)
	}
	// Handle old type system compatibility: TCon might represent String
	if t2Con, ok := t2.(*TCon); ok {
		// If trying to unify list with string, fail with better error
		if t2Con.Name == "String" {
			return nil, fmt.Errorf("type mismatch: cannot use list where string expected")
		}
		// Other TCon cases fail as before
		return nil, fmt.Errorf("cannot unify list type with %T", t2)
	}
	return nil, fmt.Errorf("cannot unify list type with %T", t2)
}

// unifyArrays unifies two array types
func (u *Unifier) unifyArrays(t1 *TArray, t2 Type, sub Substitution) (Substitution, error) {
	if t2Array, ok := t2.(*TArray); ok {
		return u.Unify(t1.Element, t2Array.Element, sub)
	}
	// Special case: TArray{Element: a} can unify with TApp("Array", a)
	if t2App, ok := t2.(*TApp); ok {
		h2, a2 := decomposeApp(t2App)
		if headCon, ok := h2.(*TCon); ok && headCon.Name == "Array" && len(a2) == 1 {
			return u.Unify(t1.Element, a2[0], sub)
		}
	}
	if t2Var, ok := t2.(*TVar2); ok {
		// Swap and retry
		return u.Unify(t2Var, t1, sub)
	}
	return nil, fmt.Errorf("cannot unify array type with %T", t2)
}

// unifyTuples unifies two tuple types
func (u *Unifier) unifyTuples(t1 *TTuple, t2 Type, sub Substitution) (Substitution, error) {
	if t2Tuple, ok := t2.(*TTuple); ok {
		if len(t1.Elements) != len(t2Tuple.Elements) {
			return nil, fmt.Errorf("tuple size mismatch: %d vs %d", len(t1.Elements), len(t2Tuple.Elements))
		}
		for i := range t1.Elements {
			var err error
			sub, err = u.Unify(t1.Elements[i], t2Tuple.Elements[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify tuple element %d: %w", i, err)
			}
		}
		return sub, nil
	}
	if t2Var, ok := t2.(*TVar2); ok {
		// Swap and retry
		return u.Unify(t2Var, t1, sub)
	}
	return nil, fmt.Errorf("cannot unify tuple type with %T", t2)
}

// unifyTypeApps unifies two type applications
func (u *Unifier) unifyTypeApps(t1 *TApp, t2 Type, sub Substitution) (Substitution, error) {
	if _, ok := t2.(*TApp); ok {
		// Both are type applications - decompose and unify
		h1, a1 := decomposeApp(t1)
		h2, a2 := decomposeApp(t2)

		// Require same head constructor
		if !equalHead(h1, h2) {
			return nil, fmt.Errorf("type constructor mismatch: %s vs %s", h1.String(), h2.String())
		}

		// Arity check
		if len(a1) != len(a2) {
			return nil, fmt.Errorf("arity mismatch: %s expects %d args, got %d", h1.String(), len(a1), len(a2))
		}

		// Unify each pair of args in order
		for i := range a1 {
			var err error
			sub, err = u.Unify(a1[i], a2[i], sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify type argument %d: %w", i, err)
			}
		}
		return sub, nil
	}
	// DX-17: TApp("list", a) can unify with TList{Element: a}
	// Use AsList helper to recognize both list representations
	if t2List, ok := t2.(*TList); ok {
		if elem, ok := AsList(t1); ok {
			// TApp("list", a) ~ TList{Element: a}
			return u.Unify(elem, t2List.Element, sub)
		}
	}
	// Special case: TApp("Array", a) can unify with TArray{Element: a}
	if t2Array, ok := t2.(*TArray); ok {
		h1, a1 := decomposeApp(t1)
		if headCon, ok := h1.(*TCon); ok && headCon.Name == "Array" && len(a1) == 1 {
			// TApp("Array", a) ~ TArray{Element: a}
			return u.Unify(a1[0], t2Array.Element, sub)
		}
	}
	if t2Var, ok := t2.(*TVar2); ok {
		// Swap and retry
		return u.Unify(t2Var, t1, sub)
	}
	// M-TAPP-FIX: Handle TApp vs TCon
	// TApp(Option, [int]) should not unify with TCon("Option") - different arities
	// But we provide a better error message
	if t2Con, ok := t2.(*TCon); ok {
		h1, a1 := decomposeApp(t1)
		if h1Con, ok := h1.(*TCon); ok && h1Con.Name == t2Con.Name {
			// Same constructor name but different application
			return nil, fmt.Errorf("type %s expects %d type argument(s), but got 0 (did you mean %s?)",
				t2Con.Name, len(a1), t1.String())
		}
		return nil, fmt.Errorf("cannot unify %s with %s", t1.String(), t2Con.Name)
	}
	return nil, fmt.Errorf("cannot unify type application with %T", t2)
}

// decomposeApp decomposes a type application into (head constructor, args)
// Handles nested TApp chains: TApp(TApp(Result, A), B) → (Result, [A, B])
func decomposeApp(t Type) (head Type, args []Type) {
	// Recursively collect args from nested TApp
	switch app := t.(type) {
	case *TApp:
		// Get args from this node
		currentArgs := app.Args

		// Check if constructor is another TApp (nested application)
		innerHead, innerArgs := decomposeApp(app.Constructor)

		// Combine args: inner args first, then current args
		allArgs := make([]Type, 0, len(innerArgs)+len(currentArgs))
		allArgs = append(allArgs, innerArgs...)
		allArgs = append(allArgs, currentArgs...)

		return innerHead, allArgs
	default:
		// Base case: not a TApp, this is the head
		return t, nil
	}
}

// equalHead checks if two type heads are equal
// For now, we check if they're the same TCon by name
func equalHead(h1, h2 Type) bool {
	// Both should be TCon for well-formed type applications
	con1, ok1 := h1.(*TCon)
	con2, ok2 := h2.(*TCon)

	if ok1 && ok2 {
		return con1.Name == con2.Name
	}

	// Fallback: use Equals for other cases (e.g., type variables)
	return h1.Equals(h2)
}

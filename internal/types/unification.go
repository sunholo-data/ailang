package types

import (
	"fmt"
)

// Substitution maps variable names to types
type Substitution map[string]Type

// Unifier handles type unification with occurs check
type Unifier struct {
	rowUnifier *RowUnifier
}

// NewUnifier creates a new unifier
func NewUnifier() *Unifier {
	return &Unifier{
		rowUnifier: NewRowUnifier(),
	}
}

// Unify attempts to unify two types, returning an updated substitution
func (u *Unifier) Unify(t1, t2 Type, sub Substitution) (Substitution, error) {
	// Apply current substitution
	t1 = ApplySubstitution(sub, t1)
	t2 = ApplySubstitution(sub, t2)

	// Check if already equal
	if t1.Equals(t2) {
		return sub, nil
	}

	switch t1 := t1.(type) {
	case *TVar2:
		// Type variable unification
		if u.occurs(t1.Name, t2, t1.Kind) {
			return nil, fmt.Errorf("occurs check failed: %s occurs in %s", t1.Name, t2.String())
		}
		if !u.kindsCompatible(t1.Kind, GetKind(t2)) {
			return nil, fmt.Errorf("kind mismatch: variable %s has kind %s, but %s has kind %s",
				t1.Name, t1.Kind, t2.String(), GetKind(t2))
		}
		sub[t1.Name] = t2
		return sub, nil

	case *RowVar:
		// Row variable unification
		if u.occurs(t1.Name, t2, t1.Kind) {
			return nil, fmt.Errorf("occurs check failed: row variable %s occurs in %s", t1.Name, t2.String())
		}
		if !u.kindsCompatible(t1.Kind, GetKind(t2)) {
			return nil, fmt.Errorf("kind mismatch: row variable %s has kind %s, but %s has kind %s",
				t1.Name, t1.Kind, t2.String(), GetKind(t2))
		}
		sub[t1.Name] = t2
		return sub, nil

	case *Row:
		// Row unification
		if t2Row, ok := t2.(*Row); ok {
			return u.rowUnifier.UnifyRows(t1, t2Row, sub)
		}
		if t2Var, ok := t2.(*RowVar); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify row with %T", t2)

	case *TCon:
		// Type constructor unification
		if t2Con, ok := t2.(*TCon); ok {
			if t1.Name == t2Con.Name {
				return sub, nil
			}
			return nil, fmt.Errorf("cannot unify type constructors: %s vs %s", t1.Name, t2Con.Name)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify type constructor %s with %T", t1.Name, t2)

	case *TFunc2:
		// Function type unification
		if t2Func, ok := t2.(*TFunc2); ok {
			if len(t1.Params) != len(t2Func.Params) {
				return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(t1.Params), len(t2Func.Params))
			}

			// Unify parameters
			for i := range t1.Params {
				var err error
				sub, err = u.Unify(t1.Params[i], t2Func.Params[i], sub)
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
			return u.Unify(t1.Return, t2Func.Return, sub)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify function type with %T", t2)

	case *TList:
		// List type unification
		if t2List, ok := t2.(*TList); ok {
			return u.Unify(t1.Element, t2List.Element, sub)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify list type with %T", t2)

	case *TTuple:
		// Tuple type unification
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

	case *TRecord2:
		// Record type unification
		if t2Rec, ok := t2.(*TRecord2); ok {
			if t1.Row == nil && t2Rec.Row == nil {
				return sub, nil
			}
			if t1.Row != nil && t2Rec.Row != nil {
				return u.Unify(t1.Row, t2Rec.Row, sub)
			}
			return nil, fmt.Errorf("cannot unify record with different row presence")
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify record type with %T", t2)

	default:
		// Handle old types for compatibility
		if t2Var, ok := t2.(*TVar2); ok {
			return u.Unify(t2Var, t1, sub)
		}
		if t2RowVar, ok := t2.(*RowVar); ok {
			return u.Unify(t2RowVar, t1, sub)
		}
		return nil, fmt.Errorf("unhandled type in unification: %T", t1)
	}
}

// occurs performs the occurs check - ensures variable doesn't occur in type
func (u *Unifier) occurs(varName string, t Type, varKind Kind) bool {
	switch t := t.(type) {
	case *TVar2:
		// Type vars only occur in type vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *RowVar:
		// Row vars only occur in row vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *Row:
		// Check if var occurs in tail
		if t.Tail != nil && u.occurs(varName, t.Tail, varKind) {
			return true
		}
		// Check if var occurs in label types (for record rows)
		if t.Kind.Equals(RecordRow) {
			for _, typ := range t.Labels {
				if u.occurs(varName, typ, varKind) {
					return true
				}
			}
		}
		// Effect labels don't contain types, just names
		return false

	case *TCon:
		return false

	case *TFunc2:
		// Check params, return, and effect row
		for _, p := range t.Params {
			if u.occurs(varName, p, varKind) {
				return true
			}
		}
		if u.occurs(varName, t.Return, varKind) {
			return true
		}
		if t.EffectRow != nil && u.occurs(varName, t.EffectRow, varKind) {
			return true
		}
		return false

	case *TList:
		return u.occurs(varName, t.Element, varKind)

	case *TTuple:
		for _, elem := range t.Elements {
			if u.occurs(varName, elem, varKind) {
				return true
			}
		}
		return false

	case *TRecord2:
		if t.Row != nil {
			return u.occurs(varName, t.Row, varKind)
		}
		return false

	default:
		// For old types, no occurs
		return false
	}
}

// kindsCompatible checks if two kinds are compatible for unification
func (u *Unifier) kindsCompatible(k1, k2 Kind) bool {
	return k1.Equals(k2)
}

// ApplySubstitution applies a substitution to a type
func ApplySubstitution(sub Substitution, t Type) Type {
	if sub == nil || len(sub) == 0 {
		return t
	}
	return t.Substitute(sub)
}

// ComposeSubstitutions composes two substitutions
func ComposeSubstitutions(s1, s2 Substitution) Substitution {
	result := make(Substitution)
	
	// Apply s2 to all values in s1
	for k, v := range s1 {
		result[k] = ApplySubstitution(s2, v)
	}
	
	// Add all mappings from s2 not in s1
	for k, v := range s2 {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}
	
	return result
}
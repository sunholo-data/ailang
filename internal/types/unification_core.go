package types

import (
	"fmt"
)

// Substitution maps variable names to types
type Substitution map[string]Type

// Unifier handles type unification with occurs check
type Unifier struct {
	rowUnifier *RowUnifier
	depth      int // Track recursion depth for cycle detection
	// aliasEnv maps type alias names to their underlying types
	// M-BUGFIX: Used to expand aliases during unification (e.g., Coord -> {x: int, y: int})
	aliasEnv map[string]Type
}

// Maximum recursion depth before we assume a cycle
const maxUnifyDepth = 1000

// NewUnifier creates a new unifier
func NewUnifier() *Unifier {
	return &Unifier{
		rowUnifier: NewRowUnifier(),
		depth:      0,
		aliasEnv:   nil, // No alias expansion by default
	}
}

// NewUnifierWithAliases creates a unifier with type alias expansion support
// M-BUGFIX: This allows ADT variants with alias parameters to work correctly
func NewUnifierWithAliases(aliases map[string]Type) *Unifier {
	return &Unifier{
		rowUnifier: NewRowUnifier(),
		depth:      0,
		aliasEnv:   aliases,
	}
}

// expandAlias expands a type alias to its underlying type if applicable
// M-BUGFIX: Handles type aliases like `type Coord = {x: int, y: int}`
// M-CROSS-MODULE: When expanding TCon to TRecord, preserve the type name
func (u *Unifier) expandAlias(t Type) Type {
	if u.aliasEnv == nil {
		return t
	}
	if con, ok := t.(*TCon); ok {
		if target, exists := u.aliasEnv[con.Name]; exists {
			// M-CROSS-MODULE: If target is a TRecord, set its TypeName
			// This preserves the nominal type identity through unification
			if rec, ok := target.(*TRecord); ok && rec.TypeName == "" {
				// Create a copy with the type name set
				return &TRecord{
					Fields:   rec.Fields,
					Row:      rec.Row,
					TypeName: con.Name,
				}
			}
			return target
		}
	}
	return t
}

// Unify attempts to unify two types, returning an updated substitution
func (u *Unifier) Unify(t1, t2 Type, sub Substitution) (Substitution, error) {
	// Depth check to catch infinite recursion
	u.depth++
	defer func() { u.depth-- }()

	if u.depth > maxUnifyDepth {
		// Panic with useful debug info instead of hanging forever
		panic(fmt.Sprintf("unification depth exceeded %d - likely cyclic types:\n  t1: %T\n  t2: %T\n  substitution size: %d",
			maxUnifyDepth, t1, t2, len(sub)))
	}

	// Apply current substitution with cycle detection
	t1 = ApplySubstitution(sub, t1)
	t2 = ApplySubstitution(sub, t2)

	// M-FIX-FLOAT-OP: Guard against nil types after substitution
	if t1 == nil || t2 == nil {
		return nil, fmt.Errorf("cannot unify nil types: t1=%v, t2=%v", t1, t2)
	}

	// M-BUGFIX: Expand type aliases before unification
	// This allows `type Coord = {x: int, y: int}` to unify with {x: int, y: int}
	t1 = u.expandAlias(t1)
	t2 = u.expandAlias(t2)

	// Check if already equal with cycle detection
	if SafeEquals(t1, t2) {
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
		// M-FIX-RECORD-UPDATE: Handle TVar2 with row kind
		if t2Var, ok := t2.(*TVar2); ok {
			// Check if it's a row variable (has row kind)
			if _, isRowKind := t2Var.Kind.(*KRow); isRowKind {
				// Swap and retry as row variable
				return u.Unify(t2Var, t1, sub)
			}
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
		return u.unifyFunctions(t1, t2, sub)

	case *TFunc:
		// M-FIX-FLOAT-OP: Handle old TFunc type that may appear after substitution chain resolution
		return u.unifyTFunc(t1, t2, sub)

	case *TList:
		return u.unifyLists(t1, t2, sub)

	case *TArray:
		return u.unifyArrays(t1, t2, sub)

	case *TTuple:
		return u.unifyTuples(t1, t2, sub)

	case *TRecord2:
		return u.unifyRecord2(t1, t2, sub)

	case *TRecord:
		return u.unifyRecord(t1, t2, sub)

	case *TRecordOpen:
		return u.unifyRecordOpen(t1, t2, sub)

	case *TApp:
		return u.unifyTypeApps(t1, t2, sub)

	default:
		// Unhandled type - no more compatibility for old type system
		return nil, fmt.Errorf("unhandled type in unification: %T", t1)
	}
}

// kindsCompatible checks if two kinds are compatible for unification
func (u *Unifier) kindsCompatible(k1, k2 Kind) bool {
	return k1.Equals(k2)
}

package types

// SafeEquals compares two types for equality with cycle detection
// This prevents infinite loops when comparing cyclic type graphs
func SafeEquals(t1, t2 Type) bool {
	visited := getTypePairBoolMap()
	defer putTypePairBoolMap(visited)
	return safeEqualsWithVisited(t1, t2, visited)
}

// typePair is used as a key for tracking visited pairs during equality checking
type typePair struct {
	t1, t2 Type
}

// safeEqualsWithVisited compares types with cycle detection
func safeEqualsWithVisited(t1, t2 Type, visited map[typePair]bool) bool {
	// Fast path: pointer equality
	if t1 == t2 {
		return true
	}

	// Handle nil cases
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	// Create pair key for cycle detection
	pair := typePair{t1, t2}
	if visited[pair] {
		return true // Already comparing this pair - assume equal to break cycle
	}
	visited[pair] = true

	switch typ1 := t1.(type) {
	case *TVar:
		if typ2, ok := t2.(*TVar); ok {
			return typ1.Name == typ2.Name
		}
		return false

	case *TVar2:
		if typ2, ok := t2.(*TVar2); ok {
			return typ1.Name == typ2.Name && typ1.Kind.Equals(typ2.Kind)
		}
		return false

	case *RowVar:
		if typ2, ok := t2.(*RowVar); ok {
			return typ1.Name == typ2.Name && typ1.Kind.Equals(typ2.Kind)
		}
		return false

	case *TCon:
		if typ2, ok := t2.(*TCon); ok {
			return typ1.Name == typ2.Name
		}
		return false

	case *TList:
		if typ2, ok := t2.(*TList); ok {
			return safeEqualsWithVisited(typ1.Element, typ2.Element, visited)
		}
		return false

	case *TArray:
		if typ2, ok := t2.(*TArray); ok {
			return safeEqualsWithVisited(typ1.Element, typ2.Element, visited)
		}
		return false

	case *TTuple:
		if typ2, ok := t2.(*TTuple); ok {
			if len(typ1.Elements) != len(typ2.Elements) {
				return false
			}
			for i := range typ1.Elements {
				if !safeEqualsWithVisited(typ1.Elements[i], typ2.Elements[i], visited) {
					return false
				}
			}
			return true
		}
		return false

	case *TFunc2:
		if typ2, ok := t2.(*TFunc2); ok {
			if len(typ1.Params) != len(typ2.Params) {
				return false
			}
			for i := range typ1.Params {
				if !safeEqualsWithVisited(typ1.Params[i], typ2.Params[i], visited) {
					return false
				}
			}
			if !safeEqualsWithVisited(typ1.Return, typ2.Return, visited) {
				return false
			}
			// Compare effect rows
			if typ1.EffectRow == nil && typ2.EffectRow == nil {
				return true
			}
			if typ1.EffectRow != nil && typ2.EffectRow != nil {
				return safeEqualsWithVisited(typ1.EffectRow, typ2.EffectRow, visited)
			}
			return false
		}
		return false

	case *TRecord:
		if typ2, ok := t2.(*TRecord); ok {
			if len(typ1.Fields) != len(typ2.Fields) {
				return false
			}
			for k, v1 := range typ1.Fields {
				v2, ok := typ2.Fields[k]
				if !ok || !safeEqualsWithVisited(v1, v2, visited) {
					return false
				}
			}
			if typ1.Row == nil && typ2.Row == nil {
				return true
			}
			if typ1.Row != nil && typ2.Row != nil {
				return safeEqualsWithVisited(typ1.Row, typ2.Row, visited)
			}
			return false
		}
		return false

	case *TRecord2:
		if typ2, ok := t2.(*TRecord2); ok {
			if typ1.Row == nil && typ2.Row == nil {
				return true
			}
			if typ1.Row != nil && typ2.Row != nil {
				return safeEqualsWithVisited(typ1.Row, typ2.Row, visited)
			}
			return false
		}
		return false

	case *Row:
		if typ2, ok := t2.(*Row); ok {
			if !typ1.Kind.Equals(typ2.Kind) {
				return false
			}
			if len(typ1.Labels) != len(typ2.Labels) {
				return false
			}
			for k, v1 := range typ1.Labels {
				v2, ok := typ2.Labels[k]
				if !ok || !safeEqualsWithVisited(v1, v2, visited) {
					return false
				}
			}
			if typ1.Tail == nil && typ2.Tail == nil {
				return true
			}
			if typ1.Tail != nil && typ2.Tail != nil {
				return safeEqualsWithVisited(typ1.Tail, typ2.Tail, visited)
			}
			return false
		}
		return false

	case *TApp:
		if typ2, ok := t2.(*TApp); ok {
			if !safeEqualsWithVisited(typ1.Constructor, typ2.Constructor, visited) {
				return false
			}
			if len(typ1.Args) != len(typ2.Args) {
				return false
			}
			for i := range typ1.Args {
				if !safeEqualsWithVisited(typ1.Args[i], typ2.Args[i], visited) {
					return false
				}
			}
			return true
		}
		return false

	default:
		// Fall back to interface method for other types
		// This is safe because we've already checked for cycles at this level
		return t1.Equals(t2)
	}
}

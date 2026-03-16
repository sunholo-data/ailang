package types

// occurs performs the occurs check - ensures variable doesn't occur in type
// This is the public entry point that creates a visited set for cycle detection
func (u *Unifier) occurs(varName string, t Type, varKind Kind) bool {
	visited := getTypeBoolMap()
	defer putTypeBoolMap(visited)
	return u.occursWithVisited(varName, t, varKind, visited)
}

// occursWithVisited is the internal implementation with cycle detection
// Cyclic types (e.g., List[NPCState] where NPCState contains List[NPCState])
// would cause infinite recursion without tracking visited types
func (u *Unifier) occursWithVisited(varName string, t Type, varKind Kind, visited map[Type]bool) bool {
	// Cycle detection: if we've already started checking this type, don't recurse
	if visited[t] {
		return false // Already visited - not an occurs error, just a cycle
	}
	visited[t] = true

	switch t := t.(type) {
	case *TVar2:
		// Type vars only occur in type vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *RowVar:
		// Row vars only occur in row vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *Row:
		// Check if var occurs in tail
		if t.Tail != nil && u.occursWithVisited(varName, t.Tail, varKind, visited) {
			return true
		}
		// Check if var occurs in label types (for record rows)
		if t.Kind.Equals(RecordRow) {
			for _, typ := range t.Labels {
				if u.occursWithVisited(varName, typ, varKind, visited) {
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
			if u.occursWithVisited(varName, p, varKind, visited) {
				return true
			}
		}
		if u.occursWithVisited(varName, t.Return, varKind, visited) {
			return true
		}
		if t.EffectRow != nil && u.occursWithVisited(varName, t.EffectRow, varKind, visited) {
			return true
		}
		return false

	case *TList:
		return u.occursWithVisited(varName, t.Element, varKind, visited)

	case *TArray:
		return u.occursWithVisited(varName, t.Element, varKind, visited)

	case *TTuple:
		for _, elem := range t.Elements {
			if u.occursWithVisited(varName, elem, varKind, visited) {
				return true
			}
		}
		return false

	case *TRecord2:
		if t.Row != nil {
			return u.occursWithVisited(varName, t.Row, varKind, visited)
		}
		return false

	case *TApp:
		// Check constructor and all args
		if u.occursWithVisited(varName, t.Constructor, varKind, visited) {
			return true
		}
		for _, arg := range t.Args {
			if u.occursWithVisited(varName, arg, varKind, visited) {
				return true
			}
		}
		return false

	default:
		// For old types, no occurs
		return false
	}
}

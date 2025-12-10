package types

// ApplySubstitution applies a substitution to a type with cycle detection
func ApplySubstitution(sub Substitution, t Type) Type {
	if len(sub) == 0 {
		return t
	}
	visited := make(map[Type]Type)
	return safeSubstitute(t, sub, visited)
}

// safeSubstitute applies substitution with cycle detection to prevent infinite loops
// on cyclic type graphs (e.g., recursive types like List[NPCState] where NPCState contains List[NPCState])
func safeSubstitute(t Type, sub Substitution, visited map[Type]Type) Type {
	// Check if we've already visited this exact type pointer
	if result, ok := visited[t]; ok {
		return result // Return the already-computed result to break the cycle
	}

	// Mark as visited with the original type first (to handle cycles)
	// We'll update this if we actually transform the type
	visited[t] = t

	switch typ := t.(type) {
	case *TVar:
		if newType, ok := sub[typ.Name]; ok {
			// M-FIX-FLOAT-OP: Follow substitution chains recursively
			// If α3 -> α7 and α7 -> float, then α3 should resolve to float
			result := safeSubstitute(newType, sub, visited)
			visited[t] = result
			return result
		}
		return t

	case *TVar2:
		if newType, ok := sub[typ.Name]; ok {
			// M-FIX-FLOAT-OP: Follow substitution chains recursively
			// If α3 -> α7 and α7 -> float, then α3 should resolve to float
			result := safeSubstitute(newType, sub, visited)
			visited[t] = result
			return result
		}
		return t

	case *RowVar:
		if newType, ok := sub[typ.Name]; ok {
			// M-FIX-FLOAT-OP: Follow substitution chains recursively
			result := safeSubstitute(newType, sub, visited)
			visited[t] = result
			return result
		}
		return t

	case *TCon:
		return t // Type constructors don't change

	case *TList:
		elem := safeSubstitute(typ.Element, sub, visited)
		if elem == typ.Element {
			return t
		}
		result := &TList{Element: elem}
		visited[t] = result
		return result

	case *TArray:
		elem := safeSubstitute(typ.Element, sub, visited)
		if elem == typ.Element {
			return t
		}
		result := &TArray{Element: elem}
		visited[t] = result
		return result

	case *TTuple:
		changed := false
		elems := make([]Type, len(typ.Elements))
		for i, e := range typ.Elements {
			elems[i] = safeSubstitute(e, sub, visited)
			if elems[i] != e {
				changed = true
			}
		}
		if !changed {
			return t
		}
		result := &TTuple{Elements: elems}
		visited[t] = result
		return result

	case *TFunc:
		changed := false
		params := make([]Type, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = safeSubstitute(p, sub, visited)
			if params[i] != p {
				changed = true
			}
		}
		ret := safeSubstitute(typ.Return, sub, visited)
		if ret != typ.Return {
			changed = true
		}
		if !changed {
			return t
		}
		result := &TFunc{Params: params, Return: ret, Effects: typ.Effects}
		visited[t] = result
		return result

	case *TFunc2:
		changed := false
		params := make([]Type, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = safeSubstitute(p, sub, visited)
			if params[i] != p {
				changed = true
			}
		}
		ret := safeSubstitute(typ.Return, sub, visited)
		if ret != typ.Return {
			changed = true
		}
		var effectRow *Row
		if typ.EffectRow != nil {
			effSub := safeSubstitute(typ.EffectRow, sub, visited)
			if effSub != typ.EffectRow {
				changed = true
			}
			if row, ok := effSub.(*Row); ok {
				effectRow = row
			}
		}
		if !changed {
			return t
		}
		result := &TFunc2{Params: params, Return: ret, EffectRow: effectRow}
		visited[t] = result
		return result

	case *TRecord:
		changed := false
		fields := make(map[string]Type)
		for name, fieldType := range typ.Fields {
			fields[name] = safeSubstitute(fieldType, sub, visited)
			if fields[name] != fieldType {
				changed = true
			}
		}
		var row Type
		if typ.Row != nil {
			row = safeSubstitute(typ.Row, sub, visited)
			if row != typ.Row {
				changed = true
			}
		}
		if !changed {
			return t
		}
		result := &TRecord{Fields: fields, Row: row}
		visited[t] = result
		return result

	case *TRecord2:
		if typ.Row == nil {
			return t
		}
		rowSub := safeSubstitute(typ.Row, sub, visited)
		if rowSub == typ.Row {
			return t
		}
		result := &TRecord2{Row: rowSub.(*Row)}
		visited[t] = result
		return result

	case *Row:
		changed := false
		labels := make(map[string]Type)
		for name, labelType := range typ.Labels {
			labels[name] = safeSubstitute(labelType, sub, visited)
			if labels[name] != labelType {
				changed = true
			}
		}
		var tail *RowVar
		if typ.Tail != nil {
			if tailSub, ok := sub[typ.Tail.Name]; ok {
				changed = true
				if subRow, ok := tailSub.(*Row); ok {
					// Merge labels from substituted row
					for k, v := range subRow.Labels {
						labels[k] = v
					}
					tail = subRow.Tail
				} else if subVar, ok := tailSub.(*RowVar); ok {
					tail = subVar
				}
			} else {
				tail = typ.Tail
			}
		}
		if !changed {
			return t
		}
		result := &Row{Kind: typ.Kind, Labels: labels, Tail: tail}
		visited[t] = result
		return result

	case *TApp:
		constr := safeSubstitute(typ.Constructor, sub, visited)
		changed := constr != typ.Constructor
		args := make([]Type, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = safeSubstitute(arg, sub, visited)
			if args[i] != arg {
				changed = true
			}
		}
		if !changed {
			return t
		}
		result := &TApp{Constructor: constr, Args: args}
		visited[t] = result
		return result

	default:
		// Fall back to the interface method for other types
		// but with the visited guard already in place
		return t.Substitute(sub)
	}
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

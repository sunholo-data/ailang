package types

import (
	"fmt"
	"sort"
	"strings"
)

// MaxStringifyDepth is the maximum depth for type stringification.
// Beyond this, types are truncated with "..." to prevent hangs on cyclic types.
const MaxStringifyDepth = 100

// SafeTypeString returns a string representation of a type with depth limiting.
// If the depth exceeds MaxStringifyDepth, returns a truncated representation.
// This prevents infinite loops on cyclic type graphs.
func SafeTypeString(t Type) string {
	visited := getTypeBoolMap()
	defer putTypeBoolMap(visited)
	return safeTypeStringWithDepth(t, 0, visited)
}

// safeTypeStringWithDepth is the internal depth-limited string implementation.
func safeTypeStringWithDepth(t Type, depth int, visited map[Type]bool) string {
	if t == nil {
		return "nil"
	}

	// Check depth limit
	if depth > MaxStringifyDepth {
		return fmt.Sprintf("<%T...depth limit>", t)
	}

	// Check for cycles
	if visited[t] {
		return fmt.Sprintf("<%T...cycle>", t)
	}
	visited[t] = true
	defer delete(visited, t)

	switch typ := t.(type) {
	case *TVar:
		return typ.Name

	case *TVar2:
		return typ.Name

	case *TCon:
		return typ.Name

	case *TFunc2:
		params := make([]string, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = safeTypeStringWithDepth(p, depth+1, visited)
		}
		ret := safeTypeStringWithDepth(typ.Return, depth+1, visited)

		effectStr := ""
		if typ.EffectRow != nil && (len(typ.EffectRow.Labels) > 0 || typ.EffectRow.Tail != nil) {
			effectStr = fmt.Sprintf(" ! %s", safeTypeStringWithDepth(typ.EffectRow, depth+1, visited))
		}

		if len(params) == 1 {
			return fmt.Sprintf("%s -> %s%s", params[0], ret, effectStr)
		}
		return fmt.Sprintf("(%s) -> %s%s", strings.Join(params, ", "), ret, effectStr)

	case *TList:
		elem := safeTypeStringWithDepth(typ.Element, depth+1, visited)
		return fmt.Sprintf("[%s]", elem)

	case *TArray:
		elem := safeTypeStringWithDepth(typ.Element, depth+1, visited)
		return fmt.Sprintf("Array[%s]", elem)

	case *TTuple:
		elems := make([]string, len(typ.Elements))
		for i, e := range typ.Elements {
			elems[i] = safeTypeStringWithDepth(e, depth+1, visited)
		}
		return fmt.Sprintf("(%s)", strings.Join(elems, ", "))

	case *TRecord:
		var fields []string
		for name, fieldType := range typ.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", name, safeTypeStringWithDepth(fieldType, depth+1, visited)))
		}
		if typ.Row != nil {
			fields = append(fields, fmt.Sprintf("...%s", safeTypeStringWithDepth(typ.Row, depth+1, visited)))
		}
		return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))

	case *TRecordOpen:
		var fields []string
		for name, fieldType := range typ.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", name, safeTypeStringWithDepth(fieldType, depth+1, visited)))
		}
		if typ.Row != nil {
			fields = append(fields, fmt.Sprintf("| %s", safeTypeStringWithDepth(typ.Row, depth+1, visited)))
		}
		return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))

	case *TApp:
		args := make([]string, len(typ.Args))
		for i, a := range typ.Args {
			args[i] = safeTypeStringWithDepth(a, depth+1, visited)
		}
		con := safeTypeStringWithDepth(typ.Constructor, depth+1, visited)
		return fmt.Sprintf("%s[%s]", con, strings.Join(args, ", "))

	case *Row:
		// Sort labels for canonical representation
		var keys []string
		for k := range typ.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			if typ.Kind.Equals(EffectRow) {
				parts = append(parts, k)
			} else {
				parts = append(parts, fmt.Sprintf("%s: %s", k, safeTypeStringWithDepth(typ.Labels[k], depth+1, visited)))
			}
		}

		if typ.Tail != nil {
			parts = append(parts, "..."+typ.Tail.Name)
		}

		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))

	case *RowVar:
		return typ.Name

	default:
		// For unknown types, use the standard String() method
		// This is safe as a fallback since we've already checked depth
		return t.String()
	}
}

// TruncatedTypeString returns a string representation that's guaranteed to be short.
// Useful for error messages and logging where space is limited.
func TruncatedTypeString(t Type, maxLen int) string {
	s := SafeTypeString(t)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

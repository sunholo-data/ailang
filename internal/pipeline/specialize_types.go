package pipeline

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/types"
)

// canonicalTypeFingerprint produces a stable, normalized string representation of types
//
// Requirements:
// - Fully qualified type constructor names
// - Normalized ordering for row/effect sets (deterministic iteration)
// - Type variables removed (only monotypes supported)
// - Hash suffix to prevent collisions
func canonicalTypeFingerprint(typs []types.Type) string {
	if len(typs) == 0 {
		return "unit"
	}

	// Build normalized string representation
	var parts []string
	for _, t := range typs {
		parts = append(parts, typeToCanonicalString(t))
	}

	// Sort for determinism (in case types come in different orders)
	sort.Strings(parts)

	// Join with stable separator
	canonical := strings.Join(parts, ":")

	// Add hash suffix for collision resistance
	hash := sha256.Sum256([]byte(canonical))
	shortHash := fmt.Sprintf("%x", hash[:2]) // First 2 bytes = 4 hex chars

	return canonical + "$" + shortHash
}

// typeToCanonicalString converts a type to its canonical string representation
func typeToCanonicalString(t types.Type) string {
	switch typ := t.(type) {
	case *types.TCon:
		return typ.Name
	case *types.TVar:
		// Type variables should not appear in monomorphized code
		// This indicates the type is still polymorphic
		return fmt.Sprintf("α_%s", typ.Name)
	case *types.TApp:
		// Application: Constructor(arg1, arg2, ...)
		args := make([]string, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = typeToCanonicalString(arg)
		}
		// Constructor is a Type, need to extract its name
		conName := "Unknown"
		if con, ok := typ.Constructor.(*types.TCon); ok {
			conName = con.Name
		}
		return fmt.Sprintf("%s(%s)", conName, strings.Join(args, ","))
	case *types.TFunc2:
		// Function: param1 -> param2 -> ... -> ret ! effects
		params := make([]string, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = typeToCanonicalString(p)
		}
		ret := typeToCanonicalString(typ.Return)

		// Include effects if present
		if typ.EffectRow != nil && len(typ.EffectRow.Labels) > 0 {
			// Sort effect labels for determinism
			var effects []string
			for eff := range typ.EffectRow.Labels {
				effects = append(effects, eff)
			}
			sort.Strings(effects)
			return fmt.Sprintf("(%s -> %s ! {%s})", strings.Join(params, " -> "), ret, strings.Join(effects, ","))
		}

		return fmt.Sprintf("(%s -> %s)", strings.Join(params, " -> "), ret)
	case *types.TRecord:
		// Record: {field1: type1, field2: type2, ...}
		if len(typ.Fields) == 0 {
			return "{}"
		}

		// Sort fields for determinism
		var fields []string
		for field, ftyp := range typ.Fields {
			fields = append(fields, fmt.Sprintf("%s:%s", field, typeToCanonicalString(ftyp)))
		}
		sort.Strings(fields)

		return fmt.Sprintf("{%s}", strings.Join(fields, ","))
	default:
		// Fallback for unknown types
		return fmt.Sprintf("%T", t)
	}
}

// typeHeads extracts the head type constructors from a list of types
//
// Example:
//
//	[Int, List(Float), Result(String, Error)] → [Int, List, Result]
//
// This is used for generating readable specialized function names.
func typeHeads(typs []types.Type) []string {
	heads := make([]string, len(typs))
	for i, t := range typs {
		heads[i] = typeHead(t)
	}
	return heads
}

// typeHead extracts the outermost type constructor
func typeHead(t types.Type) string {
	switch typ := t.(type) {
	case *types.TCon:
		return typ.Name
	case *types.TApp:
		// Extract constructor name
		if con, ok := typ.Constructor.(*types.TCon); ok {
			return con.Name
		}
		return "App"
	case *types.TFunc2:
		return "Func"
	case *types.TRecord:
		return "Record"
	case *types.TVar:
		return fmt.Sprintf("α_%s", typ.Name)
	default:
		return "Unknown"
	}
}

// generateSpecializedName creates a deterministic, readable name for a specialized function
//
// Format: _originalName$Type1$Type2$...$hash
//
// Example:
//
//	max specialized for (Int, Int) → _max$Int$Int$2f
//	sort specialized for (List(Int)) → _sort$List$8a
func generateSpecializedName(defSym string, argTypes []types.Type, fingerprint string) string {
	heads := typeHeads(argTypes)

	// Extract hash suffix from fingerprint
	parts := strings.Split(fingerprint, "$")
	hashSuffix := ""
	if len(parts) > 0 {
		hashSuffix = parts[len(parts)-1]
	}

	// Build name: _sym$Type1$Type2$hash
	name := "_" + defSym
	for _, head := range heads {
		name += "$" + head
	}
	if hashSuffix != "" {
		name += "$" + hashSuffix
	}

	return name
}

// isPolymorphic checks if a type contains type variables
func isPolymorphic(t types.Type) bool {
	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   isPolymorphic(%s) type=%T\n", t, t)
	}
	switch typ := t.(type) {
	case *types.TVar:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TVar, returning true\n")
		}
		return true
	case *types.TVar2:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TVar2, returning true\n")
		}
		return true
	case *types.TApp:
		for i, arg := range typ.Args {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     TApp arg[%d]: %s\n", i, arg)
			}
			if isPolymorphic(arg) {
				return true
			}
		}
		return false
	case *types.TFunc2:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     TFunc2 with %d params\n", len(typ.Params))
		}
		for i, p := range typ.Params {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     Checking param[%d]: %s (type=%T)\n", i, p, p)
			}
			if isPolymorphic(p) {
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> param[%d] is polymorphic, returning true\n", i)
				}
				return true
			}
		}
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     Checking return: %s\n", typ.Return)
		}
		if isPolymorphic(typ.Return) {
			return true
		}
		// Check effect row
		if typ.EffectRow != nil {
			for _, eff := range typ.EffectRow.Labels {
				if isPolymorphic(eff) {
					return true
				}
			}
		}
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TFunc2 not polymorphic, returning false\n")
		}
		return false
	case *types.TRecord:
		for _, ftyp := range typ.Fields {
			if isPolymorphic(ftyp) {
				return true
			}
		}
		// Also check row variable
		if typ.Row != nil && isPolymorphic(typ.Row) {
			return true
		}
		return false
	default:
		return false
	}
}

// allConcrete checks if all types in a list are concrete (no type variables)
func allConcrete(typs []types.Type) bool {
	for _, t := range typs {
		if isPolymorphic(t) {
			return false
		}
	}
	return true
}

// substituteType applies a type substitution to a type
func substituteType(typ types.Type, subst map[string]types.Type) types.Type {
	switch t := typ.(type) {
	case *types.TVar:
		// If we have a substitution for this variable, use it
		if concrete, ok := subst[t.Name]; ok {
			return concrete
		}
		return typ
	case *types.TVar2:
		// TVar2 from the new type system (v2)
		// If we have a substitution for this variable, use it
		if concrete, ok := subst[t.Name]; ok {
			return concrete
		}
		return typ
	case *types.TFunc2:
		// Substitute in parameters and return type
		newParams := make([]types.Type, len(t.Params))
		for i, p := range t.Params {
			newParams[i] = substituteType(p, subst)
		}
		newReturn := substituteType(t.Return, subst)
		return &types.TFunc2{
			Params:    newParams,
			Return:    newReturn,
			EffectRow: t.EffectRow, // Keep effects as-is for now
		}
	case *types.TApp:
		// Substitute in constructor and args
		newConstructor := substituteType(t.Constructor, subst)
		newArgs := make([]types.Type, len(t.Args))
		for i, arg := range t.Args {
			newArgs[i] = substituteType(arg, subst)
		}
		return &types.TApp{
			Constructor: newConstructor,
			Args:        newArgs,
		}
	default:
		// For concrete types (TCon, etc.), no substitution needed
		return typ
	}
}

// extractParamTVars extracts type variable names from a function type's parameters
// For a type like (α1 -> α2 -> β), returns ["α1", "α2"]
// This is used to build the type substitution map during specialization
func extractParamTVars(funcType types.Type, expectedParams int) []string {
	var tvars []string

	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] extractParamTVars: funcType=%v, expectedParams=%d\n", funcType, expectedParams)
	}

	// Unwrap TFunc2 to get parameter types
	switch ft := funcType.(type) {
	case *types.TFunc2:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   TFunc2.Params=%v (len=%d)\n", ft.Params, len(ft.Params))
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   TFunc2.Return=%v\n", ft.Return)
		}

		// Collect TVars from parameters
		for i, paramType := range ft.Params {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Param[%d]: type=%T, value=%v\n", i, paramType, paramType)
			}

			if tvar, ok := paramType.(*types.TVar); ok {
				tvars = append(tvars, tvar.Name)
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Param[%d]: TVar %s\n", i, tvar.Name)
				}
			} else if tvar2, ok := paramType.(*types.TVar2); ok {
				// Maybe it's TVar2?
				tvars = append(tvars, tvar2.Name)
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Param[%d]: TVar2 %s\n", i, tvar2.Name)
				}
			} else {
				// Parameter is not a TVar (already concrete)
				// Add empty string as placeholder
				tvars = append(tvars, "")
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Param[%d]: Concrete type %v (type=%T)\n", i, paramType, paramType)
				}
			}
		}

		// If the function type has more parameters (curried), extract from Return
		if len(tvars) < expectedParams {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Need more params, recursing into Return...\n")
			}
			// Recursively extract from return type
			moreTVars := extractParamTVars(ft.Return, expectedParams-len(tvars))
			tvars = append(tvars, moreTVars...)
		}

	case *types.TVar:
		// The whole function is a type variable (shouldn't happen for monomorphization)
		// but handle gracefully
		return []string{}

	default:
		// Not a function type or TVar - no parameters to extract
		return []string{}
	}

	return tvars
}

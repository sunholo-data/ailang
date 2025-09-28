package types

import (
	"fmt"
	"sort"
	"strings"
)

// NormalizeTypeName produces a canonical string representation of a type
// for use in deterministic registry keys and pretty-printing.
// Examples:
//   - int → "Int"
//   - float → "Float"  
//   - [int] → "List<Int>"
//   - (int, float) → "Tuple<Int,Float>"
//   - {x: int, y: float} → "Record<x:Int,y:Float>"
//
// Design decision: We use angle brackets <> for type parameters and
// Tuple<T1,T2> (not Pair) for consistency across all parameterized types.
func NormalizeTypeName(t Type) string {
	// Guard against nil types
	if t == nil {
		return "<unknown>"
	}
	
	switch typ := t.(type) {
	case *TCon:
		// Normalize primitive type constructor names to canonical form
		switch typ.Name {
		case "int":
			return "Int"
		case "float":
			return "Float"
		case "string":
			return "String"
		case "bool":
			return "Bool"
		case "unit", "()":
			return "Unit"
		case "bytes":
			return "Bytes"
		default:
			// User-defined types: capitalize first letter
			if len(typ.Name) > 0 {
				return strings.ToUpper(typ.Name[:1]) + typ.Name[1:]
			}
			return typ.Name
		}
	
	case *TVar:
		// Type variables should not appear in normalized names (should be ground)
		// This is an error case but we handle it gracefully
		return fmt.Sprintf("_%s", typ.Name)
	
	case *TList:
		elemType := NormalizeTypeName(typ.Element)
		return fmt.Sprintf("List<%s>", elemType)
	
	case *TTuple:
		// Canonical tuple format: Tuple<T1,T2,...>
		// Decision: Always use "Tuple", never "Pair" for consistency
		var elems []string
		for _, e := range typ.Elements {
			elems = append(elems, NormalizeTypeName(e))
		}
		return fmt.Sprintf("Tuple<%s>", strings.Join(elems, ","))
	
	case *TRecord:
		// Canonical record format: Record<f1:T1,f2:T2,...>
		// Fields are sorted alphabetically for determinism
		var fields []string
		var fieldNames []string
		for name := range typ.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		
		for _, name := range fieldNames {
			fieldType := NormalizeTypeName(typ.Fields[name])
			fields = append(fields, fmt.Sprintf("%s:%s", name, fieldType))
		}
		
		result := fmt.Sprintf("Record<%s>", strings.Join(fields, ","))
		
		// Handle row polymorphism if present
		if typ.Row != nil {
			result += fmt.Sprintf("|%s", NormalizeTypeName(typ.Row))
		}
		
		return result
	
	case *TFunc:
		// Function types: Func<P1,P2,...->R>
		// We include effects if present: Func<P1,P2->R!{E1,E2}>
		var params []string
		for _, p := range typ.Params {
			params = append(params, NormalizeTypeName(p))
		}
		
		returnType := NormalizeTypeName(typ.Return)
		
		// Format with arrow inside angle brackets for clarity
		if len(params) == 0 {
			result := fmt.Sprintf("Func<()->%s>", returnType)
			if len(typ.Effects) > 0 {
				result = addEffectsToFunc(result, typ.Effects)
			}
			return result
		}
		
		result := fmt.Sprintf("Func<%s->%s>", 
			strings.Join(params, ","), 
			returnType)
		
		if len(typ.Effects) > 0 {
			result = addEffectsToFunc(result, typ.Effects)
		}
		
		return result
	
	case *TApp:
		// Type application: Maybe<Int>, Either<Int,String>
		constr := NormalizeTypeName(typ.Constructor)
		var args []string
		for _, a := range typ.Args {
			args = append(args, NormalizeTypeName(a))
		}
		return fmt.Sprintf("%s<%s>", constr, strings.Join(args, ","))
	
	// Handle v2 types if they exist
	case *TVar2:
		return fmt.Sprintf("_%s", typ.Name)
	
	case *TFunc2:
		var params []string
		for _, p := range typ.Params {
			params = append(params, NormalizeTypeName(p))
		}
		
		returnType := NormalizeTypeName(typ.Return)
		
		if len(params) == 0 {
			result := fmt.Sprintf("Func<()->%s>", returnType)
			if typ.EffectRow != nil {
				result = fmt.Sprintf("Func<()->%s!%s>", returnType, NormalizeTypeName(typ.EffectRow))
			}
			return result
		}
		
		result := fmt.Sprintf("Func<%s->%s>",
			strings.Join(params, ","),
			returnType)
		
		// Include effect row if present
		if typ.EffectRow != nil {
			effectStr := NormalizeTypeName(typ.EffectRow)
			result = fmt.Sprintf("Func<%s->%s!%s>",
				strings.Join(params, ","),
				returnType,
				effectStr)
		}
		return result
	
	case *Row:
		// Row types for effects or records
		if typ.Kind == EffectRow {
			// Effect row: {IO,Net,...}
			var effects []string
			for label := range typ.Labels {
				effects = append(effects, label)
			}
			sort.Strings(effects)
			result := fmt.Sprintf("{%s}", strings.Join(effects, ","))
			if typ.Tail != nil {
				result += fmt.Sprintf("|%s", NormalizeTypeName(typ.Tail))
			}
			return result
		} else if typ.Kind == RecordRow {
			// Record row: handled similar to TRecord
			var fields []string
			var fieldNames []string
			for name := range typ.Labels {
				fieldNames = append(fieldNames, name)
			}
			sort.Strings(fieldNames)
			
			for _, name := range fieldNames {
				if fieldType, ok := typ.Labels[name].(Type); ok {
					fields = append(fields, fmt.Sprintf("%s:%s", name, NormalizeTypeName(fieldType)))
				}
			}
			return fmt.Sprintf("Record<%s>", strings.Join(fields, ","))
		}
		return "UnknownRow"
	
	case *RowVar:
		return fmt.Sprintf("_%s", typ.Name)
		
	case *TRecord2:
		// Use same format as TRecord
		if typ.Row != nil {
			return NormalizeTypeName(typ.Row)
		}
		return "Record<>"
		
	default:
		// Fallback for unknown types
		if t != nil {
			return t.String()
		}
		return "Unknown"
	}
}

// helper to add effects to function type
func addEffectsToFunc(funcStr string, effects []EffectType) string {
	var effectStrs []string
	for _, e := range effects {
		effectStrs = append(effectStrs, e.String())
	}
	sort.Strings(effectStrs) // Deterministic effect order
	
	// Insert effects before the closing >
	idx := strings.LastIndex(funcStr, ">")
	if idx >= 0 {
		return funcStr[:idx] + fmt.Sprintf("!{%s}", strings.Join(effectStrs, ",")) + funcStr[idx:]
	}
	return funcStr
}

// MakeDictionaryKey creates a deterministic registry key for a dictionary
// Format: <namespace>::<ClassName>::<TypeNF>::<method>
// Example: "prelude::Num::Int::add"
func MakeDictionaryKey(namespace, className string, typ Type, method string) string {
	typeNF := NormalizeTypeName(typ)
	if method == "" {
		// Dictionary reference key (no method)
		return fmt.Sprintf("%s::%s::%s", namespace, className, typeNF)
	}
	// Full method key
	return fmt.Sprintf("%s::%s::%s::%s", namespace, className, typeNF, method)
}

// CanonKey is an alias for MakeDictionaryKey - the single entry point for
// all dictionary key generation to ensure consistency across linker and evaluator.
func CanonKey(namespace, className string, typ Type, method string) string {
	return MakeDictionaryKey(namespace, className, typ, method)
}

// ParseDictionaryKey extracts components from a dictionary key
// Now uses :: separator for better visual clarity
func ParseDictionaryKey(key string) (namespace, className, typeNF, method string, err error) {
	parts := strings.Split(key, "::")
	if len(parts) < 3 || len(parts) > 4 {
		return "", "", "", "", fmt.Errorf("invalid dictionary key format: %s (expected namespace::class::type[::method])", key)
	}
	
	namespace = parts[0]
	className = parts[1]
	typeNF = parts[2]
	
	if len(parts) == 4 {
		method = parts[3]
	}
	
	return namespace, className, typeNF, method, nil
}

// IsGroundType checks if a type contains no type variables
// This is used to verify that types are fully resolved before elaboration
func IsGroundType(t Type) bool {
	switch typ := t.(type) {
	case *TCon:
		return true
	case *TVar, *TVar2, *RowVar:
		return false
	case *TList:
		return IsGroundType(typ.Element)
	case *TTuple:
		for _, elem := range typ.Elements {
			if !IsGroundType(elem) {
				return false
			}
		}
		return true
	case *TRecord:
		for _, fieldType := range typ.Fields {
			if !IsGroundType(fieldType) {
				return false
			}
		}
		if typ.Row != nil {
			return IsGroundType(typ.Row)
		}
		return true
	case *TFunc:
		for _, param := range typ.Params {
			if !IsGroundType(param) {
				return false
			}
		}
		return IsGroundType(typ.Return)
	case *TFunc2:
		for _, param := range typ.Params {
			if !IsGroundType(param) {
				return false
			}
		}
		if !IsGroundType(typ.Return) {
			return false
		}
		if typ.EffectRow != nil {
			return IsGroundType(typ.EffectRow)
		}
		return true
	case *TApp:
		if !IsGroundType(typ.Constructor) {
			return false
		}
		for _, arg := range typ.Args {
			if !IsGroundType(arg) {
				return false
			}
		}
		return true
	case *Row:
		// Check row labels if they contain types (for record rows)
		if typ.Kind == RecordRow {
			for _, val := range typ.Labels {
				if t, ok := val.(Type); ok {
					if !IsGroundType(t) {
						return false
					}
				}
			}
		}
		if typ.Tail != nil {
			return IsGroundType(typ.Tail)
		}
		return true
	case *TRecord2:
		if typ.Row != nil {
			return IsGroundType(typ.Row)
		}
		return true
	default:
		// Conservative: unknown types are considered non-ground
		return false
	}
}
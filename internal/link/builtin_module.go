package link

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// RegisterBuiltinModule creates and registers the $builtin module interface
func RegisterBuiltinModule(ml *ModuleLinker) {
	// Create the builtin module interface
	builtinIface := &iface.Iface{
		Module:  "$builtin",
		Exports: make(map[string]*iface.IfaceItem),
		Schema:  "ailang.builtin/v1",
	}

	// Get sorted list of builtin names for deterministic ordering
	builtinNames := GetBuiltinInterface()

	// Register all builtin functions from the evaluator
	for _, name := range builtinNames {
		// Parse the builtin name to determine its type
		// Format: operation_Type (e.g., "add_Int", "eq_Bool")
		typeScheme := inferBuiltinType(name)

		builtinIface.Exports[name] = &iface.IfaceItem{
			Name:   name,
			Type:   typeScheme,
			Purity: true, // All builtins are pure
			Ref: core.GlobalRef{
				Module: "$builtin",
				Name:   name,
			},
		}
	}

	// Compute deterministic digest
	digest := computeBuiltinDigest(builtinIface)
	builtinIface.Digest = digest

	// Register the interface
	ml.RegisterIface(builtinIface)
}

// inferBuiltinType infers the type scheme for a builtin function
func inferBuiltinType(name string) *types.Scheme {
	// Parse builtin name (e.g., "add_Int" -> operation="add", type="Int")
	var op, typ string
	if n, err := fmt.Sscanf(name, "%[^_]_%s", &op, &typ); err != nil || n != 2 {
		// If we can't parse it, return a generic type
		return &types.Scheme{
			TypeVars: []string{},
			Type:     &types.TVar2{Name: "?", Kind: types.Star},
		}
	}

	// Build the appropriate type based on the operation
	var resultType types.Type
	switch op {
	case "add", "sub", "mul", "div", "mod":
		// Binary arithmetic: Type -> Type -> Type
		baseType := getBaseType(typ)
		resultType = &types.TFunc2{
			Params:    []types.Type{baseType, baseType},
			EffectRow: nil, // Pure builtin
			Return:    baseType,
		}

	case "neg":
		// Unary negation: Type -> Type
		baseType := getBaseType(typ)
		resultType = &types.TFunc2{
			Params:    []types.Type{baseType},
			EffectRow: nil, // Pure builtin
			Return:    baseType,
		}

	case "eq", "ne", "lt", "le", "gt", "ge":
		// Comparison: Type -> Type -> Bool
		baseType := getBaseType(typ)
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc2{
			Params:    []types.Type{baseType, baseType},
			EffectRow: nil, // Pure builtin
			Return:    boolType,
		}

	case "concat":
		// String concatenation: String -> String -> String
		strType := &types.TCon{Name: "String"}
		resultType = &types.TFunc2{
			Params:    []types.Type{strType, strType},
			EffectRow: nil, // Pure builtin
			Return:    strType,
		}

	case "and", "or":
		// Boolean operations: Bool -> Bool -> Bool
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc2{
			Params:    []types.Type{boolType, boolType},
			EffectRow: nil, // Pure builtin
			Return:    boolType,
		}

	case "not":
		// Boolean negation: Bool -> Bool
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc2{
			Params:    []types.Type{boolType},
			EffectRow: nil, // Pure builtin
			Return:    boolType,
		}

	case "show":
		// Show: a -> String (monomorphic based on type)
		baseType := getBaseType(typ)
		strType := &types.TCon{Name: "String"}
		resultType = &types.TFunc2{
			Params:    []types.Type{baseType},
			EffectRow: nil, // Pure builtin
			Return:    strType,
		}

	default:
		// Unknown operation, return generic type
		return &types.Scheme{
			TypeVars: []string{},
			Type:     &types.TVar2{Name: "?", Kind: types.Star},
		}
	}

	return &types.Scheme{
		TypeVars: []string{}, // All builtins are monomorphic after lowering
		Type:     resultType,
	}
}

// getBaseType converts a type name string to a Type
func getBaseType(typName string) types.Type {
	switch typName {
	case "Int":
		return &types.TCon{Name: "Int"}
	case "Float":
		return &types.TCon{Name: "Float"}
	case "String":
		return &types.TCon{Name: "String"}
	case "Bool":
		return &types.TCon{Name: "Bool"}
	default:
		return &types.TVar{Name: "?"}
	}
}

// GetBuiltinInterface returns a sorted list of all builtin functions for deterministic output
func GetBuiltinInterface() []string {
	var builtins []string
	for name := range eval.Builtins {
		builtins = append(builtins, name)
	}
	sort.Strings(builtins)
	return builtins
}

// computeBuiltinDigest computes a deterministic digest for the builtin module
func computeBuiltinDigest(iface *iface.Iface) string {
	// For the builtin module, use a simple versioned digest
	// This ensures reproducibility across builds
	return "builtin-v1-stable"
}

// RegisterAdtModule creates and registers the $adt module interface for ADT constructors
func RegisterAdtModule(ml *ModuleLinker) {
	// Create the $adt module interface
	// This module is populated dynamically at link time based on loaded type declarations
	// and their constructors from interfaces
	adtIface := &iface.Iface{
		Module:       "$adt",
		Exports:      make(map[string]*iface.IfaceItem),
		Constructors: make(map[string]*iface.ConstructorScheme),
		Schema:       "ailang.adt/v1",
	}

	// The $adt module exports factory functions for each constructor
	// These are synthesized at link time from loaded interfaces
	// Format: make_<TypeName>_<CtorName>
	// Example: make_Option_Some, make_Option_None

	// Get all loaded modules and collect their constructors
	loadedModules := ml.GetLoadedModules()
	var allConstructors []*iface.ConstructorScheme

	for _, modIface := range loadedModules {
		for _, ctor := range modIface.Constructors {
			allConstructors = append(allConstructors, ctor)
		}
	}

	// Sort constructors for deterministic ordering
	sort.Slice(allConstructors, func(i, j int) bool {
		if allConstructors[i].TypeName != allConstructors[j].TypeName {
			return allConstructors[i].TypeName < allConstructors[j].TypeName
		}
		return allConstructors[i].CtorName < allConstructors[j].CtorName
	})

	// Register factory function for each constructor
	for _, ctor := range allConstructors {
		factoryName := fmt.Sprintf("make_%s_%s", ctor.TypeName, ctor.CtorName)

		// Build function type: Field1 -> Field2 -> ... -> TypeName
		var typeScheme *types.Scheme
		if ctor.Arity == 0 {
			// Nullary constructor: just returns the type
			typeScheme = &types.Scheme{
				TypeVars: []string{},
				Type:     ctor.ResultType,
			}
		} else {
			// Constructor with fields: (Field1, Field2, ...) -> TypeName
			typeScheme = &types.Scheme{
				TypeVars: []string{},
				Type: &types.TFunc2{
					Params:    ctor.FieldTypes,
					EffectRow: nil, // Constructor application is pure
					Return:    ctor.ResultType,
				},
			}
		}

		adtIface.Exports[factoryName] = &iface.IfaceItem{
			Name:   factoryName,
			Type:   typeScheme,
			Purity: true, // Constructor application is pure
			Ref: core.GlobalRef{
				Module: "$adt",
				Name:   factoryName,
			},
		}

		// Also register the constructor scheme for runtime resolution
		adtIface.Constructors[ctor.CtorName] = ctor
	}

	// Compute deterministic digest
	digest := computeAdtDigest(adtIface)
	adtIface.Digest = digest

	// Register the interface
	ml.RegisterIface(adtIface)
}

// computeAdtDigest computes a deterministic digest for the $adt module
func computeAdtDigest(iface *iface.Iface) string {
	// For the $adt module, digest depends on loaded constructors
	// This ensures reproducibility across builds
	// For now, use a simple versioned digest
	return "adt-v1-stable"
}

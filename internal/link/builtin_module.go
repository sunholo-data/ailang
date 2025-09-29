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
			Type:     &types.TVar{Name: "?"},
		}
	}

	// Build the appropriate type based on the operation
	var resultType types.Type
	switch op {
	case "add", "sub", "mul", "div", "mod":
		// Binary arithmetic: Type -> Type -> Type
		baseType := getBaseType(typ)
		resultType = &types.TFunc{
			Params: []types.Type{baseType, baseType},
			Return: baseType,
		}

	case "neg":
		// Unary negation: Type -> Type
		baseType := getBaseType(typ)
		resultType = &types.TFunc{
			Params: []types.Type{baseType},
			Return: baseType,
		}

	case "eq", "ne", "lt", "le", "gt", "ge":
		// Comparison: Type -> Type -> Bool
		baseType := getBaseType(typ)
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc{
			Params: []types.Type{baseType, baseType},
			Return: boolType,
		}

	case "concat":
		// String concatenation: String -> String -> String
		strType := &types.TCon{Name: "String"}
		resultType = &types.TFunc{
			Params: []types.Type{strType, strType},
			Return: strType,
		}

	case "and", "or":
		// Boolean operations: Bool -> Bool -> Bool
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc{
			Params: []types.Type{boolType, boolType},
			Return: boolType,
		}

	case "not":
		// Boolean negation: Bool -> Bool
		boolType := &types.TCon{Name: "Bool"}
		resultType = &types.TFunc{
			Params: []types.Type{boolType},
			Return: boolType,
		}

	case "show":
		// Show: a -> String (monomorphic based on type)
		baseType := getBaseType(typ)
		strType := &types.TCon{Name: "String"}
		resultType = &types.TFunc{
			Params: []types.Type{baseType},
			Return: strType,
		}

	default:
		// Unknown operation, return generic type
		return &types.Scheme{
			TypeVars: []string{},
			Type:     &types.TVar{Name: "?"},
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

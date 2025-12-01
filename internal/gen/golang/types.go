package golang

import (
	"fmt"

	"github.com/sunholo/ailang/internal/types"
)

// GoType represents a Go type string that will be emitted in generated code.
type GoType string

// Common Go types
const (
	GoInt64   GoType = "int64"
	GoFloat64 GoType = "float64"
	GoBool    GoType = "bool"
	GoString  GoType = "string"
	GoUnit    GoType = "struct{}"
)

// TypeMapper converts AILANG types to Go types.
type TypeMapper struct {
	// knownTypes maps AILANG type names to their Go representations
	knownTypes map[string]GoType
}

// NewTypeMapper creates a new TypeMapper with default type mappings.
func NewTypeMapper() *TypeMapper {
	return &TypeMapper{
		knownTypes: make(map[string]GoType),
	}
}

// RegisterType registers a custom type mapping.
// Used when an ADT is defined - we register its Go type name.
func (tm *TypeMapper) RegisterType(ailangName string, goType GoType) {
	tm.knownTypes[ailangName] = goType
}

// MapType converts an AILANG type to its Go representation.
// Returns the Go type string and any error.
func (tm *TypeMapper) MapType(t types.Type) (GoType, error) {
	if t == nil {
		return "", fmt.Errorf("nil type")
	}

	switch typ := t.(type) {
	case *types.TCon:
		return tm.mapTCon(typ)

	case *types.TList:
		elemType, err := tm.MapType(typ.Element)
		if err != nil {
			return "", err
		}
		return GoType(fmt.Sprintf("[]%s", elemType)), nil

	case *types.TArray:
		elemType, err := tm.MapType(typ.Element)
		if err != nil {
			return "", err
		}
		return GoType(fmt.Sprintf("[]%s", elemType)), nil

	case *types.TTuple:
		// Go doesn't have tuples, generate a struct type name
		// For now, return a placeholder
		return GoType("struct{}"), nil

	case *types.TRecord:
		// Records map to structs - will be expanded in A2
		return GoType("struct{}"), nil

	case *types.TFunc:
		return tm.mapTFunc(typ)

	case *types.TApp:
		// Type application - look up constructor
		if con, ok := typ.Constructor.(*types.TCon); ok {
			if goType, ok := tm.knownTypes[con.Name]; ok {
				return goType, nil
			}
			return GoType(ToGoTypeName(con.Name)), nil
		}
		return "", fmt.Errorf("complex type application not yet supported: %v", t)

	case *types.TVar:
		// Type variables should be resolved before codegen
		return "", fmt.Errorf("unresolved type variable in codegen: %s", typ.Name)

	default:
		return "", fmt.Errorf("unsupported type for Go codegen: %T", t)
	}
}

// mapTCon maps a type constructor to Go.
func (tm *TypeMapper) mapTCon(typ *types.TCon) (GoType, error) {
	switch typ.Name {
	case "int":
		return GoInt64, nil
	case "float":
		return GoFloat64, nil
	case "bool":
		return GoBool, nil
	case "string":
		return GoString, nil
	case "()", "unit":
		return GoUnit, nil
	default:
		// User-defined type
		if goType, ok := tm.knownTypes[typ.Name]; ok {
			return goType, nil
		}
		return GoType(ToGoTypeName(typ.Name)), nil
	}
}

// mapTFunc maps a function type to Go.
func (tm *TypeMapper) mapTFunc(typ *types.TFunc) (GoType, error) {
	var paramTypes []string
	for _, p := range typ.Params {
		pt, err := tm.MapType(p)
		if err != nil {
			return "", err
		}
		paramTypes = append(paramTypes, string(pt))
	}

	returnType, err := tm.MapType(typ.Return)
	if err != nil {
		return "", err
	}

	if len(paramTypes) == 0 {
		return GoType(fmt.Sprintf("func() %s", returnType)), nil
	}
	return GoType(fmt.Sprintf("func(%s) %s", join(paramTypes, ", "), returnType)), nil
}

// join is a helper to join strings
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// MapPrimitiveType maps a primitive AILANG type name to Go.
// Used when we only have the type name as a string.
func MapPrimitiveType(name string) (GoType, bool) {
	switch name {
	case "int":
		return GoInt64, true
	case "float":
		return GoFloat64, true
	case "bool":
		return GoBool, true
	case "string":
		return GoString, true
	case "()", "unit":
		return GoUnit, true
	default:
		return "", false
	}
}

// IsPointerType returns true if the Go type should be a pointer.
// Sum types and recursive types use pointers.
func IsPointerType(t GoType) bool {
	switch t {
	case GoInt64, GoFloat64, GoBool, GoString, GoUnit:
		return false
	default:
		// Check if it's a slice (already a reference type)
		if len(t) > 2 && t[:2] == "[]" {
			return false
		}
		// Check if it's already a pointer
		if len(t) > 0 && t[0] == '*' {
			return false
		}
		// Check if it's a func type
		if len(t) > 4 && t[:4] == "func" {
			return false
		}
		// User-defined types get pointers
		return true
	}
}

// WithPointer wraps a type in a pointer if needed.
func WithPointer(t GoType) GoType {
	if IsPointerType(t) {
		return GoType("*" + string(t))
	}
	return t
}

// ZeroValue returns the zero value literal for a Go type.
func ZeroValue(t GoType) string {
	switch t {
	case GoInt64:
		return "0"
	case GoFloat64:
		return "0.0"
	case GoBool:
		return "false"
	case GoString:
		return `""`
	case GoUnit:
		return "struct{}{}"
	default:
		// Slices and pointers have nil as zero value
		if len(t) > 0 && (t[0] == '[' || t[0] == '*') {
			return "nil"
		}
		// Struct types use empty literal
		return string(t) + "{}"
	}
}

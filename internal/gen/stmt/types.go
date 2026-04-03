package stmt

import "fmt"

// ResolvedType is a fully-resolved type with no type variables.
// Type projection is a pure function: same input always produces same output.
// If a type variable is encountered, projection MUST error (no silent defaults).
type ResolvedType interface {
	resolvedType()
	GoString() string // Go type representation (for the Go emitter)
}

// PrimitiveType represents a basic type.
type PrimitiveType struct {
	Kind PrimitiveKind
}

func (PrimitiveType) resolvedType() {}
func (p PrimitiveType) GoString() string {
	switch p.Kind {
	case PrimInt:
		return "int64"
	case PrimFloat:
		return "float64"
	case PrimBool:
		return "bool"
	case PrimString:
		return "string"
	case PrimUnit:
		return "struct{}"
	default:
		return "interface{}"
	}
}

// PrimitiveKind enumerates primitive types.
type PrimitiveKind int

const (
	PrimInt PrimitiveKind = iota
	PrimFloat
	PrimBool
	PrimString
	PrimUnit
)

// NamedType references a user-defined type (ADT or record) by name.
type NamedType struct {
	Name    string // Type name (e.g., "Color", "Position")
	Pointer bool   // Whether to use pointer semantics (true for ADTs)
}

func (NamedType) resolvedType() {}
func (n NamedType) GoString() string {
	if n.Pointer {
		return "*" + n.Name
	}
	return n.Name
}

// SliceType represents a list/array as a Go slice.
type SliceType struct {
	Elem ResolvedType
}

func (SliceType) resolvedType() {}
func (s SliceType) GoString() string {
	return "[]" + s.Elem.GoString()
}

// FuncType represents a function type.
type FuncType struct {
	Params []ResolvedType
	Return ResolvedType
}

func (FuncType) resolvedType() {}
func (f FuncType) GoString() string {
	result := "func("
	for i, p := range f.Params {
		if i > 0 {
			result += ", "
		}
		result += p.GoString()
	}
	result += ") " + f.Return.GoString()
	return result
}

// TupleType represents a tuple (compiled as a struct).
type TupleType struct {
	Elems []ResolvedType
}

func (TupleType) resolvedType() {}
func (t TupleType) GoString() string {
	return fmt.Sprintf("Tuple%d", len(t.Elems))
}

// InterfaceType represents an unresolved or dynamic type (interface{}).
// This should be used sparingly — only at boundaries where type info is lost.
type InterfaceType struct{}

func (InterfaceType) resolvedType() {}
func (InterfaceType) GoString() string {
	return "interface{}"
}

// MapType represents a map[string]interface{} (AILANG's Map type).
type MapType struct{}

func (MapType) resolvedType() {}
func (MapType) GoString() string {
	return "map[string]interface{}"
}

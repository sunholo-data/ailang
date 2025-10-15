package iface

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Iface represents a module's interface (its typed exports)
type Iface struct {
	Module       string                        // Module path, e.g., "math/gcd"
	Exports      map[string]*IfaceItem         // Exported symbols
	Constructors map[string]*ConstructorScheme // Exported ADT constructors
	Types        map[string]*TypeExport        // Exported type names
	Schema       string                        // Schema version, e.g., "ailang.iface/v1"
	Digest       string                        // Deterministic digest of interface
}

// TypeExport represents an exported type name
type TypeExport struct {
	Name  string // Type name (e.g., "Option", "Result")
	Arity int    // Number of type parameters
}

// IfaceItem represents a single exported symbol
type IfaceItem struct {
	Name   string         // Symbol name
	Type   *types.Scheme  // Generalized type scheme
	Purity bool           // Whether the function is pure
	Ref    core.GlobalRef // Global reference to this item
}

// ConstructorScheme represents the type scheme of an ADT constructor
type ConstructorScheme struct {
	TypeName   string       // The ADT name (e.g., "Option")
	CtorName   string       // Constructor name (e.g., "Some", "None")
	FieldTypes []types.Type // Field types (empty for nullary constructors)
	ResultType types.Type   // Result type after application
	Arity      int          // Number of fields
}

// NewIface creates a new module interface
func NewIface(module string) *Iface {
	return &Iface{
		Module:       module,
		Exports:      make(map[string]*IfaceItem),
		Constructors: make(map[string]*ConstructorScheme),
		Types:        make(map[string]*TypeExport),
		Schema:       "ailang.iface/v1",
	}
}

// AddExport adds an exported symbol to the interface
func (i *Iface) AddExport(name string, typ *types.Scheme, purity bool) {
	i.Exports[name] = &IfaceItem{
		Name:   name,
		Type:   typ,
		Purity: purity,
		Ref: core.GlobalRef{
			Module: i.Module,
			Name:   name,
		},
	}
}

// GetExport retrieves an exported symbol
func (i *Iface) GetExport(name string) (*IfaceItem, bool) {
	item, ok := i.Exports[name]
	return item, ok
}

// AddConstructor adds an ADT constructor to the interface
func (i *Iface) AddConstructor(typeName, ctorName string, fieldTypes []types.Type, resultType types.Type) {
	i.Constructors[ctorName] = &ConstructorScheme{
		TypeName:   typeName,
		CtorName:   ctorName,
		FieldTypes: fieldTypes,
		ResultType: resultType,
		Arity:      len(fieldTypes),
	}
}

// GetConstructor retrieves a constructor scheme
func (i *Iface) GetConstructor(name string) (*ConstructorScheme, bool) {
	ctor, ok := i.Constructors[name]
	return ctor, ok
}

// AddType adds an exported type name to the interface
func (i *Iface) AddType(name string, arity int) {
	i.Types[name] = &TypeExport{
		Name:  name,
		Arity: arity,
	}
}

// GetType retrieves an exported type
func (i *Iface) GetType(name string) (*TypeExport, bool) {
	typ, ok := i.Types[name]
	return typ, ok
}

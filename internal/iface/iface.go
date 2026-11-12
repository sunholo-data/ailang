package iface

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

const SchemaVersion = "ailang.iface/v2"

// Iface represents a module's interface (its typed exports)
type Iface struct {
	Module       string                        // Module path, e.g., "math/gcd"
	Exports      map[string]*IfaceItem         // Exported symbols
	Constructors map[string]*ConstructorScheme // Exported ADT constructors
	Types        map[string]*TypeExport        // Exported type names
	TypeAliases  map[string]types.Type         // M-FIX-RECORD-UPDATE: Type alias expansions for record types
	AliasParams  map[string][]string           // M-XMOD-ALIAS-POLY: param names for parameterized aliases (name -> ["a"]); missing = nullary
	DerivedEq    []string                      // Types this module declares `deriving (Eq)`, sorted; importers need their Eq instances
	Schema       string                        // Schema version, e.g., SchemaVersion
	Digest       string                        // Deterministic digest of interface
	// Tier is the module's default tier from std/STABILITY.toml ("" outside the
	// manifest). Every exported type carries it; a function carries its own
	// IfaceItem.Tier (M-V1-STDLIB-SURFACE-POLISH §6.1). Not hashed.
	Tier string
}

// TypeExport represents an exported type name
type TypeExport struct {
	Name  string // Type name (e.g., "Option", "Result")
	Arity int    // Number of type parameters
}

// IfaceItem represents a single exported symbol
type IfaceItem struct {
	IFC    *types.ImportedIFCSig // Explicit IFC metadata
	Name   string                // Symbol name
	Type   *types.Scheme         // Generalized type scheme
	Purity bool                  // Whether the function is pure
	Ref    core.GlobalRef        // Global reference to this item
	// Tier: the module default, or "experimental" under @experimental.
	Tier string
	// ReplacedBy / DeprecationNote come from @deprecated("replacement", "note").
	// Informational: neither changes the type, the digest or interface_hash.
	ReplacedBy      string
	DeprecationNote string
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
		TypeAliases:  make(map[string]types.Type),
		AliasParams:  make(map[string][]string),
		Schema:       SchemaVersion,
	}
}

// AddExport adds an exported symbol to the interface
func (i *Iface) AddExport(name string, typ *types.Scheme, purity bool) {
	i.Exports[name] = &IfaceItem{
		IFC:    types.IFCSummaryFromScheme(typ),
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

// AddTypeAlias adds a type alias expansion to the interface (M-FIX-RECORD-UPDATE)
// This is used for record type aliases like `type NPC = { pos: Pos, name: string }`
func (i *Iface) AddTypeAlias(name string, target types.Type) {
	i.TypeAliases[name] = target
}

// TypeAliasFor returns the alias a type constructor names, read from this
// interface (M-V1-NOMINAL-TYPE-IDENTITY): the interface's own aliases are
// stored under their bare name, aliases it embeds from another module under
// types.NominalKey(module, name). A TCon of another module is never expanded
// through a same-named alias of this one.
func (i *Iface) TypeAliasFor(con *types.TCon) (types.Type, bool) {
	if con == nil {
		return nil, false
	}
	if con.Module == "" || con.Module == i.Module {
		t, ok := i.TypeAliases[con.Name]
		return t, ok
	}
	t, ok := i.TypeAliases[con.KeyOf()]
	return t, ok
}

// GetTypeAlias retrieves a type alias expansion (M-FIX-RECORD-UPDATE)
func (i *Iface) GetTypeAlias(name string) (types.Type, bool) {
	typ, ok := i.TypeAliases[name]
	return typ, ok
}

// AddTypeAliasParams records the param names of a parameterized alias
// (M-XMOD-ALIAS-POLY). Empty/nil params are not stored (missing = nullary).
func (i *Iface) AddTypeAliasParams(name string, params []string) {
	if len(params) == 0 {
		return
	}
	if i.AliasParams == nil {
		i.AliasParams = make(map[string][]string)
	}
	i.AliasParams[name] = params
}

// GetTypeAliasParams retrieves the param names of a parameterized alias
// (M-XMOD-ALIAS-POLY). ok is false for nullary aliases.
func (i *Iface) GetTypeAliasParams(name string) ([]string, bool) {
	params, ok := i.AliasParams[name]
	return params, ok
}

// SetDerivedEq records the types this module declares `deriving (Eq)` and
// recomputes the digest. The names are part of the interface: an importer that
// uses == on one of them type-checks only while the declaration derives Eq, so
// adding or dropping `deriving (Eq)` must change what importers' cache keys see.
func (i *Iface) SetDerivedEq(names []string) error {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if len(sorted) == 0 {
		sorted = nil
	}
	i.DerivedEq = sorted
	digest, err := NewBuilder(i.Module, nil).computeDigest(i)
	if err != nil {
		return err
	}
	i.Digest = digest
	return nil
}

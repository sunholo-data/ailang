package iface

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Iface represents a module's interface (its typed exports)
type Iface struct {
	Module  string                // Module path, e.g., "math/gcd"
	Exports map[string]*IfaceItem // Exported symbols
	Schema  string                // Schema version, e.g., "ailang.iface/v1"
	Digest  string                // Deterministic digest of interface
}

// IfaceItem represents a single exported symbol
type IfaceItem struct {
	Name   string         // Symbol name
	Type   *types.Scheme  // Generalized type scheme
	Purity bool           // Whether the function is pure
	Ref    core.GlobalRef // Global reference to this item
}

// NewIface creates a new module interface
func NewIface(module string) *Iface {
	return &Iface{
		Module:  module,
		Exports: make(map[string]*IfaceItem),
		Schema:  "ailang.iface/v1",
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

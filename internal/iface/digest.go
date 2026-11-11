package iface

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
	"sort"
)

// nominalRefs lists, sorted, the defining module and name of every nominal
// type constructor in ts. The digest renders types by bare name; this is the
// qualified half (M-V1-NOMINAL-TYPE-IDENTITY §4.6), so an upstream that
// switches which `T` an export returns (a.T -> b.T) changes the digest and
// invalidates its importers' cache entries. It never reaches the registry
// interface_hash (hash_projection.go), which keeps rendering bare names (D3).
func nominalRefs(ts ...types.Type) []string {
	set := make(map[string]bool)
	for _, t := range ts {
		if t == nil {
			continue
		}
		traverse.Walk(t, func(n types.Type) {
			if c, ok := n.(*types.TCon); ok && c.Module != "" {
				set[c.KeyOf()] = true
			}
		})
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ifaceItem is used for JSON serialization
type ifaceItem struct {
	IFC     *types.ImportedIFCSig `json:"ifc"`
	Name    string                `json:"name"`
	Type    string                `json:"type"` // String representation of the scheme
	Pure    bool                  `json:"pure"`
	Effects []string              `json:"effects,omitempty"`
}

// ctorItem is used for JSON serialization of constructors
type ctorItem struct {
	TypeName   string   `json:"type_name"`
	CtorName   string   `json:"ctor_name"`
	FieldTypes []string `json:"field_types"`
	ResultType string   `json:"result_type"`
	Arity      int      `json:"arity"`
}

// computeDigest computes a deterministic digest of the interface
func (b *Builder) computeDigest(iface *Iface) (string, error) {
	// Create a deterministic JSON representation
	type jsonIface struct {
		Module       string               `json:"module"`
		Schema       string               `json:"schema"`
		Exports      map[string]ifaceItem `json:"exports"`
		Constructors map[string]ctorItem  `json:"constructors,omitempty"`
		DerivedEq    []string             `json:"derived_eq,omitempty"`
		// Nominal: export/constructor name -> qualified nominal refs.
		Nominal map[string][]string `json:"nominal,omitempty"`
	}

	// Convert to JSON-friendly format with sorted keys
	ji := jsonIface{
		Module:       iface.Module,
		Schema:       iface.Schema,
		Exports:      make(map[string]ifaceItem),
		Constructors: make(map[string]ctorItem),
		DerivedEq:    iface.DerivedEq,
		Nominal:      make(map[string][]string),
	}

	// Sort export names for deterministic ordering
	var names []string
	for name := range iface.Exports {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		item := iface.Exports[name]
		ji.Exports[name] = ifaceItem{
			IFC:     item.IFC,
			Name:    item.Name,
			Type:    b.schemeToString(item.Type),
			Pure:    item.Purity,
			Effects: []string{}, // Placeholder for future effect system
		}
		if item.Type != nil {
			if refs := nominalRefs(item.Type.Type); len(refs) > 0 {
				ji.Nominal["export:"+name] = refs
			}
		}
	}

	// Sort constructor names for deterministic ordering
	var ctorNames []string
	for name := range iface.Constructors {
		ctorNames = append(ctorNames, name)
	}
	sort.Strings(ctorNames)

	for _, name := range ctorNames {
		ctor := iface.Constructors[name]
		fieldTypeStrs := make([]string, len(ctor.FieldTypes))
		for i, ft := range ctor.FieldTypes {
			fieldTypeStrs[i] = ft.String()
		}
		ji.Constructors[name] = ctorItem{
			TypeName:   ctor.TypeName,
			CtorName:   ctor.CtorName,
			FieldTypes: fieldTypeStrs,
			ResultType: ctor.ResultType.String(),
			Arity:      ctor.Arity,
		}
		if refs := nominalRefs(append(append([]types.Type{}, ctor.FieldTypes...), ctor.ResultType)...); len(refs) > 0 {
			ji.Nominal["ctor:"+name] = refs
		}
	}

	// Marshal to canonical JSON
	data, err := json.Marshal(ji)
	if err != nil {
		return "", err
	}

	// Compute SHA256 (using standard library for now, can switch to Blake3 later)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

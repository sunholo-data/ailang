package iface

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/types"
)

// Interface digests: the content hash importers' cache keys are built from.
// Split out of builder.go (file-size gate) — no behaviour change.

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

// computeDigest computes a deterministic digest of the interface
func (b *Builder) computeDigest(iface *Iface) (string, error) {
	// Create a deterministic JSON representation
	type jsonIface struct {
		Module       string               `json:"module"`
		Schema       string               `json:"schema"`
		Exports      map[string]ifaceItem `json:"exports"`
		Constructors map[string]ctorItem  `json:"constructors,omitempty"`
		DerivedEq    []string             `json:"derived_eq,omitempty"`
	}

	// Convert to JSON-friendly format with sorted keys
	ji := jsonIface{
		Module:       iface.Module,
		Schema:       iface.Schema,
		Exports:      make(map[string]ifaceItem),
		Constructors: make(map[string]ctorItem),
		DerivedEq:    iface.DerivedEq,
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
			Name:    item.Name,
			Type:    b.schemeToString(item.Type),
			Pure:    item.Purity,
			Effects: []string{}, // Placeholder for future effect system
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

// schemeToString converts a type scheme to a deterministic string representation
func (b *Builder) schemeToString(scheme *types.Scheme) string {
	if scheme == nil {
		return "?"
	}

	// Format: ∀a b. ∀r s. type (type vars, then row vars)
	var quantifiers []string
	if len(scheme.TypeVars) > 0 {
		quantifiers = append(quantifiers, scheme.TypeVars...)
	}
	if len(scheme.RowVars) > 0 {
		// Add row vars with a different prefix for clarity
		quantifiers = append(quantifiers, scheme.RowVars...)
	}

	if len(quantifiers) > 0 {
		return fmt.Sprintf("∀%s. %s",
			strings.Join(quantifiers, " "),
			scheme.Type.String())
	}
	return scheme.Type.String()
}

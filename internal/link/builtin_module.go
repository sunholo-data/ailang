package link

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// RegisterBuiltinModule creates and registers the $builtin module interface
func RegisterBuiltinModule(ml *ModuleLinker) {
	// Use new spec-based registry (M-DX1 migration complete in v0.3.10)
	registerFromSpecRegistry(ml)
}

// registerFromSpecRegistry registers builtins from the new spec-based registry
func registerFromSpecRegistry(ml *ModuleLinker) {
	builtinIface := &iface.Iface{
		Module:  "$builtin",
		Exports: make(map[string]*iface.IfaceItem),
		Schema:  "ailang.builtin/v2", // New schema version
	}

	specs := builtins.AllSpecs()

	// Get sorted names for deterministic ordering
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Register each builtin from spec
	for _, name := range names {
		spec := specs[name]

		// Build type scheme from spec
		typ := spec.Type()
		typeScheme := &types.Scheme{
			TypeVars: []string{}, // TODO: Extract type vars if polymorphic
			Type:     typ,
		}

		builtinIface.Exports[name] = &iface.IfaceItem{
			Name:   name,
			Type:   typeScheme,
			Purity: spec.IsPure,
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

// computeBuiltinDigest computes a deterministic digest for the $builtin module
func computeBuiltinDigest(iface *iface.Iface) string {
	// For the $builtin module, digest depends on registered builtins
	// This ensures reproducibility across builds
	// For now, use a simple versioned digest
	return "builtin-v2-stable"
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

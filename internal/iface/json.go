package iface

import (
	"encoding/json"
	"sort"

	"github.com/sunholo/ailang/internal/types"
)

// InterfaceJSON represents the normalized JSON format for module interfaces
type InterfaceJSON struct {
	Module string     `json:"module"`
	Types  []TypeJSON `json:"types"`
	Funcs  []FuncJSON `json:"funcs"`
	Schema string     `json:"schema"`
}

// TypeJSON represents an exported type in normalized form
type TypeJSON struct {
	Name   string   `json:"name"`
	Params []string `json:"params,omitempty"`
	Ctors  []string `json:"ctors,omitempty"`
}

// FuncJSON represents an exported function in normalized form
type FuncJSON struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Effects []string `json:"effects"`
	Pure    bool     `json:"pure"`
}

// ToNormalizedJSON converts an Iface to normalized JSON
// Normalization rules:
// - Sort all arrays alphabetically
// - Canonicalize type variables to a, b, c, ...
// - Sort effect rows alphabetically
// - Deterministic field ordering (via struct tags)
func (i *Iface) ToNormalizedJSON() ([]byte, error) {
	result := InterfaceJSON{
		Module: i.Module,
		Schema: i.Schema,
		Types:  make([]TypeJSON, 0),
		Funcs:  make([]FuncJSON, 0),
	}

	// Build types map (type name -> constructors)
	typeToCtors := make(map[string][]string)
	for ctorName, ctor := range i.Constructors {
		typeToCtors[ctor.TypeName] = append(typeToCtors[ctor.TypeName], ctorName)
	}

	// Sort constructors within each type
	for typeName := range typeToCtors {
		sort.Strings(typeToCtors[typeName])
	}

	// Add types (sorted by name)
	typeNames := make([]string, 0, len(i.Types))
	for name := range i.Types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		typeExport := i.Types[name]

		// Generate canonical type parameters (a, b, c, ...)
		params := make([]string, typeExport.Arity)
		for j := 0; j < typeExport.Arity; j++ {
			params[j] = string(rune('a' + j))
		}

		typeJSON := TypeJSON{
			Name:   name,
			Params: params,
			Ctors:  typeToCtors[name], // Already sorted
		}

		result.Types = append(result.Types, typeJSON)
	}

	// Add functions (sorted by name)
	funcNames := make([]string, 0, len(i.Exports))
	for name := range i.Exports {
		funcNames = append(funcNames, name)
	}
	sort.Strings(funcNames)

	for _, name := range funcNames {
		export := i.Exports[name]

		// Format type with canonicalized variables
		typeStr := canonicalizeType(export.Type)

		// Extract and sort effects
		effects := extractEffects(export.Type)

		funcJSON := FuncJSON{
			Name:    name,
			Type:    typeStr,
			Effects: effects,
			Pure:    export.Purity,
		}

		result.Funcs = append(result.Funcs, funcJSON)
	}

	// Use deterministic JSON encoding
	return json.MarshalIndent(result, "", "  ")
}

// canonicalizeType converts a Scheme to canonical string form
// Type variables are renamed to a, b, c, ...
func canonicalizeType(scheme *types.Scheme) string {
	if scheme == nil || scheme.Type == nil {
		return "unknown"
	}

	// Create variable mapping
	varMap := make(map[string]string)
	varCounter := 0

	// Helper to get canonical name for type variable
	getCanonName := func(original string) string {
		if canon, ok := varMap[original]; ok {
			return canon
		}
		canon := string(rune('a' + varCounter))
		varMap[original] = canon
		varCounter++
		return canon
	}

	// Format the type with canonical variables
	return formatTypeCanonical(scheme.Type, getCanonName)
}

// formatTypeCanonical formats a type with canonical variable names
func formatTypeCanonical(t types.Type, getCanonName func(string) string) string {
	switch typ := t.(type) {
	case *types.TVar2:
		return getCanonName(typ.Name)
	case *types.TFunc2:
		// Format: (param1, param2, ...) -> result [! {effects}]
		params := make([]string, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = formatTypeCanonical(p, getCanonName)
		}

		result := "(" + joinTypes(params) + ")->" + formatTypeCanonical(typ.Return, getCanonName)

		// Add effects if present
		if typ.EffectRow != nil && len(typ.EffectRow.Labels) > 0 {
			effectNames := make([]string, 0, len(typ.EffectRow.Labels))
			for name := range typ.EffectRow.Labels {
				effectNames = append(effectNames, name)
			}
			sort.Strings(effectNames)
			result += "!{" + joinTypes(effectNames) + "}"
		}

		return result
	case *types.TList:
		return "[" + formatTypeCanonical(typ.Element, getCanonName) + "]"
	case *types.TCon:
		return typ.Name
	default:
		return t.String()
	}
}

// joinTypes joins type strings with commas
func joinTypes(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}

// extractEffects extracts and sorts effect names from a Scheme
func extractEffects(scheme *types.Scheme) []string {
	if scheme == nil || scheme.Type == nil {
		return []string{}
	}

	// Extract from TFunc2 effect row
	if funcType, ok := scheme.Type.(*types.TFunc2); ok {
		if funcType.EffectRow != nil && len(funcType.EffectRow.Labels) > 0 {
			effects := make([]string, 0, len(funcType.EffectRow.Labels))
			for name := range funcType.EffectRow.Labels {
				effects = append(effects, name)
			}
			sort.Strings(effects)
			return effects
		}
	}

	return []string{}
}

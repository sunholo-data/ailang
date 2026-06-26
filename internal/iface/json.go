package iface

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/types"
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

	// Build types map (type name -> rendered constructor signatures).
	// M-IFACE-COMPACT-ADT-FIELDS: each entry is the full "Name({fields})" / "Name(t1, t2)"
	// signature (not just the bare name) so the compact iface is usable for ADT construction.
	typeToCtors := make(map[string][]string)
	for ctorName, ctor := range i.Constructors {
		typeToCtors[ctor.TypeName] = append(typeToCtors[ctor.TypeName], renderConstructor(ctorName, ctor.FieldTypes))
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

// renderConstructor renders an ADT constructor as "Name" (nullary) or "Name(f1, f2, ...)"
// with canonical type-variable names, so the compact interface shows how to build each
// variant (e.g. HeadingBlock({text: string, level: int})). M-IFACE-COMPACT-ADT-FIELDS.
func renderConstructor(name string, fieldTypes []types.Type) string {
	if len(fieldTypes) == 0 {
		return name
	}
	varMap := make(map[string]string)
	varCounter := 0
	getCanonName := func(original string) string {
		if canon, ok := varMap[original]; ok {
			return canon
		}
		canon := string(rune('a' + varCounter))
		varMap[original] = canon
		varCounter++
		return canon
	}
	parts := make([]string, len(fieldTypes))
	for i, ft := range fieldTypes {
		parts[i] = formatTypeCanonical(ft, getCanonName)
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
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

// maxTypeRenderDepth bounds formatTypeCanonical recursion so a cyclic types.Type
// (e.g. a self-referential record produced during inference) can never hang the
// renderer. See .claude/rules/type-system.md "Safe Type Traversal".
const maxTypeRenderDepth = 32

// formatTypeCanonical formats a type with canonical variable names.
func formatTypeCanonical(t types.Type, getCanonName func(string) string) string {
	return formatTypeCanonicalDepth(t, getCanonName, 0)
}

// formatTypeCanonicalDepth is the cycle-safe worker for formatTypeCanonical.
func formatTypeCanonicalDepth(t types.Type, getCanonName func(string) string, depth int) string {
	if depth > maxTypeRenderDepth {
		return "_" // cycle / over-deep guard
	}
	switch typ := t.(type) {
	case *types.TVar2:
		return getCanonName(typ.Name)
	case *types.TFunc2:
		// Format: (param1, param2, ...) -> result [! {effects}]
		params := make([]string, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = formatTypeCanonicalDepth(p, getCanonName, depth+1)
		}

		result := "(" + joinTypes(params) + ")->" + formatTypeCanonicalDepth(typ.Return, getCanonName, depth+1)

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
		return "[" + formatTypeCanonicalDepth(typ.Element, getCanonName, depth+1) + "]"
	case *types.TCon:
		return typ.Name
	case *types.TApp:
		// Handle type applications like List[a], Option[b]
		// Format constructor and arguments shallowly to avoid cyclic type issues
		args := make([]string, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = formatTypeCanonicalDepth(arg, getCanonName, depth+1)
		}
		return formatTypeCanonicalDepth(typ.Constructor, getCanonName, depth+1) + "[" + joinTypes(args) + "]"
	case *types.TRecord:
		// M-IFACE-COMPACT-ADT-FIELDS: render records structurally ({label: type, ...})
		// so the compact interface is usable for ADT construction, instead of leaking
		// the Go internal "<*types.TRecord>". A nominal record (unified with a named
		// type) renders as that name so the consumer can look it up.
		if typ.TypeName != "" {
			return typ.TypeName
		}
		// Merge explicit Fields with any labels carried on the row. A closed record
		// has no row tail and renders without "..."; an open (row-polymorphic) record
		// renders "...<rowvar>".
		merged := make(map[string]types.Type, len(typ.Fields))
		for name, ft := range typ.Fields {
			merged[name] = ft
		}
		openTail := ""
		if row, ok := typ.Row.(*types.Row); ok {
			for name, ft := range row.Labels {
				merged[name] = ft
			}
			if row.Tail != nil {
				openTail = "..." + getCanonName(row.Tail.Name)
			}
		} else if typ.Row != nil {
			openTail = "..." + formatTypeCanonicalDepth(typ.Row, getCanonName, depth+1)
		}
		labels := make([]string, 0, len(merged))
		for name := range merged {
			labels = append(labels, name)
		}
		sort.Strings(labels)
		parts := make([]string, 0, len(labels)+1)
		for _, name := range labels {
			parts = append(parts, name+": "+formatTypeCanonicalDepth(merged[name], getCanonName, depth+1))
		}
		if openTail != "" {
			parts = append(parts, openTail)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		// Fallback: return type name without traversing (cycle-safe)
		// Don't call t.String() as it may hang on cyclic types
		return fmt.Sprintf("<%T>", t)
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

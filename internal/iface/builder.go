package iface

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Builder constructs module interfaces from typed Core programs
type Builder struct {
	module  string
	typeEnv *types.TypeEnv
}

// NewBuilder creates a new interface builder
func NewBuilder(module string, typeEnv *types.TypeEnv) *Builder {
	return &Builder{
		module:  module,
		typeEnv: typeEnv,
	}
}

// ConstructorInfo represents constructor information for interface building
type ConstructorInfo struct {
	TypeName string
	CtorName string
	Arity    int
}

// BuildInterface extracts the typed interface from a Core program
func BuildInterface(module string, prog *core.Program, typeEnv *types.TypeEnv) (*Iface, error) {
	builder := NewBuilder(module, typeEnv)
	return builder.Build(prog, nil, nil)
}

// BuildInterfaceWithConstructors builds an interface with constructor information
func BuildInterfaceWithConstructors(module string, prog *core.Program, typeEnv *types.TypeEnv, constructors map[string]*ConstructorInfo) (*Iface, error) {
	builder := NewBuilder(module, typeEnv)
	return builder.Build(prog, constructors, nil)
}

// BuildInterfaceWithTypesAndConstructors builds an interface with type declarations and constructor information
func BuildInterfaceWithTypesAndConstructors(module string, prog *core.Program, typeEnv *types.TypeEnv, astFile interface{}, constructors map[string]*ConstructorInfo) (*Iface, error) {
	builder := NewBuilder(module, typeEnv)
	return builder.Build(prog, constructors, astFile)
}

// Build constructs the interface from a Core program
func (b *Builder) Build(prog *core.Program, constructors map[string]*ConstructorInfo, astFile interface{}) (*Iface, error) {
	// DEBUG: fmt.Printf("DEBUG Build: module=%s, astFile=%v\n", b.module, astFile != nil)
	iface := NewIface(b.module)

	// Extract exportable bindings from the program
	exports, err := b.extractExports(prog)
	if err != nil {
		return nil, err
	}

	// Process each export
	// DEBUG: fmt.Printf("DEBUG: Processing %d exports for module %s\n", len(exports), b.module)
	for name, binding := range exports {
		// DEBUG: fmt.Printf("DEBUG:   Processing export %s\n", name)
		// Get the type from the environment
		typ, err := b.typeEnv.Lookup(name)
		if err != nil {
			// Skip if not in type environment (shouldn't happen after typechecking)
			// DEBUG: fmt.Printf("DEBUG: Skipping %s (not in type env): %v\n", name, err)
			continue
		}
		// DEBUG: fmt.Printf("DEBUG:   Got type for %s\n", name)

		// Generalize the type at module boundary
		scheme, err := b.generalizeType(typ, name)
		if err != nil {
			return nil, fmt.Errorf("failed to generalize export %s: %w", name, err)
		}

		// Determine purity (for now, assume pure unless marked otherwise)
		purity := b.determinePurity(binding)

		// Create interface item
		item := &IfaceItem{
			Name:   name,
			Type:   scheme,
			Purity: purity,
			Ref: core.GlobalRef{
				Module: b.module,
				Name:   name,
			},
		}

		iface.Exports[name] = item
	}

	// Add constructors to interface if provided
	for ctorName, ctorInfo := range constructors {
		// For now, we don't have full type information for constructor fields
		// We'll use placeholder types that will be refined later
		// The TypeName from the ADT declaration becomes the result type
		resultType := &types.TCon{Name: ctorInfo.TypeName}

		// Create placeholder field types (will be refined by type checker)
		fieldTypes := make([]types.Type, ctorInfo.Arity)
		for i := 0; i < ctorInfo.Arity; i++ {
			fieldTypes[i] = &types.TVar2{Name: fmt.Sprintf("a%d", i), Kind: types.Star}
		}

		iface.AddConstructor(ctorInfo.TypeName, ctorName, fieldTypes, resultType)
	}

	// Extract and add type declarations if AST is provided
	// DEBUG: fmt.Printf("DEBUG: astFile=%v (nil=%v)\n", astFile != nil, astFile == nil)
	if astFile != nil {
		// DEBUG: fmt.Printf("DEBUG: astFile type: %T\n", astFile)
		if file, ok := astFile.(*ast.File); ok {
			// DEBUG: fmt.Printf("DEBUG: Extracting types from AST, found %d Decls and %d Statements\n", len(file.Decls), len(file.Statements))
			// Check both Decls and Statements for type declarations
			allDecls := append(file.Decls, file.Statements...)
			for _, decl := range allDecls {
				if typeDecl, ok := decl.(*ast.TypeDecl); ok {
					// DEBUG: fmt.Printf("DEBUG: Found type declaration %s, Exported=%v\n", typeDecl.Name, typeDecl.Exported)
					if typeDecl.Exported {
						// Add type to interface
						arity := len(typeDecl.TypeParams)
						iface.AddType(typeDecl.Name, arity)
						// DEBUG: fmt.Printf("DEBUG: Added type %s to interface (arity %d)\n", typeDecl.Name, arity)

						// Extract constructors from algebraic types
						if algType, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
							// DEBUG: fmt.Printf("DEBUG: Type %s is algebraic with %d constructors\n", typeDecl.Name, len(algType.Constructors))
							for range algType.Constructors {
								// Add constructor to exports (will be importable)
								// The actual constructor scheme was already added above
								// Just mark it as exportable here
								// DEBUG: fmt.Printf("DEBUG: Type %s exports constructor %s\n", typeDecl.Name, ctor.Name)
							}
						}
					}
				}
			}
		}
	}

	// Compute deterministic digest
	iface.Schema = "ailang.iface/v1"
	digest, err := b.computeDigest(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to compute interface digest: %w", err)
	}
	iface.Digest = digest

	return iface, nil
}

// extractExports identifies exportable bindings
func (b *Builder) extractExports(prog *core.Program) (map[string]core.CoreExpr, error) {
	exports := make(map[string]core.CoreExpr)

	// DEBUG: Show metadata
	// if prog.Meta != nil {
	// 	fmt.Printf("DEBUG BuildInterface: module %s has metadata with %d entries\n", b.module, len(prog.Meta))
	// 	for name, meta := range prog.Meta {
	// 		fmt.Printf("  %s: IsExport=%v, IsPure=%v\n", name, meta.IsExport, meta.IsPure)
	// 	}
	// } else {
	// 	fmt.Printf("DEBUG BuildInterface: module %s has NO metadata\n", b.module)
	// }

	// Use metadata to determine exports
	if prog.Meta != nil {
		for _, decl := range prog.Decls {
			switch d := decl.(type) {
			case *core.Let:
				// DEBUG: fmt.Printf("DEBUG: Found Let %s\n", d.Name)
				if meta, ok := prog.Meta[d.Name]; ok {
					// Only export explicitly marked functions that don't start with underscore
					if meta.IsExport && !strings.HasPrefix(d.Name, "_") {
						// DEBUG: fmt.Printf("DEBUG: Adding export %s\n", d.Name)
						exports[d.Name] = d.Value
					}
				}
			case *core.LetRec:
				// DEBUG: fmt.Printf("DEBUG: Found LetRec with %d bindings\n", len(d.Bindings))
				for _, binding := range d.Bindings {
					// DEBUG: fmt.Printf("DEBUG:   Binding %s\n", binding.Name)
					if meta, ok := prog.Meta[binding.Name]; ok {
						// DEBUG: fmt.Printf("DEBUG:     Has metadata: IsExport=%v\n", meta.IsExport)
						if meta.IsExport && !strings.HasPrefix(binding.Name, "_") {
							// DEBUG: fmt.Printf("DEBUG: Adding export %s from LetRec\n", binding.Name)
							exports[binding.Name] = binding.Value
						}
					}
					// No else needed - if no metadata, we skip the binding
				}
			}
		}
	} else {
		// Fallback: no metadata means no exports (safer than exporting everything)
		return exports, nil
	}

	return exports, nil
}

// generalizeType converts a type to a type scheme, generalizing at module boundary
func (b *Builder) generalizeType(typ interface{}, name string) (*types.Scheme, error) {
	// If already a scheme, canonicalize it
	if scheme, ok := typ.(*types.Scheme); ok {
		return b.canonicalizeScheme(scheme)
	}

	// If it's a monotype, generalize it
	if monotype, ok := typ.(types.Type); ok {
		// Get free type variables
		// TODO: Implement proper free variable collection for types
		freeVars := []string{}
		freeRowVars := []string{}

		// Check for escaping type variables (shouldn't happen after proper typechecking)
		if len(freeVars) > 0 {
			// Check if these are legitimate polymorphic variables
			envFreeVars := b.typeEnv.FreeTypeVars()
			for _, v := range freeVars {
				if !envFreeVars[v] {
					// This is a legitimate polymorphic variable, OK to generalize
					continue
				}
				// This variable escapes from the environment - error
				return nil, fmt.Errorf("type variable %s escapes in export %s", v, name)
			}
		}

		// Create scheme with quantified variables
		quantified := make([]string, len(freeVars))
		copy(quantified, freeVars)
		sort.Strings(quantified) // Deterministic ordering

		rowVars := make([]string, len(freeRowVars))
		copy(rowVars, freeRowVars)
		sort.Strings(rowVars) // Deterministic ordering

		return &types.Scheme{
			TypeVars: quantified,
			RowVars:  rowVars,
			Type:     monotype,
		}, nil
	}

	return nil, fmt.Errorf("unexpected type kind for export %s: %T", name, typ)
}

// canonicalizeScheme ensures deterministic representation of a scheme
func (b *Builder) canonicalizeScheme(scheme *types.Scheme) (*types.Scheme, error) {
	// Sort quantified variables for deterministic ordering
	typeVars := make([]string, len(scheme.TypeVars))
	copy(typeVars, scheme.TypeVars)
	sort.Strings(typeVars)

	rowVars := make([]string, len(scheme.RowVars))
	copy(rowVars, scheme.RowVars)
	sort.Strings(rowVars)

	// TODO: Alpha-normalize the type to ensure consistent variable naming
	// For now, just return with sorted quantifiers
	return &types.Scheme{
		TypeVars: typeVars,
		RowVars:  rowVars,
		Type:     scheme.Type,
	}, nil
}

// determinePurity analyzes an expression to determine if it's pure
func (b *Builder) determinePurity(expr core.CoreExpr) bool {
	// TODO: Implement actual purity analysis
	// For now, assume functions are pure unless they have IO/effect annotations
	switch expr.(type) {
	case *core.Lambda:
		return true
	case *core.Lit:
		return true
	default:
		// Conservative: assume impure if we're not sure
		return true // For now, default to pure
	}
}

// ifaceItem is used for JSON serialization
type ifaceItem struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // String representation of the scheme
	Pure    bool     `json:"pure"`
	Effects []string `json:"effects,omitempty"`
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
	}

	// Convert to JSON-friendly format with sorted keys
	ji := jsonIface{
		Module:       iface.Module,
		Schema:       iface.Schema,
		Exports:      make(map[string]ifaceItem),
		Constructors: make(map[string]ctorItem),
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

// contains checks if a string slice contains a value

package iface

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Builder constructs module interfaces from typed Core programs
type Builder struct {
	module  string
	typeEnv *types.TypeEnv
}

// astTypeToInternalType converts an AST type to an internal type
// This is used during interface building to get actual constructor field types
func astTypeToInternalType(t ast.Type) types.Type {
	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return types.TInt
		case "float":
			return types.TFloat
		case "string":
			return types.TString
		case "bool":
			return types.TBool
		case "()":
			return types.TUnit
		case "bytes":
			return types.TBytes
		default:
			// Type variable or constructor
			if len(typ.Name) > 0 && typ.Name[0] >= 'a' && typ.Name[0] <= 'z' {
				return &types.TVar2{Name: typ.Name, Kind: types.Star}
			}
			return &types.TCon{Name: typ.Name}
		}

	case *ast.FuncType:
		paramTypes := make([]types.Type, len(typ.Params))
		for i, p := range typ.Params {
			paramTypes[i] = astTypeToInternalType(p)
		}
		var effectRow *types.Row
		if len(typ.Effects) > 0 {
			labels := make(map[string]types.Type)
			var tail *types.RowVar
			for _, e := range typ.Effects {
				if e.IsRowVar {
					tail = &types.RowVar{Name: e.Name, Kind: types.EffectRow}
				} else {
					labels[e.Name] = types.TUnit
				}
			}
			effectRow = &types.Row{
				Kind:   types.EffectRow,
				Labels: labels,
				Tail:   tail,
			}
		} else {
			effectRow = types.EmptyEffectRow()
		}
		return &types.TFunc2{
			Params:    paramTypes,
			EffectRow: effectRow,
			Return:    astTypeToInternalType(typ.Return),
		}

	case *ast.ListType:
		return &types.TList{
			Element: astTypeToInternalType(typ.Element),
		}

	case *ast.ArrayType:
		return &types.TArray{
			Element: astTypeToInternalType(typ.Element),
		}

	case *ast.TupleType:
		elements := make([]types.Type, len(typ.Elements))
		for i, e := range typ.Elements {
			elements[i] = astTypeToInternalType(e)
		}
		return &types.TTuple{Elements: elements}

	case *ast.RecordType:
		// Convert record type fields
		labels := make(map[string]types.Type)
		for _, f := range typ.Fields {
			labels[f.Name] = astTypeToInternalType(f.Type)
		}
		return &types.TRecord{Fields: labels, Row: nil}

	case *ast.TypeVar:
		// Type variables in constructor fields (e.g., 'a' in Ok(a), 'e' in Err(e))
		return &types.TVar2{Name: typ.Name, Kind: types.Star}

	case *ast.TypeApp:
		// Type applications (e.g., Option[int], Result[a, e])
		args := make([]types.Type, len(typ.Args))
		for i, a := range typ.Args {
			args[i] = astTypeToInternalType(a)
		}
		return &types.TApp{
			Constructor: &types.TCon{Name: typ.Constructor},
			Args:        args,
		}

	case *ast.LabelledType:
		// IFC label — wrap the base type in TLabelled so the label survives
		// iface serialisation (M6). Refinements (`{not LABEL}`) are sink-side
		// only and are not stored on the type itself.
		base := astTypeToInternalType(typ.Base)
		if typ.Label != nil {
			return types.WithLabel(base, types.LabelConst(typ.Label.Name))
		}
		return base

	default:
		// No silent fallback — fail loudly per CLAUDE.md Section 2
		panic(fmt.Sprintf("astTypeToInternalType: unhandled ast.Type variant: %T", t))
	}
}

// extractLabel returns the IFC label declared on an AST type, or ⊥ if none.
// Refinements ({not LABEL}) are sink-side metadata and are not stored on the type.
func extractLabel(t ast.Type) types.Label {
	if lt, ok := t.(*ast.LabelledType); ok && lt.Label != nil {
		return types.LabelConst(lt.Label.Name)
	}
	return types.LabelBottom()
}

// applyLabelsFromAST takes a typechecker-produced TFunc2 (with labels stripped)
// and the original AST FuncDecl, and re-wraps each param/return position with
// the label that appeared in the surface syntax. Non-function types pass through.
//
// M-IFACE-NESTED-EFFECTS (v0.21.x bug fix): also re-applies the effect row to
// function-typed parameters. The type checker erases nested function-type
// effect rows during unification (only the outer function's effect row is
// preserved in the inferred scheme). Without restoration, callers see a
// `(...) -> () ! {}` parameter signature and get "incompatible closed rows"
// when they pass a lambda whose body has any effects (e.g. `! {IO}`).
// Discovered during M-MOTOKO-V021-EFFECT-ROW-MIGRATION while migrating
// motoko_agent to AILANG v0.21+.
func applyLabelsFromAST(t types.Type, fd *ast.FuncDecl) types.Type {
	fn, ok := t.(*types.TFunc2)
	if !ok {
		return t
	}
	newParams := make([]types.Type, len(fn.Params))
	copy(newParams, fn.Params)
	for i, p := range fd.Params {
		if i >= len(newParams) {
			break
		}
		if l := extractLabel(p.Type); l != types.LabelBottom() {
			newParams[i] = types.WithLabel(newParams[i], l)
		}
		newParams[i] = restoreNestedEffectRow(newParams[i], p.Type)
	}
	newRet := fn.Return
	if fd.ReturnType != nil {
		if l := extractLabel(fd.ReturnType); l != types.LabelBottom() {
			newRet = types.WithLabel(newRet, l)
		}
		newRet = restoreNestedEffectRow(newRet, fd.ReturnType)
	}
	return &types.TFunc2{
		Params:    newParams,
		EffectRow: fn.EffectRow,
		Return:    newRet,
	}
}

// restoreNestedEffectRow walks a typechecker-produced type and re-applies
// effect rows from the AST onto any nested *types.TFunc2 whose corresponding
// AST node is an *ast.FuncType with a non-empty effect annotation. This
// repairs the v0.21+ regression where unification erased nested function-type
// effect rows.
//
// Only restores from the AST when the AST explicitly annotated effects;
// otherwise the typechecker's row is left alone. This keeps row-polymorphic
// callback types (declared without `! {...}`) intact.
func restoreNestedEffectRow(typed types.Type, astT ast.Type) types.Type {
	astFunc, ok := astT.(*ast.FuncType)
	if !ok {
		return typed
	}
	typedFunc, ok := typed.(*types.TFunc2)
	if !ok {
		return typed
	}
	// Recurse into the nested function's params and return first.
	newParams := make([]types.Type, len(typedFunc.Params))
	for i, p := range typedFunc.Params {
		if i < len(astFunc.Params) {
			newParams[i] = restoreNestedEffectRow(p, astFunc.Params[i])
		} else {
			newParams[i] = p
		}
	}
	newRet := typedFunc.Return
	if astFunc.Return != nil {
		newRet = restoreNestedEffectRow(newRet, astFunc.Return)
	}
	effectRow := typedFunc.EffectRow
	if len(astFunc.Effects) > 0 {
		labels := make(map[string]types.Type)
		var tail *types.RowVar
		for _, e := range astFunc.Effects {
			if e.IsRowVar {
				tail = &types.RowVar{Name: e.Name, Kind: types.EffectRow}
			} else {
				labels[e.Name] = types.TUnit
			}
		}
		effectRow = &types.Row{
			Kind:   types.EffectRow,
			Labels: labels,
			Tail:   tail,
		}
	}
	return &types.TFunc2{
		Params:    newParams,
		EffectRow: effectRow,
		Return:    newRet,
	}
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
	TypeName       string
	CtorName       string
	Arity          int
	TypeParamCount int          // M-TAPP-FIX: Number of type parameters (e.g., Option[a] = 1)
	FieldTypes     []types.Type // M-IFACE-COMPACT-ADT-FIELDS: field types so the compact iface can render Ctor({...})
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
	iface := NewIface(b.module)

	// Extract exportable bindings from the program
	exports, err := b.extractExports(prog)
	if err != nil {
		return nil, err
	}

	// Build a map from function name to its AST declaration so we can recover
	// IFC labels (M6): the type checker strips labels for unification purposes,
	// but they must survive into the iface for cross-module flow analysis.
	funcDecls := map[string]*ast.FuncDecl{}
	if file, ok := astFile.(*ast.File); ok {
		for _, fd := range file.Funcs {
			funcDecls[fd.Name] = fd
		}
	}

	// Process each export
	for name, binding := range exports {
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

		// M6: re-apply IFC labels from the AST onto param/return positions of
		// function schemes. The type checker erases labels during unification;
		// this restores them for serialisation. Non-function schemes are unchanged.
		if fd, ok := funcDecls[name]; ok && scheme != nil {
			scheme.Type = applyLabelsFromAST(scheme.Type, fd)
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

		// M-TAPP-FIX: Build TApp for parameterized ADTs (e.g., Option[a])
		var resultType types.Type
		if ctorInfo.TypeParamCount > 0 {
			typeArgs := make([]types.Type, ctorInfo.TypeParamCount)
			for i := 0; i < ctorInfo.TypeParamCount; i++ {
				typeArgs[i] = &types.TVar2{Name: fmt.Sprintf("t%d", i), Kind: types.Star}
			}
			resultType = &types.TApp{
				Constructor: &types.TCon{Name: ctorInfo.TypeName},
				Args:        typeArgs,
			}
		} else {
			resultType = &types.TCon{Name: ctorInfo.TypeName}
		}

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

						// M-FIX-RECORD-UPDATE: Add type alias for record types
						// This allows cross-module record update to expand type names to their underlying structure
						if recordType, ok := typeDecl.Definition.(*ast.RecordType); ok {
							internalType := astTypeToInternalType(recordType)
							iface.AddTypeAlias(typeDecl.Name, internalType)
							// DEBUG: fmt.Printf("DEBUG: Added type alias %s -> %s\n", typeDecl.Name, internalType)
						}

						// M-XMOD-ALIAS: Add type alias for NON-record targets
						// (e.g. `type Row = Json`, `type UserId = int`, `type Handler = (Req) -> Resp`)
						// so they expand across module boundaries too. Mirrors the elaborator's
						// *ast.TypeAlias handling (elaborate/file_funcs.go), which is why these
						// aliases already work WITHIN a module but not across imports pre-fix.
						// Record aliases are covered above via *ast.RecordType; sum types and
						// nominal newtypes (`type Gen = Gen(int)`) are *ast.AlgebraicType and are
						// intentionally NOT matched here, so they stay nominal.
						if aliasType, ok := typeDecl.Definition.(*ast.TypeAlias); ok {
							if internalType := astTypeToInternalType(aliasType.Target); internalType != nil {
								iface.AddTypeAlias(typeDecl.Name, internalType)
							}
						}

						// Extract constructors from algebraic types with ACTUAL field types
						// This fixes the type pollution bug where placeholder TVar2s were shared
						if algType, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
							// DEBUG: fmt.Printf("DEBUG: Type %s is algebraic with %d constructors\n", typeDecl.Name, len(algType.Constructors))
							for _, ctor := range algType.Constructors {
								// Convert AST field types to internal types
								fieldTypes := make([]types.Type, len(ctor.Fields))
								for i, field := range ctor.Fields {
									fieldTypes[i] = astTypeToInternalType(field.Type)
								}

								// M-TAPP-FIX: Build TApp for parameterized ADTs
								var resultType types.Type
								if len(typeDecl.TypeParams) > 0 {
									typeArgs := make([]types.Type, len(typeDecl.TypeParams))
									for i, param := range typeDecl.TypeParams {
										typeArgs[i] = &types.TVar2{Name: param, Kind: types.Star}
									}
									resultType = &types.TApp{
										Constructor: &types.TCon{Name: typeDecl.Name},
										Args:        typeArgs,
									}
								} else {
									resultType = &types.TCon{Name: typeDecl.Name}
								}

								// Update/add constructor with actual field types (overwriting placeholders)
								iface.AddConstructor(typeDecl.Name, ctor.Name, fieldTypes, resultType)
								// DEBUG: fmt.Printf("DEBUG: Type %s exports constructor %s with fields %v\n", typeDecl.Name, ctor.Name, fieldTypes)
							}

							// M-STREAM-DX/M4: Register type alias for single-constructor ADTs wrapping a record
							// Enables cross-module field access on newtype-pattern ADTs (e.g., `item.name`)
							if len(algType.Constructors) == 1 && len(typeDecl.TypeParams) == 0 {
								singleCtor := algType.Constructors[0]
								if len(singleCtor.Fields) == 1 && singleCtor.Fields[0].Type != nil {
									if _, isRecord := singleCtor.Fields[0].Type.(*ast.RecordType); isRecord {
										recordType := astTypeToInternalType(singleCtor.Fields[0].Type)
										if recordType != nil {
											iface.AddTypeAlias(typeDecl.Name, recordType)
										}
									}
								}
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

	// Use metadata to determine exports
	if prog.Meta != nil {
		for _, decl := range prog.Decls {
			// Recursively extract exports from nested Let/LetRec structures
			b.extractExportsFromExpr(decl, prog.Meta, exports)
		}
	} else {
		// Fallback: no metadata means no exports (safer than exporting everything)
		return exports, nil
	}

	return exports, nil
}

// extractExportsFromExpr recursively extracts exports from nested Let/LetRec structures
// This handles module-level lets that wrap function declarations
func (b *Builder) extractExportsFromExpr(expr core.CoreExpr, meta map[string]*core.DeclMeta, exports map[string]core.CoreExpr) {
	switch d := expr.(type) {
	case *core.Let:
		if m, ok := meta[d.Name]; ok {
			// Only export explicitly marked functions that don't start with underscore
			if m.IsExport && !strings.HasPrefix(d.Name, "_") {
				exports[d.Name] = d.Value
			}
		}
		// Recursively check the body for nested Let/LetRec
		if d.Body != nil {
			b.extractExportsFromExpr(d.Body, meta, exports)
		}

	case *core.LetRec:
		for _, binding := range d.Bindings {
			if m, ok := meta[binding.Name]; ok {
				if m.IsExport && !strings.HasPrefix(binding.Name, "_") {
					exports[binding.Name] = binding.Value
				}
			}
		}
		// Recursively check the body for nested Let/LetRec
		if d.Body != nil {
			b.extractExportsFromExpr(d.Body, meta, exports)
		}
	}
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

// determinePurity analyzes an expression to determine if it's pure.
// TODO: implement real analysis — currently returns true unconditionally
// (the type-class/row-effect system is the actual source of purity info).
func (b *Builder) determinePurity(_ core.CoreExpr) bool {
	return true
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

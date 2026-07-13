package elaborate

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/types"
)

// Elaborator transforms surface AST to Core ANF
type Elaborator struct {
	nextID       uint64
	surfaceSpans map[uint64]ast.Pos  // Map Core IDs to surface positions
	effectAnnots map[uint64][]string // Map Core IDs to effect annotations from AST
	freshVarNum  int                 // For generating fresh variable names
	moduleLoader *loader.ModuleLoader
	filePath     string                      // Current file path for relative imports
	globalEnv    map[string]core.GlobalRef   // Global environment for imports (name -> GlobalRef)
	constructors map[string]*ConstructorInfo // Available constructors (name -> info)
	warnings     []*ExhaustivenessWarning    // Accumulated warnings
	exChecker    *ExhaustivenessChecker      // Exhaustiveness checker
	// typeAliases stores type aliases for expansion during type checking
	// M-BUGFIX: Maps alias names to their underlying types (e.g., "Coord" -> {x: int, y: int})
	typeAliases map[string]types.Type
	// aliasParams stores the type-parameter names for PARAMETERIZED aliases
	// (M-XMOD-ALIAS-POLY). Maps alias name -> ordered param names
	// (e.g. "Box" -> ["a"] for `type Box[a] = {items: [a]}`). A missing entry
	// means the alias is nullary (arity 0). Kept as a sibling map so the
	// existing typeAliases[name] -> body path is unchanged.
	aliasParams map[string][]string
	// M-FIX-FLOAT-OP: Parameter type annotations from function declarations
	// Maps Lambda NodeID -> parameter types (preserves float annotations through elaboration)
	paramTypeAnnots map[uint64][]types.Type
	// M-FIX-FLOAT-OP: Return type annotations from function declarations
	// Maps Lambda NodeID -> return type (ensures PI() -> float is actually float)
	returnTypeAnnots map[uint64]types.Type
	// M-TYPE-LIST-ELEMENT-SOUNDNESS: let-binding type annotations.
	// Maps the resulting core.Let NodeID -> the annotated type, so that
	// `let xs: [string] = [42]` actually constrains inference (otherwise the
	// annotation was silently dropped and the int element leaked through).
	letTypeAnnots map[uint64]types.Type
	// M-CAPABILITY-BUDGETS: Full effect annotations with budgets
	// Maps Lambda NodeID -> full effect annotations (preserves @limit=N through elaboration)
	effectAnnotsFull map[uint64][]ast.EffectAnnotation
	// M-DX19: Types that derive Eq automatically
	// Tracks ADT/record types with `deriving (Eq)` clause
	derivedEqTypes map[string]bool
}

// ConstructorInfo holds information about an available constructor
type ConstructorInfo struct {
	TypeName       string       // The ADT type name (e.g., "Option")
	CtorName       string       // Constructor name (e.g., "Some")
	Arity          int          // Number of fields
	IsImported     bool         // Whether this constructor is imported
	TypeParamCount int          // M-TAPP-FIX: Number of type parameters (e.g., Option[a] = 1)
	TypeParamNames []string     // M-POLY-ADT: Type parameter names (e.g., ["a"] for Result[a])
	FieldTypes     []types.Type // M-POLY-ADT: Actual field types from AST (e.g., [string] for Err(string))
}

// NewElaborator creates a new elaborator
func NewElaborator() *Elaborator {
	return &Elaborator{
		nextID:           1,
		surfaceSpans:     make(map[uint64]ast.Pos),
		effectAnnots:     make(map[uint64][]string),
		effectAnnotsFull: make(map[uint64][]ast.EffectAnnotation), // M-CAPABILITY-BUDGETS
		freshVarNum:      0,
		globalEnv:        make(map[string]core.GlobalRef),
		constructors:     make(map[string]*ConstructorInfo),
		warnings:         []*ExhaustivenessWarning{},
		exChecker:        NewExhaustivenessChecker(),
		typeAliases:      make(map[string]types.Type),   // M-BUGFIX: Initialize type aliases
		paramTypeAnnots:  make(map[uint64][]types.Type), // M-FIX-FLOAT-OP: Initialize param annotations
		returnTypeAnnots: make(map[uint64]types.Type),   // M-FIX-FLOAT-OP: Initialize return annotations
		letTypeAnnots:    make(map[uint64]types.Type),   // M-TYPE-LIST-ELEMENT-SOUNDNESS: let annotations
		derivedEqTypes:   make(map[string]bool),         // M-DX19: Initialize derived Eq types
	}
}

// NewElaboratorWithPath creates a new elaborator with file path for imports
func NewElaboratorWithPath(filePath string) *Elaborator {
	dir := filepath.Dir(filePath)
	return &Elaborator{
		nextID:           1,
		surfaceSpans:     make(map[uint64]ast.Pos),
		effectAnnots:     make(map[uint64][]string),
		effectAnnotsFull: make(map[uint64][]ast.EffectAnnotation), // M-CAPABILITY-BUDGETS
		freshVarNum:      0,
		moduleLoader:     loader.NewModuleLoader(dir),
		filePath:         filePath,
		globalEnv:        make(map[string]core.GlobalRef),
		constructors:     make(map[string]*ConstructorInfo),
		warnings:         []*ExhaustivenessWarning{},
		exChecker:        NewExhaustivenessChecker(),
		typeAliases:      make(map[string]types.Type),   // M-BUGFIX: Initialize type aliases
		paramTypeAnnots:  make(map[uint64][]types.Type), // M-FIX-FLOAT-OP: Initialize param annotations
		returnTypeAnnots: make(map[uint64]types.Type),   // M-FIX-FLOAT-OP: Initialize return annotations
		letTypeAnnots:    make(map[uint64]types.Type),   // M-TYPE-LIST-ELEMENT-SOUNDNESS: let annotations
		derivedEqTypes:   make(map[string]bool),         // M-DX19: Initialize derived Eq types
	}
}

// SetGlobalEnv sets the global environment for import resolution
// WARNING: This REPLACES the entire globalEnv map. If you've already called
// AddBuiltinsToGlobalEnv(), use MergeGlobalEnv() instead to avoid losing builtins.
func (e *Elaborator) SetGlobalEnv(env map[string]core.GlobalRef) {
	e.globalEnv = env
}

// MergeGlobalEnv adds entries to the existing global environment
// Use this after AddBuiltinsToGlobalEnv() to preserve builtin references
// while adding import aliases and direct symbol imports.
func (e *Elaborator) MergeGlobalEnv(env map[string]core.GlobalRef) {
	for name, ref := range env {
		e.globalEnv[name] = ref
	}
}

// SetModuleLoader sets the module loader for import resolution
func (e *Elaborator) SetModuleLoader(ml *loader.ModuleLoader) {
	e.moduleLoader = ml
}

// AddBuiltinsToGlobalEnv adds all builtin functions to the global environment
func (e *Elaborator) AddBuiltinsToGlobalEnv() {
	// Add all registered builtins to global environment
	// Use AllSpecs() to get the complete set of builtins, not just the limited Registry
	// This ensures builtins like _string_intToStr, _clock_now, etc. are properly elaborated
	// as VarGlobal references instead of Var (which would fail at runtime with "undefined variable")
	for name := range builtins.AllSpecs() {
		e.globalEnv[name] = core.GlobalRef{
			Module: "$builtin",
			Name:   name,
		}
	}
}

// RegisterConstructor adds a constructor to the elaborator's constructor map
// M-TAPP-FIX: Added typeParamCount to track ADT type parameters
// M-POLY-ADT: For backward compatibility, calls RegisterConstructorWithFields with nil fieldTypes
func (e *Elaborator) RegisterConstructor(typeName, ctorName string, arity int, isImported bool, typeParamCount int) {
	e.RegisterConstructorWithFields(typeName, ctorName, arity, isImported, typeParamCount, nil, nil)
}

// RegisterConstructorWithFields adds a constructor with actual field types
// M-POLY-ADT: Stores field types to correctly build constructor type schemes
// This fixes the bug where Err(string) in Result[a] was incorrectly typed as ∀a. a -> Result[a]
func (e *Elaborator) RegisterConstructorWithFields(typeName, ctorName string, arity int, isImported bool, typeParamCount int, typeParamNames []string, fieldTypes []types.Type) {
	e.constructors[ctorName] = &ConstructorInfo{
		TypeName:       typeName,
		CtorName:       ctorName,
		Arity:          arity,
		IsImported:     isImported,
		TypeParamCount: typeParamCount,
		TypeParamNames: typeParamNames,
		FieldTypes:     fieldTypes,
	}
}

// GetConstructors returns all constructors defined in this module (not imported)
func (e *Elaborator) GetConstructors() map[string]*ConstructorInfo {
	localConstructors := make(map[string]*ConstructorInfo)
	for name, info := range e.constructors {
		if !info.IsImported {
			localConstructors[name] = info
		}
	}
	return localConstructors
}

// RegisterTypeAlias registers a type alias for expansion during type checking
// M-BUGFIX: This allows `type Coord = {x: int, y: int}` to work with ADT variants
// M-TYPENAME-NESTED-PROPAGATION: Set TypeName on TRecord so unification can propagate it
func (e *Elaborator) RegisterTypeAlias(name string, target types.Type) {
	if e.typeAliases == nil {
		e.typeAliases = make(map[string]types.Type)
	}
	// M-TYPENAME-NESTED-PROPAGATION: If target is a TRecord, set its TypeName
	// This ensures that when the alias is expanded during unification, the
	// TRecord carries the nominal type identity for codegen
	if rec, ok := target.(*types.TRecord); ok && rec.TypeName == "" {
		rec.TypeName = name
	}
	e.typeAliases[name] = target
}

// GetTypeAliases returns all type aliases registered during elaboration
// M-BUGFIX: Used to pass aliases to the type checker for expansion during unification
func (e *Elaborator) GetTypeAliases() map[string]types.Type {
	return e.typeAliases
}

// RegisterTypeAliasParams records the ordered type-parameter names for a
// parameterized alias (M-XMOD-ALIAS-POLY). A nil/empty list is not stored
// (a missing entry naturally means "nullary alias").
func (e *Elaborator) RegisterTypeAliasParams(name string, params []string) {
	if len(params) == 0 {
		return
	}
	if e.aliasParams == nil {
		e.aliasParams = make(map[string][]string)
	}
	e.aliasParams[name] = params
}

// GetTypeAliasParams returns the parameter names for all parameterized aliases
// registered during elaboration (M-XMOD-ALIAS-POLY).
func (e *Elaborator) GetTypeAliasParams() map[string][]string {
	return e.aliasParams
}

// GetEffectAnnotation returns the effect annotation for a Core node ID
func (e *Elaborator) GetEffectAnnotation(nodeID uint64) []string {
	return e.effectAnnots[nodeID]
}

// GetEffectAnnotationsFull returns all full effect annotations (Lambda NodeID -> full annotations)
// M-CAPABILITY-BUDGETS: Used to pass budget annotations from func declarations to type checker
func (e *Elaborator) GetEffectAnnotationsFull() map[uint64][]ast.EffectAnnotation {
	return e.effectAnnotsFull
}

// GetParamTypeAnnotations returns all parameter type annotations (Lambda NodeID -> param types)
// M-FIX-FLOAT-OP: Used to pass float annotations from func declarations to type checker
func (e *Elaborator) GetParamTypeAnnotations() map[uint64][]types.Type {
	return e.paramTypeAnnots
}

// GetReturnTypeAnnotations returns all return type annotations (Lambda NodeID -> return type)
// M-FIX-FLOAT-OP: Used to ensure PI() -> float actually returns float
func (e *Elaborator) GetReturnTypeAnnotations() map[uint64]types.Type {
	return e.returnTypeAnnots
}

// GetLetTypeAnnotations returns all let-binding type annotations (Let NodeID -> annotated type)
// M-TYPE-LIST-ELEMENT-SOUNDNESS: Used so `let xs: [string] = [42]` is actually
// checked instead of having its annotation silently dropped during elaboration.
func (e *Elaborator) GetLetTypeAnnotations() map[uint64]types.Type {
	return e.letTypeAnnots
}

// GetDerivedEqTypes returns all types that have `deriving (Eq)` clause
// M-DX19: Used to register Eq instances for derived types in the type checker
func (e *Elaborator) GetDerivedEqTypes() []string {
	result := make([]string, 0, len(e.derivedEqTypes))
	for typeName := range e.derivedEqTypes {
		result = append(result, typeName)
	}
	return result
}

// GetWarnings returns accumulated exhaustiveness warnings
func (e *Elaborator) GetWarnings() []*ExhaustivenessWarning {
	return e.warnings
}

// ClearWarnings clears accumulated warnings
func (e *Elaborator) ClearWarnings() {
	e.warnings = []*ExhaustivenessWarning{}
}

// GetSurfaceSpan retrieves the original surface span for a Core node ID
func (e *Elaborator) GetSurfaceSpan(nodeID uint64) (ast.Pos, bool) {
	span, ok := e.surfaceSpans[nodeID]
	return span, ok
}

// Helper types and functions

type binding struct {
	Name  string
	Value core.CoreExpr
}

// makeNode creates a new CoreNode with unique ID
func (e *Elaborator) makeNode(pos ast.Pos) core.CoreNode {
	id := e.nextID
	e.nextID++
	e.surfaceSpans[id] = pos
	return core.CoreNode{
		NodeID:   id,
		CoreSpan: pos,
		OrigSpan: pos,
	}
}

// freshVar generates a fresh variable name
func (e *Elaborator) freshVar() string {
	e.freshVarNum++
	return fmt.Sprintf("$tmp%d", e.freshVarNum)
}

// normalizeToAtomic ensures expression is atomic, introducing let bindings if needed
func (e *Elaborator) normalizeToAtomic(expr ast.Expr) (core.CoreExpr, []binding, error) {
	normalized, err := e.normalize(expr)
	if err != nil {
		return nil, nil, err
	}

	if core.IsAtomic(normalized) {
		return normalized, nil, nil
	}

	// ANF completion: if the normalized expression is a Let, extract the inner bindings.
	// This handles deeply nested records where normalize() returns a Let chain.
	// We need to flatten this so our new binding doesn't have a Let as its value.
	innerBindings, flattenedValue := extractLetBindings(normalized)

	// Need to bind the (now flattened) non-atomic expression
	freshName := e.freshVar()
	bind := binding{Name: freshName, Value: flattenedValue}
	varRef := &core.Var{
		CoreNode: e.makeNode(expr.Position()),
		Name:     freshName,
	}

	// Prepend inner bindings to our new binding
	allBindings := append(innerBindings, bind)

	return varRef, allBindings, nil
}

// wrapWithBindings wraps expression with let bindings
func (e *Elaborator) wrapWithBindings(expr core.CoreExpr, bindings []binding) core.CoreExpr {
	result := expr
	// Apply bindings in reverse order (innermost first)
	for i := len(bindings) - 1; i >= 0; i-- {
		bind := bindings[i]
		result = &core.Let{
			CoreNode: e.makeNode(bind.Value.Span()),
			Name:     bind.Name,
			Value:    bind.Value,
			Body:     result,
		}
	}
	return result
}

// extractLetBindings extracts top-level Let bindings from an expression.
// This is used for ANF completion - ensuring no Let appears as a let RHS.
//
// For input: Let x = e1 in Let y = e2 in body
// Returns:   ([{x,e1}, {y,e2}], body)
//
// The returned bindings are in correct order: outermost Let first.
// This function does NOT descend into lambda bodies, LetRec, or other subexpressions.
func extractLetBindings(expr core.CoreExpr) ([]binding, core.CoreExpr) {
	var bindings []binding

	current := expr
	for {
		letExpr, ok := current.(*core.Let)
		if !ok {
			// Not a Let - we've reached the innermost body
			break
		}

		// Extract this binding
		bindings = append(bindings, binding{
			Name:  letExpr.Name,
			Value: letExpr.Value,
		})

		// Continue with the body
		current = letExpr.Body
	}

	return bindings, current
}

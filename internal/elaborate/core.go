package elaborate

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/types"
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
}

// ConstructorInfo holds information about an available constructor
type ConstructorInfo struct {
	TypeName   string // The ADT type name (e.g., "Option")
	CtorName   string // Constructor name (e.g., "Some")
	Arity      int    // Number of fields
	IsImported bool   // Whether this constructor is imported
}

// NewElaborator creates a new elaborator
func NewElaborator() *Elaborator {
	return &Elaborator{
		nextID:       1,
		surfaceSpans: make(map[uint64]ast.Pos),
		effectAnnots: make(map[uint64][]string),
		freshVarNum:  0,
		globalEnv:    make(map[string]core.GlobalRef),
		constructors: make(map[string]*ConstructorInfo),
		warnings:     []*ExhaustivenessWarning{},
		exChecker:    NewExhaustivenessChecker(),
		typeAliases:  make(map[string]types.Type), // M-BUGFIX: Initialize type aliases
	}
}

// NewElaboratorWithPath creates a new elaborator with file path for imports
func NewElaboratorWithPath(filePath string) *Elaborator {
	dir := filepath.Dir(filePath)
	return &Elaborator{
		nextID:       1,
		surfaceSpans: make(map[uint64]ast.Pos),
		effectAnnots: make(map[uint64][]string),
		freshVarNum:  0,
		moduleLoader: loader.NewModuleLoader(dir),
		filePath:     filePath,
		globalEnv:    make(map[string]core.GlobalRef),
		constructors: make(map[string]*ConstructorInfo),
		warnings:     []*ExhaustivenessWarning{},
		exChecker:    NewExhaustivenessChecker(),
		typeAliases:  make(map[string]types.Type), // M-BUGFIX: Initialize type aliases
	}
}

// SetGlobalEnv sets the global environment for import resolution
func (e *Elaborator) SetGlobalEnv(env map[string]core.GlobalRef) {
	e.globalEnv = env
}

// SetModuleLoader sets the module loader for import resolution
func (e *Elaborator) SetModuleLoader(ml *loader.ModuleLoader) {
	e.moduleLoader = ml
}

// AddBuiltinsToGlobalEnv adds all builtin functions to the global environment
func (e *Elaborator) AddBuiltinsToGlobalEnv() {
	// Add all registered builtins to global environment
	for name := range builtins.Registry {
		e.globalEnv[name] = core.GlobalRef{
			Module: "$builtin",
			Name:   name,
		}
	}
}

// RegisterConstructor adds a constructor to the elaborator's constructor map
func (e *Elaborator) RegisterConstructor(typeName, ctorName string, arity int, isImported bool) {
	e.constructors[ctorName] = &ConstructorInfo{
		TypeName:   typeName,
		CtorName:   ctorName,
		Arity:      arity,
		IsImported: isImported,
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
func (e *Elaborator) RegisterTypeAlias(name string, target types.Type) {
	if e.typeAliases == nil {
		e.typeAliases = make(map[string]types.Type)
	}
	e.typeAliases[name] = target
}

// GetTypeAliases returns all type aliases registered during elaboration
// M-BUGFIX: Used to pass aliases to the type checker for expansion during unification
func (e *Elaborator) GetTypeAliases() map[string]types.Type {
	return e.typeAliases
}

// GetEffectAnnotation returns the effect annotation for a Core node ID
func (e *Elaborator) GetEffectAnnotation(nodeID uint64) []string {
	return e.effectAnnots[nodeID]
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

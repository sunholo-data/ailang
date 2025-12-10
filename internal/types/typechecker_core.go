// Package types implements Hindley-Milner type inference for AILANG's Core AST.
//
// # CoreTypeInfo Contract (M-DX4)
//
// CoreTypeInfo (CoreTI) is a mapping from Core NodeID to inferred Type. It is the
// source of truth for type-guided code generation during lowering.
//
// **Contract**:
//   - CoreTI is TOTAL for all Core nodes after type checking completes
//   - Types may be type variables (TVar) before specialization/monomorphization
//   - Lowering of overloaded operators requires non-TVar heads (concrete types)
//   - If a Core node has no CoreTI entry, that is a COMPILER BUG
//
// **Phase Requirements**:
//   - Pre-monomorphization: CoreTI may contain TVars for polymorphic code (VALID)
//   - Post-monomorphization: Specialized bodies should have concrete types
//   - Post-VarResolution (v0.3.18+): Monomorphic Var nodes get concrete types
//   - Lowering phase: Operators need concrete heads; TVars trigger fallback
//
// **Validation**:
//   - ValidateCoreTypeInfo (internal/pipeline) checks 100% coverage before lowering
//   - Validation accepts TVars as valid (checks presence, not concreteness)
//   - Use --debug-compile flag to see telemetry of CoreTI hits/misses during lowering
//
// **Why TVars Remain After Type Inference (v0.3.18)**:
//
// After Hindley-Milner unification and ApplySubstitution, some Var nodes may still
// have TVars in CoreTI. This happens because:
//
//  1. Let-bound variables: The substitution tracks Let bindings but doesn't always
//     resolve Var references to their binding's concrete type
//  2. Polymorphic preservation: Lambda parameters intentionally keep TVars until
//     call-site specialization (M-POLY-B)
//
// The VarResolver (internal/pipeline/resolve_vars.go) is a WORKAROUND that propagates
// monomorphic types from Let bindings to Var usages. It only propagates concrete types
// (Int, Float, String, Bool, List) and preserves polymorphism for lambda params.
//
// **Future (M-POLY-B, v0.4.1+)**: Var-bound polymorphic lambdas will be re-elaborated
// after monomorphization, which will naturally resolve all operator types in specialized
// bodies. The VarResolver is a pragmatic bridge until then.
//
// **Debug**:
//   - ailang debug ast <file> --show-types --compact: Inspect CoreTI for a file
//   - ailang run --debug-compile <file>: See lowering telemetry (CoreTI coverage)
//   - Look for "Var type resolution complete" in debug output
//
// See: design_docs/planned/v0_3_18/M-DX4-SPRINT-PLAN.md
package types

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// CoreTypeChecker type checks Core AST and produces TypedAST
type CoreTypeChecker struct {
	instanceEnv         *InstanceEnv      // Type class instances
	defaultingConfig    *DefaultingConfig // Numeric defaulting configuration
	debugMode           bool              // Enable debug output
	useRecordsV2        bool              // Emit TRecord2 instead of TRecord (AILANG_RECORDS_V2)
	errors              []error
	resolvedConstraints map[uint64]*ResolvedConstraint // NodeID → resolved constraint
	globalTypes         map[string]*Scheme             // Global types for imports (module.name -> Scheme)
	instantiations      []Instantiation                // Track polymorphic instantiations for debugging
	trackInstantiations bool                           // Whether to track instantiations
	varCounter          int                            // Counter for generating fresh variable names
	effectAnnots        map[uint64][]string            // Effect annotations from elaboration (NodeID → effects)
	returnTypeAnnots    map[uint64]Type                // Return type annotations from elaboration (Lambda NodeID → return type)
	CoreTI              CoreTypeInfo                   // Core NodeID → inferred types (principal types for lowering)
	constructorTypes    map[string]string              // M-DX25.4: Constructor name → ADT type name (e.g., "Up" → "Direction")
	// aliasEnv maps type alias names to their underlying types
	// M-BUGFIX: Used for alias expansion during unification
	aliasEnv map[string]Type
	// M-FIX-FLOAT-OP: Parameter type annotations from function declarations
	// Maps Lambda NodeID -> parameter types to preserve float annotations through elaboration
	paramTypeAnnots map[uint64][]Type
}

// Instantiation records a polymorphic type instantiation for debugging
type Instantiation struct {
	Location     string   // File position "line:col"
	VarName      string   // Variable name
	FreshVars    []string // Fresh type variables generated
	Instantiated Type     // The instantiated monotype
}

// DumpInstantiations returns a JSON-serializable map of instantiations
func (tc *CoreTypeChecker) DumpInstantiations() map[string]interface{} {
	if !tc.trackInstantiations {
		return nil
	}

	result := make(map[string]interface{})
	result["instantiations"] = make([]map[string]interface{}, 0, len(tc.instantiations))

	for _, inst := range tc.instantiations {
		entry := map[string]interface{}{
			"location": inst.Location,
			"var":      inst.VarName,
			"fresh":    inst.FreshVars,
			"type":     inst.Instantiated.String(),
		}
		result["instantiations"] = append(result["instantiations"].([]map[string]interface{}), entry)
	}

	return result
}

// EnableInstantiationTracking turns on tracking of polymorphic instantiations
func (tc *CoreTypeChecker) EnableInstantiationTracking() {
	tc.trackInstantiations = true
	tc.instantiations = make([]Instantiation, 0)
}

// ResolvedConstraint records a resolved class constraint at a specific node
// This is used by the elaborator to insert dictionary passing
type ResolvedConstraint struct {
	NodeID    uint64 // Core node ID where constraint was resolved
	ClassName string // "Num", "Eq", "Ord", etc.
	Type      Type   // Normalized ground type (Int, Float, etc.)
	Method    string // Method name for operators: "add", "eq", "lt", etc.
}

// NewCoreTypeChecker creates a new Core type checker
func NewCoreTypeChecker() *CoreTypeChecker {
	var instanceEnv *InstanceEnv

	// Auto-import std/prelude instances unless explicitly disabled
	if os.Getenv("AILANG_NO_PRELUDE") == "1" {
		// Explicit mode: start with empty environment
		instanceEnv = NewInstanceEnv()
	} else {
		// Default mode: pre-load Eq, Ord, Num, Show instances
		// This eliminates the need for "import std/prelude (Ord, Eq)"
		instanceEnv = LoadBuiltinInstances()
	}

	// Set up default types for numeric literals
	instanceEnv.SetDefault("Num", &TCon{Name: "int"})
	instanceEnv.SetDefault("Fractional", &TCon{Name: "float"})

	// Check environment flag for records v2
	useRecordsV2 := os.Getenv("AILANG_RECORDS_V2") == "1"

	return &CoreTypeChecker{
		instanceEnv:         instanceEnv,
		defaultingConfig:    NewDefaultingConfig(), // Standard defaulting config
		debugMode:           false,
		useRecordsV2:        useRecordsV2,
		errors:              []error{},
		resolvedConstraints: make(map[uint64]*ResolvedConstraint),
		globalTypes:         make(map[string]*Scheme),
		effectAnnots:        make(map[uint64][]string),
		returnTypeAnnots:    make(map[uint64]Type),
		CoreTI:              NewCoreTypeInfo(),
		constructorTypes:    make(map[string]string),
		aliasEnv:            make(map[string]Type),   // M-BUGFIX: Initialize alias environment
		paramTypeAnnots:     make(map[uint64][]Type), // M-FIX-FLOAT-OP: Initialize param annotations
	}
}

// NewCoreTypeCheckerWithInstances creates a type checker with preloaded instances
func NewCoreTypeCheckerWithInstances(instances *InstanceEnv) *CoreTypeChecker {
	// Check environment flag for records v2
	useRecordsV2 := os.Getenv("AILANG_RECORDS_V2") == "1"

	return &CoreTypeChecker{
		instanceEnv:         instances,
		defaultingConfig:    NewDefaultingConfig(),
		debugMode:           false,
		useRecordsV2:        useRecordsV2,
		errors:              []error{},
		resolvedConstraints: make(map[uint64]*ResolvedConstraint),
		globalTypes:         make(map[string]*Scheme),
		effectAnnots:        make(map[uint64][]string),
		returnTypeAnnots:    make(map[uint64]Type),
		CoreTI:              NewCoreTypeInfo(),
		constructorTypes:    make(map[string]string),
		aliasEnv:            make(map[string]Type),   // M-BUGFIX: Initialize alias environment
		paramTypeAnnots:     make(map[uint64][]Type), // M-FIX-FLOAT-OP: Initialize param annotations
	}
}

// RegisterTypeAlias registers a type alias for expansion during unification
// M-BUGFIX: This allows `type Coord = {x: int, y: int}` to work with ADT variants
func (tc *CoreTypeChecker) RegisterTypeAlias(name string, target Type) {
	if tc.aliasEnv == nil {
		tc.aliasEnv = make(map[string]Type)
	}
	tc.aliasEnv[name] = target
}

// GetAliasEnv returns the type alias environment for use in unification
// M-BUGFIX: Used to create UnifierWithAliases
func (tc *CoreTypeChecker) GetAliasEnv() map[string]Type {
	return tc.aliasEnv
}

// SetGlobalTypes sets the global types for import resolution
func (tc *CoreTypeChecker) SetGlobalTypes(types map[string]*Scheme) {
	tc.globalTypes = types
}

// SetGlobalType sets a single global type scheme
func (tc *CoreTypeChecker) SetGlobalType(key string, scheme *Scheme) {
	if tc.globalTypes == nil {
		tc.globalTypes = make(map[string]*Scheme)
	}
	tc.globalTypes[key] = scheme
}

// SetConstructorTypes sets the constructor → ADT type mappings.
// M-DX25.4: Used to infer correct types for pattern matching on ADTs.
func (tc *CoreTypeChecker) SetConstructorTypes(ctors map[string]string) {
	tc.constructorTypes = ctors
}

// RegisterConstructorType registers a single constructor → ADT type mapping.
// M-DX25.4: Used to infer correct types for pattern matching on ADTs.
func (tc *CoreTypeChecker) RegisterConstructorType(ctorName, typeName string) {
	if tc.constructorTypes == nil {
		tc.constructorTypes = make(map[string]string)
	}
	tc.constructorTypes[ctorName] = typeName
}

// SetDebugMode enables debug output for defaulting traces
func (tc *CoreTypeChecker) SetDebugMode(debug bool) {
	tc.debugMode = debug
}

// EnableTraceDefaulting enables defaulting trace output
func (tc *CoreTypeChecker) EnableTraceDefaulting(enable bool) {
	tc.debugMode = enable
}

// SetDefaultingConfig sets a custom defaulting configuration
func (tc *CoreTypeChecker) SetDefaultingConfig(config *DefaultingConfig) {
	tc.defaultingConfig = config
}

// SetEffectAnnotations sets effect annotations from elaboration
func (tc *CoreTypeChecker) SetEffectAnnotations(annots map[uint64][]string) {
	tc.effectAnnots = annots
}

// SetParamTypeAnnotations sets parameter type annotations from elaboration
// M-FIX-FLOAT-OP: This preserves float parameter annotations through elaboration
func (tc *CoreTypeChecker) SetParamTypeAnnotations(annots map[uint64][]Type) {
	tc.paramTypeAnnots = annots
}

// SetReturnTypeAnnotations sets return type annotations from elaboration
// M-FIX-FLOAT-OP: This ensures PI() -> float ACTUALLY constrains inference to return float
func (tc *CoreTypeChecker) SetReturnTypeAnnotations(annots map[uint64]Type) {
	tc.returnTypeAnnots = annots
}

// InferWithConstraints infers type with constraints for a Core expression
// Returns: typed expression, updated env, qualified type, constraints, error
func (tc *CoreTypeChecker) InferWithConstraints(expr core.CoreExpr, env *TypeEnv) (typedast.TypedNode, *TypeEnv, Type, []Constraint, error) {
	// M-BUGFIX: Create unifier with alias environment for type alias expansion
	var unifier *Unifier
	if len(tc.aliasEnv) > 0 {
		unifier = NewUnifierWithAliases(tc.aliasEnv)
	} else {
		unifier = NewUnifier()
	}

	// Create inference context
	ctx := &InferenceContext{
		env:                  env,
		unifier:              unifier,
		constraints:          []TypeConstraint{},
		freshCounter:         0,
		path:                 []string{},
		qualifiedConstraints: []ClassConstraint{},
	}

	// Infer type (returns updated env)
	typedNode, updatedEnv, err := tc.inferCore(ctx, expr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Get the inferred type
	inferredType := typedNode.GetType()

	// Apply substitution if we have one
	var finalType Type
	if typ, ok := inferredType.(Type); ok {
		finalType = typ
	} else {
		finalType = &TCon{Name: "Unknown"}
	}

	// Solve constraints and apply defaulting (proper way)
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Apply defaulting to unsolved constraints
	defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguitiesTopLevel(finalType, unsolved)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("defaulting failed: %w", err)
	}

	// Compose substitutions if defaulting was applied
	if len(defaultingSub) > 0 {
		sub = composeSubstitutions(defaultingSub, sub)
		finalType = defaultedType
		unsolved = defaultedConstraints
	}

	// M-DX4 FIX: Apply FULL substitution (unification + defaulting) to CoreTypeInfo
	// This ensures CoreTI has concrete types (Int, Float, etc.) instead of type variables.
	// Must apply the composed substitution to resolve chains (e.g., α37 → α38 → Float).
	tc.CoreTI.ApplySubstitution(sub)

	// Apply final substitution to typed node
	typedNode = tc.applySubstitutionToTyped(sub, typedNode)

	// Resolve ground constraints
	ground, nonGround := tc.partitionConstraints(unsolved)
	if err := tc.resolveGroundConstraints(ground, expr); err != nil {
		return nil, updatedEnv, nil, nil, err
	}

	// Fill in operator methods
	tc.FillOperatorMethods(expr)

	// Convert ClassConstraints to Constraints for return value
	constraints := make([]Constraint, len(nonGround))
	for i, cc := range nonGround {
		constraints[i] = Constraint{
			Class: cc.Class,
			Type:  cc.Type,
		}
	}

	// Return with updated env, constraints and defaulted type
	return typedNode, updatedEnv, finalType, constraints, nil
}

// GetResolvedConstraints returns the map of resolved constraints
// Used by the elaborator for dictionary passing transformation
func (tc *CoreTypeChecker) GetResolvedConstraints() map[uint64]*ResolvedConstraint {
	// CRITICAL: Final groundness check before export to elaborator
	for nodeID, rc := range tc.resolvedConstraints {
		if !isGround(rc.Type) {
			panic(fmt.Sprintf("CRITICAL BUG: exporting non-ground ResolvedConstraint[%d] with type %s - this should never happen after defaulting", nodeID, rc.Type))
		}
	}
	return tc.resolvedConstraints
}

// CheckCoreProgram type checks a Core program and produces TypedAST
func (tc *CoreTypeChecker) CheckCoreProgram(prog *core.Program) (*typedast.TypedProgram, error) {
	typed := &typedast.TypedProgram{
		Decls: make([]typedast.TypedNode, 0),
	}

	// Create global environment with builtins
	globalEnv := NewTypeEnvWithBuiltins()

	for _, decl := range prog.Decls {
		typedNode, env, err := tc.CheckCoreExpr(decl, globalEnv)
		if err != nil {
			tc.errors = append(tc.errors, err)
			continue
		}
		typed.Decls = append(typed.Decls, typedNode)
		globalEnv = env // Update environment with new bindings
	}

	// Report all errors
	if len(tc.errors) > 0 {
		return nil, tc.formatErrors()
	}

	return typed, nil
}

// CheckCoreExpr type checks a Core expression (exported for testing)
func (tc *CoreTypeChecker) CheckCoreExpr(expr core.CoreExpr, env *TypeEnv) (typedast.TypedNode, *TypeEnv, error) {
	ctx := NewInferenceContext()
	ctx.env = env

	// Infer type and effects
	typedNode, newEnv, err := tc.inferCore(ctx, expr)
	if err != nil {
		return nil, env, err
	}

	// Solve type equality constraints first
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		return nil, env, err
	}

	if tc.debugMode {
		fmt.Printf("[debug] Unification substitution: %v\n", sub)
		fmt.Printf("[debug] Unsolved after unification: ")
		for _, c := range unsolved {
			fmt.Printf("%s[%s] ", c.Class, c.Type)
		}
		fmt.Println()
	}

	// CRITICAL: Apply defaulting at top-level/REPL generalization boundary
	// This happens AFTER unification, BEFORE constraint partitioning
	if tc.debugMode {
		fmt.Printf("[debug] Unsolved constraints before defaulting: %d\n", len(unsolved))
		for _, c := range unsolved {
			fmt.Printf("  - %s[%s]\n", c.Class, c.Type)
		}
	}

	// Apply spec-compliant defaulting at this generalization boundary
	// For top-level expressions, also default non-ambiguous numeric literals
	exprType := typedNode.GetType().(Type)
	defaultingSub, _, defaultedConstraints, err := tc.defaultAmbiguitiesTopLevel(exprType, unsolved)
	if err != nil {
		return nil, newEnv, fmt.Errorf("defaulting failed: %w", err)
	}

	// Apply defaulting substitution everywhere if any defaults were applied
	if len(defaultingSub) > 0 {
		// Compose with existing substitution
		sub = composeSubstitutions(defaultingSub, sub)

		// Use defaulted values (constraints are already substituted by defaultAmbiguities)
		// exprType = defaultedType // Not used after this point
		unsolved = defaultedConstraints

		if tc.debugMode {
			fmt.Printf("[debug] Applied defaulting substitution: %v\n", defaultingSub)
			fmt.Printf("[debug] Defaulted constraints: ")
			for _, c := range defaultedConstraints {
				fmt.Printf("%s[%s] ", c.Class, c.Type)
			}
			fmt.Println()
		}
	} else if tc.debugMode {
		fmt.Println("[debug] No defaulting applied")
	}

	// M-DX4 FIX V2: Apply FULL substitution (unification + defaulting) to CoreTypeInfo
	// Must be AFTER composition so we have the complete substitution with chains resolved
	tc.CoreTI.ApplySubstitution(sub)

	// Apply the complete substitution (unification + defaulting) to the typed node
	typedNode = tc.applySubstitutionToTyped(sub, typedNode)

	// The constraints from defaulting should already be properly substituted
	// Don't double-apply substitution
	groundConstraints := unsolved

	// Partition into ground and non-ground constraints
	ground, nonGround := tc.partitionConstraints(groundConstraints)

	// Resolve ground constraints using instance environment
	if err := tc.resolveGroundConstraints(ground, expr); err != nil {
		return nil, env, err
	}

	// Non-ground constraints become part of qualified type schemes
	// (will be handled during generalization)
	if len(nonGround) > 0 {
		// Store for later use in type scheme
		ctx.qualifiedConstraints = nonGround
	}

	// Apply substitution to typed node
	typedNode = tc.applySubstitutionToTyped(sub, typedNode)

	// Fill in operator methods for resolved constraints
	tc.FillOperatorMethods(expr)

	return typedNode, newEnv, nil
}

// inferCore performs type inference on Core expressions
func (tc *CoreTypeChecker) inferCore(ctx *InferenceContext, expr core.CoreExpr) (typedast.TypedNode, *TypeEnv, error) {
	var typedNode typedast.TypedNode
	var env *TypeEnv
	var err error

	switch e := expr.(type) {
	case *core.Lit:
		typedNode, env, err = tc.inferLit(ctx, e)

	case *core.Var:
		typedNode, env, err = tc.inferVar(ctx, e)

	case *core.VarGlobal:
		typedNode, env, err = tc.inferVarGlobal(ctx, e)

	case *core.Lambda:
		typedNode, env, err = tc.inferLambda(ctx, e)

	case *core.Let:
		typedNode, env, err = tc.inferLet(ctx, e)

	case *core.LetRec:
		typedNode, env, err = tc.inferLetRec(ctx, e)

	case *core.App:
		typedNode, env, err = tc.inferApp(ctx, e)

	case *core.If:
		typedNode, env, err = tc.inferIf(ctx, e)

	case *core.BinOp:
		typedNode, env, err = tc.inferBinOp(ctx, e)

	case *core.UnOp:
		typedNode, env, err = tc.inferUnOp(ctx, e)

	case *core.Record:
		typedNode, env, err = tc.inferRecord(ctx, e)

	case *core.RecordAccess:
		typedNode, env, err = tc.inferRecordAccess(ctx, e)

	case *core.RecordUpdate:
		typedNode, env, err = tc.inferRecordUpdate(ctx, e)

	case *core.List:
		typedNode, env, err = tc.inferList(ctx, e)

	case *core.Array:
		typedNode, env, err = tc.inferArray(ctx, e)

	case *core.Tuple:
		typedNode, env, err = tc.inferTuple(ctx, e)

	case *core.Match:
		typedNode, env, err = tc.inferMatch(ctx, e)

	case *core.Intrinsic:
		typedNode, env, err = tc.inferIntrinsic(ctx, e)

	default:
		return nil, ctx.env, fmt.Errorf("type inference not implemented for %T", expr)
	}

	// If inference succeeded, store the type in CoreTI for operator lowering
	if err == nil && typedNode != nil && expr != nil {
		// Get the inferred type from the typed node
		if inferredType, ok := typedNode.GetType().(Type); ok {
			// Store mapping: Core NodeID → Type (principal type after inference)
			tc.CoreTI.Set(expr.ID(), inferredType)
		}
	}

	return typedNode, env, err
}

// formatErrors formats all collected errors
func (tc *CoreTypeChecker) formatErrors() error {
	if len(tc.errors) == 0 {
		return nil
	}

	// Format errors with diagnostics
	var messages []string
	for _, err := range tc.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Errorf("Type checking failed:\n%s", strings.Join(messages, "\n"))
}

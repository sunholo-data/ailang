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
	}
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

// InferWithConstraints infers type with constraints for a Core expression
// Returns: typed expression, updated env, qualified type, constraints, error
func (tc *CoreTypeChecker) InferWithConstraints(expr core.CoreExpr, env *TypeEnv) (typedast.TypedNode, *TypeEnv, Type, []Constraint, error) {
	// Create inference context
	ctx := &InferenceContext{
		env:                  env,
		unifier:              NewUnifier(),
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
	switch e := expr.(type) {
	case *core.Lit:
		return tc.inferLit(ctx, e)

	case *core.Var:
		return tc.inferVar(ctx, e)

	case *core.VarGlobal:
		return tc.inferVarGlobal(ctx, e)

	case *core.Lambda:
		return tc.inferLambda(ctx, e)

	case *core.Let:
		return tc.inferLet(ctx, e)

	case *core.LetRec:
		return tc.inferLetRec(ctx, e)

	case *core.App:
		return tc.inferApp(ctx, e)

	case *core.If:
		return tc.inferIf(ctx, e)

	case *core.BinOp:
		return tc.inferBinOp(ctx, e)

	case *core.UnOp:
		return tc.inferUnOp(ctx, e)

	case *core.Record:
		return tc.inferRecord(ctx, e)

	case *core.RecordAccess:
		return tc.inferRecordAccess(ctx, e)

	case *core.List:
		return tc.inferList(ctx, e)

	case *core.Tuple:
		return tc.inferTuple(ctx, e)

	case *core.Match:
		return tc.inferMatch(ctx, e)

	case *core.Intrinsic:
		return tc.inferIntrinsic(ctx, e)

	default:
		return nil, ctx.env, fmt.Errorf("type inference not implemented for %T", expr)
	}
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

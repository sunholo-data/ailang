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
	instanceEnv := NewInstanceEnv()
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

	// Convert ClassConstraints to Constraints
	constraints := make([]Constraint, len(ctx.qualifiedConstraints))
	for i, cc := range ctx.qualifiedConstraints {
		constraints[i] = Constraint{
			Class: cc.Class,
			Type:  cc.Type,
		}
	}

	// Now resolve constraints and apply defaulting
	tc.resolveConstraints(ctx, typedNode)

	// Apply defaulting to the final type if it's still a type variable
	if _, ok := finalType.(*TVar2); ok {
		// Check if we can default this based on constraints
		for _, c := range constraints {
			if c.Class == "Num" {
				// Default to int
				finalType = &TCon{Name: "int"}
				break
			} else if c.Class == "Fractional" {
				// Default to float
				finalType = &TCon{Name: "float"}
				break
			}
		}
	}

	// Return with updated env, constraints and defaulted type
	return typedNode, updatedEnv, finalType, constraints, nil
}

// resolveConstraints resolves class constraints and applies defaulting
func (tc *CoreTypeChecker) resolveConstraints(ctx *InferenceContext, node typedast.TypedNode) {
	// For each class constraint, resolve it
	for _, cc := range ctx.qualifiedConstraints {
		// Determine the method name based on the operator
		methodName := ""
		switch cc.Class {
		case "Num":
			// Check which operator triggered this constraint
			methodName = "add" // Default to add for now
		case "Eq":
			methodName = "eq"
		case "Ord":
			methodName = "lt"
		}

		// Apply defaulting if the type is still a variable
		resolvedType := cc.Type
		switch resolvedType.(type) {
		case *TVar, *TVar2:
			// For Ord/Eq, try to default to Int
			// For Num, use the configured default
			var defaultType Type
			switch cc.Class {
			case "Num":
				defaultType = tc.instanceEnv.GetDefault("Num")
			case "Fractional":
				defaultType = tc.instanceEnv.GetDefault("Fractional")
			case "Ord", "Eq":
				// Default Ord/Eq to Int when ambiguous
				defaultType = &TCon{Name: "int"}
			}

			if defaultType != nil {
				resolvedType = defaultType
			}
		}

		// Store resolved constraint - ensure we have a concrete type
		var finalType Type = resolvedType

		// If we still have a type variable after defaulting attempt, that's an error
		switch resolvedType.(type) {
		case *TVar, *TVar2:
			// This shouldn't happen - defaulting should have resolved it
			// Type variable not resolved, using fallback
			// Use a fallback concrete type based on class
			switch cc.Class {
			case "Num":
				finalType = &TCon{Name: "int"}
			case "Fractional":
				finalType = &TCon{Name: "float"}
			case "Ord", "Eq":
				finalType = &TCon{Name: "int"}
			default:
				finalType = &TCon{Name: "int"} // Safe fallback
			}
		}

		// Now normalize the concrete type
		normalizedTypeName := NormalizeTypeName(finalType)
		normalizedType := &TCon{Name: normalizedTypeName}

		// Registered constraint resolution
		tc.resolvedConstraints[cc.NodeID] = &ResolvedConstraint{
			ClassName: cc.Class,
			Type:      normalizedType, // Normalized type for dictionary consistency
			Method:    methodName,
		}
	}
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

// inferLit infers type of literal
func (tc *CoreTypeChecker) inferLit(ctx *InferenceContext, lit *core.Lit) (*typedast.TypedLit, *TypeEnv, error) {
	var typ Type
	switch lit.Kind {
	case core.IntLit:
		// For integer literals, create a type variable with Num constraint
		// This allows defaulting to kick in later
		tv := ctx.freshType(Star)
		ctx.addConstraint(ClassConstraint{
			Class:  "Num",
			Type:   tv,
			Path:   []string{fmt.Sprintf("literal at %v", lit.Span())},
			NodeID: lit.ID(),
		})
		typ = tv
	case core.FloatLit:
		// For float literals, create a type variable with Fractional constraint
		tv := ctx.freshType(Star)
		ctx.addConstraint(ClassConstraint{
			Class:  "Fractional",
			Type:   tv,
			Path:   []string{fmt.Sprintf("literal at %v", lit.Span())},
			NodeID: lit.ID(),
		})
		typ = tv
	case core.StringLit:
		typ = TString
	case core.BoolLit:
		typ = TBool
	case core.UnitLit:
		typ = TUnit
	default:
		return nil, ctx.env, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}

	return &typedast.TypedLit{
		TypedExpr: typedast.TypedExpr{
			NodeID:    lit.ID(),
			Span:      lit.Span(),
			Type:      typ,
			EffectRow: EmptyEffectRow(),
			Core:      lit,
		},
		Kind:  lit.Kind,
		Value: lit.Value,
	}, ctx.env, nil
}

// inferVar infers type of variable
func (tc *CoreTypeChecker) inferVar(ctx *InferenceContext, v *core.Var) (*typedast.TypedVar, *TypeEnv, error) {
	typ, err := ctx.env.Lookup(v.Name)
	if err != nil {
		return nil, ctx.env, fmt.Errorf("undefined variable: %s at %s", v.Name, v.Span())
	}

	// Instantiate if it's a scheme
	var monotype Type
	if scheme, ok := typ.(*Scheme); ok {
		// Track fresh variables before instantiation
		var freshVars []string
		if tc.trackInstantiations {
			// Capture fresh type variables that will be generated
			for range scheme.TypeVars {
				freshVars = append(freshVars, fmt.Sprintf("t%d", tc.varCounter))
				tc.varCounter++
			}
		}

		monotype = scheme.Instantiate(ctx.freshType)

		// Record instantiation after it happens
		if tc.trackInstantiations {
			tc.instantiations = append(tc.instantiations, Instantiation{
				Location:     v.Span().String(),
				VarName:      v.Name,
				FreshVars:    freshVars,
				Instantiated: monotype,
			})
		}
	} else if t, ok := typ.(Type); ok {
		monotype = t
	} else {
		return nil, ctx.env, fmt.Errorf("invalid type in environment: %T", typ)
	}

	return &typedast.TypedVar{
		TypedExpr: typedast.TypedExpr{
			NodeID:    v.ID(),
			Span:      v.Span(),
			Type:      monotype,
			EffectRow: EmptyEffectRow(),
			Core:      v,
		},
		Name: v.Name,
	}, ctx.env, nil
}

// inferVarGlobal infers type of global variable reference
func (tc *CoreTypeChecker) inferVarGlobal(ctx *InferenceContext, v *core.VarGlobal) (*typedast.TypedVar, *TypeEnv, error) {
	// Look up the type in the global types
	key := fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name)
	scheme, ok := tc.globalTypes[key]
	if !ok {
		return nil, ctx.env, fmt.Errorf("undefined global variable: %s from %s", v.Ref.Name, v.Ref.Module)
	}

	// Track fresh variables before instantiation
	var freshVars []string
	if tc.trackInstantiations {
		// Capture fresh type variables that will be generated
		for range scheme.TypeVars {
			freshVars = append(freshVars, fmt.Sprintf("t%d", tc.varCounter))
			tc.varCounter++
		}
	}

	// Instantiate the scheme
	monotype := scheme.Instantiate(ctx.freshType)

	// Record instantiation after it happens
	if tc.trackInstantiations {
		tc.instantiations = append(tc.instantiations, Instantiation{
			Location:     v.Span().String(),
			VarName:      fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name),
			FreshVars:    freshVars,
			Instantiated: monotype,
		})
	}

	return &typedast.TypedVar{
		TypedExpr: typedast.TypedExpr{
			NodeID:    v.ID(),
			Span:      v.Span(),
			Type:      monotype,
			EffectRow: EmptyEffectRow(),
			Core:      v,
		},
		Name: fmt.Sprintf("%s.%s", v.Ref.Module, v.Ref.Name),
	}, ctx.env, nil
}

// inferLambda infers type of lambda with linear capture analysis
func (tc *CoreTypeChecker) inferLambda(ctx *InferenceContext, lam *core.Lambda) (*typedast.TypedLambda, *TypeEnv, error) {
	// Fresh type variables for parameters
	paramTypes := make([]Type, len(lam.Params))
	newEnv := ctx.env

	for i, param := range lam.Params {
		paramType := ctx.freshTypeVar()
		paramTypes[i] = paramType
		newEnv = newEnv.Extend(param, paramType)
	}

	// Save old env and use new one for body
	oldEnv := ctx.env
	ctx.env = newEnv

	// Infer body type
	bodyNode, _, err := tc.inferCore(ctx, lam.Body)
	if err != nil {
		return nil, oldEnv, err
	}

	// Check for linear capture violations
	captured := tc.findCapturedVars(lam, oldEnv)
	for _, cap := range captured {
		if tc.isLinearCapability(cap) {
			return nil, oldEnv, fmt.Errorf("lambda at %s captures linear capability %s; pass it as a parameter instead",
				lam.Span(), cap)
		}
	}

	// Restore environment
	ctx.env = oldEnv

	// Lambda type with effect annotation
	var funcEffectRow *Row

	// Check for explicit effect annotation from AST
	if effectNames := tc.effectAnnots[lam.ID()]; len(effectNames) > 0 {
		// Use explicit annotation
		var err error
		funcEffectRow, err = ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, oldEnv, fmt.Errorf("invalid effect annotation at %s: %w", lam.Span(), err)
		}
	} else {
		// Infer from body (existing behavior)
		if effRow := bodyNode.GetEffectRow(); effRow != nil {
			funcEffectRow = effRow.(*Row)
		}
	}

	funcType := &TFunc2{
		Params:    paramTypes,
		EffectRow: funcEffectRow,
		Return:    bodyNode.GetType().(Type),
	}

	return &typedast.TypedLambda{
		TypedExpr: typedast.TypedExpr{
			NodeID:    lam.ID(),
			Span:      lam.Span(),
			Type:      funcType,
			EffectRow: interface{}(EmptyEffectRow()), // Lambda itself is pure
			Core:      lam,
		},
		Params:     lam.Params,
		ParamTypes: toInterfaceSlice(paramTypes),
		Body:       bodyNode,
	}, ctx.env, nil
}

// inferLet infers type of let binding with generalization
func (tc *CoreTypeChecker) inferLet(ctx *InferenceContext, let *core.Let) (*typedast.TypedLet, *TypeEnv, error) {
	// Infer value type
	valueNode, _, err := tc.inferCore(ctx, let.Value)
	if err != nil {
		return nil, ctx.env, err
	}

	// CRITICAL: Apply defaulting BEFORE generalization
	// This is a generalization boundary where defaulting must happen
	valueType := getType(valueNode)
	valueEffects := getEffectRow(valueNode)

	// Get unsolved constraints from current context
	_, unsolvedConstraints, err := ctx.SolveConstraints()
	if err != nil {
		return nil, ctx.env, err
	}

	// Apply defaulting at this generalization boundary
	defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguities(valueType, unsolvedConstraints)
	if err != nil {
		return nil, ctx.env, fmt.Errorf("defaulting failed for let binding %s: %w", let.Name, err)
	}

	// Apply defaulting substitution everywhere if any defaults were applied
	if len(defaultingSub) > 0 {
		defaultedType, defaultedConstraints, valueNode, _ = tc.ApplySubstEverywhere(
			defaultingSub, defaultedType, defaultedConstraints, valueNode, nil, let.Name)
	}

	// Generalize if value is syntactic value (value restriction)
	var binding interface{}
	if isCoreValue(let.Value) {
		// After defaulting, only non-ground constraints should remain for generalization
		nonGroundConstraints := []ClassConstraint{}
		for _, c := range defaultedConstraints {
			if !isGround(c.Type) {
				nonGroundConstraints = append(nonGroundConstraints, c)
			}
		}
		binding = tc.generalizeWithConstraints(defaultedType, valueEffects, nonGroundConstraints)
	} else {
		binding = defaultedType
	}

	// Extend environment
	var newEnv *TypeEnv
	var scheme *Scheme
	if s, ok := binding.(*Scheme); ok {
		scheme = s
		newEnv = ctx.env.ExtendScheme(let.Name, s)
	} else {
		newEnv = ctx.env.Extend(let.Name, binding.(Type))
		// Create trivial scheme for consistency
		scheme = &Scheme{Type: binding.(Type)}
	}

	// Save env and infer body
	oldEnv := ctx.env
	ctx.env = newEnv
	bodyNode, finalEnv, err := tc.inferCore(ctx, let.Body)
	if err != nil {
		return nil, oldEnv, err
	}
	ctx.env = oldEnv

	return &typedast.TypedLet{
		TypedExpr: typedast.TypedExpr{
			NodeID:    let.ID(),
			Span:      let.Span(),
			Type:      bodyNode.GetType(),
			EffectRow: combineEffects(valueEffects, getEffectRow(bodyNode)),
			Core:      let,
		},
		Name:   let.Name,
		Scheme: scheme, // Generalized type at binding site
		Value:  valueNode,
		Body:   bodyNode,
	}, finalEnv, nil
}

// inferLetRec infers type of recursive bindings
func (tc *CoreTypeChecker) inferLetRec(ctx *InferenceContext, letrec *core.LetRec) (*typedast.TypedLetRec, *TypeEnv, error) {
	// Create fresh type variables for all bindings
	bindingTypes := make(map[string]Type)
	for _, binding := range letrec.Bindings {
		bindingTypes[binding.Name] = ctx.freshTypeVar()
	}

	// Extend environment with all bindings
	newEnv := ctx.env
	for name, typ := range bindingTypes {
		newEnv = newEnv.Extend(name, typ)
	}

	// Save and update environment
	oldEnv := ctx.env
	ctx.env = newEnv

	// Infer types of all values and collect constraints
	var allValueNodes []typedast.TypedNode
	var allValueTypes []Type
	for _, binding := range letrec.Bindings {
		valueNode, _, err := tc.inferCore(ctx, binding.Value)
		if err != nil {
			return nil, oldEnv, err
		}

		allValueNodes = append(allValueNodes, valueNode)
		allValueTypes = append(allValueTypes, getType(valueNode))

		// Unify with expected type
		ctx.addConstraint(TypeEq{
			Left:  bindingTypes[binding.Name],
			Right: getType(valueNode),
			Path:  []string{binding.Name},
		})
	}

	// CRITICAL: Apply defaulting ONCE for the entire SCC after solving mutual block
	_, unsolvedConstraints, err := ctx.SolveConstraints()
	if err != nil {
		return nil, oldEnv, err
	}

	// Apply defaulting to the entire mutual block (once per SCC)
	for i, binding := range letrec.Bindings {
		valueType := allValueTypes[i]
		valueNode := allValueNodes[i]

		// Apply defaulting at this generalization boundary
		defaultingSub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguities(valueType, unsolvedConstraints)
		if err != nil {
			return nil, oldEnv, fmt.Errorf("defaulting failed for letrec binding %s: %w", binding.Name, err)
		}

		// Apply defaulting substitution everywhere if any defaults were applied
		if len(defaultingSub) > 0 {
			defaultedType, _, valueNode, _ = tc.ApplySubstEverywhere(
				defaultingSub, defaultedType, defaultedConstraints, valueNode, nil, binding.Name)

			// Update the stored values
			allValueTypes[i] = defaultedType
			allValueNodes[i] = valueNode
		}
	}

	// Now generalize each binding after defaulting
	typedBindings := make([]typedast.TypedRecBinding, len(letrec.Bindings))
	for i, binding := range letrec.Bindings {
		valueType := allValueTypes[i]
		valueNode := allValueNodes[i]

		// Get remaining non-ground constraints after defaulting
		_, remainingConstraints, err := ctx.SolveConstraints()
		if err != nil {
			return nil, oldEnv, err
		}

		nonGroundConstraints := []ClassConstraint{}
		for _, c := range remainingConstraints {
			if !isGround(c.Type) {
				nonGroundConstraints = append(nonGroundConstraints, c)
			}
		}

		// Generalize for recursion
		scheme := tc.generalizeWithConstraints(valueType, getEffectRow(valueNode), nonGroundConstraints)

		typedBindings[i] = typedast.TypedRecBinding{
			Name:   binding.Name,
			Scheme: scheme,
			Value:  valueNode,
		}

		// Update environment with generalized type
		newEnv = newEnv.ExtendScheme(binding.Name, scheme)
	}

	// Update context environment for body
	ctx.env = newEnv

	// Infer body type
	bodyNode, finalEnv, err := tc.inferCore(ctx, letrec.Body)
	if err != nil {
		return nil, oldEnv, err
	}

	// Restore environment
	ctx.env = oldEnv

	// Combine effects from all bindings and body
	var allEffects []*Row
	for _, binding := range typedBindings {
		allEffects = append(allEffects, getEffectRow(binding.Value))
	}
	allEffects = append(allEffects, getEffectRow(bodyNode))

	return &typedast.TypedLetRec{
		TypedExpr: typedast.TypedExpr{
			NodeID:    letrec.ID(),
			Span:      letrec.Span(),
			Type:      bodyNode.GetType(),
			EffectRow: combineEffectList(allEffects),
			Core:      letrec,
		},
		Bindings: typedBindings,
		Body:     bodyNode,
	}, finalEnv, nil
}

// generalizeWithConstraints creates a type scheme with explicit constraints
func (tc *CoreTypeChecker) generalizeWithConstraints(typ Type, effects *Row, constraints []ClassConstraint) *Scheme {
	// Find free type variables in type but not in environment
	typeFreeVars := make(map[string]bool)
	collectFreeVars(typ, typeFreeVars)

	// For now, simplified generalization
	// In a full implementation, would check against environment free vars
	generalizedTypeVars := []string{}
	for v := range typeFreeVars {
		generalizedTypeVars = append(generalizedTypeVars, v)
	}

	// Convert class constraints to scheme constraints
	schemeConstraints := []Constraint{}
	for _, c := range constraints {
		schemeConstraints = append(schemeConstraints, Constraint{
			Class: c.Class,
			Type:  c.Type,
		})
	}

	return &Scheme{
		TypeVars:    generalizedTypeVars,
		RowVars:     []string{}, // Simplified for now
		Constraints: schemeConstraints,
		Type:        typ,
	}
}

// inferApp infers type of function application
func (tc *CoreTypeChecker) inferApp(ctx *InferenceContext, app *core.App) (*typedast.TypedApp, *TypeEnv, error) {
	// Infer function type
	funcNode, _, err := tc.inferCore(ctx, app.Func)
	if err != nil {
		return nil, ctx.env, err
	}

	// Infer argument types
	var argNodes []typedast.TypedNode
	var argTypes []Type
	var allEffects []*Row

	for _, arg := range app.Args {
		argNode, _, err := tc.inferCore(ctx, arg)
		if err != nil {
			return nil, ctx.env, err
		}
		argNodes = append(argNodes, argNode)
		argTypes = append(argTypes, getType(argNode))
		allEffects = append(allEffects, getEffectRow(argNode))
	}

	// Create result type variable
	resultType := ctx.freshTypeVar()
	effectRow := ctx.freshEffectRow()

	// Unify function type with expected type
	expectedFuncType := &TFunc2{
		Params:    argTypes,
		EffectRow: effectRow,
		Return:    resultType,
	}

	ctx.addConstraint(TypeEq{
		Left:  getType(funcNode),
		Right: expectedFuncType,
		Path:  []string{"function application at " + app.Span().String()},
	})

	// Combine effects
	allEffects = append(allEffects, getEffectRow(funcNode), effectRow)

	return &typedast.TypedApp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    app.ID(),
			Span:      app.Span(),
			Type:      resultType,
			EffectRow: combineEffectList(allEffects),
			Core:      app,
		},
		Func: funcNode,
		Args: argNodes,
	}, ctx.env, nil
}

// inferIf infers type of conditional
func (tc *CoreTypeChecker) inferIf(ctx *InferenceContext, ifExpr *core.If) (*typedast.TypedIf, *TypeEnv, error) {
	// Infer condition type
	condNode, _, err := tc.inferCore(ctx, ifExpr.Cond)
	if err != nil {
		return nil, ctx.env, err
	}

	// Condition must be bool
	ctx.addConstraint(TypeEq{
		Left:  getType(condNode),
		Right: TBool,
		Path:  []string{"if condition at " + ifExpr.Span().String()},
	})

	// Infer branch types
	thenNode, _, err := tc.inferCore(ctx, ifExpr.Then)
	if err != nil {
		return nil, ctx.env, err
	}

	elseNode, _, err := tc.inferCore(ctx, ifExpr.Else)
	if err != nil {
		return nil, ctx.env, err
	}

	// Branches must have same type
	ctx.addConstraint(TypeEq{
		Left:  getType(thenNode),
		Right: getType(elseNode),
		Path:  []string{"if branches at " + ifExpr.Span().String()},
	})

	// Combine effects from all parts
	effects := combineEffectList([]*Row{
		getEffectRow(condNode),
		getEffectRow(thenNode),
		getEffectRow(elseNode),
	})

	return &typedast.TypedIf{
		TypedExpr: typedast.TypedExpr{
			NodeID:    ifExpr.ID(),
			Span:      ifExpr.Span(),
			Type:      getType(thenNode),
			EffectRow: effects,
			Core:      ifExpr,
		},
		Cond: condNode,
		Then: thenNode,
		Else: elseNode,
	}, ctx.env, nil
}

// OperatorMethod returns the method name for an operator.
// Exported for use by the elaborator during dictionary-passing transformation.
// Binary operators map to their corresponding type class methods.
// Unary minus is handled as "neg" (negate) method in the Num class.
func OperatorMethod(op string, isUnary bool) string {
	// Handle unary operators
	if isUnary {
		switch op {
		case "-":
			return "neg" // Unary minus uses Num.neg method
		case "!":
			return "not" // Boolean not (if we have a Bool class)
		default:
			return ""
		}
	}

	// Binary operators
	switch op {
	case "+":
		return "add"
	case "-":
		return "sub"
	case "*":
		return "mul"
	case "/":
		return "div"
	case "==":
		return "eq"
	case "!=":
		return "neq"
	case "<":
		return "lt"
	case "<=":
		return "lte"
	case ">":
		return "gt"
	case ">=":
		return "gte"
	default:
		return ""
	}
}

// inferIntrinsic infers type of intrinsic operation
func (tc *CoreTypeChecker) inferIntrinsic(ctx *InferenceContext, intrinsic *core.Intrinsic) (*typedast.TypedBinOp, *TypeEnv, error) {
	// For binary intrinsics, delegate to inferBinOp logic
	if len(intrinsic.Args) == 2 {
		// Convert back to BinOp for type checking (temporary)
		opStr := map[core.IntrinsicOp]string{
			core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*", core.OpDiv: "/", core.OpMod: "%",
			core.OpEq: "==", core.OpNe: "!=", core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
			core.OpConcat: "++", core.OpAnd: "&&", core.OpOr: "||",
		}[intrinsic.Op]

		binop := &core.BinOp{
			CoreNode: intrinsic.CoreNode,
			Op:       opStr,
			Left:     intrinsic.Args[0],
			Right:    intrinsic.Args[1],
		}
		return tc.inferBinOp(ctx, binop)
	}

	// For unary intrinsics
	if len(intrinsic.Args) == 1 {
		opStr := map[core.IntrinsicOp]string{
			core.OpNot: "not", core.OpNeg: "-",
		}[intrinsic.Op]

		unop := &core.UnOp{
			CoreNode: intrinsic.CoreNode,
			Op:       opStr,
			Operand:  intrinsic.Args[0],
		}
		// We need to adapt the unary result
		unResult, env, err := tc.inferUnOp(ctx, unop)
		if err != nil {
			return nil, env, err
		}
		// Convert UnOp result to BinOp result (hack for now)
		return &typedast.TypedBinOp{
			TypedExpr: unResult.TypedExpr,
			Op:        opStr,
			Left:      unResult.Operand,
			Right:     &typedast.TypedLit{TypedExpr: typedast.TypedExpr{Type: TUnit}}, // dummy
		}, env, nil
	}

	return nil, ctx.env, fmt.Errorf("unexpected intrinsic arity: %d", len(intrinsic.Args))
}

// inferBinOp infers type of binary operation
func (tc *CoreTypeChecker) inferBinOp(ctx *InferenceContext, binop *core.BinOp) (*typedast.TypedBinOp, *TypeEnv, error) {
	// Infer operand types
	leftNode, _, err := tc.inferCore(ctx, binop.Left)
	if err != nil {
		return nil, ctx.env, err
	}

	rightNode, _, err := tc.inferCore(ctx, binop.Right)
	if err != nil {
		return nil, ctx.env, err
	}

	// Determine result type based on operator
	var resultType Type

	switch binop.Op {
	case "+", "-", "*", "/", "%":
		// Arithmetic operators - unify operand types first
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"arithmetic at " + binop.Span().String()},
		})

		// The result type is the same as the operand types
		resultType = getType(leftNode)

		// IMPORTANT: use the unified type to decide the most specific numeric class
		// This looks at constraints on the unified type, not individual nodes
		cls := tc.mostSpecificNumericClass(ctx, resultType)

		// Attach ONE class constraint to the unified type
		ctx.addConstraint(ClassConstraint{
			Class:  cls, // "Fractional" or "Num"
			Type:   resultType,
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(), // keep this for operator→method linking
		})

	case "++":
		// Concatenation: works for both strings and lists
		leftType := getType(leftNode)
		rightType := getType(rightNode)

		// DEBUG output (commented out - pollutes output)
		//fmt.Printf("DEBUG ++ operator: left=%T(%v), right=%T(%v)\n", leftType, leftType, rightType, rightType)

		// Check type patterns
		_, leftIsList := leftType.(*TList)
		_, rightIsList := rightType.(*TList)
		_, leftIsVar := leftType.(*TVar2)
		_, rightIsVar := rightType.(*TVar2)

		// Check if both are strings (TCon "String"/"string" or TString)
		leftIsString := false
		rightIsString := false

		if leftType == TString {
			leftIsString = true
		} else if leftCon, ok := leftType.(*TCon); ok && (leftCon.Name == "String" || leftCon.Name == "string") {
			leftIsString = true
		}

		if rightType == TString {
			rightIsString = true
		} else if rightCon, ok := rightType.(*TCon); ok && (rightCon.Name == "String" || rightCon.Name == "string") {
			rightIsString = true
		}

		// Decision tree:
		// 1. If at least one is a concrete list → list concat
		// 2. If at least one is a concrete string → string concat
		// 3. If both are type variables → list concat (more polymorphic)
		// 4. Otherwise → string concat (fallback)

		if leftIsList || rightIsList {
			// At least one is definitely a list → list concat
			elemType := ctx.freshTypeVar()

			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat right at " + binop.Span().String()},
			})

			resultType = &TList{Element: elemType}
		} else if leftIsString || rightIsString {
			// At least one is a concrete string → string concat
			// The type variable (if any) will be unified with String
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: TString,
				Path:  []string{"string concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: TString,
				Path:  []string{"string concat right at " + binop.Span().String()},
			})
			resultType = TString
		} else if leftIsVar && rightIsVar {
			// Both are type variables - default to list concat (more polymorphic)
			elemType := ctx.freshTypeVar()

			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat right at " + binop.Span().String()},
			})

			resultType = &TList{Element: elemType}
		} else {
			// Fallback: assume string concat
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: TString,
				Path:  []string{"string concat at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: TString,
				Path:  []string{"string concat at " + binop.Span().String()},
			})
			resultType = TString
		}

	case "<", ">", "<=", ">=":
		// Comparison operators - require Ord constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Ord",
			Type:   getType(leftNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(ClassConstraint{
			Class:  "Ord",
			Type:   getType(rightNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"comparison at " + binop.Span().String()},
		})
		resultType = TBool

	case "==", "!=":
		// Equality - require Eq constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Eq",
			Type:   getType(leftNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(ClassConstraint{
			Class:  "Eq",
			Type:   getType(rightNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"equality at " + binop.Span().String()},
		})
		resultType = TBool

	case "&&", "||":
		// Boolean operators
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(rightNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		resultType = TBool

	default:
		return nil, ctx.env, fmt.Errorf("unknown binary operator: %s", binop.Op)
	}

	// Combine effects
	effects := combineEffects(getEffectRow(leftNode), getEffectRow(rightNode))

	return &typedast.TypedBinOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    binop.ID(),
			Span:      binop.Span(),
			Type:      resultType,
			EffectRow: effects,
			Core:      binop,
		},
		Op:    binop.Op,
		Left:  leftNode,
		Right: rightNode,
	}, ctx.env, nil
}

// inferUnOp infers type of unary operation
func (tc *CoreTypeChecker) inferUnOp(ctx *InferenceContext, unop *core.UnOp) (*typedast.TypedUnOp, *TypeEnv, error) {
	// Infer operand type
	operandNode, _, err := tc.inferCore(ctx, unop.Operand)
	if err != nil {
		return nil, ctx.env, err
	}

	var resultType Type

	switch unop.Op {
	case "-":
		// Negation - requires Num constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Num",
			Type:   getType(operandNode),
			Path:   []string{unop.Span().String()},
			NodeID: unop.ID(),
		})
		resultType = getType(operandNode)

	case "not":
		// Boolean negation
		ctx.addConstraint(TypeEq{
			Left:  getType(operandNode),
			Right: TBool,
			Path:  []string{"not at " + unop.Span().String()},
		})
		resultType = TBool

	default:
		return nil, ctx.env, fmt.Errorf("unknown unary operator: %s", unop.Op)
	}

	return &typedast.TypedUnOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    unop.ID(),
			Span:      unop.Span(),
			Type:      resultType,
			EffectRow: getEffectRow(operandNode),
			Core:      unop,
		},
		Op:      unop.Op,
		Operand: operandNode,
	}, ctx.env, nil
}

// inferRecord infers type of record construction
func (tc *CoreTypeChecker) inferRecord(ctx *InferenceContext, rec *core.Record) (*typedast.TypedRecord, *TypeEnv, error) {
	fields := make(map[string]typedast.TypedNode)
	fieldTypes := make(map[string]Type)
	var allEffects []*Row

	for name, value := range rec.Fields {
		valueNode, _, err := tc.inferCore(ctx, value)
		if err != nil {
			return nil, ctx.env, err
		}
		fields[name] = valueNode
		fieldTypes[name] = getType(valueNode)
		allEffects = append(allEffects, getEffectRow(valueNode))
	}

	// Create record type - use TRecord2 if flag is set
	var recordType Type
	if tc.useRecordsV2 {
		recordType = &TRecord2{
			Row: &Row{
				Kind:   RecordRow,
				Labels: fieldTypes,
				Tail:   nil, // Closed record literal
			},
		}
	} else {
		recordType = &TRecord{
			Fields: fieldTypes,
			Row: &Row{
				Kind:   RecordRow,
				Labels: fieldTypes,
				Tail:   nil, // Closed record for now
			},
		}
	}

	return &typedast.TypedRecord{
		TypedExpr: typedast.TypedExpr{
			NodeID:    rec.ID(),
			Span:      rec.Span(),
			Type:      recordType,
			EffectRow: combineEffectList(allEffects),
			Core:      rec,
		},
		Fields: fields,
	}, ctx.env, nil
}

// inferRecordAccess infers type of field access
func (tc *CoreTypeChecker) inferRecordAccess(ctx *InferenceContext, acc *core.RecordAccess) (*typedast.TypedRecordAccess, *TypeEnv, error) {
	// Infer record type
	recordNode, _, err := tc.inferCore(ctx, acc.Record)
	if err != nil {
		return nil, ctx.env, err
	}

	// Create fresh type variable for field
	fieldType := ctx.freshTypeVar()
	rowVar := &RowVar{Name: "r", Kind: RecordRow}

	// Expected record type with field - use TRecordOpen for subsumption
	// This allows {id:int} to unify with {id:int, email:string}
	expectedRecord := &TRecordOpen{
		Fields: map[string]Type{acc.Field: fieldType},
		Row:    rowVar, // Mark as open for unknown fields
	}

	ctx.addConstraint(TypeEq{
		Left:  getType(recordNode),
		Right: expectedRecord,
		Path:  []string{"field access at " + acc.Span().String()},
	})

	return &typedast.TypedRecordAccess{
		TypedExpr: typedast.TypedExpr{
			NodeID:    acc.ID(),
			Span:      acc.Span(),
			Type:      fieldType,
			EffectRow: getEffectRow(recordNode),
			Core:      acc,
		},
		Record: recordNode,
		Field:  acc.Field,
	}, ctx.env, nil
}

// inferList infers type of list construction
func (tc *CoreTypeChecker) inferList(ctx *InferenceContext, list *core.List) (*typedast.TypedList, *TypeEnv, error) {
	if len(list.Elements) == 0 {
		// Empty list - polymorphic
		elemType := ctx.freshTypeVar()
		return &typedast.TypedList{
			TypedExpr: typedast.TypedExpr{
				NodeID:    list.ID(),
				Span:      list.Span(),
				Type:      &TList{Element: elemType},
				EffectRow: EmptyEffectRow(),
				Core:      list,
			},
			Elements: nil,
		}, ctx.env, nil
	}

	// Non-empty list - all elements must have same type
	var elements []typedast.TypedNode
	var allEffects []*Row

	firstElem, _, err := tc.inferCore(ctx, list.Elements[0])
	if err != nil {
		return nil, ctx.env, err
	}
	elements = append(elements, firstElem)
	allEffects = append(allEffects, getEffectRow(firstElem))
	elemType := getType(firstElem)

	for i := 1; i < len(list.Elements); i++ {
		elemNode, _, err := tc.inferCore(ctx, list.Elements[i])
		if err != nil {
			return nil, ctx.env, err
		}
		elements = append(elements, elemNode)
		allEffects = append(allEffects, getEffectRow(elemNode))

		// All elements must have same type
		ctx.addConstraint(TypeEq{
			Left:  getType(elemNode),
			Right: elemType,
			Path:  []string{fmt.Sprintf("list element %d at %s", i, list.Span())},
		})
	}

	return &typedast.TypedList{
		TypedExpr: typedast.TypedExpr{
			NodeID:    list.ID(),
			Span:      list.Span(),
			Type:      &TList{Element: elemType},
			EffectRow: combineEffectList(allEffects),
			Core:      list,
		},
		Elements: elements,
	}, ctx.env, nil
}

// inferTuple infers type of tuple construction
func (tc *CoreTypeChecker) inferTuple(ctx *InferenceContext, tuple *core.Tuple) (*typedast.TypedTuple, *TypeEnv, error) {
	// Infer types for all elements
	var elements []typedast.TypedNode
	var elemTypes []Type
	var allEffects []*Row

	for _, elem := range tuple.Elements {
		elemNode, _, err := tc.inferCore(ctx, elem)
		if err != nil {
			return nil, ctx.env, err
		}
		elements = append(elements, elemNode)
		elemTypes = append(elemTypes, getType(elemNode))
		allEffects = append(allEffects, getEffectRow(elemNode))
	}

	return &typedast.TypedTuple{
		TypedExpr: typedast.TypedExpr{
			NodeID:    tuple.ID(),
			Span:      tuple.Span(),
			Type:      &TTuple{Elements: elemTypes},
			EffectRow: combineEffectList(allEffects),
			Core:      tuple,
		},
		Elements: elements,
	}, ctx.env, nil
}

// inferMatch infers type of pattern matching
func (tc *CoreTypeChecker) inferMatch(ctx *InferenceContext, match *core.Match) (*typedast.TypedMatch, *TypeEnv, error) {
	// Infer scrutinee type
	scrutineeNode, _, err := tc.inferCore(ctx, match.Scrutinee)
	if err != nil {
		return nil, ctx.env, err
	}

	// Check exhaustiveness (simplified for now)
	// TODO: Implement full exhaustiveness checking
	exhaustive := match.Exhaustive

	// Infer types of all arms
	var arms []typedast.TypedMatchArm
	var resultType Type
	var allEffects []*Row

	for i, arm := range match.Arms {
		// Type check pattern and get bindings
		patternBindings, typedPattern, err := tc.checkPattern(arm.Pattern, getType(scrutineeNode), ctx)
		if err != nil {
			return nil, ctx.env, err
		}

		// Extend environment with pattern bindings
		armEnv := ctx.env
		for name, typ := range patternBindings {
			armEnv = armEnv.Extend(name, typ)
		}

		// Save and update environment
		oldEnv := ctx.env
		ctx.env = armEnv

		// Check guard if present
		var guardNode typedast.TypedNode
		if arm.Guard != nil {
			guardNode, _, err = tc.inferCore(ctx, arm.Guard)
			if err != nil {
				return nil, oldEnv, err
			}
			// Guard must be boolean
			ctx.addConstraint(TypeEq{
				Left:  getType(guardNode),
				Right: TBool,
				Path:  []string{fmt.Sprintf("match guard %d at %s", i, match.Span())},
			})
			allEffects = append(allEffects, getEffectRow(guardNode))
		}

		// Type check body
		bodyNode, _, err := tc.inferCore(ctx, arm.Body)
		if err != nil {
			return nil, oldEnv, err
		}
		allEffects = append(allEffects, getEffectRow(bodyNode))

		// Restore environment
		ctx.env = oldEnv

		// All arms must have same result type
		if i == 0 {
			resultType = getType(bodyNode)
		} else {
			ctx.addConstraint(TypeEq{
				Left:  getType(bodyNode),
				Right: resultType,
				Path:  []string{fmt.Sprintf("match arm %d at %s", i, match.Span())},
			})
		}

		arms = append(arms, typedast.TypedMatchArm{
			Pattern: typedPattern,
			Guard:   guardNode,
			Body:    bodyNode,
		})
	}

	// Add scrutinee effects
	allEffects = append(allEffects, getEffectRow(scrutineeNode))

	return &typedast.TypedMatch{
		TypedExpr: typedast.TypedExpr{
			NodeID:    match.ID(),
			Span:      match.Span(),
			Type:      resultType,
			EffectRow: combineEffectList(allEffects),
			Core:      match,
		},
		Scrutinee:  scrutineeNode,
		Arms:       arms,
		Exhaustive: exhaustive,
	}, ctx.env, nil
}

// checkPattern type checks a pattern and returns bindings
func (tc *CoreTypeChecker) checkPattern(pat core.CorePattern, scrutType Type, ctx *InferenceContext) (map[string]Type, typedast.TypedPattern, error) {
	switch p := pat.(type) {
	case *core.VarPattern:
		// Variable pattern binds to scrutinee type
		return map[string]Type{p.Name: scrutType},
			typedast.TypedVarPattern{Name: p.Name, Type: scrutType}, nil

	case *core.LitPattern:
		// Literal pattern - scrutinee must match literal type
		var litType Type
		switch p.Value.(type) {
		case int, int64:
			litType = TInt
		case float32, float64:
			litType = TFloat
		case string:
			litType = TString
		case bool:
			litType = TBool
		default:
			return nil, nil, fmt.Errorf("unknown literal type in pattern: %T", p.Value)
		}

		ctx.addConstraint(TypeEq{
			Left:  scrutType,
			Right: litType,
			Path:  []string{"literal pattern"},
		})

		return nil, typedast.TypedLitPattern{Value: p.Value}, nil

	case *core.WildcardPattern:
		// Wildcard matches anything, binds nothing
		return nil, typedast.TypedWildcardPattern{}, nil

	case *core.ConstructorPattern:
		// Constructor pattern - need to lookup constructor scheme
		// TODO: This needs access to the module interface to get constructor schemes
		// For now, we'll do basic checking without constructor validation

		// Recursively check nested patterns
		// We need to know the field types of this constructor
		// For now, create fresh type variables for each field
		bindings := make(map[string]Type)
		typedArgs := make([]typedast.TypedPattern, len(p.Args))

		for i, argPat := range p.Args {
			// Create fresh type variable for each argument
			argType := ctx.freshTypeVar()
			argBindings, typedArg, err := tc.checkPattern(argPat, argType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range argBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedArgs[i] = typedArg
		}

		return bindings, typedast.TypedConstructorPattern{
			Name: p.Name,
			Args: typedArgs,
		}, nil

	case *core.TuplePattern:
		// Tuple pattern - scrutinee must be tuple type
		// Extract element types from scrutinee
		var elemTypes []Type

		// Try to extract tuple type from scrutinee
		if tupleTy, ok := scrutType.(*TTuple); ok {
			elemTypes = tupleTy.Elements
		} else {
			// Create fresh type variables and add constraint
			elemTypes = make([]Type, len(p.Elements))
			for i := range p.Elements {
				elemTypes[i] = ctx.freshTypeVar()
			}
			ctx.addConstraint(TypeEq{
				Left:  scrutType,
				Right: &TTuple{Elements: elemTypes},
				Path:  []string{"tuple pattern"},
			})
		}

		// Check that arity matches
		if len(p.Elements) != len(elemTypes) {
			return nil, nil, fmt.Errorf("tuple pattern has %d elements but scrutinee has %d",
				len(p.Elements), len(elemTypes))
		}

		// Recursively check each element pattern
		bindings := make(map[string]Type)
		typedElems := make([]typedast.TypedPattern, len(p.Elements))

		for i, elemPat := range p.Elements {
			elemBindings, typedElem, err := tc.checkPattern(elemPat, elemTypes[i], ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range elemBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedElems[i] = typedElem
		}

		return bindings, typedast.TypedTuplePattern{
			Elements: typedElems,
		}, nil

	case *core.ListPattern:
		// List pattern - scrutinee must be list type
		// Extract element type from scrutinee list
		var elemType Type

		// Try to extract list type from scrutinee
		if listTy, ok := scrutType.(*TList); ok {
			elemType = listTy.Element
		} else {
			// Create fresh type variable for elements
			elemType = ctx.freshTypeVar()
			ctx.addConstraint(TypeEq{
				Left:  scrutType,
				Right: &TList{Element: elemType},
				Path:  []string{"list pattern"},
			})
		}

		// Recursively check each element pattern
		bindings := make(map[string]Type)
		typedElems := make([]typedast.TypedPattern, len(p.Elements))

		for i, elemPat := range p.Elements {
			elemBindings, typedElem, err := tc.checkPattern(elemPat, elemType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge bindings
			for name, typ := range elemBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedElems[i] = typedElem
		}

		// Type check tail pattern if present
		var typedTail *typedast.TypedPattern
		if p.Tail != nil {
			// Tail must have list type (same as scrutinee)
			tailBindings, tail, err := tc.checkPattern(*p.Tail, scrutType, ctx)
			if err != nil {
				return nil, nil, err
			}
			// Merge tail bindings
			for name, typ := range tailBindings {
				if existing, ok := bindings[name]; ok {
					// Variable bound multiple times - must unify
					ctx.addConstraint(TypeEq{
						Left:  existing,
						Right: typ,
						Path:  []string{fmt.Sprintf("pattern variable %s", name)},
					})
				} else {
					bindings[name] = typ
				}
			}
			typedTail = &tail
		}

		return bindings, typedast.TypedListPattern{
			Elements: typedElems,
			Tail:     typedTail,
		}, nil

	default:
		return nil, nil, fmt.Errorf("pattern type checking not implemented for %T", pat)
	}
}

// Helper functions

// toInterfaceSlice converts []Type to []interface{}
func toInterfaceSlice(types []Type) []interface{} {
	result := make([]interface{}, len(types))
	for i, t := range types {
		result[i] = t
	}
	return result
}

// getType safely extracts Type from interface{}
func getType(node typedast.TypedNode) Type {
	if t := node.GetType(); t != nil {
		return t.(Type)
	}
	return nil
}

// getEffectRow safely extracts Row from interface{}
func getEffectRow(node typedast.TypedNode) *Row {
	if r := node.GetEffectRow(); r != nil {
		return r.(*Row)
	}
	return EmptyEffectRow()
}

// isCoreValue checks if expression is a syntactic value (for value restriction)
func isCoreValue(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Lit, *core.Lambda:
		return true
	case *core.Var:
		// Variables are values if bound to values
		return true
	case *core.Record:
		// Record is value if all fields are values
		for _, field := range e.Fields {
			if !isCoreValue(field) {
				return false
			}
		}
		return true
	case *core.List:
		// List is value if all elements are values
		for _, elem := range e.Elements {
			if !isCoreValue(elem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// findCapturedVars finds variables captured by a lambda
func (tc *CoreTypeChecker) findCapturedVars(lam *core.Lambda, outerEnv *TypeEnv) []string {
	// This is simplified - full implementation would traverse body
	// and check which variables are from outer scope
	var captured []string
	// TODO: Implement proper free variable analysis
	return captured
}

// isLinearCapability checks if a variable is a linear capability
func (tc *CoreTypeChecker) isLinearCapability(name string) bool {
	// Check if name is a known capability like FS, Net, etc
	capabilities := []string{"FS", "Net", "IO", "Async"}
	for _, cap := range capabilities {
		if name == cap {
			return true
		}
	}
	return false
}

// combineEffects combines two effect rows
func combineEffects(e1, e2 *Row) *Row {
	if e1 == nil || (len(e1.Labels) == 0 && e1.Tail == nil) {
		return e2
	}
	if e2 == nil || (len(e2.Labels) == 0 && e2.Tail == nil) {
		return e1
	}

	// Combine labels
	combined := make(map[string]Type)
	for k, v := range e1.Labels {
		combined[k] = v
	}
	for k, v := range e2.Labels {
		combined[k] = v
	}

	// For now, ignore tail variables in combination
	// Full implementation would handle row unification
	return &Row{
		Kind:   EffectRow,
		Labels: combined,
		Tail:   nil,
	}
}

// combineEffectList combines multiple effect rows
func combineEffectList(effects []*Row) *Row {
	result := EmptyEffectRow()
	for _, e := range effects {
		result = combineEffects(result, e)
	}
	return result
}

// applySubstitutionToTyped applies substitution to typed nodes
func (tc *CoreTypeChecker) applySubstitutionToTyped(sub Substitution, node typedast.TypedNode) typedast.TypedNode {
	// Apply substitution to the type in the node
	if typ, ok := node.GetType().(Type); ok {
		substitutedType := ApplySubstitution(sub, typ)

		// We need to update the type in the node
		// Since TypedNode is an interface, we need to handle each concrete type
		switch n := node.(type) {
		case *typedast.TypedLit:
			n.Type = substitutedType
			return n
		case *typedast.TypedVar:
			n.Type = substitutedType
			return n
		case *typedast.TypedLambda:
			n.Type = substitutedType
			// Recursively apply to body
			n.Body = tc.applySubstitutionToTyped(sub, n.Body)
			return n
		case *typedast.TypedLet:
			n.Type = substitutedType
			// Recursively apply to value and body
			n.Value = tc.applySubstitutionToTyped(sub, n.Value)
			n.Body = tc.applySubstitutionToTyped(sub, n.Body)
			return n
		case *typedast.TypedBinOp:
			n.Type = substitutedType
			n.Left = tc.applySubstitutionToTyped(sub, n.Left)
			n.Right = tc.applySubstitutionToTyped(sub, n.Right)
			return n
		case *typedast.TypedApp:
			n.Type = substitutedType
			n.Func = tc.applySubstitutionToTyped(sub, n.Func)
			for i, arg := range n.Args {
				n.Args[i] = tc.applySubstitutionToTyped(sub, arg)
			}
			return n
		// Add more cases as needed
		default:
			// For other types, just return as is (temporary)
			return node
		}
	}
	return node
}

// defaultAmbiguities applies spec-compliant numeric defaulting at generalization boundaries
// This is the ONLY place where defaulting should happen in the entire system
func (tc *CoreTypeChecker) defaultAmbiguities(
	monotype Type,
	constraints []ClassConstraint,
) (Substitution, Type, []ClassConstraint, error) {

	if !tc.defaultingConfig.Enabled {
		return make(Substitution), monotype, constraints, nil
	}

	// Step 1: Compute ambiguous type variables A = ftv(C) \ ftv(τ)
	constraintVars := make(map[string]bool)
	for _, c := range constraints {
		collectConstraintVars(c.Type, constraintVars)
	}

	monotypeVars := make(map[string]bool)
	collectFreeVars(monotype, monotypeVars)

	ambiguousVars := make(map[string]bool)
	for v := range constraintVars {
		if !monotypeVars[v] {
			ambiguousVars[v] = true
		}
	}

	if tc.debugMode && len(ambiguousVars) > 0 {
		fmt.Printf("[debug] Ambiguous vars: ")
		for v := range ambiguousVars {
			fmt.Printf("%s ", v)
		}
		fmt.Printf("\n[debug] Monotype vars: ")
		for v := range monotypeVars {
			fmt.Printf("%s ", v)
		}
		fmt.Println()
	}

	if len(ambiguousVars) == 0 {
		return make(Substitution), monotype, constraints, nil
	}

	// Step 2: For each ambiguous var α, collect class set Kα
	varClasses := make(map[string]map[string]bool)
	for _, c := range constraints {
		if varName := extractVarName(c.Type); varName != "" && ambiguousVars[varName] {
			if varClasses[varName] == nil {
				varClasses[varName] = make(map[string]bool)
			}
			varClasses[varName][c.Class] = true
		}
	}

	// Step 3: Apply module defaults with conflict detection
	sub := make(Substitution)
	// traces := []DefaultingTrace{} // Not used

	for varName, classes := range varClasses {
		defaultType, err := tc.pickDefault(classes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("ambiguous type variable %s with classes %v: %w",
				varName, getClassNames(classes), err)
		}

		if defaultType != nil {
			sub[varName] = defaultType

			// Record trace for deterministic output
			trace := DefaultingTrace{
				TypeVar:   varName,
				ClassName: getFirstClassName(classes), // Representative class
				Default:   defaultType,
				Location:  "generalization boundary",
			}
			// traces = append(traces, trace) // Not used after this
			tc.defaultingConfig.Traces = append(tc.defaultingConfig.Traces, trace)

			if tc.debugMode {
				tc.logDefaulting(trace)
			}
		}
	}

	// Step 4: Apply substitution consistently everywhere
	if len(sub) > 0 {
		monotype = ApplySubstitution(sub, monotype)
		constraints = tc.applySubstitutionToConstraints(sub, constraints)

		// SAFETY CHECK: Ensure defaulting only affects Star-kinded types
		for varName, defaultType := range sub {
			if !isStarKinded(defaultType) {
				return nil, nil, nil, fmt.Errorf("INTERNAL ERROR: defaulting variable %s to non-Star type %s", varName, defaultType)
			}
		}
	}

	return sub, monotype, constraints, nil
}

// defaultAmbiguitiesTopLevel applies defaulting at top-level, including non-ambiguous numeric literals
func (tc *CoreTypeChecker) defaultAmbiguitiesTopLevel(
	monotype Type,
	constraints []ClassConstraint,
) (Substitution, Type, []ClassConstraint, error) {

	if !tc.defaultingConfig.Enabled {
		return make(Substitution), monotype, constraints, nil
	}

	// At top-level, we want to default ANY type variable with defaultable constraints
	// not just ambiguous ones (this gives the REPL experience users expect)

	// Collect all type variables in constraints that have defaultable classes
	defaultableVars := make(map[string]map[string]bool)
	for _, c := range constraints {
		if varName := extractVarName(c.Type); varName != "" {
			// Check if this class is defaultable
			if tc.isDefaultableClass(c.Class) {
				if defaultableVars[varName] == nil {
					defaultableVars[varName] = make(map[string]bool)
				}
				defaultableVars[varName][c.Class] = true
			}
		}
	}

	if len(defaultableVars) == 0 {
		return make(Substitution), monotype, constraints, nil
	}

	// Apply defaults to all defaultable variables
	sub := make(Substitution)
	// traces := []DefaultingTrace{} // Not used

	for varName, classes := range defaultableVars {
		defaultType, err := tc.pickDefault(classes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("ambiguous type variable %s with classes %v: %w",
				varName, getClassNames(classes), err)
		}

		if defaultType != nil {
			sub[varName] = defaultType

			trace := DefaultingTrace{
				TypeVar:   varName,
				ClassName: getFirstClassName(classes),
				Default:   defaultType,
				Location:  "top-level",
			}
			// traces = append(traces, trace) // Not used after this
			tc.defaultingConfig.Traces = append(tc.defaultingConfig.Traces, trace)

			if tc.debugMode {
				tc.logDefaulting(trace)
			}
		}
	}

	// Apply substitution
	if len(sub) > 0 {
		monotype = ApplySubstitution(sub, monotype)
		constraints = tc.applySubstitutionToConstraints(sub, constraints)

		// Safety check
		for varName, defaultType := range sub {
			if !isStarKinded(defaultType) {
				return nil, nil, nil, fmt.Errorf("INTERNAL ERROR: defaulting variable %s to non-Star type %s", varName, defaultType)
			}
		}
	}

	return sub, monotype, constraints, nil
}

// isDefaultableClass checks if a class can be defaulted
func (tc *CoreTypeChecker) isDefaultableClass(className string) bool {
	switch className {
	case "Num", "Fractional":
		return true
	default:
		return false
	}
}

// pickDefault applies module-scoped defaulting rules
func (tc *CoreTypeChecker) pickDefault(classes map[string]bool) (Type, error) {
	// Define neutral classes that don't affect numeric defaulting
	// These classes don't choose a numeric representation
	neutral := map[string]bool{
		"Eq":   true,
		"Ord":  true,
		"Show": true,
	}

	// Filter out neutral classes to find primary numeric constraints
	var primary []string
	for class := range classes {
		if !neutral[class] {
			primary = append(primary, class)
		}
	}

	// Handle defaulting based on remaining primary constraints
	switch {
	case len(primary) == 0:
		// Only neutral constraints present (Eq, Ord, Show)
		// Default to Int for Ord/Eq/Show when no numeric context
		// This handles comparisons like `x > y` where x, y are already Int
		if classes["Ord"] || classes["Eq"] {
			return &TCon{Name: "int"}, nil
		}
		// For Show-only, also default to Int
		if classes["Show"] {
			return &TCon{Name: "int"}, nil
		}
		return nil, fmt.Errorf("ambiguous type requires annotation")

	case len(primary) == 1 && primary[0] == "Num":
		// Pure Num constraint (possibly with neutral constraints like Eq, Ord)
		if def := tc.instanceEnv.DefaultFor("Num"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Num; add type annotation")

	case len(primary) == 1 && primary[0] == "Fractional":
		// Pure Fractional constraint (possibly with neutral constraints)
		if def := tc.instanceEnv.DefaultFor("Fractional"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Fractional; add type annotation")

	case len(primary) == 2 && classes["Fractional"] && classes["Num"]:
		// Fractional implies Num, so this is effectively just Fractional
		if def := tc.instanceEnv.DefaultFor("Fractional"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Fractional; add type annotation")

	default:
		// Mixed non-neutral constraints → require annotation
		// This maintains spec compliance: only default within a single family
		return nil, fmt.Errorf("mixed constraints require type annotation")
	}
}

// Helper functions for defaulting

func collectConstraintVars(t Type, vars map[string]bool) {
	switch typ := t.(type) {
	case *TVar:
		vars[typ.Name] = true
	case *TVar2:
		vars[typ.Name] = true
	case *TApp:
		collectConstraintVars(typ.Constructor, vars)
		for _, arg := range typ.Args {
			collectConstraintVars(arg, vars)
		}
	case *TFunc:
		for _, p := range typ.Params {
			collectConstraintVars(p, vars)
		}
		collectConstraintVars(typ.Return, vars)
	case *TFunc2:
		for _, p := range typ.Params {
			collectConstraintVars(p, vars)
		}
		collectConstraintVars(typ.Return, vars)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			collectConstraintVars(fieldType, vars)
		}
	}
}

func collectFreeVars(t Type, vars map[string]bool) {
	switch typ := t.(type) {
	case *TVar:
		vars[typ.Name] = true
	case *TVar2:
		vars[typ.Name] = true
	case *TApp:
		collectFreeVars(typ.Constructor, vars)
		for _, arg := range typ.Args {
			collectFreeVars(arg, vars)
		}
	case *TFunc:
		for _, p := range typ.Params {
			collectFreeVars(p, vars)
		}
		collectFreeVars(typ.Return, vars)
	case *TFunc2:
		for _, p := range typ.Params {
			collectFreeVars(p, vars)
		}
		collectFreeVars(typ.Return, vars)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			collectFreeVars(fieldType, vars)
		}
	}
}

func extractVarName(t Type) string {
	switch typ := t.(type) {
	case *TVar:
		return typ.Name
	case *TVar2:
		return typ.Name
	default:
		return ""
	}
}

func getClassNames(classes map[string]bool) []string {
	names := make([]string, 0, len(classes))
	for name := range classes {
		names = append(names, name)
	}
	// Sort for deterministic output
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

func getFirstClassName(classes map[string]bool) string {
	names := getClassNames(classes)
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

// isStarKinded checks that a type has kind Star (not effect row or record row)
func isStarKinded(t Type) bool {
	switch t.(type) {
	case *TVar:
		return true // Assume Star for TVar (simplified)
	case *TVar2:
		return true // Assume Star for TVar2 (simplified)
	case *Row:
		return false // Rows are not Star-kinded
	case *RowVar:
		return false // Row variables are not Star-kinded
	default:
		return true // TCon, TInt, TFloat, etc. are Star-kinded
	}
}

// ApplySubstEverywhere applies substitution coherently to all relevant data structures
func (tc *CoreTypeChecker) ApplySubstEverywhere(
	sub Substitution,
	monotype Type,
	constraints []ClassConstraint,
	typedNode typedast.TypedNode,
	envEntry interface{},
	bindingName string,
) (Type, []ClassConstraint, typedast.TypedNode, interface{}) {

	// Apply to monotype
	newMonotype := ApplySubstitution(sub, monotype)

	// Apply to constraints
	newConstraints := tc.applySubstitutionToConstraints(sub, constraints)

	// Apply to TypedAST
	newTypedNode := tc.applySubstitutionToTyped(sub, typedNode)

	// Apply to environment entry
	var newEnvEntry interface{}
	if scheme, ok := envEntry.(*Scheme); ok {
		// Apply substitution to the underlying type in the scheme
		newScheme := &Scheme{
			TypeVars:    scheme.TypeVars,
			RowVars:     scheme.RowVars,
			Constraints: scheme.Constraints,
			Type:        ApplySubstitution(sub, scheme.Type),
		}
		newEnvEntry = newScheme
	} else if typ, ok := envEntry.(Type); ok {
		newEnvEntry = ApplySubstitution(sub, typ)
	} else {
		newEnvEntry = envEntry
	}

	// Apply to resolved constraints
	tc.applySubstitutionToResolvedConstraints(sub)

	return newMonotype, newConstraints, newTypedNode, newEnvEntry
}

// applySubstitutionToResolvedConstraints updates the resolved constraints map
func (tc *CoreTypeChecker) applySubstitutionToResolvedConstraints(sub Substitution) {
	for nodeID, rc := range tc.resolvedConstraints {
		rc.Type = ApplySubstitution(sub, rc.Type)
		tc.resolvedConstraints[nodeID] = rc
	}
}

// applySubstitutionToConstraints applies a substitution to class constraints
func (tc *CoreTypeChecker) applySubstitutionToConstraints(sub Substitution, constraints []ClassConstraint) []ClassConstraint {
	result := make([]ClassConstraint, len(constraints))
	for i, c := range constraints {
		result[i] = ClassConstraint{
			Class:  c.Class,
			Type:   c.Type.Substitute(sub),
			Path:   c.Path,
			NodeID: c.NodeID,
		}
	}
	return result
}

// composeSubstitutions composes two substitutions: (S2 ∘ S1)(t) = S2(S1(t))
func composeSubstitutions(s1, s2 Substitution) Substitution {
	result := make(Substitution)

	// Apply s2 to the codomain of s1
	for v, t := range s1 {
		result[v] = ApplySubstitution(s2, t)
	}

	// Add bindings from s2 that aren't in s1
	for v, t := range s2 {
		if _, exists := result[v]; !exists {
			result[v] = t
		}
	}

	return result
}

// partitionConstraints separates ground (concrete) from non-ground (polymorphic) constraints
func (tc *CoreTypeChecker) partitionConstraints(constraints []ClassConstraint) (ground, nonGround []ClassConstraint) {
	for _, c := range constraints {
		if isGround(c.Type) {
			ground = append(ground, c)
		} else {
			nonGround = append(nonGround, c)
		}
	}
	return
}

// isGround checks if a type is ground (contains no type variables)
func isGround(t Type) bool {
	switch typ := t.(type) {
	case *TVar:
		return false
	case *TApp:
		// Check constructor
		if !isGround(typ.Constructor) {
			return false
		}
		// Check all args
		for _, arg := range typ.Args {
			if !isGround(arg) {
				return false
			}
		}
		return true
	case *TFunc:
		for _, p := range typ.Params {
			if !isGround(p) {
				return false
			}
		}
		return isGround(typ.Return)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			if !isGround(fieldType) {
				return false
			}
		}
		return true
	case *Row:
		// Check all label types
		for _, labelType := range typ.Labels {
			if !isGround(labelType) {
				return false
			}
		}
		// If there's a tail variable, it's not ground
		if typ.Tail != nil {
			return false
		}
		return true
	case *RowVar:
		// Row variables are not ground
		return false
	default:
		return true // TCon, TInt, TFloat, TString, TBool, TUnit
	}
}

// resolveGroundConstraints resolves ground class constraints using the instance environment
func (tc *CoreTypeChecker) resolveGroundConstraints(constraints []ClassConstraint, expr core.CoreExpr) error {
	for _, c := range constraints {
		// CRITICAL: Assert that constraint type is ground before resolution
		if !isGround(c.Type) {
			return fmt.Errorf("INTERNAL ERROR: attempting to resolve non-ground constraint %s[%s] - defaulting failed to make this ground", c.Class, c.Type)
		}

		// Look up instance in the environment
		_, err := tc.instanceEnv.Lookup(c.Class, c.Type)
		if err != nil {
			// No instance found - return error with hint
			if missingErr, ok := err.(*MissingInstanceError); ok {
				return fmt.Errorf("at %s: %v", c.Path[0], missingErr)
			}
			return err
		}

		// Instance found - record the resolved constraint if it has a NodeID
		if c.NodeID != 0 {
			// CRITICAL: Double-check that the type we're recording is ground
			if !isGround(c.Type) {
				return fmt.Errorf("INTERNAL ERROR: storing non-ground type %s in ResolvedConstraints for node %d", c.Type, c.NodeID)
			}

			// We need to determine the method based on the node
			// This will be done when we scan the Core AST
			// Create normalized type for dictionary lookup consistency
			normalizedType := &TCon{Name: NormalizeTypeName(c.Type)}
			// fmt.Printf("DEBUG RESOLVE: NodeID=%d, Class=%s, OrigType=%v, NormType=%s\n",
			// 	c.NodeID, c.Class, c.Type, normalizedType.Name)
			tc.resolvedConstraints[c.NodeID] = &ResolvedConstraint{
				NodeID:    c.NodeID,
				ClassName: c.Class,
				Type:      normalizedType, // Normalized type (float→Float, int→Int)
				Method:    "",             // Will be filled in during Core traversal
			}
		}
	}
	return nil
}

// mostSpecificNumericClass returns "Fractional" if any ClassConstraint on tUnified is Fractional,
// otherwise "Num". It ignores neutral classes (Eq/Ord/Show).
func (tc *CoreTypeChecker) mostSpecificNumericClass(ctx *InferenceContext, t Type) string {
	anyFractional := false

	// Walk all constraints currently in context
	for _, c := range ctx.qualifiedConstraints {
		if isNeutralClass(c.Class) { // Eq, Ord, Show
			continue
		}
		// Compare the *unified* types, not raw pointers
		if typesEqual(c.Type, t) {
			if c.Class == "Fractional" {
				anyFractional = true
			}
		}
	}
	if anyFractional {
		return "Fractional"
	}
	return "Num"
}

// isNeutralClass returns true for classes that don't influence numeric defaulting
func isNeutralClass(class string) bool {
	switch class {
	case "Eq", "Ord", "Show":
		return true
	default:
		return false
	}
}

// typesEqual compares types for equality (used for constraint matching)
func typesEqual(t1, t2 Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	switch typ1 := t1.(type) {
	case *TCon:
		if typ2, ok := t2.(*TCon); ok {
			return typ1.Name == typ2.Name
		}
	case *TVar:
		if typ2, ok := t2.(*TVar); ok {
			return typ1.Name == typ2.Name
		}
	case *TVar2:
		if typ2, ok := t2.(*TVar2); ok {
			return typ1.Name == typ2.Name
		}
	}

	// For more complex types, use string representation as fallback
	return t1.String() == t2.String()
}

// FillOperatorMethods fills in the Method field for resolved constraints
// by traversing the Core AST and matching NodeIDs (exported for REPL)
func (tc *CoreTypeChecker) FillOperatorMethods(expr core.CoreExpr) {
	// fmt.Printf("DEBUG FillOperatorMethods called with %T\n", expr)
	tc.walkCore(expr)
}

// walkCore recursively walks the Core AST to fill operator methods
func (tc *CoreTypeChecker) walkCore(expr core.CoreExpr) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *core.BinOp:
		// If we have a resolved constraint for this node, fill in the method
		if rc, ok := tc.resolvedConstraints[e.ID()]; ok {
			method := OperatorMethod(e.Op, false)
			// fmt.Printf("DEBUG BinOp: node=%d, op='%s' -> method='%s'\n", e.ID(), e.Op, method)
			rc.Method = method
		}
		// else {
		// 	// fmt.Printf("DEBUG BinOp: node=%d, op='%s' (NO CONSTRAINT)\n", e.ID(), e.Op)
		// }
		// Recurse on operands
		tc.walkCore(e.Left)
		tc.walkCore(e.Right)

	case *core.UnOp:
		// Fill in the method name for unary operators
		if rc, ok := tc.resolvedConstraints[e.ID()]; ok {
			rc.Method = OperatorMethod(e.Op, true)
		}
		tc.walkCore(e.Operand)

	case *core.Intrinsic:
		// Intrinsic nodes pass through - they'll be handled by OpLowering
		for _, arg := range e.Args {
			tc.walkCore(arg)
		}

	case *core.Let:
		tc.walkCore(e.Value)
		tc.walkCore(e.Body)

	case *core.LetRec:
		for _, binding := range e.Bindings {
			tc.walkCore(binding.Value)
		}
		tc.walkCore(e.Body)

	case *core.Lambda:
		tc.walkCore(e.Body)

	case *core.App:
		tc.walkCore(e.Func)
		for _, arg := range e.Args {
			tc.walkCore(arg)
		}

	case *core.If:
		tc.walkCore(e.Cond)
		tc.walkCore(e.Then)
		tc.walkCore(e.Else)

	case *core.Match:
		tc.walkCore(e.Scrutinee)
		for _, arm := range e.Arms {
			tc.walkCore(arm.Body)
		}

	case *core.Record:
		for _, field := range e.Fields {
			tc.walkCore(field)
		}

	case *core.RecordAccess:
		tc.walkCore(e.Record)

	case *core.List:
		for _, elem := range e.Elements {
			tc.walkCore(elem)
		}

	// Atomic expressions don't need recursion
	case *core.Var, *core.Lit, *core.DictRef:
		return
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

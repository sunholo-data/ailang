package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// GlobalResolver resolves global references to values
type GlobalResolver interface {
	ResolveValue(ref core.GlobalRef) (Value, error)
}

// BudgetEnforcer is implemented by effect contexts that support budget enforcement
// This interface avoids import cycles between eval and effects packages
type BudgetEnforcer interface {
	// WithBudgetLimits creates a new context with the given budget limits
	// Returns the new context that should be used for evaluation
	WithBudgetLimits(limits map[string]int) interface{}
}

// MinBudgetEnforcer is implemented by effect contexts that support minimum budget requirements
// M-DX25 M4: Minimum budgets ensure effects were actually exercised.
type MinBudgetEnforcer interface {
	// SetMinBudgets sets minimum usage requirements on the context's budget
	SetMinBudgets(minLimits map[string]int)
}

// MinimumChecker is implemented by effect contexts that can verify minimum usage
// M-DX25 M4: Called on scope exit to ensure effects were exercised.
type MinimumChecker interface {
	// CheckMinimums verifies all minimum requirements are met
	// Returns nil if all minimums satisfied, error otherwise
	CheckMinimums(position string) error
}

// ScopeCharger is implemented by effect contexts that support scoped budget charging
// M-DX25: When a scoped function returns, the caller is charged the callee's declared budget
type ScopeCharger interface {
	// PopScopeAndChargeCaller charges the caller context with declared semantic budgets
	// Called when restoring old context after function evaluation
	PopScopeAndChargeCaller()
}

// TraceRecorder is implemented by effect contexts that support semantic trace collection.
// M-TRACE-EXPORT: Used to record function calls during AILANG program execution.
type TraceRecorder interface {
	HasTraceCollector() bool
	RecordFunctionEnter(name string, args []string)
	RecordFunctionExit(name string, result string)
}

// CoreEvaluator evaluates Core AST programs after dictionary elaboration
type CoreEvaluator struct {
	env                   *Environment
	registry              *types.DictionaryRegistry
	resolver              GlobalResolver     // Resolver for global references
	experimentalBinopShim bool               // Feature flag for operator shim
	effContext            interface{}        // Effect context (interface{} avoids import cycle with effects package)
	recursionDepth        int                // Current recursion depth (for stack overflow detection)
	maxRecursionDepth     int                // Maximum allowed recursion depth (default: 10,000)
	coreTypeInfo          types.CoreTypeInfo // Type info for looking up effect budgets on closures
}

// Env returns the current environment (for module evaluation)
// This allows the runtime to add bindings so subsequent module-level declarations
// can reference earlier ones.
func (e *CoreEvaluator) Env() *Environment {
	return e.env
}

// SetEnv replaces the evaluator's current environment.
// Used by ModuleRuntime to isolate per-module scopes (M-MODULE-SCOPE).
func (e *CoreEvaluator) SetEnv(env *Environment) {
	e.env = env
}

// NewCoreEvaluatorWithRegistry creates a new Core evaluator with dictionary support
func NewCoreEvaluatorWithRegistry(registry *types.DictionaryRegistry) *CoreEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	return &CoreEvaluator{
		env:               env,
		registry:          registry,
		maxRecursionDepth: 10000, // Default: 10,000
	}
}

// NewCoreEvaluator creates a new core evaluator without a registry (for REPL)
func NewCoreEvaluator() *CoreEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	return &CoreEvaluator{
		env:               env,
		registry:          types.NewDictionaryRegistry(),
		maxRecursionDepth: 10000, // Default: 10,000
	}
}

// AddDictionary adds a dictionary to the evaluator (for REPL)
func (e *CoreEvaluator) AddDictionary(key string, dict core.DictValue) {
	// Register each method in the dictionary
	for method, impl := range dict.Methods {
		e.registry.Register("prelude", dict.TypeClass, dict.Type, method, impl)
	}
}

// SetGlobalResolver sets the resolver for global references
func (e *CoreEvaluator) SetGlobalResolver(resolver GlobalResolver) {
	e.resolver = resolver
}

// SetEffContext sets the effect context for this evaluator
//
// The effect context provides capability grants for effect operations.
// It uses interface{} to avoid import cycles with the effects package.
//
// Parameters:
//   - ctx: The effect context (should be *effects.EffContext)
//
// Example:
//
//	evaluator.SetEffContext(effCtx)
func (e *CoreEvaluator) SetEffContext(ctx interface{}) {
	e.effContext = ctx
	// M-ITERATIVE-LIST: Auto-wire FnCaller/FnCallerN for iterative builtins
	e.wireFnCallers()
}

// wireFnCallers wires FnCaller/FnCallerN on the EffContext using interface
// to avoid import cycles with the effects package.
func (e *CoreEvaluator) wireFnCallers() {
	type fnCallerWirer interface {
		SetFnCaller(func(Value, Value) (Value, error))
		SetFnCallerN(func(Value, []Value) (Value, error))
	}
	if ctx, ok := e.effContext.(fnCallerWirer); ok {
		ctx.SetFnCaller(e.CallValue)
		ctx.SetFnCallerN(e.CallValueN)
	}
}

// GetEffContext returns the current effect context
//
// Returns nil if no effect context has been set.
func (e *CoreEvaluator) GetEffContext() interface{} {
	return e.effContext
}

// SetCoreTypeInfo sets the type info for looking up effect budgets on closures
//
// When set, the evaluator will extract effect budgets from function types
// and populate EffectBudgets on FunctionValues during closure creation.
func (e *CoreEvaluator) SetCoreTypeInfo(cti types.CoreTypeInfo) {
	e.coreTypeInfo = cti
}

// SetDictionaryRegistry replaces the dictionary registry.
// M-DX19: Used to inject derived type class instances from the pipeline.
func (e *CoreEvaluator) SetDictionaryRegistry(reg *types.DictionaryRegistry) {
	e.registry = reg
}

// GetEnvironmentBindings returns all bindings in the current environment
func (e *CoreEvaluator) GetEnvironmentBindings() map[string]Value {
	return e.env.GetAllBindings()
}

// CallFunction calls a function value with the given arguments
//
// This is a helper for invoking FunctionValues from outside the evaluator,
// such as from the module runtime when calling entrypoints.
//
// Parameters:
//   - fn: The function value to call
//   - args: The arguments to pass to the function
//
// Returns:
//   - The result value from executing the function
//   - An error if execution fails
func (e *CoreEvaluator) CallFunction(fn *FunctionValue, args []Value) (Value, error) {
	// M-DOCPARSE-DX M1: Auto-curry support for CallFunction
	if len(args) > len(fn.Params) {
		return e.applyFunction(fn, args)
	}

	// Verify argument count
	if len(args) < len(fn.Params) {
		return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
	}

	// Create new environment with parameters bound
	newEnv := fn.Env.Clone()
	for i, param := range fn.Params {
		newEnv.Set(param, args[i])
	}

	// M-CAPABILITY-BUDGETS: Set up budget scoping if function has effect budgets
	var oldEffContext interface{}
	hasBudgetScope := (len(fn.EffectBudgets) > 0 || len(fn.EffectMinBudgets) > 0) && e.effContext != nil
	if hasBudgetScope {
		// Use BudgetEnforcer interface to avoid import cycle
		if enforcer, ok := e.effContext.(BudgetEnforcer); ok {
			oldEffContext = e.effContext
			e.effContext = enforcer.WithBudgetLimits(fn.EffectBudgets)

			// M-DX25 M4: Set min budgets if present
			if len(fn.EffectMinBudgets) > 0 {
				if minEnforcer, ok := e.effContext.(MinBudgetEnforcer); ok {
					minEnforcer.SetMinBudgets(fn.EffectMinBudgets)
				}
			}
		}
	}

	// Evaluate body in new environment
	oldEnv := e.env
	e.env = newEnv

	// M-VERIFY-CONTRACTS: Check preconditions before executing body
	if err := e.checkPreconditions(fn); err != nil {
		e.env = oldEnv
		return nil, err
	}

	var result Value
	var err error
	if coreBody, ok := fn.Body.(core.CoreExpr); ok {
		result, err = e.evalCore(coreBody)
	} else {
		err = fmt.Errorf("function body is not Core AST")
	}

	// M-VERIFY-CONTRACTS: Check postconditions before returning (if no error)
	if err == nil {
		if postErr := e.checkPostconditions(fn, result); postErr != nil {
			e.env = oldEnv
			return nil, postErr
		}
	}

	e.env = oldEnv

	// M-DX25: Handle budget scope exit
	if oldEffContext != nil {
		// M-DX25 M4: Check minimums before restoring context (if no error already)
		if err == nil {
			if checker, ok := e.effContext.(MinimumChecker); ok {
				err = checker.CheckMinimums("")
			}
		}

		// M-DX25: Charge caller with callee's declared budget
		if charger, ok := e.effContext.(ScopeCharger); ok {
			charger.PopScopeAndChargeCaller()
		}
		e.effContext = oldEffContext
	}

	return result, err
}

// CallValue calls a function value with a single argument.
// Unlike CallFunction, this accepts eval.Value to support both FunctionValue and BuiltinFunction.
// Used by M-STREAM-BIDI for invoking event handlers from Go code.
func (e *CoreEvaluator) CallValue(fn Value, arg Value) (Value, error) {
	switch f := fn.(type) {
	case *FunctionValue:
		return e.CallFunction(f, []Value{arg})
	case *BuiltinFunction:
		return f.Fn([]Value{arg})
	default:
		return nil, fmt.Errorf("cannot call non-function value: %T", fn)
	}
}

// CallValueN calls a function value with multiple arguments.
// M-ITERATIVE-LIST: Used by iterative builtins (e.g., foldl) that need multi-arg callbacks.
func (e *CoreEvaluator) CallValueN(fn Value, args []Value) (Value, error) {
	switch f := fn.(type) {
	case *FunctionValue:
		return e.CallFunction(f, args)
	case *BuiltinFunction:
		return f.Fn(args)
	default:
		return nil, fmt.Errorf("cannot call non-function value: %T", fn)
	}
}

// EvalLetRecBindings evaluates a LetRec and returns its bindings without evaluating the body
//
// This uses the same 3-phase RefCell algorithm as evalCoreLetRec to ensure proper
// recursion support in module code. The algorithm:
//  1. Pre-allocate RefCell indirection cells for all bindings
//  2. Evaluate RHS under recursive environment (lambdas safe, non-lambdas strict)
//  3. Return initialized values from cells
//
// This is called by the module runtime when loading module declarations.
func (e *CoreEvaluator) EvalLetRecBindings(letrec *core.LetRec) (map[string]Value, error) {
	// Phase 1: Pre-allocate indirection cells and extend environment
	recEnv := e.env.NewChildEnvironment()
	cells := make(map[string]*RefCell, len(letrec.Bindings))

	for _, binding := range letrec.Bindings {
		cell := &RefCell{} // Uninitialized cell
		cells[binding.Name] = cell
		recEnv.Set(binding.Name, &IndirectValue{Cell: cell})
	}

	// Phase 2: Evaluate RHS under recursive environment
	oldEnv := e.env
	e.env = recEnv
	defer func() { e.env = oldEnv }()

	bindings := make(map[string]Value, len(letrec.Bindings))
	for _, binding := range letrec.Bindings {
		// Optimize for lambda RHS: build closure immediately (safe, body executes later)
		if lam, ok := isLambda(binding.Value); ok {
			fv, err := e.buildClosure(lam, recEnv)
			if err != nil {
				return nil, err
			}
			cells[binding.Name].Val = fv
			cells[binding.Name].Init = true
			bindings[binding.Name] = fv
			continue
		}

		// Non-lambda RHS: strict evaluation with cycle detection
		cells[binding.Name].Visiting = true
		val, err := e.evalCore(binding.Value)
		cells[binding.Name].Visiting = false
		if err != nil {
			return nil, err
		}

		cells[binding.Name].Val = val
		cells[binding.Name].Init = true
		bindings[binding.Name] = val
	}

	// Phase 3: Return bindings (cells are already in environment)
	return bindings, nil
}

// SetExperimentalBinopShim enables the experimental operator shim
func (e *CoreEvaluator) SetExperimentalBinopShim(enabled bool) {
	e.experimentalBinopShim = enabled
}

// SetMaxRecursionDepth sets the maximum allowed recursion depth
func (e *CoreEvaluator) SetMaxRecursionDepth(max int) {
	e.maxRecursionDepth = max
}

// SetResolver sets the resolver for global references
func (e *CoreEvaluator) SetResolver(resolver GlobalResolver) {
	e.resolver = resolver
}

// Eval evaluates a single expression (simplified for REPL)
func (e *CoreEvaluator) Eval(expr core.CoreExpr) (Value, error) {
	return e.evalCore(expr)
}

// EvalCoreProgram evaluates a Core program
func (e *CoreEvaluator) EvalCoreProgram(prog *core.Program) (Value, error) {
	var lastResult Value = &UnitValue{}

	for _, decl := range prog.Decls {
		result, err := e.evalCore(decl)
		if err != nil {
			return nil, err
		}
		lastResult = result
	}

	return lastResult, nil
}

// ContractChecker is an interface for checking contract violations via EffContext
// M-VERIFY-CONTRACTS: Used to integrate with the effects.ContractContext
type ContractChecker interface {
	CheckRequires(cond bool, msg, location string) error
	CheckEnsures(cond bool, msg, location string) error
	IsContractCheckingEnabled() bool
}

// checkPreconditions checks all preconditions (requires blocks) before function execution
// M-VERIFY-CONTRACTS: Evaluates contract expressions in the current environment
func (e *CoreEvaluator) checkPreconditions(fn *FunctionValue) error {
	if len(fn.Preconditions) == 0 {
		return nil
	}

	// Check if contract checking is enabled
	checker, ok := e.effContext.(ContractChecker)
	if !ok || !checker.IsContractCheckingEnabled() {
		return nil // Contract checking disabled
	}

	for _, pre := range fn.Preconditions {
		// Evaluate the contract expression
		coreExpr, ok := pre.Expr.(core.CoreExpr)
		if !ok {
			continue // Skip non-Core expressions
		}

		result, err := e.evalCore(coreExpr)
		if err != nil {
			return fmt.Errorf("precondition evaluation error at %s: %w", pre.Location, err)
		}

		// Check result is boolean
		boolVal, ok := result.(*BoolValue)
		if !ok {
			return fmt.Errorf("precondition at %s must return bool, got %T", pre.Location, result)
		}

		// Report via ContractContext
		if err := checker.CheckRequires(boolVal.Value, pre.Message, pre.Location); err != nil {
			return err
		}
	}

	return nil
}

// checkPostconditions checks all postconditions (ensures blocks) after function execution
// M-VERIFY-CONTRACTS: Evaluates contract expressions with 'result' bound to the return value
func (e *CoreEvaluator) checkPostconditions(fn *FunctionValue, result Value) error {
	if len(fn.Postconditions) == 0 {
		return nil
	}

	// Check if contract checking is enabled
	checker, ok := e.effContext.(ContractChecker)
	if !ok || !checker.IsContractCheckingEnabled() {
		return nil // Contract checking disabled
	}

	// Bind 'result' for postcondition checking
	e.env.Set("result", result)
	defer func() {
		// Clean up result binding
		// Note: We don't need to unset since the env will be restored anyway
	}()

	for _, post := range fn.Postconditions {
		// Evaluate the contract expression
		coreExpr, ok := post.Expr.(core.CoreExpr)
		if !ok {
			continue // Skip non-Core expressions
		}

		checkResult, err := e.evalCore(coreExpr)
		if err != nil {
			return fmt.Errorf("postcondition evaluation error at %s: %w", post.Location, err)
		}

		// Check result is boolean
		boolVal, ok := checkResult.(*BoolValue)
		if !ok {
			return fmt.Errorf("postcondition at %s must return bool, got %T", post.Location, checkResult)
		}

		// Report via ContractContext
		if err := checker.CheckEnsures(boolVal.Value, post.Message, post.Location); err != nil {
			return err
		}
	}

	return nil
}

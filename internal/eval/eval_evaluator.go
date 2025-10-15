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

// CoreEvaluator evaluates Core AST programs after dictionary elaboration
type CoreEvaluator struct {
	env                   *Environment
	registry              *types.DictionaryRegistry
	resolver              GlobalResolver // Resolver for global references
	experimentalBinopShim bool           // Feature flag for operator shim
	effContext            interface{}    // Effect context (interface{} avoids import cycle with effects package)
	recursionDepth        int            // Current recursion depth (for stack overflow detection)
	maxRecursionDepth     int            // Maximum allowed recursion depth (default: 10,000)
}

// Env returns the current environment (for module evaluation)
// This allows the runtime to add bindings so subsequent module-level declarations
// can reference earlier ones.
func (e *CoreEvaluator) Env() *Environment {
	return e.env
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
}

// GetEffContext returns the current effect context
//
// Returns nil if no effect context has been set.
func (e *CoreEvaluator) GetEffContext() interface{} {
	return e.effContext
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
	// Verify argument count
	if len(args) != len(fn.Params) {
		return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
	}

	// Create new environment with parameters bound
	newEnv := fn.Env.Clone()
	for i, param := range fn.Params {
		newEnv.Set(param, args[i])
	}

	// Evaluate body in new environment
	oldEnv := e.env
	e.env = newEnv

	var result Value
	var err error
	if coreBody, ok := fn.Body.(core.CoreExpr); ok {
		result, err = e.evalCore(coreBody)
	} else {
		err = fmt.Errorf("function body is not Core AST")
	}

	e.env = oldEnv
	return result, err
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

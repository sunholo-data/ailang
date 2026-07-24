package eval

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// GlobalResolver resolves global references to values
type GlobalResolver interface {
	ResolveValue(ref core.GlobalRef) (Value, error)
}

// FallbackResolver tries the primary resolver first, then falls back to the
// secondary resolver. Used by M-DX-XPKG-RESOLVE to combine the caller's resolver
// (for builtins/effect context) with the function's defining resolver (for constructors
// and module-specific lookups that the caller's resolver can't see).
type FallbackResolver struct {
	Primary   GlobalResolver // Caller's resolver (builtins, effects)
	Secondary GlobalResolver // Function's defining resolver (constructors, module scope)
}

// ResolveValue tries Primary first, then Secondary on failure.
func (f *FallbackResolver) ResolveValue(ref core.GlobalRef) (Value, error) {
	val, err := f.Primary.ResolveValue(ref)
	if err == nil {
		return val, nil
	}
	// Primary failed — try secondary (defining module's resolver)
	return f.Secondary.ResolveValue(ref)
}

// resolverCovers reports whether the given resolver chain already reaches `target`
// as either a Primary or Secondary at any depth. Used by function-application sites
// (M-PERF6-PHASE4 M2a) to skip re-wrapping in a new FallbackResolver when the chain
// already has a path to the callee's defining resolver — prevents O(depth) growth
// on tight-loop cross-module calls like `map(\cell. parseXlsxCell(cell, ss), cells)`.
func resolverCovers(chain GlobalResolver, target GlobalResolver) bool {
	if chain == target {
		return true
	}
	fr, ok := chain.(*FallbackResolver)
	if !ok {
		return false
	}
	return resolverCovers(fr.Primary, target) || resolverCovers(fr.Secondary, target)
}

// BudgetFrameEnforcer is implemented by effect contexts that support the
// hierarchical per-invocation budget frame model (M-BUDGET-SCOPING-BUG).
//
// Frames are pushed on entry to any function whose signature carries an
// @limit/@min annotation, and popped unwind-safe on EVERY exit. The frame stack
// lives on shared per-execution state so it survives the shallow copy performed
// by WithBudget — enforcement (bubbling charge + pre-op @limit check) is done by
// the effects package against every active frame.
//
// This interface avoids import cycles between eval and effects packages.
type BudgetFrameEnforcer interface {
	// PushBudgetFrame pushes a new per-invocation frame with the given
	// annotations onto the shared frame stack.
	PushBudgetFrame(fnName string, limits, mins map[string]int)
	// PopBudgetFrame pops the innermost frame. If bodyErr is nil (normal exit)
	// the frame's @min requirements are checked and any violation is returned.
	// If bodyErr is non-nil (error/exceptional exit) the frame is still popped
	// but the @min check is SUPPRESSED and bodyErr is returned unchanged.
	PopBudgetFrame(fnName string, bodyErr error) error
}

// RandModeEnforcer is implemented by effect contexts that support mode-aware
// Rand dispatch (M-EFFECT-REPLAY-CONTRACTS). A non-os Rand mode (seeded/crypto)
// declared on a function's effect row is pushed on entry and popped unwind-safe
// on exit, so the innermost explicit mode is in effect for Rand draws while that
// function (and any os-mode stdlib wrapper it calls) runs. Bare/os-mode
// functions push nothing, leaving the global source untouched. This interface
// avoids import cycles between eval and effects.
type RandModeEnforcer interface {
	PushRandMode(mode string)
	PopRandMode(mode string)
}

// budgetChargeScoper is implemented by effect contexts that maintain a budget
// charge-scope depth (M-BUDGET-SCOPING-BUG). The evaluator resets this depth
// across AILANG function-call boundaries so a builtin's charge scope does not
// leak into a user callback invoked from inside that builtin.
type budgetChargeScoper interface {
	SaveAndResetBudgetChargeScope() int
	RestoreBudgetChargeScope(prev int)
}

// enterBudgetChargeBoundary resets the effect context's budget charge-scope depth
// on entry to an AILANG function body and returns a restore closure. In the
// common case (depth already 0, e.g. a normal top-level call) this is a no-op.
func (e *CoreEvaluator) enterBudgetChargeBoundary() func() {
	scoper, ok := e.effContext.(budgetChargeScoper)
	if !ok {
		return func() {}
	}
	prev := scoper.SaveAndResetBudgetChargeScope()
	if prev == 0 {
		return func() {}
	}
	return func() { scoper.RestoreBudgetChargeScope(prev) }
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

// Fork creates a new evaluator for concurrent request handling.
// It shares read-only state (registry, config) but creates fresh per-request
// state (env, resolver, recursionDepth). This is the concurrency primitive:
// each HTTP request goroutine gets its own Fork.
func (e *CoreEvaluator) Fork() *CoreEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	forked := &CoreEvaluator{
		env:                   env,
		registry:              e.registry, // shared, read-only after init
		experimentalBinopShim: e.experimentalBinopShim,
		maxRecursionDepth:     e.maxRecursionDepth,
		// resolver: nil        — set by CallEntrypoint per request
		// recursionDepth: 0    — fresh per request
		// coreTypeInfo: zero   — set during evaluation
	}

	// Clone EffContext so each request has its own FnCaller/FnCallerN bindings.
	// The shallow copy shares config (Caps, Env, Clock, Net) but the forked
	// evaluator's CallValue/CallValueN get wired to this fork, not the parent.
	if e.effContext != nil {
		type cloneable interface {
			Clone() interface{}
		}
		if c, ok := e.effContext.(cloneable); ok {
			forked.effContext = c.Clone()
		} else {
			forked.effContext = e.effContext
		}
		forked.wireFnCallers()
	}

	return forked
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
func (e *CoreEvaluator) CallFunction(fn *FunctionValue, args []Value) (retVal Value, err error) {
	// M-ZERO-ARG-SURFACES (v0.22.0): zero-arg exports (`export func f() -> T`)
	// compile to a single implicit unit param named "_" (parser convention in
	// internal/parser/parser_func.go). External callers (apiserver, WASM
	// ailangCall, bytecode entrypoint) pass zero args; inject UnitValue here so
	// every surface inherits the behavior. Mirrors and replaces the apiserver
	// retry-on-error (commits 8cc21027, 4075a402) and the REPL/WASM InvokeExport
	// workaround, both of which can now be deleted.
	//
	// Name-based detection ("_" param) rather than IsZeroArgExport flag plumbing
	// keeps the change shallow. A user-written `\_. body` lambda matched against
	// a zero-arg call gets a harmless UnitValue bound to `_` — semantically a
	// no-op since `_` is by convention an ignored binding.
	if isZeroArgUnitInjection(fn, args) {
		args = []Value{&UnitValue{}}
	}

	// M-DOCPARSE-DX M1: Auto-curry support for CallFunction
	if len(args) > len(fn.Params) {
		return e.applyFunction(fn, args)
	}

	// Verify argument count
	if len(args) < len(fn.Params) {
		return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
	}

	// Create new environment with parameters bound.
	// M-PERF6 Phase 4a: Use NewChildEnvironment (O(1)) instead of Clone (O(n)).
	// Params go into an empty child scope; lookups for captured names traverse
	// the parent chain. Semantically equivalent to Clone+Set because AILANG
	// never mutates existing bindings in the defining environment.
	newEnv := fn.Env.NewChildEnvironment()
	for i, param := range fn.Params {
		newEnv.Set(param, args[i])
	}

	// M-BUDGET-SCOPING-BUG: push a per-invocation budget frame if the signature
	// is annotated. The deferred pop is unwind-safe: it fires on every exit path
	// (normal, error, precondition early-return) so a callee error can never leak
	// a stale frame. On normal exit it runs the frame's @min check; on error exit
	// the @min check is suppressed and the body error propagates unchanged.
	if e.effContext != nil {
		defer e.enterBudgetChargeBoundary()()
		if e.pushBudgetFrameIfAnnotated(fn, "") {
			defer e.deferredPopBudgetFrame("", &err)
		}
		// M-EFFECT-REPLAY-CONTRACTS: push the declared non-os Rand mode for the
		// dynamic extent of this call; unwind-safe pop on every exit path.
		if mode := e.pushRandModeIfDeclared(fn); mode != "" {
			defer e.deferredPopRandMode(mode)
		}
	}

	// Evaluate body in new environment
	oldEnv := e.env
	e.env = newEnv

	// M-VERIFY-CONTRACTS: Check preconditions before executing body
	if preErr := e.checkPreconditions(fn); preErr != nil {
		e.env = oldEnv
		err = preErr
		return nil, err
	}

	var result Value
	if coreBody, ok := fn.Body.(core.CoreExpr); ok {
		result, err = e.evalCore(coreBody)
	} else {
		err = fmt.Errorf("function body is not Core AST")
	}

	// M-VERIFY-CONTRACTS: Check postconditions before returning (if no error)
	if err == nil {
		if postErr := e.checkPostconditions(fn, result); postErr != nil {
			e.env = oldEnv
			err = postErr
			return nil, err
		}
	}

	e.env = oldEnv

	return result, err
}

// pushBudgetFrameIfAnnotated pushes a per-invocation budget frame when the
// function's signature carries any @limit/@min annotation (M-BUDGET-SCOPING-BUG).
//
// It returns whether a frame was pushed. The caller MUST pair a successful push
// with a deferred deferredPopBudgetFrame so the frame is popped on EVERY exit
// path (normal, error, precondition early-return) — the pop is NOT guarded by a
// manual assignment at the bottom of the function, which historically leaked
// frames on precondition failure.
func (e *CoreEvaluator) pushBudgetFrameIfAnnotated(fn *FunctionValue, fnName string) bool {
	if len(fn.EffectBudgets) == 0 && len(fn.EffectMinBudgets) == 0 {
		return false
	}
	enforcer, ok := e.effContext.(BudgetFrameEnforcer)
	if !ok {
		return false
	}
	enforcer.PushBudgetFrame(fnName, fn.EffectBudgets, fn.EffectMinBudgets)
	return true
}

// pushRandModeIfDeclared pushes the function's declared non-os Rand mode onto
// the effect context (M-EFFECT-REPLAY-CONTRACTS) and returns the mode that was
// pushed (or "" if none). The caller MUST pair a non-empty return with a
// deferred PopRandMode(mode) so the mode is popped on every exit path. os-mode /
// mode-less functions push nothing (EffectRandMode is "" for them), so the mode
// stack stays empty and Rand draws use the unchanged global source.
func (e *CoreEvaluator) pushRandModeIfDeclared(fn *FunctionValue) string {
	if fn.EffectRandMode == "" {
		return ""
	}
	enforcer, ok := e.effContext.(RandModeEnforcer)
	if !ok {
		return ""
	}
	enforcer.PushRandMode(fn.EffectRandMode)
	return fn.EffectRandMode
}

// deferredPopRandMode pops a Rand mode pushed by pushRandModeIfDeclared. A no-op
// when mode is "" (nothing was pushed). Intended for use in a defer so the pop
// is unwind-safe.
func (e *CoreEvaluator) deferredPopRandMode(mode string) {
	if mode == "" {
		return
	}
	if enforcer, ok := e.effContext.(RandModeEnforcer); ok {
		enforcer.PopRandMode(mode)
	}
}

// deferredPopBudgetFrame pops the innermost budget frame, running the frame's
// @min check on normal exit and suppressing it on error exit. It merges any
// @min violation into *errp only when the body did not already error (so a real
// error is never masked by a @min violation). Intended for use in a defer.
func (e *CoreEvaluator) deferredPopBudgetFrame(fnName string, errp *error) {
	enforcer, ok := e.effContext.(BudgetFrameEnforcer)
	if !ok {
		return
	}
	if popErr := enforcer.PopBudgetFrame(fnName, *errp); popErr != nil {
		*errp = popErr
	}
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

package runtime

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/petermattis/goid"
	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

var debugConcurrencyBuiltins = os.Getenv("DEBUG_CONCURRENCY") == "1"

// suppress unused import
var _ = log.Printf

// BuiltinRegistry holds native Go implementations of builtin functions
//
// Builtins are functions implemented in Go that can be called from AILANG modules.
// They are identified by names starting with underscore (e.g., _io_print).
//
// The registry provides:
//   - Type-safe function implementations routed through the effect system
//   - Runtime access via GetBuiltin()
//   - Automatic registration of stdlib functions
//
// Thread-safety: The registry is initialized once and read-only after that,
// so it is safe to use concurrently.
type BuiltinRegistry struct {
	builtins  map[string]eval.Value
	evaluator *eval.CoreEvaluator // Reference to shared evaluator for EffContext access

	// Per-goroutine evaluator overrides for concurrent serve-api requests.
	// When a forked evaluator is active, it registers itself here so builtins
	// use the forked evaluator's EffContext (with correct FnCaller binding)
	// instead of the shared evaluator's.
	goroutineEvals sync.Map // map[int64]*eval.CoreEvaluator

	// goroutineEvalCount tracks how many forked evaluators are active.
	// When 0 (the common case for CLI/REPL), getEffContext skips the
	// sync.Map lookup entirely — no goroutine ID extraction needed.
	goroutineEvalCount atomic.Int64
}

// NewBuiltinRegistry creates a new builtin registry with all stdlib functions registered
//
// Parameters:
//   - evaluator: The evaluator (needed to access EffContext during builtin calls)
//
// Returns:
//   - A fully-initialized BuiltinRegistry
func NewBuiltinRegistry(evaluator *eval.CoreEvaluator) *BuiltinRegistry {
	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: evaluator,
	}

	// Use new spec-based registry (M-DX1 migration complete in v0.3.10)
	br.registerFromSpecRegistry()

	return br
}

// Get looks up a builtin function by name
//
// Parameters:
//   - name: The builtin function name (e.g., "_io_print")
//
// Returns:
//   - The builtin function value if found
//   - A boolean indicating whether the builtin was found
func (br *BuiltinRegistry) Get(name string) (eval.Value, bool) {
	val, ok := br.builtins[name]
	return val, ok
}

// registerFromSpecRegistry registers builtins from the new spec-based registry
// This is the new centralized registration path (enabled with AILANG_BUILTINS_REGISTRY=1)
func (br *BuiltinRegistry) registerFromSpecRegistry() {
	specs := builtins.AllSpecs()

	for name, spec := range specs {
		// Capture spec for closure
		builtinSpec := spec

		br.builtins[name] = &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if debugConcurrencyBuiltins {
					log.Printf("[BUILTIN] %s enter (goroutine %d)", builtinSpec.Name, goid.Get())
				}
				ctx := br.getEffContext()
				if ctx == nil && !builtinSpec.IsPure {
					return nil, fmt.Errorf("%s: no effect context available", builtinSpec.Name)
				}

				// M-CAPABILITY-BUDGETS: Check capability and consume budget before calling effect builtin
				if ctx != nil && builtinSpec.Effect != "" {
					if err := ctx.RequireCapWithBudget(builtinSpec.Effect, ""); err != nil {
						return nil, err
					}
				}

				result, err := builtinSpec.Impl(ctx, args)
				if debugConcurrencyBuiltins {
					log.Printf("[BUILTIN] %s done (goroutine %d, err=%v)", builtinSpec.Name, goid.Get(), err)
				}
				return result, err
			},
		}
	}
}

// SetGoroutineEvaluator registers a forked evaluator for the current goroutine.
// Builtins will use this evaluator's EffContext instead of the shared one.
// Must be paired with ClearGoroutineEvaluator when the request completes.
func (br *BuiltinRegistry) SetGoroutineEvaluator(e *eval.CoreEvaluator) {
	br.goroutineEvals.Store(goid.Get(), e)
	br.goroutineEvalCount.Add(1)
}

// ClearGoroutineEvaluator removes the goroutine-local evaluator override.
func (br *BuiltinRegistry) ClearGoroutineEvaluator() {
	br.goroutineEvals.Delete(goid.Get())
	br.goroutineEvalCount.Add(-1)
}

// getEffContext retrieves the EffContext for the current goroutine.
//
// Resolution order:
//  1. Check goroutine-local evaluator override (set by Fork for concurrent requests)
//  2. Fall back to shared evaluator (single-threaded CLI, REPL, module eval)
//
// M-ITERATIVE-LIST: If no EffContext exists but evaluator is available,
// creates a minimal default EffContext with FnCaller/FnCallerN wired.
func (br *BuiltinRegistry) getEffContext() *effects.EffContext {
	// Fast path: no forked evaluators registered (CLI, REPL, single-goroutine).
	// Skips goid.Get() and sync.Map lookup entirely.
	evaluator := br.evaluator
	if br.goroutineEvalCount.Load() > 0 {
		// Slow path: concurrent serve-api — look up per-goroutine evaluator
		if goroutineEval, ok := br.goroutineEvals.Load(goid.Get()); ok {
			evaluator = goroutineEval.(*eval.CoreEvaluator)
		}
	}

	if evaluator == nil {
		return nil
	}
	ctx := evaluator.GetEffContext()
	if ctx == nil {
		// M-ITERATIVE-LIST: Create a minimal default EffContext with FnCallers wired
		// so pure iterative builtins (_list_map, etc.) can call AILANG callbacks.
		defaultCtx := effects.NewEffContext(nil)
		defaultCtx.FnCaller = evaluator.CallValue
		defaultCtx.FnCallerN = evaluator.CallValueN
		evaluator.SetEffContext(defaultCtx)
		return defaultCtx
	}
	effCtx, ok := ctx.(*effects.EffContext)
	if !ok {
		return nil
	}
	return effCtx
}

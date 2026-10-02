package eval

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// Tail-call elimination (M-EVAL-TAIL-CALLS, #1486,
// design_docs/planned/v0_51_1/m-eval-tail-calls.md).
//
// Every AILANG call used to be Go recursion (evalCoreApp → evalCore(body)), so
// a loop written as a tail call hit RT_REC_003 at 10,000 iterations while the
// bytecode VM, which reuses its frame on TAIL_CALL, ran it in constant space.
// applyFunctionValue is now the one place a FunctionValue body runs. When the
// body's tail expression is itself a call, evalCoreApp returns a *tailCall
// instead of nesting, and the loop below replaces the current frame with the
// callee's.

// tailCall is a call reached in tail position, handed back to the enclosing
// applyFunctionValue loop. Only evalCoreApp creates it, and only when its tail
// argument is true, which only applyFunctionValue requests, and only for a frame
// that may be replaced. It never escapes this file's loop.
type tailCall struct {
	fn   *FunctionValue
	args []Value
	name string
}

func (*tailCall) Type() string { return "tailcall" }
func (t *tailCall) String() string {
	return fmt.Sprintf("<tail call %s/%d>", t.name, len(t.args))
}

// frameContract is what a frame does besides running its body. It is the
// caller's contract for the first frame (D3); every tail-called frame follows
// appFrame, because without TCE it would have been reached through evalCoreApp.
type frameContract struct {
	countDepth   bool // recursionDepth guard (RT_REC_003)
	trace        bool // RecordFunctionEnter/Exit
	wrapResolver bool // FallbackResolver over fn.Resolver
}

var (
	// appFrame is evalCoreApp's contract.
	appFrame = frameContract{countDepth: true, trace: true, wrapResolver: true}
	// callFunctionFrame is CallFunction's: no depth guard, no trace, no
	// resolver wrap (entry points and builtin callbacks).
	callFunctionFrame = frameContract{}
)

// replaceable reports whether fn's frame may be discarded when its body ends in
// a tail call: only if nothing runs after the body (D4). An active `ensures`
// evaluates the result, a budget frame pops with an @min check, and a rand mode
// pops; such frames stay nested.
func (e *CoreEvaluator) replaceable(fn *FunctionValue) bool {
	if len(fn.EffectBudgets) > 0 || len(fn.EffectMinBudgets) > 0 || fn.EffectRandMode != "" || fn.EffectNetScope != "" {
		return false
	}
	if len(fn.Postconditions) > 0 {
		if c, ok := e.effContext.(ContractChecker); ok && c.IsContractCheckingEnabled() {
			return false
		}
	}
	return true
}

// applyFunctionValue runs fn on exactly len(fn.Params) args, following tail
// calls in a loop. Per frame it does what a nested call does today: trace
// enter, child env, budget charge boundary, budget frame and rand mode (nested
// frames only), resolver wrap, requires, body, ensures. Trace exits are
// replayed innermost-first at the end, by today's per-frame rules (D6): a body
// error still emits the exit; a frame whose requires/ensures fails, or whose
// body is not Core, emits none.
func (e *CoreEvaluator) applyFunctionValue(fn *FunctionValue, args []Value, name string, first frameContract) (result Value, err error) {
	baseEnv, baseResolver := e.env, e.resolver
	defer func() { e.env, e.resolver = baseEnv, baseResolver }()

	if first.countDepth {
		e.recursionDepth++
		if e.recursionDepth > e.maxRecursionDepth {
			e.recursionDepth--
			return nil, &RecursionLimitError{Limit: e.maxRecursionDepth}
		}
		defer func() { e.recursionDepth-- }()
	}

	// Budget charge boundaries entered per frame, restored innermost-first.
	// Only non-no-op restores are kept; at a tail call the charge depth is 0,
	// so in practice there is at most one.
	var restores []func()
	defer func() {
		for i := len(restores) - 1; i >= 0; i-- {
			restores[i]()
		}
	}()

	recorder, _ := e.effContext.(TraceRecorder)
	tracing := recorder != nil && recorder.HasTraceCollector() && recorder.RecordsFunctionCalls()
	var traced []string // names of frames that emitted an enter, outermost first
	replayExits := func(res Value) {
		for i := len(traced) - 1; i >= 0; i-- {
			recorder.RecordFunctionExit(traced[i], recorder.RenderTraceValue(res))
		}
	}

	frame := first
	for {
		if frame.trace && tracing {
			argStrs := make([]string, len(args))
			for i, a := range args {
				argStrs[i] = recorder.RenderTraceValue(a)
			}
			recorder.RecordFunctionEnter(name, argStrs)
			traced = append(traced, name)
		}

		newEnv := fn.Env.NewChildEnvironment()
		for i, param := range fn.Params {
			newEnv.Set(param, args[i])
		}

		canReplace := e.replaceable(fn)
		if e.effContext != nil {
			if r := e.enterBudgetChargeBoundary(); r != nil {
				restores = append(restores, r)
			}
			// Only a nested (non-replaceable) frame can own a budget frame or
			// rand mode, and a nested frame is always the loop's last, so a
			// function-level defer pops it exactly when the nested call would.
			if !canReplace {
				if e.pushBudgetFrameIfAnnotated(fn, name) {
					defer e.deferredPopBudgetFrame(name, &err)
				}
				if mode := e.pushRandModeIfDeclared(fn); mode != "" {
					defer e.deferredPopRandMode(mode)
				}
				if scope := e.pushNetScopeIfDeclared(fn); scope != "" {
					defer e.deferredPopNetScope(scope)
				}
			}
		}

		e.env = newEnv
		// Wrap the CURRENT chain, as the nested call does today, so each frame
		// sees today's resolver chain; resolverCovers stops growth once a
		// module's resolver is present.
		if frame.wrapResolver && fn.Resolver != nil && !resolverCovers(e.resolver, fn.Resolver) {
			e.resolver = &FallbackResolver{Primary: e.resolver, Secondary: fn.Resolver}
		}

		if preErr := e.checkPreconditions(fn); preErr != nil {
			if frame.trace && tracing {
				traced = traced[:len(traced)-1]
			}
			replayExits(nil)
			return nil, preErr
		}

		coreBody, ok := fn.Body.(core.CoreExpr)
		if !ok {
			if frame.trace && tracing {
				traced = traced[:len(traced)-1]
			}
			replayExits(nil)
			return nil, fmt.Errorf("function body is not Core AST")
		}

		result, err = e.evalCoreT(coreBody, canReplace)
		if err == nil {
			if tc, isTail := result.(*tailCall); isTail {
				fn, args, name, frame = tc.fn, tc.args, tc.name, appFrame
				continue
			}
			if postErr := e.checkPostconditions(fn, result); postErr != nil {
				if frame.trace && tracing {
					traced = traced[:len(traced)-1]
				}
				replayExits(nil)
				return nil, postErr
			}
		}

		if tracing {
			replayExits(result)
		}
		return result, err
	}
}

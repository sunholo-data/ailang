package runner

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/vm"
)

// bytecodeBridge implements vm.EvalInterop for the `ailang run --bytecode`
// path. When the VM hits a function whose prototype is marked EvalOnly
// (because the bytecode compiler couldn't lower it), the VM hands the call
// to the bridge. The bridge:
//
//  1. Looks up the function value in the entry module's binding map.
//  2. Converts the bridged VM-side arguments to evaluator values.
//  3. Calls the evaluator's CallValueN with the converted args.
//  4. Converts the result back to a VM value and returns it.
//
// The bridge supports M-BYTECODE-VM Tier-1 value shapes (Int, Float, Bool,
// Unit, String, List, Tuple, Record). ADTs require a tag↔ctor-name registry
// that the bridge does not yet hold; passing an ADT across the boundary
// returns a clear error rather than silently mis-converting. Closures are
// likewise out of scope for M3 — the bridge will land them in M-BYTECODE-2E
// alongside effect-trap callbacks.
//
// Per M-BYTECODE-VM §11, this type lives outside internal/vm (here in the
// runner, formerly cmd/ailang) because the VM package must not import
// internal/eval. The vm.EvalInterop interface is the seam.
type bytecodeBridge struct {
	rt        *runtime.ModuleRuntime
	inst      *runtime.ModuleInstance
	evaluator *eval.CoreEvaluator
}

// newBytecodeBridge constructs a bridge bound to the given runtime, entry
// module instance, and evaluator. All three are required — the bridge
// returns errors rather than panicking if any is nil at call time.
func newBytecodeBridge(rt *runtime.ModuleRuntime, inst *runtime.ModuleInstance, evaluator *eval.CoreEvaluator) *bytecodeBridge {
	return &bytecodeBridge{rt: rt, inst: inst, evaluator: evaluator}
}

// CallEvalFunc resolves the named function in the entry module instance and
// dispatches it through the evaluator with bridged arguments. It implements
// vm.EvalInterop.
func (b *bytecodeBridge) CallEvalFunc(name string, args []bytecode.Value) (bytecode.Value, error) {
	if b == nil || b.inst == nil || b.evaluator == nil {
		return bytecode.Value{}, fmt.Errorf("bridge not fully initialized")
	}
	// Canonical names ("module/path.name") may refer to functions in a
	// different module than the entry instance. Resolve in this order:
	//   1. Try the entry module's binding map with the full name (legacy).
	//   2. If the name contains a "." split at the LAST dot and look up the
	//      function in the named module's instance via the runtime.
	//   3. Fall back to resolving the bare tail name in the entry instance.
	//
	// GetBinding (not GetExport) so we can call private functions too — the
	// EvalOnly stub may be a helper that is only reachable from inside the
	// module.
	fnVal, err := b.resolveFunc(name)
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("resolve %q: %w", name, err)
	}

	bridgedArgs := make([]eval.Value, len(args))
	for i, a := range args {
		ev, convErr := vm.BytecodeToEval(a)
		if convErr != nil {
			return bytecode.Value{}, fmt.Errorf("arg %d: %w", i, convErr)
		}
		bridgedArgs[i] = ev
	}

	result, err := b.evaluator.CallValueN(fnVal, bridgedArgs)
	if err != nil {
		return bytecode.Value{}, err
	}

	return vm.EvalToBytecode(result)
}

// resolveFunc walks the name resolution fallback chain described in
// CallEvalFunc. Returns an error if no module instance in the runtime holds
// the binding under any of the tried spellings.
func (b *bytecodeBridge) resolveFunc(name string) (eval.Value, error) {
	// 1. Entry instance, full name — handles the legacy bare-name case and
	//    any function already bound under its canonical form.
	if v, err := b.inst.GetBinding(name); err == nil {
		return v, nil
	}

	// 2. Canonical "module/path.name" → look up that module's instance.
	if dot := strings.LastIndex(name, "."); dot > 0 {
		modPath := name[:dot]
		bare := name[dot+1:]
		if b.rt != nil {
			if modInst := b.rt.GetInstance(modPath); modInst != nil {
				if v, err := modInst.GetBinding(bare); err == nil {
					return v, nil
				}
			}
		}
		// 3. Fall back to the bare tail name in the entry instance (e.g.
		//    cross-module helper registered bare during Phase 1 compat).
		if v, err := b.inst.GetBinding(bare); err == nil {
			return v, nil
		}
		return nil, fmt.Errorf("binding not found in module %q or entry instance", modPath)
	}

	return nil, fmt.Errorf("binding %q not found in entry instance", name)
}

package vm

import (
	"github.com/sunholo-data/ailang/internal/bytecode"
)

// EvalInterop is the VM-side view of the evaluator. The VM uses it to dispatch
// function calls whose target prototype is marked EvalOnly: the function exists
// in the program but the bytecode compiler couldn't compile it (e.g. it uses
// effects, dynamic dispatch, or unsupported builtins). At call time the VM
// hands the bridged arguments to the implementation, which is responsible for
// running the named function in the evaluator and returning a bridged result.
//
// Per M-BYTECODE-VM §11, the VM package does NOT import internal/eval. The
// implementation lives in cmd/ailang (or any caller that wires the VM) and
// performs bytecode.Value ↔ eval.Value conversion internally.
//
// CallEvalFunc is the only entry point. It takes a function name (matching
// bytecode.FuncPrototype.Name) and the already-evaluated VM arguments, and
// returns the function's result as a VM value or a non-nil error.
//
// Errors returned from CallEvalFunc are wrapped into VMError by the VM with
// the call site's source location, so the implementation does not need to
// add file/line info itself.
type EvalInterop interface {
	CallEvalFunc(name string, args []bytecode.Value) (bytecode.Value, error)
}

package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// Closure calls from Go (CallClosure, used by HOF builtins), and
// over-application: calling a closure with more arguments than it declares.
//
// The type checker accepts a curried lambda where a multi-parameter function
// is expected, so `foldl(\acc. \x. acc + x, 0, xs)` hands foldl a one-parameter
// closure that it calls with two arguments. The evaluator applies the first
// parameter's worth of arguments and then the rest to the function that comes
// back (eval_operations.go, "Apply remaining args to the intermediate
// result"). The VM refused with `expects 1 args, got 2`, and under --bytecode
// that arrived after earlier output, so the run could not be replayed and
// failed. overApply gives the VM the evaluator's rule. Calling with FEWER
// arguments than declared is an error on both engines and stays one.

// overApply calls closure, which declares fewer parameters than len(args):
// the first NumParams arguments go to closure, the rest to its result.
func (vm *VM) overApply(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error) {
	proto := closure.AsClosure().Proto.(*bytecode.FuncPrototype)
	n := int(proto.NumParams)
	head, err := vm.CallClosure(closure, args[:n])
	if err != nil {
		return bytecode.Value{}, err
	}
	if head.Tag != bytecode.TagClosure {
		return bytecode.Value{}, fmt.Errorf("%s expects %d args, got %d (intermediate result is not a function)", proto.Name, n, len(args))
	}
	return vm.CallClosure(head, args[n:])
}

// frameArgs copies the n call arguments that follow the callee slot at base.
// The copy matters: the nested calls overApply makes must not see a register
// the current frame later overwrites.
func frameArgs(frame *Frame, base uint8, n int) []bytecode.Value {
	args := make([]bytecode.Value, n)
	copy(args, frame.Regs[int(base)+1:int(base)+1+n])
	return args
}

// CallClosure invokes a closure value with the given arguments and returns
// the result. Used by HOF builtins to call their function arguments.
// This pushes a new frame onto the VM stack, runs it, and returns.
func (vm *VM) CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error) {
	if closure.Tag != bytecode.TagClosure {
		return bytecode.Value{}, fmt.Errorf("CallClosure: expected closure, got %s", closure.Tag)
	}
	c := closure.AsClosure()
	proto, ok := c.Proto.(*bytecode.FuncPrototype)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("CallClosure: non-FuncPrototype")
	}
	if len(args) > int(proto.NumParams) {
		return vm.overApply(closure, args)
	}
	if int(proto.NumParams) != len(args) {
		return bytecode.Value{}, fmt.Errorf("CallClosure: %s expects %d args, got %d",
			proto.Name, proto.NumParams, len(args))
	}
	if proto.EvalOnly {
		if vm.Interop == nil {
			return bytecode.Value{}, fmt.Errorf("CallClosure: %s is evaluator-only (%s) but no interop bridge",
				proto.Name, proto.EvalReason)
		}
		return vm.Interop.CallEvalFunc(proto.Name, args)
	}
	if len(vm.Stack) >= vm.MaxStack {
		return bytecode.Value{}, ErrStackOverflow
	}
	// Push frame with Caller=nil so OpReturn returns the value directly
	// to vm.run, which returns it to us. The frame comes from the free list
	// and goes back to it on RETURN: a HOF builtin calls this once per
	// element, so it must not allocate (#1501).
	base := len(vm.Stack)
	frame := vm.acquireFrame(proto, 0, nil)
	copy(frame.Regs, args)
	for i, cap := range c.Captures {
		frame.Regs[int(proto.NumParams)+i] = cap
	}
	vm.Stack = append(vm.Stack, frame)
	result, err := vm.run(frame)
	if err != nil {
		// A fault unwinds with its frames still pushed; drop them so the
		// caller's stack depth is what it was before this call. They are
		// not recycled (releaseFrame is only for frames RETURN popped).
		clear(vm.Stack[base:])
		vm.Stack = vm.Stack[:base]
	}
	return result, err
}

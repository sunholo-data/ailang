// Package vm implements the AILANG bytecode register virtual machine.
//
// The VM is intentionally isolated from the rest of the compiler. Per the
// M-BYTECODE-VM design doc §11, this package imports only internal/bytecode.
// It does NOT import internal/eval, internal/gen/stmt, internal/lower, or
// internal/core. The VM ↔ evaluator boundary (Phase 2E) will be added via a
// narrow interface, not a direct dependency.
//
// Phase 2B scope: register frames, dispatch loop, all non-effectful opcodes.
// BUILTIN_TRAP and EFFECT_TRAP return clear "not implemented" errors.
package vm

import "github.com/sunholo-data/ailang/internal/bytecode"

// Frame is a single call activation. Frames are allocated on each CALL and
// reused on TAIL_CALL (the register slab is resized in place if the callee
// has a different register count).
type Frame struct {
	// Proto is the function being executed.
	Proto *bytecode.FuncPrototype

	// IP is the index of the next instruction to execute in Proto.Instructions.
	IP int

	// Regs is this frame's register slab. Sized to Proto.NumRegs at creation.
	Regs []bytecode.Value

	// ReturnReg is the register in the *caller's* frame where this frame's
	// return value should be written when this frame returns. Ignored for
	// the entry frame.
	ReturnReg uint8

	// Caller links to the parent frame, or nil for the entry frame.
	Caller *Frame
}

// acquireFrame returns a frame for proto, recycling one from the VM's free
// list when possible (M-FOLDL-CONS-COST-MODEL Phase 3, #1501). HOF builtins
// re-enter the VM once per element through CallClosure, so a fresh Frame plus
// register slab per call was three allocations per element of every
// map/filter/foldl. Pooled frames arrive with every register zeroed (see
// releaseFrame), the same state newFrame's make() gives.
//
// The pool is a LIFO free list, never a single scratch frame: a callback that
// itself calls a HOF builtin holds its frame live while the inner callbacks
// acquire and release theirs. It is per-VM; a VM is single-goroutine.
func (vm *VM) acquireFrame(proto *bytecode.FuncPrototype, returnReg uint8, caller *Frame) *Frame {
	n := len(vm.framePool)
	if n == 0 {
		return newFrame(proto, returnReg, caller)
	}
	f := vm.framePool[n-1]
	vm.framePool[n-1] = nil
	vm.framePool = vm.framePool[:n-1]
	f.Proto = proto
	f.IP = 0
	f.ReturnReg = returnReg
	f.Caller = caller
	if int(proto.NumRegs) > cap(f.Regs) {
		f.Regs = make([]bytecode.Value, proto.NumRegs)
	} else {
		f.Regs = f.Regs[:proto.NumRegs]
	}
	return f
}

// releaseFrame returns a frame that has been popped from vm.Stack, and that
// nothing references any more, to the free list. Only the RETURN paths call
// it: a frame abandoned by an error is left to the GC, so a frame is never
// pooled while a caller link or error path can still reach it.
//
// Registers are cleared here rather than on acquire so a pooled frame never
// pins a value (a large list in a dead register would otherwise stay live).
// The whole capacity is cleared, not just len(Regs): a TAIL_CALL to a smaller
// callee (resetForTailCall) shortens the slab and leaves the old values past len.
// The pool never holds more frames than MaxStack, the most that can be live.
func (vm *VM) releaseFrame(f *Frame) {
	clear(f.Regs[:cap(f.Regs)])
	f.Proto = nil
	f.Caller = nil
	if len(vm.framePool) < vm.MaxStack {
		vm.framePool = append(vm.framePool, f)
	}
}

// newFrame allocates a frame for the given prototype.
func newFrame(proto *bytecode.FuncPrototype, returnReg uint8, caller *Frame) *Frame {
	return &Frame{
		Proto:     proto,
		IP:        0,
		Regs:      make([]bytecode.Value, proto.NumRegs),
		ReturnReg: returnReg,
		Caller:    caller,
	}
}

// resetForTailCall reconfigures this frame to run proto for TAIL_CALL. The
// argc arguments at Regs[argStart:] move to the front (copy is overlap-safe),
// every other register up to NumRegs is zeroed, and the captures follow the
// parameters. The caller's IP/ReturnReg/Caller are preserved on purpose so
// the tail-called function returns directly to the original caller.
//
// The end state is exactly that of zeroing the whole slab and then writing
// the arguments and captures, without stashing the arguments elsewhere or
// zeroing their registers twice (M-VM-QUICK-WINS W4).
func (f *Frame) resetForTailCall(proto *bytecode.FuncPrototype, argStart, argc int, caps []bytecode.Value) {
	n := int(proto.NumRegs)
	if n > cap(f.Regs) {
		regs := make([]bytecode.Value, n)
		copy(regs, f.Regs[argStart:argStart+argc])
		f.Regs = regs
	} else {
		copy(f.Regs[:argc], f.Regs[argStart:argStart+argc])
		f.Regs = f.Regs[:n]
		clear(f.Regs[argc:])
	}
	f.Proto = proto
	f.IP = 0
	for i, c := range caps {
		f.Regs[int(proto.NumParams)+i] = c
	}
}

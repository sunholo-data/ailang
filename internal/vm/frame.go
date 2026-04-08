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

import "github.com/sunholo/ailang/internal/bytecode"

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

// reuseFor reconfigures this frame to execute proto, resizing Regs if needed.
// Used by TAIL_CALL — the caller's IP/ReturnReg/Caller are preserved on
// purpose so the tail-called function returns directly to the original caller.
func (f *Frame) reuseFor(proto *bytecode.FuncPrototype) {
	f.Proto = proto
	f.IP = 0
	if int(proto.NumRegs) > cap(f.Regs) {
		f.Regs = make([]bytecode.Value, proto.NumRegs)
	} else {
		f.Regs = f.Regs[:proto.NumRegs]
		// Zero the slab so a tail call doesn't observe stale values from the
		// previous activation. (Argument registers are written by the caller
		// before the jump, so we zero everything first then expect arg writes.)
		for i := range f.Regs {
			f.Regs[i] = bytecode.Value{}
		}
	}
}

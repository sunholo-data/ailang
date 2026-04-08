package compiler

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// compileCall lowers a function call.
//
// Calling convention (matches the Phase 2B VM):
//   - Allocate a contiguous block: [callee, arg0, arg1, ..., argN-1]
//   - For top-level functions referenced by name: emit OpClosure into the
//     callee slot to materialize a closure value pointing at the prototype.
//   - For first-class function values (a VarRef to a parameter or let-bound
//     variable holding a function): MOVE the closure value into the callee slot.
//   - Emit each argument expression directly into its target slot via
//     compileExprIntoSlot.
//   - Emit OpCall callee, B=arity, C=1 (one result; the callee slot becomes
//     the result register on return).
func (fc *funcCompiler) compileCall(e stmt.Call) (uint8, error) {
	arity := len(e.Args)

	// Resolve the callee. Two cases:
	//   1. Top-level function reference (VarRef name matches funcIdx, and not
	//      a local). Emit OpClosure for it. Also handle GlobalRef.
	//   2. First-class value (VarRef to a local/param, Lambda, etc.). Emit a
	//      MOVE from its source register into the callee slot.
	calleeKind, calleeName, err := fc.classifyCallee(e.Func)
	if err != nil {
		return 0, err
	}

	// Allocate the contiguous frame: callee + args.
	base, err := fc.regs.allocContig(arity + 1)
	if err != nil {
		return 0, err
	}

	// Materialize callee into base.
	if err := fc.materializeCallee(base, calleeKind, calleeName, e.Func); err != nil {
		return 0, err
	}

	// Materialize args into base+1, base+2, ...
	for i, arg := range e.Args {
		if err := fc.compileExprIntoSlot(arg, base+uint8(i+1)); err != nil {
			return 0, err
		}
	}

	// CALL: A=base, B=arity, C=1 (single result lands back in `base`).
	fc.emit(bytecode.EncodeABC(bytecode.OpCall, base, uint8(arity), 1))

	// Free arg registers; the result occupies `base` and stays live.
	if arity > 0 {
		fc.regs.freeContig(base+1, arity)
	}
	return base, nil
}

// calleeKind classifies how to materialize a call's function expression.
type calleeKind int

const (
	calleeTopLevel calleeKind = iota // emit OpClosure with NestedProtos[idx]
	calleeValue                      // emit MOVE from a register
)

// classifyCallee inspects the Call.Func expression and decides whether the
// callee is a top-level prototype reference or a first-class value.
func (fc *funcCompiler) classifyCallee(e stmt.Expr) (calleeKind, string, error) {
	switch ex := e.(type) {
	case stmt.VarRef:
		// Local variable shadows take precedence — `apply(f, x)` where f is
		// a parameter must NOT resolve to a top-level function of the same name.
		if _, ok := fc.locals.lookup(ex.Name); ok {
			return calleeValue, ex.Name, nil
		}
		if _, ok := fc.funcIdx[ex.Name]; ok {
			return calleeTopLevel, ex.Name, nil
		}
		return 0, "", fmt.Errorf("compiler: call to unbound name %q", ex.Name)
	case stmt.GlobalRef:
		if _, ok := fc.funcIdx[ex.Name]; ok {
			return calleeTopLevel, ex.Name, nil
		}
		return 0, "", fmt.Errorf("compiler: call to unknown global %q", ex.Name)
	}
	// Lambda or other expressions: evaluate as a value.
	return calleeValue, "", nil
}

// materializeCallee writes the function value into the callee slot of a
// pre-allocated call frame.
func (fc *funcCompiler) materializeCallee(slot uint8, kind calleeKind, name string, raw stmt.Expr) error {
	if kind == calleeTopLevel {
		// Look up (or register) the prototype index in the current function's
		// NestedProtos table. The same top-level function may appear multiple
		// times in the same body, so dedupe.
		nestedIdx, err := fc.lookupNestedProto(fc.funcIdx[name])
		if err != nil {
			return err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpClosure, slot, nestedIdx))
		// OpClosure with NumCaptures=0 (true for top-level functions) consumes
		// no following pseudo-MOVEs, so this is a complete instruction.
		return nil
	}
	// First-class value: evaluate the expression and copy into slot.
	src, err := fc.compileExpr(raw)
	if err != nil {
		return err
	}
	if src != slot {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, slot, src, 0))
	}
	if !fc.isPinned(src) && src != slot {
		fc.regs.freeTemp(src)
	}
	return nil
}

// lookupNestedProto returns the index of imageProtoIdx in the current function's
// NestedProtos table, appending if necessary. Returns the local index suitable
// for an OpClosure Bx field.
func (fc *funcCompiler) lookupNestedProto(imageProtoIdx int) (uint16, error) {
	for i, idx := range fc.proto.NestedProtos {
		if idx == imageProtoIdx {
			if i > 0xFFFF {
				return 0, fmt.Errorf("nested proto table overflow")
			}
			return uint16(i), nil
		}
	}
	fc.proto.NestedProtos = append(fc.proto.NestedProtos, imageProtoIdx)
	idx := len(fc.proto.NestedProtos) - 1
	if idx > 0xFFFF {
		return 0, fmt.Errorf("nested proto table overflow")
	}
	return uint16(idx), nil
}

// compileExprIntoSlot evaluates e and ensures the result lands in dst. If
// the natural register for the result is already dst (rare), no copy is
// emitted; otherwise an OpMove copies the value.
func (fc *funcCompiler) compileExprIntoSlot(e stmt.Expr, dst uint8) error {
	src, err := fc.compileExpr(e)
	if err != nil {
		return err
	}
	if src != dst {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, dst, src, 0))
	}
	if !fc.isPinned(src) && src != dst {
		fc.regs.freeTemp(src)
	}
	return nil
}

// --- Tail call detection ----------------------------------------------------
//
// A ReturnStmt whose Value is a Call to a known top-level function is lowered
// as OpTailCall, which reuses the current frame instead of pushing a new one.
// Tail call eligibility is conservative:
//   - The call's callee must be a known top-level function (calleeTopLevel),
//     since the VM's TAIL_CALL semantics expect a closure value as A.
//   - First-class function values (calleeValue) also work because the VM
//     reads R[A] as a closure regardless.

// compileReturnExpr lowers an expression in tail position. The result is
// guaranteed to leave the function via OpReturn or OpTailCall — no instruction
// after this point in the current control-flow path will execute.
//
// Tail-recursive cases:
//   - Call         → emit OpTailCall
//   - IfExpr       → recursively compile each branch in tail position
//   - everything else → compile as a value and emit OpReturn
func (fc *funcCompiler) compileReturnExpr(e stmt.Expr) error {
	switch ex := e.(type) {
	case stmt.Call:
		_, err := fc.compileTailReturn(ex)
		return err
	case stmt.IfExpr:
		// Tail-position if: each branch returns directly. We do NOT need a
		// jump-over-else because both branches terminate the function.
		condReg, err := fc.compileExpr(ex.Cond)
		if err != nil {
			return err
		}
		condIsTemp := !fc.isPinned(condReg)
		jumpToElse := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, condReg)
		if condIsTemp {
			fc.regs.freeTemp(condReg)
		}
		// Then branch in tail position.
		if err := fc.compileReturnExpr(ex.Then); err != nil {
			return err
		}
		fc.patchJump(jumpToElse)
		// Else branch in tail position.
		return fc.compileReturnExpr(ex.Else)
	}
	r, err := fc.compileExpr(e)
	if err != nil {
		return err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpReturn, r, 0, 0))
	return nil
}

func (fc *funcCompiler) compileTailReturn(call stmt.Call) (bool, error) {
	arity := len(call.Args)
	calleeKind, calleeName, err := fc.classifyCallee(call.Func)
	if err != nil {
		return false, err
	}
	base, err := fc.regs.allocContig(arity + 1)
	if err != nil {
		return false, err
	}
	if err := fc.materializeCallee(base, calleeKind, calleeName, call.Func); err != nil {
		return false, err
	}
	for i, arg := range call.Args {
		if err := fc.compileExprIntoSlot(arg, base+uint8(i+1)); err != nil {
			return false, err
		}
	}
	// TAIL_CALL: A=base, B=arity. The VM reuses the current frame.
	fc.emit(bytecode.EncodeABC(bytecode.OpTailCall, base, uint8(arity), 0))
	// Note: no register cleanup — the frame is gone after this instruction.
	return true, nil
}

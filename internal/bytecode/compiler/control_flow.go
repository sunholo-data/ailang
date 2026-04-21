package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// emitJumpPlaceholder writes a jump (or jump-if-false) instruction with a
// placeholder offset and returns the index of the emitted instruction. Use
// patchJump to fix the offset once the target is known.
//
// op must be either OpJump or OpJumpIfFalse. condReg is ignored for OpJump.
func (fc *funcCompiler) emitJumpPlaceholder(op bytecode.OpCode, condReg uint8) int {
	// SBx=0 means "jump to next instruction" — a no-op offset that we'll
	// overwrite when patchJump runs.
	switch op {
	case bytecode.OpJump:
		fc.emit(bytecode.EncodeASBx(op, 0, 0))
	case bytecode.OpJumpIfFalse:
		fc.emit(bytecode.EncodeASBx(op, condReg, 0))
	default:
		panic(fmt.Sprintf("emitJumpPlaceholder: bad op %v", op))
	}
	return len(fc.proto.Instructions) - 1
}

// patchJump rewrites the SBx field of the jump instruction at jumpIdx so it
// targets the *current* end of the instruction stream. Per the VM dispatch
// loop, the SBx is interpreted as `IP += 1 + SBx` where IP is the jump's own
// address — so SBx = (target - jumpIdx - 1).
func (fc *funcCompiler) patchJump(jumpIdx int) {
	target := len(fc.proto.Instructions)
	offset := target - jumpIdx - 1
	old := fc.proto.Instructions[jumpIdx]
	op := old.Op()
	a := old.A()
	fc.proto.Instructions[jumpIdx] = bytecode.EncodeASBx(op, a, offset)
}

// --- IfStmt -----------------------------------------------------------------

func (fc *funcCompiler) compileIfStmt(s stmt.IfStmt) error {
	condReg, err := fc.compileExpr(s.Cond)
	if err != nil {
		return err
	}
	condIsTemp := !fc.isPinned(condReg)

	// JUMP_IF_FALSE → else branch
	jumpToElse := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, condReg)
	if condIsTemp {
		fc.regs.freeTemp(condReg)
	}

	// Then branch.
	fc.locals.push()
	for _, ts := range s.Then {
		if err := fc.compileStmt(ts); err != nil {
			return err
		}
	}
	fc.locals.pop()

	if len(s.Else) == 0 {
		fc.patchJump(jumpToElse)
		return nil
	}

	// Skip the else branch from the end of the then branch.
	jumpOverElse := fc.emitJumpPlaceholder(bytecode.OpJump, 0)
	fc.patchJump(jumpToElse)

	fc.locals.push()
	for _, es := range s.Else {
		if err := fc.compileStmt(es); err != nil {
			return err
		}
	}
	fc.locals.pop()

	fc.patchJump(jumpOverElse)
	return nil
}

// --- IfExpr -----------------------------------------------------------------
//
// Lowered to: evaluate cond → JUMP_IF_FALSE to else_label →
//
//	eval then → MOVE result → JUMP to end_label →
//	else_label: eval else → MOVE result → end_label.
//
// The result lives in a freshly-allocated register (returned).
func (fc *funcCompiler) compileIfExpr(e stmt.IfExpr) (uint8, error) {
	condReg, err := fc.compileExpr(e.Cond)
	if err != nil {
		return 0, err
	}
	condIsTemp := !fc.isPinned(condReg)
	jumpToElse := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, condReg)
	if condIsTemp {
		fc.regs.freeTemp(condReg)
	}

	// Allocate the result register up front so both branches write to the same slot.
	result, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}

	// Then branch.
	thenReg, err := fc.compileExpr(e.Then)
	if err != nil {
		return 0, err
	}
	if thenReg != result {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, result, thenReg, 0))
	}
	if !fc.isPinned(thenReg) && thenReg != result {
		fc.regs.freeTemp(thenReg)
	}
	jumpOverElse := fc.emitJumpPlaceholder(bytecode.OpJump, 0)

	// Else branch.
	fc.patchJump(jumpToElse)
	elseReg, err := fc.compileExpr(e.Else)
	if err != nil {
		return 0, err
	}
	if elseReg != result {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, result, elseReg, 0))
	}
	if !fc.isPinned(elseReg) && elseReg != result {
		fc.regs.freeTemp(elseReg)
	}

	fc.patchJump(jumpOverElse)
	return result, nil
}

// --- Boolean short-circuit --------------------------------------------------
//
// AND lowered as:
//   eval lhs → result_reg
//   JUMP_IF_FALSE result_reg → end (lhs already in result_reg, false)
//   eval rhs → result_reg
//   end:
//
// OR lowered as:
//   eval lhs → result_reg
//   NOT lhs → tmp        (we need JUMP_IF_TRUE; synthesized via NOT + JUMP_IF_FALSE)
//   JUMP_IF_FALSE tmp → end (lhs already in result_reg, true)
//   eval rhs → result_reg
//   end:
//
// Both branches write the final value to result_reg.

func (fc *funcCompiler) compileShortCircuit(e stmt.BinOp) (uint8, error) {
	switch e.Op {
	case stmt.OpAnd:
		return fc.compileAnd(e)
	case stmt.OpOr:
		return fc.compileOr(e)
	}
	return 0, fmt.Errorf("compiler: not a short-circuit op: %v", e.Op)
}

func (fc *funcCompiler) compileAnd(e stmt.BinOp) (uint8, error) {
	// Evaluate LHS into a fresh result register.
	result, err := fc.compileExprToFresh(e.Left)
	if err != nil {
		return 0, err
	}
	// If false, skip RHS — result already holds the false value.
	jumpEnd := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, result)

	// RHS evaluated into the same result slot.
	rhs, err := fc.compileExpr(e.Right)
	if err != nil {
		return 0, err
	}
	if rhs != result {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, result, rhs, 0))
		if !fc.isPinned(rhs) {
			fc.regs.freeTemp(rhs)
		}
	}
	fc.patchJump(jumpEnd)
	return result, nil
}

func (fc *funcCompiler) compileOr(e stmt.BinOp) (uint8, error) {
	result, err := fc.compileExprToFresh(e.Left)
	if err != nil {
		return 0, err
	}
	// We need "skip RHS if LHS is true". JUMP_IF_FALSE jumps when condition
	// is false, so synthesize: NOT(lhs) → tmp; JUMP_IF_FALSE tmp → end.
	tmp, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpNot, tmp, result, 0))
	jumpEnd := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, tmp)
	fc.regs.freeTemp(tmp)

	rhs, err := fc.compileExpr(e.Right)
	if err != nil {
		return 0, err
	}
	if rhs != result {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, result, rhs, 0))
		if !fc.isPinned(rhs) {
			fc.regs.freeTemp(rhs)
		}
	}
	fc.patchJump(jumpEnd)
	return result, nil
}

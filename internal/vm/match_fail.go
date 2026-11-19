package vm

import (
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// matchFail executes OpMatchFail: no arm of a match matched the scrutinee
// R[A]. It raises the same RT002 error, with the same text, as the evaluator
// (M-V1-MATCH-EXHAUSTIVENESS M0). The position is the "file:line:col"
// constant Bx; when the lowering knew none it falls back to the frame's
// file:line. Before this opcode a match that fell off its last arm yielded
// Unit and the program failed later somewhere unrelated — or exited 0.
func (vm *VM) matchFail(frame *Frame, inst bytecode.Instruction) *VMError {
	mf := &ailerrors.MatchFailureError{}
	scrut := frame.Regs[inst.A()]
	switch scrut.Tag {
	case bytecode.TagADT:
		mf.Ctor = scrut.AsADT().Ctor
	case bytecode.TagList:
		mf.Shape = ailerrors.ListShape(len(scrut.AsList()))
	default:
		mf.Shape = strings.ToLower(scrut.Tag.String())
	}
	e := vm.errAt(frame, "", inst)
	if pos, ok := frame.Proto.LookupConstant(int(inst.Bx()), vm.Image); ok && pos.Tag == bytecode.TagString && pos.AsString() != "" {
		mf.Pos = pos.AsString()
	} else if e.File != "" && e.Line > 0 {
		mf.Pos = e.File + ":" + strconv.Itoa(e.Line)
	}
	e.Msg = mf.Error()
	e.Cause = mf
	return e
}

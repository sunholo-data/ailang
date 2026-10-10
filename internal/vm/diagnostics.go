package vm

import "github.com/sunholo-data/ailang/internal/bytecode"

// errWrap is errAt for an instruction that failed with err: the message is
// prefix + err's text and err is kept as the VMError's Cause.
func (vm *VM) errWrap(frame *Frame, prefix string, err error, inst bytecode.Instruction) *VMError {
	e := vm.errAt(frame, prefix+err.Error(), inst)
	e.Cause = err
	return e
}

// errAt builds a VMError with source-location info from the current frame.
func (vm *VM) errAt(frame *Frame, msg string, inst bytecode.Instruction) *VMError {
	line := 0
	if frame.IP >= 0 && frame.IP < len(frame.Proto.LineInfo) {
		line = frame.Proto.LineInfo[frame.IP]
	}
	return &VMError{
		Msg:      msg,
		Func:     frame.Proto.Name,
		File:     frame.Proto.File,
		Line:     line,
		IP:       frame.IP,
		OpString: inst.Op().String(),
	}
}

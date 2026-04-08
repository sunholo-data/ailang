package compiler

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// BuiltinTable is the canonical, source-ordered table of pure builtins
// reachable from the Phase 2C golden corpus. Indices in this table become
// the B field of OpBuiltinCall and are interpreted by the VM's builtin
// dispatch table (internal/vm/builtins.go).
//
// Adding a builtin is a two-step process:
//  1. Append its name here.
//  2. Add a matching entry in vm.BuiltinTable.
//
// The two tables MUST stay in lockstep — there is a runtime sanity check
// at VM startup that the lengths agree.
var BuiltinTable = []string{
	"_show",
	"_len",
	"_list_get",
	"_list_tail",
	"_concat_String",
}

var builtinIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(BuiltinTable))
	for i, name := range BuiltinTable {
		m[name] = uint8(i)
	}
	return m
}()

// compileBuiltinCall lowers a stmt.BuiltinCall into either OpBuiltinCall
// (for builtins listed in BuiltinTable) or OpBuiltinTrap (for any other
// effectful/unsupported builtin — these will be wired through the
// evaluator in Phase 2E).
func (fc *funcCompiler) compileBuiltinCall(e stmt.BuiltinCall) (uint8, error) {
	idx, isPure := builtinIndex[e.Name]
	n := len(e.Args)
	if n > 255 {
		return 0, fmt.Errorf("compiler: builtin %s has %d args, exceeds 255", e.Name, n)
	}

	// Allocate dst followed by n contiguous arg registers (matches OpMakeADT
	// layout: VM reads args from R[A+1..A+C]).
	block, err := fc.regs.allocContig(n + 1)
	if err != nil {
		return 0, err
	}
	dst := block
	for i, arg := range e.Args {
		if err := fc.compileExprIntoSlot(arg, dst+uint8(i+1)); err != nil {
			return 0, err
		}
	}
	if n > 0 {
		fc.regs.freeContig(dst+1, n)
	}
	if isPure {
		fc.emit(bytecode.EncodeABC(bytecode.OpBuiltinCall, dst, idx, uint8(n)))
	} else {
		// Effectful / not-yet-wired: emit a trap. The image needs an entry
		// in the trap name table — for Phase 2C we just stash the name as a
		// local constant and reference its index in Bx, with C=argc.
		// Leaves a clear runtime error for users that hit it.
		nameIdx, err := fc.addLocalConst(bytecode.NewString(e.Name))
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpBuiltinTrap, dst, nameIdx))
	}
	return dst, nil
}

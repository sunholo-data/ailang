package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// compileSwitch lowers a SwitchStmt over an ADT scrutinee.
//
// Strategy: GET_TAG into a tag register, then a flat chain of EQ + JUMP_IF_FALSE
// per case. Each case body is preceded by GET_FIELD instructions extracting
// the bindings from the scrutinee. Cases jump over each other to a shared
// "end" label.
//
// This is a decision *list* (linear scan), not a decision *table*. The corpus
// has at most 4 cases per switch so the linear scan is fine.
func (fc *funcCompiler) compileSwitch(s stmt.SwitchStmt) error {
	scrutReg, err := fc.compileExpr(s.Scrutinee)
	if err != nil {
		return err
	}
	scrutIsTemp := !fc.isPinned(scrutReg)

	// Get the ADT info to translate tag names to ordinals. The ADT is the
	// one the type checker stamped on the case patterns (lower.patternADTName);
	// a missing or unknown one is refused, never inferred from the tags (#2303).
	info, err := fc.adtInfo(s.ADTName, "switch")
	if err != nil {
		return err
	}

	// Compute the tag once.
	tagReg, err := fc.regs.allocTemp()
	if err != nil {
		return err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpGetTag, tagReg, scrutReg, 0))

	// We'll patch jumps that point to the end of the entire switch statement
	// after each case body completes.
	var endJumps []int

	for _, c := range s.Cases {
		ordinal, ok := info.tagOrdinal[c.Tag]
		if !ok {
			return fmt.Errorf("compiler: unknown tag %s.%s in switch case", s.ADTName, c.Tag)
		}
		// Compare tag against ordinal.
		ordConstIdx, err := fc.addLocalConst(bytecode.NewInt(int64(ordinal)))
		if err != nil {
			return err
		}
		ordReg, err := fc.regs.allocTemp()
		if err != nil {
			return err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, ordReg, ordConstIdx))
		cmpReg, err := fc.regs.allocTemp()
		if err != nil {
			return err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpEq, cmpReg, tagReg, ordReg))
		fc.regs.freeTemp(ordReg)

		// JUMP_IF_FALSE to next case test.
		nextCaseJump := fc.emitJumpPlaceholder(bytecode.OpJumpIfFalse, cmpReg)
		fc.regs.freeTemp(cmpReg)

		// Inside the case: extract bindings via GET_FIELD, scoped to this case.
		// Using bindScoped so registers are recycled when the scope pops.
		fc.locals.push()
		for _, b := range c.Bindings {
			if b.FieldIndex > 255 {
				fc.locals.pop()
				return fmt.Errorf("compiler: binding field index %d exceeds 255", b.FieldIndex)
			}
			bindReg, err := fc.regs.allocPinned()
			if err != nil {
				fc.locals.pop()
				return err
			}
			fc.emit(bytecode.EncodeABC(bytecode.OpGetField, bindReg, scrutReg, uint8(b.FieldIndex)))
			fc.locals.bindScoped(b.Name, bindReg)
		}
		// Compile the case body.
		if err := fc.compileStmts(inlineCtorTemps(c.Body)); err != nil {
			fc.locals.pop()
			return err
		}
		fc.locals.pop()

		// Jump over remaining cases to end-of-switch.
		endJumps = append(endJumps, fc.emitJumpPlaceholder(bytecode.OpJump, 0))

		// Patch the JUMP_IF_FALSE to land here (next case test).
		fc.patchJump(nextCaseJump)
	}

	// Default branch.
	if len(s.Default) > 0 {
		fc.locals.push()
		if err := fc.compileStmts(inlineCtorTemps(s.Default)); err != nil {
			fc.locals.pop()
			return err
		}
		fc.locals.pop()
	}

	// Patch all "skip to end" jumps to land here.
	for _, j := range endJumps {
		fc.patchJump(j)
	}

	fc.regs.freeTemp(tagReg)
	if scrutIsTemp {
		fc.regs.freeTemp(scrutReg)
	}
	return nil
}

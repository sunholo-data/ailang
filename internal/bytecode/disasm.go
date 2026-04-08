package bytecode

import (
	"fmt"
	"strings"
)

// Disassemble produces a human-readable listing of an entire BytecodeImage:
// header (entry point, totals), constant pool, globals, and one block per
// prototype with annotated instructions.
//
// The output is intentionally text-only and stable enough for golden tests.
// It is the primary debugging surface for Phase 2D — `ailang disasm` calls
// directly into this function.
func Disassemble(img *BytecodeImage) string {
	if img == nil {
		return "<nil image>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== BytecodeImage ===\n")
	fmt.Fprintf(&b, "Prototypes: %d\n", len(img.Prototypes))
	fmt.Fprintf(&b, "Constants:  %d\n", len(img.Constants))
	fmt.Fprintf(&b, "Globals:    %d\n", len(img.Globals))
	if img.EntryPoint >= 0 && img.EntryPoint < len(img.Prototypes) {
		fmt.Fprintf(&b, "Entry:      %d (%s)\n", img.EntryPoint, img.Prototypes[img.EntryPoint].Name)
	} else {
		fmt.Fprintf(&b, "Entry:      <none>\n")
	}

	if len(img.Constants) > 0 {
		fmt.Fprintf(&b, "\n--- Constant Pool ---\n")
		for i, v := range img.Constants {
			fmt.Fprintf(&b, "  #%d = %s\n", i, formatConstant(v))
		}
	}

	if len(img.Globals) > 0 {
		fmt.Fprintf(&b, "\n--- Globals ---\n")
		for i, v := range img.Globals {
			fmt.Fprintf(&b, "  G%d = %s\n", i, formatConstant(v))
		}
	}

	// Counts for the summary header so it's easy to see at a glance how many
	// functions are running on the VM vs bridged back to the evaluator.
	evalOnly := 0
	for _, p := range img.Prototypes {
		if p.EvalOnly {
			evalOnly++
		}
	}
	if evalOnly > 0 {
		fmt.Fprintf(&b, "EvalOnly:   %d / %d\n", evalOnly, len(img.Prototypes))
	}

	for i, p := range img.Prototypes {
		if p.EvalOnly {
			fmt.Fprintf(&b, "\n--- Prototype %d: %s [EvalOnly: %s] ---\n", i, p.Name, p.EvalReason)
		} else {
			fmt.Fprintf(&b, "\n--- Prototype %d: %s ---\n", i, p.Name)
		}
		writePrototype(&b, p, img)
	}
	return b.String()
}

// DisassembleFunc disassembles a single FuncPrototype against an image (for
// constant resolution). Used by tests that don't want a full image dump.
func DisassembleFunc(p *FuncPrototype, img *BytecodeImage) string {
	if p == nil {
		return "<nil proto>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Prototype: %s\n", p.Name)
	writePrototype(&b, p, img)
	return b.String()
}

func writePrototype(b *strings.Builder, p *FuncPrototype, img *BytecodeImage) {
	fmt.Fprintf(b, "  params=%d  regs=%d  captures=%d  insns=%d\n",
		p.NumParams, p.NumRegs, p.NumCaptures, len(p.Instructions))
	if len(p.NestedProtos) > 0 {
		fmt.Fprintf(b, "  nested protos: %v\n", p.NestedProtos)
	}
	if len(p.Constants) > 0 {
		// Show local→pool index mapping. Useful when comparing two prototypes
		// that share constants from the same image pool.
		fmt.Fprintf(b, "  local consts: ")
		for i, poolIdx := range p.Constants {
			if i > 0 {
				fmt.Fprintf(b, " ")
			}
			fmt.Fprintf(b, "[%d→#%d]", i, poolIdx)
		}
		fmt.Fprintf(b, "\n")
	}

	hasLines := len(p.LineInfo) == len(p.Instructions)
	prevLine := -1
	for ip, inst := range p.Instructions {
		// Line marker (only when it changes, to keep output compact).
		if hasLines {
			line := p.LineInfo[ip]
			if line != 0 && line != prevLine {
				fmt.Fprintf(b, "  ; line %d\n", line)
				prevLine = line
			}
		}
		fmt.Fprintf(b, "  %04d  %s\n", ip, formatInstruction(inst, ip, p, img))
	}
}

// formatInstruction renders one instruction with operand annotations:
//   - LOAD_CONST: resolved constant value
//   - JUMP / JUMP_IF_FALSE: absolute target IP
//   - CLOSURE: nested prototype name
//   - BUILTIN_CALL: builtin name (when an index→name table is available)
//
// Falls back to the raw Instruction.String() if no annotation applies.
func formatInstruction(inst Instruction, ip int, p *FuncPrototype, img *BytecodeImage) string {
	op := inst.Op()
	switch op {
	case OpLoadConst:
		base := fmt.Sprintf("LOAD_CONST   r%d, #%d", inst.A(), inst.Bx())
		if v, ok := p.LookupConstant(int(inst.Bx()), img); ok {
			return fmt.Sprintf("%-30s ; %s", base, formatConstant(v))
		}
		return base

	case OpLoadGlobal:
		base := fmt.Sprintf("LOAD_GLOBAL  r%d, G%d", inst.A(), inst.Bx())
		if int(inst.Bx()) < len(img.Globals) {
			return fmt.Sprintf("%-30s ; %s", base, formatConstant(img.Globals[inst.Bx()]))
		}
		return base

	case OpJump:
		target := ip + 1 + inst.SBx()
		return fmt.Sprintf("JUMP         %+d           ; -> %04d", inst.SBx(), target)

	case OpJumpIfFalse:
		target := ip + 1 + inst.SBx()
		return fmt.Sprintf("JUMP_IF_FALSE r%d, %+d     ; -> %04d", inst.A(), inst.SBx(), target)

	case OpClosure:
		base := fmt.Sprintf("CLOSURE      r%d, P%d", inst.A(), inst.Bx())
		if int(inst.Bx()) < len(p.NestedProtos) {
			tblIdx := p.NestedProtos[inst.Bx()]
			if tblIdx >= 0 && tblIdx < len(img.Prototypes) {
				return fmt.Sprintf("%-30s ; %s", base, img.Prototypes[tblIdx].Name)
			}
		}
		return base

	case OpCall:
		base := fmt.Sprintf("CALL         r%d, args=%d, results=%d", inst.A(), inst.B(), inst.C())
		if name := lookupCalleeName(p, img, ip, inst.A()); name != "" {
			return fmt.Sprintf("%-30s ; %s", base, name)
		}
		return base

	case OpTailCall:
		base := fmt.Sprintf("TAIL_CALL    r%d, args=%d", inst.A(), inst.B())
		if name := lookupCalleeName(p, img, ip, inst.A()); name != "" {
			return fmt.Sprintf("%-30s ; %s", base, name)
		}
		return base

	case OpBuiltinCall:
		return fmt.Sprintf("BUILTIN_CALL r%d, builtin#%d, argc=%d", inst.A(), inst.Bx(), inst.C())

	case OpBuiltinTrap:
		base := fmt.Sprintf("BUILTIN_TRAP r%d, name#%d", inst.A(), inst.Bx())
		if v, ok := p.LookupConstant(int(inst.Bx()), img); ok {
			return fmt.Sprintf("%-30s ; %s", base, formatConstant(v))
		}
		return base

	case OpEffectTrap:
		return fmt.Sprintf("EFFECT_TRAP  r%d, effect#%d", inst.A(), inst.Bx())

	case OpMakeADT:
		return fmt.Sprintf("MAKE_ADT     r%d, tag=%d, fields=%d", inst.A(), inst.B(), inst.C())

	case OpMakeList, OpMakeTuple, OpMakeRecord:
		return fmt.Sprintf("%-12s r%d, src=r%d, count=%d", op, inst.A(), inst.B(), inst.C())

	case OpReturn, OpLoadNil:
		return fmt.Sprintf("%-12s r%d", op, inst.A())

	case OpMove, OpNeg, OpNot, OpGetTag:
		return fmt.Sprintf("%-12s r%d, r%d", op, inst.A(), inst.B())

	case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpEq, OpLt, OpLe, OpConcat, OpCons, OpGetIndex, OpGetField:
		return fmt.Sprintf("%-12s r%d, r%d, r%d", op, inst.A(), inst.B(), inst.C())
	}
	return inst.String()
}

// lookupCalleeName scans backwards from a CALL/TAIL_CALL instruction to find
// the most recent OpClosure that materialized the callee register, and returns
// the canonical name of the nested prototype. Returns "" if no such CLOSURE is
// found in the current prototype (e.g. first-class value calls or calls via
// MOVE from a parameter/local).
//
// This makes CALL/TAIL_CALL disassembly self-describing even though the callee
// is technically runtime-resolved: `CALL r12, args=2, results=1 ; std/io.println`.
// Added for M-BYTECODE-MULTIMODULE M2 (acceptance criterion #4).
func lookupCalleeName(p *FuncPrototype, img *BytecodeImage, callIP int, calleeReg uint8) string {
	for j := callIP - 1; j >= 0; j-- {
		prev := p.Instructions[j]
		if prev.Op() != OpClosure {
			// Any write to calleeReg from something other than OpClosure
			// (OpMove, OpGetField, etc.) means the callee is a first-class
			// value and we can't statically name it — give up.
			if writesToReg(prev, calleeReg) {
				return ""
			}
			continue
		}
		if prev.A() != calleeReg {
			continue
		}
		if int(prev.Bx()) >= len(p.NestedProtos) {
			return ""
		}
		tblIdx := p.NestedProtos[prev.Bx()]
		if tblIdx < 0 || tblIdx >= len(img.Prototypes) {
			return ""
		}
		return img.Prototypes[tblIdx].Name
	}
	return ""
}

// writesToReg reports whether inst writes its result to register r. Used by
// lookupCalleeName to detect "the callee slot was overwritten by a non-closure
// instruction" and stop the back-scan.
func writesToReg(inst Instruction, r uint8) bool {
	if inst.A() != r {
		return false
	}
	switch inst.Op() {
	case OpJump, OpJumpIfFalse, OpReturn, OpCall, OpTailCall:
		return false
	}
	return true
}

// formatConstant produces a short representation of a constant pool entry,
// suitable for inline annotations in the disassembly listing.
func formatConstant(v Value) string {
	switch v.Tag {
	case TagInt:
		return fmt.Sprintf("int %d", v.Int)
	case TagFloat:
		return fmt.Sprintf("float %g", v.Flt)
	case TagBool:
		return fmt.Sprintf("bool %t", v.Bool)
	case TagString:
		s := v.AsString()
		if len(s) > 40 {
			s = s[:37] + "..."
		}
		return fmt.Sprintf("string %q", s)
	case TagUnit:
		return "unit"
	case TagList:
		return fmt.Sprintf("list[%d]", len(v.AsList()))
	case TagTuple:
		return fmt.Sprintf("tuple[%d]", len(v.AsTuple()))
	case TagADT:
		return fmt.Sprintf("adt(tag=%d)", v.Obj.(*ADTObj).Tag)
	default:
		return fmt.Sprintf("<%s>", v.Tag)
	}
}

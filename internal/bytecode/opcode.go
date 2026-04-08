// Package bytecode defines the AILANG bytecode instruction set, image format,
// and Value type used by the register VM (internal/vm).
//
// This package is the compilation target for Statement IR (Phase 2C) and the
// input to the VM dispatch loop (Phase 2B). Per the M-BYTECODE-VM design doc
// (§11), this package must NOT import internal/vm. The import direction is
// always vm → bytecode.
//
// Instructions are 32-bit words with two layouts:
//
//	┌──────────┬──────────┬──────────┬──────────┐
//	│  OpCode  │    A     │    B     │    C     │   ABC form
//	│  8 bits  │  8 bits  │  8 bits  │  8 bits  │
//	└──────────┴──────────┴──────────┴──────────┘
//
//	┌──────────┬──────────┬─────────────────────┐
//	│  OpCode  │    A     │       Bx (16 bits)  │   ABx form
//	│  8 bits  │  8 bits  │                     │
//	└──────────┴──────────┴─────────────────────┘
//
// Bx is interpreted as unsigned (constant pool indices, prototype indices).
// SBx is the same field interpreted as a signed offset (jump targets), with
// bias 0x8000 — i.e. SBx = int(Bx) - 0x8000, range [-32768, 32767].
package bytecode

import "fmt"

// OpCode is the 8-bit instruction tag.
type OpCode uint8

// Opcode definitions. Numeric values are part of the bytecode format and must
// remain stable; append new opcodes at the end. Opcodes correspond directly to
// the design doc M-BYTECODE-VM §4.2.
const (
	// Loads
	OpLoadConst  OpCode = iota // R[A] = Constants[Bx]                       (ABx)
	OpLoadNil                  // R[A] = Unit                                (A)
	OpMove                     // R[A] = R[B]                                (AB)
	OpLoadGlobal               // R[A] = Globals[Bx]                         (ABx)

	// Arithmetic (int and float, dispatched by value tag at runtime)
	OpAdd // R[A] = R[B] + R[C]                                             (ABC)
	OpSub // R[A] = R[B] - R[C]                                             (ABC)
	OpMul // R[A] = R[B] * R[C]                                             (ABC)
	OpDiv // R[A] = R[B] / R[C]                                             (ABC)
	OpMod // R[A] = R[B] % R[C]                                             (ABC)
	OpNeg // R[A] = -R[B]                                                   (AB)

	// Comparison (result is Bool)
	OpEq // R[A] = R[B] == R[C]                                             (ABC)
	OpLt // R[A] = R[B] < R[C]                                              (ABC)
	OpLe // R[A] = R[B] <= R[C]                                             (ABC)

	// Logic (NOT only — AND/OR are compiled as conditional jumps)
	OpNot // R[A] = !R[B]                                                   (AB)

	// String
	OpConcat // R[A] = R[B] ++ R[C]                                         (ABC)

	// Control flow
	OpJump        // IP += SBx                                              (SBx)
	OpJumpIfFalse // if !R[A] then IP += SBx                                (A, SBx)
	OpCall        // R[A..A+C-1] = R[A](R[A+1..A+B])                        (ABC)
	OpTailCall    // return R[A](R[A+1..A+B])  -- reuses current frame      (AB)
	OpReturn      // return R[A]                                            (A)

	// Closures
	OpClosure // R[A] = new closure from Prototypes[Bx], followed by C MOVE
	// instructions specifying capture sources                    (ABx, then C MOVEs)

	// Collections
	OpMakeList   // R[A] = [R[B]..R[B+C-1]]                                 (ABC)
	OpMakeTuple  // R[A] = (R[B]..R[B+C-1])                                 (ABC)
	OpMakeRecord // R[A] = {fields from R[B]..R[B+C-1]} -- field names from
	// the immediately following pseudo-instructions encoded
	// as constant pool indices                                (ABC)
	OpCons     // R[A] = R[B] :: R[C]                                     (ABC)
	OpGetField // R[A] = R[B].Fields[C]  -- C indexes record's sorted
	// field name table inherited from constant pool           (ABC)
	OpGetIndex // R[A] = R[B][R[C]]  -- list integer indexing only        (ABC)

	// ADT
	OpMakeADT // R[A] = ADT{tag=B, fields=R[A+1..A+C]}                      (ABC)
	OpGetTag  // R[A] = R[B].tag  -- for switch dispatch                    (AB)

	// Builtins
	OpBuiltinCall // R[A] = BuiltinTable[Bx](R[A+1..A+C])                   (ABx, C)
	// Note: encoding uses ABx + a following ABC pseudo-op
	// is avoided here. C is read from a second instruction
	// word — see the assembler in vm/assemble.go (M4).
	OpBuiltinTrap // R[A] = evaluator.CallBuiltin(Bx, R[A+1..A+C])          (Phase 2C/2E)
	OpEffectTrap  // yield to evaluator for effect Bx, arg in R[A]          (Phase 2E)

	// Sentinel — keep last. Used by tests and disassemblers to bound the
	// opcode space. Not a real instruction.
	opCount
)

// String returns the human-readable opcode name. Used by the disassembler and
// error messages.
func (op OpCode) String() string {
	switch op {
	case OpLoadConst:
		return "LOAD_CONST"
	case OpLoadNil:
		return "LOAD_NIL"
	case OpMove:
		return "MOVE"
	case OpLoadGlobal:
		return "LOAD_GLOBAL"
	case OpAdd:
		return "ADD"
	case OpSub:
		return "SUB"
	case OpMul:
		return "MUL"
	case OpDiv:
		return "DIV"
	case OpMod:
		return "MOD"
	case OpNeg:
		return "NEG"
	case OpEq:
		return "EQ"
	case OpLt:
		return "LT"
	case OpLe:
		return "LE"
	case OpNot:
		return "NOT"
	case OpConcat:
		return "CONCAT"
	case OpJump:
		return "JUMP"
	case OpJumpIfFalse:
		return "JUMP_IF_FALSE"
	case OpCall:
		return "CALL"
	case OpTailCall:
		return "TAIL_CALL"
	case OpReturn:
		return "RETURN"
	case OpClosure:
		return "CLOSURE"
	case OpMakeList:
		return "MAKE_LIST"
	case OpMakeTuple:
		return "MAKE_TUPLE"
	case OpMakeRecord:
		return "MAKE_RECORD"
	case OpCons:
		return "CONS"
	case OpGetField:
		return "GET_FIELD"
	case OpGetIndex:
		return "GET_INDEX"
	case OpMakeADT:
		return "MAKE_ADT"
	case OpGetTag:
		return "GET_TAG"
	case OpBuiltinCall:
		return "BUILTIN_CALL"
	case OpBuiltinTrap:
		return "BUILTIN_TRAP"
	case OpEffectTrap:
		return "EFFECT_TRAP"
	}
	return fmt.Sprintf("UNKNOWN(%d)", uint8(op))
}

// OpCount returns the number of defined opcodes (excluding the sentinel).
// Used by tests and disassemblers.
func OpCount() int { return int(opCount) }

// sBxBias is the bias added to a signed jump offset to encode it as an unsigned
// 16-bit field. SBx range is [-0x8000, 0x7FFF].
const sBxBias = 0x8000

// Instruction is a single 32-bit bytecode word.
type Instruction uint32

// EncodeABC builds an ABC-form instruction. Each operand must fit in 8 bits.
func EncodeABC(op OpCode, a, b, c uint8) Instruction {
	return Instruction(uint32(op)) |
		Instruction(uint32(a))<<8 |
		Instruction(uint32(b))<<16 |
		Instruction(uint32(c))<<24
}

// EncodeABx builds an ABx-form instruction. Bx is unsigned 16 bits.
func EncodeABx(op OpCode, a uint8, bx uint16) Instruction {
	return Instruction(uint32(op)) |
		Instruction(uint32(a))<<8 |
		Instruction(uint32(bx))<<16
}

// EncodeASBx builds an ABx-form instruction with a signed 16-bit operand.
// sbx must be in the range [-0x8000, 0x7FFF].
func EncodeASBx(op OpCode, a uint8, sbx int) Instruction {
	if sbx < -sBxBias || sbx > sBxBias-1 {
		panic(fmt.Sprintf("bytecode: SBx out of range: %d", sbx))
	}
	return EncodeABx(op, a, uint16(sbx+sBxBias))
}

// Op extracts the opcode.
func (i Instruction) Op() OpCode { return OpCode(i & 0xFF) }

// A extracts the A operand (8 bits).
func (i Instruction) A() uint8 { return uint8((i >> 8) & 0xFF) }

// B extracts the B operand (8 bits, ABC form).
func (i Instruction) B() uint8 { return uint8((i >> 16) & 0xFF) }

// C extracts the C operand (8 bits, ABC form).
func (i Instruction) C() uint8 { return uint8((i >> 24) & 0xFF) }

// Bx extracts the wide unsigned operand (16 bits, ABx form).
func (i Instruction) Bx() uint16 { return uint16((i >> 16) & 0xFFFF) }

// SBx extracts the wide signed operand (16 bits, ABx form, biased).
func (i Instruction) SBx() int { return int(i.Bx()) - sBxBias }

// String returns a human-readable disassembly of a single instruction. The
// output is intentionally minimal; full disassembly with constant resolution
// happens in Phase 2D's disassembler.
func (i Instruction) String() string {
	op := i.Op()
	switch op {
	case OpLoadConst, OpLoadGlobal, OpClosure, OpBuiltinCall, OpBuiltinTrap, OpEffectTrap:
		return fmt.Sprintf("%s r%d, #%d", op, i.A(), i.Bx())
	case OpJump:
		return fmt.Sprintf("%s %+d", op, i.SBx())
	case OpJumpIfFalse:
		return fmt.Sprintf("%s r%d, %+d", op, i.A(), i.SBx())
	case OpLoadNil, OpReturn:
		return fmt.Sprintf("%s r%d", op, i.A())
	case OpMove, OpNeg, OpNot, OpGetTag, OpTailCall:
		return fmt.Sprintf("%s r%d, r%d", op, i.A(), i.B())
	default:
		return fmt.Sprintf("%s r%d, r%d, r%d", op, i.A(), i.B(), i.C())
	}
}

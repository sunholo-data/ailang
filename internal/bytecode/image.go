package bytecode

import "fmt"

// FuncPrototype is a compiled function: instructions, register layout,
// constant references, nested prototype indices (for closures), and source
// line information for diagnostics.
//
// FuncPrototype satisfies the FuncPrototypeRef interface so it can be used
// directly as a closure target without an adapter.
type FuncPrototype struct {
	// Name is a human-readable identifier for stack traces and disassembly.
	Name string

	// NumRegs is the size of the register frame allocated when this function
	// is called. The compiler (Phase 2C) computes this from VarDecl count
	// plus a few scratch slots.
	NumRegs uint8

	// NumParams is the number of parameters. They occupy registers
	// [0, NumParams). The remaining registers are locals.
	NumParams uint8

	// IsVariadic indicates the last parameter consumes a list of remaining
	// arguments. Phase 2B does not require this; included so the type doesn't
	// need a breaking change in 2C.
	IsVariadic bool

	// NumCaptures is the number of values this function captures from its
	// enclosing scope when it is instantiated as a closure. Used by the VM
	// to know how many pseudo-MOVE instructions to consume after CLOSURE.
	// Top-level functions have NumCaptures=0.
	NumCaptures uint8

	// Instructions is the bytecode body.
	Instructions []Instruction

	// Constants holds indices into the parent BytecodeImage's constant pool.
	// Storing indices (not values) keeps the prototype small and the
	// deduplication centralized.
	//
	// LOAD_CONST's Bx field is an index into THIS slice (the prototype's
	// local constant table), not directly into the image pool. The
	// indirection lets a prototype use only the constants it actually
	// references.
	Constants []int

	// NestedProtos holds indices into the parent BytecodeImage's prototype
	// table. CLOSURE's Bx field indexes into this slice.
	NestedProtos []int

	// LineInfo maps instruction index → source line number. Used to attach
	// source locations to runtime errors. Length should equal len(Instructions);
	// a zero entry means "no source line available".
	LineInfo []int

	// File is the source file the function was compiled from. Used together
	// with LineInfo to format runtime errors as `<file>:<line>`. Empty when
	// the file is unknown (e.g. hand-built test prototypes).
	File string

	// EvalOnly marks this prototype as a stub: the function exists in the
	// program but the bytecode compiler couldn't compile it (or chose not to)
	// and the VM must trap to the evaluator via VM.Interop on every call.
	//
	// When true, Instructions and LineInfo are nil and NumRegs is zero. The
	// VM checks this flag inside OpCall/OpTailCall before pushing a frame.
	// Other prototypes may still hold OpClosure references to this prototype
	// via NestedProtos — that is the whole point of M-BYTECODE-2D M3.
	EvalOnly bool

	// EvalReason is a human-readable explanation of why the function was
	// marked EvalOnly (typically the compile error). Used in error messages
	// and in `ailang disasm` output. Only meaningful when EvalOnly is true.
	EvalReason string
}

// ProtoName implements FuncPrototypeRef.
func (p *FuncPrototype) ProtoName() string { return p.Name }

// NumRegisters implements FuncPrototypeRef.
func (p *FuncPrototype) NumRegisters() uint8 { return p.NumRegs }

// NumParameters implements FuncPrototypeRef.
func (p *FuncPrototype) NumParameters() uint8 { return p.NumParams }

// LookupConstant resolves a local constant index (the Bx of a LOAD_CONST) to
// the actual Value via the supplied image's pool. Out-of-range indices return
// the second value as false; the VM should treat that as a corrupt image.
func (p *FuncPrototype) LookupConstant(localIdx int, img *BytecodeImage) (Value, bool) {
	if localIdx < 0 || localIdx >= len(p.Constants) {
		return Value{}, false
	}
	poolIdx := p.Constants[localIdx]
	if poolIdx < 0 || poolIdx >= len(img.Constants) {
		return Value{}, false
	}
	return img.Constants[poolIdx], true
}

// BytecodeImage is the in-memory bytecode unit. It bundles a constant pool,
// a prototype table, and an entry point. Phase 2B is in-memory only — there
// is no on-disk format yet (Phase 2D will add serialization if needed).
type BytecodeImage struct {
	// Constants is the deduplicated constant pool. Constants are added via
	// AddConstant which dedupes by structural equality.
	Constants []Value

	// Prototypes is the table of compiled functions. The order is the
	// canonical numeric prototype identity used by CLOSURE indices.
	Prototypes []*FuncPrototype

	// EntryPoint is the index in Prototypes of the function the VM should
	// invoke when running the image. -1 means "no entry point" (e.g. a
	// library image with only callable functions).
	EntryPoint int

	// Globals is a flat array of mutable global slots. LOAD_GLOBAL Bx indexes
	// into this slice. Phase 2B uses this only for hand-assembled tests; the
	// compiler in Phase 2C will populate it from top-level let bindings.
	Globals []Value
}

// NewImage returns an empty bytecode image with no entry point.
func NewImage() *BytecodeImage {
	return &BytecodeImage{
		EntryPoint: -1,
	}
}

// AddConstant adds a value to the constant pool, deduplicating by structural
// equality (Value.Equal). Returns the pool index. The dedup is intentionally
// linear — Phase 2B images are tiny. If profiling later shows this matters,
// we can add a hash table keyed by a canonical representation.
func (img *BytecodeImage) AddConstant(v Value) int {
	for i, existing := range img.Constants {
		if existing.Equal(v) {
			return i
		}
	}
	img.Constants = append(img.Constants, v)
	return len(img.Constants) - 1
}

// AddPrototype appends a prototype and returns its index. The returned index
// is what CLOSURE instructions reference (via the parent prototype's
// NestedProtos table) and what EntryPoint is set to.
func (img *BytecodeImage) AddPrototype(p *FuncPrototype) int {
	img.Prototypes = append(img.Prototypes, p)
	return len(img.Prototypes) - 1
}

// SetEntryPoint marks a prototype index as the image's entry point. Returns
// an error if the index is out of range.
func (img *BytecodeImage) SetEntryPoint(idx int) error {
	if idx < 0 || idx >= len(img.Prototypes) {
		return fmt.Errorf("bytecode: entry point index %d out of range [0,%d)", idx, len(img.Prototypes))
	}
	img.EntryPoint = idx
	return nil
}

// Validate performs structural sanity checks on an image:
//   - every instruction's prototype/constant references are in range
//   - jump targets land within the same prototype
//   - LineInfo (if non-nil) has the right length
//
// This is intended to be called by tests and by the VM before execution as
// a defensive check. It does not verify type correctness or semantic well-
// formedness — that's the compiler's job.
func (img *BytecodeImage) Validate() error {
	for protoIdx := range img.Prototypes {
		if err := img.ValidatePrototype(protoIdx); err != nil {
			return err
		}
	}
	if img.EntryPoint >= 0 && img.EntryPoint >= len(img.Prototypes) {
		return fmt.Errorf("bytecode: entry point %d out of range", img.EntryPoint)
	}
	return nil
}

// ValidatePrototype runs the structural checks from Validate against a single
// prototype. The compiler uses this to validate per-function after lowering,
// so a single buggy proto can be rolled back and tagged EvalOnly rather than
// aborting the whole image (see compiler.go Phase 2).
func (img *BytecodeImage) ValidatePrototype(protoIdx int) error {
	if protoIdx < 0 || protoIdx >= len(img.Prototypes) {
		return fmt.Errorf("bytecode: prototype index %d out of range [0,%d)", protoIdx, len(img.Prototypes))
	}
	p := img.Prototypes[protoIdx]
	if p == nil {
		return fmt.Errorf("bytecode: prototype %d is nil", protoIdx)
	}
	// EvalOnly stubs have no body — only the metadata callers need to push
	// args correctly (Name, NumParams, File). Skip the body checks.
	if p.EvalOnly {
		if len(p.Instructions) != 0 || len(p.LineInfo) != 0 {
			return fmt.Errorf("bytecode: proto %q (idx %d): EvalOnly stub must have empty Instructions and LineInfo", p.Name, protoIdx)
		}
		return nil
	}
	if p.NumParams > p.NumRegs {
		return fmt.Errorf("bytecode: proto %q (idx %d): NumParams=%d exceeds NumRegs=%d", p.Name, protoIdx, p.NumParams, p.NumRegs)
	}
	if p.LineInfo != nil && len(p.LineInfo) != len(p.Instructions) {
		return fmt.Errorf("bytecode: proto %q (idx %d): LineInfo length %d != Instructions length %d", p.Name, protoIdx, len(p.LineInfo), len(p.Instructions))
	}
	for ip, inst := range p.Instructions {
		if err := img.validateInstruction(p, protoIdx, ip, inst); err != nil {
			return err
		}
	}
	return nil
}

// validateInstruction checks a single instruction. Kept separate to keep
// Validate readable.
func (img *BytecodeImage) validateInstruction(p *FuncPrototype, protoIdx, ip int, inst Instruction) error {
	op := inst.Op()
	loc := func() string {
		return fmt.Sprintf("proto %q (idx %d) ip %d (%s)", p.Name, protoIdx, ip, op)
	}

	checkReg := func(r uint8, label string) error {
		if uint8(r) >= p.NumRegs {
			return fmt.Errorf("bytecode: %s: %s register r%d exceeds NumRegs=%d", loc(), label, r, p.NumRegs)
		}
		return nil
	}

	checkLocalConst := func(idx uint16) error {
		if int(idx) >= len(p.Constants) {
			return fmt.Errorf("bytecode: %s: constant index %d out of range [0,%d)", loc(), idx, len(p.Constants))
		}
		poolIdx := p.Constants[idx]
		if poolIdx < 0 || poolIdx >= len(img.Constants) {
			return fmt.Errorf("bytecode: %s: constant pool index %d out of range [0,%d)", loc(), poolIdx, len(img.Constants))
		}
		return nil
	}

	checkProto := func(idx uint16) error {
		if int(idx) >= len(p.NestedProtos) {
			return fmt.Errorf("bytecode: %s: nested proto index %d out of range [0,%d)", loc(), idx, len(p.NestedProtos))
		}
		protoTblIdx := p.NestedProtos[idx]
		if protoTblIdx < 0 || protoTblIdx >= len(img.Prototypes) {
			return fmt.Errorf("bytecode: %s: prototype table index %d out of range [0,%d)", loc(), protoTblIdx, len(img.Prototypes))
		}
		return nil
	}

	checkJump := func(sbx int) error {
		target := ip + 1 + sbx
		if target < 0 || target >= len(p.Instructions) {
			return fmt.Errorf("bytecode: %s: jump target %d out of range [0,%d)", loc(), target, len(p.Instructions))
		}
		return nil
	}

	switch op {
	case OpLoadConst:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		return checkLocalConst(inst.Bx())
	case OpLoadNil, OpReturn:
		return checkReg(inst.A(), "dest")
	case OpLoadGlobal:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		if int(inst.Bx()) >= len(img.Globals) {
			return fmt.Errorf("bytecode: %s: global index %d out of range [0,%d)", loc(), inst.Bx(), len(img.Globals))
		}
		return nil
	case OpMove, OpNeg, OpNot, OpGetTag:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		return checkReg(inst.B(), "src")
	case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpEq, OpLt, OpLe, OpConcat, OpCons, OpGetIndex:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		if err := checkReg(inst.B(), "lhs"); err != nil {
			return err
		}
		return checkReg(inst.C(), "rhs")
	case OpGetField:
		// A=dest register, B=record register, C=sorted field index (NOT a register).
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		return checkReg(inst.B(), "record")
	case OpJump:
		return checkJump(inst.SBx())
	case OpJumpIfFalse:
		if err := checkReg(inst.A(), "cond"); err != nil {
			return err
		}
		return checkJump(inst.SBx())
	case OpCall:
		// A = function reg & dest, B = arg count, C = result count (1 for now)
		if err := checkReg(inst.A(), "callee"); err != nil {
			return err
		}
		// Args occupy A+1..A+B; check the highest one.
		highArg := uint16(inst.A()) + uint16(inst.B())
		if highArg >= uint16(p.NumRegs) {
			return fmt.Errorf("bytecode: %s: call args overflow registers (high=%d, NumRegs=%d)", loc(), highArg, p.NumRegs)
		}
		return nil
	case OpTailCall:
		if err := checkReg(inst.A(), "callee"); err != nil {
			return err
		}
		highArg := uint16(inst.A()) + uint16(inst.B())
		if highArg >= uint16(p.NumRegs) {
			return fmt.Errorf("bytecode: %s: tail call args overflow registers (high=%d, NumRegs=%d)", loc(), highArg, p.NumRegs)
		}
		return nil
	case OpClosure:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		return checkProto(inst.Bx())
	case OpMakeList, OpMakeTuple, OpMakeRecord, OpMakeADT:
		if err := checkReg(inst.A(), "dest"); err != nil {
			return err
		}
		// Validate element register range; for MakeADT, A+1..A+C are field regs.
		if op == OpMakeADT {
			highReg := uint16(inst.A()) + uint16(inst.C())
			if highReg >= uint16(p.NumRegs) {
				return fmt.Errorf("bytecode: %s: ADT field regs overflow (high=%d, NumRegs=%d)", loc(), highReg, p.NumRegs)
			}
		} else {
			highReg := uint16(inst.B()) + uint16(inst.C())
			if highReg > uint16(p.NumRegs) {
				return fmt.Errorf("bytecode: %s: element regs overflow (high=%d, NumRegs=%d)", loc(), highReg, p.NumRegs)
			}
		}
		return nil
	case OpBuiltinCall, OpBuiltinTrap, OpEffectTrap:
		// Builtin/effect indices live in a runtime-side table; we can't
		// validate them statically beyond bounds-checking the dest register.
		return checkReg(inst.A(), "dest")
	}
	return nil
}

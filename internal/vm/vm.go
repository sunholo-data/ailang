package vm

import (
	"errors"
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
)

// DefaultMaxStack is the default frame depth limit. Matches the evaluator's
// recursion limit so divergence behavior is consistent (§3.6).
const DefaultMaxStack = 1000

// VMError is a runtime error from the VM. It carries the source location of
// the faulting instruction (when available) and the instruction itself for
// debug.
type VMError struct {
	Msg      string
	Func     string
	File     string
	Line     int
	IP       int
	OpString string
}

func (e *VMError) Error() string {
	loc := ""
	switch {
	case e.File != "" && e.Line > 0:
		loc = fmt.Sprintf("%s:%d", e.File, e.Line)
	case e.Line > 0:
		loc = fmt.Sprintf("line %d", e.Line)
	}
	if loc != "" {
		return fmt.Sprintf("vm: %s (in %s at %s, ip %d, op %s)", e.Msg, e.Func, loc, e.IP, e.OpString)
	}
	return fmt.Sprintf("vm: %s (in %s at ip %d, op %s)", e.Msg, e.Func, e.IP, e.OpString)
}

// ErrStackOverflow is returned when the call stack exceeds MaxStack frames.
var ErrStackOverflow = errors.New("vm: stack overflow")

// VM is the bytecode virtual machine state.
type VM struct {
	Image    *bytecode.BytecodeImage
	MaxStack int

	// Stack tracks the active frames for overflow detection. The currently
	// executing frame is the top; entries below it are paused callers.
	Stack []*Frame

	// Interop is the optional bridge to the tree-walking evaluator. Set by
	// the CLI when running a mixed-mode program: when OpCall/OpTailCall hits
	// a callee whose prototype is marked EvalOnly, the VM hands the call to
	// Interop instead of pushing a new frame. If Interop is nil, the VM
	// returns an explicit error instead of silently mis-running the function.
	Interop EvalInterop
}

// NewVM constructs a VM bound to an image. The image is not validated here;
// the caller should call img.Validate() upfront for hand-assembled images.
func NewVM(img *bytecode.BytecodeImage) *VM {
	return &VM{
		Image:    img,
		MaxStack: DefaultMaxStack,
	}
}

// Run executes proto with the given args and returns the result. Used as the
// VM's primary entry point. The entry frame's ReturnReg is unused.
func (vm *VM) Run(proto *bytecode.FuncPrototype, args []bytecode.Value) (bytecode.Value, error) {
	if proto == nil {
		return bytecode.Value{}, &VMError{Msg: "nil entry prototype", Func: "<entry>"}
	}
	if len(args) != int(proto.NumParams) {
		return bytecode.Value{}, &VMError{
			Msg:  fmt.Sprintf("entry expected %d args, got %d", proto.NumParams, len(args)),
			Func: proto.Name,
		}
	}
	frame := newFrame(proto, 0, nil)
	copy(frame.Regs, args)
	vm.Stack = append(vm.Stack, frame)
	defer func() { vm.Stack = vm.Stack[:0] }()
	return vm.run(frame)
}

// run is the dispatch loop. It executes from the given frame until a
// top-level RETURN unwinds the stack to nothing, then returns the result.
func (vm *VM) run(frame *Frame) (bytecode.Value, error) {
	for {
		if frame.IP < 0 || frame.IP >= len(frame.Proto.Instructions) {
			return bytecode.Value{}, vm.errAt(frame, "instruction pointer out of range", bytecode.Instruction(0))
		}
		inst := frame.Proto.Instructions[frame.IP]
		op := inst.Op()

		switch op {

		// --- Loads -------------------------------------------------------

		case bytecode.OpLoadConst:
			v, ok := frame.Proto.LookupConstant(int(inst.Bx()), vm.Image)
			if !ok {
				return bytecode.Value{}, vm.errAt(frame, "constant lookup failed", inst)
			}
			frame.Regs[inst.A()] = v
			frame.IP++

		case bytecode.OpLoadNil:
			frame.Regs[inst.A()] = bytecode.Unit()
			frame.IP++

		case bytecode.OpMove:
			frame.Regs[inst.A()] = frame.Regs[inst.B()]
			frame.IP++

		case bytecode.OpLoadGlobal:
			idx := int(inst.Bx())
			if idx < 0 || idx >= len(vm.Image.Globals) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("global %d out of range", idx), inst)
			}
			frame.Regs[inst.A()] = vm.Image.Globals[idx]
			frame.IP++

		// --- Arithmetic --------------------------------------------------

		case bytecode.OpAdd, bytecode.OpSub, bytecode.OpMul, bytecode.OpDiv, bytecode.OpMod:
			lhs := frame.Regs[inst.B()]
			rhs := frame.Regs[inst.C()]
			res, err := arith(op, lhs, rhs)
			if err != nil {
				return bytecode.Value{}, vm.errAt(frame, err.Error(), inst)
			}
			frame.Regs[inst.A()] = res
			frame.IP++

		case bytecode.OpNeg:
			v := frame.Regs[inst.B()]
			switch v.Tag {
			case bytecode.TagInt:
				frame.Regs[inst.A()] = bytecode.NewInt(-v.Int)
			case bytecode.TagFloat:
				frame.Regs[inst.A()] = bytecode.NewFloat(-v.Flt)
			default:
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("NEG on %s", v.Tag), inst)
			}
			frame.IP++

		// --- Comparison --------------------------------------------------

		case bytecode.OpEq:
			// EQ uses runtime IEEE semantics for floats (NaN != NaN), unlike
			// Value.Equal which is for dedup.
			lhs := frame.Regs[inst.B()]
			rhs := frame.Regs[inst.C()]
			frame.Regs[inst.A()] = bytecode.NewBool(runtimeEq(lhs, rhs))
			frame.IP++

		case bytecode.OpLt, bytecode.OpLe:
			lhs := frame.Regs[inst.B()]
			rhs := frame.Regs[inst.C()]
			res, err := compare(op, lhs, rhs)
			if err != nil {
				return bytecode.Value{}, vm.errAt(frame, err.Error(), inst)
			}
			frame.Regs[inst.A()] = bytecode.NewBool(res)
			frame.IP++

		// --- Logic -------------------------------------------------------

		case bytecode.OpNot:
			v := frame.Regs[inst.B()]
			if v.Tag != bytecode.TagBool {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("NOT on %s", v.Tag), inst)
			}
			frame.Regs[inst.A()] = bytecode.NewBool(!v.Bool)
			frame.IP++

		// --- String ------------------------------------------------------

		case bytecode.OpConcat:
			lhs := frame.Regs[inst.B()]
			rhs := frame.Regs[inst.C()]
			if lhs.Tag != bytecode.TagString || rhs.Tag != bytecode.TagString {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CONCAT on %s and %s", lhs.Tag, rhs.Tag), inst)
			}
			frame.Regs[inst.A()] = bytecode.NewString(lhs.AsString() + rhs.AsString())
			frame.IP++

		// --- Control flow ------------------------------------------------

		case bytecode.OpJump:
			frame.IP = frame.IP + 1 + inst.SBx()

		case bytecode.OpJumpIfFalse:
			cond := frame.Regs[inst.A()]
			if cond.Tag != bytecode.TagBool {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("JUMP_IF_FALSE on %s", cond.Tag), inst)
			}
			if !cond.Bool {
				frame.IP = frame.IP + 1 + inst.SBx()
			} else {
				frame.IP++
			}

		case bytecode.OpCall:
			callee := frame.Regs[inst.A()]
			if callee.Tag != bytecode.TagClosure {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CALL on non-closure (%s)", callee.Tag), inst)
			}
			closure := callee.AsClosure()
			calleeProto, ok := closure.Proto.(*bytecode.FuncPrototype)
			if !ok {
				return bytecode.Value{}, vm.errAt(frame, "CALL on closure with non-FuncPrototype", inst)
			}
			argCount := int(inst.B())
			if int(calleeProto.NumParams) != argCount {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CALL: %s expects %d args, got %d", calleeProto.Name, calleeProto.NumParams, argCount), inst)
			}
			// EvalOnly stub: dispatch through the evaluator interop bridge.
			// No new frame is pushed; the result is written into the callee's
			// register slot (matching the convention OpReturn uses) and the
			// caller's IP advances past the CALL.
			if calleeProto.EvalOnly {
				if vm.Interop == nil {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CALL: %s is evaluator-only (%s) but no interop bridge is wired", calleeProto.Name, calleeProto.EvalReason), inst)
				}
				args := make([]bytecode.Value, argCount)
				for i := 0; i < argCount; i++ {
					args[i] = frame.Regs[int(inst.A())+1+i]
				}
				result, err := vm.Interop.CallEvalFunc(calleeProto.Name, args)
				if err != nil {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CALL via eval interop: %v", err), inst)
				}
				frame.Regs[inst.A()] = result
				frame.IP++
				continue
			}
			if len(vm.Stack) >= vm.MaxStack {
				return bytecode.Value{}, ErrStackOverflow
			}
			// Advance the caller's IP past the CALL before pushing the new
			// frame so that on RETURN we resume at the next instruction.
			frame.IP++
			newF := newFrame(calleeProto, inst.A(), frame)
			for i := 0; i < argCount; i++ {
				newF.Regs[i] = frame.Regs[int(inst.A())+1+i]
			}
			// Captures live in the slot above the parameters.
			for i, cap := range closure.Captures {
				newF.Regs[int(calleeProto.NumParams)+i] = cap
			}
			vm.Stack = append(vm.Stack, newF)
			frame = newF

		case bytecode.OpTailCall:
			callee := frame.Regs[inst.A()]
			if callee.Tag != bytecode.TagClosure {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("TAIL_CALL on non-closure (%s)", callee.Tag), inst)
			}
			closure := callee.AsClosure()
			calleeProto, ok := closure.Proto.(*bytecode.FuncPrototype)
			if !ok {
				return bytecode.Value{}, vm.errAt(frame, "TAIL_CALL on closure with non-FuncPrototype", inst)
			}
			argCount := int(inst.B())
			if int(calleeProto.NumParams) != argCount {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("TAIL_CALL: %s expects %d args, got %d", calleeProto.Name, calleeProto.NumParams, argCount), inst)
			}
			// EvalOnly stub in tail position: bridge to the evaluator and
			// then return the result through the current frame, just as if
			// the function had run inline.
			if calleeProto.EvalOnly {
				if vm.Interop == nil {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("TAIL_CALL: %s is evaluator-only (%s) but no interop bridge is wired", calleeProto.Name, calleeProto.EvalReason), inst)
				}
				args := make([]bytecode.Value, argCount)
				for i := 0; i < argCount; i++ {
					args[i] = frame.Regs[int(inst.A())+1+i]
				}
				result, err := vm.Interop.CallEvalFunc(calleeProto.Name, args)
				if err != nil {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("TAIL_CALL via eval interop: %v", err), inst)
				}
				// Pop the current frame and write the result to the caller's
				// return register, mirroring OpReturn's path.
				vm.Stack = vm.Stack[:len(vm.Stack)-1]
				caller := frame.Caller
				if caller == nil {
					return result, nil
				}
				caller.Regs[frame.ReturnReg] = result
				frame = caller
				continue
			}
			// Stash args BEFORE we resize the register slab — they live in
			// the current frame's registers and will be overwritten by reuseFor.
			args := make([]bytecode.Value, argCount)
			for i := 0; i < argCount; i++ {
				args[i] = frame.Regs[int(inst.A())+1+i]
			}
			caps := closure.Captures // captures are heap, no copy needed
			frame.reuseFor(calleeProto)
			copy(frame.Regs, args)
			for i, c := range caps {
				frame.Regs[int(calleeProto.NumParams)+i] = c
			}
			// Frame stack depth does NOT grow — that's the whole point.

		case bytecode.OpReturn:
			retVal := frame.Regs[inst.A()]
			// Pop the current frame.
			vm.Stack = vm.Stack[:len(vm.Stack)-1]
			caller := frame.Caller
			if caller == nil {
				return retVal, nil
			}
			caller.Regs[frame.ReturnReg] = retVal
			frame = caller

		// --- Closures ----------------------------------------------------

		case bytecode.OpClosure:
			protoTblIdx := int(inst.Bx())
			if protoTblIdx >= len(frame.Proto.NestedProtos) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("nested proto index %d out of range", protoTblIdx), inst)
			}
			imageIdx := frame.Proto.NestedProtos[protoTblIdx]
			if imageIdx < 0 || imageIdx >= len(vm.Image.Prototypes) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("image proto index %d out of range", imageIdx), inst)
			}
			innerProto := vm.Image.Prototypes[imageIdx]
			// Read NumCaptures pseudo-MOVE instructions for capture sources.
			caps := make([]bytecode.Value, innerProto.NumCaptures)
			for i := uint8(0); i < innerProto.NumCaptures; i++ {
				captureInst := frame.Proto.Instructions[frame.IP+1+int(i)]
				if captureInst.Op() != bytecode.OpMove {
					return bytecode.Value{}, vm.errAt(frame, "CLOSURE: expected MOVE pseudo-instruction for capture", captureInst)
				}
				caps[i] = frame.Regs[captureInst.B()]
			}
			frame.Regs[inst.A()] = bytecode.NewClosure(innerProto, caps)
			frame.IP += 1 + int(innerProto.NumCaptures)

		// --- Collections -------------------------------------------------

		case bytecode.OpMakeList:
			start, count := int(inst.B()), int(inst.C())
			elems := make([]bytecode.Value, count)
			for i := 0; i < count; i++ {
				elems[i] = frame.Regs[start+i]
			}
			frame.Regs[inst.A()] = bytecode.NewList(elems)
			frame.IP++

		case bytecode.OpMakeTuple:
			start, count := int(inst.B()), int(inst.C())
			elems := make([]bytecode.Value, count)
			for i := 0; i < count; i++ {
				elems[i] = frame.Regs[start+i]
			}
			frame.Regs[inst.A()] = bytecode.NewTuple(elems)
			frame.IP++

		case bytecode.OpMakeRecord:
			// Layout: A=dst register, B=value register base, C=field count.
			// Field names are read from the C immediately following pseudo
			// LOAD_CONST instructions, whose Bx field is the local constant
			// index of the (string) field name. This mirrors the CLOSURE +
			// pseudo-MOVE pattern used for captures.
			valueBase, count := int(inst.B()), int(inst.C())
			fields := make([]bytecode.RecordField, count)
			for i := 0; i < count; i++ {
				nameInst := frame.Proto.Instructions[frame.IP+1+i]
				if nameInst.Op() != bytecode.OpLoadConst {
					return bytecode.Value{}, vm.errAt(frame, "MAKE_RECORD: expected pseudo-LOAD_CONST for field name", nameInst)
				}
				nameVal, ok := frame.Proto.LookupConstant(int(nameInst.Bx()), vm.Image)
				if !ok || nameVal.Tag != bytecode.TagString {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("MAKE_RECORD: bad field-name constant at local index %d", nameInst.Bx()), nameInst)
				}
				fields[i] = bytecode.RecordField{
					Name:  nameVal.AsString(),
					Value: frame.Regs[valueBase+i],
				}
			}
			frame.Regs[inst.A()] = bytecode.NewRecord(fields)
			frame.IP += 1 + count

		case bytecode.OpCons:
			head := frame.Regs[inst.B()]
			tail := frame.Regs[inst.C()]
			if tail.Tag != bytecode.TagList {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("CONS: tail is %s, not List", tail.Tag), inst)
			}
			old := tail.AsList()
			elems := make([]bytecode.Value, 0, len(old)+1)
			elems = append(elems, head)
			elems = append(elems, old...)
			frame.Regs[inst.A()] = bytecode.NewList(elems)
			frame.IP++

		case bytecode.OpGetField:
			rec := frame.Regs[inst.B()]
			idx := int(inst.C())
			switch rec.Tag {
			case bytecode.TagRecord:
				fields := rec.AsRecord()
				if idx >= len(fields) {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_FIELD: index %d exceeds field count %d", idx, len(fields)), inst)
				}
				frame.Regs[inst.A()] = fields[idx].Value
			case bytecode.TagADT:
				fields := rec.AsADT().Fields
				if idx >= len(fields) {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_FIELD: index %d exceeds ADT field count %d", idx, len(fields)), inst)
				}
				frame.Regs[inst.A()] = fields[idx]
			case bytecode.TagTuple:
				elems := rec.AsTuple()
				if idx >= len(elems) {
					return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_FIELD: index %d exceeds tuple arity %d", idx, len(elems)), inst)
				}
				frame.Regs[inst.A()] = elems[idx]
			default:
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_FIELD on %s", rec.Tag), inst)
			}
			frame.IP++

		case bytecode.OpGetIndex:
			lst := frame.Regs[inst.B()]
			idxV := frame.Regs[inst.C()]
			if lst.Tag != bytecode.TagList {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_INDEX on %s", lst.Tag), inst)
			}
			if idxV.Tag != bytecode.TagInt {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_INDEX with %s index", idxV.Tag), inst)
			}
			elems := lst.AsList()
			i := int(idxV.Int)
			if i < 0 || i >= len(elems) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_INDEX: %d out of range [0,%d)", i, len(elems)), inst)
			}
			frame.Regs[inst.A()] = elems[i]
			frame.IP++

		// --- ADT ---------------------------------------------------------

		case bytecode.OpMakeADT:
			tag := int(inst.B())
			count := int(inst.C())
			fields := make([]bytecode.Value, count)
			for i := 0; i < count; i++ {
				fields[i] = frame.Regs[int(inst.A())+1+i]
			}
			frame.Regs[inst.A()] = bytecode.NewADT(tag, fields)
			frame.IP++

		case bytecode.OpGetTag:
			v := frame.Regs[inst.B()]
			if v.Tag != bytecode.TagADT {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("GET_TAG on %s", v.Tag), inst)
			}
			frame.Regs[inst.A()] = bytecode.NewInt(int64(v.AsADT().Tag))
			frame.IP++

		// --- Builtins / Effects (Phase 2C/2E stubs) ----------------------

		case bytecode.OpBuiltinCall:
			builtinIdx := int(inst.B())
			argc := int(inst.C())
			if builtinIdx >= len(BuiltinTable) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("BUILTIN_CALL: unknown builtin index %d", builtinIdx), inst)
			}
			argBase := int(inst.A()) + 1
			args := frame.Regs[argBase : argBase+argc]
			result, err := BuiltinTable[builtinIdx](args)
			if err != nil {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("BUILTIN_CALL: %v", err), inst)
			}
			frame.Regs[inst.A()] = result
			frame.IP++
		case bytecode.OpBuiltinTrap:
			name := "<unknown>"
			if v, ok := frame.Proto.LookupConstant(int(inst.Bx()), vm.Image); ok && v.Tag == bytecode.TagString {
				name = v.AsString()
			}
			return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("BUILTIN_TRAP: %s not yet wired (Phase 2E)", name), inst)
		case bytecode.OpEffectTrap:
			return bytecode.Value{}, vm.errAt(frame, "EFFECT_TRAP not implemented in Phase 2B", inst)

		default:
			return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("unknown opcode %d", op), inst)
		}
	}
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

// --- Arithmetic helpers -----------------------------------------------------

// arith dispatches a binary arithmetic operation by tag. Both operands must
// have the same numeric tag (no implicit int↔float coercion in Phase 2B).
func arith(op bytecode.OpCode, lhs, rhs bytecode.Value) (bytecode.Value, error) {
	if lhs.Tag != rhs.Tag {
		return bytecode.Value{}, fmt.Errorf("arith %s: type mismatch %s vs %s", op, lhs.Tag, rhs.Tag)
	}
	switch lhs.Tag {
	case bytecode.TagInt:
		l, r := lhs.Int, rhs.Int
		switch op {
		case bytecode.OpAdd:
			return bytecode.NewInt(l + r), nil
		case bytecode.OpSub:
			return bytecode.NewInt(l - r), nil
		case bytecode.OpMul:
			return bytecode.NewInt(l * r), nil
		case bytecode.OpDiv:
			if r == 0 {
				return bytecode.Value{}, fmt.Errorf("division by zero")
			}
			return bytecode.NewInt(l / r), nil
		case bytecode.OpMod:
			if r == 0 {
				return bytecode.Value{}, fmt.Errorf("modulo by zero")
			}
			return bytecode.NewInt(l % r), nil
		}
	case bytecode.TagFloat:
		l, r := lhs.Flt, rhs.Flt
		switch op {
		case bytecode.OpAdd:
			return bytecode.NewFloat(l + r), nil
		case bytecode.OpSub:
			return bytecode.NewFloat(l - r), nil
		case bytecode.OpMul:
			return bytecode.NewFloat(l * r), nil
		case bytecode.OpDiv:
			return bytecode.NewFloat(l / r), nil
		case bytecode.OpMod:
			return bytecode.Value{}, fmt.Errorf("MOD on Float not supported")
		}
	}
	return bytecode.Value{}, fmt.Errorf("arith %s on %s not supported", op, lhs.Tag)
}

// runtimeEq is the IEEE-aware equality used by OpEq. Unlike Value.Equal it
// treats NaN != NaN. Used for ==, not for dedup.
func runtimeEq(lhs, rhs bytecode.Value) bool {
	if lhs.Tag != rhs.Tag {
		return false
	}
	switch lhs.Tag {
	case bytecode.TagFloat:
		// IEEE: NaN != anything, including itself.
		if lhs.Flt != lhs.Flt || rhs.Flt != rhs.Flt {
			return false
		}
		return lhs.Flt == rhs.Flt
	case bytecode.TagInt:
		return lhs.Int == rhs.Int
	case bytecode.TagBool:
		return lhs.Bool == rhs.Bool
	case bytecode.TagUnit:
		return true
	case bytecode.TagString:
		return lhs.AsString() == rhs.AsString()
	}
	// For lists/records/tuples/ADTs, fall through to structural equality. We
	// reuse Value.Equal here since the NaN concern is float-specific and we've
	// already handled the float case above.
	return lhs.Equal(rhs)
}

// compare implements LT and LE for ordered numeric and string types.
func compare(op bytecode.OpCode, lhs, rhs bytecode.Value) (bool, error) {
	if lhs.Tag != rhs.Tag {
		return false, fmt.Errorf("compare %s: type mismatch %s vs %s", op, lhs.Tag, rhs.Tag)
	}
	switch lhs.Tag {
	case bytecode.TagInt:
		switch op {
		case bytecode.OpLt:
			return lhs.Int < rhs.Int, nil
		case bytecode.OpLe:
			return lhs.Int <= rhs.Int, nil
		}
	case bytecode.TagFloat:
		switch op {
		case bytecode.OpLt:
			return lhs.Flt < rhs.Flt, nil
		case bytecode.OpLe:
			return lhs.Flt <= rhs.Flt, nil
		}
	case bytecode.TagString:
		switch op {
		case bytecode.OpLt:
			return lhs.AsString() < rhs.AsString(), nil
		case bytecode.OpLe:
			return lhs.AsString() <= rhs.AsString(), nil
		}
	}
	return false, fmt.Errorf("compare %s on %s not supported", op, lhs.Tag)
}

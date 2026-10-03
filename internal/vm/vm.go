package vm

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
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
	// Cause is the error the instruction failed with, when there was one
	// (an arithmetic fault, a builtin or eval-interop error). Unwrap exposes
	// it, so a typed error such as RT001's *errors.DivByZeroError stays
	// reachable with errors.As through the VM frame (#1449).
	Cause error
}

// Unwrap returns the underlying cause, or nil.
func (e *VMError) Unwrap() error { return e.Cause }

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

	// framePool is the free list of frames popped by RETURN, reused by every
	// frame push (acquireFrame/releaseFrame in frame.go, #1501).
	framePool []*Frame
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
	frame := vm.acquireFrame(proto, 0, nil)
	copy(frame.Regs, args)
	vm.Stack = append(vm.Stack, frame)
	defer func() { clear(vm.Stack); vm.Stack = vm.Stack[:0] }()
	return vm.run(frame)
}

// CallClosure invokes a closure value with the given arguments and returns
// the result. Used by HOF builtins to call their function arguments.
// This pushes a new frame onto the VM stack, runs it, and returns.
func (vm *VM) CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error) {
	if closure.Tag != bytecode.TagClosure {
		return bytecode.Value{}, fmt.Errorf("CallClosure: expected closure, got %s", closure.Tag)
	}
	c := closure.AsClosure()
	proto, ok := c.Proto.(*bytecode.FuncPrototype)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("CallClosure: non-FuncPrototype")
	}
	if int(proto.NumParams) != len(args) {
		return bytecode.Value{}, fmt.Errorf("CallClosure: %s expects %d args, got %d",
			proto.Name, proto.NumParams, len(args))
	}
	if proto.EvalOnly {
		if vm.Interop == nil {
			return bytecode.Value{}, fmt.Errorf("CallClosure: %s is evaluator-only (%s) but no interop bridge",
				proto.Name, proto.EvalReason)
		}
		return vm.Interop.CallEvalFunc(proto.Name, args)
	}
	if len(vm.Stack) >= vm.MaxStack {
		return bytecode.Value{}, ErrStackOverflow
	}
	// Push frame with Caller=nil so OpReturn returns the value directly
	// to vm.run, which returns it to us. The frame comes from the free list
	// and goes back to it on RETURN: a HOF builtin calls this once per
	// element, so it must not allocate (#1501).
	base := len(vm.Stack)
	frame := vm.acquireFrame(proto, 0, nil)
	copy(frame.Regs, args)
	for i, cap := range c.Captures {
		frame.Regs[int(proto.NumParams)+i] = cap
	}
	vm.Stack = append(vm.Stack, frame)
	result, err := vm.run(frame)
	if err != nil {
		// A fault unwinds with its frames still pushed; drop them so the
		// caller's stack depth is what it was before this call. They are
		// not recycled (releaseFrame is only for frames RETURN popped).
		clear(vm.Stack[base:])
		vm.Stack = vm.Stack[:base]
	}
	return result, err
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
				return bytecode.Value{}, vm.errWrap(frame, "", err, inst)
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
				return bytecode.Value{}, vm.errWrap(frame, "", err, inst)
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
					return bytecode.Value{}, vm.errWrap(frame, "CALL via eval interop: ", err, inst)
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
			newF := vm.acquireFrame(calleeProto, inst.A(), frame)
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
					return bytecode.Value{}, vm.errWrap(frame, "TAIL_CALL via eval interop: ", err, inst)
				}
				// Pop the current frame and write the result to the caller's
				// return register, mirroring OpReturn's path.
				vm.Stack = vm.Stack[:len(vm.Stack)-1]
				caller := frame.Caller
				if caller != nil {
					caller.Regs[frame.ReturnReg] = result
				}
				vm.releaseFrame(frame)
				if caller == nil {
					return result, nil
				}
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
			// Pop the current frame and recycle it: nothing references a
			// returned frame (closures capture values, not registers).
			vm.Stack = vm.Stack[:len(vm.Stack)-1]
			caller := frame.Caller
			if caller != nil {
				caller.Regs[frame.ReturnReg] = retVal
			}
			vm.releaseFrame(frame)
			if caller == nil {
				return retVal, nil
			}
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

		case bytecode.OpUpdateRecord:
			base := frame.Regs[inst.B()]
			if base.Tag != bytecode.TagRecord {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("UPDATE_RECORD: base is %s, not Record", base.Tag), inst)
			}
			count := int(inst.C())
			fields := append([]bytecode.RecordField(nil), base.AsRecord()...)
			for i := 0; i < count; i++ {
				nameInst := frame.Proto.Instructions[frame.IP+1+i]
				if nameInst.Op() != bytecode.OpLoadConst {
					return bytecode.Value{}, vm.errAt(frame, "UPDATE_RECORD: expected pseudo-LOAD_CONST for field name", nameInst)
				}
				nameVal, ok := frame.Proto.LookupConstant(int(nameInst.Bx()), vm.Image)
				if !ok || nameVal.Tag != bytecode.TagString {
					return bytecode.Value{}, vm.errAt(frame, "UPDATE_RECORD: bad field-name constant", nameInst)
				}
				name := nameVal.AsString()
				value := frame.Regs[int(inst.A())+1+i]
				replaced := false
				for j := range fields {
					if fields[j].Name == name {
						fields[j].Value = value
						replaced = true
						break
					}
				}
				if !replaced {
					fields = append(fields, bytecode.RecordField{Name: name, Value: value})
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
			nameInst := frame.Proto.Instructions[frame.IP+1]
			ctor, ok := frame.Proto.LookupConstant(int(nameInst.Bx()), vm.Image)
			if nameInst.Op() != bytecode.OpLoadConst || !ok || ctor.Tag != bytecode.TagString {
				return bytecode.Value{}, vm.errAt(frame, "MAKE_ADT: expected pseudo-LOAD_CONST for constructor name", nameInst)
			}
			frame.Regs[inst.A()] = bytecode.NewADT(tag, ctor.AsString(), fields)
			frame.IP += 2

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
				return bytecode.Value{}, vm.errWrap(frame, "BUILTIN_CALL: ", err, inst)
			}
			frame.Regs[inst.A()] = result
			frame.IP++
		case bytecode.OpBuiltinCallHOF:
			hofIdx := int(inst.B())
			argc := int(inst.C())
			if hofIdx >= len(HOFBuiltinTable) {
				return bytecode.Value{}, vm.errAt(frame, fmt.Sprintf("BUILTIN_CALL_HOF: unknown HOF builtin index %d", hofIdx), inst)
			}
			argBase := int(inst.A()) + 1
			args := frame.Regs[argBase : argBase+argc]
			result, err := HOFBuiltinTable[hofIdx](vm, args)
			if err != nil {
				return bytecode.Value{}, vm.errWrap(frame, "BUILTIN_CALL_HOF: ", err, inst)
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
			if err := ailerrors.CheckIntDivisor(ailerrors.OpDivision, r); err != nil {
				return bytecode.Value{}, err
			}
			return bytecode.NewInt(l / r), nil
		case bytecode.OpMod:
			if err := ailerrors.CheckIntDivisor(ailerrors.OpModulo, r); err != nil {
				return bytecode.Value{}, err
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

// runtimeEq is the equality used by OpEq: types.FloatEq (IEEE, NaN != NaN)
// for floats wherever they sit, bare or nested in a list, tuple, record or
// ADT (M-FLOAT-EQ-ONE-SEMANTICS, #1274). It used to fall through to
// Value.Equal for composites, whose NaN == NaN is for constant-pool dedup,
// so [nan] == [nan] was true on the VM while nan == nan was false.
// Cycle-safety: values are finite trees built at runtime.
func runtimeEq(lhs, rhs bytecode.Value) bool {
	if lhs.Tag != rhs.Tag {
		return false
	}
	switch lhs.Tag {
	case bytecode.TagFloat:
		return types.FloatEq(lhs.Flt, rhs.Flt)
	case bytecode.TagInt:
		return lhs.Int == rhs.Int
	case bytecode.TagBool:
		return lhs.Bool == rhs.Bool
	case bytecode.TagUnit:
		return true
	case bytecode.TagString:
		return lhs.AsString() == rhs.AsString()
	case bytecode.TagList:
		return runtimeEqAll(lhs.Obj.(*bytecode.ListObj).Elems, rhs.Obj.(*bytecode.ListObj).Elems)
	case bytecode.TagTuple:
		return runtimeEqAll(lhs.Obj.(*bytecode.TupleObj).Elems, rhs.Obj.(*bytecode.TupleObj).Elems)
	case bytecode.TagBytes:
		return string(lhs.AsBytes().B) == string(rhs.AsBytes().B)
	case bytecode.TagArray:
		a, b := lhs.AsArray(), rhs.AsArray()
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !runtimeEq(a.At(i), b.At(i)) {
				return false
			}
		}
		return true
	case bytecode.TagRecord:
		a, b := lhs.Obj.(*bytecode.RecordObj).Fields, rhs.Obj.(*bytecode.RecordObj).Fields
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].Name != b[i].Name || !runtimeEq(a[i].Value, b[i].Value) {
				return false
			}
		}
		return true
	case bytecode.TagADT:
		a, b := lhs.Obj.(*bytecode.ADTObj), rhs.Obj.(*bytecode.ADTObj)
		return a.Tag == b.Tag && runtimeEqAll(a.Fields, b.Fields)
	}
	// Closures: the type checker rejects Eq on functions; keep the old answer.
	return lhs.Equal(rhs)
}

func runtimeEqAll(a, b []bytecode.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !runtimeEq(a[i], b[i]) {
			return false
		}
	}
	return true
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
			return types.FloatLt(lhs.Flt, rhs.Flt), nil
		case bytecode.OpLe:
			return types.FloatLte(lhs.Flt, rhs.Flt), nil
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

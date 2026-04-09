package compiler

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// compileExpr lowers an expression and returns the register holding its value.
//
// The contract is: the returned register holds the result, and the caller is
// responsible for freeing it (via regs.freeTemp) when done — UNLESS the result
// is a pinned local (a VarRef to a parameter or named binding), in which case
// the register must NOT be freed. Callers detect this by checking whether the
// expression is a VarRef; the helper compileExprToFresh always returns a
// freshly-allocated, freeable register by emitting an OpMove if needed.
func (fc *funcCompiler) compileExpr(e stmt.Expr) (uint8, error) {
	switch e := e.(type) {
	case stmt.LitInt:
		return fc.compileLitInt(e)
	case stmt.LitFloat:
		return fc.compileLitFloat(e)
	case stmt.LitBool:
		return fc.compileLitBool(e)
	case stmt.LitString:
		return fc.compileLitString(e)
	case stmt.LitUnit:
		return fc.compileLitUnit()
	case stmt.VarRef:
		return fc.compileVarRef(e)
	case stmt.BinOp:
		return fc.compileBinOp(e)
	case stmt.UnOp:
		return fc.compileUnOp(e)
	case stmt.IfExpr:
		return fc.compileIfExpr(e)
	case stmt.Call:
		return fc.compileCall(e)
	case stmt.Lambda:
		return fc.compileLambda(e)
	case stmt.ListLit:
		return fc.compileListLit(e)
	case stmt.TupleLit:
		return fc.compileTupleLit(e)
	case stmt.Cons:
		return fc.compileCons(e)
	case stmt.RecordLit:
		return fc.compileRecordLit(e)
	case stmt.RecordUpdate:
		return fc.compileRecordUpdate(e)
	case stmt.FieldAccess:
		return fc.compileFieldAccess(e)
	case stmt.ADTConstructor:
		return fc.compileADTConstructor(e)
	case stmt.BuiltinCall:
		return fc.compileBuiltinCall(e)
	case stmt.GlobalRef:
		// A bare GlobalRef is a first-class function reference. Materialize
		// it as a closure value in a fresh register.
		canonical := canonicalFuncName(e.Module, e.Name)
		idx, ok := fc.funcIdx[canonical]
		if !ok {
			// Fall back to bare name for compatibility with programs that
			// registered prototypes without module prefixes.
			if barIdx, barOk := fc.funcIdx[e.Name]; barOk {
				idx = barIdx
			} else {
				return 0, fmt.Errorf("compiler: unknown global %q", canonical)
			}
		}
		dst, err := fc.regs.allocTemp()
		if err != nil {
			return 0, err
		}
		nestedIdx, err := fc.lookupNestedProto(idx)
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpClosure, dst, nestedIdx))
		return dst, nil
	}
	return 0, fmt.Errorf("compiler: unsupported expression %T", e)
}

// compileExprToFresh lowers an expression and guarantees the returned register
// is freshly allocated (and therefore safe to free). If the expression yields
// a pinned local, an OpMove is emitted to copy it into a temp.
func (fc *funcCompiler) compileExprToFresh(e stmt.Expr) (uint8, error) {
	if vr, ok := e.(stmt.VarRef); ok {
		src, ok := fc.locals.lookup(vr.Name)
		if !ok {
			return 0, fmt.Errorf("compiler: unbound variable %q", vr.Name)
		}
		dst, err := fc.regs.allocTemp()
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, dst, src, 0))
		return dst, nil
	}
	return fc.compileExpr(e)
}

// isPinned reports whether r is a parameter or named local. Pinned registers
// must not be passed to regs.freeTemp.
func (fc *funcCompiler) isPinned(r uint8) bool {
	for _, frame := range fc.locals.frames {
		for _, reg := range frame.names {
			if reg == r {
				return true
			}
		}
	}
	return false
}

// freeIfTemp returns r to the free list iff it is not a pinned local.
func (fc *funcCompiler) freeIfTemp(r uint8) {
	if !fc.isPinned(r) {
		fc.regs.freeTemp(r)
	}
}

// --- Literal lowering -------------------------------------------------------

func (fc *funcCompiler) compileLitInt(e stmt.LitInt) (uint8, error) {
	idx, err := fc.addLocalConst(bytecode.NewInt(e.Value))
	if err != nil {
		return 0, err
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, dst, idx))
	return dst, nil
}

func (fc *funcCompiler) compileLitFloat(e stmt.LitFloat) (uint8, error) {
	idx, err := fc.addLocalConst(bytecode.NewFloat(e.Value))
	if err != nil {
		return 0, err
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, dst, idx))
	return dst, nil
}

func (fc *funcCompiler) compileLitBool(e stmt.LitBool) (uint8, error) {
	idx, err := fc.addLocalConst(bytecode.NewBool(e.Value))
	if err != nil {
		return 0, err
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, dst, idx))
	return dst, nil
}

func (fc *funcCompiler) compileLitString(e stmt.LitString) (uint8, error) {
	idx, err := fc.addLocalConst(bytecode.NewString(e.Value))
	if err != nil {
		return 0, err
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, dst, idx))
	return dst, nil
}

func (fc *funcCompiler) compileLitUnit() (uint8, error) {
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpLoadNil, dst, 0, 0))
	return dst, nil
}

// --- VarRef -----------------------------------------------------------------

func (fc *funcCompiler) compileVarRef(e stmt.VarRef) (uint8, error) {
	if r, ok := fc.locals.lookup(e.Name); ok {
		return r, nil
	}
	// Fall back to a top-level function reference: materialize the closure.
	// Try the current module first (multi-module image), then the bare
	// name (single-module or cross-module helper registered bare).
	canonical := canonicalFuncName(fc.currentModule, e.Name)
	idx, ok := fc.funcIdx[canonical]
	if !ok {
		idx, ok = fc.funcIdx[e.Name]
	}
	if ok {
		dst, err := fc.regs.allocTemp()
		if err != nil {
			return 0, err
		}
		nestedIdx, err := fc.lookupNestedProto(idx)
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpClosure, dst, nestedIdx))
		return dst, nil
	}
	return 0, fmt.Errorf("compiler: unbound variable %q", e.Name)
}

// --- BinOp / UnOp (M1: arithmetic only) -------------------------------------

func (fc *funcCompiler) compileBinOp(e stmt.BinOp) (uint8, error) {
	// Short-circuit operators are control flow, not BinOp opcodes.
	if e.Op == stmt.OpAnd || e.Op == stmt.OpOr {
		return fc.compileShortCircuit(e)
	}
	// Comparisons may need operand swapping (Gt/Gte) or NOT wrapping (Neq).
	if isCmpOp(e.Op) {
		return fc.compileCmp(e)
	}

	op, ok := simpleBinOpcode(e.Op)
	if !ok {
		return 0, fmt.Errorf("compiler: unsupported BinOp %v", e.Op)
	}

	lhs, err := fc.compileExpr(e.Left)
	if err != nil {
		return 0, err
	}
	rhs, err := fc.compileExpr(e.Right)
	if err != nil {
		return 0, err
	}

	// Free operand temps before allocating the result so the dest can reuse
	// one of their slots — keeps register pressure tight on long expressions.
	lhsTemp := !fc.isPinned(lhs)
	rhsTemp := !fc.isPinned(rhs)
	if rhsTemp {
		fc.regs.freeTemp(rhs)
	}
	if lhsTemp {
		fc.regs.freeTemp(lhs)
	}

	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(op, dst, lhs, rhs))
	return dst, nil
}

// compileCmp lowers comparison operators using only OpEq, OpLt, OpLe.
//
//	Eq  → OpEq(a, b)
//	Neq → NOT(OpEq(a, b))
//	Lt  → OpLt(a, b)
//	Lte → OpLe(a, b)
//	Gt  → OpLt(b, a)        (operand swap)
//	Gte → OpLe(b, a)        (operand swap)
func (fc *funcCompiler) compileCmp(e stmt.BinOp) (uint8, error) {
	left, right := e.Left, e.Right
	var op bytecode.OpCode
	negate := false
	switch e.Op {
	case stmt.OpEq:
		op = bytecode.OpEq
	case stmt.OpNeq:
		op = bytecode.OpEq
		negate = true
	case stmt.OpLt:
		op = bytecode.OpLt
	case stmt.OpLte:
		op = bytecode.OpLe
	case stmt.OpGt:
		op = bytecode.OpLt
		left, right = right, left
	case stmt.OpGte:
		op = bytecode.OpLe
		left, right = right, left
	default:
		return 0, fmt.Errorf("compiler: not a comparison op: %v", e.Op)
	}

	lhs, err := fc.compileExpr(left)
	if err != nil {
		return 0, err
	}
	rhs, err := fc.compileExpr(right)
	if err != nil {
		return 0, err
	}
	if !fc.isPinned(rhs) {
		fc.regs.freeTemp(rhs)
	}
	if !fc.isPinned(lhs) {
		fc.regs.freeTemp(lhs)
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(op, dst, lhs, rhs))
	if negate {
		fc.emit(bytecode.EncodeABC(bytecode.OpNot, dst, dst, 0))
	}
	return dst, nil
}

func isCmpOp(k stmt.BinOpKind) bool {
	switch k {
	case stmt.OpEq, stmt.OpNeq, stmt.OpLt, stmt.OpLte, stmt.OpGt, stmt.OpGte:
		return true
	}
	return false
}

// simpleBinOpcode maps a Statement IR BinOpKind to a single-instruction
// bytecode opcode (arithmetic + string concat). Comparisons, short-circuit
// logic, and any kind requiring multi-instruction lowering are NOT handled
// here — those have dedicated paths.
func simpleBinOpcode(k stmt.BinOpKind) (bytecode.OpCode, bool) {
	switch k {
	case stmt.OpAdd:
		return bytecode.OpAdd, true
	case stmt.OpSub:
		return bytecode.OpSub, true
	case stmt.OpMul:
		return bytecode.OpMul, true
	case stmt.OpDiv:
		return bytecode.OpDiv, true
	case stmt.OpMod:
		return bytecode.OpMod, true
	case stmt.OpConcat:
		return bytecode.OpConcat, true
	}
	return 0, false
}

func (fc *funcCompiler) compileUnOp(e stmt.UnOp) (uint8, error) {
	var op bytecode.OpCode
	switch e.Op {
	case stmt.OpNeg:
		op = bytecode.OpNeg
	case stmt.OpNot:
		op = bytecode.OpNot
	default:
		return 0, fmt.Errorf("compiler: unsupported UnOp %v", e.Op)
	}
	src, err := fc.compileExpr(e.Operand)
	if err != nil {
		return 0, err
	}
	if !fc.isPinned(src) {
		fc.regs.freeTemp(src)
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(op, dst, src, 0))
	return dst, nil
}

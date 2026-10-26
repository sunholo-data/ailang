package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// encodeIntrinsic handles pre-lowered Intrinsic nodes.
// After op_lowering these should not appear, but handle them for robustness.
func encodeIntrinsic(intr *core.Intrinsic) (string, error) {
	opMap := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*",
		core.OpDiv: smtIntDivTrunc, core.OpMod: smtIntModTrunc,
		core.OpEq: "=", core.OpNe: "distinct",
		core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpAnd: "and", core.OpOr: "or", core.OpNot: "not",
		core.OpNeg:        "-",
		core.OpConcat:     "str.++",
		core.OpBitwiseAnd: "bvand", core.OpBitwiseXor: "bvxor",
		core.OpBitwiseNot: "bvnot",
		core.OpShiftLeft:  "bvshl", core.OpShiftRight: "bvashr",
	}
	smtOp, ok := opMap[intr.Op]
	if !ok {
		return "", fmt.Errorf("unsupported intrinsic op: %v", intr.Op)
	}
	return encodeBuiltinOp(smtOp, intr.Args)
}

// encodeBinOp handles pre-lowered BinOp nodes.
func encodeBinOp(bop *core.BinOp) (string, error) {
	opMap := map[string]string{
		"+": "+", "-": "-", "*": "*", "/": smtIntDivTrunc, "%": smtIntModTrunc,
		"==": "=", "!=": "distinct",
		"<": "<", "<=": "<=", ">": ">", ">=": ">=",
		"&&": "and", "||": "or",
		"&": "bvand", "^": "bvxor",
		"<<": "bvshl", ">>": "bvashr",
	}
	smtOp, ok := opMap[bop.Op]
	if !ok {
		return "", fmt.Errorf("unsupported binary op: %s", bop.Op)
	}
	left, err := EncodeExpr(bop.Left)
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(bop.Right)
	if err != nil {
		return "", err
	}
	return applyBinaryOp(smtOp, left, right), nil
}

// smtIntDivTrunc and smtIntModTrunc stand for AILANG's Int `/` and `%`. They are
// not SMT-LIB operators: applyBinaryOp expands them through encodeTruncDivMod.
// SMT-LIB `div`/`mod` are Euclidean (-7 div 2 = -4, -7 mod 2 = 1), but AILANG
// follows Go and truncates toward zero (-7 / 2 = -3, -7 % 2 = -1). Emitting the
// SMT operators directly proved contracts that are false at runtime (#1960).
const (
	smtIntDivTrunc = "$ail.idiv"
	smtIntModTrunc = "$ail.imod"
)

// applyBinaryOp renders `(op left right)`, expanding the truncating Int
// division and remainder sentinels.
func applyBinaryOp(smtOp, left, right string) string {
	if smtOp == smtIntDivTrunc || smtOp == smtIntModTrunc {
		return encodeTruncDivMod(smtOp == smtIntModTrunc, left, right)
	}
	return fmt.Sprintf("(%s %s %s)", smtOp, left, right)
}

// encodeTruncDivMod encodes Go-semantics Int division, or the remainder when isMod:
//
//	q = (ite (= (>= n 0) (>= d 0)) (div |n| |d|) (- (div |n| |d|)))   ; rounds toward zero
//	r = (- n (* d q))                                                  ; sign follows n
//
// The operands are bound once by a parallel let, so a compound operand is encoded
// once and cannot be captured by the let names of a nested division. Division by
// zero stays uninterpreted, as before: the runtime raises RT001 there, so the
// function returns nothing and any value is sound for partial correctness.
func encodeTruncDivMod(isMod bool, left, right string) string {
	const n, d = "$ail_n", "$ail_d"
	q := fmt.Sprintf("(ite (= (>= %[1]s 0) (>= %[2]s 0)) (div (abs %[1]s) (abs %[2]s)) (- (div (abs %[1]s) (abs %[2]s))))", n, d)
	body := q
	if isMod {
		body = fmt.Sprintf("(- %s (* %s %s))", n, d, q)
	}
	return fmt.Sprintf("(let ((%s %s) (%s %s)) %s)", n, left, d, right, body)
}

// encodeUnOp handles pre-lowered UnOp nodes.
func encodeUnOp(uop *core.UnOp) (string, error) {
	opMap := map[string]string{
		"not": "not", "-": "-",
	}
	smtOp, ok := opMap[uop.Op]
	if !ok {
		return "", fmt.Errorf("unsupported unary op: %s", uop.Op)
	}
	operand, err := EncodeExpr(uop.Operand)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s)", smtOp, operand), nil
}

// encodeDictApp handles DictApp nodes — type class method applications.
// DictApp appears when dictionary passing is used instead of op_lowering.
// It maps type class method names (e.g., "ge", "add") to SMT-LIB operators.
func encodeDictApp(da *core.DictApp) (string, error) {
	// Map type class method names to SMT-LIB operators
	methodToSMT := map[string]string{
		// Num class
		"add": "+", "sub": "-", "mul": "*", "div": smtIntDivTrunc, "neg": "-",
		// Eq class
		"eq": "=", "neq": "distinct", "ne": "distinct",
		// Ord class
		"lt": "<", "lte": "<=", "le": "<=",
		"gt": ">", "gte": ">=", "ge": ">=",
		// Bool (if accessed via type class)
		"and": "and", "or": "or", "not": "not",
	}

	smtOp, ok := methodToSMT[da.Method]
	if !ok {
		return "", fmt.Errorf("unsupported type class method %q in SMT encoding", da.Method)
	}

	return encodeBuiltinOp(smtOp, da.Args)
}

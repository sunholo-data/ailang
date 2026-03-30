package smt

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// encodeIntrinsic handles pre-lowered Intrinsic nodes.
// After op_lowering these should not appear, but handle them for robustness.
func encodeIntrinsic(intr *core.Intrinsic) (string, error) {
	opMap := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*",
		core.OpDiv: "div", core.OpMod: "mod",
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
		"+": "+", "-": "-", "*": "*", "/": "div", "%": "mod",
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
	return fmt.Sprintf("(%s %s %s)", smtOp, left, right), nil
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
		"add": "+", "sub": "-", "mul": "*", "div": "div", "neg": "-",
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

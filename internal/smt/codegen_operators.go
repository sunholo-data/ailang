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
		core.OpDiv: "/", core.OpMod: "%",
		core.OpEq: "=", core.OpNe: "distinct",
		core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpAnd: "and", core.OpOr: "or", core.OpNot: "not",
		core.OpNeg:        "-",
		core.OpConcat:     "str.++",
		core.OpBitwiseAnd: "bvand", core.OpBitwiseXor: "bvxor",
		core.OpBitwiseNot: "bvnot",
		core.OpShiftLeft:  "bvshl", core.OpShiftRight: "bvashr",
	}
	op, ok := opMap[intr.Op]
	if !ok {
		return "", fmt.Errorf("unsupported intrinsic op: %v", intr.Op)
	}
	if len(intr.Args) == 2 {
		smtOp, flip, err := resolvePreLoweredOp(op, intr.Args[0], intr.Args[1])
		if err != nil {
			return "", err
		}
		return encodeBinaryArgs(smtOp, flip, intr.Args[0], intr.Args[1])
	}
	return encodeBuiltinOp(op, intr.Args)
}

// encodeBinOp handles pre-lowered BinOp nodes.
func encodeBinOp(bop *core.BinOp) (string, error) {
	opMap := map[string]string{
		"+": "+", "-": "-", "*": "*", "/": "/", "%": "%",
		"==": "=", "!=": "distinct",
		"<": "<", "<=": "<=", ">": ">", ">=": ">=",
		"&&": "and", "||": "or",
		"&": "bvand", "^": "bvxor",
		"<<": "bvshl", ">>": "bvashr",
	}
	op, ok := opMap[bop.Op]
	if !ok {
		return "", fmt.Errorf("unsupported binary op: %s", bop.Op)
	}
	smtOp, flip, err := resolvePreLoweredOp(op, bop.Left, bop.Right)
	if err != nil {
		return "", err
	}
	return encodeBinaryArgs(smtOp, flip, bop.Left, bop.Right)
}

// resolvePreLoweredOp picks the sort-directed encoding of `/`, `%` and the
// ordering operators on pre-lowered Intrinsic/BinOp nodes. These shapes do not
// reach the encoder after op_lowering (pinned by the contract pipeline test);
// when neither operand carries a sort they keep the historical Int reading,
// which int_divmod_test.go pins. Any Float or String evidence wins.
func resolvePreLoweredOp(op string, left, right core.CoreExpr) (string, bool, error) {
	sort := operandsSort(left, right)
	switch op {
	case "/", "%":
		if sort == "" {
			sort = sortInt
		}
		smtOp, err := divModOp(op == "%", sort)
		return smtOp, false, err
	case "<", "<=", ">", ">=":
		if sort == sortString {
			return orderOp(op, sort)
		}
	}
	return op, false, nil
}

// encodeBinaryArgs encodes both operands once and applies smtOp, swapping the
// operands when flip is set.
func encodeBinaryArgs(smtOp string, flip bool, left, right core.CoreExpr) (string, error) {
	l, err := EncodeExpr(left)
	if err != nil {
		return "", err
	}
	r, err := EncodeExpr(right)
	if err != nil {
		return "", err
	}
	if flip {
		l, r = r, l
	}
	return applyBinaryOp(smtOp, l, r), nil
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
// division and remainder sentinels and the Float remainder sentinel.
func applyBinaryOp(smtOp, left, right string) string {
	if smtOp == smtIntDivTrunc || smtOp == smtIntModTrunc {
		return encodeTruncDivMod(smtOp == smtIntModTrunc, left, right)
	}
	if smtOp == smtRealModTrunc {
		return encodeRealMod(left, right)
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
// DictApp is the normal post-pipeline shape for an operator with a resolved
// class constraint (e.g. float `/` is DictApp{Fractional Float, "div"}). Methods
// whose meaning depends on the instance type — division, remainder, ordering —
// dispatch on the dictionary's TypeName; an unknown instance skips (#2062).
func encodeDictApp(da *core.DictApp) (string, error) {
	sortAgnostic := map[string]string{
		// Num class (+, -, * and negation share one spelling on Int and Real)
		"add": "+", "sub": "-", "mul": "*", "neg": "-",
		// Eq class
		"eq": "=", "neq": "distinct", "ne": "distinct",
		// Bool (if accessed via type class)
		"and": "and", "or": "or", "not": "not",
	}
	ordering := map[string]string{
		"lt": "<", "lte": "<=", "le": "<=",
		"gt": ">", "gte": ">=", "ge": ">=",
	}

	if smtOp, ok := sortAgnostic[da.Method]; ok {
		return encodeBuiltinOp(smtOp, da.Args)
	}
	sort := dictSort(da)
	switch {
	case da.Method == "div" || da.Method == "mod":
		smtOp, err := divModOp(da.Method == "mod", sort)
		if err != nil {
			return "", err
		}
		return encodeBuiltinOp(smtOp, da.Args)
	case ordering[da.Method] != "":
		smtOp, flip, err := orderOp(ordering[da.Method], sort)
		if err != nil {
			return "", err
		}
		if len(da.Args) != 2 {
			return "", fmt.Errorf("type class method %q with unexpected arity %d", da.Method, len(da.Args))
		}
		return encodeBinaryArgs(smtOp, flip, da.Args[0], da.Args[1])
	}
	return "", fmt.Errorf("unsupported type class method %q in SMT encoding", da.Method)
}

package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Simple expression normalization: literals, lambdas, operators

// normalizeLiteral handles literals
func (e *Elaborator) normalizeLiteral(lit *ast.Literal) (core.CoreExpr, error) {
	var kind core.LitKind
	switch lit.Kind {
	case ast.IntLit:
		kind = core.IntLit
	case ast.FloatLit:
		kind = core.FloatLit
	case ast.StringLit:
		kind = core.StringLit
	case ast.BoolLit:
		kind = core.BoolLit
	case ast.UnitLit:
		kind = core.UnitLit
	default:
		return nil, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}

	return &core.Lit{
		CoreNode: e.makeNode(lit.Position()),
		Kind:     kind,
		Value:    lit.Value,
	}, nil
}

// normalizeLambda handles lambda expressions
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
	// Extract parameter names
	params := make([]string, len(lam.Params))
	for i, p := range lam.Params {
		params[i] = p.Name
	}

	// Normalize body
	body, err := e.normalize(lam.Body)
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(lam.Position()),
		Params:   params,
		Body:     body,
	}

	// Store effect annotations if present
	if len(lam.Effects) > 0 {
		// Extract effect names for backward compatibility
		effectNames := ast.EffectNames(lam.Effects)
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = effectNames
		// M-CAPABILITY-BUDGETS: Also store full annotations with budgets
		e.effectAnnotsFull[coreLam.ID()] = lam.Effects
	}

	return coreLam, nil
}

// normalizeFuncLit handles function literal expressions (func(x) -> T { body })
// Desugars to Lambda: func(x: int) -> int { x + 1 } ≡ \x. x + 1
func (e *Elaborator) normalizeFuncLit(funcLit *ast.FuncLit) (core.CoreExpr, error) {
	// Extract parameter names (type annotations are handled by type checker)
	params := make([]string, len(funcLit.Params))
	for i, p := range funcLit.Params {
		params[i] = p.Name
	}

	// Normalize body
	body, err := e.normalize(funcLit.Body)
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(funcLit.Position()),
		Params:   params,
		Body:     body,
	}

	// Store effect annotations if present
	if len(funcLit.Effects) > 0 {
		// Extract effect names for backward compatibility
		effectNames := ast.EffectNames(funcLit.Effects)
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = effectNames
		// M-CAPABILITY-BUDGETS: Also store full annotations with budgets
		e.effectAnnotsFull[coreLam.ID()] = funcLit.Effects
	}

	return coreLam, nil
}

// normalizeBinaryOp handles binary operations with ANF transformation
func (e *Elaborator) normalizeBinaryOp(binop *ast.BinaryOp) (core.CoreExpr, error) {
	// Normalize operands to atomic values
	left, leftBinds, err := e.normalizeToAtomic(binop.Left)
	if err != nil {
		return nil, err
	}

	right, rightBinds, err := e.normalizeToAtomic(binop.Right)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch binop.Op {
	case "+":
		op = core.OpAdd
	case "-":
		op = core.OpSub
	case "*":
		op = core.OpMul
	case "/":
		op = core.OpDiv
	case "%":
		op = core.OpMod
	case "==":
		op = core.OpEq
	case "!=":
		op = core.OpNe
	case "<":
		op = core.OpLt
	case "<=":
		op = core.OpLe
	case ">":
		op = core.OpGt
	case ">=":
		op = core.OpGe
	case "++":
		op = core.OpConcat
	case "&&":
		op = core.OpAnd
	case "||":
		op = core.OpOr
	case "&":
		op = core.OpBitwiseAnd
	case "^":
		op = core.OpBitwiseXor
	case "<<":
		op = core.OpShiftLeft
	case ">>":
		op = core.OpShiftRight
	default:
		// For compatibility, still create BinOp for unknown operators
		result := &core.BinOp{
			CoreNode: e.makeNode(binop.Position()),
			Op:       binop.Op,
			Left:     left,
			Right:    right,
		}
		return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(binop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{left, right},
	}

	// Wrap with let bindings from normalization
	return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
}

// normalizeUnaryOp handles unary operations
func (e *Elaborator) normalizeUnaryOp(unop *ast.UnaryOp) (core.CoreExpr, error) {
	operand, binds, err := e.normalizeToAtomic(unop.Expr)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch unop.Op {
	case "-":
		op = core.OpNeg
	case "not":
		op = core.OpNot
	case "~":
		op = core.OpBitwiseNot
	default:
		// For compatibility, still create UnOp for unknown operators
		result := &core.UnOp{
			CoreNode: e.makeNode(unop.Position()),
			Op:       unop.Op,
			Operand:  operand,
		}
		return e.wrapWithBindings(result, binds), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(unop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{operand},
	}

	return e.wrapWithBindings(result, binds), nil
}

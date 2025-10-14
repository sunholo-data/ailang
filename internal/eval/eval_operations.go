package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// evalCoreApp evaluates function application
func (e *CoreEvaluator) evalCoreApp(app *core.App) (Value, error) {
	// Evaluate function
	fnVal, err := e.evalCore(app.Func)
	if err != nil {
		return nil, err
	}

	// Force IndirectValue if needed (for LetRec recursion)
	if iv, ok := fnVal.(*IndirectValue); ok {
		fnVal, err = iv.Force()
		if err != nil {
			return nil, err
		}
	}

	// Evaluate arguments
	var args []Value
	for _, arg := range app.Args {
		argVal, err := e.evalCore(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// Apply function
	switch fn := fnVal.(type) {
	case *FunctionValue:
		// Recursion depth guard
		e.recursionDepth++
		if e.recursionDepth > e.maxRecursionDepth {
			e.recursionDepth--
			return nil, fmt.Errorf("RT_REC_003: max recursion depth %d exceeded. Try smaller input, enable tail recursion, or increase with --max-recursion-depth", e.maxRecursionDepth)
		}
		defer func() { e.recursionDepth-- }()

		if len(args) != len(fn.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
		}

		// Create new environment with parameters bound
		newEnv := fn.Env.Clone()
		for i, param := range fn.Params {
			newEnv.Set(param, args[i])
		}

		// Evaluate body
		oldEnv := e.env
		e.env = newEnv

		// Body could be Core or TypedAST depending on origin
		var result Value
		if coreBody, ok := fn.Body.(core.CoreExpr); ok {
			result, err = e.evalCore(coreBody)
		} else {
			return nil, fmt.Errorf("function body is not Core AST")
		}

		e.env = oldEnv
		return result, err

	case *BuiltinFunction:
		return fn.Fn(args)

	default:
		return nil, fmt.Errorf("cannot apply non-function value: %T", fnVal)
	}
}

// evalCoreBinOp evaluates binary operation
func (e *CoreEvaluator) evalCoreBinOp(binop *core.BinOp) (Value, error) {
	// Evaluate operands
	leftVal, err := e.evalCore(binop.Left)
	if err != nil {
		return nil, err
	}

	rightVal, err := e.evalCore(binop.Right)
	if err != nil {
		return nil, err
	}

	// Apply operation based on operator and types
	return e.applyBinOp(binop.Op, leftVal, rightVal)
}

// evalCoreUnOp evaluates unary operation
func (e *CoreEvaluator) evalCoreUnOp(unop *core.UnOp) (Value, error) {
	// Evaluate operand
	operandVal, err := e.evalCore(unop.Operand)
	if err != nil {
		return nil, err
	}

	// Apply operation
	return applyUnOp(unop.Op, operandVal)
}

// evalIntrinsic evaluates an intrinsic operation
// This should typically be handled by OpLowering pass, but we provide
// a fallback implementation using the experimental binop shim
func (e *CoreEvaluator) evalIntrinsic(intrinsic *core.Intrinsic) (Value, error) {
	// Evaluate arguments
	args := make([]Value, len(intrinsic.Args))
	for i, arg := range intrinsic.Args {
		val, err := e.evalCore(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Map intrinsic to operator for shim
	if e.experimentalBinopShim {
		// Binary operations
		if len(args) == 2 {
			var op string
			switch intrinsic.Op {
			case core.OpAdd:
				op = "+"
			case core.OpSub:
				op = "-"
			case core.OpMul:
				op = "*"
			case core.OpDiv:
				op = "/"
			case core.OpMod:
				op = "%"
			case core.OpEq:
				op = "=="
			case core.OpNe:
				op = "!="
			case core.OpLt:
				op = "<"
			case core.OpLe:
				op = "<="
			case core.OpGt:
				op = ">"
			case core.OpGe:
				op = ">="
			case core.OpConcat:
				op = "++"
			case core.OpAnd:
				op = "&&"
			case core.OpOr:
				op = "||"
			default:
				return nil, fmt.Errorf("unknown intrinsic operation: %v", intrinsic.Op)
			}
			return e.applyBinOp(op, args[0], args[1])
		}

		// Unary operations
		if len(args) == 1 {
			var op string
			switch intrinsic.Op {
			case core.OpNot:
				op = "not"
			case core.OpNeg:
				op = "-"
			default:
				return nil, fmt.Errorf("unknown unary intrinsic: %v", intrinsic.Op)
			}
			return applyUnOp(op, args[0])
		}
	}

	return nil, fmt.Errorf("intrinsic operations require OpLowering pass or --experimental-binop-shim flag")
}

// applyBinOp should NOT be called in dictionary-passing system except for special operators
// This is a fail-fast guard to ensure BinOp nodes are properly elaborated to DictApp
func (e *CoreEvaluator) applyBinOp(op string, left, right Value) (Value, error) {
	// Special case: string concatenation doesn't use type classes
	if op == "++" {
		lStr, lOk := left.(*StringValue)
		rStr, rOk := right.(*StringValue)
		if !lOk || !rOk {
			return nil, fmt.Errorf("'++' requires string operands")
		}
		return &StringValue{Value: lStr.Value + rStr.Value}, nil
	}

	// Special case: boolean operators don't use type classes
	if op == "&&" || op == "||" {
		lBool, lOk := left.(*BoolValue)
		rBool, rOk := right.(*BoolValue)
		if !lOk || !rOk {
			return nil, fmt.Errorf("'%s' requires boolean operands", op)
		}

		switch op {
		case "&&":
			return &BoolValue{Value: lBool.Value && rBool.Value}, nil
		case "||":
			return &BoolValue{Value: lBool.Value || rBool.Value}, nil
		}
	}

	// Experimental operator shim for basic arithmetic
	if e.experimentalBinopShim {
		// Try Int operations
		if lInt, lOk := left.(*IntValue); lOk {
			if rInt, rOk := right.(*IntValue); rOk {
				switch op {
				case "+":
					return &IntValue{Value: lInt.Value + rInt.Value}, nil
				case "-":
					return &IntValue{Value: lInt.Value - rInt.Value}, nil
				case "*":
					return &IntValue{Value: lInt.Value * rInt.Value}, nil
				case "/":
					if rInt.Value == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return &IntValue{Value: lInt.Value / rInt.Value}, nil
				case "%":
					if rInt.Value == 0 {
						return nil, fmt.Errorf("modulo by zero")
					}
					return &IntValue{Value: lInt.Value % rInt.Value}, nil
				case "==":
					return &BoolValue{Value: lInt.Value == rInt.Value}, nil
				case "!=":
					return &BoolValue{Value: lInt.Value != rInt.Value}, nil
				case "<":
					return &BoolValue{Value: lInt.Value < rInt.Value}, nil
				case ">":
					return &BoolValue{Value: lInt.Value > rInt.Value}, nil
				case "<=":
					return &BoolValue{Value: lInt.Value <= rInt.Value}, nil
				case ">=":
					return &BoolValue{Value: lInt.Value >= rInt.Value}, nil
				}
			}
		}

		// Try Float operations
		if lFloat, lOk := left.(*FloatValue); lOk {
			if rFloat, rOk := right.(*FloatValue); rOk {
				switch op {
				case "+":
					return &FloatValue{Value: lFloat.Value + rFloat.Value}, nil
				case "-":
					return &FloatValue{Value: lFloat.Value - rFloat.Value}, nil
				case "*":
					return &FloatValue{Value: lFloat.Value * rFloat.Value}, nil
				case "/":
					if rFloat.Value == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return &FloatValue{Value: lFloat.Value / rFloat.Value}, nil
				case "==":
					return &BoolValue{Value: lFloat.Value == rFloat.Value}, nil
				case "!=":
					return &BoolValue{Value: lFloat.Value != rFloat.Value}, nil
				case "<":
					return &BoolValue{Value: lFloat.Value < rFloat.Value}, nil
				case ">":
					return &BoolValue{Value: lFloat.Value > rFloat.Value}, nil
				case "<=":
					return &BoolValue{Value: lFloat.Value <= rFloat.Value}, nil
				case ">=":
					return &BoolValue{Value: lFloat.Value >= rFloat.Value}, nil
				}
			}
		}
	}

	// All other operators must go through dictionary elaboration
	return nil, fmt.Errorf("internal: BinOp reached evaluator; dictionaries not elaborated (op='%s')", op)
}

// applyUnOp applies a unary operator to a value
func applyUnOp(op string, operand Value) (Value, error) {
	switch op {
	case "-":
		switch v := operand.(type) {
		case *IntValue:
			return &IntValue{Value: -v.Value}, nil
		case *FloatValue:
			return &FloatValue{Value: -v.Value}, nil
		}

	case "!":
		if v, ok := operand.(*BoolValue); ok {
			return &BoolValue{Value: !v.Value}, nil
		}
	}

	return nil, fmt.Errorf("cannot apply unary operator %s to %T", op, operand)
}

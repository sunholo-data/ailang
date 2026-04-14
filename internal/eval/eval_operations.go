package eval

import (
	"fmt"
	"log"
	"os"

	"github.com/sunholo/ailang/internal/core"
)

// debugEvalApp enables debug output for function application when DEBUG_EVAL_APP=1
var debugEvalApp = os.Getenv("DEBUG_EVAL_APP") == "1"

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
	if debugEvalApp {
		funcDesc := fmt.Sprintf("%T", fnVal)
		if bf, ok := fnVal.(*BuiltinFunction); ok {
			funcDesc = fmt.Sprintf("Builtin(%s)", bf.Name)
		}
		argStrs := make([]string, len(args))
		for i, a := range args {
			argStrs[i] = fmt.Sprintf("%T(%s)", a, a.String())
		}
		log.Printf("[DEBUG_EVAL_APP] Apply %s to %v", funcDesc, argStrs)
	}
	switch fn := fnVal.(type) {
	case *FunctionValue:
		// Recursion depth guard
		e.recursionDepth++
		if e.recursionDepth > e.maxRecursionDepth {
			e.recursionDepth--
			return nil, fmt.Errorf("RT_REC_003: max recursion depth %d exceeded. Try smaller input, enable tail recursion, or increase with --max-recursion-depth", e.maxRecursionDepth)
		}
		defer func() { e.recursionDepth-- }()

		// M-DOCPARSE-DX M1: Auto-curry support.
		// When more args than params (e.g., f(a, b) where f = \x. \y. ...),
		// apply first batch to get intermediate function, then apply rest.
		if len(args) > len(fn.Params) {
			// Apply first len(fn.Params) args
			firstArgs := args[:len(fn.Params)]
			restArgs := args[len(fn.Params):]

			newEnv := fn.Env.NewChildEnvironment()
			for i, param := range fn.Params {
				newEnv.Set(param, firstArgs[i])
			}

			oldEnv := e.env
			e.env = newEnv
			// M-DX-XPKG-RESOLVE: fallback resolver for curry body
			// M-PERF6-PHASE4 M2a: skip re-wrap if chain already covers fn.Resolver
			var oldResolver GlobalResolver
			if fn.Resolver != nil && !resolverCovers(e.resolver, fn.Resolver) {
				oldResolver = e.resolver
				e.resolver = &FallbackResolver{
					Primary:   e.resolver,
					Secondary: fn.Resolver,
				}
			}
			var intermediate Value
			if coreBody, ok := fn.Body.(core.CoreExpr); ok {
				intermediate, err = e.evalCore(coreBody)
			} else {
				e.env = oldEnv
				if oldResolver != nil {
					e.resolver = oldResolver
				}
				return nil, fmt.Errorf("function body is not Core AST")
			}
			e.env = oldEnv
			if oldResolver != nil {
				e.resolver = oldResolver
			}
			if err != nil {
				return nil, err
			}

			// Apply remaining args to the intermediate result
			if innerFn, ok := intermediate.(*FunctionValue); ok {
				return e.applyFunction(innerFn, restArgs)
			}
			return nil, fmt.Errorf("function expects %d arguments, got %d (intermediate result is not a function)", len(fn.Params), len(args))
		}

		if len(args) < len(fn.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
		}

		// M-TRACE-EXPORT: Record function entry
		funcName := extractFuncName(app.Func)
		if recorder, ok := e.effContext.(TraceRecorder); ok && recorder.HasTraceCollector() {
			argStrs := make([]string, len(args))
			for i, a := range args {
				argStrs[i] = a.String()
			}
			recorder.RecordFunctionEnter(funcName, argStrs)
		}

		// Create new environment with parameters bound
		newEnv := fn.Env.NewChildEnvironment()
		for i, param := range fn.Params {
			newEnv.Set(param, args[i])
		}

		// M-CAPABILITY-BUDGETS: Set up budget scoping if function has effect budgets
		var oldEffContext interface{}
		hasBudgetScope := (len(fn.EffectBudgets) > 0 || len(fn.EffectMinBudgets) > 0) && e.effContext != nil
		if hasBudgetScope {
			// Use BudgetEnforcer interface to avoid import cycle
			if enforcer, ok := e.effContext.(BudgetEnforcer); ok {
				oldEffContext = e.effContext
				e.effContext = enforcer.WithBudgetLimits(fn.EffectBudgets)

				// M-DX25 M4: Set min budgets if present
				if len(fn.EffectMinBudgets) > 0 {
					if minEnforcer, ok := e.effContext.(MinBudgetEnforcer); ok {
						minEnforcer.SetMinBudgets(fn.EffectMinBudgets)
					}
				}
			}
		}

		// Evaluate body
		oldEnv := e.env
		e.env = newEnv

		// M-DX-XPKG-RESOLVE: Create a fallback resolver that tries the caller's
		// resolver first (for builtins/effect context), then falls back to the
		// function's defining module resolver (for constructors like Some/None
		// that the caller might not have in scope).
		// M-PERF6-PHASE4 M2a: skip re-wrap if chain already covers fn.Resolver
		var oldResolver GlobalResolver
		if fn.Resolver != nil && !resolverCovers(e.resolver, fn.Resolver) {
			oldResolver = e.resolver
			e.resolver = &FallbackResolver{
				Primary:   e.resolver,
				Secondary: fn.Resolver,
			}
		}

		// M-VERIFY-CONTRACTS: Check preconditions before executing body
		if err := e.checkPreconditions(fn); err != nil {
			e.env = oldEnv
			if oldResolver != nil {
				e.resolver = oldResolver
			}
			return nil, err
		}

		// Body could be Core or TypedAST depending on origin
		var result Value
		if coreBody, ok := fn.Body.(core.CoreExpr); ok {
			result, err = e.evalCore(coreBody)
		} else {
			if oldResolver != nil {
				e.resolver = oldResolver
			}
			return nil, fmt.Errorf("function body is not Core AST")
		}

		// M-VERIFY-CONTRACTS: Check postconditions before returning (if no error)
		if err == nil {
			if postErr := e.checkPostconditions(fn, result); postErr != nil {
				e.env = oldEnv
				if oldResolver != nil {
					e.resolver = oldResolver
				}
				return nil, postErr
			}
		}

		e.env = oldEnv
		if oldResolver != nil {
			e.resolver = oldResolver
		}

		// M-TRACE-EXPORT: Record function exit
		if recorder, ok := e.effContext.(TraceRecorder); ok && recorder.HasTraceCollector() {
			resultStr := ""
			if result != nil {
				resultStr = result.String()
			}
			recorder.RecordFunctionExit(funcName, resultStr)
		}

		// M-DX25: Handle budget scope exit
		if oldEffContext != nil {
			// M-DX25 M4: Check minimums before restoring context (if no error already)
			if err == nil {
				if checker, ok := e.effContext.(MinimumChecker); ok {
					err = checker.CheckMinimums("")
				}
			}

			// M-DX25: Charge caller with callee's declared budget
			if charger, ok := e.effContext.(ScopeCharger); ok {
				charger.PopScopeAndChargeCaller()
			}
			e.effContext = oldEffContext
		}

		if debugEvalApp {
			if result != nil {
				log.Printf("[DEBUG_EVAL_APP] FunctionValue returned %T(%s)", result, result.String())
			} else {
				log.Printf("[DEBUG_EVAL_APP] FunctionValue returned nil, err=%v", err)
			}
		}
		return result, err

	case *BuiltinFunction:
		result, err := fn.Fn(args)
		if debugEvalApp {
			if result != nil {
				log.Printf("[DEBUG_EVAL_APP] Builtin(%s) returned %T(%s)", fn.Name, result, result.String())
			} else {
				log.Printf("[DEBUG_EVAL_APP] Builtin(%s) returned nil, err=%v", fn.Name, err)
			}
		}
		return result, err

	case *ConstructorClosure:
		// ADT constructor application - creates a TaggedValue
		return fn.Apply(args)

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
			case core.OpBitwiseAnd:
				op = "&"
			case core.OpBitwiseXor:
				op = "^"
			case core.OpShiftLeft:
				op = "<<"
			case core.OpShiftRight:
				op = ">>"
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
			case core.OpBitwiseNot:
				op = "~"
			default:
				return nil, fmt.Errorf("unknown unary intrinsic: %v", intrinsic.Op)
			}
			return applyUnOp(op, args[0])
		}
	}

	// M-CONTRACT-OPLOWERING-FIX: Comparison/equality ops may be deferred by the lowerer
	// when CoreTI can't resolve their type (e.g., inside contract expressions).
	// Handle them here with runtime type dispatch rather than requiring the shim flag.
	if len(args) == 2 {
		var op string
		switch intrinsic.Op {
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
		}
		if op != "" {
			return e.applyBinOp(op, args[0], args[1])
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

	// Special case: boolean operators don't use type classes.
	// As of v0.11.3, && and || are desugared to core.If at elaboration time
	// (see internal/elaborate/expr_simple.go normalizeShortCircuit), so this
	// branch should be unreachable. DEBUG_STRICT=1 panics to catch regressions;
	// otherwise we fall back to eager evaluation for backward compatibility.
	if op == "&&" || op == "||" {
		if os.Getenv("DEBUG_STRICT") == "1" {
			panic(fmt.Sprintf("applyBinOp reached %q branch — short-circuit desugar in elaborate should have emitted core.If. Regression?", op))
		}
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

	// M-CONTRACT-OPLOWERING-FIX: Comparison/equality ops may survive to the evaluator
	// from contract expressions where dictionary elaboration or OpLowering couldn't
	// fully resolve types. Handle them with runtime type dispatch.
	if isComparisonOp(op) {
		if lInt, lOk := left.(*IntValue); lOk {
			if rInt, rOk := right.(*IntValue); rOk {
				switch op {
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
		if lFloat, lOk := left.(*FloatValue); lOk {
			if rFloat, rOk := right.(*FloatValue); rOk {
				switch op {
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
		if lStr, lOk := left.(*StringValue); lOk {
			if rStr, rOk := right.(*StringValue); rOk {
				switch op {
				case "==":
					return &BoolValue{Value: lStr.Value == rStr.Value}, nil
				case "!=":
					return &BoolValue{Value: lStr.Value != rStr.Value}, nil
				case "<":
					return &BoolValue{Value: lStr.Value < rStr.Value}, nil
				case ">":
					return &BoolValue{Value: lStr.Value > rStr.Value}, nil
				case "<=":
					return &BoolValue{Value: lStr.Value <= rStr.Value}, nil
				case ">=":
					return &BoolValue{Value: lStr.Value >= rStr.Value}, nil
				}
			}
		}
		return nil, fmt.Errorf("comparison '%s' on unsupported types: %T and %T", op, left, right)
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
				case "&":
					return &IntValue{Value: lInt.Value & rInt.Value}, nil
				case "^":
					return &IntValue{Value: lInt.Value ^ rInt.Value}, nil
				case "<<":
					if rInt.Value < 0 {
						return nil, fmt.Errorf("negative shift amount: %d", rInt.Value)
					}
					return &IntValue{Value: lInt.Value << uint(rInt.Value)}, nil
				case ">>":
					if rInt.Value < 0 {
						return nil, fmt.Errorf("negative shift amount: %d", rInt.Value)
					}
					return &IntValue{Value: lInt.Value >> uint(rInt.Value)}, nil
				}
			}
		}

		// Try String operations (comparison/equality only)
		if lStr, lOk := left.(*StringValue); lOk {
			if rStr, rOk := right.(*StringValue); rOk {
				switch op {
				case "==":
					return &BoolValue{Value: lStr.Value == rStr.Value}, nil
				case "!=":
					return &BoolValue{Value: lStr.Value != rStr.Value}, nil
				case "<":
					return &BoolValue{Value: lStr.Value < rStr.Value}, nil
				case ">":
					return &BoolValue{Value: lStr.Value > rStr.Value}, nil
				case "<=":
					return &BoolValue{Value: lStr.Value <= rStr.Value}, nil
				case ">=":
					return &BoolValue{Value: lStr.Value >= rStr.Value}, nil
				case "++":
					return &StringValue{Value: lStr.Value + rStr.Value}, nil
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

// isComparisonOp returns true for comparison and equality operators
func isComparisonOp(op string) bool {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	}
	return false
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

	case "~":
		if v, ok := operand.(*IntValue); ok {
			return &IntValue{Value: ^v.Value}, nil
		}
		return nil, fmt.Errorf("'~' requires int operand, got %T", operand)
	}

	return nil, fmt.Errorf("cannot apply unary operator %s to %T", op, operand)
}

// extractFuncName extracts a human-readable function name from a Core expression.
// applyFunction applies args to a FunctionValue, supporting auto-currying.
// Used by the auto-curry path when intermediate results need further application.
func (e *CoreEvaluator) applyFunction(fn *FunctionValue, args []Value) (Value, error) {
	// M-DX-XPKG-RESOLVE: fallback resolver for function's defining module
	// M-PERF6-PHASE4 M2a: skip re-wrap if chain already covers fn.Resolver
	var oldResolver GlobalResolver
	if fn.Resolver != nil && !resolverCovers(e.resolver, fn.Resolver) {
		oldResolver = e.resolver
		e.resolver = &FallbackResolver{
			Primary:   e.resolver,
			Secondary: fn.Resolver,
		}
	}
	defer func() {
		if oldResolver != nil {
			e.resolver = oldResolver
		}
	}()

	if len(args) > len(fn.Params) {
		// Still more args than params — recurse
		firstArgs := args[:len(fn.Params)]
		restArgs := args[len(fn.Params):]

		newEnv := fn.Env.NewChildEnvironment()
		for i, param := range fn.Params {
			newEnv.Set(param, firstArgs[i])
		}

		oldEnv := e.env
		e.env = newEnv
		var intermediate Value
		var err error
		if coreBody, ok := fn.Body.(core.CoreExpr); ok {
			intermediate, err = e.evalCore(coreBody)
		} else {
			e.env = oldEnv
			return nil, fmt.Errorf("function body is not Core AST")
		}
		e.env = oldEnv
		if err != nil {
			return nil, err
		}

		if innerFn, ok := intermediate.(*FunctionValue); ok {
			return e.applyFunction(innerFn, restArgs)
		}
		return nil, fmt.Errorf("auto-curry: intermediate result is not a function")
	}

	if len(args) < len(fn.Params) {
		return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
	}

	// Exact match — evaluate directly
	newEnv := fn.Env.NewChildEnvironment()
	for i, param := range fn.Params {
		newEnv.Set(param, args[i])
	}

	oldEnv := e.env
	e.env = newEnv
	var result Value
	var err error
	if coreBody, ok := fn.Body.(core.CoreExpr); ok {
		result, err = e.evalCore(coreBody)
	} else {
		e.env = oldEnv
		return nil, fmt.Errorf("function body is not Core AST")
	}
	e.env = oldEnv
	return result, err
}

// M-TRACE-EXPORT: Used to label function enter/exit trace events.
func extractFuncName(expr core.CoreExpr) string {
	switch e := expr.(type) {
	case *core.Var:
		return e.Name
	case *core.VarGlobal:
		return e.Ref.Module + "." + e.Ref.Name
	default:
		return "<lambda>"
	}
}

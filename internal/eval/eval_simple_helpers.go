package eval

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// toTextValue converts a value to text: strings as-is, anything else as show
// renders it (show never quotes strings).
func toTextValue(v Value) string {
	if s, ok := v.(*StringValue); ok {
		return s.Value
	}
	return Show(v)
}

// evalModule evaluates a module
func (e *SimpleEvaluator) evalModule(module *ast.Module) (Value, error) {
	var result Value = &UnitValue{}

	for _, decl := range module.Decls {
		val, err := e.evalNode(decl)
		if err != nil {
			return nil, err
		}
		if val != nil {
			result = val
		}
	}

	return result, nil
}

// applyFunctionAST applies args to a FunctionValue using the AST evaluator, supporting auto-currying.
func (e *SimpleEvaluator) applyFunctionAST(fn *FunctionValue, args []Value) (Value, error) {
	if len(args) > len(fn.Params) {
		firstArgs := args[:len(fn.Params)]
		restArgs := args[len(fn.Params):]

		fnEnv := fn.Env.NewChildEnvironment()
		for i, param := range fn.Params {
			fnEnv.Set(param, firstArgs[i])
		}

		oldEnv := e.env
		e.env = fnEnv
		var intermediate Value
		var err error
		if body, ok := fn.Body.(ast.Expr); ok {
			intermediate, err = e.evalExpr(body)
		} else {
			err = fmt.Errorf("function body is not an ast.Expr")
		}
		e.env = oldEnv
		if err != nil {
			return nil, err
		}

		if innerFn, ok := intermediate.(*FunctionValue); ok {
			return e.applyFunctionAST(innerFn, restArgs)
		}
		return nil, fmt.Errorf("auto-curry: intermediate result is not a function")
	}

	if len(args) < len(fn.Params) {
		return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
	}

	fnEnv := fn.Env.NewChildEnvironment()
	for i, param := range fn.Params {
		fnEnv.Set(param, args[i])
	}

	oldEnv := e.env
	e.env = fnEnv
	var result Value
	var err error
	if body, ok := fn.Body.(ast.Expr); ok {
		result, err = e.evalExpr(body)
	} else {
		err = fmt.Errorf("function body is not an ast.Expr")
	}
	e.env = oldEnv
	return result, err
}

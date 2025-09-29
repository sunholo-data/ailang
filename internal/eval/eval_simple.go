package eval

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
)

// SimpleEvaluator for basic testing
type SimpleEvaluator struct {
	env *Environment
}

// NewSimple creates a simple evaluator
func NewSimple() *SimpleEvaluator {
	env := NewEnvironment()

	// Register print builtin
	env.Set("print", &BuiltinFunction{
		Name: "print",
		Fn: func(args []Value) (Value, error) {
			for _, arg := range args {
				fmt.Print(arg.String())
			}
			fmt.Println()
			return &UnitValue{}, nil
		},
	})

	// Register show builtin - converts any value to a string
	env.Set("show", &BuiltinFunction{
		Name: "show",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("show expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: showValue(args[0], 0)}, nil
		},
	})

	// Register toText builtin - unquoted version for pretty printing
	env.Set("toText", &BuiltinFunction{
		Name: "toText",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("toText expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: toTextValue(args[0])}, nil
		},
	})

	return &SimpleEvaluator{env: env}
}

// EvalProgram evaluates a program
func (e *SimpleEvaluator) EvalProgram(program *ast.Program) (Value, error) {
	// Handle new File structure
	if program.File != nil {
		return e.evalFile(program.File)
	}
	// Legacy: handle Module structure
	if program.Module != nil {
		return e.evalModule(program.Module)
	}
	return &UnitValue{}, nil
}

// evalFile evaluates a File
func (e *SimpleEvaluator) evalFile(file *ast.File) (Value, error) {
	// Process declarations
	var lastVal Value = &UnitValue{}

	for _, decl := range file.Decls {
		val, err := e.evalNode(decl)
		if err != nil {
			return nil, err
		}
		lastVal = val
	}

	// If there's a main function, call it
	if mainVal, ok := e.env.Get("main"); ok {
		if fn, ok := mainVal.(*FunctionValue); ok {
			// Call main with no arguments
			// Create new environment for function body
			fnEnv := fn.Env.NewChildEnvironment()
			// No parameters to bind for main

			// Evaluate function body
			oldEnv := e.env
			e.env = fnEnv

			// Body should be an ast.Expr
			if body, ok := fn.Body.(ast.Expr); ok {
				result, err := e.evalExpr(body)
				e.env = oldEnv
				return result, err
			}
			e.env = oldEnv
			return nil, fmt.Errorf("function body is not an expression")
		}
	}

	return lastVal, nil
}

// evalNode evaluates any AST node
func (e *SimpleEvaluator) evalNode(node ast.Node) (Value, error) {
	switch n := node.(type) {
	case ast.Expr:
		return e.evalExpr(n)
	case *ast.Module:
		return e.evalModule(n)
	case *ast.FuncDecl:
		// Store function in environment
		params := make([]string, len(n.Params))
		for i, p := range n.Params {
			params[i] = p.Name
		}
		fn := &FunctionValue{
			Params: params,
			Body:   n.Body,
			Env:    e.env,
		}
		e.env.Set(n.Name, fn)
		return fn, nil
	default:
		// Try to evaluate as expression
		if expr, ok := node.(ast.Expr); ok {
			return e.evalExpr(expr)
		}
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// evalExpr evaluates an expression
func (e *SimpleEvaluator) evalExpr(expr ast.Expr) (Value, error) {
	switch ex := expr.(type) {
	case *ast.Literal:
		return e.evalLiteral(ex)

	case *ast.Identifier:
		val, ok := e.env.Get(ex.Name)
		if !ok {
			return nil, fmt.Errorf("undefined identifier: %s", ex.Name)
		}
		return val, nil

	case *ast.BinaryOp:
		left, err := e.evalExpr(ex.Left)
		if err != nil {
			return nil, err
		}
		right, err := e.evalExpr(ex.Right)
		if err != nil {
			return nil, err
		}
		return e.evalBinOp(ex.Op, left, right)

	case *ast.UnaryOp:
		operand, err := e.evalExpr(ex.Expr)
		if err != nil {
			return nil, err
		}
		return e.evalUnOp(ex.Op, operand)

	case *ast.FuncCall:
		return e.evalCall(ex)

	case *ast.Let:
		// Evaluate value
		val, err := e.evalExpr(ex.Value)
		if err != nil {
			return nil, err
		}

		// Create new environment with binding
		newEnv := e.env.NewChildEnvironment()
		newEnv.Set(ex.Name, val)

		// Evaluate body in new environment
		oldEnv := e.env
		e.env = newEnv
		result, err := e.evalExpr(ex.Body)
		e.env = oldEnv

		return result, err

	case *ast.If:
		cond, err := e.evalExpr(ex.Condition)
		if err != nil {
			return nil, err
		}

		boolVal, ok := cond.(*BoolValue)
		if !ok {
			// Try to convert to bool
			if intVal, ok := cond.(*IntValue); ok {
				boolVal = &BoolValue{Value: intVal.Value != 0}
			} else {
				return nil, fmt.Errorf("if condition must be boolean, got %s", cond.Type())
			}
		}

		if boolVal.Value {
			return e.evalExpr(ex.Then)
		} else if ex.Else != nil {
			return e.evalExpr(ex.Else)
		}
		return &UnitValue{}, nil

	case *ast.Lambda:
		params := make([]string, len(ex.Params))
		for i, p := range ex.Params {
			params[i] = p.Name
		}
		return &FunctionValue{
			Params: params,
			Body:   ex.Body,
			Env:    e.env,
		}, nil

	case *ast.List:
		elements := make([]Value, len(ex.Elements))
		for i, elem := range ex.Elements {
			val, err := e.evalExpr(elem)
			if err != nil {
				return nil, err
			}
			elements[i] = val
		}
		return &ListValue{Elements: elements}, nil

	case *ast.Record:
		fields := make(map[string]Value)
		for _, field := range ex.Fields {
			val, err := e.evalExpr(field.Value)
			if err != nil {
				return nil, err
			}
			fields[field.Name] = val
		}
		return &RecordValue{Fields: fields}, nil

	case *ast.RecordAccess:
		record, err := e.evalExpr(ex.Record)
		if err != nil {
			return nil, err
		}

		recordVal, ok := record.(*RecordValue)
		if !ok {
			return nil, fmt.Errorf("cannot access field %s on non-record value: %T", ex.Field, record)
		}

		value, exists := recordVal.Fields[ex.Field]
		if !exists {
			return nil, fmt.Errorf("field '%s' does not exist on record", ex.Field)
		}

		return value, nil

	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// evalLiteral evaluates a literal
func (e *SimpleEvaluator) evalLiteral(lit *ast.Literal) (Value, error) {
	switch lit.Kind {
	case ast.IntLit:
		switch v := lit.Value.(type) {
		case int64:
			return &IntValue{Value: int(v)}, nil
		case int:
			return &IntValue{Value: v}, nil
		default:
			return nil, fmt.Errorf("invalid int literal: %T", lit.Value)
		}
	case ast.FloatLit:
		return &FloatValue{Value: lit.Value.(float64)}, nil
	case ast.StringLit:
		return &StringValue{Value: lit.Value.(string)}, nil
	case ast.BoolLit:
		return &BoolValue{Value: lit.Value.(bool)}, nil
	case ast.UnitLit:
		return &UnitValue{}, nil
	default:
		return nil, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}
}

// evalCall evaluates a function call
func (e *SimpleEvaluator) evalCall(call *ast.FuncCall) (Value, error) {
	fn, err := e.evalExpr(call.Func)
	if err != nil {
		return nil, err
	}

	// Evaluate arguments
	args := make([]Value, len(call.Args))
	for i, arg := range call.Args {
		val, err := e.evalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	switch f := fn.(type) {
	case *FunctionValue:
		if len(args) != len(f.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(f.Params), len(args))
		}

		// Create new environment for function body
		fnEnv := f.Env.NewChildEnvironment()
		for i, param := range f.Params {
			fnEnv.Set(param, args[i])
		}

		// Evaluate function body
		oldEnv := e.env
		e.env = fnEnv
		// f.Body is interface{} that could be ast.Expr
		var result Value
		var err error
		if body, ok := f.Body.(ast.Expr); ok {
			result, err = e.evalExpr(body)
		} else {
			err = fmt.Errorf("function body is not an ast.Expr")
		}
		e.env = oldEnv

		return result, err

	case *BuiltinFunction:
		return f.Fn(args)

	default:
		return nil, fmt.Errorf("cannot call non-function value: %s", fn.Type())
	}
}

// evalBinOp evaluates a binary operation
func (e *SimpleEvaluator) evalBinOp(op string, left, right Value) (Value, error) {
	switch op {
	case "+":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &IntValue{Value: l.Value + r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &FloatValue{Value: l.Value + r.Value}, nil
			}
		}
		return nil, fmt.Errorf("'+' requires numeric types (use '++' for string concatenation)")

	case "++":
		lStr, lOk := left.(*StringValue)
		rStr, rOk := right.(*StringValue)
		if !lOk || !rOk {
			return nil, fmt.Errorf("'++' requires string operands")
		}
		return &StringValue{Value: lStr.Value + rStr.Value}, nil

	case "-":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &IntValue{Value: l.Value - r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &FloatValue{Value: l.Value - r.Value}, nil
			}
		}
		return nil, fmt.Errorf("- expects numeric types")

	case "*":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &IntValue{Value: l.Value * r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &FloatValue{Value: l.Value * r.Value}, nil
			}
		}
		return nil, fmt.Errorf("* expects numeric types")

	case "/":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				if r.Value == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				return &IntValue{Value: l.Value / r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				if r.Value == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				return &FloatValue{Value: l.Value / r.Value}, nil
			}
		}
		return nil, fmt.Errorf("/ expects numeric types")

	case "==":
		return &BoolValue{Value: e.valuesEqual(left, right)}, nil

	case "!=":
		return &BoolValue{Value: !e.valuesEqual(left, right)}, nil

	case "<":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &BoolValue{Value: l.Value < r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &BoolValue{Value: l.Value < r.Value}, nil
			}
		}
		return nil, fmt.Errorf("< expects numeric types")

	case ">":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &BoolValue{Value: l.Value > r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &BoolValue{Value: l.Value > r.Value}, nil
			}
		}
		return nil, fmt.Errorf("> expects numeric types")

	case "<=":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &BoolValue{Value: l.Value <= r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &BoolValue{Value: l.Value <= r.Value}, nil
			}
		}
		return nil, fmt.Errorf("<= expects numeric types")

	case ">=":
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				return &BoolValue{Value: l.Value >= r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				return &BoolValue{Value: l.Value >= r.Value}, nil
			}
		}
		return nil, fmt.Errorf(">= expects numeric types")

	case "&&":
		lBool, ok := left.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("&& expects boolean operands")
		}
		if !lBool.Value {
			return &BoolValue{Value: false}, nil
		}
		rBool, ok := right.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("&& expects boolean operands")
		}
		return &BoolValue{Value: rBool.Value}, nil

	case "||":
		lBool, ok := left.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("|| expects boolean operands")
		}
		if lBool.Value {
			return &BoolValue{Value: true}, nil
		}
		rBool, ok := right.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("|| expects boolean operands")
		}
		return &BoolValue{Value: rBool.Value}, nil

	case "%":
		// Modulo operator
		switch l := left.(type) {
		case *IntValue:
			if r, ok := right.(*IntValue); ok {
				if r.Value == 0 {
					return nil, fmt.Errorf("modulo by zero")
				}
				return &IntValue{Value: l.Value % r.Value}, nil
			}
		case *FloatValue:
			if r, ok := right.(*FloatValue); ok {
				if r.Value == 0 {
					return nil, fmt.Errorf("modulo by zero")
				}
				// Go's math.Mod for floating point
				return &FloatValue{Value: math.Mod(l.Value, r.Value)}, nil
			}
		}
		return nil, fmt.Errorf("%% expects numeric operands")

	default:
		return nil, fmt.Errorf("unknown operator: %s", op)
	}
}

// evalUnOp evaluates a unary operation
func (e *SimpleEvaluator) evalUnOp(op string, operand Value) (Value, error) {
	switch op {
	case "-":
		switch v := operand.(type) {
		case *IntValue:
			return &IntValue{Value: -v.Value}, nil
		case *FloatValue:
			return &FloatValue{Value: -v.Value}, nil
		default:
			return nil, fmt.Errorf("- expects numeric operand")
		}
	case "!":
		switch v := operand.(type) {
		case *BoolValue:
			return &BoolValue{Value: !v.Value}, nil
		default:
			return nil, fmt.Errorf("! expects boolean operand")
		}
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

// valuesEqual checks if two values are equal
func (e *SimpleEvaluator) valuesEqual(left, right Value) bool {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return l.Value == r.Value
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return l.Value == r.Value
		}
	case *StringValue:
		if r, ok := right.(*StringValue); ok {
			return l.Value == r.Value
		}
	case *BoolValue:
		if r, ok := right.(*BoolValue); ok {
			return l.Value == r.Value
		}
	case *UnitValue:
		_, ok := right.(*UnitValue)
		return ok
	}
	return false
}

// Constants for show function
const (
	maxDepth      = 3
	maxWidth      = 80
	elisionPrefix = 20
	elisionSuffix = 20
)

// showValue converts a value to its canonical string representation
// with proper quoting, escaping, and deterministic output
func showValue(v Value, depth int) string {
	if depth > maxDepth {
		return "..."
	}

	switch val := v.(type) {
	case *IntValue:
		return strconv.Itoa(val.Value)

	case *FloatValue:
		// Handle special cases
		if math.IsNaN(val.Value) {
			return "NaN"
		}
		if math.IsInf(val.Value, 1) {
			return "Inf"
		}
		if math.IsInf(val.Value, -1) {
			return "-Inf"
		}
		// Handle negative zero explicitly
		if val.Value == 0 && math.Signbit(val.Value) {
			return "-0.0"
		}
		// Use %g for cleaner output, but ensure precision
		return fmt.Sprintf("%g", val.Value)

	case *StringValue:
		// Quote and escape the string using JSON rules
		return strconv.Quote(val.Value)

	case *BoolValue:
		if val.Value {
			return "true"
		}
		return "false"

	case *UnitValue:
		return "()"

	case *ListValue:
		if len(val.Elements) == 0 {
			return "[]"
		}
		var parts []string
		for _, elem := range val.Elements {
			parts = append(parts, showValue(elem, depth+1))
		}
		result := "[" + strings.Join(parts, ", ") + "]"
		return truncateIfNeeded(result)

	case *RecordValue:
		if len(val.Fields) == 0 {
			return "{}"
		}
		// Sort keys for deterministic output
		keys := make([]string, 0, len(val.Fields))
		for k := range val.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Bytewise sort

		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, showValue(val.Fields[k], depth+1)))
		}
		result := "{" + strings.Join(parts, ", ") + "}"
		return truncateIfNeeded(result)

	case *FunctionValue:
		return "<function>"

	case *BuiltinFunction:
		return fmt.Sprintf("<builtin: %s>", val.Name)

	case *ErrorValue:
		return fmt.Sprintf("Error: %s", val.Message)

	default:
		return "<unknown>"
	}
}

// toTextValue converts a value to string without quotes (for pretty printing)
func toTextValue(v Value) string {
	switch val := v.(type) {
	case *StringValue:
		// Return string without quotes
		return val.Value
	default:
		// For all other types, use show but strip quotes if string
		result := showValue(v, 0)
		// If it's a quoted string from show, unquote it
		if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
			unquoted, err := strconv.Unquote(result)
			if err == nil {
				return unquoted
			}
		}
		return result
	}
}

// truncateIfNeeded elides the middle of long strings to keep under maxWidth
func truncateIfNeeded(s string) string {
	if len(s) <= maxWidth {
		return s
	}

	// Preserve prefix and suffix, elide middle
	if len(s) > elisionPrefix+elisionSuffix+3 {
		return s[:elisionPrefix] + "..." + s[len(s)-elisionSuffix:]
	}
	return s
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

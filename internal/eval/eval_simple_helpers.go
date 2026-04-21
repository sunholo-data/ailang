package eval

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

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
		// Use %g for cleaner output, but ensure at least one decimal point
		s := fmt.Sprintf("%g", val.Value)
		// Ensure at least one decimal point for floats (5 -> 5.0)
		if !strings.Contains(s, ".") && !strings.Contains(s, "e") && !strings.Contains(s, "E") {
			s = s + ".0"
		}
		return s

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

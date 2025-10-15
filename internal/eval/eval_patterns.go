package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/dtree"
	"github.com/sunholo/ailang/internal/types"
)

// evalCoreMatch evaluates pattern matching
func (e *CoreEvaluator) evalCoreMatch(match *core.Match) (Value, error) {
	// Evaluate scrutinee
	scrutineeVal, err := e.evalCore(match.Scrutinee)
	if err != nil {
		return nil, err
	}

	// Decision tree optimization: compile to tree if beneficial
	// Note: Decision tree compilation is available but disabled by default
	// This is a runtime optimization that doesn't change semantics
	useDecisionTree := false // Can be enabled via flag in future
	if useDecisionTree {
		compiler := dtree.NewDecisionTreeCompiler(match.Arms)
		tree := compiler.Compile()
		return e.evalDecisionTree(scrutineeVal, tree, match.Arms)
	}

	// Linear evaluation (current default implementation)
	// Try each arm
	for _, arm := range match.Arms {
		bindings, matched := matchPattern(arm.Pattern, scrutineeVal)
		if !matched {
			continue
		}

		// Check guard if present
		if arm.Guard != nil {
			// Push bindings for guard evaluation
			newEnv := e.env.NewChildEnvironment()
			for name, val := range bindings {
				newEnv.Set(name, val)
			}

			oldEnv := e.env
			e.env = newEnv
			guardVal, err := e.evalCore(arm.Guard)
			e.env = oldEnv

			if err != nil {
				return nil, fmt.Errorf("guard evaluation failed: %w", err)
			}

			// Guard must evaluate to Bool
			boolVal, ok := guardVal.(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("guard must evaluate to Bool, got %T", guardVal)
			}

			// If guard is false, try next arm
			if !boolVal.Value {
				continue
			}
		}

		// Pattern matched and guard passed - evaluate body with bindings
		newEnv := e.env.NewChildEnvironment()
		for name, val := range bindings {
			newEnv.Set(name, val)
		}

		oldEnv := e.env
		e.env = newEnv
		result, err := e.evalCore(arm.Body)
		e.env = oldEnv

		return result, err
	}

	return nil, fmt.Errorf("no pattern matched in match expression")
}

// matchPattern attempts to match a pattern against a value
func matchPattern(pattern core.CorePattern, value Value) (map[string]Value, bool) {
	bindings := make(map[string]Value)

	switch p := pattern.(type) {
	case *core.VarPattern:
		// Variable pattern always matches and binds
		bindings[p.Name] = value
		return bindings, true

	case *core.LitPattern:
		// Literal pattern matches if values are equal
		switch v := value.(type) {
		case *IntValue:
			if i, ok := p.Value.(int); ok && i == v.Value {
				return bindings, true
			}
		case *FloatValue:
			if f, ok := p.Value.(float64); ok && f == v.Value {
				return bindings, true
			}
		case *StringValue:
			if s, ok := p.Value.(string); ok && s == v.Value {
				return bindings, true
			}
		case *BoolValue:
			if b, ok := p.Value.(bool); ok && b == v.Value {
				return bindings, true
			}
		}
		return nil, false

	case *core.WildcardPattern:
		// Wildcard always matches without binding
		return bindings, true

	case *core.TuplePattern:
		// Tuple pattern - value must be a tuple with matching arity
		tupleVal, ok := value.(*TupleValue)
		if !ok {
			return nil, false
		}

		if len(p.Elements) != len(tupleVal.Elements) {
			return nil, false
		}

		// Match each element pattern
		for i, elemPattern := range p.Elements {
			elemBindings, ok := matchPattern(elemPattern, tupleVal.Elements[i])
			if !ok {
				return nil, false
			}
			// Merge bindings
			for k, v := range elemBindings {
				bindings[k] = v
			}
		}
		return bindings, true

	case *core.ConstructorPattern:
		// Constructor pattern - value must be a TaggedValue with matching constructor
		tagged, ok := value.(*TaggedValue)
		if !ok {
			return nil, false
		}

		// Check if constructor name matches
		if tagged.CtorName != p.Name {
			return nil, false
		}

		// Check arity
		if len(p.Args) != len(tagged.Fields) {
			return nil, false
		}

		// Match field patterns recursively
		for i, argPattern := range p.Args {
			argBindings, ok := matchPattern(argPattern, tagged.Fields[i])
			if !ok {
				return nil, false
			}
			// Merge bindings
			for k, v := range argBindings {
				bindings[k] = v
			}
		}
		return bindings, true

	default:
		// Other patterns not yet implemented
		return nil, false
	}
}

// evalDictRef evaluates a dictionary reference
func (e *CoreEvaluator) evalDictRef(ref *core.DictRef) (Value, error) {
	// Create a dictionary value that contains the methods
	// The dictionary is a record with method implementations

	// Look up all methods for this class/type combination
	methods := make(map[string]Value)

	// Create type for normalized key generation
	typ := &types.TCon{Name: ref.TypeName}

	// Common methods for each class
	var methodNames []string
	switch ref.ClassName {
	case "Num":
		methodNames = []string{"add", "sub", "mul", "div", "neg", "abs", "fromInt"}
	case "Fractional":
		methodNames = []string{"add", "sub", "mul", "div", "neg", "abs", "fromInt", "divide", "recip", "fromRational"}
	case "Eq":
		methodNames = []string{"eq", "neq"}
	case "Ord":
		methodNames = []string{"lt", "lte", "gt", "gte", "min", "max"}
	default:
		return nil, fmt.Errorf("unknown type class: %s", ref.ClassName)
	}

	// Collect all methods
	for _, method := range methodNames {
		key := types.MakeDictionaryKey("prelude", ref.ClassName, typ, method)
		entry, ok := e.registry.Lookup(key)
		if !ok {
			return nil, fmt.Errorf("missing dictionary method: %s", key)
		}

		// Check if the implementation is already a BuiltinFunction
		if builtin, ok := entry.Impl.(*BuiltinFunction); ok {
			methods[method] = builtin
		} else {
			// Wrap the implementation as a builtin function
			methods[method] = &BuiltinFunction{
				Name: method,
				Fn:   wrapDictionaryMethod(entry.Impl),
			}
		}
	}

	// Return dictionary as a record
	return &RecordValue{Fields: methods}, nil
}

// evalDictAbs evaluates dictionary abstraction
func (e *CoreEvaluator) evalDictAbs(abs *core.DictAbs) (Value, error) {
	// Dictionary abstraction introduces dictionary parameters
	// We need to evaluate the body with dictionaries in scope

	// For now, we'll just evaluate the body
	// In a full implementation, this would handle polymorphic dictionary passing
	return e.evalCore(abs.Body)
}

// evalDictApp evaluates dictionary application
func (e *CoreEvaluator) evalDictApp(app *core.DictApp) (Value, error) {
	// Evaluate the dictionary
	dictVal, err := e.evalCore(app.Dict)
	if err != nil {
		return nil, err
	}

	// Dictionary should be a record with methods
	dict, ok := dictVal.(*RecordValue)
	if !ok {
		return nil, fmt.Errorf("dictionary must be a record, got %T", dictVal)
	}

	// Look up the method
	// fmt.Printf("DEBUG: DictApp looking for method '%s' in dictionary with fields: %v\n", app.Method, getFieldNames(dict.Fields))
	methodVal, ok := dict.Fields[app.Method]
	if !ok {
		return nil, fmt.Errorf("dictionary missing method: %s", app.Method)
	}

	// Evaluate arguments
	var args []Value
	for _, argExpr := range app.Args {
		argVal, err := e.evalCore(argExpr)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// Apply the method with proper type checking
	switch method := methodVal.(type) {
	case *BuiltinFunction:
		// Proper BuiltinFunction - use its Fn
		return method.Fn(args)
	default:
		// Raw function that slipped through - this should not happen with proper registration
		return nil, fmt.Errorf("unsupported dictionary method type: %T", methodVal)
	}
}

// wrapDictionaryMethod wraps a Go function as a Value function
func wrapDictionaryMethod(impl interface{}) func([]Value) (Value, error) {
	// If it's already a BuiltinFunction, extract its Fn
	if builtin, ok := impl.(*BuiltinFunction); ok {
		return builtin.Fn
	}

	return func(args []Value) (Value, error) {
		// This is a simplified wrapper - a full implementation would handle
		// all type conversions properly

		switch fn := impl.(type) {
		case func(int64, int64) int64:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int64(x.Value), int64(y.Value))
			return &IntValue{Value: int(result)}, nil

		case func(int, int) int:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(x.Value, y.Value)
			return &IntValue{Value: result}, nil

		case func(float64, float64) float64:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*FloatValue)
			y, ok2 := args[1].(*FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			result := fn(x.Value, y.Value)
			return &FloatValue{Value: result}, nil

		case func(int64, int64) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int64(x.Value), int64(y.Value))
			return &BoolValue{Value: result}, nil

		case func(int, int) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int(x.Value), int(y.Value))
			return &BoolValue{Value: result}, nil

		case func(float64, float64) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*FloatValue)
			y, ok2 := args[1].(*FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			result := fn(x.Value, y.Value)
			return &BoolValue{Value: result}, nil

		case func(int) int:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("expected int argument")
			}
			result := fn(x.Value)
			return &IntValue{Value: result}, nil

		case func(float64) float64:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("expected float argument")
			}
			result := fn(x.Value)
			return &FloatValue{Value: result}, nil

		case func(bool) bool:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("expected bool argument")
			}
			result := fn(x.Value)
			return &BoolValue{Value: result}, nil

		case func(string, string) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*StringValue)
			y, ok2 := args[1].(*StringValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected string arguments")
			}
			result := fn(x.Value, y.Value)
			return &BoolValue{Value: result}, nil

		case func(string, string) string:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*StringValue)
			y, ok2 := args[1].(*StringValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected string arguments")
			}
			result := fn(x.Value, y.Value)
			return &StringValue{Value: result}, nil

		default:
			return nil, fmt.Errorf("unsupported dictionary method type: %T", impl)
		}
	}
}

// ADT Runtime Helpers

// isTag checks if a value is a TaggedValue with the given type and constructor names
func isTag(v Value, typeName, ctorName string) bool {
	tagged, ok := v.(*TaggedValue)
	if !ok {
		return false
	}
	return tagged.TypeName == typeName && tagged.CtorName == ctorName
}

// getField extracts a field from a TaggedValue by index (bounds-checked)
func getField(v Value, index int) (Value, error) {
	tagged, ok := v.(*TaggedValue)
	if !ok {
		return nil, fmt.Errorf("EVA_RT002: getField called on non-tagged value: %s", v.Type())
	}
	if index < 0 || index >= len(tagged.Fields) {
		return nil, fmt.Errorf("EVA_RT002: field index %d out of bounds for constructor %s (has %d fields)",
			index, tagged.CtorName, len(tagged.Fields))
	}
	return tagged.Fields[index], nil
}

// Helper functions

// isLambda checks if a Core expression is a Lambda
func isLambda(expr core.CoreExpr) (*core.Lambda, bool) {
	if lam, ok := expr.(*core.Lambda); ok {
		return lam, true
	}
	return nil, false
}

// buildClosure creates a FunctionValue from a Lambda, capturing the given environment
func (e *CoreEvaluator) buildClosure(lam *core.Lambda, env *Environment) (*FunctionValue, error) {
	return &FunctionValue{
		Params: lam.Params,
		Body:   lam.Body,
		Env:    env,
		Typed:  false,
	}, nil
}

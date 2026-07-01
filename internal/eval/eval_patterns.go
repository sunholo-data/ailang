package eval

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/dtree"
	"github.com/sunholo-data/ailang/internal/types"
)

// debugMatch enables debug output for pattern matching when DEBUG_MATCH=1
var debugMatch = os.Getenv("DEBUG_MATCH") == "1"

// evalCoreMatch evaluates pattern matching
func (e *CoreEvaluator) evalCoreMatch(match *core.Match) (Value, error) {
	if debugMatch {
		fmt.Printf("[MATCH] Evaluating match with %d arms\n", len(match.Arms))
	}

	// Evaluate scrutinee
	scrutineeVal, err := e.evalCore(match.Scrutinee)
	if err != nil {
		return nil, err
	}

	if debugMatch {
		fmt.Printf("[MATCH] Scrutinee value: %v (type: %T)\n", scrutineeVal, scrutineeVal)
	}

	// Decision tree optimization: compile to tree if beneficial
	// DISABLED by default due to guard failure bug: when a guard evaluates to false,
	// the dtree returns an error instead of falling through to the next arm.
	// Also missing: list/record/tuple pattern support, guard backtracking.
	// Enable with AILANG_DTREE=1 for experimentation (no guards in match).
	useDecisionTree := os.Getenv("AILANG_DTREE") == "1"
	if useDecisionTree {
		compiler := dtree.NewDecisionTreeCompiler(match.Arms)
		tree := compiler.Compile()
		return e.evalDecisionTree(scrutineeVal, tree, match.Arms)
	}

	// Linear evaluation (current default implementation)
	// Try each arm
	for i, arm := range match.Arms {
		if debugMatch {
			fmt.Printf("[MATCH] Trying arm %d, pattern type: %T\n", i, arm.Pattern)
		}
		bindings, matched := matchPattern(arm.Pattern, scrutineeVal)
		if debugMatch {
			fmt.Printf("[MATCH] Arm %d matched: %v, bindings: %v\n", i, matched, bindings)
		}
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
		if debugMatch {
			fmt.Printf("[MATCH] LitPattern: pattern value=%v (type %T), scrutinee=%v (type %T)\n",
				p.Value, p.Value, value, value)
		}
		switch v := value.(type) {
		case *IntValue:
			if debugMatch {
				fmt.Printf("[MATCH] IntValue comparison: pattern=%v (type %T), value=%d\n",
					p.Value, p.Value, v.Value)
			}
			// Check both int and int64 types (parser may use either)
			if i, ok := p.Value.(int); ok && i == v.Value {
				return bindings, true
			}
			if i64, ok := p.Value.(int64); ok && int(i64) == v.Value {
				if debugMatch {
					fmt.Printf("[MATCH] int64 match succeeded!\n")
				}
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

	case *core.ListPattern:
		// List pattern - value must be a ListValue
		listVal, ok := value.(*ListValue)
		if !ok {
			return nil, false
		}

		// Case 1: Pattern is [elem1, elem2, ..., elemN] (exact match, no tail)
		if p.Tail == nil {
			// Must have exactly the same number of elements
			if len(p.Elements) != len(listVal.Elements) {
				return nil, false
			}

			// Match each element pattern
			for i, elemPattern := range p.Elements {
				elemBindings, ok := matchPattern(elemPattern, listVal.Elements[i])
				if !ok {
					return nil, false
				}
				// Merge bindings
				for k, v := range elemBindings {
					bindings[k] = v
				}
			}
			return bindings, true
		}

		// Case 2: Pattern is [elem1, elem2, ..., elemN, ...tail]
		// List must have at least len(p.Elements) elements
		if len(listVal.Elements) < len(p.Elements) {
			return nil, false
		}

		// Match the head elements
		for i, elemPattern := range p.Elements {
			elemBindings, ok := matchPattern(elemPattern, listVal.Elements[i])
			if !ok {
				return nil, false
			}
			// Merge bindings
			for k, v := range elemBindings {
				bindings[k] = v
			}
		}

		// Match the tail (remaining elements)
		tailElements := listVal.Elements[len(p.Elements):]
		tailList := &ListValue{Elements: tailElements}
		tailBindings, ok := matchPattern(*p.Tail, tailList)
		if !ok {
			return nil, false
		}
		// Merge tail bindings
		for k, v := range tailBindings {
			bindings[k] = v
		}
		return bindings, true

	case *core.RecordPattern:
		// Record pattern - value must be a RecordValue
		recVal, ok := value.(*RecordValue)
		if !ok {
			return nil, false
		}

		// Match each field pattern
		for fieldName, fieldPattern := range p.Fields {
			fieldValue, ok := recVal.Fields[fieldName]
			if !ok {
				// Field not present in value
				return nil, false
			}

			fieldBindings, ok := matchPattern(fieldPattern, fieldValue)
			if !ok {
				return nil, false
			}
			// Merge bindings
			for k, v := range fieldBindings {
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
			// Determinism fix (M-DX19): the type checker already proved this class
			// instance exists for the type. A missing Eq method here means the
			// derived-instance registration didn't land — a non-deterministic
			// ordering gap that made `deriving (Eq)` flakily fail at module-eval
			// time with "missing dictionary method: prelude::Eq::<T>::eq". Derived
			// Eq is ALWAYS structural, so synthesize it deterministically rather
			// than depending on the registry having been populated in time.
			if ref.ClassName == "Eq" {
				methods[method] = &BuiltinFunction{
					Name: "derived_" + method + "_" + ref.TypeName,
					Fn:   makeADTEqualityFn(ref.TypeName, method == "eq"),
				}
				continue
			}
			return nil, fmt.Errorf("missing dictionary method: %s", key)
		}

		// M-DX19: Check for derived ADT equality marker
		if derived, ok := entry.Impl.(*types.DerivedADTEquality); ok {
			// Synthesize structural equality for TaggedValue
			if method == "eq" {
				methods[method] = &BuiltinFunction{
					Name: "derived_eq_" + derived.TypeName,
					Fn:   makeADTEqualityFn(derived.TypeName, true),
				}
			} else if method == "neq" {
				methods[method] = &BuiltinFunction{
					Name: "derived_neq_" + derived.TypeName,
					Fn:   makeADTEqualityFn(derived.TypeName, false),
				}
			}
			continue
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

// makeADTEqualityFn creates an equality function for ADT types with deriving (Eq).
// M-DX19: Compares TaggedValue instances structurally.
// If isEq is true, returns eq function (true if equal).
// If isEq is false, returns neq function (true if not equal).
func makeADTEqualityFn(typeName string, isEq bool) func([]Value) (Value, error) {
	return func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("derived Eq for %s expects 2 arguments, got %d", typeName, len(args))
		}

		// Both values must be TaggedValue of the same ADT type
		a, okA := args[0].(*TaggedValue)
		b, okB := args[1].(*TaggedValue)

		if !okA || !okB {
			return nil, fmt.Errorf("derived Eq for %s: expected TaggedValue, got %T and %T", typeName, args[0], args[1])
		}

		// Compare structurally
		equal := taggedValuesEqual(a, b)

		if isEq {
			return &BoolValue{Value: equal}, nil
		}
		return &BoolValue{Value: !equal}, nil
	}
}

// taggedValuesEqual compares two TaggedValue instances structurally.
// M-DX19: Returns true if they have the same constructor and all fields are equal.
func taggedValuesEqual(a, b *TaggedValue) bool {
	// Must have same constructor
	if a.CtorName != b.CtorName {
		return false
	}

	// Must have same number of fields
	if len(a.Fields) != len(b.Fields) {
		return false
	}

	// Compare each field recursively
	for i := range a.Fields {
		if !valuesStructurallyEqual(a.Fields[i], b.Fields[i]) {
			return false
		}
	}

	return true
}

// valuesStructurallyEqual compares two values for structural equality.
// M-DX19: Supports primitive types, TaggedValues, lists, records, and tuples.
func valuesStructurallyEqual(a, b Value) bool {
	// Handle nil case
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare by type
	switch av := a.(type) {
	case *IntValue:
		bv, ok := b.(*IntValue)
		return ok && av.Value == bv.Value
	case *FloatValue:
		bv, ok := b.(*FloatValue)
		return ok && av.Value == bv.Value
	case *StringValue:
		bv, ok := b.(*StringValue)
		return ok && av.Value == bv.Value
	case *BoolValue:
		bv, ok := b.(*BoolValue)
		return ok && av.Value == bv.Value
	case *UnitValue:
		_, ok := b.(*UnitValue)
		return ok
	case *TaggedValue:
		bv, ok := b.(*TaggedValue)
		return ok && taggedValuesEqual(av, bv)
	case *ListValue:
		bv, ok := b.(*ListValue)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !valuesStructurallyEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case *TupleValue:
		bv, ok := b.(*TupleValue)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !valuesStructurallyEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case *RecordValue:
		bv, ok := b.(*RecordValue)
		if !ok || len(av.Fields) != len(bv.Fields) {
			return false
		}
		for k, v := range av.Fields {
			bField, exists := bv.Fields[k]
			if !exists || !valuesStructurallyEqual(v, bField) {
				return false
			}
		}
		return true
	default:
		// Unknown types are not equal
		return false
	}
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
	// M-POLY-ARITH: Evaluate arguments first so we can check their actual types.
	// After Num defaulting, DictRef.TypeName may be "Int" even when the actual
	// runtime arguments are Float (e.g., let add = \x. \y. x + y in add(3.14)(2.71)).
	// By evaluating args first, we can correct the dictionary type before lookup.
	var args []Value
	for _, argExpr := range app.Args {
		argVal, err := e.evalCore(argExpr)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// M-POLY-ARITH: Correct DictRef type based on actual argument types.
	// This fixes the case where Num defaulting resolves the lambda to int -> int -> int
	// but the call site provides float arguments.
	dict := app.Dict
	if ref, ok := dict.(*core.DictRef); ok && len(args) > 0 {
		if actualType := valueTypeName(args[0]); actualType != "" && actualType != ref.TypeName {
			dict = &core.DictRef{
				CoreNode:  ref.CoreNode,
				ClassName: ref.ClassName,
				TypeName:  actualType,
			}
		}
	}

	// Evaluate the dictionary (with corrected type if needed)
	dictVal, err := e.evalCore(dict)
	if err != nil {
		return nil, err
	}

	// Dictionary should be a record with methods
	dictRecord, ok := dictVal.(*RecordValue)
	if !ok {
		return nil, fmt.Errorf("dictionary must be a record, got %T", dictVal)
	}

	// Look up the method
	methodVal, ok := dictRecord.Fields[app.Method]
	if !ok {
		return nil, fmt.Errorf("dictionary missing method: %s", app.Method)
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

// valueTypeName returns the AILANG type name for a runtime value.
// Used by M-POLY-ARITH to correct DictRef.TypeName based on actual argument types.
func valueTypeName(v Value) string {
	switch v.(type) {
	case *IntValue:
		return "Int"
	case *FloatValue:
		return "Float"
	case *StringValue:
		return "String"
	case *BoolValue:
		return "Bool"
	default:
		return ""
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

		// M-DX19: Support for bool equality comparison (Eq[Bool])
		case func(bool, bool) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*BoolValue)
			y, ok2 := args[1].(*BoolValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected bool arguments")
			}
			result := fn(x.Value, y.Value)
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
		Params:   lam.Params,
		Body:     lam.Body,
		Env:      env,
		Resolver: e.resolver, // M-DX-XPKG-RESOLVE: capture defining module's resolver
		Typed:    false,
	}, nil
}

package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/typedast"
)

// Helper functions for TypedEvaluator

// valuesEqual checks if two values are equal
func (e *TypedEvaluator) valuesEqual(left, right interface{}) bool {
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
	case int:
		if r, ok := right.(int); ok {
			return l == r
		}
	case float64:
		if r, ok := right.(float64); ok {
			return l == r
		}
	case string:
		if r, ok := right.(string); ok {
			return l == r
		}
	case bool:
		if r, ok := right.(bool); ok {
			return l == r
		}
	}
	return false
}

// recordTrace records a function call trace
func (e *TypedEvaluator) recordTrace(app *typedast.TypedApp, fn Value, args []Value) {
	if e.trace == nil || !e.trace.Enabled {
		return
	}

	// TODO: Extract scheme and effects from typed nodes
	// For now, create a placeholder trace
	var inputs []string
	for _, arg := range args {
		inputs = append(inputs, boundedShow(arg, 3, 10))
	}

	entry := TraceEntry{
		CallSiteID:  app.NodeID,
		FnID:        0,   // TODO: Extract from function
		FnScheme:    nil, // TODO: Extract scheme
		CallEffects: nil, // TODO: Type assertion needed
		Inputs:      inputs,
		Seed:        e.seed,
		VirtualTime: e.virtualTime,
		Timestamp:   e.getTimestamp(),
	}

	e.trace.Entries = append(e.trace.Entries, entry)
}

// getTimestamp returns current timestamp (virtual or real)
func (e *TypedEvaluator) getTimestamp() int64 {
	if e.virtualTime {
		// TODO: Implement virtual time
		return 0
	}
	// TODO: Get real timestamp
	return 0
}

// boundedShow produces bounded string representation
func boundedShow(v Value, maxDepth, maxWidth int) string {
	// TODO: Implement bounded show with depth/width limits
	return showValue(v, 0)
}

// registerBuiltins registers builtin functions
func registerBuiltins(env *Environment) {
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

	// Register show builtin
	env.Set("show", &BuiltinFunction{
		Name: "show",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("show expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: showValue(args[0], 0)}, nil
		},
	})

	// Register toText builtin
	env.Set("toText", &BuiltinFunction{
		Name: "toText",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("toText expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: toTextValue(args[0])}, nil
		},
	})
}

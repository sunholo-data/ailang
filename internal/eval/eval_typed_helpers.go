package eval

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/typedast"
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

// capRequirer is implemented by effect contexts that gate effects on granted
// capabilities. Declared locally rather than imported from internal/effects,
// to avoid the import cycle — same pattern as budgetChargeScoper.
type capRequirer interface {
	RequireCap(name string) error
}

// requireCap enforces a capability for a PRELUDE builtin that performs a real
// effect.
//
// Why this exists: the prelude's println wrote to stdout via fmt directly,
// while `import std/io (println)` routes through _io_println -> effects.Call ->
// RequireCap. Both forms declare the SAME effect (! {IO}) and both pass effect
// checking, but only the imported one was gated — so `--caps` was evadable
// simply by not importing std/io, and a program with NO capabilities at all
// could still perform IO. Found 2026-08-17 via a motoko eval failure where the
// benchmark rewarded the ungated form and failed the imported one.
//
// A nil evaluator, or an effect context that does not implement capRequirer,
// means no capability system is wired for this evaluator (REPL / SimpleEvaluator
// / unit tests) and enforcement is skipped — those paths never granted caps in
// the first place, so failing closed there would break them without closing any
// hole that `ailang run` leaves open.
func requireCap(e *CoreEvaluator, name string) error {
	if e == nil {
		return nil
	}
	r, ok := e.effContext.(capRequirer)
	if !ok {
		return nil
	}
	return r.RequireCap(name)
}

// registerBuiltins registers builtin functions.
//
// e may be nil for evaluators with no effect context; see requireCap.
func registerBuiltins(env *Environment, e *CoreEvaluator) {
	// Register println builtin (prelude provides this without import)
	// Note: print (no newline) requires: import std/io (print)
	env.Set("println", &BuiltinFunction{
		Name: "println",
		Fn: func(args []Value) (Value, error) {
			// Gate on IO exactly as the std/io import path does.
			if err := requireCap(e, "IO"); err != nil {
				return nil, err
			}
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

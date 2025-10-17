package eval

import "fmt"

// CallBuiltin calls a builtin function with the given arguments
//
// DEPRECATED: This function is no longer used for effect-based builtins (IO, FS).
// Effect-based builtins now route through internal/runtime/builtins.go and the
// effect system for capability checking. This function is kept for backward
// compatibility with non-effect builtins and for internal validation.
//
// For effect-based operations, use runtime.ModuleRuntime with EffContext instead.
func CallBuiltin(name string, args []Value) (Value, error) {
	builtin, ok := Builtins[name]
	if !ok {
		return nil, fmt.Errorf("unknown builtin function: %s", name)
	}

	if len(args) != builtin.NumArgs {
		return nil, fmt.Errorf("builtin %s expects %d arguments, got %d",
			name, builtin.NumArgs, len(args))
	}

	// Handle different arities
	switch builtin.NumArgs {
	case 0:
		return callBuiltin0Args(name, builtin)
	case 1:
		return callBuiltin1Arg(name, builtin, args[0])
	case 2:
		return callBuiltin2Args(name, builtin, args[0], args[1])
	case 3:
		return callBuiltin3Args(name, builtin, args[0], args[1], args[2])
	default:
		return nil, fmt.Errorf("unsupported arity %d for builtin %s", builtin.NumArgs, name)
	}
}

// callBuiltin0Args handles zero-argument builtins
func callBuiltin0Args(name string, builtin *BuiltinFunc) (Value, error) {
	switch impl := builtin.Impl.(type) {
	case func() (*StringValue, error):
		return impl()
	case func() (*UnitValue, error):
		return impl()
	default:
		return nil, fmt.Errorf("unsupported 0-arg builtin implementation for %s", name)
	}
}

// callBuiltin1Arg handles single-argument builtins
func callBuiltin1Arg(name string, builtin *BuiltinFunc, arg Value) (Value, error) {
	// Try generic Value -> Value first
	if fn, ok := builtin.Impl.(func(Value) (Value, error)); ok {
		return fn(arg)
	}

	// Try typed versions
	switch impl := builtin.Impl.(type) {
	case func(*IntValue) (*IntValue, error):
		a, ok := arg.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Int argument", name)
		}
		return impl(a)

	case func(*FloatValue) (*FloatValue, error):
		a, ok := arg.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float argument", name)
		}
		return impl(a)

	case func(*IntValue) (*FloatValue, error):
		a, ok := arg.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Int argument", name)
		}
		return impl(a)

	case func(*FloatValue) (*IntValue, error):
		a, ok := arg.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float argument", name)
		}
		return impl(a)

	case func(*BoolValue) (*BoolValue, error):
		a, ok := arg.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Bool argument", name)
		}
		return impl(a)

	case func(*StringValue) (*IntValue, error):
		a, ok := arg.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String argument", name)
		}
		return impl(a)

	case func(*StringValue) (*StringValue, error):
		a, ok := arg.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String argument", name)
		}
		return impl(a)

	case func(*StringValue) (*UnitValue, error):
		a, ok := arg.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String argument", name)
		}
		return impl(a)

	case func(Value) (*StringValue, error):
		// Generic Value -> StringValue (for ADT processing like JSON encoding)
		return impl(arg)

	default:
		return nil, fmt.Errorf("unsupported builtin implementation for %s", name)
	}
}

// callBuiltin2Args handles two-argument builtins
func callBuiltin2Args(name string, builtin *BuiltinFunc, arg0, arg1 Value) (Value, error) {
	// Try generic Value, Value -> Value first
	if fn, ok := builtin.Impl.(func(Value, Value) (Value, error)); ok {
		return fn(arg0, arg1)
	}

	// Try typed versions
	switch impl := builtin.Impl.(type) {
	case func(*IntValue, *IntValue) (*IntValue, error):
		a, ok := arg0.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Int arguments", name)
		}
		b, ok := arg1.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Int arguments", name)
		}
		return impl(a, b)

	case func(*FloatValue, *FloatValue) (*FloatValue, error):
		a, ok := arg0.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float arguments", name)
		}
		b, ok := arg1.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float arguments", name)
		}
		return impl(a, b)

	case func(*StringValue, *StringValue) (*StringValue, error):
		a, ok := arg0.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		b, ok := arg1.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		return impl(a, b)

	case func(*BoolValue, *BoolValue) (*BoolValue, error):
		a, ok := arg0.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Bool arguments", name)
		}
		b, ok := arg1.(*BoolValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Bool arguments", name)
		}
		return impl(a, b)

	case func(*IntValue, *IntValue) (*BoolValue, error):
		a, ok := arg0.(*IntValue)
		if !ok {
			return nil, buildTypeMismatchError(name, "Int", arg0)
		}
		b, ok := arg1.(*IntValue)
		if !ok {
			return nil, buildTypeMismatchError(name, "Int", arg1)
		}
		return impl(a, b)

	case func(*FloatValue, *FloatValue) (*BoolValue, error):
		a, ok := arg0.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float arguments", name)
		}
		b, ok := arg1.(*FloatValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects Float arguments", name)
		}
		return impl(a, b)

	case func(*StringValue, *StringValue) (*BoolValue, error):
		a, ok := arg0.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		b, ok := arg1.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		return impl(a, b)

	case func(*StringValue, *StringValue) (*IntValue, error):
		a, ok := arg0.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		b, ok := arg1.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String arguments", name)
		}
		return impl(a, b)

	default:
		return nil, fmt.Errorf("unsupported builtin implementation for %s", name)
	}
}

// callBuiltin3Args handles three-argument builtins
func callBuiltin3Args(name string, builtin *BuiltinFunc, arg0, arg1, arg2 Value) (Value, error) {
	switch impl := builtin.Impl.(type) {
	case func(*StringValue, *IntValue, *IntValue) (*StringValue, error):
		a, ok := arg0.(*StringValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
		}
		b, ok := arg1.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
		}
		c, ok := arg2.(*IntValue)
		if !ok {
			return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
		}
		return impl(a, b, c)

	default:
		return nil, fmt.Errorf("unsupported 3-arg builtin implementation for %s", name)
	}
}

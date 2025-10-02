package eval

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// BuiltinFunc represents a built-in function
type BuiltinFunc struct {
	Name    string
	Impl    interface{} // The actual Go function
	NumArgs int         // Expected number of arguments
	IsPure  bool        // Whether the function is pure
}

// Builtins is the registry of built-in functions
var Builtins = make(map[string]*BuiltinFunc)

func init() {
	registerArithmeticBuiltins()
	registerComparisonBuiltins()
	registerStringBuiltins()
	registerBooleanBuiltins()
	registerStringPrimitiveBuiltins()
	registerIOBuiltins()
}

// registerArithmeticBuiltins registers integer and float arithmetic operations
func registerArithmeticBuiltins() {
	// Integer operations
	Builtins["add_Int"] = &BuiltinFunc{
		Name:    "add_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["sub_Int"] = &BuiltinFunc{
		Name:    "sub_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value - b.Value}, nil
		},
	}

	Builtins["mul_Int"] = &BuiltinFunc{
		Name:    "mul_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value * b.Value}, nil
		},
	}

	Builtins["div_Int"] = &BuiltinFunc{
		Name:    "div_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			if b.Value == 0 {
				return nil, NewRuntimeError("RT_DIV0", "Division by zero", nil)
			}
			return &IntValue{Value: a.Value / b.Value}, nil
		},
	}

	Builtins["mod_Int"] = &BuiltinFunc{
		Name:    "mod_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			if b.Value == 0 {
				return nil, NewRuntimeError("RT_DIV0", "Modulo by zero", nil)
			}
			return &IntValue{Value: a.Value % b.Value}, nil
		},
	}

	Builtins["neg_Int"] = &BuiltinFunc{
		Name:    "neg_Int",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *IntValue) (*IntValue, error) {
			return &IntValue{Value: -a.Value}, nil
		},
	}

	// Float operations
	Builtins["add_Float"] = &BuiltinFunc{
		Name:    "add_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["sub_Float"] = &BuiltinFunc{
		Name:    "sub_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value - b.Value}, nil
		},
	}

	Builtins["mul_Float"] = &BuiltinFunc{
		Name:    "mul_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value * b.Value}, nil
		},
	}

	Builtins["div_Float"] = &BuiltinFunc{
		Name:    "div_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			if b.Value == 0.0 {
				// IEEE 754 behavior: division by zero produces infinity
				if a.Value >= 0 {
					return &FloatValue{Value: math.Inf(1)}, nil
				} else {
					return &FloatValue{Value: math.Inf(-1)}, nil
				}
			}
			return &FloatValue{Value: a.Value / b.Value}, nil
		},
	}

	Builtins["mod_Float"] = &BuiltinFunc{
		Name:    "mod_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			if b.Value == 0.0 {
				return &FloatValue{Value: math.NaN()}, nil
			}
			return &FloatValue{Value: math.Mod(a.Value, b.Value)}, nil
		},
	}

	Builtins["neg_Float"] = &BuiltinFunc{
		Name:    "neg_Float",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: -a.Value}, nil
		},
	}
}

// registerComparisonBuiltins registers comparison operations
func registerComparisonBuiltins() {
	// Integer comparisons
	Builtins["eq_Int"] = &BuiltinFunc{
		Name:    "eq_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Int"] = &BuiltinFunc{
		Name:    "ne_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_Int"] = &BuiltinFunc{
		Name:    "lt_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_Int"] = &BuiltinFunc{
		Name:    "le_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_Int"] = &BuiltinFunc{
		Name:    "gt_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_Int"] = &BuiltinFunc{
		Name:    "ge_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}

	// Float comparisons with NaN handling
	// NaN comparisons: all comparisons with NaN return false except !=
	Builtins["eq_Float"] = &BuiltinFunc{
		Name:    "eq_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// NaN is not equal to anything, including itself
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Float"] = &BuiltinFunc{
		Name:    "ne_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// NaN is not equal to anything, so != returns true for any NaN
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: true}, nil
			}
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_Float"] = &BuiltinFunc{
		Name:    "lt_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// Any comparison with NaN returns false
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_Float"] = &BuiltinFunc{
		Name:    "le_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_Float"] = &BuiltinFunc{
		Name:    "gt_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_Float"] = &BuiltinFunc{
		Name:    "ge_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}
}

// registerStringBuiltins registers string operations
func registerStringBuiltins() {
	Builtins["concat_String"] = &BuiltinFunc{
		Name:    "concat_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*StringValue, error) {
			return &StringValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["eq_String"] = &BuiltinFunc{
		Name:    "eq_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_String"] = &BuiltinFunc{
		Name:    "ne_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_String"] = &BuiltinFunc{
		Name:    "lt_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_String"] = &BuiltinFunc{
		Name:    "le_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_String"] = &BuiltinFunc{
		Name:    "gt_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_String"] = &BuiltinFunc{
		Name:    "ge_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}
}

// registerBooleanBuiltins registers boolean operations
func registerBooleanBuiltins() {
	Builtins["and_Bool"] = &BuiltinFunc{
		Name:    "and_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value && b.Value}, nil
		},
	}

	Builtins["or_Bool"] = &BuiltinFunc{
		Name:    "or_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value || b.Value}, nil
		},
	}

	Builtins["not_Bool"] = &BuiltinFunc{
		Name:    "not_Bool",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: !a.Value}, nil
		},
	}

	Builtins["eq_Bool"] = &BuiltinFunc{
		Name:    "eq_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Bool"] = &BuiltinFunc{
		Name:    "ne_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}
}

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
		// Zero-argument builtins (e.g., _io_readLine)
		switch impl := builtin.Impl.(type) {
		case func() (*StringValue, error):
			return impl()
		case func() (*UnitValue, error):
			return impl()
		default:
			return nil, fmt.Errorf("unsupported 0-arg builtin implementation for %s", name)
		}

	case 1:
		fn, ok := builtin.Impl.(func(Value) (Value, error))
		if ok {
			return fn(args[0])
		}
		// Try typed versions
		switch impl := builtin.Impl.(type) {
		case func(*IntValue) (*IntValue, error):
			a, ok := args[0].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Int argument", name)
			}
			return impl(a)
		case func(*FloatValue) (*FloatValue, error):
			a, ok := args[0].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Float argument", name)
			}
			return impl(a)
		case func(*BoolValue) (*BoolValue, error):
			a, ok := args[0].(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Bool argument", name)
			}
			return impl(a)
		case func(*StringValue) (*IntValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String argument", name)
			}
			return impl(a)
		case func(*StringValue) (*StringValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String argument", name)
			}
			return impl(a)
		case func(*StringValue) (*UnitValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String argument", name)
			}
			return impl(a)
		default:
			return nil, fmt.Errorf("unsupported builtin implementation for %s", name)
		}

	case 2:
		fn, ok := builtin.Impl.(func(Value, Value) (Value, error))
		if ok {
			return fn(args[0], args[1])
		}
		// Try typed versions
		switch impl := builtin.Impl.(type) {
		case func(*IntValue, *IntValue) (*IntValue, error):
			a, ok := args[0].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Int arguments", name)
			}
			b, ok := args[1].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Int arguments", name)
			}
			return impl(a, b)
		case func(*FloatValue, *FloatValue) (*FloatValue, error):
			a, ok := args[0].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Float arguments", name)
			}
			b, ok := args[1].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Float arguments", name)
			}
			return impl(a, b)
		case func(*StringValue, *StringValue) (*StringValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			b, ok := args[1].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			return impl(a, b)
		case func(*BoolValue, *BoolValue) (*BoolValue, error):
			a, ok := args[0].(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Bool arguments", name)
			}
			b, ok := args[1].(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Bool arguments", name)
			}
			return impl(a, b)
		case func(*IntValue, *IntValue) (*BoolValue, error):
			a, ok := args[0].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Int arguments", name)
			}
			b, ok := args[1].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Int arguments", name)
			}
			return impl(a, b)
		case func(*FloatValue, *FloatValue) (*BoolValue, error):
			a, ok := args[0].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Float arguments", name)
			}
			b, ok := args[1].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects Float arguments", name)
			}
			return impl(a, b)
		case func(*StringValue, *StringValue) (*BoolValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			b, ok := args[1].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			return impl(a, b)
		case func(*StringValue, *StringValue) (*IntValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			b, ok := args[1].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String arguments", name)
			}
			return impl(a, b)
		default:
			return nil, fmt.Errorf("unsupported builtin implementation for %s", name)
		}

	case 3:
		// Three-argument builtins (e.g., _str_slice)
		switch impl := builtin.Impl.(type) {
		case func(*StringValue, *IntValue, *IntValue) (*StringValue, error):
			a, ok := args[0].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
			}
			b, ok := args[1].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
			}
			c, ok := args[2].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("builtin %s expects String, Int, Int arguments", name)
			}
			return impl(a, b, c)
		default:
			return nil, fmt.Errorf("unsupported 3-arg builtin implementation for %s", name)
		}

	default:
		return nil, fmt.Errorf("unsupported arity %d for builtin %s", builtin.NumArgs, name)
	}
}

// registerStringPrimitiveBuiltins registers low-level string operations
// These are the critical primitives that can't be efficiently implemented in AILANG
func registerStringPrimitiveBuiltins() {
	// _str_len: UTF-8 aware string length (returns number of runes, not bytes)
	Builtins["_str_len"] = &BuiltinFunc{
		Name:    "_str_len",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*IntValue, error) {
			count := utf8.RuneCountInString(s.Value)
			return &IntValue{Value: count}, nil
		},
	}

	// _str_slice: substring with UTF-8 rune indices (not byte indices)
	// Indices are inclusive start, exclusive end
	Builtins["_str_slice"] = &BuiltinFunc{
		Name:    "_str_slice",
		NumArgs: 3,
		IsPure:  true,
		Impl: func(s *StringValue, start *IntValue, end *IntValue) (*StringValue, error) {
			runes := []rune(s.Value)
			length := len(runes)

			// Clamp indices to valid range
			st := start.Value
			if st < 0 {
				st = 0
			}
			if st > length {
				st = length
			}

			en := end.Value
			if en < st {
				en = st
			}
			if en > length {
				en = length
			}

			return &StringValue{Value: string(runes[st:en])}, nil
		},
	}

	// _str_compare: lexicographic comparison (-1, 0, 1)
	Builtins["_str_compare"] = &BuiltinFunc{
		Name:    "_str_compare",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a *StringValue, b *StringValue) (*IntValue, error) {
			if a.Value < b.Value {
				return &IntValue{Value: -1}, nil
			} else if a.Value > b.Value {
				return &IntValue{Value: 1}, nil
			} else {
				return &IntValue{Value: 0}, nil
			}
		},
	}

	// _str_find: find first occurrence of substring (returns -1 if not found)
	Builtins["_str_find"] = &BuiltinFunc{
		Name:    "_str_find",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(s *StringValue, sub *StringValue) (*IntValue, error) {
			// Find byte index first
			byteIdx := strings.Index(s.Value, sub.Value)
			if byteIdx == -1 {
				return &IntValue{Value: -1}, nil
			}
			// Convert byte index to rune index
			runeIdx := utf8.RuneCountInString(s.Value[:byteIdx])
			return &IntValue{Value: runeIdx}, nil
		},
	}

	// _str_upper: Unicode-aware uppercase conversion
	Builtins["_str_upper"] = &BuiltinFunc{
		Name:    "_str_upper",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.ToUpper(s.Value)}, nil
		},
	}

	// _str_lower: Unicode-aware lowercase conversion
	Builtins["_str_lower"] = &BuiltinFunc{
		Name:    "_str_lower",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.ToLower(s.Value)}, nil
		},
	}

	// _str_trim: Unicode-aware whitespace trimming
	Builtins["_str_trim"] = &BuiltinFunc{
		Name:    "_str_trim",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.TrimSpace(s.Value)}, nil
		},
	}
}

// registerIOBuiltins registers I/O operations with effects
func registerIOBuiltins() {
	// _io_print: print without newline (effectful: ! {IO})
	Builtins["_io_print"] = &BuiltinFunc{
		Name:    "_io_print",
		NumArgs: 1,
		IsPure:  false, // Effectful: IO
		Impl: func(s *StringValue) (*UnitValue, error) {
			fmt.Print(s.Value)
			return &UnitValue{}, nil
		},
	}

	// _io_println: print with newline (effectful: ! {IO})
	Builtins["_io_println"] = &BuiltinFunc{
		Name:    "_io_println",
		NumArgs: 1,
		IsPure:  false, // Effectful: IO
		Impl: func(s *StringValue) (*UnitValue, error) {
			fmt.Println(s.Value)
			return &UnitValue{}, nil
		},
	}

	// _io_readLine: read line from stdin (effectful: ! {IO})
	// TODO: implement proper readline support
	Builtins["_io_readLine"] = &BuiltinFunc{
		Name:    "_io_readLine",
		NumArgs: 0,
		IsPure:  false, // Effectful: IO
		Impl: func() (*StringValue, error) {
			// For v0.1.0, return empty string as stub
			// Full implementation will use bufio.Scanner
			return &StringValue{Value: ""}, nil
		},
	}
}

// NewRuntimeError creates a runtime error with structured information
func NewRuntimeError(code, message string, pos interface{}) error {
	// TODO: integrate with error encoder
	return fmt.Errorf("[%s] %s", code, message)
}

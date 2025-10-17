package eval

// Package eval/builtins provides the built-in function registry for AILANG.
//
// # Architecture
//
// The builtin system is split into several files by category:
//   - builtins.go: Main registry, types, and initialization (THIS FILE)
//   - builtins_arithmetic.go: Integer and float arithmetic (add, sub, mul, div, mod, neg)
//   - builtins_comparison.go: Comparison operations (eq, ne, lt, le, gt, ge)
//   - builtins_string.go: String operations (concat, slice, find, upper, lower, trim)
//   - builtins_boolean.go: Boolean operations (and, or, not)
//   - builtins_conversion.go: Type conversions (intToFloat, floatToInt)
//   - builtins_io.go: I/O operations (print, println, readLine)
//   - builtins_json.go: JSON encoding operations
//   - builtins_call.go: CallBuiltin dispatcher for invoking builtins
//   - builtins_errors.go: Error handling utilities
//
// # Usage
//
// Built-in functions are automatically registered during package initialization.
// To call a builtin:
//
//   result, err := CallBuiltin("add_Int", []Value{intVal1, intVal2})
//
// # Note on Effect-Based Builtins
//
// Most builtins in this package are pure functions or simple effectful operations.
// The new effect-based builtin system (internal/builtins/) provides a more
// structured approach for builtins that require capability checking (IO, FS, Net).
//
// This legacy system is maintained for backward compatibility and for simple
// builtins that don't need the full effect system.
//
// # See Also
//
//   - internal/builtins: New effect-based builtin registry
//   - internal/runtime: Runtime execution with effect context
//   - internal/effects: Effect system implementation

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
	registerConversionBuiltins()
	registerStringBuiltins()
	registerBooleanBuiltins()
	registerStringPrimitiveBuiltins()
	registerIOBuiltins()
	registerJSONBuiltins()
}

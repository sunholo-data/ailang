package eval

import "fmt"

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

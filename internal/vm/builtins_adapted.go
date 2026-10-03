package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// nativeBuiltinCount is the number of hand-written BuiltinTable entries, taken
// before the adapted entries are appended.
var nativeBuiltinCount = len(BuiltinTable)

// Adapted pure builtins (#1447). bytecode.AdaptedBuiltinNames lists the pure
// registry builtins whose signature converts both ways; the compiler gives them
// the OpBuiltinCall indices after the native entries, and init appends the
// matching dispatch entries here, in the same order. Each one converts its
// arguments to evaluator values, calls the builtin's registered Go Impl (not
// the tree-walking evaluator), and converts the result back.
func init() {
	for _, ir := range bytecode.AdaptedBuiltinNames {
		BuiltinTable = append(BuiltinTable, adaptBuiltin(ir))
	}
	if err := validateBuiltinTables(); err != nil {
		panic(err)
	}
}

func adaptBuiltin(ir string) BuiltinFunc {
	name := bytecode.AdaptedRegistryName(ir)
	spec, ok := builtins.GetSpec(name)
	if !ok {
		panic(fmt.Sprintf("vm: adapted builtin %q is not in the registry", name))
	}
	return func(args []bytecode.Value) (result bytecode.Value, err error) {
		evArgs := make([]eval.Value, len(args))
		for i, a := range args {
			ev, convErr := BytecodeToEval(a)
			if convErr != nil {
				return bytecode.Value{}, fmt.Errorf("%s: arg %d: %w", ir, i, convErr)
			}
			evArgs[i] = ev
		}
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s: builtin panicked: %v", ir, r)
			}
		}()
		out, callErr := spec.Impl(nil, evArgs)
		if callErr != nil {
			return bytecode.Value{}, callErr
		}
		bv, convErr := EvalToBytecode(out)
		if convErr != nil {
			return bytecode.Value{}, fmt.Errorf("%s: result: %w", ir, convErr)
		}
		return bv, nil
	}
}

// validateBuiltinTables checks that the VM dispatch tables line up with the
// compiler's index space: native entries, then adapted entries, and the HOF
// table. A mismatch would dispatch an OpBuiltinCall to the wrong function.
func validateBuiltinTables() error {
	if nativeBuiltinCount != len(bytecode.BuiltinNames) {
		return fmt.Errorf("vm: %d native builtins, compiler has %d", nativeBuiltinCount, len(bytecode.BuiltinNames))
	}
	if want := len(bytecode.BuiltinNames) + len(bytecode.AdaptedBuiltinNames); len(BuiltinTable) != want {
		return fmt.Errorf("vm: BuiltinTable has %d entries, compiler index space has %d", len(BuiltinTable), want)
	}
	if len(HOFBuiltinTable) != len(bytecode.HOFBuiltinNames) {
		return fmt.Errorf("vm: %d HOF builtins, compiler has %d", len(HOFBuiltinTable), len(bytecode.HOFBuiltinNames))
	}
	return nil
}

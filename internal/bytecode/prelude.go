package bytecode

import "sort"

// PreludeFunc says how the VM runs one implicit-prelude function.
type PreludeFunc struct {
	Effect string // an EffectBuiltinNames entry that implements it
	Arity  int
}

// PreludeFuncs maps each implicit-prelude function (the names
// pipeline.PreludeSurface() injects into entry modules without an import)
// to the effect builtin that implements it on the VM (M-VM-COVERAGE G2).
//
// pipeline.PreludeSurface() stays the single source of truth for WHAT the
// prelude contains. This table only says how the VM runs each entry, and a
// drift test (internal/runner, TestPreludeSurfaceHasVMImplementation)
// fails when the two disagree in either direction.
//
// The compiler consults it last, after locals, local letrec members and
// every top-level function, which is the prelude's own rule: user
// definitions shadow the prelude.
var PreludeFuncs = map[string]PreludeFunc{
	"println": {Effect: "__io_println", Arity: 1},
}

// PreludeNames lists the PreludeFuncs names in sorted order.
func PreludeNames() []string {
	names := make([]string, 0, len(PreludeFuncs))
	for name := range PreludeFuncs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

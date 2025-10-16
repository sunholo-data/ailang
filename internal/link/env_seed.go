package link

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	// Register our builtin env factory with types package
	// This breaks the import cycle: types calls link for builtin env
	types.SetBuiltinEnvFactory(newTypeEnvWithBuiltinsImpl)
}

// stubModuleLoader is a minimal ModuleLoader for seeding type env
// (we only need the $builtin interface, which is registered directly)
type stubModuleLoader struct{}

func (s *stubModuleLoader) LoadInterface(modulePath string) (*iface.Iface, error) {
	return nil, nil // Not used for $builtin
}

func (s *stubModuleLoader) EvaluateExport(ref core.GlobalRef) (eval.Value, error) {
	return nil, nil // Not used for type checking
}

// newTypeEnvWithBuiltinsImpl is the actual implementation, registered via init()
// This ensures the typechecker sees the exact same types (with effect rows intact)
// that the linker exports from the spec-based registry.
func newTypeEnvWithBuiltinsImpl() *types.TypeEnv {
	env := types.NewTypeEnv()

	// Ensure $builtin is registered (from spec registry)
	ml := NewModuleLinker(&stubModuleLoader{})
	RegisterBuiltinModule(ml)

	bi := ml.GetIface("$builtin")
	if bi == nil {
		return env
	}

	// Bind all exported builtins with their full type schemes (including effect rows)
	for name, item := range bi.Exports {
		env.BindScheme(name, item.Type)
	}

	return env
}

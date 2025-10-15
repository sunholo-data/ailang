package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// EffectImpl is the function signature for effect builtin implementations
type EffectImpl func(*effects.EffContext, []eval.Value) (eval.Value, error)

// BuiltinSpec defines a complete specification for a builtin function
// This consolidates all the information needed for registration:
// - Metadata (name, arity, purity)
// - Type signature
// - Implementation
type BuiltinSpec struct {
	Module  string            // Module path (e.g., "std/net", "std/io")
	Name    string            // Builtin name with _ prefix (e.g., "_net_httpRequest")
	NumArgs int               // Number of arguments (for arity checking)
	IsPure  bool              // true = no side effects, false = has effects
	Effect  string            // "" for pure functions, "Net"/"IO"/"FS" for effects
	Type    func() types.Type // Type signature constructor (must return non-nil)
	Impl    EffectImpl        // Implementation function
}

// specRegistry holds all registered builtin specifications
// This is the single source of truth for builtins
var specRegistry = make(map[string]*BuiltinSpec)

// frozen indicates whether the registry has been initialized
// Once frozen, no more registrations are allowed
var frozen = false

// RegisterEffectBuiltin registers a new effect builtin with complete validation
//
// This is the ONLY function you should use to register new builtins.
// It performs comprehensive validation and consolidates all registration steps.
//
// Example:
//
//	RegisterEffectBuiltin(BuiltinSpec{
//	    Module:  "std/net",
//	    Name:    "_net_httpRequest",
//	    NumArgs: 4,
//	    IsPure:  false,
//	    Effect:  "Net",
//	    Type:    makeHTTPRequestType,
//	    Impl:    effects.NetHTTPRequest,
//	})
//
// Validation performed:
//   - Name must not be empty
//   - Type function must not be nil
//   - Type function must return non-nil
//   - NumArgs must match type signature arity
//   - No duplicate registrations
//   - Impl function must not be nil
func RegisterEffectBuiltin(spec BuiltinSpec) error {
	if frozen {
		return fmt.Errorf("builtin registry is frozen, cannot register %s", spec.Name)
	}

	// 1. Validate name
	if spec.Name == "" {
		return fmt.Errorf("builtin name cannot be empty")
	}

	// 2. Validate Type function exists
	if spec.Type == nil {
		return fmt.Errorf("builtin %s: Type function is nil", spec.Name)
	}

	// 3. Build type and validate it's non-nil
	typ := spec.Type()
	if typ == nil {
		return fmt.Errorf("builtin %s: Type() returned nil", spec.Name)
	}

	// 4. Validate arity matches type signature
	if funcType, ok := typ.(*types.TFunc2); ok {
		if len(funcType.Params) != spec.NumArgs {
			return fmt.Errorf(
				"builtin %s: NumArgs=%d but type signature has %d arguments",
				spec.Name, spec.NumArgs, len(funcType.Params))
		}
	}

	// 5. Validate Impl function exists
	if spec.Impl == nil {
		return fmt.Errorf("builtin %s: Impl function is nil", spec.Name)
	}

	// 6. Check for duplicate registration
	if _, exists := specRegistry[spec.Name]; exists {
		return fmt.Errorf("builtin %s already registered", spec.Name)
	}

	// 7. Store in registry
	specRegistry[spec.Name] = &spec

	return nil
}

// GetSpec retrieves a builtin specification by name
func GetSpec(name string) (*BuiltinSpec, bool) {
	spec, ok := specRegistry[name]
	return spec, ok
}

// AllSpecs returns all registered builtin specifications
func AllSpecs() map[string]*BuiltinSpec {
	// Return copy to prevent external mutation
	result := make(map[string]*BuiltinSpec, len(specRegistry))
	for k, v := range specRegistry {
		result[k] = v
	}
	return result
}

// AllNames returns all registered builtin names
func AllNames() []string {
	names := make([]string, 0, len(specRegistry))
	for name := range specRegistry {
		names = append(names, name)
	}
	return names
}

// Init freezes the registry after all registrations are complete
// This wires up the registry to runtime dispatch and link interface
func Init() error {
	if frozen {
		return fmt.Errorf("builtins already initialized")
	}

	// TODO: Wire to runtime dispatch (M-DX1.1 implementation)
	// TODO: Build link interface (M-DX1.1 implementation)

	frozen = true
	return nil
}

// IsFrozen returns whether the registry has been initialized
func IsFrozen() bool {
	return frozen
}

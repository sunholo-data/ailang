package pipeline

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/types"
)

// CanonBuiltin is a canonical representation of a builtin for testing
// This allows deterministic comparison across three sources:
// 1. Spec registry (internal/builtins/spec.go)
// 2. Linker interface (internal/link/builtin_module.go)
// 3. Type environment (internal/types/env.go via internal/link/env_seed.go)
type CanonBuiltin struct {
	Name    string   // Builtin name (e.g., "_io_print")
	Arity   int      // Number of arguments
	Effects []string // Sorted effect labels (e.g., ["IO"])
	Pure    bool     // true = no side effects
}

func (c CanonBuiltin) String() string {
	eff := "pure"
	if !c.Pure {
		eff = "! {" + strings.Join(c.Effects, ",") + "}"
	}
	return fmt.Sprintf("%s/%d %s", c.Name, c.Arity, eff)
}

// TestBuiltinConsistency_ThreeWayParity is the CRITICAL regression guard for v0.3.10-style bugs.
//
// The v0.3.10 regression occurred because:
// 1. Spec registry had correct types with effect rows
// 2. Linker interface exported them correctly
// 3. BUT TypeEnv initialization lost the effect rows during copying
//
// This test ensures all three representations agree on:
// - Which builtins exist (names)
// - How many arguments they take (arity)
// - Which effects they have (effect labels)
// - Whether they're pure (purity)
//
// If this test fails, it means one of the three systems is out of sync.
func TestBuiltinConsistency_ThreeWayParity(t *testing.T) {
	// 1. Get canonical view from spec registry
	specBuiltins := canonicalFromSpecs(builtins.AllSpecs())

	// 2. Get canonical view from linker's $builtin interface
	linkBuiltins := canonicalFromInterface(t)

	// 3. Get canonical view from typechecker's type env
	typeEnvBuiltins := canonicalFromTypeEnv(types.NewTypeEnvWithBuiltins())

	// Assert all three match exactly
	require.Equal(t, specBuiltins, linkBuiltins,
		"CONSISTENCY VIOLATION: Spec registry ≠ Linker interface\n"+
			"This means internal/link/builtin_module.go is not reading from the spec registry correctly.\n"+
			"Diff:\n%s", diffBuiltins(specBuiltins, linkBuiltins))

	require.Equal(t, specBuiltins, typeEnvBuiltins,
		"CONSISTENCY VIOLATION: Spec registry ≠ Type env\n"+
			"This means internal/link/env_seed.go is losing information during TypeEnv initialization.\n"+
			"This is the EXACT bug from v0.3.10 (lost effect rows)!\n"+
			"Diff:\n%s", diffBuiltins(specBuiltins, typeEnvBuiltins))

	// Sanity check: we should have builtins
	assert.Greater(t, len(specBuiltins), 40,
		"Expected at least 40 builtins, got %d. Registry may not be initialized.", len(specBuiltins))
}

// TestBuiltinConsistency_SpecRegistryComplete verifies the spec registry has expected builtins
func TestBuiltinConsistency_SpecRegistryComplete(t *testing.T) {
	specs := builtins.AllSpecs()

	// Check critical builtins exist
	criticalBuiltins := []struct {
		name   string
		arity  int
		effect string
	}{
		{"_io_print", 1, "IO"},
		{"_io_println", 1, "IO"},
		{"_io_readLine", 0, "IO"},
		{"_net_httpRequest", 4, "Net"},
		{"_str_len", 1, ""},
		{"concat_String", 2, ""}, // String concatenation (pure)
	}

	for _, expected := range criticalBuiltins {
		t.Run(expected.name, func(t *testing.T) {
			spec, ok := specs[expected.name]
			require.True(t, ok, "Critical builtin %s missing from spec registry", expected.name)
			assert.Equal(t, expected.arity, spec.NumArgs, "Arity mismatch for %s", expected.name)
			assert.Equal(t, expected.effect, spec.Effect, "Effect mismatch for %s", expected.name)

			// Verify type can be constructed
			typ := spec.Type()
			require.NotNil(t, typ, "Type() returned nil for %s", expected.name)
		})
	}
}

// TestBuiltinConsistency_EffectLabelsMatchDeclaration verifies effect labels in types match spec declarations
func TestBuiltinConsistency_EffectLabelsMatchDeclaration(t *testing.T) {
	specs := builtins.AllSpecs()

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			typ := spec.Type()
			require.NotNil(t, typ, "Type() returned nil for %s", name)

			// Extract effect labels from type
			effects := extractEffects(typ)

			// Check consistency with spec.Effect and spec.IsPure
			if spec.IsPure {
				assert.Empty(t, effects,
					"Builtin %s marked IsPure=true but has effects %v", name, effects)
				assert.Empty(t, spec.Effect,
					"Builtin %s marked IsPure=true but has Effect=%q", name, spec.Effect)
			} else {
				assert.NotEmpty(t, spec.Effect,
					"Builtin %s marked IsPure=false but has no Effect", name)

				// If spec declares an effect, type must have it
				if spec.Effect != "" {
					assert.Contains(t, effects, spec.Effect,
						"Builtin %s declares Effect=%q but type has effects %v",
						name, spec.Effect, effects)
				}
			}
		})
	}
}

// canonicalFromSpecs extracts canonical builtins from the spec registry
func canonicalFromSpecs(specs map[string]*builtins.BuiltinSpec) []CanonBuiltin {
	var result []CanonBuiltin
	for name, spec := range specs {
		effects := extractEffects(spec.Type())
		sort.Strings(effects) // CRITICAL: deterministic order

		result = append(result, CanonBuiltin{
			Name:    name,
			Arity:   spec.NumArgs,
			Effects: effects,
			Pure:    spec.IsPure,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name // Deterministic sort
	})
	return result
}

// canonicalFromInterface extracts canonical builtins from the linker's $builtin interface
func canonicalFromInterface(t *testing.T) []CanonBuiltin {
	// Create a module linker and register $builtin
	ml := link.NewModuleLinker(&stubModuleLoader{})
	link.RegisterBuiltinModule(ml)

	bi := ml.GetIface("$builtin")
	require.NotNil(t, bi, "$builtin interface not registered")

	var result []CanonBuiltin
	for name, item := range bi.Exports {
		// Extract type from scheme
		typ := item.Type.Type

		effects := extractEffects(typ)
		sort.Strings(effects)

		// Extract arity
		arity := 0
		if fn, ok := typ.(*types.TFunc2); ok {
			arity = len(fn.Params)
		}

		result = append(result, CanonBuiltin{
			Name:    name,
			Arity:   arity,
			Effects: effects,
			Pure:    item.Purity,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// canonicalFromTypeEnv extracts canonical builtins from the type environment
func canonicalFromTypeEnv(env *types.TypeEnv) []CanonBuiltin {
	// We need to enumerate all builtins - get names from spec registry
	specs := builtins.AllSpecs()

	var result []CanonBuiltin
	for name := range specs {
		// Lookup in type env
		binding, err := env.Lookup(name)
		if err != nil {
			// Builtin missing from type env!
			continue
		}

		// Extract type
		var typ types.Type
		if scheme, ok := binding.(*types.Scheme); ok {
			typ = scheme.Type
		} else {
			typ = binding.(types.Type)
		}

		effects := extractEffects(typ)
		sort.Strings(effects)

		// Extract arity
		arity := 0
		if fn, ok := typ.(*types.TFunc2); ok {
			arity = len(fn.Params)
		}

		// Determine purity (pure if no effects)
		pure := len(effects) == 0

		result = append(result, CanonBuiltin{
			Name:    name,
			Arity:   arity,
			Effects: effects,
			Pure:    pure,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// extractEffects pulls effect labels from a TFunc2.EffectRow
func extractEffects(typ types.Type) []string {
	fn, ok := typ.(*types.TFunc2)
	if !ok || fn.EffectRow == nil {
		return nil
	}

	var labels []string
	for label := range fn.EffectRow.Labels {
		labels = append(labels, label)
	}
	return labels
}

// diffBuiltins generates a human-readable diff of two builtin lists
func diffBuiltins(a, b []CanonBuiltin) string {
	aMap := make(map[string]CanonBuiltin)
	bMap := make(map[string]CanonBuiltin)

	for _, builtin := range a {
		aMap[builtin.Name] = builtin
	}
	for _, builtin := range b {
		bMap[builtin.Name] = builtin
	}

	var diffs []string

	// Check for missing in b
	for name := range aMap {
		if _, ok := bMap[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("  - %s: present in A, missing in B", name))
		}
	}

	// Check for missing in a
	for name := range bMap {
		if _, ok := aMap[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("  + %s: missing in A, present in B", name))
		}
	}

	// Check for differences
	for name, aBuiltin := range aMap {
		if bBuiltin, ok := bMap[name]; ok {
			if !canonBuiltinsEqual(aBuiltin, bBuiltin) {
				diffs = append(diffs, fmt.Sprintf("  ~ %s:\n    A: %s\n    B: %s",
					name, aBuiltin.String(), bBuiltin.String()))
			}
		}
	}

	if len(diffs) == 0 {
		return "  (no differences)"
	}

	sort.Strings(diffs)
	return strings.Join(diffs, "\n")
}

// canonBuiltinsEqual compares two CanonBuiltin structs for equality
func canonBuiltinsEqual(a, b CanonBuiltin) bool {
	if a.Name != b.Name || a.Arity != b.Arity || a.Pure != b.Pure {
		return false
	}
	if len(a.Effects) != len(b.Effects) {
		return false
	}
	for i := range a.Effects {
		if a.Effects[i] != b.Effects[i] {
			return false
		}
	}
	return true
}

// stubModuleLoader is a minimal ModuleLoader for testing
type stubModuleLoader struct{}

func (s *stubModuleLoader) LoadInterface(modulePath string) (*iface.Iface, error) {
	return nil, nil
}

func (s *stubModuleLoader) EvaluateExport(ref core.GlobalRef) (eval.Value, error) {
	return nil, nil
}

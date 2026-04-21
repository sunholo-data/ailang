// Package testing provides inline test execution for AILANG.
// This file implements pure cluster extraction for cross-function dependencies.

package testing

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// PureCluster represents a set of pure functions that can be tested together.
type PureCluster struct {
	// FuncName is the function under test
	FuncName string
	// Bindings are all the RecBindings needed (in dependency order)
	Bindings []core.RecBinding
	// Names is the set of function names in the cluster
	Names map[string]bool
}

// PurityError is returned when a function depends on an effectful function.
type PurityError struct {
	// FuncUnderTest is the function being tested
	FuncUnderTest string
	// EffectfulFunc is the function that has effects
	EffectfulFunc string
	// Effects is the list of effects the function has
	Effects []string
}

func (e *PurityError) Error() string {
	return fmt.Sprintf(
		"cannot test '%s': dependency '%s' has effects %v",
		e.FuncUnderTest, e.EffectfulFunc, e.Effects,
	)
}

// IsPure checks if a binding's type indicates it's a pure function.
// A function is pure if its effect row is nil or empty.
func IsPure(binding *core.RecBinding, coreTypeInfo types.CoreTypeInfo) bool {
	// Get the type of the binding's value (should be the lambda)
	if binding.Value == nil {
		return true // No value means no effects
	}

	// Get the node's type from CoreTypeInfo
	nodeID := binding.Value.ID()
	typ, ok := coreTypeInfo.Get(nodeID)
	if !ok {
		// If no type info, assume pure (conservative for non-function values)
		return true
	}

	// Check if it's a function type with effects
	funcType, ok := typ.(*types.TFunc2)
	if !ok {
		// Not a function type, so no effects
		return true
	}

	// Pure if EffectRow is nil or has no labels
	if funcType.EffectRow == nil {
		return true
	}
	return len(funcType.EffectRow.Labels) == 0
}

// GetEffectNames returns the effect names for a binding, or nil if pure.
func GetEffectNames(binding *core.RecBinding, coreTypeInfo types.CoreTypeInfo) []string {
	if binding.Value == nil {
		return nil
	}

	nodeID := binding.Value.ID()
	typ, ok := coreTypeInfo.Get(nodeID)
	if !ok {
		return nil
	}

	funcType, ok := typ.(*types.TFunc2)
	if !ok {
		return nil
	}

	if funcType.EffectRow == nil || len(funcType.EffectRow.Labels) == 0 {
		return nil
	}

	effects := make([]string, 0, len(funcType.EffectRow.Labels))
	for name := range funcType.EffectRow.Labels {
		effects = append(effects, name)
	}
	sort.Strings(effects)
	return effects
}

// ExtractPureCluster extracts the pure cluster for a function.
// Returns error if any dependency has effects.
func ExtractPureCluster(
	funcName string,
	g *CallGraph,
	sccs [][]string,
	coreTypeInfo types.CoreTypeInfo,
) (*PureCluster, error) {
	// Get the dependency closure
	closure := GetDependencyClosure(g, sccs, funcName)
	if closure == nil {
		return nil, fmt.Errorf("function '%s' not found in call graph", funcName)
	}

	// Check that all functions in the closure are pure
	for _, name := range closure {
		binding, ok := g.Bindings[name]
		if !ok {
			continue // Skip missing bindings (shouldn't happen)
		}

		if !IsPure(binding, coreTypeInfo) {
			effects := GetEffectNames(binding, coreTypeInfo)
			return nil, &PurityError{
				FuncUnderTest: funcName,
				EffectfulFunc: name,
				Effects:       effects,
			}
		}
	}

	// Collect bindings in topological order (dependencies first)
	bindings := make([]core.RecBinding, 0, len(closure))
	names := make(map[string]bool)

	// We need reverse topological order for dependencies.
	// SCCs are returned in reverse topological order by Tarjan,
	// so we can use that to order bindings.
	sccOrder := make(map[string]int)
	for i, scc := range sccs {
		for _, name := range scc {
			sccOrder[name] = i
		}
	}

	// Sort closure by SCC order (higher SCC index = earlier in dependency chain)
	sortedClosure := make([]string, len(closure))
	copy(sortedClosure, closure)
	sort.Slice(sortedClosure, func(i, j int) bool {
		return sccOrder[sortedClosure[i]] > sccOrder[sortedClosure[j]]
	})

	for _, name := range sortedClosure {
		binding, ok := g.Bindings[name]
		if ok {
			bindings = append(bindings, *binding)
			names[name] = true
		}
	}

	return &PureCluster{
		FuncName: funcName,
		Bindings: bindings,
		Names:    names,
	}, nil
}

// HasDependencies returns true if the function has dependencies beyond itself.
func (pc *PureCluster) HasDependencies() bool {
	return len(pc.Names) > 1
}

// DependencyNames returns the names of dependencies (excluding the function under test).
func (pc *PureCluster) DependencyNames() []string {
	deps := make([]string, 0, len(pc.Names)-1)
	for name := range pc.Names {
		if name != pc.FuncName {
			deps = append(deps, name)
		}
	}
	sort.Strings(deps)
	return deps
}

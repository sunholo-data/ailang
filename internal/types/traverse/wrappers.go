package traverse

import (
	"github.com/sunholo/ailang/internal/types"
)

// CollectFreeVars returns all free type variable names in a type.
// Safe for cyclic types - will not hang on self-referential type graphs.
//
// Example:
//
//	vars := traverse.CollectFreeVars(funcType)
//	// vars might be: {"a": true, "b": true}
func CollectFreeVars(t types.Type) map[string]bool {
	vars := make(map[string]bool)
	Walk(t, func(typ types.Type) {
		switch v := typ.(type) {
		case *types.TVar:
			vars[v.Name] = true
		case *types.TVar2:
			vars[v.Name] = true
		case *types.RowVar:
			vars[v.Name] = true
		}
	})
	return vars
}

// ContainsType checks if a type graph contains a specific target type.
// Uses pointer equality for comparison.
// Safe for cyclic types.
//
// Example:
//
//	if traverse.ContainsType(funcType, targetVar) {
//	    // targetVar appears somewhere in funcType
//	}
func ContainsType(t, target types.Type) bool {
	found := false
	Walk(t, func(typ types.Type) {
		if typ == target {
			found = true
		}
	})
	return found
}

// ContainsTypeByName checks if a type graph contains a type variable with the given name.
// Safe for cyclic types.
//
// Example:
//
//	if traverse.ContainsTypeByName(funcType, "a") {
//	    // Type variable "a" appears somewhere in funcType
//	}
func ContainsTypeByName(t types.Type, name string) bool {
	found := false
	Walk(t, func(typ types.Type) {
		switch v := typ.(type) {
		case *types.TVar:
			if v.Name == name {
				found = true
			}
		case *types.TVar2:
			if v.Name == name {
				found = true
			}
		case *types.RowVar:
			if v.Name == name {
				found = true
			}
		}
	})
	return found
}

// CountTypes returns the total number of type nodes in a type graph.
// Safe for cyclic types - cycles are counted once.
//
// Useful for complexity estimation and debugging.
func CountTypes(t types.Type) int {
	count := 0
	Walk(t, func(typ types.Type) {
		count++
	})
	return count
}

// CollectTypesByKind returns all types of a specific kind in a type graph.
// The predicate function determines which types to collect.
// Safe for cyclic types.
//
// Example:
//
//	funcs := traverse.CollectTypesByKind(t, func(typ types.Type) bool {
//	    _, ok := typ.(*types.TFunc2)
//	    return ok
//	})
func CollectTypesByKind(t types.Type, predicate func(types.Type) bool) []types.Type {
	var result []types.Type
	Walk(t, func(typ types.Type) {
		if predicate(typ) {
			result = append(result, typ)
		}
	})
	return result
}

// AllTypes returns all types in a type graph in pre-order traversal.
// Safe for cyclic types - cycles are visited once.
func AllTypes(t types.Type) []types.Type {
	var result []types.Type
	Walk(t, func(typ types.Type) {
		result = append(result, typ)
	})
	return result
}

// HasCycles checks if a type graph contains any cycles.
// Returns true if the type references itself directly or indirectly.
//
// Example:
//
//	if traverse.HasCycles(adtType) {
//	    // This is a recursive type like List[T] or Tree[T]
//	}
func HasCycles(t types.Type) bool {
	hasCycle := false
	NewVisitor().WithOnCycle(func(typ types.Type) {
		hasCycle = true
	}).Visit(t, func(typ types.Type) {})
	return hasCycle
}

// HasTypeVars checks if a type contains any type variables (TVar, TVar2, RowVar).
// Safe for cyclic types - will not hang on recursive ADTs.
//
// This is the cycle-safe replacement for isPolymorphic checks.
// Use this in monomorphization and specialization passes.
//
// Example:
//
//	if traverse.HasTypeVars(funcType) {
//	    // Type is polymorphic, needs specialization
//	}
func HasTypeVars(t types.Type) bool {
	hasVars := false
	Walk(t, func(typ types.Type) {
		switch typ.(type) {
		case *types.TVar, *types.TVar2, *types.RowVar:
			hasVars = true
		}
	})
	return hasVars
}

// IsMonomorphic checks if a type is fully concrete (no type variables).
// This is the inverse of HasTypeVars.
// Safe for cyclic types.
//
// Example:
//
//	if traverse.IsMonomorphic(funcType) {
//	    // Type can be used directly without specialization
//	}
func IsMonomorphic(t types.Type) bool {
	return !HasTypeVars(t)
}

// Depth returns the maximum nesting depth of a type graph.
// Safe for cyclic types - cycles are detected and counted once.
//
// Useful for complexity analysis and debugging.
func Depth(t types.Type) int {
	if t == nil {
		return 0
	}

	maxChildDepth := 0
	v := &depthVisitor{
		visited:  make(map[types.Type]bool),
		maxDepth: DefaultMaxDepth,
	}
	v.computeDepth(t, 0, &maxChildDepth)
	return maxChildDepth
}

type depthVisitor struct {
	visited  map[types.Type]bool
	maxDepth int
}

func (v *depthVisitor) computeDepth(t types.Type, currentDepth int, maxDepth *int) {
	if t == nil {
		return
	}
	if currentDepth > *maxDepth {
		*maxDepth = currentDepth
	}
	if v.visited[t] || currentDepth > v.maxDepth {
		return
	}
	v.visited[t] = true

	// Get children using TypeVisitor's children method
	tv := &TypeVisitor{}
	for _, child := range tv.children(t) {
		v.computeDepth(child, currentDepth+1, maxDepth)
	}

	delete(v.visited, t)
}

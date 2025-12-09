// Package traverse provides safe type traversal with automatic cycle detection.
// This prevents infinite loops when traversing cyclic type graphs (e.g., recursive ADTs).
//
// API Design Rule (from M-PERF2 post-mortem):
// Every function of shape `func(Type) T` MUST document cycle-safety.
// Either use traverse.Visit or add a `visited` parameter.
package traverse

import (
	"fmt"

	"github.com/sunholo/ailang/internal/types"
)

// DefaultMaxDepth is the maximum traversal depth before panic.
// This prevents hangs on pathological cases even if cycle detection fails.
const DefaultMaxDepth = 1000

// TypeVisitor traverses type graphs with automatic cycle detection.
// Use NewVisitor() to create instances.
type TypeVisitor struct {
	visited  map[types.Type]bool
	depth    int
	maxDepth int

	// OnCycle is called when a cycle is detected.
	// If nil, cycles are silently skipped.
	OnCycle func(typ types.Type)
}

// NewVisitor creates a TypeVisitor with default settings.
func NewVisitor() *TypeVisitor {
	return &TypeVisitor{
		visited:  make(map[types.Type]bool),
		maxDepth: DefaultMaxDepth,
	}
}

// WithMaxDepth sets a custom maximum depth (for testing).
func (v *TypeVisitor) WithMaxDepth(depth int) *TypeVisitor {
	v.maxDepth = depth
	return v
}

// WithOnCycle sets a callback for cycle detection.
func (v *TypeVisitor) WithOnCycle(fn func(types.Type)) *TypeVisitor {
	v.OnCycle = fn
	return v
}

// Visit traverses a type graph, calling fn for each type node.
// Automatically detects cycles and enforces depth limits.
//
// The visitor uses pre-order traversal: fn is called before children.
// Cycles are detected and skipped (or OnCycle is called if set).
func (v *TypeVisitor) Visit(t types.Type, fn func(types.Type)) {
	if t == nil {
		return
	}

	// Depth limit check
	if v.depth > v.maxDepth {
		panic(fmt.Sprintf("traverse: depth limit %d exceeded on %T", v.maxDepth, t))
	}

	// Cycle detection
	if v.visited[t] {
		if v.OnCycle != nil {
			v.OnCycle(t)
		}
		return
	}

	// Mark as visiting
	v.visited[t] = true
	v.depth++
	defer func() {
		v.depth--
		delete(v.visited, t)
	}()

	// Call the visitor function
	fn(t)

	// Recursively visit children
	for _, child := range v.children(t) {
		v.Visit(child, fn)
	}
}

// children returns all child types for a given type.
// This covers all Type variants in the type system.
func (v *TypeVisitor) children(t types.Type) []types.Type {
	switch typ := t.(type) {
	// Simple types with no children
	case *types.TVar:
		return nil
	case *types.TVar2:
		return nil
	case *types.TCon:
		return nil
	case *types.RowVar:
		return nil

	// Function types
	case *types.TFunc:
		children := make([]types.Type, 0, len(typ.Params)+1)
		children = append(children, typ.Params...)
		children = append(children, typ.Return)
		return children

	case *types.TFunc2:
		children := make([]types.Type, 0, len(typ.Params)+2)
		children = append(children, typ.Params...)
		children = append(children, typ.Return)
		if typ.EffectRow != nil {
			children = append(children, typ.EffectRow)
		}
		return children

	// Collection types
	case *types.TList:
		return []types.Type{typ.Element}

	case *types.TArray:
		return []types.Type{typ.Element}

	case *types.TTuple:
		return typ.Elements

	// Record types
	case *types.TRecord:
		children := make([]types.Type, 0, len(typ.Fields)+1)
		for _, fieldType := range typ.Fields {
			children = append(children, fieldType)
		}
		if typ.Row != nil {
			children = append(children, typ.Row)
		}
		return children

	case *types.TRecordOpen:
		children := make([]types.Type, 0, len(typ.Fields)+1)
		for _, fieldType := range typ.Fields {
			children = append(children, fieldType)
		}
		if typ.Row != nil {
			children = append(children, typ.Row)
		}
		return children

	case *types.TRecord2:
		if typ.Row != nil {
			return []types.Type{typ.Row}
		}
		return nil

	// Type application
	case *types.TApp:
		children := make([]types.Type, 0, len(typ.Args)+1)
		children = append(children, typ.Constructor)
		children = append(children, typ.Args...)
		return children

	// Row type
	case *types.Row:
		children := make([]types.Type, 0, len(typ.Labels)+1)
		// Only include label types for record rows (effect rows have no types)
		if typ.Kind.Equals(types.RecordRow) {
			for _, labelType := range typ.Labels {
				children = append(children, labelType)
			}
		}
		if typ.Tail != nil {
			children = append(children, typ.Tail)
		}
		return children

	default:
		// Unknown type - no children
		return nil
	}
}

// Walk is a convenience function that creates a visitor and traverses.
// For simple one-off traversals where you don't need OnCycle.
func Walk(t types.Type, fn func(types.Type)) {
	NewVisitor().Visit(t, fn)
}

// WalkWithCycleCallback traverses with a cycle callback.
func WalkWithCycleCallback(t types.Type, fn func(types.Type), onCycle func(types.Type)) {
	NewVisitor().WithOnCycle(onCycle).Visit(t, fn)
}

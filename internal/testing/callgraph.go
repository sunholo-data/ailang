// Package testing provides inline test execution for AILANG.
// This file implements call graph construction and SCC computation
// for cross-function dependency support (M-TESTING-DEPS).

package testing

import (
	"github.com/sunholo/ailang/internal/core"
)

// CallGraph represents dependencies between functions in a module.
// Nodes are function names, edges are directed "calls" relationships.
type CallGraph struct {
	// Nodes is the set of all function names in the module
	Nodes map[string]bool
	// Edges maps each function to the functions it calls
	Edges map[string][]string
	// Bindings maps function names to their RecBinding
	Bindings map[string]*core.RecBinding
}

// NewCallGraph creates an empty call graph.
func NewCallGraph() *CallGraph {
	return &CallGraph{
		Nodes:    make(map[string]bool),
		Edges:    make(map[string][]string),
		Bindings: make(map[string]*core.RecBinding),
	}
}

// BuildCallGraph constructs a call graph from a Core program.
// It identifies all top-level function bindings and their dependencies.
func BuildCallGraph(prog *core.Program) *CallGraph {
	g := NewCallGraph()

	// First pass: collect all function bindings (from LetRec and Let)
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			for i := range d.Bindings {
				binding := &d.Bindings[i]
				g.Nodes[binding.Name] = true
				g.Bindings[binding.Name] = binding
			}
		case *core.Let:
			// Non-recursive functions
			binding := &core.RecBinding{
				Name:  d.Name,
				Value: d.Value,
			}
			g.Nodes[d.Name] = true
			g.Bindings[d.Name] = binding
		}
	}

	// Second pass: find edges (function calls)
	for name, binding := range g.Bindings {
		refs := collectVarReferences(binding.Value, g.Nodes)
		// Remove self-reference for edge purposes (self-recursion is handled by SCC)
		edges := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref != name {
				edges = append(edges, ref)
			}
		}
		g.Edges[name] = edges
	}

	return g
}

// collectVarReferences walks a Core expression and collects all Var references
// that correspond to function names in the module.
func collectVarReferences(expr core.CoreExpr, functionNames map[string]bool) []string {
	refs := make(map[string]bool)
	collectVarRefsWalk(expr, functionNames, refs)

	result := make([]string, 0, len(refs))
	for name := range refs {
		result = append(result, name)
	}
	return result
}

// collectVarRefsWalk recursively walks a Core expression.
func collectVarRefsWalk(expr core.CoreExpr, functionNames map[string]bool, refs map[string]bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *core.Var:
		if functionNames[e.Name] {
			refs[e.Name] = true
		}

	case *core.Lambda:
		collectVarRefsWalk(e.Body, functionNames, refs)

	case *core.Let:
		collectVarRefsWalk(e.Value, functionNames, refs)
		collectVarRefsWalk(e.Body, functionNames, refs)

	case *core.LetRec:
		for _, b := range e.Bindings {
			collectVarRefsWalk(b.Value, functionNames, refs)
		}
		collectVarRefsWalk(e.Body, functionNames, refs)

	case *core.App:
		collectVarRefsWalk(e.Func, functionNames, refs)
		for _, arg := range e.Args {
			collectVarRefsWalk(arg, functionNames, refs)
		}

	case *core.If:
		collectVarRefsWalk(e.Cond, functionNames, refs)
		collectVarRefsWalk(e.Then, functionNames, refs)
		collectVarRefsWalk(e.Else, functionNames, refs)

	case *core.Match:
		collectVarRefsWalk(e.Scrutinee, functionNames, refs)
		for _, arm := range e.Arms {
			collectVarRefsWalk(arm.Body, functionNames, refs)
		}

	case *core.BinOp:
		collectVarRefsWalk(e.Left, functionNames, refs)
		collectVarRefsWalk(e.Right, functionNames, refs)

	case *core.UnOp:
		collectVarRefsWalk(e.Operand, functionNames, refs)

	case *core.Record:
		for _, fieldExpr := range e.Fields {
			collectVarRefsWalk(fieldExpr, functionNames, refs)
		}

	case *core.RecordAccess:
		collectVarRefsWalk(e.Record, functionNames, refs)

	case *core.RecordUpdate:
		collectVarRefsWalk(e.Base, functionNames, refs)
		for _, updateExpr := range e.Updates {
			collectVarRefsWalk(updateExpr, functionNames, refs)
		}

	case *core.List:
		for _, elem := range e.Elements {
			collectVarRefsWalk(elem, functionNames, refs)
		}

	case *core.Tuple:
		for _, elem := range e.Elements {
			collectVarRefsWalk(elem, functionNames, refs)
		}

	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectVarRefsWalk(arg, functionNames, refs)
		}

	case *core.DictAbs:
		collectVarRefsWalk(e.Body, functionNames, refs)

	case *core.DictApp:
		collectVarRefsWalk(e.Dict, functionNames, refs)
		for _, arg := range e.Args {
			collectVarRefsWalk(arg, functionNames, refs)
		}

	// Literals and global references don't contain local var references
	case *core.Lit, *core.VarGlobal:
		// No var references to local functions
	}
}

// ComputeSCCs computes the strongly connected components of the call graph
// using Tarjan's algorithm. Returns SCCs in reverse topological order
// (dependencies come before dependents).
func ComputeSCCs(g *CallGraph) [][]string {
	t := &tarjanState{
		graph:    g,
		index:    0,
		stack:    make([]string, 0),
		onStack:  make(map[string]bool),
		indices:  make(map[string]int),
		lowlinks: make(map[string]int),
		sccs:     make([][]string, 0),
	}

	for node := range g.Nodes {
		if _, visited := t.indices[node]; !visited {
			t.strongConnect(node)
		}
	}

	return t.sccs
}

// tarjanState holds the state for Tarjan's SCC algorithm.
type tarjanState struct {
	graph    *CallGraph
	index    int
	stack    []string
	onStack  map[string]bool
	indices  map[string]int
	lowlinks map[string]int
	sccs     [][]string
}

// strongConnect is the recursive part of Tarjan's algorithm.
func (t *tarjanState) strongConnect(v string) {
	// Set the depth index for v
	t.indices[v] = t.index
	t.lowlinks[v] = t.index
	t.index++

	// Push v onto the stack
	t.stack = append(t.stack, v)
	t.onStack[v] = true

	// Consider successors of v
	for _, w := range t.graph.Edges[v] {
		if _, visited := t.indices[w]; !visited {
			// Successor w has not yet been visited; recurse
			t.strongConnect(w)
			if t.lowlinks[w] < t.lowlinks[v] {
				t.lowlinks[v] = t.lowlinks[w]
			}
		} else if t.onStack[w] {
			// Successor w is on the stack and hence in the current SCC
			if t.indices[w] < t.lowlinks[v] {
				t.lowlinks[v] = t.indices[w]
			}
		}
	}

	// If v is a root node, pop the stack and generate an SCC
	if t.lowlinks[v] == t.indices[v] {
		scc := make([]string, 0)
		for {
			w := t.stack[len(t.stack)-1]
			t.stack = t.stack[:len(t.stack)-1]
			t.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		t.sccs = append(t.sccs, scc)
	}
}

// FindSCCContaining returns the SCC that contains the given function name.
func FindSCCContaining(sccs [][]string, funcName string) []string {
	for _, scc := range sccs {
		for _, name := range scc {
			if name == funcName {
				return scc
			}
		}
	}
	return nil
}

// GetDependencyClosure returns all functions reachable from the given function,
// including the function itself and its SCC.
func GetDependencyClosure(g *CallGraph, sccs [][]string, funcName string) []string {
	// Find the SCC containing the target function
	targetSCC := FindSCCContaining(sccs, funcName)
	if targetSCC == nil {
		return nil
	}

	// BFS to find all reachable SCCs
	visited := make(map[string]bool)
	result := make([]string, 0)

	// Start with all functions in the target SCC
	queue := make([]string, 0)
	for _, name := range targetSCC {
		visited[name] = true
		result = append(result, name)
		queue = append(queue, name)
	}

	// BFS through dependencies
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range g.Edges[current] {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				queue = append(queue, dep)
			}
		}
	}

	return result
}

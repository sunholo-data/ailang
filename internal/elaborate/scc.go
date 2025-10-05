// Package elaborate provides SCC detection for mutual recursion
package elaborate

import (
	"github.com/sunholo/ailang/internal/ast"
)

// FuncSig represents a function signature for call graph analysis
type FuncSig struct {
	Name     string
	NodeSID  string // Surface SID
	Body     ast.Expr
	Params   []string
	IsPure   bool
	IsExport bool
	Tests    []*ast.TestCase
	Props    []*ast.Property
	FuncDecl *ast.FuncDecl // Original declaration
}

// CallGraph represents a dependency graph between functions
type CallGraph struct {
	nodes   []string
	edges   map[string][]string
	nodeSet map[string]bool
}

// NewCallGraph creates a new call graph
func NewCallGraph() *CallGraph {
	return &CallGraph{
		edges:   make(map[string][]string),
		nodeSet: make(map[string]bool),
	}
}

// AddNode adds a function to the graph
func (g *CallGraph) AddNode(name string) {
	if !g.nodeSet[name] {
		g.nodes = append(g.nodes, name)
		g.nodeSet[name] = true
		g.edges[name] = []string{}
	}
}

// AddEdge adds a dependency from caller to callee
func (g *CallGraph) AddEdge(caller, callee string) {
	g.AddNode(caller)
	g.AddNode(callee)
	g.edges[caller] = append(g.edges[caller], callee)
}

// SCCs computes strongly connected components using Tarjan's algorithm
func (g *CallGraph) SCCs() [][]string {
	index := 0
	stack := []string{}
	indices := make(map[string]int)
	lowlinks := make(map[string]int)
	onStack := make(map[string]bool)
	var sccs [][]string

	var strongconnect func(string)
	strongconnect = func(v string) {
		indices[v] = index
		lowlinks[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		// Consider successors
		for _, w := range g.edges[v] {
			if _, ok := indices[w]; !ok {
				// Successor w has not yet been visited
				strongconnect(w)
				lowlinks[v] = min(lowlinks[v], lowlinks[w])
			} else if onStack[w] {
				// Successor w is in stack S and hence in the current SCC
				lowlinks[v] = min(lowlinks[v], indices[w])
			}
		}

		// If v is a root node, pop the stack and print an SCC
		if lowlinks[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	// Find SCCs for all nodes
	for _, node := range g.nodes {
		if _, ok := indices[node]; !ok {
			strongconnect(node)
		}
	}

	return sccs
}

// BuildCallGraph analyzes functions to build a call graph
func BuildCallGraph(funcs []*FuncSig, symbols map[string]*FuncSig, imports map[string]string) *CallGraph {
	graph := NewCallGraph()

	// Add all function nodes
	for _, f := range funcs {
		graph.AddNode(f.Name)
	}

	// Analyze each function body for calls
	for _, f := range funcs {
		refs := findReferences(f.Body)
		for _, ref := range refs {
			// Only add edge if reference is to a local function
			if _, isLocal := symbols[ref]; isLocal {
				// Check it's not an imported name
				if _, isImported := imports[ref]; !isImported {
					graph.AddEdge(f.Name, ref)
				}
			}
		}
	}

	return graph
}

// findReferences finds all identifier references in an expression
func findReferences(expr ast.Expr) []string {
	var refs []string

	switch ex := expr.(type) {
	case *ast.Identifier:
		refs = append(refs, ex.Name)

	case *ast.BinaryOp:
		refs = append(refs, findReferences(ex.Left)...)
		refs = append(refs, findReferences(ex.Right)...)

	case *ast.UnaryOp:
		refs = append(refs, findReferences(ex.Expr)...)

	case *ast.If:
		refs = append(refs, findReferences(ex.Condition)...)
		refs = append(refs, findReferences(ex.Then)...)
		refs = append(refs, findReferences(ex.Else)...)

	case *ast.Let:
		// Value might reference functions
		refs = append(refs, findReferences(ex.Value)...)
		// Body has ex.Name in scope, filter it out later if needed
		refs = append(refs, findReferences(ex.Body)...)

	case *ast.Lambda:
		// Lambda body might reference functions
		refs = append(refs, findReferences(ex.Body)...)

	case *ast.FuncCall:
		refs = append(refs, findReferences(ex.Func)...)
		for _, arg := range ex.Args {
			refs = append(refs, findReferences(arg)...)
		}

	case *ast.List:
		for _, elem := range ex.Elements {
			refs = append(refs, findReferences(elem)...)
		}

	case *ast.Record:
		for _, field := range ex.Fields {
			refs = append(refs, findReferences(field.Value)...)
		}

	case *ast.RecordAccess:
		refs = append(refs, findReferences(ex.Record)...)

	case *ast.Match:
		refs = append(refs, findReferences(ex.Expr)...)
		for _, c := range ex.Cases {
			refs = append(refs, findReferences(c.Body)...)
		}

	case *ast.Tuple:
		for _, elem := range ex.Elements {
			refs = append(refs, findReferences(elem)...)
		}

	case *ast.Block:
		// Blocks can contain function references in any expression
		for _, expr := range ex.Exprs {
			refs = append(refs, findReferences(expr)...)
		}
	}

	return refs
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

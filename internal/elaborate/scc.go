// Package elaborate provides SCC detection for mutual recursion
package elaborate

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/ast"
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

// findReferences finds all identifier references in an expression.
//
// This drives intra-module call-graph construction (BuildCallGraph), which
// feeds SCC ordering, which the core type checker relies on for decl order
// (typechecker_core.go CheckCoreProgram threads the env strictly forward). A
// module-local function is only put in scope before its callers when an edge
// exists; a missing edge lets Tarjan emit the callee AFTER the caller, so the
// caller fails with a FALSE "undefined variable" for a function that exists.
//
// Therefore this traversal MUST be EXHAUSTIVE over every expression position:
// a local reference hiding in any un-visited sub-expression silently drops its
// call-graph edge and reintroduces the #327 bug family ("resolution diverges
// by syntactic position"). When a new *ast.Expr variant is added, add its case
// here; DEBUG_STRICT surfaces omissions loudly instead of dropping edges.
func findReferences(expr ast.Expr) []string {
	var refs []string

	switch ex := expr.(type) {
	case nil:
		// Optional sub-expressions (e.g. an absent match guard) are nil.

	case *ast.Identifier:
		refs = append(refs, ex.Name)

	case *ast.Literal:
		// Leaf: no references.

	case *ast.Error:
		// Parse-error placeholder: no references.

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

	case *ast.LetRec:
		// Value might reference functions (including itself for recursion)
		refs = append(refs, findReferences(ex.Value)...)
		// Body has ex.Name in scope
		refs = append(refs, findReferences(ex.Body)...)

	case *ast.Lambda:
		// Lambda body might reference functions
		refs = append(refs, findReferences(ex.Body)...)

	case *ast.FuncLit:
		// FuncLit body might reference functions (same as Lambda)
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

	case *ast.Array:
		for _, elem := range ex.Elements {
			refs = append(refs, findReferences(elem)...)
		}

	case *ast.Record:
		for _, field := range ex.Fields {
			refs = append(refs, findReferences(field.Value)...)
		}

	case *ast.RecordAccess:
		refs = append(refs, findReferences(ex.Record)...)

	case *ast.RecordUpdate:
		// {base | f: e, ...} — the base AND every updated field's value are
		// expression positions. Omitting these was the #327 root cause: a local
		// function called only inside `{ s | f: localFn(...) }` produced no
		// call-graph edge and mis-ordered as "undefined variable".
		refs = append(refs, findReferences(ex.Base)...)
		for _, field := range ex.Fields {
			refs = append(refs, findReferences(field.Value)...)
		}

	case *ast.Match:
		refs = append(refs, findReferences(ex.Expr)...)
		for _, c := range ex.Cases {
			// Guard expressions are real expression positions too — a local
			// function called only in a `_ if localFn(...) =>` guard must still
			// produce an edge (same family as the record-update case).
			refs = append(refs, findReferences(c.Guard)...)
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

	default:
		// A new *ast.Expr variant was added without a case here. Dropping it
		// silently would reintroduce the #327 bug family for that position, so
		// fail loudly under DEBUG_STRICT rather than fabricate an empty edge set.
		if os.Getenv("DEBUG_STRICT") != "" {
			panic(fmt.Sprintf("findReferences: unhandled ast.Expr variant %T — "+
				"add a case (missing edges cause false 'undefined variable', see #327)", expr))
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

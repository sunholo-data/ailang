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

// BuildCallGraph analyzes functions AND module-level lets to build a call graph.
//
// M-MODULE-LET-FUNC-RESOLUTION (#366): module-level lets are first-class nodes of
// this graph, not a wrapping special case. Their value expressions are walked by
// the same exhaustive findReferences, so a let value can depend on a module func
// (either declaration order), a func can depend on a let, and let→let / func→func
// all get real ordering edges. Tarjan then emits every decl in dependency order,
// interleaving lets and funcs — which is exactly what the strict forward-env core
// type checker (typechecker_core.go CheckCoreProgram) needs to resolve a
// module-scope name identically in every DECL class, not just every func body.
//
// A name is "local" (edge-worthy) if it is a module func OR a module let, and is
// NOT an imported name (imports resolve via globalEnv, so they need no ordering).
func BuildCallGraph(funcs []*FuncSig, lets []*ModuleLet, symbols map[string]*FuncSig, imports map[string]string) *CallGraph {
	graph := NewCallGraph()

	// A local name is edge-worthy: module func OR module let. Imports are excluded
	// (they resolve via the global env and need no intra-module ordering edge).
	localNames := make(map[string]bool, len(symbols)+len(lets))
	for name := range symbols {
		localNames[name] = true
	}
	for _, l := range lets {
		localNames[l.Name] = true
	}

	addEdges := func(from string, body ast.Expr) {
		for _, ref := range findReferences(body) {
			if !localNames[ref] {
				continue
			}
			if _, isImported := imports[ref]; isImported {
				continue
			}
			graph.AddEdge(from, ref)
		}
	}

	// Add all function nodes, then all let nodes (nodes with no edges still emit).
	for _, f := range funcs {
		graph.AddNode(f.Name)
	}
	for _, l := range lets {
		graph.AddNode(l.Name)
	}

	// Analyze each function body AND each module-let value for local references.
	for _, f := range funcs {
		addEdges(f.Name, f.Body)
	}
	for _, l := range lets {
		addEdges(l.Name, l.Value)
	}

	return graph
}

// astPatternBinders mirrors elaboratePattern's syntactic binder classification.
// Constructors and record field labels are not variable bindings.
func astPatternBinders(pat ast.Pattern) []string {
	var names []string
	var walk func(ast.Pattern)
	walk = func(pat ast.Pattern) {
		switch p := pat.(type) {
		case *ast.Identifier:
			if p.Name != "_" && !isUpperIdent(p.Name) {
				names = append(names, p.Name)
			}
		case *ast.ConstructorPattern:
			for _, sub := range p.Patterns {
				walk(sub)
			}
		case *ast.ConsPattern:
			walk(p.Head)
			walk(p.Tail)
		case *ast.ListPattern:
			for _, sub := range p.Elements {
				walk(sub)
			}
			walk(p.Rest)
		case *ast.TuplePattern:
			for _, sub := range p.Elements {
				walk(sub)
			}
		case *ast.RecordPattern:
			for _, field := range p.Fields {
				walk(field.Pattern)
			}
		}
	}
	walk(pat)
	return names
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
	return findFreeReferences(expr, nil)
}

// withBound creates a child scope; sibling expressions keep their parent's set.
func withBound(bound map[string]bool, names ...string) map[string]bool {
	child := make(map[string]bool, len(bound)+len(names))
	for name := range bound {
		child[name] = true
	}
	for _, name := range names {
		if name != "" && name != "_" {
			child[name] = true
		}
	}
	return child
}

func paramNames(params []*ast.Param) []string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, p.Name)
	}
	return names
}

func findFreeReferences(expr ast.Expr, bound map[string]bool) []string {
	var refs []string

	switch ex := expr.(type) {
	case nil:
		// Optional sub-expressions (e.g. an absent match guard) are nil.

	case *ast.Identifier:
		if !bound[ex.Name] {
			refs = append(refs, ex.Name)
		}

	case *ast.Literal:
		// Leaf: no references.

	case *ast.Error:
		// Parse-error placeholder: no references.

	case *ast.BinaryOp:
		refs = append(refs, findFreeReferences(ex.Left, bound)...)
		refs = append(refs, findFreeReferences(ex.Right, bound)...)

	case *ast.UnaryOp:
		refs = append(refs, findFreeReferences(ex.Expr, bound)...)

	case *ast.If:
		refs = append(refs, findFreeReferences(ex.Condition, bound)...)
		refs = append(refs, findFreeReferences(ex.Then, bound)...)
		refs = append(refs, findFreeReferences(ex.Else, bound)...)

	case *ast.Let:
		refs = append(refs, findFreeReferences(ex.Value, bound)...)
		refs = append(refs, findFreeReferences(ex.Body, withBound(bound, ex.Name))...)

	case *ast.LetRec:
		// A bodyless module letrec must retain its self-edge for LetRec emission.
		scope := bound
		if ex.Body != nil {
			scope = withBound(bound, ex.Name)
		}
		refs = append(refs, findFreeReferences(ex.Value, scope)...)
		refs = append(refs, findFreeReferences(ex.Body, scope)...)

	case *ast.Lambda:
		refs = append(refs, findFreeReferences(ex.Body, withBound(bound, paramNames(ex.Params)...))...)

	case *ast.FuncLit:
		refs = append(refs, findFreeReferences(ex.Body, withBound(bound, paramNames(ex.Params)...))...)

	case *ast.FuncCall:
		refs = append(refs, findFreeReferences(ex.Func, bound)...)
		for _, arg := range ex.Args {
			refs = append(refs, findFreeReferences(arg, bound)...)
		}

	case *ast.List:
		for _, elem := range ex.Elements {
			refs = append(refs, findFreeReferences(elem, bound)...)
		}

	case *ast.Array:
		for _, elem := range ex.Elements {
			refs = append(refs, findFreeReferences(elem, bound)...)
		}

	case *ast.Record:
		for _, field := range ex.Fields {
			refs = append(refs, findFreeReferences(field.Value, bound)...)
		}

	case *ast.RecordAccess:
		refs = append(refs, findFreeReferences(ex.Record, bound)...)

	case *ast.RecordUpdate:
		// {base | f: e, ...} — the base AND every updated field's value are
		// expression positions. Omitting these was the #327 root cause: a local
		// function called only inside `{ s | f: localFn(...) }` produced no
		// call-graph edge and mis-ordered as "undefined variable".
		refs = append(refs, findFreeReferences(ex.Base, bound)...)
		for _, field := range ex.Fields {
			refs = append(refs, findFreeReferences(field.Value, bound)...)
		}

	case *ast.Match:
		refs = append(refs, findFreeReferences(ex.Expr, bound)...)
		for _, c := range ex.Cases {
			// Guard expressions are real expression positions too — a local
			// function called only in a `_ if localFn(...) =>` guard must still
			// produce an edge (same family as the record-update case).
			scope := withBound(bound, astPatternBinders(c.Pattern)...)
			refs = append(refs, findFreeReferences(c.Guard, scope)...)
			refs = append(refs, findFreeReferences(c.Body, scope)...)
		}

	case *ast.Tuple:
		for _, elem := range ex.Elements {
			refs = append(refs, findFreeReferences(elem, bound)...)
		}

	case *ast.Block:
		scope := bound
		for _, item := range ex.Exprs {
			refs = append(refs, findFreeReferences(item, scope)...)
			// Only statement bindings extend the scope of subsequent block items.
			switch binding := item.(type) {
			case *ast.Let:
				if binding.Body == nil {
					scope = withBound(scope, binding.Name)
				}
			case *ast.LetRec:
				if binding.Body == nil {
					scope = withBound(scope, binding.Name)
				}
			}
		}

	case *ast.ForallExpr:
		refs = append(refs, findFreeReferences(ex.Lo, bound)...)
		refs = append(refs, findFreeReferences(ex.Hi, bound)...)
		refs = append(refs, findFreeReferences(ex.Body, withBound(bound, ex.Var))...)
	case *ast.Send:
		refs = append(refs, findFreeReferences(ex.Channel, bound)...)
		refs = append(refs, findFreeReferences(ex.Value, bound)...)
	case *ast.Recv:
		refs = append(refs, findFreeReferences(ex.Channel, bound)...)
	case *ast.AssertStmt:
		refs = append(refs, findFreeReferences(ex.Condition, bound)...)
	case *ast.QuasiQuote:
		for _, interpolation := range ex.Interpolations {
			refs = append(refs, findFreeReferences(interpolation.Expr, bound)...)
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

package testing

import (
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// Helper to create a simple Core program for testing
func makeTestProgram(bindings ...core.RecBinding) *core.Program {
	return &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: bindings,
				Body:     &core.Lit{Value: 0},
			},
		},
	}
}

// Helper to create a RecBinding
func makeBinding(name string, body core.CoreExpr) core.RecBinding {
	return core.RecBinding{
		Name:  name,
		Value: body,
	}
}

// Helper to create a Var reference
func makeVar(name string) *core.Var {
	return &core.Var{Name: name}
}

// Helper to create an App (function call)
func makeApp(fn core.CoreExpr, args ...core.CoreExpr) *core.App {
	return &core.App{Func: fn, Args: args}
}

// Helper to create a Lambda with unique NodeID
func makeLambda(params []string, body core.CoreExpr) *core.Lambda {
	return &core.Lambda{
		CoreNode: core.CoreNode{NodeID: nextNodeID()},
		Params:   params,
		Body:     body,
	}
}

// Helper to create an If expression
func makeIf(cond, then, els core.CoreExpr) *core.If {
	return &core.If{Cond: cond, Then: then, Else: els}
}

// Helper to create a BinOp
func makeBinOp(op string, left, right core.CoreExpr) *core.BinOp {
	return &core.BinOp{Op: op, Left: left, Right: right}
}

func TestBuildCallGraph_SingleFunction(t *testing.T) {
	// Single self-recursive function: factorial
	// factorial(n) = if n <= 1 then 1 else n * factorial(n-1)
	factorialBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("<=", makeVar("n"), &core.Lit{Value: int64(1)}),
			&core.Lit{Value: int64(1)},
			makeBinOp("*",
				makeVar("n"),
				makeApp(makeVar("factorial"), makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)})),
			),
		),
	)

	prog := makeTestProgram(makeBinding("factorial", factorialBody))
	g := BuildCallGraph(prog)

	// Verify nodes
	if !g.Nodes["factorial"] {
		t.Error("Expected 'factorial' in nodes")
	}
	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}

	// Self-recursive calls are not included in edges (handled by SCC)
	if len(g.Edges["factorial"]) != 0 {
		t.Errorf("Expected no edges for self-recursive function, got %v", g.Edges["factorial"])
	}
}

func TestBuildCallGraph_Chain(t *testing.T) {
	// Chain: a -> b -> c
	// a() = b()
	// b() = c()
	// c() = 42
	aBody := makeLambda(nil, makeApp(makeVar("b")))
	bBody := makeLambda(nil, makeApp(makeVar("c")))
	cBody := makeLambda(nil, &core.Lit{Value: int64(42)})

	prog := makeTestProgram(
		makeBinding("a", aBody),
		makeBinding("b", bBody),
		makeBinding("c", cBody),
	)
	g := BuildCallGraph(prog)

	// Verify nodes
	if len(g.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(g.Nodes))
	}

	// Verify edges
	if len(g.Edges["a"]) != 1 || g.Edges["a"][0] != "b" {
		t.Errorf("Expected a -> [b], got %v", g.Edges["a"])
	}
	if len(g.Edges["b"]) != 1 || g.Edges["b"][0] != "c" {
		t.Errorf("Expected b -> [c], got %v", g.Edges["b"])
	}
	if len(g.Edges["c"]) != 0 {
		t.Errorf("Expected c -> [], got %v", g.Edges["c"])
	}
}

func TestBuildCallGraph_MutualRecursion(t *testing.T) {
	// Mutual recursion: isEven <-> isOdd
	// isEven(n) = if n == 0 then true else isOdd(n-1)
	// isOdd(n) = if n == 0 then false else isEven(n-1)
	isEvenBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("==", makeVar("n"), &core.Lit{Value: int64(0)}),
			&core.Lit{Value: true},
			makeApp(makeVar("isOdd"), makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)})),
		),
	)
	isOddBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("==", makeVar("n"), &core.Lit{Value: int64(0)}),
			&core.Lit{Value: false},
			makeApp(makeVar("isEven"), makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)})),
		),
	)

	prog := makeTestProgram(
		makeBinding("isEven", isEvenBody),
		makeBinding("isOdd", isOddBody),
	)
	g := BuildCallGraph(prog)

	// Verify edges
	if len(g.Edges["isEven"]) != 1 || g.Edges["isEven"][0] != "isOdd" {
		t.Errorf("Expected isEven -> [isOdd], got %v", g.Edges["isEven"])
	}
	if len(g.Edges["isOdd"]) != 1 || g.Edges["isOdd"][0] != "isEven" {
		t.Errorf("Expected isOdd -> [isEven], got %v", g.Edges["isOdd"])
	}
}

func TestBuildCallGraph_Diamond(t *testing.T) {
	// Diamond: a -> b, a -> c, b -> d, c -> d
	aBody := makeLambda(nil, makeBinOp("+", makeApp(makeVar("b")), makeApp(makeVar("c"))))
	bBody := makeLambda(nil, makeApp(makeVar("d")))
	cBody := makeLambda(nil, makeApp(makeVar("d")))
	dBody := makeLambda(nil, &core.Lit{Value: int64(1)})

	prog := makeTestProgram(
		makeBinding("a", aBody),
		makeBinding("b", bBody),
		makeBinding("c", cBody),
		makeBinding("d", dBody),
	)
	g := BuildCallGraph(prog)

	// Verify a has two edges
	aEdges := g.Edges["a"]
	sort.Strings(aEdges)
	if len(aEdges) != 2 {
		t.Errorf("Expected a -> [b, c], got %v", aEdges)
	}
}

func TestComputeSCCs_SingleFunction(t *testing.T) {
	// Single self-recursive function
	factorialBody := makeLambda([]string{"n"},
		makeApp(makeVar("factorial"), makeVar("n")),
	)
	prog := makeTestProgram(makeBinding("factorial", factorialBody))
	g := BuildCallGraph(prog)

	sccs := ComputeSCCs(g)

	if len(sccs) != 1 {
		t.Errorf("Expected 1 SCC, got %d: %v", len(sccs), sccs)
	}
	if len(sccs[0]) != 1 || sccs[0][0] != "factorial" {
		t.Errorf("Expected SCC [factorial], got %v", sccs[0])
	}
}

func TestComputeSCCs_Chain(t *testing.T) {
	// Chain: a -> b -> c (three singleton SCCs)
	aBody := makeLambda(nil, makeApp(makeVar("b")))
	bBody := makeLambda(nil, makeApp(makeVar("c")))
	cBody := makeLambda(nil, &core.Lit{Value: int64(42)})

	prog := makeTestProgram(
		makeBinding("a", aBody),
		makeBinding("b", bBody),
		makeBinding("c", cBody),
	)
	g := BuildCallGraph(prog)

	sccs := ComputeSCCs(g)

	if len(sccs) != 3 {
		t.Errorf("Expected 3 SCCs, got %d: %v", len(sccs), sccs)
	}

	// Each SCC should be a singleton
	for _, scc := range sccs {
		if len(scc) != 1 {
			t.Errorf("Expected singleton SCC, got %v", scc)
		}
	}
}

func TestComputeSCCs_MutualRecursion(t *testing.T) {
	// Mutual recursion: isEven <-> isOdd (one SCC with both)
	isEvenBody := makeLambda([]string{"n"}, makeApp(makeVar("isOdd"), makeVar("n")))
	isOddBody := makeLambda([]string{"n"}, makeApp(makeVar("isEven"), makeVar("n")))

	prog := makeTestProgram(
		makeBinding("isEven", isEvenBody),
		makeBinding("isOdd", isOddBody),
	)
	g := BuildCallGraph(prog)

	sccs := ComputeSCCs(g)

	if len(sccs) != 1 {
		t.Errorf("Expected 1 SCC, got %d: %v", len(sccs), sccs)
	}
	if len(sccs[0]) != 2 {
		t.Errorf("Expected SCC with 2 elements, got %v", sccs[0])
	}

	// Both functions should be in the same SCC
	sccSet := make(map[string]bool)
	for _, name := range sccs[0] {
		sccSet[name] = true
	}
	if !sccSet["isEven"] || !sccSet["isOdd"] {
		t.Errorf("Expected SCC to contain isEven and isOdd, got %v", sccs[0])
	}
}

func TestFindSCCContaining(t *testing.T) {
	// Setup: chain a -> b -> c + mutual d <-> e
	aBody := makeLambda(nil, makeApp(makeVar("b")))
	bBody := makeLambda(nil, makeApp(makeVar("c")))
	cBody := makeLambda(nil, &core.Lit{Value: int64(42)})
	dBody := makeLambda(nil, makeApp(makeVar("e")))
	eBody := makeLambda(nil, makeApp(makeVar("d")))

	prog := makeTestProgram(
		makeBinding("a", aBody),
		makeBinding("b", bBody),
		makeBinding("c", cBody),
		makeBinding("d", dBody),
		makeBinding("e", eBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	// Find SCC containing 'a' (singleton)
	sccA := FindSCCContaining(sccs, "a")
	if len(sccA) != 1 || sccA[0] != "a" {
		t.Errorf("Expected SCC [a], got %v", sccA)
	}

	// Find SCC containing 'd' (should include 'e')
	sccD := FindSCCContaining(sccs, "d")
	if len(sccD) != 2 {
		t.Errorf("Expected SCC with 2 elements for d, got %v", sccD)
	}

	// Non-existent function
	sccX := FindSCCContaining(sccs, "nonexistent")
	if sccX != nil {
		t.Errorf("Expected nil for nonexistent function, got %v", sccX)
	}
}

func TestGetDependencyClosure(t *testing.T) {
	// Setup: lcm -> gcd (lcm depends on gcd)
	// gcd(a, b) = if b == 0 then a else gcd(b, a % b)
	// lcm(a, b) = (a * b) / gcd(a, b)
	gcdBody := makeLambda([]string{"a", "b"},
		makeIf(
			makeBinOp("==", makeVar("b"), &core.Lit{Value: int64(0)}),
			makeVar("a"),
			makeApp(makeVar("gcd"), makeVar("b"), makeBinOp("%", makeVar("a"), makeVar("b"))),
		),
	)
	lcmBody := makeLambda([]string{"a", "b"},
		makeBinOp("/",
			makeBinOp("*", makeVar("a"), makeVar("b")),
			makeApp(makeVar("gcd"), makeVar("a"), makeVar("b")),
		),
	)

	prog := makeTestProgram(
		makeBinding("gcd", gcdBody),
		makeBinding("lcm", lcmBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	// Get closure for lcm (should include lcm and gcd)
	closure := GetDependencyClosure(g, sccs, "lcm")
	closureSet := make(map[string]bool)
	for _, name := range closure {
		closureSet[name] = true
	}

	if !closureSet["lcm"] {
		t.Error("Closure for lcm should include lcm")
	}
	if !closureSet["gcd"] {
		t.Error("Closure for lcm should include gcd")
	}
	if len(closure) != 2 {
		t.Errorf("Expected closure of size 2, got %d: %v", len(closure), closure)
	}

	// Get closure for gcd (should only include gcd)
	closureGcd := GetDependencyClosure(g, sccs, "gcd")
	if len(closureGcd) != 1 || closureGcd[0] != "gcd" {
		t.Errorf("Expected closure [gcd], got %v", closureGcd)
	}
}

func TestGetDependencyClosure_MutualRecursion(t *testing.T) {
	// isEven <-> isOdd (both in same SCC)
	isEvenBody := makeLambda([]string{"n"}, makeApp(makeVar("isOdd"), makeVar("n")))
	isOddBody := makeLambda([]string{"n"}, makeApp(makeVar("isEven"), makeVar("n")))

	prog := makeTestProgram(
		makeBinding("isEven", isEvenBody),
		makeBinding("isOdd", isOddBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	// Get closure for isEven (should include both)
	closure := GetDependencyClosure(g, sccs, "isEven")
	closureSet := make(map[string]bool)
	for _, name := range closure {
		closureSet[name] = true
	}

	if !closureSet["isEven"] || !closureSet["isOdd"] {
		t.Errorf("Expected closure to include isEven and isOdd, got %v", closure)
	}
}

func TestGetDependencyClosure_Chain(t *testing.T) {
	// a -> b -> c (closure of a includes all three)
	aBody := makeLambda(nil, makeApp(makeVar("b")))
	bBody := makeLambda(nil, makeApp(makeVar("c")))
	cBody := makeLambda(nil, &core.Lit{Value: int64(42)})

	prog := makeTestProgram(
		makeBinding("a", aBody),
		makeBinding("b", bBody),
		makeBinding("c", cBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	// Get closure for a (should include a, b, c)
	closure := GetDependencyClosure(g, sccs, "a")
	closureSet := make(map[string]bool)
	for _, name := range closure {
		closureSet[name] = true
	}

	if len(closure) != 3 {
		t.Errorf("Expected closure of size 3, got %d: %v", len(closure), closure)
	}
	if !closureSet["a"] || !closureSet["b"] || !closureSet["c"] {
		t.Errorf("Expected closure to include a, b, c, got %v", closure)
	}
}

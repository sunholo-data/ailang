package types

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// M-IFC-RECURSIVE-LAMBDA-FOLD: a lambda that reaches itself through recursion
// must give a finite summary, and folding must keep every demand.

func parseFoldModule(t *testing.T, src string) *ast.File {
	t.Helper()
	p := parser.New(lexer.New(src, "fold.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	return file
}

// ifcWork runs the IFC pass and returns its diagnostics and the work it used.
// A pass that does not converge must fail here, not hang the suite.
func ifcWork(t *testing.T, src string) ([]*TypeCheckError, int) {
	t.Helper()
	file := parseFoldModule(t, src)
	c := newIFCChecker(file)
	done := make(chan []*TypeCheckError, 1)
	go func() { done <- guardIFCAnalysis(func() []*TypeCheckError { return c.run(file) }) }()
	select {
	case errs := <-done:
		for _, e := range errs {
			if e.Kind == IFCAnalysisLimitError {
				t.Fatalf("summary did not converge: %s", e.Message)
			}
		}
		return errs, ifcAnalysisWork - c.analysisRemaining
	case <-time.After(30 * time.Second):
		t.Fatal("IFC pass still running after 30s: the fixed point does not converge")
		return nil, 0
	}
}

// The specstore 0.3.0 shape (headingsIn in audit.ail): a function recursing
// through a lambda passed to an imported combinator. Before the fold every
// fixed-point round built a deeper λ and the pass ran to the work budget
// (1.4 GB, 60 s on the 239-line module).
const recursiveLambdaSrc = `module test/fold
func report(s: string{not client}) -> () = ()
func textOf(b: string) -> string<client> = b
func headingsIn(blocks: [string]) -> [string<client>] =
  flatMap(\b. if b == "section" then headingsIn(split(b, ",")) else [textOf(b)], blocks)
`

func TestIFCFold_RecursiveLambdaConverges(t *testing.T) {
	errs, work := ifcWork(t, recursiveLambdaSrc)
	if len(errs) != 0 {
		t.Fatalf("clean module reported %v", errs)
	}
	if work > 2000 {
		t.Fatalf("fixed point used %d work units; a converging summary needs a few hundred", work)
	}
}

// A demand inside the recursive lambda must survive the fold: client data
// reaching the lambda through the recursion is still refused at the sink.
func TestIFCFold_KeepsDemandOfRecursiveLambda(t *testing.T) {
	src := `module test/fold
func report(s: string{not client}) -> () = ()
func secret() -> string<client> = "k"
func walk(xs: [string]) -> [()] =
  flatMap(\b. if b == "" then [report(b)] else walk(split(b, ",")), xs)
func use() -> [()] = walk([secret()])
`
	errs, work := ifcWork(t, src)
	if work > 2000 {
		t.Fatalf("fixed point used %d work units", work)
	}
	if len(errs) != 1 || errs[0].Kind != SinkRefinementError || !strings.Contains(errs[0].Message, "report") {
		t.Fatalf("want one {not client} violation at report, got %v", errs)
	}
}

// foldSelf merges a nested instance of the same literal into the outer one
// (union of deferred applications) and resolveSelf restores it on read.
func TestIFCFold_MergesNestedInstance(t *testing.T) {
	sink := single(&ifcFn{kind: fnDeclared, name: "sink", params: []ifcDemand{{param: "s", forbid: []string{"client"}}}})
	other := single(&ifcFn{kind: fnDeclared, name: "other", params: []ifcDemand{{param: "s", forbid: []string{"pii"}}}})
	inner := &ifcFn{kind: fnLambda, name: "lambda", owner: "λ1",
		deferred: []ifcDeferred{{target: other, args: []ifcVal{{label: LabelVar("ρλ1#0")}}}}}
	outer := &ifcFn{kind: fnLambda, name: "lambda", owner: "λ1",
		deferred: []ifcDeferred{{target: sink, args: []ifcVal{{label: LabelVar("ρλ1#0")}}}},
		result:   joinCallable(single(inner), single(&ifcFn{kind: fnOpaque}))}
	foldSelf(outer)

	if len(outer.deferred) != 2 {
		t.Fatalf("nested instance's deferred application lost: %d entries", len(outer.deferred))
	}
	if strings.Contains(callableKey(outer.result), "L:λ1") || !strings.Contains(callableKey(outer.result), "S:λ1") {
		t.Fatalf("nested instance not replaced by a self-reference: %s", callableKey(outer.result))
	}
	resolved := resolveSelf(outer.result, outer)
	found := false
	for _, a := range resolved.alts {
		found = found || a == outer
		if a.kind == fnSelf {
			t.Fatal("self-reference survived resolution")
		}
	}
	if !found {
		t.Fatalf("self-reference did not resolve to the instance: %s", callableKey(resolved))
	}
}

package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// M-TRACE-LABEL-AWARE: the IFC pass reports every call site at which a
// labelled value is passed, captured or produced, so the evaluator can redact
// the trace there. These tests pin which sites are reported.

// traceSitesFor parses src and returns the reported sites keyed by callee name
// (each fixture calls a given name once).
func traceSitesFor(t *testing.T, src string) map[string]string {
	t.Helper()
	p := parser.New(lexer.New(src, "p.ail"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	errs, sites := CheckModuleIFCWithTraceSites(prog.File)
	if len(errs) > 0 {
		t.Fatalf("fixture must be IFC-clean: %v", errs[0])
	}
	byCallee := map[string]string{}
	walkCalls(prog.File, func(call *ast.FuncCall) {
		if label, ok := sites[call.Pos]; ok {
			byCallee[calleeName(call.Func)] = label
		}
	})
	return byCallee
}

// walkCalls visits every FuncCall reachable from a module's function bodies.
// It covers only the shapes these fixtures use.
func walkCalls(file *ast.File, visit func(*ast.FuncCall)) {
	var walk func(e ast.Expr)
	walk = func(e ast.Expr) {
		switch x := e.(type) {
		case *ast.FuncCall:
			visit(x)
			walk(x.Func)
			for _, a := range x.Args {
				walk(a)
			}
		case *ast.Let:
			walk(x.Value)
			walk(x.Body)
		case *ast.Block:
			for _, s := range x.Exprs {
				walk(s)
			}
		case *ast.Lambda:
			walk(x.Body)
		case *ast.List:
			for _, el := range x.Elements {
				walk(el)
			}
		case *ast.If:
			walk(x.Condition)
			walk(x.Then)
			walk(x.Else)
		case *ast.BinaryOp:
			walk(x.Left)
			walk(x.Right)
		}
	}
	for _, fn := range file.Funcs {
		walk(fn.Body)
	}
}

func TestIFCTraceSitesReportLabelledCalls(t *testing.T) {
	src := `module p
func classify(raw: string) -> string<secret> ! {} = raw
func firstChars(s: string<secret>, n: int) -> int ! {} = n
func plain(n: int) -> int ! {} = n
func forward(x: string) -> string ! {} = x
export func main() -> int ! {} {
  let token = classify("CANARY");
  let copied = forward(token);
  let f = \u. copied;
  let g = f(0);
  firstChars(token, plain(3))
}
`
	got := traceSitesFor(t, src)
	want := map[string]string{
		"classify":   "secret", // produces a labelled value from a literal
		"forward":    "secret", // passes one
		"f":          "secret", // applies a closure that captured one
		"firstChars": "secret",
	}
	for callee, label := range want {
		if got[callee] != label {
			t.Errorf("site %s: label %q, want %q (all sites: %v)", callee, got[callee], label, got)
		}
	}
	// The control: a call that neither receives nor produces a labelled value
	// must not be redacted, or the trace loses its value for no reason.
	if label, ok := got["plain"]; ok {
		t.Errorf("plain(3) carries no label but was reported as %q", label)
	}
}

func TestIFCTraceSitesInsideLabelledParameterBody(t *testing.T) {
	// Inside a function whose parameter is declared labelled, every call that
	// receives that parameter is a site, whoever calls the function.
	src := `module p
func inner(s: string) -> string ! {} = s
export func outer(s: string<secret>) -> string ! {} = inner(s)
`
	got := traceSitesFor(t, src)
	if got["inner"] != "secret" {
		t.Fatalf("inner(s) inside a <secret> parameter body: %q, want secret (sites: %v)", got["inner"], got)
	}
}

func TestIFCTraceSitesJoinLabelNames(t *testing.T) {
	src := `module p
func both(a: string<secret>, b: string<pii>) -> string ! {} = a
export func run(a: string<secret>, b: string<pii>) -> string ! {} = both(a, b)
`
	got := traceSitesFor(t, src)
	if got["both"] != "pii,secret" {
		t.Fatalf("both(a, b): %q, want the sorted join \"pii,secret\"", got["both"])
	}
}

// The site recorder runs in a defer inside callVal; while an analysis-limit
// panic unwinds through it, it must not panic itself, or the fail-closed
// refusal becomes a compiler crash.
func TestIFCTraceSitesSurviveAnalysisLimit(t *testing.T) {
	src := `module p
func inner(s: string) -> string ! {} = s
export func outer(s: string<secret>) -> string ! {} = inner(inner(inner(s)))
`
	p := parser.New(lexer.New(src, "p.ail"))
	prog := p.Parse()
	for budget := 0; budget < 40; budget++ {
		sites := map[ast.Pos]map[string]bool{}
		errs := guardIFCAnalysis(func() []*TypeCheckError {
			c := newIFCChecker(prog.File)
			c.traceSites = sites
			c.analysisRemaining = budget
			return c.run(prog.File)
		})
		if len(errs) == 0 {
			return // the budget sufficed; every smaller one refused cleanly
		}
		if errs[0].Kind != IFCAnalysisLimitError {
			t.Fatalf("budget %d: %v", budget, errs[0])
		}
	}
	t.Fatal("no budget below 40 let the fixture through; the test proves nothing")
}

func TestIFCTraceSitesNoLabelsNoSites(t *testing.T) {
	src := `module p
func inner(s: string) -> string ! {} = s
export func outer(s: string) -> string ! {} = inner(s)
`
	p := parser.New(lexer.New(src, "p.ail"))
	prog := p.Parse()
	if _, sites := CheckModuleIFCWithTraceSites(prog.File); len(sites) != 0 {
		t.Fatalf("a module with no labels reported %d sites: %v", len(sites), sites)
	}
}

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestRegisterStandardEvalSession_IsWiredIntoDispatch pins the CALL SITE, not the
// helper.
//
// TestRegisterStandardEvalSession_ResolvesOpenRouterSessionID calls
// registerStandardEvalSession directly, so it proves the writer works and proves
// nothing about whether anything invokes it. Measured: deleting the call from
// runSingleBenchmark leaves that test — and the whole cmd/ailang suite — green.
// That is the same defect this change exists to fix (M-MISSION-LOOP-UNIFIED-TELEMETRY
// M1 shipped a resolver whose writer was never wired, and passed 5/5 of its own
// acceptance criteria), so it is worth a gate of its own.
//
// runSingleBenchmark needs a live model call and cannot be exercised in a unit
// test, so the reachable pin is structural: assert the call appears in that
// function's body.
func TestRegisterStandardEvalSession_IsWiredIntoDispatch(t *testing.T) {
	const (
		file    = "eval_benchmark.go"
		caller  = "runSingleBenchmark"
		callee  = "registerStandardEvalSession"
		control = "SetCorrelation" // known-positive: must be found by the same walk
	)

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}

	var body *ast.BlockStmt
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == caller && fn.Body != nil {
			body = fn.Body
			break
		}
	}
	if body == nil {
		t.Fatalf("instrument failure: %s not found in %s", caller, file)
	}

	calls := map[string]int{}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			calls[fn.Name]++
		case *ast.SelectorExpr:
			calls[fn.Sel.Name]++
		}
		return true
	})

	// Known-positive control: an empty walk must fail loudly rather than pass.
	if calls[control] == 0 {
		t.Fatalf("instrument failure: control %q not called in %s — the walk found nothing", control, caller)
	}

	if calls[callee] == 0 {
		t.Fatalf("%s is never called from %s: the OpenRouter session_id is registered nowhere on the "+
			"standard-mode path, so every Broadcast span it emits is unjoinable (control %q found %d times)",
			callee, caller, control, calls[control])
	}
}

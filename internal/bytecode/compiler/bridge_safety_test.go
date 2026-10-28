package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// M-VM-COVERAGE G4 safety net and G3: see bridge_safety.go and
// compileTopLevelRef.

var dirADT = stmt.TypeDecl{Name: "Dir", Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{{Tag: "U"}, {Tag: "D"}}}}

// evalOnlyFunc is a typed function the lowering could not compile (it runs
// on the evaluator through the bridge).
func evalOnlyFunc(name string, params []stmt.Param, ret stmt.ResolvedType) stmt.FuncDecl {
	return stmt.FuncDecl{Name: name, Params: params, ReturnType: ret, LowerError: "test: evaluator-only"}
}

func callerOf(name, callee string, params []stmt.Param, args ...stmt.Expr) stmt.FuncDecl {
	return stmt.FuncDecl{Name: name, Params: params, ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt}, Return: call(callee, args...)}
}

func dirParam(name string) stmt.Param {
	return stmt.Param{Name: name, Type: stmt.NamedType{Name: "Dir", Pointer: true}}
}

func compileProg(t *testing.T, prog *stmt.Program) *bytecode.BytecodeImage {
	t.Helper()
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return img
}

func TestBridgeSafety_ADTArgumentMakesCallerEvalOnly(t *testing.T) {
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{dirADT},
		FuncDecls: []stmt.FuncDecl{
			evalOnlyFunc("step", []stmt.Param{dirParam("d"), intParam("n")}, stmt.PrimitiveType{Kind: stmt.PrimInt}),
			callerOf("main", "step", []stmt.Param{dirParam("d")}, v("d"), lit(5)),
		},
	}
	p := entryProto(t, compileProg(t, prog), "main")
	if !p.EvalOnly || !strings.Contains(p.EvalReason, "ADT Dir") {
		t.Fatalf("EvalOnly=%v reason=%q: a native caller passing an ADT to an evaluator-only function must fall back", p.EvalOnly, p.EvalReason)
	}
}

func TestBridgeSafety_PlainSignatureStaysNative(t *testing.T) {
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		evalOnlyFunc("plain", []stmt.Param{intParam("n"), {Name: "s", Type: stmt.PrimitiveType{Kind: stmt.PrimString}}},
			stmt.NamedType{Name: "Result", Pointer: true}),
		callerOf("main", "plain", nil, lit(1), stmt.LitString{Value: "x"}),
	}}
	if p := entryProto(t, compileProg(t, prog), "main"); p.EvalOnly {
		t.Fatalf("an int/string -> Result edge is bridgeable; main went evaluator-only: %s", p.EvalReason)
	}
}

// Making a function evaluator-only exposes ITS callers in turn.
func TestBridgeSafety_Transitive(t *testing.T) {
	prog := &stmt.Program{
		TypeDecls: []stmt.TypeDecl{dirADT},
		FuncDecls: []stmt.FuncDecl{
			evalOnlyFunc("leaf", []stmt.Param{stmt.Param{Name: "f", Type: stmt.FuncType{
				Params: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimInt}}, Return: stmt.PrimitiveType{Kind: stmt.PrimInt}}}},
				stmt.PrimitiveType{Kind: stmt.PrimInt}),
			callerOf("mid", "leaf", []stmt.Param{dirParam("d")}, v("f0")),
			callerOf("main", "mid", []stmt.Param{dirParam("d")}, v("d")),
		},
	}
	// mid references f0, which is unbound: give mid a real closure argument.
	prog.FuncDecls[1].Body = []stmt.Stmt{stmt.VarDecl{Name: "f0", Value: stmt.Lambda{Params: []stmt.Param{intParam("x")}, Return: v("x")}}}
	img := compileProg(t, prog)
	if mid := entryProto(t, img, "mid"); !mid.EvalOnly || !strings.Contains(mid.EvalReason, "function value") {
		t.Fatalf("mid: EvalOnly=%v reason=%q, want evaluator-only for passing a closure", mid.EvalOnly, mid.EvalReason)
	}
	if main := entryProto(t, img, "main"); !main.EvalOnly || !strings.Contains(main.EvalReason, "mid") {
		t.Fatalf("main: EvalOnly=%v reason=%q, want evaluator-only through mid", main.EvalOnly, main.EvalReason)
	}
}

// A stub whose lowering failed has placeholder types; it must count as unknown.
func TestBridgeSafety_StubSignatureIsUnknown(t *testing.T) {
	stub := evalOnlyFunc("broken", []stmt.Param{intParam("n")}, stmt.PrimitiveType{Kind: stmt.PrimUnit})
	stub.StubSignature = true
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{stub, callerOf("main", "broken", nil, lit(1))}}
	if p := entryProto(t, compileProg(t, prog), "main"); !p.EvalOnly || !strings.Contains(p.EvalReason, "unknown") {
		t.Fatalf("EvalOnly=%v reason=%q, want evaluator-only for an unknown signature", p.EvalOnly, p.EvalReason)
	}
}

// G3: a reference to a top-level value binding yields its value, not the
// nullary function that computes it; calling a value binding that holds a
// function calls that function.
func TestCompile_ValueDeclReference(t *testing.T) {
	prog := &stmt.Program{FuncDecls: []stmt.FuncDecl{
		{Name: "base", Value: true, Return: bin(stmt.OpAdd, lit(40), lit(1))},
		{Name: "double", Value: true, Return: stmt.Lambda{Params: []stmt.Param{intParam("x")}, Return: bin(stmt.OpMul, v("x"), lit(2))}},
		{Name: "main", Exported: true, Return: bin(stmt.OpAdd, v("base"), call("double", v("base")))},
	}}
	got := runProgram(t, prog, "main", nil)
	if got.Tag != bytecode.TagInt || got.Int != 41+82 {
		t.Fatalf("main() = %v, want 123 (a Closure here is G3)", got)
	}
}

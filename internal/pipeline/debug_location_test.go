package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

func debugWrapperCall(name string, line int, args ...core.CoreExpr) *core.App {
	pos := ast.Pos{File: "api/handlers.ail", Line: line, Column: 3}
	node := core.CoreNode{CoreSpan: pos, OrigSpan: pos}
	return &core.App{
		CoreNode: node,
		Func:     &core.VarGlobal{CoreNode: node, Ref: core.GlobalRef{Module: "std/debug", Name: name}},
		Args:     args,
	}
}

func lastLitString(t *testing.T, app *core.App) string {
	t.Helper()
	lit, ok := app.Args[len(app.Args)-1].(*core.Lit)
	if !ok || lit.Kind != core.StringLit {
		t.Fatalf("last arg is not a string literal: %#v", app.Args[len(app.Args)-1])
	}
	return lit.Value.(string)
}

// A call to std/debug.log(msg) becomes $builtin._debug_log(msg, "file:line")
// with the CALL SITE's position — this is the "location auto-injected by
// the compiler" that std/debug.ail's "unknown" placeholder stood in for.
func TestInjectDebugLocation_LogRewritesToBuiltinWithCallSite(t *testing.T) {
	prog := &core.Program{Decls: []core.CoreExpr{debugWrapperCall("log", 42, strLit("hi"))}}
	out := (&DebugLocationInjector{}).Inject(prog)

	app, ok := out.Decls[0].(*core.App)
	if !ok {
		t.Fatalf("decl is %T, want *core.App", out.Decls[0])
	}
	fn, ok := app.Func.(*core.VarGlobal)
	if !ok || fn.Ref.Module != "$builtin" || fn.Ref.Name != "_debug_log" {
		t.Fatalf("callee = %#v, want $builtin._debug_log", app.Func)
	}
	if len(app.Args) != 2 {
		t.Fatalf("args = %d, want 2 (msg, location)", len(app.Args))
	}
	if got := lastLitString(t, app); got != "api/handlers.ail:42" {
		t.Errorf("location = %q, want api/handlers.ail:42", got)
	}
	// The rewritten App keeps the call's own position for diagnostics.
	if app.OriginalSpan().Line != 42 {
		t.Errorf("rewritten App lost its span: %+v", app.OriginalSpan())
	}
}

func TestInjectDebugLocation_CheckRewritesWithThreeArgs(t *testing.T) {
	cond := &core.Var{Name: "ok"}
	prog := &core.Program{Decls: []core.CoreExpr{debugWrapperCall("check", 7, cond, strLit("inv"))}}
	out := (&DebugLocationInjector{}).Inject(prog)
	app := out.Decls[0].(*core.App)
	fn := app.Func.(*core.VarGlobal)
	if fn.Ref.Name != "_debug_check" || len(app.Args) != 3 {
		t.Fatalf("got %s/%d args, want _debug_check/3", fn.Ref.Name, len(app.Args))
	}
	if app.Args[0] != cond {
		t.Errorf("original args must be preserved in order")
	}
	if got := lastLitString(t, app); got != "api/handlers.ail:7" {
		t.Errorf("location = %q", got)
	}
}

// The pass recurses: a call nested inside a let body is still rewritten.
func TestInjectDebugLocation_RecursesIntoNestedExprs(t *testing.T) {
	inner := debugWrapperCall("log", 9, strLit("nested"))
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "_", Value: inner, Body: &core.Var{Name: "x"}}}}
	out := (&DebugLocationInjector{}).Inject(prog)
	let := out.Decls[0].(*core.Let)
	app := let.Value.(*core.App)
	if app.Func.(*core.VarGlobal).Ref.Name != "_debug_log" {
		t.Fatalf("nested call not rewritten: %#v", let.Value)
	}
	if got := lastLitString(t, app); got != "api/handlers.ail:9" {
		t.Errorf("location = %q", got)
	}
}

// Things the pass must leave alone: a wrapper used as a first-class value
// (not an App), a call with the wrong arity (already rewritten, or a local
// shadow), a call to some other module's `log`, and the builtin itself.
func TestInjectDebugLocation_LeavesNonWrapperCallsAlone(t *testing.T) {
	pos := ast.Pos{File: "f.ail", Line: 1}
	node := core.CoreNode{CoreSpan: pos, OrigSpan: pos}
	firstClass := &core.VarGlobal{CoreNode: node, Ref: core.GlobalRef{Module: "std/debug", Name: "log"}}
	otherModule := &core.App{CoreNode: node,
		Func: &core.VarGlobal{CoreNode: node, Ref: core.GlobalRef{Module: "my/logger", Name: "log"}},
		Args: []core.CoreExpr{strLit("x")}}
	alreadyBuiltin := &core.App{CoreNode: node,
		Func: &core.VarGlobal{CoreNode: node, Ref: core.GlobalRef{Module: "$builtin", Name: "_debug_log"}},
		Args: []core.CoreExpr{strLit("x"), strLit("std/debug.ail:26")}}
	wrongArity := debugWrapperCall("log", 3, strLit("a"), strLit("b"))

	prog := &core.Program{Decls: []core.CoreExpr{firstClass, otherModule, alreadyBuiltin, wrongArity}}
	out := (&DebugLocationInjector{}).Inject(prog)

	if vg, ok := out.Decls[0].(*core.VarGlobal); !ok || vg.Ref.Name != "log" {
		t.Errorf("first-class wrapper reference was rewritten: %#v", out.Decls[0])
	}
	if app := out.Decls[1].(*core.App); app.Func.(*core.VarGlobal).Ref.Module != "my/logger" || len(app.Args) != 1 {
		t.Errorf("other module's log was rewritten: %#v", app)
	}
	if app := out.Decls[2].(*core.App); len(app.Args) != 2 || lastLitString(t, app) != "std/debug.ail:26" {
		t.Errorf("builtin call was double-injected: %#v", app)
	}
	if app := out.Decls[3].(*core.App); len(app.Args) != 2 || app.Func.(*core.VarGlobal).Ref.Name != "log" {
		t.Errorf("wrong-arity call was rewritten: %#v", app)
	}
}

// A node with no usable position still gets the builtin (so erasure and
// codegen see one shape) but the location falls back to "unknown" rather
// than "%!s:0".
func TestInjectDebugLocation_NoPositionFallsBackToUnknown(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/debug", Name: "log"}},
		Args: []core.CoreExpr{strLit("m")},
	}
	out := (&DebugLocationInjector{}).Inject(&core.Program{Decls: []core.CoreExpr{app}})
	if got := lastLitString(t, out.Decls[0].(*core.App)); got != "unknown" {
		t.Errorf("location = %q, want unknown", got)
	}
}

func TestDisplayPath_RelativeToCwdWhenUnderIt(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	under := filepath.Join(cwd, "api", "orders.ail")
	if got := displayPath(under); got != "api/orders.ail" {
		t.Errorf("under cwd: got %q, want api/orders.ail", got)
	}
	outside := filepath.Join(filepath.Dir(cwd), "elsewhere.ail")
	if got := displayPath(outside); got != filepath.ToSlash(outside) {
		t.Errorf("outside cwd must stay absolute: got %q", got)
	}
	if got := displayPath(filepath.Join("rel", "x.ail")); got != "rel/x.ail" {
		t.Errorf("relative stays relative (slash-normalised): got %q", got)
	}
}

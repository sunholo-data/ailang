package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// Helper to create a Core node with a specific ID and type in CTI.
func litInt(id uint64, val int64) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.IntLit,
		Value:    val,
	}
}

func litBool(id uint64, val bool) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.BoolLit,
		Value:    val,
	}
}

func litStr(id uint64, val string) *core.Lit {
	return &core.Lit{
		CoreNode: core.CoreNode{NodeID: id},
		Kind:     core.StringLit,
		Value:    val,
	}
}

func coreVar(id uint64, name string) *core.Var {
	return &core.Var{
		CoreNode: core.CoreNode{NodeID: id},
		Name:     name,
	}
}

func makeCTI(entries map[uint64]types.Type) types.CoreTypeInfo {
	cti := make(types.CoreTypeInfo)
	for k, v := range entries {
		cti[k] = v
	}
	return cti
}

// --- Program lowering tests ---

func TestLowerProgram_Simple(t *testing.T) {
	// A simple program with one function: func add(x, y) = x + y
	lamID := uint64(100)
	cti := makeCTI(map[uint64]types.Type{
		lamID: &types.TFunc2{
			Params: []types.Type{types.TInt, types.TInt},
			Return: types.TInt,
		},
	})

	coreProg := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				CoreNode: core.CoreNode{NodeID: 1},
				Name:     "add",
				Value: &core.Lambda{
					CoreNode: core.CoreNode{NodeID: lamID},
					Params:   []string{"x", "y"},
					Body: &core.BinOp{
						CoreNode: core.CoreNode{NodeID: 101},
						Op:       "+",
						Left:     coreVar(102, "x"),
						Right:    coreVar(103, "y"),
					},
				},
				Body: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.UnitLit},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"add": {Name: "add", IsExport: true},
		},
	}

	prog, err := LowerProgram(coreProg, cti, nil, "main")
	if err != nil {
		t.Fatalf("LowerProgram failed: %v", err)
	}

	if prog.Package != "main" {
		t.Errorf("expected package main, got %s", prog.Package)
	}
	if len(prog.FuncDecls) != 1 {
		t.Fatalf("expected 1 func, got %d", len(prog.FuncDecls))
	}

	fd := prog.FuncDecls[0]
	if fd.Name != "add" {
		t.Errorf("expected func name add, got %s", fd.Name)
	}
	if !fd.Exported {
		t.Error("expected exported")
	}
	if len(fd.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fd.Params))
	}
	if fd.Params[0].Type.GoString() != "int64" {
		t.Errorf("expected param type int64, got %s", fd.Params[0].Type.GoString())
	}
	if fd.ReturnType.GoString() != "int64" {
		t.Errorf("expected return type int64, got %s", fd.ReturnType.GoString())
	}
}

// TestLower_PreservesTmpBindings — M4_LOWER_TMP_SCOPE regression test.
//
// Reproduces the "$tmpN / user-var unbound" bug observed in docparse/main.ail:
// when a Let's VALUE is an If (or Match) whose branches contain their own
// Let-chains, the inner bindings were silently dropped by lowerLetExpr. The
// fix teaches flattenValue to hoist If/Match branches into IfStmt/SwitchStmt
// with a temp variable, preserving the inner bindings as proper statements.
//
// Shape of the failing input:
//
//	let combinedBlocks =
//	  if cond then
//	    let inner = 42 in inner
//	  else
//	    0
//	in combinedBlocks
//
// Before the fix, the VarDecl for `inner` was dropped and `combinedBlocks`
// would reference an undeclared variable via an IfExpr whose Then collapsed
// to just `inner`. After the fix, the resulting statement sequence must
// bind `inner` before it is used.
func TestLower_PreservesTmpBindings(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{
		1:  types.TInt, // outer Let type
		2:  types.TBool,
		3:  types.TInt, // inner Let type
		4:  types.TInt,
		5:  types.TInt,
		6:  types.TInt,
		10: types.TInt,
	})

	// Core:
	//   Let combinedBlocks = If(cond, Let(inner = 42 in inner), 0)
	//       in combinedBlocks
	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "inner",
		Value:    litInt(4, 42),
		Body:     coreVar(5, "inner"),
	}
	ifExpr := &core.If{
		CoreNode: core.CoreNode{NodeID: 10},
		Cond:     litBool(2, true),
		Then:     innerLet,
		Else:     litInt(6, 0),
	}
	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "combinedBlocks",
		Value:    ifExpr,
		Body:     coreVar(11, "combinedBlocks"),
	}

	stmts, ret := FlattenBlock(outerLet, cti)

	// Collect every variable name that is declared or assigned anywhere
	// in the emitted statement sequence (including inside IfStmt branches).
	declared := map[string]bool{}
	var collectDecls func(ss []stmt.Stmt)
	collectDecls = func(ss []stmt.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case stmt.VarDecl:
				declared[s.Name] = true
			case stmt.AssignStmt:
				declared[s.Name] = true
			case stmt.IfStmt:
				collectDecls(s.Then)
				collectDecls(s.Else)
			case stmt.SwitchStmt:
				for _, c := range s.Cases {
					collectDecls(c.Body)
				}
				collectDecls(s.Default)
			}
		}
	}
	collectDecls(stmts)

	// Collect every variable name REFERENCED in the emitted statements/return.
	referenced := map[string]bool{}
	var collectRefsExpr func(e stmt.Expr)
	var collectRefsStmts func(ss []stmt.Stmt)
	collectRefsExpr = func(e stmt.Expr) {
		switch e := e.(type) {
		case stmt.VarRef:
			referenced[e.Name] = true
		case stmt.BinOp:
			collectRefsExpr(e.Left)
			collectRefsExpr(e.Right)
		case stmt.UnOp:
			collectRefsExpr(e.Operand)
		case stmt.IfExpr:
			collectRefsExpr(e.Cond)
			collectRefsExpr(e.Then)
			collectRefsExpr(e.Else)
		case stmt.Call:
			collectRefsExpr(e.Func)
			for _, a := range e.Args {
				collectRefsExpr(a)
			}
		}
	}
	collectRefsStmts = func(ss []stmt.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case stmt.VarDecl:
				collectRefsExpr(s.Value)
			case stmt.AssignStmt:
				collectRefsExpr(s.Value)
			case stmt.ExprStmt:
				collectRefsExpr(s.Value)
			case stmt.ReturnStmt:
				collectRefsExpr(s.Value)
			case stmt.IfStmt:
				collectRefsExpr(s.Cond)
				collectRefsStmts(s.Then)
				collectRefsStmts(s.Else)
			case stmt.SwitchStmt:
				collectRefsExpr(s.Scrutinee)
				for _, c := range s.Cases {
					collectRefsStmts(c.Body)
				}
				collectRefsStmts(s.Default)
			}
		}
	}
	collectRefsStmts(stmts)
	if ret != nil {
		collectRefsExpr(ret)
	}

	// The critical assertion: every referenced variable must have a
	// corresponding declaration somewhere. Before the fix, `inner` is
	// referenced but never declared because lowerLetExpr dropped it.
	for name := range referenced {
		if !declared[name] {
			t.Errorf("variable %q referenced but never declared; "+
				"stmts=%#v ret=%#v", name, stmts, ret)
		}
	}

	// Targeted check: `inner` must be declared. Without this guardrail
	// the loop above could be bypassed if the lowering accidentally
	// rewrote the reference to something else.
	if !declared["inner"] {
		t.Errorf("expected `inner` to be declared in lowered statements; "+
			"got stmts=%#v ret=%#v", stmts, ret)
	}
}

package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

func coreLet(id uint64, name string, value, body core.CoreExpr) *core.Let {
	return &core.Let{CoreNode: core.CoreNode{NodeID: id}, Name: name, Value: value, Body: body}
}

// let r = 1 in let s = (let r = 2 in r) in r — the #2375 shape.
func blockShadowCore() *core.Let {
	return coreLet(1, "r", litInt(2, 1),
		coreLet(3, "s", coreLet(4, "r", litInt(5, 2), coreVar(6, "r")),
			coreVar(7, "r")))
}

// CheckUniqueLocals flags a declaration that reuses a name still in scope,
// and allows the same name in sibling branches (match lowering copies arms).
func TestCheckUniqueLocals(t *testing.T) {
	decl := func(n string) stmt.Stmt { return stmt.VarDecl{Name: n, Value: stmt.LitInt{Value: 1}} }
	shadow := stmt.FuncDecl{Name: "f", Body: []stmt.Stmt{decl("r"), decl("s"), decl("r")}}
	if err := CheckUniqueLocals(shadow); err == nil {
		t.Error("a second r in the same list was not flagged")
	}
	lam := stmt.FuncDecl{Name: "g", Params: []stmt.Param{{Name: "r"}},
		Return: stmt.Lambda{Params: []stmt.Param{{Name: "r"}}, Return: stmt.VarRef{Name: "r"}}}
	if err := CheckUniqueLocals(lam); err == nil {
		t.Error("a lambda parameter reusing an enclosing name was not flagged")
	}
	siblings := stmt.FuncDecl{Name: "h", Body: []stmt.Stmt{
		stmt.IfStmt{Cond: stmt.LitBool{Value: true}, Then: []stmt.Stmt{decl("k")}, Else: []stmt.Stmt{decl("k")}},
		decl("$tmp1"), decl("$tmp1"), decl("_"), decl("_"),
	}}
	if err := CheckUniqueLocals(siblings); err != nil {
		t.Errorf("sibling branches / compiler names flagged: %v", err)
	}
	if err := CheckUniqueLocals(stmt.FuncDecl{Name: "u", Body: func() []stmt.Stmt {
		s, _ := FlattenBlock(core.UniqueLocals(blockShadowCore()), makeCTI(nil))
		return s
	}()}); err != nil {
		t.Errorf("the #2375 shape after uniqueLocals: %v", err)
	}
	if err := CheckUniqueLocals(stmt.FuncDecl{Name: "u0", Body: func() []stmt.Stmt {
		s, _ := FlattenBlock(blockShadowCore(), makeCTI(nil))
		return s
	}()}); err == nil {
		t.Error("the #2375 shape without uniqueLocals was not flagged")
	}
}

// A let left nested in a value (the elaborator no longer floats a source
// binder, #2388) keeps its source line when flattenValue hoists it.
func TestFlattenValue_NestedLetKeepsLine(t *testing.T) {
	inner := coreLet(4, "r", litInt(5, 2), coreVar(6, "r"))
	inner.OrigSpan = ast.Pos{Line: 7}
	stmts, _ := FlattenBlock(coreLet(3, "s", inner, coreVar(7, "s")), makeCTI(nil))
	for _, s := range stmts {
		if d, ok := s.(stmt.VarDecl); ok && d.Name == "r" {
			if d.Line != 7 {
				t.Fatalf("hoisted `let r` has line %d, want 7", d.Line)
			}
			return
		}
	}
	t.Fatalf("no VarDecl r in %v", stmts)
}

// The lowered statements of the #2375 shape no longer declare r twice.
func TestFlattenBlock_ShadowedLetGetsOwnName(t *testing.T) {
	stmts, ret := FlattenBlock(core.UniqueLocals(blockShadowCore()), makeCTI(nil))
	names := map[string]int{}
	for _, s := range stmts {
		if d, ok := s.(stmt.VarDecl); ok {
			names[d.Name]++
		}
	}
	if names["r"] != 1 {
		t.Fatalf("r declared %d times in %v", names["r"], stmts)
	}
	if ref, ok := ret.(stmt.VarRef); !ok || ref.Name != "r" {
		t.Fatalf("return = %#v, want a read of the outer r", ret)
	}
}

// The lowering makes names unique itself, so it is sound on any Core, not only
// on the pipeline's (which core.UniqueProgramLocals already renamed): a
// hand-built program with the #2375 shadowing lowers to a function that keeps
// the invariant.
func TestLowerProgram_UniquifiesShadowingCore(t *testing.T) {
	decl := coreLet(100, "f", &core.Lambda{CoreNode: core.CoreNode{NodeID: 101}, Params: []string{"q"}, Body: blockShadowCore()}, nil)
	prog, err := LowerProgram(&core.Program{Decls: []core.CoreExpr{decl}}, makeCTI(nil), nil, "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.FuncDecls) != 1 {
		t.Fatalf("got %d functions", len(prog.FuncDecls))
	}
	if err := CheckUniqueLocals(prog.FuncDecls[0]); err != nil {
		t.Fatalf("lowered without unique names: %v", err)
	}
}

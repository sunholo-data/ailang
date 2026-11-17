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

func TestUniqueLocals_RenamesShadowingBinderOnly(t *testing.T) {
	in := blockShadowCore()
	before := in.String()
	out, ok := uniqueLocals(in).(*core.Let)
	if !ok {
		t.Fatalf("got %T", out)
	}
	if in.String() != before {
		t.Fatalf("input mutated: %s -> %s", before, in.String())
	}
	if out.Name != "r" {
		t.Errorf("outer binder renamed to %q", out.Name)
	}
	s := out.Body.(*core.Let)
	inner := s.Value.(*core.Let)
	if inner.Name == "r" || inner.Body.(*core.Var).Name != inner.Name {
		t.Errorf("inner binder %q, its read %q: want a fresh name read by its body", inner.Name, inner.Body.(*core.Var).Name)
	}
	if inner.ID() != 4 || inner.Body.ID() != 6 {
		t.Errorf("renamed nodes lost their NodeIDs (%d, %d)", inner.ID(), inner.Body.ID())
	}
	if got := s.Body.(*core.Var).Name; got != "r" {
		t.Errorf("read after the block = %q, want the outer r", got)
	}
}

// Code without shadowing comes back as the same pointer: lowering it is
// unchanged.
func TestUniqueLocals_NoShadowIsIdentity(t *testing.T) {
	in := coreLet(1, "a", litInt(2, 1), coreLet(3, "b", coreVar(4, "a"), coreVar(5, "b")))
	if out := uniqueLocals(in); out != core.CoreExpr(in) {
		t.Fatalf("unshadowed input was copied: %s", out)
	}
}

// `_` is never read, and match lowering skips a `_` pattern by name: it is
// never renamed, however often it is bound.
func TestUniqueLocals_UnderscoreKeepsName(t *testing.T) {
	in := coreLet(1, "_", litInt(2, 1), coreLet(3, "_", litInt(4, 2), litInt(5, 3)))
	if out := uniqueLocals(in); out != core.CoreExpr(in) {
		t.Fatalf("`_` binders renamed: %s", out)
	}
}

// A local named like a free (top-level) reference is renamed, so the free
// reference cannot resolve to it in the flat name space.
func TestUniqueLocals_LocalNamedLikeFreeReference(t *testing.T) {
	// let s = (let f = 2 in f) in f(s)
	in := coreLet(1, "s", coreLet(2, "f", litInt(3, 2), coreVar(4, "f")),
		&core.App{CoreNode: core.CoreNode{NodeID: 5}, Func: coreVar(6, "f"), Args: []core.CoreExpr{coreVar(7, "s")}})
	out := uniqueLocals(in).(*core.Let)
	if local := out.Value.(*core.Let).Name; local == "f" {
		t.Fatalf("local f kept the name of the free reference f")
	}
	if fn := out.Body.(*core.App).Func.(*core.Var).Name; fn != "f" {
		t.Fatalf("free reference renamed to %q", fn)
	}
}

// Lambda parameters, pattern variables and letrec members are binders too.
func TestUniqueLocals_AllBinderKinds(t *testing.T) {
	r := func(id uint64) core.CoreExpr { return coreVar(id, "r") }
	lam := &core.Lambda{CoreNode: core.CoreNode{NodeID: 10}, Params: []string{"r"}, Body: r(11)}
	match := &core.Match{CoreNode: core.CoreNode{NodeID: 20}, Scrutinee: r(21), Arms: []core.MatchArm{
		{Pattern: &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "r"}}}, Body: r(22)},
	}}
	rec := &core.LetRec{CoreNode: core.CoreNode{NodeID: 30}, Bindings: []core.RecBinding{{Name: "r", Value: r(31)}}, Body: r(32)}
	in := coreLet(1, "r", litInt(2, 1), coreLet(3, "a", lam, coreLet(4, "b", match, coreLet(5, "c", rec, r(6)))))

	out := uniqueLocals(in).(*core.Let)
	a := out.Body.(*core.Let)
	b := a.Body.(*core.Let)
	c := b.Body.(*core.Let)
	l := a.Value.(*core.Lambda)
	if l.Params[0] == "r" || l.Body.(*core.Var).Name != l.Params[0] {
		t.Errorf("lambda param %q read as %q", l.Params[0], l.Body.(*core.Var).Name)
	}
	m := b.Value.(*core.Match)
	if m.Scrutinee.(*core.Var).Name != "r" {
		t.Errorf("scrutinee %q must read the outer r", m.Scrutinee.(*core.Var).Name)
	}
	pv := m.Arms[0].Pattern.(*core.ConstructorPattern).Args[0].(*core.VarPattern).Name
	if pv == "r" || m.Arms[0].Body.(*core.Var).Name != pv {
		t.Errorf("pattern variable %q read as %q", pv, m.Arms[0].Body.(*core.Var).Name)
	}
	lr := c.Value.(*core.LetRec)
	if n := lr.Bindings[0].Name; n == "r" || lr.Bindings[0].Value.(*core.Var).Name != n || lr.Body.(*core.Var).Name != n {
		t.Errorf("letrec member %q read as %q / %q", n, lr.Bindings[0].Value.(*core.Var).Name, lr.Body.(*core.Var).Name)
	}
	if got := c.Body.(*core.Var).Name; got != "r" {
		t.Errorf("final read = %q, want the outer r", got)
	}
	seen := map[string]bool{"r": true}
	for _, n := range []string{l.Params[0], pv, lr.Bindings[0].Name} {
		if seen[n] {
			t.Errorf("binder name %q used twice", n)
		}
		seen[n] = true
	}
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
		s, _ := FlattenBlock(uniqueLocals(blockShadowCore()), makeCTI(nil))
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
	stmts, ret := FlattenBlock(uniqueLocals(blockShadowCore()), makeCTI(nil))
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

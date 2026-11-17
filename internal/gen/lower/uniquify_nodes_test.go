package lower

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// shadowIn builds `let r = 1 in let s = WRAP(let r = 2 in r) in r`: the inner
// let sits inside the node kind wrap builds. After uniqueLocals the inner r
// must have its own name, read by its own body, and the final read must stay
// the outer r. Every Core node kind that can hold a sub-expression is covered,
// so a kind the pass skipped would keep the inner name.
func shadowIn(wrap func(inner core.CoreExpr) core.CoreExpr) core.CoreExpr {
	inner := coreLet(10, "r", litInt(11, 2), coreVar(12, "r"))
	return coreLet(1, "r", litInt(2, 1), coreLet(3, "s", wrap(inner), coreVar(4, "r")))
}

func node(id uint64) core.CoreNode { return core.CoreNode{NodeID: id} }

func TestUniqueLocals_EveryNodeKind(t *testing.T) {
	x := func(id uint64) core.CoreExpr { return coreVar(id, "x") }
	cases := map[string]func(core.CoreExpr) core.CoreExpr{
		"app": func(e core.CoreExpr) core.CoreExpr {
			return &core.App{CoreNode: node(20), Func: x(21), Args: []core.CoreExpr{e}}
		},
		"app-func": func(e core.CoreExpr) core.CoreExpr {
			return &core.App{CoreNode: node(20), Func: e, Args: []core.CoreExpr{x(21)}}
		},
		"if-cond": func(e core.CoreExpr) core.CoreExpr {
			return &core.If{CoreNode: node(20), Cond: e, Then: x(21), Else: x(22)}
		},
		"if-else": func(e core.CoreExpr) core.CoreExpr {
			return &core.If{CoreNode: node(20), Cond: x(21), Then: x(22), Else: e}
		},
		"binop": func(e core.CoreExpr) core.CoreExpr {
			return &core.BinOp{CoreNode: node(20), Op: "+", Left: x(21), Right: e}
		},
		"unop": func(e core.CoreExpr) core.CoreExpr { return &core.UnOp{CoreNode: node(20), Op: "-", Operand: e} },
		"intrinsic": func(e core.CoreExpr) core.CoreExpr {
			return &core.Intrinsic{CoreNode: node(20), Op: core.OpAdd, Args: []core.CoreExpr{e, x(21)}}
		},
		"record": func(e core.CoreExpr) core.CoreExpr {
			return &core.Record{CoreNode: node(20), Fields: map[string]core.CoreExpr{"a": x(21), "b": e}}
		},
		"recordaccess": func(e core.CoreExpr) core.CoreExpr {
			return &core.RecordAccess{CoreNode: node(20), Record: e, Field: "a"}
		},
		"recordupdate": func(e core.CoreExpr) core.CoreExpr {
			return &core.RecordUpdate{CoreNode: node(20), Base: x(21), Updates: map[string]core.CoreExpr{"a": e}}
		},
		"recordupdate-base": func(e core.CoreExpr) core.CoreExpr {
			return &core.RecordUpdate{CoreNode: node(20), Base: e, Updates: map[string]core.CoreExpr{"a": x(21)}}
		},
		"list": func(e core.CoreExpr) core.CoreExpr {
			return &core.List{CoreNode: node(20), Elements: []core.CoreExpr{x(21), e}}
		},
		"array": func(e core.CoreExpr) core.CoreExpr {
			return &core.Array{CoreNode: node(20), Elements: []core.CoreExpr{e}}
		},
		"tuple": func(e core.CoreExpr) core.CoreExpr {
			return &core.Tuple{CoreNode: node(20), Elements: []core.CoreExpr{e, x(21)}}
		},
		"lambda": func(e core.CoreExpr) core.CoreExpr {
			return &core.Lambda{CoreNode: node(20), Params: []string{"q"}, Body: e}
		},
		"dictabs": func(e core.CoreExpr) core.CoreExpr {
			return &core.DictAbs{CoreNode: node(20), Params: []core.DictParam{{Name: "dict_Num_a"}}, Body: e}
		},
		"dictapp": func(e core.CoreExpr) core.CoreExpr {
			return &core.DictApp{CoreNode: node(20), Dict: x(21), Method: "add", Args: []core.CoreExpr{e}}
		},
		"forall": func(e core.CoreExpr) core.CoreExpr {
			return &core.Forall{CoreNode: node(20), Var: "i", Lo: litInt(21, 0), Hi: litInt(22, 3), Body: e}
		},
		"letrec-value": func(e core.CoreExpr) core.CoreExpr {
			return &core.LetRec{CoreNode: node(20), Bindings: []core.RecBinding{{Name: "g", Value: e}}, Body: x(21)}
		},
		"match-scrutinee": func(e core.CoreExpr) core.CoreExpr {
			return &core.Match{CoreNode: node(20), Scrutinee: e, Arms: []core.MatchArm{{Pattern: &core.WildcardPattern{}, Body: x(21)}}}
		},
		"match-guard": func(e core.CoreExpr) core.CoreExpr {
			return &core.Match{CoreNode: node(20), Scrutinee: x(21), Arms: []core.MatchArm{{Pattern: &core.WildcardPattern{}, Guard: e, Body: x(22)}}}
		},
	}
	for name, wrap := range cases {
		t.Run(name, func(t *testing.T) {
			in := shadowIn(wrap)
			before := in.String()
			out := uniqueLocals(in)
			if in.String() != before {
				t.Fatalf("input mutated")
			}
			s := out.String()
			if strings.Count(s, "r$s1") < 2 {
				t.Fatalf("inner r not renamed (binder and read) in %s", s)
			}
			if !strings.HasSuffix(s, " in r") {
				t.Fatalf("final read is not the outer r: %s", s)
			}
		})
	}
}

// Patterns of every kind rename the variables they bind, and the arm reads
// the renamed one.
func TestUniqueLocals_EveryPatternKind(t *testing.T) {
	vp := func() core.CorePattern { return &core.VarPattern{Name: "r"} }
	tail := core.CorePattern(&core.VarPattern{Name: "r"})
	pats := map[string]core.CorePattern{
		"var":         vp(),
		"constructor": &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{vp()}},
		"tuple":       &core.TuplePattern{Elements: []core.CorePattern{&core.WildcardPattern{}, vp()}},
		"list":        &core.ListPattern{Elements: []core.CorePattern{vp()}},
		"list-tail":   &core.ListPattern{Elements: []core.CorePattern{&core.LitPattern{Value: int64(1)}}, Tail: &tail},
		"record":      &core.RecordPattern{Fields: map[string]core.CorePattern{"a": &core.WildcardPattern{}, "b": vp()}},
	}
	for name, pat := range pats {
		t.Run(name, func(t *testing.T) {
			m := &core.Match{CoreNode: node(20), Scrutinee: coreVar(21, "r"), Arms: []core.MatchArm{{Pattern: pat, Body: coreVar(22, "r")}}}
			in := coreLet(1, "r", litInt(2, 1), coreLet(3, "s", m, coreVar(4, "r")))
			out := uniqueLocals(in).(*core.Let).Body.(*core.Let).Value.(*core.Match)
			read := out.Arms[0].Body.(*core.Var).Name
			if read == "r" || !patternBinds(out.Arms[0].Pattern, read) {
				t.Fatalf("arm reads %q; pattern %s must bind it", read, out.Arms[0].Pattern)
			}
			if sc := out.Scrutinee.(*core.Var).Name; sc != "r" {
				t.Fatalf("the scrutinee reads %q, want the outer r", sc)
			}
		})
	}
}

func patternBinds(p core.CorePattern, name string) bool {
	switch p := p.(type) {
	case *core.VarPattern:
		return p.Name == name
	case *core.ConstructorPattern:
		return anyPatternBinds(p.Args, name)
	case *core.TuplePattern:
		return anyPatternBinds(p.Elements, name)
	case *core.ListPattern:
		return anyPatternBinds(p.Elements, name) || (p.Tail != nil && patternBinds(*p.Tail, name))
	case *core.RecordPattern:
		for _, f := range p.Fields {
			if patternBinds(f, name) {
				return true
			}
		}
	}
	return false
}

func anyPatternBinds(ps []core.CorePattern, name string) bool {
	for _, p := range ps {
		if patternBinds(p, name) {
			return true
		}
	}
	return false
}

// Guards are in the arm's scope: they read the arm's pattern variables, or
// the outer binding when the arm binds another name.
func TestUniqueLocals_MatchGuards(t *testing.T) {
	gt := func(id uint64, name string) core.CoreExpr {
		return &core.BinOp{CoreNode: node(id), Op: ">", Left: coreVar(id+1, name), Right: litInt(id+2, 5)}
	}
	some := func(n string) core.CorePattern {
		return &core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: n}}}
	}
	// \x. \o. match o { Some(x) if x > 5 => x, Some(y) if x > 5 => y, _ => 0 }
	m := &core.Match{CoreNode: node(30), Scrutinee: coreVar(31, "o"), Arms: []core.MatchArm{
		{Pattern: some("x"), Guard: gt(40, "x"), Body: coreVar(43, "x")},
		{Pattern: some("y"), Guard: gt(50, "x"), Body: coreVar(53, "y")},
		{Pattern: &core.WildcardPattern{}, Body: litInt(60, 0)},
	}}
	lam := &core.Lambda{CoreNode: node(1), Params: []string{"x"}, Body: &core.Lambda{CoreNode: node(2), Params: []string{"o"}, Body: m}}
	out := uniqueLocals(lam).(*core.Lambda).Body.(*core.Lambda).Body.(*core.Match)
	pv := out.Arms[0].Pattern.(*core.ConstructorPattern).Args[0].(*core.VarPattern).Name
	if pv == "x" {
		t.Fatal("pattern x (shadowing the parameter) kept its name")
	}
	if g := out.Arms[0].Guard.(*core.BinOp).Left.(*core.Var).Name; g != pv {
		t.Fatalf("guard of arm 1 reads %q, want the pattern variable %q", g, pv)
	}
	if g := out.Arms[1].Guard.(*core.BinOp).Left.(*core.Var).Name; g != "x" {
		t.Fatalf("guard of arm 2 reads %q, want the parameter x", g)
	}

	// A match nested in a guard binds the name again: its arm reads its own.
	nested := &core.Match{CoreNode: node(70), Scrutinee: coreVar(71, "o"), Arms: []core.MatchArm{
		{Pattern: some("x"), Body: gt(72, "x")},
	}}
	m2 := &core.Match{CoreNode: node(80), Scrutinee: coreVar(81, "o"), Arms: []core.MatchArm{
		{Pattern: some("x"), Guard: nested, Body: coreVar(82, "x")},
	}}
	lam2 := &core.Lambda{CoreNode: node(3), Params: []string{"x", "o"}, Body: m2}
	out2 := uniqueLocals(lam2).(*core.Lambda).Body.(*core.Match)
	outerPV := out2.Arms[0].Pattern.(*core.ConstructorPattern).Args[0].(*core.VarPattern).Name
	inner := out2.Arms[0].Guard.(*core.Match)
	innerPV := inner.Arms[0].Pattern.(*core.ConstructorPattern).Args[0].(*core.VarPattern).Name
	if outerPV == "x" || innerPV == "x" || innerPV == outerPV {
		t.Fatalf("names: param x, arm %q, guard's arm %q: want three distinct", outerPV, innerPV)
	}
	if r := inner.Arms[0].Body.(*core.BinOp).Left.(*core.Var).Name; r != innerPV {
		t.Fatalf("nested arm reads %q, want %q", r, innerPV)
	}
	if r := out2.Arms[0].Body.(*core.Var).Name; r != outerPV {
		t.Fatalf("arm body reads %q, want %q", r, outerPV)
	}
}

// Record field values are expressions like any other: a binder inside one
// is renamed and its reads follow; the field NAMES never change.
func TestUniqueLocals_RecordFieldsVisited(t *testing.T) {
	inner := coreLet(10, "r", litInt(11, 2), coreVar(12, "r"))
	rec := &core.Record{CoreNode: node(20), Fields: map[string]core.CoreExpr{"r": inner, "q": coreVar(21, "r")}}
	out := uniqueLocals(coreLet(1, "r", litInt(2, 1), rec)).(*core.Let).Body.(*core.Record)
	if _, ok := out.Fields["r"]; !ok {
		t.Fatalf("field name r changed: %v", out.Fields)
	}
	l := out.Fields["r"].(*core.Let)
	if l.Name == "r" || l.Body.(*core.Var).Name != l.Name {
		t.Fatalf("binder inside field r: %s", l)
	}
	if q := out.Fields["q"].(*core.Var).Name; q != "r" {
		t.Fatalf("field q reads %q, want the outer r", q)
	}
}

// Unknown node and pattern types panic (lowerTopLevelDeclSafe turns that into
// an evaluator-only function, so a skipped subtree cannot keep a stale name);
// nil passes through. The shims satisfy the sealed interfaces by embedding them.
type exprShim struct{ core.CoreExpr }

type patternShim struct{ core.CorePattern }

func TestUniqueLocals_UnknownTypesPanic(t *testing.T) {
	if uniqueLocals(nil) != nil {
		t.Fatal("nil in, non-nil out")
	}
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: no panic", name)
			}
		}()
		f()
	}
	mustPanic("expression", func() { uniqueLocals(exprShim{}) })
	mustPanic("pattern", func() {
		m := &core.Match{CoreNode: node(1), Scrutinee: coreVar(2, "o"), Arms: []core.MatchArm{{Pattern: patternShim{}, Body: litInt(3, 0)}}}
		uniqueLocals(m)
	})
}

// CheckUniqueLocals reaches a lambda wherever it sits: in every statement
// kind and every expression kind. Each case hides `\r. r` there, inside a
// function whose parameter is r, so the checker must flag it.
func TestCheckUniqueLocals_ReachesEveryPosition(t *testing.T) {
	lam := stmt.Lambda{Params: []stmt.Param{{Name: "r"}}, Return: stmt.VarRef{Name: "r"}}
	one := stmt.LitInt{Value: 1}
	exprs := map[string]stmt.Expr{
		"binop":        stmt.BinOp{Left: one, Right: lam},
		"unop":         stmt.UnOp{Operand: lam},
		"call":         stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{lam}},
		"call-func":    stmt.Call{Func: lam, Args: []stmt.Expr{one}},
		"field":        stmt.FieldAccess{Record: lam},
		"record":       stmt.RecordLit{Fields: []stmt.FieldInit{{Name: "a", Value: lam}}},
		"recordupdate": stmt.RecordUpdate{Base: one, Fields: []stmt.FieldInit{{Name: "a", Value: lam}}},
		"list":         stmt.ListLit{Elems: []stmt.Expr{lam}},
		"array":        stmt.ArrayLit{Elems: []stmt.Expr{lam}},
		"tuple":        stmt.TupleLit{Elems: []stmt.Expr{one, lam}},
		"cons":         stmt.Cons{Head: lam, Tail: one},
		"adt":          stmt.ADTConstructor{TypeName: "T", Tag: "C", Args: []stmt.Expr{lam}},
		"typeassert":   stmt.TypeAssert{Value: lam},
		"ifexpr":       stmt.IfExpr{Cond: one, Then: one, Else: lam},
		"builtin":      stmt.BuiltinCall{Name: "_x", Args: []stmt.Expr{lam}},
		"tageq":        stmt.ADTTagEq{Value: lam},
		"nested":       stmt.Lambda{Params: []stmt.Param{{Name: "q"}}, Body: []stmt.Stmt{stmt.ExprStmt{Value: lam}}},
	}
	for name, e := range exprs {
		stmts := map[string]stmt.Stmt{
			"vardecl": stmt.VarDecl{Name: "v", Value: e},
			"assign":  stmt.AssignStmt{Name: "v", Value: e},
			"return":  stmt.ReturnStmt{Value: e},
			"expr":    stmt.ExprStmt{Value: e},
			"fail":    stmt.MatchFail{Scrutinee: e},
			"if":      stmt.IfStmt{Cond: e},
			"switch": stmt.SwitchStmt{Scrutinee: e,
				Cases:   []stmt.SwitchCase{{Tag: "C", Body: []stmt.Stmt{stmt.ExprStmt{Value: one}}}},
				Default: []stmt.Stmt{stmt.ExprStmt{Value: one}}},
			"case-body": stmt.SwitchStmt{Scrutinee: one, Cases: []stmt.SwitchCase{{Tag: "C", Body: []stmt.Stmt{stmt.ExprStmt{Value: e}}}}},
			"default":   stmt.SwitchStmt{Scrutinee: one, Default: []stmt.Stmt{stmt.ExprStmt{Value: e}}},
			"then":      stmt.IfStmt{Cond: one, Then: []stmt.Stmt{stmt.ExprStmt{Value: e}}},
		}
		for sname, s := range stmts {
			fd := stmt.FuncDecl{Name: name + "/" + sname, Params: []stmt.Param{{Name: "r"}}, Body: []stmt.Stmt{s}}
			if err := CheckUniqueLocals(fd); err == nil {
				t.Errorf("%s in %s: lambda parameter r reusing the parameter r was not flagged", name, sname)
			}
		}
		if err := CheckUniqueLocals(stmt.FuncDecl{Name: name + "/result", Params: []stmt.Param{{Name: "r"}}, Return: e}); err == nil {
			t.Errorf("%s as the function's result was not flagged", name)
		}
	}
	// Switch-case bindings are declarations in the case's scope.
	sw := stmt.FuncDecl{Name: "sw", Params: []stmt.Param{{Name: "r"}}, Body: []stmt.Stmt{
		stmt.SwitchStmt{Scrutinee: one, Cases: []stmt.SwitchCase{{Tag: "C", Bindings: []stmt.Binding{{Name: "r"}}}}},
	}}
	if err := CheckUniqueLocals(sw); err == nil {
		t.Error("a switch binding reusing the parameter r was not flagged")
	}
}

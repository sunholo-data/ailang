package core

import (
	"strings"
	"testing"
)

func litInt(id uint64, v int64) *Lit {
	return &Lit{CoreNode: CoreNode{NodeID: id}, Kind: IntLit, Value: v}
}

func coreVar(id uint64, name string) *Var {
	return &Var{CoreNode: CoreNode{NodeID: id}, Name: name}
}

func coreLet(id uint64, name string, value, body CoreExpr) *Let {
	return &Let{CoreNode: CoreNode{NodeID: id}, Name: name, Value: value, Body: body}
}

// let r = 1 in let s = (let r = 2 in r) in r — the #2375 shape.
func blockShadowCore() *Let {
	return coreLet(1, "r", litInt(2, 1),
		coreLet(3, "s", coreLet(4, "r", litInt(5, 2), coreVar(6, "r")),
			coreVar(7, "r")))
}

func TestUniqueLocals_RenamesShadowingBinderOnly(t *testing.T) {
	in := blockShadowCore()
	before := in.String()
	out, ok := UniqueLocals(in).(*Let)
	if !ok {
		t.Fatalf("got %T", out)
	}
	if in.String() != before {
		t.Fatalf("input mutated: %s -> %s", before, in.String())
	}
	if out.Name != "r" {
		t.Errorf("outer binder renamed to %q", out.Name)
	}
	s := out.Body.(*Let)
	inner := s.Value.(*Let)
	if inner.Name == "r" || inner.Body.(*Var).Name != inner.Name {
		t.Errorf("inner binder %q, its read %q: want a fresh name read by its body", inner.Name, inner.Body.(*Var).Name)
	}
	if inner.ID() != 4 || inner.Body.ID() != 6 {
		t.Errorf("renamed nodes lost their NodeIDs (%d, %d)", inner.ID(), inner.Body.ID())
	}
	if got := s.Body.(*Var).Name; got != "r" {
		t.Errorf("read after the block = %q, want the outer r", got)
	}
}

// Code without shadowing comes back as the same pointer: lowering it is
// unchanged.
func TestUniqueLocals_NoShadowIsIdentity(t *testing.T) {
	in := coreLet(1, "a", litInt(2, 1), coreLet(3, "b", coreVar(4, "a"), coreVar(5, "b")))
	if out := UniqueLocals(in); out != CoreExpr(in) {
		t.Fatalf("unshadowed input was copied: %s", out)
	}
}

// `_` is never read, and match lowering skips a `_` pattern by name: it is
// never renamed, however often it is bound.
func TestUniqueLocals_UnderscoreKeepsName(t *testing.T) {
	in := coreLet(1, "_", litInt(2, 1), coreLet(3, "_", litInt(4, 2), litInt(5, 3)))
	if out := UniqueLocals(in); out != CoreExpr(in) {
		t.Fatalf("`_` binders renamed: %s", out)
	}
}

// A local named like a free (top-level) reference is renamed, so the free
// reference cannot resolve to it in the flat name space.
func TestUniqueLocals_LocalNamedLikeFreeReference(t *testing.T) {
	// let s = (let f = 2 in f) in f(s)
	in := coreLet(1, "s", coreLet(2, "f", litInt(3, 2), coreVar(4, "f")),
		&App{CoreNode: CoreNode{NodeID: 5}, Func: coreVar(6, "f"), Args: []CoreExpr{coreVar(7, "s")}})
	out := UniqueLocals(in).(*Let)
	if local := out.Value.(*Let).Name; local == "f" {
		t.Fatalf("local f kept the name of the free reference f")
	}
	if fn := out.Body.(*App).Func.(*Var).Name; fn != "f" {
		t.Fatalf("free reference renamed to %q", fn)
	}
}

// Lambda parameters, pattern variables and letrec members are binders too.
func TestUniqueLocals_AllBinderKinds(t *testing.T) {
	r := func(id uint64) CoreExpr { return coreVar(id, "r") }
	lam := &Lambda{CoreNode: CoreNode{NodeID: 10}, Params: []string{"r"}, Body: r(11)}
	match := &Match{CoreNode: CoreNode{NodeID: 20}, Scrutinee: r(21), Arms: []MatchArm{
		{Pattern: &ConstructorPattern{Name: "Some", Args: []CorePattern{&VarPattern{Name: "r"}}}, Body: r(22)},
	}}
	rec := &LetRec{CoreNode: CoreNode{NodeID: 30}, Bindings: []RecBinding{{Name: "r", Value: r(31)}}, Body: r(32)}
	in := coreLet(1, "r", litInt(2, 1), coreLet(3, "a", lam, coreLet(4, "b", match, coreLet(5, "c", rec, r(6)))))

	out := UniqueLocals(in).(*Let)
	a := out.Body.(*Let)
	b := a.Body.(*Let)
	c := b.Body.(*Let)
	l := a.Value.(*Lambda)
	if l.Params[0] == "r" || l.Body.(*Var).Name != l.Params[0] {
		t.Errorf("lambda param %q read as %q", l.Params[0], l.Body.(*Var).Name)
	}
	m := b.Value.(*Match)
	if m.Scrutinee.(*Var).Name != "r" {
		t.Errorf("scrutinee %q must read the outer r", m.Scrutinee.(*Var).Name)
	}
	pv := m.Arms[0].Pattern.(*ConstructorPattern).Args[0].(*VarPattern).Name
	if pv == "r" || m.Arms[0].Body.(*Var).Name != pv {
		t.Errorf("pattern variable %q read as %q", pv, m.Arms[0].Body.(*Var).Name)
	}
	lr := c.Value.(*LetRec)
	if n := lr.Bindings[0].Name; n == "r" || lr.Bindings[0].Value.(*Var).Name != n || lr.Body.(*Var).Name != n {
		t.Errorf("letrec member %q read as %q / %q", n, lr.Bindings[0].Value.(*Var).Name, lr.Body.(*Var).Name)
	}
	if got := c.Body.(*Var).Name; got != "r" {
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

// shadowIn builds `let r = 1 in let s = WRAP(let r = 2 in r) in r`: the inner
// let sits inside the node kind wrap builds. After uniqueLocals the inner r
// must have its own name, read by its own body, and the final read must stay
// the outer r. Every Core node kind that can hold a sub-expression is covered,
// so a kind the pass skipped would keep the inner name.
func shadowIn(wrap func(inner CoreExpr) CoreExpr) CoreExpr {
	inner := coreLet(10, "r", litInt(11, 2), coreVar(12, "r"))
	return coreLet(1, "r", litInt(2, 1), coreLet(3, "s", wrap(inner), coreVar(4, "r")))
}

func node(id uint64) CoreNode { return CoreNode{NodeID: id} }

func TestUniqueLocals_EveryNodeKind(t *testing.T) {
	x := func(id uint64) CoreExpr { return coreVar(id, "x") }
	cases := map[string]func(CoreExpr) CoreExpr{
		"app": func(e CoreExpr) CoreExpr {
			return &App{CoreNode: node(20), Func: x(21), Args: []CoreExpr{e}}
		},
		"app-func": func(e CoreExpr) CoreExpr {
			return &App{CoreNode: node(20), Func: e, Args: []CoreExpr{x(21)}}
		},
		"if-cond": func(e CoreExpr) CoreExpr {
			return &If{CoreNode: node(20), Cond: e, Then: x(21), Else: x(22)}
		},
		"if-else": func(e CoreExpr) CoreExpr {
			return &If{CoreNode: node(20), Cond: x(21), Then: x(22), Else: e}
		},
		"binop": func(e CoreExpr) CoreExpr {
			return &BinOp{CoreNode: node(20), Op: "+", Left: x(21), Right: e}
		},
		"unop": func(e CoreExpr) CoreExpr { return &UnOp{CoreNode: node(20), Op: "-", Operand: e} },
		"intrinsic": func(e CoreExpr) CoreExpr {
			return &Intrinsic{CoreNode: node(20), Op: OpAdd, Args: []CoreExpr{e, x(21)}}
		},
		"record": func(e CoreExpr) CoreExpr {
			return &Record{CoreNode: node(20), Fields: map[string]CoreExpr{"a": x(21), "b": e}}
		},
		"recordaccess": func(e CoreExpr) CoreExpr {
			return &RecordAccess{CoreNode: node(20), Record: e, Field: "a"}
		},
		"recordupdate": func(e CoreExpr) CoreExpr {
			return &RecordUpdate{CoreNode: node(20), Base: x(21), Updates: map[string]CoreExpr{"a": e}}
		},
		"recordupdate-base": func(e CoreExpr) CoreExpr {
			return &RecordUpdate{CoreNode: node(20), Base: e, Updates: map[string]CoreExpr{"a": x(21)}}
		},
		"list": func(e CoreExpr) CoreExpr {
			return &List{CoreNode: node(20), Elements: []CoreExpr{x(21), e}}
		},
		"array": func(e CoreExpr) CoreExpr {
			return &Array{CoreNode: node(20), Elements: []CoreExpr{e}}
		},
		"tuple": func(e CoreExpr) CoreExpr {
			return &Tuple{CoreNode: node(20), Elements: []CoreExpr{e, x(21)}}
		},
		"lambda": func(e CoreExpr) CoreExpr {
			return &Lambda{CoreNode: node(20), Params: []string{"q"}, Body: e}
		},
		"dictabs": func(e CoreExpr) CoreExpr {
			return &DictAbs{CoreNode: node(20), Params: []DictParam{{Name: "dict_Num_a"}}, Body: e}
		},
		"dictapp": func(e CoreExpr) CoreExpr {
			return &DictApp{CoreNode: node(20), Dict: x(21), Method: "add", Args: []CoreExpr{e}}
		},
		"forall": func(e CoreExpr) CoreExpr {
			return &Forall{CoreNode: node(20), Var: "i", Lo: litInt(21, 0), Hi: litInt(22, 3), Body: e}
		},
		"letrec-value": func(e CoreExpr) CoreExpr {
			return &LetRec{CoreNode: node(20), Bindings: []RecBinding{{Name: "g", Value: e}}, Body: x(21)}
		},
		"match-scrutinee": func(e CoreExpr) CoreExpr {
			return &Match{CoreNode: node(20), Scrutinee: e, Arms: []MatchArm{{Pattern: &WildcardPattern{}, Body: x(21)}}}
		},
		"match-guard": func(e CoreExpr) CoreExpr {
			return &Match{CoreNode: node(20), Scrutinee: x(21), Arms: []MatchArm{{Pattern: &WildcardPattern{}, Guard: e, Body: x(22)}}}
		},
	}
	for name, wrap := range cases {
		t.Run(name, func(t *testing.T) {
			in := shadowIn(wrap)
			before := in.String()
			out := UniqueLocals(in)
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
	vp := func() CorePattern { return &VarPattern{Name: "r"} }
	tail := CorePattern(&VarPattern{Name: "r"})
	pats := map[string]CorePattern{
		"var":         vp(),
		"constructor": &ConstructorPattern{Name: "Some", Args: []CorePattern{vp()}},
		"tuple":       &TuplePattern{Elements: []CorePattern{&WildcardPattern{}, vp()}},
		"list":        &ListPattern{Elements: []CorePattern{vp()}},
		"list-tail":   &ListPattern{Elements: []CorePattern{&LitPattern{Value: int64(1)}}, Tail: &tail},
		"record":      &RecordPattern{Fields: map[string]CorePattern{"a": &WildcardPattern{}, "b": vp()}},
	}
	for name, pat := range pats {
		t.Run(name, func(t *testing.T) {
			m := &Match{CoreNode: node(20), Scrutinee: coreVar(21, "r"), Arms: []MatchArm{{Pattern: pat, Body: coreVar(22, "r")}}}
			in := coreLet(1, "r", litInt(2, 1), coreLet(3, "s", m, coreVar(4, "r")))
			out := UniqueLocals(in).(*Let).Body.(*Let).Value.(*Match)
			read := out.Arms[0].Body.(*Var).Name
			if read == "r" || !patternBinds(out.Arms[0].Pattern, read) {
				t.Fatalf("arm reads %q; pattern %s must bind it", read, out.Arms[0].Pattern)
			}
			if sc := out.Scrutinee.(*Var).Name; sc != "r" {
				t.Fatalf("the scrutinee reads %q, want the outer r", sc)
			}
		})
	}
}

func patternBinds(p CorePattern, name string) bool {
	switch p := p.(type) {
	case *VarPattern:
		return p.Name == name
	case *ConstructorPattern:
		return anyPatternBinds(p.Args, name)
	case *TuplePattern:
		return anyPatternBinds(p.Elements, name)
	case *ListPattern:
		return anyPatternBinds(p.Elements, name) || (p.Tail != nil && patternBinds(*p.Tail, name))
	case *RecordPattern:
		for _, f := range p.Fields {
			if patternBinds(f, name) {
				return true
			}
		}
	}
	return false
}

func anyPatternBinds(ps []CorePattern, name string) bool {
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
	gt := func(id uint64, name string) CoreExpr {
		return &BinOp{CoreNode: node(id), Op: ">", Left: coreVar(id+1, name), Right: litInt(id+2, 5)}
	}
	some := func(n string) CorePattern {
		return &ConstructorPattern{Name: "Some", Args: []CorePattern{&VarPattern{Name: n}}}
	}
	// \x. \o. match o { Some(x) if x > 5 => x, Some(y) if x > 5 => y, _ => 0 }
	m := &Match{CoreNode: node(30), Scrutinee: coreVar(31, "o"), Arms: []MatchArm{
		{Pattern: some("x"), Guard: gt(40, "x"), Body: coreVar(43, "x")},
		{Pattern: some("y"), Guard: gt(50, "x"), Body: coreVar(53, "y")},
		{Pattern: &WildcardPattern{}, Body: litInt(60, 0)},
	}}
	lam := &Lambda{CoreNode: node(1), Params: []string{"x"}, Body: &Lambda{CoreNode: node(2), Params: []string{"o"}, Body: m}}
	out := UniqueLocals(lam).(*Lambda).Body.(*Lambda).Body.(*Match)
	pv := out.Arms[0].Pattern.(*ConstructorPattern).Args[0].(*VarPattern).Name
	if pv == "x" {
		t.Fatal("pattern x (shadowing the parameter) kept its name")
	}
	if g := out.Arms[0].Guard.(*BinOp).Left.(*Var).Name; g != pv {
		t.Fatalf("guard of arm 1 reads %q, want the pattern variable %q", g, pv)
	}
	if g := out.Arms[1].Guard.(*BinOp).Left.(*Var).Name; g != "x" {
		t.Fatalf("guard of arm 2 reads %q, want the parameter x", g)
	}

	// A match nested in a guard binds the name again: its arm reads its own.
	nested := &Match{CoreNode: node(70), Scrutinee: coreVar(71, "o"), Arms: []MatchArm{
		{Pattern: some("x"), Body: gt(72, "x")},
	}}
	m2 := &Match{CoreNode: node(80), Scrutinee: coreVar(81, "o"), Arms: []MatchArm{
		{Pattern: some("x"), Guard: nested, Body: coreVar(82, "x")},
	}}
	lam2 := &Lambda{CoreNode: node(3), Params: []string{"x", "o"}, Body: m2}
	out2 := UniqueLocals(lam2).(*Lambda).Body.(*Match)
	outerPV := out2.Arms[0].Pattern.(*ConstructorPattern).Args[0].(*VarPattern).Name
	inner := out2.Arms[0].Guard.(*Match)
	innerPV := inner.Arms[0].Pattern.(*ConstructorPattern).Args[0].(*VarPattern).Name
	if outerPV == "x" || innerPV == "x" || innerPV == outerPV {
		t.Fatalf("names: param x, arm %q, guard's arm %q: want three distinct", outerPV, innerPV)
	}
	if r := inner.Arms[0].Body.(*BinOp).Left.(*Var).Name; r != innerPV {
		t.Fatalf("nested arm reads %q, want %q", r, innerPV)
	}
	if r := out2.Arms[0].Body.(*Var).Name; r != outerPV {
		t.Fatalf("arm body reads %q, want %q", r, outerPV)
	}
}

// Record field values are expressions like any other: a binder inside one
// is renamed and its reads follow; the field NAMES never change.
func TestUniqueLocals_RecordFieldsVisited(t *testing.T) {
	inner := coreLet(10, "r", litInt(11, 2), coreVar(12, "r"))
	rec := &Record{CoreNode: node(20), Fields: map[string]CoreExpr{"r": inner, "q": coreVar(21, "r")}}
	out := UniqueLocals(coreLet(1, "r", litInt(2, 1), rec)).(*Let).Body.(*Record)
	if _, ok := out.Fields["r"]; !ok {
		t.Fatalf("field name r changed: %v", out.Fields)
	}
	l := out.Fields["r"].(*Let)
	if l.Name == "r" || l.Body.(*Var).Name != l.Name {
		t.Fatalf("binder inside field r: %s", l)
	}
	if q := out.Fields["q"].(*Var).Name; q != "r" {
		t.Fatalf("field q reads %q, want the outer r", q)
	}
}

// Unknown node and pattern types panic (lowerTopLevelDeclSafe turns that into
// an evaluator-only function, so a skipped subtree cannot keep a stale name);
// nil passes through. The shims satisfy the sealed interfaces by embedding them.
type exprShim struct{ CoreExpr }

type patternShim struct{ CorePattern }

func TestUniqueLocals_UnknownTypesPanic(t *testing.T) {
	if UniqueLocals(nil) != nil {
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
	mustPanic("expression", func() { UniqueLocals(exprShim{}) })
	mustPanic("pattern", func() {
		m := &Match{CoreNode: node(1), Scrutinee: coreVar(2, "o"), Arms: []MatchArm{{Pattern: patternShim{}, Body: litInt(3, 0)}}}
		UniqueLocals(m)
	})
}

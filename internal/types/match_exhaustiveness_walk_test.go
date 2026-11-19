package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// The walk must reach a match wherever Core can hold one: a match hidden in
// any of these positions that the walk skipped would never be checked.
func TestVerifyMatchExhaustivenessReachesEveryCoreForm(t *testing.T) {
	color := &TCon{Name: "Color", Module: "m"}
	wrap := map[string]func(core.CoreExpr) core.CoreExpr{
		"lambda":        func(m core.CoreExpr) core.CoreExpr { return &core.Lambda{Body: m} },
		"let value":     func(m core.CoreExpr) core.CoreExpr { return &core.Let{Value: m, Body: &core.Lit{}} },
		"let body":      func(m core.CoreExpr) core.CoreExpr { return &core.Let{Value: &core.Lit{}, Body: m} },
		"letrec body":   func(m core.CoreExpr) core.CoreExpr { return &core.LetRec{Body: m} },
		"app func":      func(m core.CoreExpr) core.CoreExpr { return &core.App{Func: m} },
		"app arg":       func(m core.CoreExpr) core.CoreExpr { return &core.App{Func: &core.Var{}, Args: []core.CoreExpr{m}} },
		"if cond":       func(m core.CoreExpr) core.CoreExpr { return &core.If{Cond: m, Then: &core.Lit{}, Else: &core.Lit{}} },
		"if then":       func(m core.CoreExpr) core.CoreExpr { return &core.If{Cond: &core.Lit{}, Then: m, Else: &core.Lit{}} },
		"if else":       func(m core.CoreExpr) core.CoreExpr { return &core.If{Cond: &core.Lit{}, Then: &core.Lit{}, Else: m} },
		"binop left":    func(m core.CoreExpr) core.CoreExpr { return &core.BinOp{Left: m, Right: &core.Lit{}} },
		"binop right":   func(m core.CoreExpr) core.CoreExpr { return &core.BinOp{Left: &core.Lit{}, Right: m} },
		"unop":          func(m core.CoreExpr) core.CoreExpr { return &core.UnOp{Operand: m} },
		"intrinsic":     func(m core.CoreExpr) core.CoreExpr { return &core.Intrinsic{Args: []core.CoreExpr{m}} },
		"record":        func(m core.CoreExpr) core.CoreExpr { return &core.Record{Fields: map[string]core.CoreExpr{"f": m}} },
		"record access": func(m core.CoreExpr) core.CoreExpr { return &core.RecordAccess{Record: m} },
		"record base":   func(m core.CoreExpr) core.CoreExpr { return &core.RecordUpdate{Base: m} },
		"record update": func(m core.CoreExpr) core.CoreExpr {
			return &core.RecordUpdate{Base: &core.Var{}, Updates: map[string]core.CoreExpr{"f": m}}
		},
		"list":     func(m core.CoreExpr) core.CoreExpr { return &core.List{Elements: []core.CoreExpr{m}} },
		"array":    func(m core.CoreExpr) core.CoreExpr { return &core.Array{Elements: []core.CoreExpr{m}} },
		"tuple":    func(m core.CoreExpr) core.CoreExpr { return &core.Tuple{Elements: []core.CoreExpr{m}} },
		"dict abs": func(m core.CoreExpr) core.CoreExpr { return &core.DictAbs{Body: m} },
		"dict app": func(m core.CoreExpr) core.CoreExpr {
			return &core.DictApp{Dict: &core.DictRef{}, Args: []core.CoreExpr{m}}
		},
		"forall lo":    func(m core.CoreExpr) core.CoreExpr { return &core.Forall{Lo: m, Hi: &core.Lit{}, Body: &core.Lit{}} },
		"forall hi":    func(m core.CoreExpr) core.CoreExpr { return &core.Forall{Lo: &core.Lit{}, Hi: m, Body: &core.Lit{}} },
		"forall body":  func(m core.CoreExpr) core.CoreExpr { return &core.Forall{Lo: &core.Lit{}, Hi: &core.Lit{}, Body: m} },
		"match guard":  nil, // below
		"global, leaf": nil,
	}
	for name, w := range wrap {
		t.Run(name, func(t *testing.T) {
			tc := newMatchTC()
			inner := matchOn(1, 9, color, tc, ctorArm("m", "Color", "Red", false))
			var expr core.CoreExpr
			switch name {
			case "match guard":
				outer := matchOn(2, 1, color, tc, wildArm(), wildArm())
				outer.Arms[0].Guard = inner
				expr = outer
			case "global, leaf":
				_, err := tc.VerifyMatchExhaustiveness(&core.VarGlobal{}, &core.Var{}, &core.Lit{}, &core.DictRef{})
				if err != nil {
					t.Fatal(err)
				}
				return
			default:
				expr = w(inner)
			}
			_, err := tc.VerifyMatchExhaustiveness(expr)
			var me *MatchError
			if !errors.As(err, &me) || me.Pos.Line != 9 {
				t.Fatalf("the match inside %s was not checked: %v", name, err)
			}
		})
	}
}

// Every pattern form converts, and the witness for each is the right one.
func TestVerifyMatchExhaustivenessPatternForms(t *testing.T) {
	tc := newMatchTC()
	wild := &core.WildcardPattern{}
	v := &core.VarPattern{Name: "x"}
	tail := core.CorePattern(&core.VarPattern{Name: "r"})
	arm := func(p core.CorePattern) core.MatchArm { return core.MatchArm{Pattern: p, Body: &core.Lit{}} }
	cases := []struct {
		name string
		typ  Type
		arms []core.MatchArm
		want string // "" = exhaustive
	}{
		{"tuple of bools", &TTuple{Elements: []Type{TBool, TBool}},
			[]core.MatchArm{arm(&core.TuplePattern{Elements: []core.CorePattern{&core.LitPattern{Value: true}, wild}})}, "(false, _)"},
		{"list with a rest", &TList{Element: TInt},
			[]core.MatchArm{arm(&core.ListPattern{}), arm(&core.ListPattern{Elements: []core.CorePattern{v}, Tail: &tail})}, ""},
		{"list missing []", &TList{Element: TInt},
			[]core.MatchArm{arm(&core.ListPattern{Elements: []core.CorePattern{v}, Tail: &tail})}, "[]"},
		{"record", &TRecord{Fields: map[string]Type{"a": TBool}},
			[]core.MatchArm{arm(&core.RecordPattern{Fields: map[string]core.CorePattern{"a": &core.LitPattern{Value: false}}})}, "{a: true, ...}"},
		{"float32 literal", TFloat,
			[]core.MatchArm{arm(&core.LitPattern{Value: float32(1.5)})}, "_ (any float other than 1.5)"},
		{"string literal, variable closes it", TString,
			[]core.MatchArm{arm(&core.LitPattern{Value: "a"}), arm(v)}, ""},
		{"type variable scrutinee", &TVar{Name: "a"},
			[]core.MatchArm{arm(&core.LitPattern{Value: int64(3)})}, "_ (any int other than 3)"},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := matchOn(uint64(10+i), 4, c.typ, tc, c.arms...)
			_, err := tc.VerifyMatchExhaustiveness(m)
			if c.want == "" {
				if err != nil || !m.Exhaustive {
					t.Fatalf("want exhaustive, got %v", err)
				}
				return
			}
			var me *MatchError
			if !errors.As(err, &me) || !strings.HasSuffix(me.Message, "no arm matches: "+c.want) {
				t.Fatalf("want missing %q, got %v", c.want, err)
			}
			if _, isVar := c.typ.(*TVar); isVar && strings.Contains(me.Message, " on ") {
				t.Errorf("a type variable must not be named as the scrutinee's type: %s", me.Message)
			}
		})
	}
}

// An unstamped (by-name) constructor pattern takes its ADT from the type it
// was checked against — after substitution — and from nothing else.
func TestVerifyMatchExhaustivenessByNameIdentity(t *testing.T) {
	byName := func() *core.ConstructorPattern { return &core.ConstructorPattern{Name: "Violet"} }
	arm := func(p core.CorePattern) core.MatchArm { return core.MatchArm{Pattern: p, Body: &core.Lit{}} }

	// Recorded as a type variable, decided by a later substitution.
	tc := newMatchTC()
	p := byName()
	tc.recordCtorPatternType(p, &TVar2{Name: "t1"})
	tc.substituteCtorPatternTypes(Substitution{"t1": &TCon{Name: "Color", Module: "n"}})
	m := matchOn(1, 2, &TCon{Name: "Color", Module: "n"}, tc, arm(p), arm(&core.ConstructorPattern{Name: "Red", TypeName: "Color", TypeModule: "n"}))
	if _, err := tc.VerifyMatchExhaustiveness(m); err != nil || !m.Exhaustive {
		t.Fatalf("substituted identity: %v", err)
	}

	if p.TypeName != "Color" || p.TypeModule != "n" {
		t.Errorf("substituted identity not stamped: %s/%s", p.TypeModule, p.TypeName)
	}

	// Still a type variable: nothing fixes the value's type — TC_MATCH_007,
	// with a catch-all too (#2303).
	for _, catchAll := range []bool{false, true} {
		tc = newMatchTC()
		p = byName()
		tc.recordCtorPatternType(p, &TVar2{Name: "t9"})
		arms := []core.MatchArm{arm(p)}
		if catchAll {
			arms = append(arms, wildArm())
		}
		_, err := tc.VerifyMatchExhaustiveness(matchOn(2, 3, &TVar2{Name: "t9"}, tc, arms...))
		var me *MatchError
		if !errors.As(err, &me) || me.Code != TC_MATCH_007 || p.TypeName != "" {
			t.Fatalf("catch-all %v: want TC_MATCH_007 and no stamp, got %v (stamp %q)", catchAll, err, p.TypeName)
		}
	}

	// Never type-checked (no record) and not in this module's scope: no
	// guess from the one loaded ADT that declares it.
	tc = newMatchTC()
	unseen := byName()
	m = matchOn(3, 4, &TVar2{Name: "t2"}, tc, arm(unseen), wildArm())
	if _, err := tc.VerifyMatchExhaustiveness(m); err == nil || !strings.Contains(err.Error(), "TC_MATCH_007") || unseen.TypeName != "" {
		t.Fatalf("unseen, out of scope: want TC_MATCH_007, got %v (stamp %q)", err, unseen.TypeName)
	}
	// Never type-checked, in scope: the ADT its name denotes here, stamped.
	tc = newMatchTC()
	tc.constructorTypes = map[string]string{"Violet": "Color"}
	tc.ctorADTModule = map[string]string{"Violet": "n"}
	unseen = byName()
	m = matchOn(3, 4, &TVar2{Name: "t2"}, tc, arm(unseen), wildArm())
	if _, err := tc.VerifyMatchExhaustiveness(m); err != nil || unseen.TypeName != "Color" || unseen.TypeModule != "n" {
		t.Fatalf("unseen, in scope: %v (stamp %s/%s)", err, unseen.TypeModule, unseen.TypeName)
	}

	// No candidate at all.
	tc = newMatchTC()
	_, err := tc.VerifyMatchExhaustiveness(matchOn(4, 5, TInt, tc, arm(&core.ConstructorPattern{Name: "Nowhere"})))
	if err == nil || !strings.Contains(err.Error(), "TC_MATCH_007: cannot tell which type Nowhere belongs to") ||
		!strings.Contains(err.Error(), "import the type that declares Nowhere") {
		t.Fatalf("want TC_MATCH_007 with no candidates, got %v", err)
	}

	// Fixed to a type that has no such constructor: the arm can never match.
	tc = newMatchTC()
	p = byName()
	tc.recordCtorPatternType(p, &TCon{Name: "Color", Module: "m"})
	_, err = tc.VerifyMatchExhaustiveness(matchOn(5, 6, &TCon{Name: "Color", Module: "m"}, tc, arm(p), wildArm()))
	if err == nil || !strings.Contains(err.Error(), "TC_MATCH_007: constructor Violet is matched by name against a value of type Color, which has no constructor Violet") {
		t.Fatalf("foreign by-name constructor: %v", err)
	}

	// A stamped pattern is never recorded.
	stamped := &core.ConstructorPattern{Name: "Red", TypeName: "Color", TypeModule: "m"}
	tc.recordCtorPatternType(stamped, TInt)
	if _, ok := tc.ctorPatternTypes[stamped]; ok {
		t.Error("a stamped pattern was recorded")
	}
}

// The diagnostics' own shapes: budget, joined errors, warnings, ordering.
func TestMatchDiagnosticShapes(t *testing.T) {
	pos := ast.Pos{File: "m.ail", Line: 3, Column: 7}
	b := newBudgetError(pos, exhaust.ErrBudget)
	if b.Code != TC_MATCH_004 || b.Suggestion != suggestCatchAll || !strings.Contains(b.Error(), "too large to check") {
		t.Errorf("budget error: %+v", b)
	}
	errs := MatchErrors{b, newUndecidedError(pos, errUndecidable)}
	if strings.Count(errs.Error(), "\n") < 2 || len(errs.Unwrap()) != 2 {
		t.Errorf("joined errors: %q", errs.Error())
	}
	w := &MatchWarning{Pos: pos, Arm: 2, Pattern: "Idle"}
	if l, c := w.Position(); l != 3 || c != 7 || w.WarningFile() != "m.ail" || w.Suggestion() == "" ||
		!strings.HasPrefix(w.String(), "warning: TC_MATCH_003 at m.ail:3:7: arm 2 (Idle)") {
		t.Errorf("warning: %s", w)
	}
	nullary := newPatternArityError(pos, &patternArityError{pattern: &core.ConstructorPattern{Name: "E", Args: []core.CorePattern{&core.WildcardPattern{}}}, arity: 0})
	if nullary.Suggestion != "write E" || nullary.Pos != pos {
		t.Errorf("nullary arity fix: %+v", nullary)
	}
	if !posLess(ast.Pos{File: "a"}, ast.Pos{File: "b"}) || !posLess(ast.Pos{Line: 1}, ast.Pos{Line: 2}) ||
		!posLess(ast.Pos{Line: 1, Column: 1}, ast.Pos{Line: 1, Column: 2}) || posLess(ast.Pos{Line: 2}, ast.Pos{Line: 1}) {
		t.Error("posLess")
	}
	// A match the elaborator gave no surface span reports its Core span.
	m := &core.Match{CoreNode: core.CoreNode{CoreSpan: ast.Pos{Line: 8, Column: 2}}}
	if got := matchPosition(m); got.Line != 8 {
		t.Errorf("matchPosition fallback: %v", got)
	}
}

// A catch-all rescues a match whose coverage cannot be decided — but never a
// constructor pattern whose type cannot be named (#2303: the back ends test
// the ordinal of the pattern's own type), at any depth, never a wrong-arity
// pattern (#2243: the engines disagree on the arm), and never a
// scrutinee/pattern disagreement (a checker bug).
func TestVerifyMatchExhaustivenessCatchAllRescuesOnlyCoverage(t *testing.T) {
	arm := func(p core.CorePattern) core.MatchArm { return core.MatchArm{Pattern: p, Body: &core.Lit{}} }
	unfixedViolet := func(tc *CoreTypeChecker, args ...core.CorePattern) *core.ConstructorPattern {
		p := &core.ConstructorPattern{Name: "Violet", Args: args, Pos: ast.Pos{Line: 5, Column: 9}}
		tc.recordCtorPatternType(p, &TVar2{Name: "t1"})
		return p
	}
	two := []core.CorePattern{&core.VarPattern{Name: "a"}, &core.VarPattern{Name: "b"}}

	// Placed (stamped) and wrong arity, under a catch-all.
	tc := newMatchTC()
	placed := &core.ConstructorPattern{Name: "Violet", TypeName: "Color", TypeModule: "n", Args: two, Pos: ast.Pos{Line: 5, Column: 9}}
	m := matchOn(1, 5, &TCon{Name: "Color", Module: "n"}, tc, arm(placed), wildArm())
	_, err := tc.VerifyMatchExhaustiveness(m)
	var me *MatchError
	if !errors.As(err, &me) || me.Code != TC_MATCH_006 || me.Suggestion != "write Violet" {
		t.Fatalf("wrong-arity pattern under a catch-all: %v", err)
	}

	// Unplaced, top level and nested, right arity or not, under a catch-all.
	for _, c := range []struct {
		name string
		pat  func(tc *CoreTypeChecker) core.CorePattern
	}{
		{"right arity", func(tc *CoreTypeChecker) core.CorePattern { return unfixedViolet(tc) }},
		{"wrong arity", func(tc *CoreTypeChecker) core.CorePattern { return unfixedViolet(tc, two...) }},
		{"nested in a tuple", func(tc *CoreTypeChecker) core.CorePattern {
			return &core.TuplePattern{Elements: []core.CorePattern{unfixedViolet(tc, two[0]), &core.WildcardPattern{}}}
		}},
		{"nested in a stamped constructor", func(tc *CoreTypeChecker) core.CorePattern {
			return &core.ConstructorPattern{Name: "Red", TypeName: "Color", TypeModule: "n", Args: []core.CorePattern{unfixedViolet(tc)}}
		}},
	} {
		tc = newMatchTC()
		m = matchOn(2, 6, &TVar2{Name: "t1"}, tc, arm(c.pat(tc)), wildArm())
		if _, err := tc.VerifyMatchExhaustiveness(m); !errors.As(err, &me) || me.Code != TC_MATCH_007 || m.Exhaustive {
			t.Fatalf("%s, unplaced, under a catch-all: %v", c.name, err)
		}
		if me.Pos.Line != 5 || me.Pos.Column != 9 || me.Pos.File != "m.ail" {
			t.Errorf("%s: located at %v, want the pattern m.ail:5:9", c.name, me.Pos)
		}
	}

	// Without a catch-all, the same code and the fix that decides it.
	tc = newMatchTC()
	m = matchOn(4, 8, &TVar2{Name: "t1"}, tc, arm(unfixedViolet(tc)))
	_, err = tc.VerifyMatchExhaustiveness(m)
	if !errors.As(err, &me) || me.Code != TC_MATCH_007 || me.Suggestion != "import n (Color)" ||
		strings.Contains(me.Message, "t1") || !strings.Contains(me.Message, "cannot tell which type Violet belongs to here") {
		t.Fatalf("by-name on an untyped value: %v (suggestion %q)", err, me.Suggestion)
	}

	tc = newMatchTC()
	m = matchOn(5, 9, &TCon{Name: "Color", Module: "n"}, tc, ctorArm("m", "Color", "Red", false), wildArm())
	if _, err := tc.VerifyMatchExhaustiveness(m); err == nil || !strings.Contains(err.Error(), "compiler bug") {
		t.Fatalf("a disagreement must be reported under a catch-all: %v", err)
	}
}

// A by-name constructor whose type is not fixed is TC_MATCH_007 whatever its
// argument count and whatever the loaded types declaring it agree on (#2303,
// which also retires #2272's arity verdict that depended on them). The
// suggestion names every candidate, sorted, so it is the same on every run
// (the candidates come from a map).
func TestVerifyMatchExhaustivenessUnresolvedByNameCandidates(t *testing.T) {
	arm := func(p core.CorePattern) core.MatchArm { return core.MatchArm{Pattern: p, Body: &core.Lit{}} }
	want := "cannot tell which type Msg belongs to here: it is matched by name (neither its type nor the constructor is imported here), and nothing fixes the type of the value matched"
	for run := 0; run < 50; run++ {
		for _, n := range []int{1, 2, 3} {
			tc := NewCoreTypeChecker()
			tc.CoreTI = CoreTypeInfo{}
			tc.SetMatchUniverse(MatchUniverse{
				NominalKey("ar/y", "Pkt"): {{Name: "Done", Arity: 1}, {Name: "Msg", Arity: 2}},
				NominalKey("ar/x", "Ev"):  {{Name: "Done"}, {Name: "Msg", Arity: 1}},
				NominalKey("ar/z", "Zz"):  {{Name: "Other"}},
			})
			args := make([]core.CorePattern, n)
			for i := range args {
				args[i] = &core.WildcardPattern{}
			}
			p := &core.ConstructorPattern{Name: "Msg", Args: args, Pos: ast.Pos{Line: 6, Column: 28}}
			tc.recordCtorPatternType(p, &TVar2{Name: "t1"})
			_, err := tc.VerifyMatchExhaustiveness(matchOn(1, 6, &TVar2{Name: "t1"}, tc, arm(p), wildArm()))
			var me *MatchError
			if !errors.As(err, &me) || me.Code != TC_MATCH_007 || me.Message != want ||
				me.Suggestion != "import the type you mean: import ar/x (Ev) or import ar/y (Pkt)" ||
				len(me.Notes) != 1 || me.Notes[0] != "Loaded types declaring Msg: ar/x's Ev, ar/y's Pkt. A catch-all arm does not decide which one is meant." {
				t.Fatalf("run %d, %d args: %v (suggestion %q)", run, n, err, me.Suggestion)
			}
		}
	}
	// Candidates that agree on the arity: the same error, not an arity one.
	tc := NewCoreTypeChecker()
	tc.CoreTI = CoreTypeInfo{}
	tc.SetMatchUniverse(MatchUniverse{NominalKey("a", "T"): {{Name: "K", Arity: 2}}, NominalKey("b", "U"): {{Name: "K", Arity: 2}}})
	p := &core.ConstructorPattern{Name: "K", Args: []core.CorePattern{&core.WildcardPattern{}}}
	tc.recordCtorPatternType(p, &TVar2{Name: "t1"})
	_, err := tc.VerifyMatchExhaustiveness(matchOn(2, 3, &TVar2{Name: "t1"}, tc, arm(p), wildArm()))
	var me *MatchError
	if !errors.As(err, &me) || me.Code != TC_MATCH_007 {
		t.Fatalf("agreeing candidates: %v", err)
	}
	// The structured report carries the code and the candidates.
	rep, ok := ailerrors.AsReport(err)
	if !ok || rep.Code != TC_MATCH_007 || rep.Data["constructor"] != "K" {
		t.Fatalf("report: %+v", rep)
	}
}

// At most five witnesses are named; more are summarised.
func TestVerifyMatchExhaustivenessManyWitnesses(t *testing.T) {
	tc := newMatchTC()
	var sigs []exhaust.CtorSig
	for _, n := range []string{"C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8"} {
		sigs = append(sigs, exhaust.CtorSig{Name: n})
	}
	tc.SetMatchUniverse(MatchUniverse{NominalKey("m", "Big"): sigs})
	m := matchOn(1, 2, &TCon{Name: "Big", Module: "m"}, tc, ctorArm("m", "Big", "C1", false))
	_, err := tc.VerifyMatchExhaustiveness(m)
	var me *MatchError
	if !errors.As(err, &me) || !strings.HasSuffix(me.Message, "no arm matches: C2, C3, C4, C5, C6, and more") {
		t.Fatalf("got %v", err)
	}
	m = matchOn(2, 3, &TCon{Name: "Big", Module: "m"}, tc, ctorArm("m", "Big", "C1", false), ctorArm("m", "Big", "C2", false), ctorArm("m", "Big", "C3", false))
	if _, err := tc.VerifyMatchExhaustiveness(m); !errors.As(err, &me) || strings.Contains(me.Message, "and more") {
		t.Fatalf("exactly five missing: %v", err)
	}
}

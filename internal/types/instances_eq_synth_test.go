package types

import (
	"errors"
	"strings"
	"testing"
)

func listOf(t Type) Type   { return &TApp{Constructor: &TCon{Name: "list"}, Args: []Type{t}} }
func optionOf(t Type) Type { return &TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{t}} }
func resultOf(t, e Type) Type {
	return &TApp{Constructor: &TCon{Name: "Result"}, Args: []Type{t, e}}
}

var intToInt = &TFunc2{Params: []Type{TInt}, Return: TInt}

// TestEqSynthesis pins which shapes get a synthesized, structural Eq.
func TestEqSynthesis(t *testing.T) {
	env := LoadBuiltinInstances()
	if err := env.AddDerivedEqForADT("Color"); err != nil {
		t.Fatal(err)
	}
	if err := env.AddDerivedEqForADT("Point"); err != nil {
		t.Fatal(err)
	}
	color := &TCon{Name: "Color"}
	point := &TRecord{Fields: map[string]Type{"x": TInt, "y": TInt}, TypeName: "Point"}

	cases := []struct {
		name string
		typ  Type
		ok   bool
	}{
		{"list of int", listOf(TInt), true},
		{"legacy list", &TList{Element: TString}, true},
		{"option of derived ADT", optionOf(color), true},
		{"result", resultOf(TInt, TString), true},
		{"tuple", &TTuple{Elements: []Type{TInt, TString, TBool}}, true},
		{"derived record", point, true},
		{"nested", listOf(optionOf(&TTuple{Elements: []Type{point, listOf(TInt)}})), true},
		{"list of functions", listOf(intToInt), false},
		{"option of non-derived ADT", optionOf(&TCon{Name: "Shape"}), false},
		{"tuple with function", &TTuple{Elements: []Type{TInt, intToInt}}, false},
		{"anonymous record", &TRecord{Fields: map[string]Type{"x": TInt}}, false},
		{"record of non-derived alias", &TRecord{Fields: map[string]Type{"x": TInt}, TypeName: "Q"}, false},
		{"polymorphic user ADT", &TApp{Constructor: &TCon{Name: "Tree"}, Args: []Type{TInt}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inst, err := env.Lookup("Eq", c.typ)
			if c.ok {
				if err != nil {
					t.Fatalf("Eq[%s]: unexpected error %v", c.typ, err)
				}
				if !inst.Structural {
					t.Errorf("Eq[%s] must be a structural (synthesized) instance", c.typ)
				}
				return
			}
			var missing *MissingInstanceError
			if !errors.As(err, &missing) {
				t.Fatalf("Eq[%s]: want MissingInstanceError, got %v", c.typ, err)
			}
		})
	}
}

// TestEqSynthesisNamesTheInnermostPart: a missing element instance must name the
// element, not an intermediate container.
func TestEqSynthesisNamesTheInnermostPart(t *testing.T) {
	env := LoadBuiltinInstances()
	_, err := env.Lookup("Eq", listOf(optionOf(intToInt)))
	var missing *MissingInstanceError
	if !errors.As(err, &missing) {
		t.Fatalf("want MissingInstanceError, got %v", err)
	}
	if !strings.Contains(missing.Hint, "needs == on int -> int") {
		t.Errorf("hint should name the function element: %q", missing.Hint)
	}
	if !strings.Contains(missing.Hint, "Functions have no ==") {
		t.Errorf("hint should carry the element's own fix: %q", missing.Hint)
	}
}

// nestLists wraps int in n list layers.
func nestLists(n int) Type {
	var typ Type = TInt
	for i := 0; i < n; i++ {
		typ = listOf(typ)
	}
	return typ
}

// TestEqSynthesisDepthCap: 8 nested containers resolve; 9 fail with the distinct
// E_EQ_SYNTH_DEPTH error naming the outermost type, never a plain "No instance".
// The 08-29 attempt shipped a cap that was dead code (`if depth > 0 && false`);
// this test fails if the depth increment in synthesizeEq is removed, because the
// 9-deep lookup would then succeed.
func TestEqSynthesisDepthCap(t *testing.T) {
	env := LoadBuiltinInstances()
	if _, err := env.Lookup("Eq", nestLists(eqSynthDepthCap)); err != nil {
		t.Fatalf("depth %d must resolve: %v", eqSynthDepthCap, err)
	}
	deep := nestLists(eqSynthDepthCap + 1)
	_, err := env.Lookup("Eq", deep)
	var depthErr *EqSynthDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("depth %d: want EqSynthDepthError, got %v", eqSynthDepthCap+1, err)
	}
	if !depthErr.Type.Equals(deep) {
		t.Errorf("depth error should name the outermost type %s, got %s", deep, depthErr.Type)
	}
	if !strings.Contains(err.Error(), "E_EQ_SYNTH_DEPTH") {
		t.Errorf("error must carry its code: %v", err)
	}
}

// TestReduceEqConstraints pins R-D7: a non-ground Eq is decomposed, never
// skipped. Bare variables survive as residuals; ground parts must resolve;
// a function behind a residual variable is rejected.
func TestReduceEqConstraints(t *testing.T) {
	env := LoadBuiltinInstances()
	alpha := &TVar2{Name: "α1", Kind: Star}
	cases := []struct {
		name     string
		typ      Type
		residual int  // number of Eq[var] constraints left
		ok       bool // false: must be rejected
	}{
		{"bare variable", alpha, 1, true},
		{"Option[α] (None == None)", optionOf(alpha), 1, true},
		{"[α] ([] == [])", &TList{Element: alpha}, 1, true},
		{"Result[int, α]", resultOf(TInt, alpha), 1, true},
		{"(int, Option[α])", &TTuple{Elements: []Type{TInt, optionOf(alpha)}}, 1, true},
		{"Result[α, int -> int] (Err(f) == Err(f))", resultOf(alpha, intToInt), 0, false},
		{"α -> int", &TFunc2{Params: []Type{alpha}, Return: TInt}, 0, false},
		{"polymorphic user ADT Tree[α]", &TApp{Constructor: &TCon{Name: "Tree"}, Args: []Type{alpha}}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := env.ReduceEqConstraints([]ClassConstraint{{Class: "Eq", Type: c.typ, Path: []string{"here"}}})
			if !c.ok {
				if err == nil {
					t.Fatalf("Eq[%s] must be rejected, reduced to %v", c.typ, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eq[%s]: %v", c.typ, err)
			}
			if len(out) != c.residual {
				t.Fatalf("Eq[%s]: want %d residual, got %v", c.typ, c.residual, out)
			}
		})
	}
}

// TestConstraintIsGroundIsScopedToEq: only Eq looks inside structures. Every
// other class keeps the shallow rule, so Num[int -> β] resolves now and fails,
// which is what makes `42(1)` a type error (internal/repl TestLoadModuleTypeError).
func TestConstraintIsGroundIsScopedToEq(t *testing.T) {
	beta := &TVar2{Name: "β1", Kind: Star}
	fn := &TFunc2{Params: []Type{TInt}, Return: beta}
	if !constraintIsGround(ClassConstraint{Class: "Num", Type: fn}) {
		t.Error("Num[int -> β] must be resolved now (and fail), not generalized")
	}
	if constraintIsGround(ClassConstraint{Class: "Eq", Type: &TList{Element: beta}}) {
		t.Error("Eq[[β]] must be non-ground so it is reduced, not rejected")
	}
}

// TestReduceEqConstraintsDepthCap: the non-ground reduction is capped like
// synthesis. Removing the depth increment in reduceEq makes this fail
// (sprint evaluator finding 3: no test covered it).
func TestReduceEqConstraintsDepthCap(t *testing.T) {
	env := LoadBuiltinInstances()
	var deep Type = &TVar2{Name: "α1", Kind: Star}
	for i := 0; i < eqSynthDepthCap+1; i++ {
		deep = &TList{Element: deep}
	}
	_, err := env.ReduceEqConstraints([]ClassConstraint{{Class: "Eq", Type: deep, Path: []string{"here"}}})
	if err == nil || !strings.Contains(err.Error(), "E_EQ_SYNTH_DEPTH") {
		t.Fatalf("depth %d residual Eq: want E_EQ_SYNTH_DEPTH, got %v", eqSynthDepthCap+1, err)
	}
}

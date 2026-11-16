package types

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// One of each type form, nested deep enough to exceed the exact-render bound,
// so the bounded printer and the node counter walk every case.
func everyForm(leaf Type) Type {
	fn := &TFunc2{Params: []Type{leaf}, Return: leaf, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}}}
	return &TTuple{Elements: []Type{
		&TList{Element: leaf},
		&TArray{Element: leaf},
		&TMap{Key: TString, Value: leaf},
		&TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{leaf}},
		fn,
		&TRecord{Fields: map[string]Type{"a": leaf, "b": leaf}},
		&TRecord{Fields: map[string]Type{"n": leaf}, TypeName: "Named"},
		&TRecordOpen{Fields: map[string]Type{"o": leaf}, Row: &RowVar{Name: "r", Kind: RecordRow}},
		&TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"z": leaf}}},
		&TLabelled{Inner: leaf, L: LabelBottom()},
		&TVar2{Name: "v", Kind: Star},
	}}
}

func TestDisplayType_EveryFormRendersBounded(t *testing.T) {
	var big Type = TInt
	for i := 0; i < 8; i++ {
		big = everyForm(big)
	}
	s := displayType(big)
	if !strings.Contains(s, "shown in part") || len(s) > displayTypeChars+200 {
		t.Fatalf("bounded render of %d bytes: %.120s", len(s), s)
	}
	for _, want := range []string{"[", "Array[", "Map[", "Option[", ") -> ", "{a: ", "Named"} {
		if !strings.Contains(s, want) {
			t.Errorf("bounded render lacks %q: %.300s", want, s)
		}
	}
	if displayType(nil) != "<nil>" {
		t.Error("nil type")
	}
	// A multi-byte name cut at the budget stays valid UTF-8.
	p := &boundedPrinter{budget: 3}
	p.str("aé€")
	if p.b.String() != "aé" {
		t.Errorf("cut %q, want %q", p.b.String(), "aé")
	}
}

func TestFormat_BoundedOnlyBeyondTheTypeBound(t *testing.T) {
	small := everyForm(TInt)
	for _, x := range small.(*TTuple).Elements {
		if got := fmt.Sprintf("%v", x); got != x.String() {
			t.Errorf("%T formats %q, String() is %q", x, got, x.String())
		}
	}
	if got := fmt.Sprintf("%q", small); got != fmt.Sprintf("%q", small.String()) {
		t.Errorf("%%q: %s", got)
	}
	if got := fmt.Sprintf("%d", small); !strings.HasPrefix(got, "%!d(") {
		t.Errorf("%%d: %s", got)
	}
	big := doublingTuple(TInt, 20)
	if got := fmt.Sprintf("type %s", big); !strings.Contains(got, "shown in part") {
		t.Errorf("a type beyond the bound formatted in full (%d bytes)", len(got))
	}
	for _, x := range []Type{&Row{Kind: EffectRow, Labels: map[string]Type{"IO": doublingTuple(TInt, 17)}}, &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"z": doublingTuple(TInt, 17)}}}} {
		if !exceedsNodes(x, maxTypeTreeNodes) {
			t.Errorf("%T over the bound not counted", x)
		}
	}
	if exceedsNodes(nil, 0) {
		t.Error("nil counted")
	}
}

func TestTypeSize_Helpers(t *testing.T) {
	for n, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 65536: "65,536", 1234567: "1,234,567"} {
		if got := groupDigits(n); got != want {
			t.Errorf("groupDigits(%d) = %q", n, got)
		}
	}
	if letBindingWhat("$tmp3") != "an intermediate expression" || letBindingWhat("x") != "let binding x" {
		t.Error("letBindingWhat")
	}
	if positionString(ast.Pos{}) != "" || positionString(ast.Pos{File: "m.ail", Line: 2, Column: 3}) == "" {
		t.Error("positionString")
	}
	if NewTypeSizeCheck().Check(nil, "x") != nil {
		t.Error("a nil type was rejected")
	}
	if len(TypeSizeCodes()) != 2 {
		t.Error("TypeSizeCodes")
	}
	msg := typeWorkExceeded{pos: "m.ail:1:1", steps: 10, limit: 9, nodes: 3}.Error()
	if !strings.Contains(msg, TC_TYPE_WORK_001) || !strings.Contains(msg, "at m.ail:1:1") {
		t.Errorf("work message: %s", msg)
	}
	detail := typeTooLarge{what: "x", pos: "m.ail:1:1", detail: "its types nest more than 1000 deep"}.Error()
	if !strings.Contains(detail, "nest more than 1000 deep") || !strings.Contains(detail, "at m.ail:1:1") {
		t.Errorf("detail message: %s", detail)
	}
	s := newTypeSizer()
	if s.size(nil) != 0 || s.size(everyForm(TInt)) < 20 {
		t.Error("sizer on every form")
	}
	var nilWork *unifyWork
	nilWork.step() // an unbounded unifier counts nothing
}

func TestAliasApplicationKey_Forms(t *testing.T) {
	c := newTypeCanon()
	if k, ok := aliasApplicationKey(&TCon{Name: "A", Module: "m"}, c); !ok || k != "m::A" {
		t.Errorf("TCon key %q", k)
	}
	inner := &TApp{Constructor: &TCon{Name: "F"}, Args: []Type{TInt}}
	if k, ok := aliasApplicationKey(&TApp{Constructor: inner, Args: []Type{TString}}, c); !ok || !strings.HasPrefix(k, "F|") || strings.Count(k, "|") != 2 {
		t.Errorf("nested application key %q", k)
	}
	if _, ok := aliasApplicationKey(&TApp{Constructor: &TVar2{Name: "f"}}, c); ok {
		t.Error("a variable head keyed")
	}
	if _, ok := aliasApplicationKey(&TList{Element: TInt}, c); ok {
		t.Error("a list keyed")
	}
	// Two expansions rebuild `(int, int)`; the key is the same for both.
	pair := func() Type { return &TTuple{Elements: []Type{TInt, &TCon{Name: "int"}}} }
	k1, _ := aliasApplicationKey(&TApp{Constructor: &TCon{Name: "Q"}, Args: []Type{pair()}}, c)
	k2, _ := aliasApplicationKey(&TApp{Constructor: &TCon{Name: "Q"}, Args: []Type{pair()}}, c)
	if k1 != k2 {
		t.Errorf("structurally equal arguments keyed apart: %q %q", k1, k2)
	}
	fn := func() Type { return &TFunc2{Params: []Type{TInt}, Return: TInt} }
	if f1, f2 := fn(), fn(); c.id(f1) == c.id(f2) {
		t.Error("two function types numbered alike: they are compared by identity only")
	}
	for _, x := range []Type{nil, &TVar{Name: "a"}, &TVar2{Name: "a"}, &TList{Element: TInt}, &TArray{Element: TInt}, &TMap{Key: TInt, Value: TInt}} {
		first := c.id(x)
		if again := c.id(x); first != again {
			t.Errorf("%v numbered twice", x)
		}
	}
}

func TestSameShape_Forms(t *testing.T) {
	a, b := &TVar2{Name: "a", Kind: Star}, &TVar{Name: "b"}
	bodies := func(x Type) Type {
		return &TTuple{Elements: []Type{
			&TList{Element: x}, &TArray{Element: x}, &TMap{Key: TString, Value: x},
			&TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{x}}, b,
			&TRecord{Fields: map[string]Type{"f": x}},
		}}
	}
	u := NewUnifierWithAliasesAndParams(map[string]Type{
		"S1": bodies(a), "S2": bodies(a), "S3": bodies(&TList{Element: a}),
		"K1": &TApp{Constructor: &TCon{Name: "S1"}, Args: []Type{a}},
		"K2": &TApp{Constructor: &TCon{Name: "S2"}, Args: []Type{a}},
		"K3": &TApp{Constructor: &TCon{Name: "S1"}, Args: []Type{TInt}},
		"L1": &TRecord{Fields: map[string]Type{"f": a}},
		"L2": &TRecord{Fields: map[string]Type{"g": a}},
	}, map[string][]string{"S1": {"a"}, "S2": {"a"}, "S3": {"a"}, "K1": {"a"}, "K2": {"a"}, "K3": {"a"}, "L1": {"a"}, "L2": {"a"}})
	if !u.aliasesEquivalent("S1", "S2") || !u.aliasesEquivalent("K1", "K2") {
		t.Error("equivalent bodies of every compared form not equivalent")
	}
	for _, p := range [][2]string{{"S1", "S3"}, {"K1", "K3"}, {"L1", "L2"}, {"S1", "missing"}} {
		if u.aliasesEquivalent(p[0], p[1]) {
			t.Errorf("%s ~ %s", p[0], p[1])
		}
	}
	if u.sameShape(nil, TInt, nil) || !u.sameShape(nil, nil, nil) {
		t.Error("nil bodies")
	}
	if u.sameShape(&TCon{Name: "S1"}, TInt, nil) || u.sameShape(TInt, &TCon{Name: "S1"}, nil) {
		t.Error("an alias against a non-alias compared equal")
	}
	if u.sameAliasApplication(TInt, TInt) || u.sameAliasApplication(&TCon{Name: "S1"}, &TCon{Name: "S1"}) {
		t.Error("an application of the wrong arity, or a non-alias, matched")
	}
	if NewUnifier().sameAliasApplication(&TCon{Name: "S1"}, &TCon{Name: "S1"}) {
		t.Error("an alias matched with no aliases in scope")
	}
	for _, c := range [][2]Type{{&TVar{Name: "x"}, &TVar{Name: "x"}}, {&TVar2{Name: "x"}, &TVar2{Name: "x"}}, {TInt, &TCon{Name: "int"}}} {
		if !u.sameArgument(c[0], c[1]) {
			t.Errorf("sameArgument(%v, %v)", c[0], c[1])
		}
	}
	if u.sameArgument(&TList{Element: TInt}, &TList{Element: TString}) {
		t.Error("[int] and [string] taken as one argument")
	}
}

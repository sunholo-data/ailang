package types

import (
	"fmt"
	"testing"
)

// M-EFFECT-LATENT-NESTED-CALLBACKS Phase 3 (AC7): erase ⊆ charge.
//
// The #1326 contract says an implicit callback's effects are the CALLER's to
// pay: the callee body erases the implicit tail (site c, implicitRowsAbove) and
// every call site charges it (site a, resolvedCallbackMask). #1718 was the two
// disagreeing. For soundness, any tail a callee body can erase — through the
// parameter binder itself or any binder destructured out of it (tuple, list,
// Option/ADT payload, record field) — must belong to an arrow that the call
// site charges for that parameter.
//
// The check is exhaustive over a generated family of parameter shapes (every
// data container, nested to depth 2, alias-wrapped, with implicit, concrete and
// row-polymorphic arrows), so a new container shape added to one site and not
// the other turns it red.

func invariantArrows() map[string]func(tail string) *TFunc2 {
	return map[string]func(string) *TFunc2{
		"implicit": func(tail string) *TFunc2 { return latentArrow(tail) },
		"concrete": func(tail string) *TFunc2 {
			fn := latentArrow(tail, "IO")
			fn.ImplicitEffectContract, fn.ConcreteEffectContract = false, true
			return fn
		},
		"rowpoly": func(tail string) *TFunc2 {
			fn := latentArrow(tail)
			fn.ImplicitEffectContract = false
			return fn
		},
	}
}

func invariantWrappers() map[string]func(Type) Type {
	return map[string]func(Type) Type{
		"list":   func(t Type) Type { return &TList{Element: t} },
		"array":  func(t Type) Type { return &TArray{Element: t} },
		"map":    func(t Type) Type { return &TMap{Key: TString, Value: t} },
		"tuple":  func(t Type) Type { return &TTuple{Elements: []Type{t, TInt}} },
		"option": func(t Type) Type { return &TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{t}} },
		"record": func(t Type) Type { return &TRecord{Fields: map[string]Type{"f": t, "x": TInt}} },
		"open":   func(t Type) Type { return &TRecordOpen{Fields: map[string]Type{"f": t}} },
		"row2":   func(t Type) Type { return &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"f": t}}} },
	}
}

// dataSubterms lists t and every type reachable from it through data
// positions: the binder types a pattern match over t can introduce.
func dataSubterms(t Type, expand func(Type) Type) []Type {
	out := []Type{t}
	var children []Type
	switch x := expand(t).(type) {
	case *TList:
		children = []Type{x.Element}
	case *TArray:
		children = []Type{x.Element}
	case *TMap:
		children = []Type{x.Key, x.Value}
	case *TTuple:
		children = x.Elements
	case *TApp:
		children = x.Args
	case *TRecord:
		for _, name := range sortedTypeKeys(x.Fields) {
			children = append(children, x.Fields[name])
		}
	case *TRecordOpen:
		for _, name := range sortedTypeKeys(x.Fields) {
			children = append(children, x.Fields[name])
		}
	case *TRecord2:
		for _, name := range sortedTypeKeys(x.Row.Labels) {
			children = append(children, x.Row.Labels[name])
		}
	}
	for _, c := range children {
		out = append(out, dataSubterms(c, expand)...)
	}
	return out
}

func assertEraseWithinCharge(t *testing.T, tc *CoreTypeChecker, name string, param Type) {
	t.Helper()
	charged := map[string]bool{}
	mask := tc.resolvedCallbackMask(&TFunc2{Params: []Type{param}, EffectRow: EmptyEffectRow(), Return: TInt})
	if len(mask) == 1 && mask[0] {
		for _, fn := range tc.dataArrows(param) {
			if isLatentCallback(fn) {
				charged[fn.EffectRow.Tail.Name] = true
			}
		}
	}
	base := NewTypeEnv()
	for _, binder := range dataSubterms(param, tc.effectAliasHead) {
		for tail := range base.Extend("b", binder).implicitRowsAbove(base, tc.effectAliasHead) {
			if !charged[tail] {
				t.Errorf("%s: callee body erases tail %s (binder %s) that no call site charges (mask=%v)", name, tail, binder, mask)
			}
		}
	}
}

func TestLatentEraseWithinCharge(t *testing.T) {
	tc := NewCoreTypeChecker()
	n := 0
	for an, mk := range invariantArrows() {
		tail := "ε_" + an
		tc.RegisterTypeAlias("A_"+an, &TRecord{Fields: map[string]Type{"f": mk(tail)}})
		assertEraseWithinCharge(t, tc, an+"/alias_record", &TCon{Name: "A_" + an})
		for wn, wrap := range invariantWrappers() {
			assertEraseWithinCharge(t, tc, fmt.Sprintf("%s/%s", an, wn), wrap(mk(tail)))
			n++
			for wn2, wrap2 := range invariantWrappers() {
				assertEraseWithinCharge(t, tc, fmt.Sprintf("%s/%s/%s", an, wn2, wn), wrap2(wrap(mk(tail))))
				assertEraseWithinCharge(t, tc, fmt.Sprintf("%s/%s/alias", an, wn2), wrap2(&TCon{Name: "A_" + an}))
				n += 2
			}
		}
	}
	if n < 100 {
		t.Fatalf("generated only %d shapes", n)
	}
}

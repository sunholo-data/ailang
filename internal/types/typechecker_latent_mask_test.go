package types

import (
	"github.com/sunholo-data/ailang/internal/core"
	"testing"
)

func TestLatentMaskPreInstantiation(t *testing.T) {
	tc := NewCoreTypeChecker()
	callback := &TFunc2{Params: []Type{TInt}, Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "e", Kind: EffectRow}}}
	fn := &TFunc2{Params: []Type{callback}, Return: TInt, EffectRow: EmptyEffectRow()}
	scheme := &Scheme{Type: fn, RowVars: []string{"e"}}
	ctx := &InferenceContext{env: NewTypeEnv().ExtendScheme("apply", scheme)}
	mask := tc.preApplicationMask(ctx, &core.Var{Name: "apply"}, fn)
	// The concrete callee CoreTI no longer has an open callback row.
	closed := ApplySubstitution(Substitution{"e": &Row{Kind: EffectRow, Labels: map[string]Type{"IO": Unit()}}}, fn)
	if callbackMask(closed)[0] {
		t.Fatal("control did not close callback")
	}
	tc.publishLatentMask(17, mask)
	published, ok := tc.LatentParamMask(17)
	if !ok || len(published) != 1 || !published[0] {
		t.Fatal("lost pre-instantiation openness")
	}
	published[0] = false
	again, _ := tc.LatentParamMask(17)
	if !again[0] {
		t.Fatal("accessor exposes mutable publication")
	}
	tc.publishLatentMask(18, []bool{false})
	if mask, ok := tc.LatentParamMask(18); !ok || mask[0] {
		t.Fatal("false mask missing")
	}
	if _, ok := tc.LatentParamMask(19); ok {
		t.Fatal("absent mask present")
	}
}

func TestLatentContractSurvivesTypeCopiesAndCache(t *testing.T) {
	cb := &TFunc2{Params: []Type{TInt}, Return: TInt, ConcreteEffectContract: true, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": Unit()}, Tail: &RowVar{Name: "compatibility", Kind: EffectRow}}}
	fn := &TFunc2{Params: []Type{cb}, Return: TInt}
	scheme := &Scheme{Type: fn, RowVars: []string{"compatibility"}}
	data, err := MarshalScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalScheme(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []Type{restored.Type, ApplySubstitution(Substitution{"compatibility": &RowVar{Name: "fresh", Kind: EffectRow}}, fn), fn.Substitute(map[string]Type{"unused": TInt})} {
		if callbackMask(typ)[0] {
			t.Fatal("concrete contract became polymorphic through copy/cache")
		}
	}
}

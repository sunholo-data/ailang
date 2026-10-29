package types

import (
	"github.com/sunholo-data/ailang/internal/typedast"
	"testing"
)

func TestAliasCallbackOwnershipPublication(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.RegisterTypeAlias("Policies", &TRecord{Fields: map[string]Type{"authority": latentArrow("authority"), "appraisal": latentArrow("appraisal")}})
	alias := &TCon{Name: "Policies"}
	base := NewTypeEnv()
	ctx := &InferenceContext{env: base.Extend("policies", alias), baseEnv: base}
	arg := &typedast.TypedVar{TypedExpr: typedast.TypedExpr{Type: alias}, Name: "policies"}
	tc.publishApplicationWithCallee(1, []bool{true}, EmptyEffectRow(), ctx, []typedast.TypedNode{arg})
	record := tc.applicationEffects[1]
	for _, name := range []string{"authority", "appraisal"} {
		if !record.implicitOwned[name] || !record.owned[name] {
			t.Errorf("alias binder must own %s in both sets: implicit=%v owned=%v", name, record.implicitOwned, record.owned)
		}
	}
	tc.substituteApplicationRows(Substitution{
		"authority": &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "fresh_authority", Kind: EffectRow}},
		"appraisal": &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "fresh_appraisal", Kind: EffectRow}},
	})
	got, _ := tc.ApplicationEffects(1)
	if got.CallbackRows[0] == nil || got.CallbackRows[0].Tail != nil || len(got.CallbackRows[0].Labels) != 0 {
		t.Fatalf("forwarded implicit alias must publish a known empty callback row after solver renaming: %#v", got.CallbackRows[0])
	}
}

// Alias expansion only grants ownership to inferred callback contracts. A named
// explicit row is still a real obligation, including conflicts between two rows.
func TestAliasDeclaredTailsRemainUnowned(t *testing.T) {
	tc := NewCoreTypeChecker()
	left, right := latentArrow("declared_left"), latentArrow("declared_right")
	left.ImplicitEffectContract, right.ImplicitEffectContract = false, false
	tc.RegisterTypeAlias("Policies", &TRecord{Fields: map[string]Type{"left": left, "right": right}})
	alias := &TCon{Name: "Policies"}
	base := NewTypeEnv()
	ctx := &InferenceContext{env: base.Extend("p", alias), baseEnv: base}
	arg := &typedast.TypedVar{TypedExpr: typedast.TypedExpr{Type: alias}, Name: "p"}
	tc.publishApplicationWithCallee(2, []bool{true}, EmptyEffectRow(), ctx, []typedast.TypedNode{arg})
	record := tc.applicationEffects[2]
	if len(record.implicitOwned) != 0 {
		t.Fatalf("declared row cannot be erased: %v", record.implicitOwned)
	}
	if record.CallbackRows[0] != nil {
		t.Fatalf("distinct declared tails must retain the fail-loud conflict: %v", record.CallbackRows[0])
	}
}

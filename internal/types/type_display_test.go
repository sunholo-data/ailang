package types

import (
	"strings"
	"testing"
	"time"
)

// A diagnostic renders a large type in part, in bounded time and memory
// (#2363 review: a type error through a doubling alias wrote out 2^n leaves).
// A small type renders exactly as String() does.
func TestDisplayType_BoundedForLargeTypes(t *testing.T) {
	small := &TTuple{Elements: []Type{TInt, &TList{Element: TString}}}
	if got := displayType(small); got != small.String() {
		t.Fatalf("small type rendered %q, want %q", got, small.String())
	}
	var s string
	within(t, 5*time.Second, func() { s = displayType(doublingTuple(TInt, 40)) })
	if len(s) > displayTypeChars+200 || !strings.Contains(s, "shown in part") || !strings.Contains(s, "2,199,023,255,551 nodes") {
		t.Fatalf("large type rendered as %d bytes: %.200s", len(s), s)
	}
	msg := (&TypeCheckError{Message: "m", Expected: doublingTuple(TInt, 40), Actual: TInt}).Error()
	if len(msg) > 2*displayTypeChars+400 {
		t.Fatalf("TypeCheckError rendered its expected type in full: %d bytes", len(msg))
	}
}

// The callback scan expands an alias to fresh nodes each time, so it also
// remembers alias applications by head and argument identity: a doubling
// chain that passes its parameter through is scanned once per alias.
func TestDataArrows_AliasApplicationsExpandedOnce(t *testing.T) {
	aliases, params := map[string]Type{}, map[string][]string{}
	twinChain(aliases, params, "Q", 60)
	u := NewUnifierWithAliasesAndParams(aliases, params)
	fn := &TFunc2{Params: []Type{TInt}, Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "e", Kind: EffectRow}}}
	var arrows []*TFunc2
	within(t, 5*time.Second, func() {
		arrows = dataArrowsWith(&TApp{Constructor: &TCon{Name: "Q59"}, Args: []Type{fn}}, u.expandAlias)
	})
	if len(arrows) != 1 || arrows[0] != fn {
		t.Fatalf("want the one arrow once, got %d", len(arrows))
	}
}

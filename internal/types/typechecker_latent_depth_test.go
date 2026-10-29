package types

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func nestedCallbackType(depth int) Type {
	var typ Type = latentArrow("deep")
	for i := 0; i < depth; i++ {
		typ = &TList{Element: typ}
	}
	return typ
}

func TestDataArrows_DeepCallbacksRemainCharged(t *testing.T) {
	for _, depth := range []int{16, 17, 32, 256} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			tc := NewCoreTypeChecker()
			typ := nestedCallbackType(depth)
			if got := arrowTails(tc.dataArrows(typ)); len(got) != 1 || got[0] != "deep" {
				t.Fatalf("lost callback at depth %d: %v", depth, got)
			}
			assertEraseWithinCharge(t, tc, "deep-list", typ)
		})
	}
}

func TestDataArrows_CyclesRetainReachableCallbacks(t *testing.T) {
	callback := latentArrow("cycle")
	cyclic := &TTuple{}
	cyclic.Elements = []Type{cyclic, callback}
	if got := arrowTails(dataArrowsWith(cyclic, nil)); len(got) != 1 || got[0] != "cycle" {
		t.Fatalf("cyclic tuple callback: %v", got)
	}
	tc := NewCoreTypeChecker()
	tc.RegisterTypeAlias("Loop", &TList{Element: &TTuple{Elements: []Type{&TCon{Name: "Loop"}, callback}}})
	if got := arrowTails(tc.dataArrows(&TCon{Name: "Loop"})); len(got) != 1 || got[0] != "cycle" {
		t.Fatalf("recursive alias callback: %v", got)
	}
}

func TestLatentDataLimitReturnsTypeError(t *testing.T) {
	for _, withConstraints := range []bool{false, true} {
		t.Run(fmt.Sprint(withConstraints), func(t *testing.T) {
			tc := NewCoreTypeChecker()
			typ := nestedCallbackType(maxDataArrowDepth + 1)
			env := NewTypeEnv().Extend("f", &TFunc2{Params: []Type{typ}, Return: TInt, EffectRow: EmptyEffectRow()}).Extend("x", typ)
			expr := &core.App{CoreNode: core.CoreNode{NodeID: 3}, Func: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "f"}, Args: []core.CoreExpr{&core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"}}}
			var err error
			if withConstraints {
				_, _, _, _, err = tc.InferWithConstraints(expr, env)
			} else {
				_, _, err = tc.CheckCoreExpr(expr, env)
			}
			if err == nil || !strings.Contains(err.Error(), "latent callback type nesting exceeds limit") {
				t.Fatalf("want ordinary traversal-limit type error, got %v", err)
			}
		})
	}
}

func TestLatentDataLimitExpandingAliasReturnsTypeError(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.RegisterTypeAliasParams("Grow", []string{"a"})
	tc.RegisterTypeAlias("Grow", &TList{Element: &TApp{Constructor: &TCon{Name: "Grow"}, Args: []Type{&TList{Element: &TVar2{Name: "a"}}}}})
	typ := &TApp{Constructor: &TCon{Name: "Grow"}, Args: []Type{TInt}}
	env := NewTypeEnv().Extend("f", &TFunc2{Params: []Type{typ}, Return: TInt, EffectRow: EmptyEffectRow()}).Extend("x", typ)
	expr := &core.App{CoreNode: core.CoreNode{NodeID: 3}, Func: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "f"}, Args: []core.CoreExpr{&core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"}}}
	_, _, _, _, err := tc.InferWithConstraints(expr, env)
	if err == nil || !strings.Contains(err.Error(), "latent callback type nesting exceeds limit") {
		t.Fatalf("expanding alias must fail as a type error: %v", err)
	}
}

func TestLatentDataRecoveryRethrowsUnrelatedPanic(t *testing.T) {
	const original = "unrelated compiler invariant"
	defer func() {
		if got := recover(); got != original {
			t.Fatalf("recovery hid unrelated panic: %v", got)
		}
	}()
	func() {
		var err error
		defer recoverLatentDataDepth(&err)
		panic(original)
	}()
}

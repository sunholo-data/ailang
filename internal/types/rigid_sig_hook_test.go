package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// The hook a contract/test checker (#2283) calls: an expression over a
// declaration's parameters may not specialise the declaration's rigid
// variables — `ident[a](x: a)` with `requires { x + 1 > 0 }` is TC_RIGID_001.
func TestInferWithRigid_ContractCannotSpecialise(t *testing.T) {
	sig := &RigidSig{Func: "ident", Vars: []RigidVar{{Name: "a@ident", Display: "a", Explicit: true}}}
	scheme := &Scheme{TypeVars: []string{"a"}, Type: &TFunc2{Params: []Type{&TVar2{Name: "a", Kind: Star}}, Return: &TVar2{Name: "a", Kind: Star}, EffectRow: EmptyEffectRow()}}
	inst, ok := sig.Instantiate(scheme).(*TFunc2)
	if !ok || inst.Params[0].String() != "a@ident" {
		t.Fatalf("Instantiate = %v, want the parameter at the rigid name", sig.Instantiate(scheme))
	}
	id := uint64(9000)
	node := func() core.CoreNode { id++; return core.CoreNode{NodeID: id} }
	x := func() core.CoreExpr { return &core.Var{CoreNode: node(), Name: "x"} }
	lit := func(n int) core.CoreExpr { return &core.Lit{CoreNode: node(), Kind: core.IntLit, Value: n} }
	bin := func(op core.IntrinsicOp, l, r core.CoreExpr) core.CoreExpr {
		return &core.Intrinsic{CoreNode: node(), Op: op, Args: []core.CoreExpr{l, r}}
	}

	for _, c := range []struct {
		name string
		expr core.CoreExpr
		kind string // "" = accepted
	}{
		{"x + 1 > 0 needs a literal at a", bin(core.OpGt, bin(core.OpAdd, x(), lit(1)), lit(0)), "literal"},
		{"x == x keeps a general", bin(core.OpEq, x(), x()), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			tc := NewCoreTypeChecker()
			env := NewTypeEnvWithBuiltins().Extend("x", inst.Params[0])
			_, _, _, _, err := tc.InferWithRigid(c.expr, env, sig)
			if c.kind == "" {
				if err != nil {
					t.Fatalf("rejected: %v", err)
				}
				return
			}
			var re *RigidSigError
			if !errors.As(err, &re) || re.Kind() != c.kind {
				t.Fatalf("want TC_RIGID_001 kind %q, got %v", c.kind, err)
			}
			// The same expression without the hook is accepted: the hook is the rule.
			if _, _, _, _, err := NewCoreTypeChecker().InferWithConstraints(c.expr, env); err != nil {
				t.Fatalf("control: plain inference rejected it too: %v", err)
			}
		})
	}
}

// A signature variable that resolves to a variable the binding's scheme does
// not quantify (owned by an enclosing scope) is not general either.
func TestFindRigidViolation_Escape(t *testing.T) {
	sig := &RigidSig{Func: "f", Vars: []RigidVar{{Name: "a@f", Display: "a"}}}
	same := func(t Type) Type { return t }
	if v := findRigidViolation(sig, map[string]bool{"a@f": true}, nil, same); v != nil {
		t.Fatalf("quantified variable reported: %+v", v)
	}
	v := findRigidViolation(sig, map[string]bool{}, nil, same)
	if v == nil || v.kind != "escape" {
		t.Fatalf("unquantified variable: got %+v, want escape", v)
	}
	if err := v.toError(sig); err == nil || !strings.Contains(err.Error(), "outside `f`") {
		t.Fatalf("message: %v", err)
	}
}

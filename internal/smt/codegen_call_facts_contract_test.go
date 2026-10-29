package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func finiteSpec() ContractSpec {
	return ContractSpec{FuncName: "bounded", Params: []FunctionParam{{Name: "n", Type: types.TInt}}, ReturnSort: "Int",
		Requires: []*core.Contract{{Kind: core.RequiresKind, Expr: builtinApp("ge_Int", &core.Var{Name: "n"}, intLit(0))}},
		Ensures:  []*core.Contract{{Kind: core.EnsuresKind, Expr: builtinApp("ge_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"})}}}
}

func TestContractCallIdentityAndGuard(t *testing.T) {
	spec := finiteSpec()
	a, err := EncodeCalleeByContract(spec, []core.CoreExpr{intLit(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	same, err := EncodeCalleeByContract(spec, []core.CoreExpr{intLit(1)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	other, err := EncodeCalleeByContract(spec, []core.CoreExpr{intLit(2)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.ResultConst != same.ResultConst || a.ResultConst == other.ResultConst {
		t.Fatalf("call identity: %q %q %q", a.ResultConst, same.ResultConst, other.ResultConst)
	}
	if !strings.Contains(a.SMTLib, "declare-fun") || strings.Contains(a.SMTLib, "forall") || !strings.Contains(a.SMTLib, "(=> (let ") {
		t.Fatal(a.SMTLib)
	}
}

func TestContractCallArityAndUnknownRequires(t *testing.T) {
	spec := finiteSpec()
	for _, args := range [][]core.CoreExpr{nil, {intLit(1), intLit(2)}} {
		if _, err := EncodeCalleeByContract(spec, args, 0); err == nil {
			t.Fatalf("accepted arity %d", len(args))
		}
	}
	spec.Requires[0].Expr = &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}
	d, err := EncodeCalleeByContract(spec, []core.CoreExpr{intLit(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(d.SMTLib, "assert") {
		t.Fatalf("unknown requires must contribute no ensures: %s", d.SMTLib)
	}
}

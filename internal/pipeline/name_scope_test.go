package pipeline

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// nameScopeSource: every pass that resolves a variable by NAME must respect
// lexical scope. A lambda parameter, a pattern variable or a letrec member
// shadows an outer binding of its name, and an outer binding comes back after
// an inner let's body.
//   - Specializer: `\f. f(3)` was rewritten into a call of the outer
//     `let f = \x. x` (3 instead of 4, 13, ...).
//   - OpLowerer: a pattern variable of type a took the type of an earlier
//     `let x = 1`, lowering == to eq_Int on strings (crash), / to float
//     division on ints.
//   - VarResolver: same name-keyed map; a let deleted the outer entry.
const nameScopeSource = `module main

func cmp[a](o: Option[a], k: a) -> bool {
  let x = 1;
  match o { Some(x) => x == k, None => false }
}

func eqAfter[a](k: a, m: a) -> bool {
  let x = k;
  let s = { let x = 2; x };
  x == m
}

func dv[a](x: a, m: a) -> a {
  let s = { let x = 2.5; x };
  x / m
}

export pure func main() -> string {
  let f = \x. x;
  let g = \f. f(3);
  let h = \o. match o { Some(f) => f(3), None => 0 };
  let tp = \x. match x { (x, y) => x == y };
  let lr = { letrec f = \n. if n <= 0 then 10 else f(n - 1); f(2) };
  "${show(cmp(Some("s"), "s"))} ${show(eqAfter("a", "a"))} ${show(dv(7, 2))} ${show(g(\z. z + 1))} ${show(h(Some(\z. z + 10)))} ${show(tp(("a", "a")))} ${show(lr)} ${show(f(5))}"
}
`

func TestNameResolvingPassesRespectScope(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("main.ail", []byte(nameScopeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "main.ail"})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, result.Errors)
	}
	if got, want := executePipelineMain(t, result), "true true 3 4 13 true 10 5"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The scoping helpers restore exactly what was there before.
func TestNameScopeHelpersRestore(t *testing.T) {
	r := NewVarResolver(types.NewCoreTypeInfo())
	r.bindings["x"] = types.TInt
	restore := r.shadow("x", "y")
	if _, ok := r.bindings["x"]; ok {
		t.Fatal("VarResolver.shadow left x visible")
	}
	r.bindings["y"] = types.TString
	restore()
	if r.bindings["x"] != types.TInt {
		t.Fatal("VarResolver.shadow did not restore x")
	}
	if _, ok := r.bindings["y"]; ok {
		t.Fatal("VarResolver.shadow left the inner y behind")
	}

	l := NewOpLowerer(types.NewTypeEnv(), types.NewCoreTypeInfo())
	outer := &core.Lit{Kind: core.IntLit, Value: int64(1)}
	l.bindings["x"] = outer
	restore = l.shadow("x")
	l.bindings["x"] = &core.Lit{Kind: core.StringLit, Value: "s"}
	restore()
	if l.bindings["x"] != core.CoreExpr(outer) {
		t.Fatal("OpLowerer.shadow did not restore the outer x")
	}

	env := map[string]types.Type{"f": types.TInt, "g": types.TInt}
	bindings := map[string]core.CoreExpr{"f": outer, "g": outer}
	shadowNames(env, bindings, []string{"f"})
	if _, ok := bindings["f"]; ok || env["g"] == nil || bindings["g"] == nil {
		t.Fatalf("shadowNames: env=%v bindings=%v", env, bindings)
	}
	if names := recNames([]core.RecBinding{{Name: "a"}, {Name: "b"}}); len(names) != 2 || names[1] != "b" {
		t.Fatalf("recNames = %v", names)
	}
}

// scopeProbe: let x = 1 in (\x. x == k) — the lambda's x is a type variable,
// the outer x is int.
func scopeProbe() (*core.Let, *core.Var, *core.Intrinsic, types.CoreTypeInfo) {
	lit := &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(1)}
	inner := &core.Var{CoreNode: core.CoreNode{NodeID: 5}, Name: "x"}
	eq := &core.Intrinsic{CoreNode: core.CoreNode{NodeID: 4}, Op: core.OpEq, Args: []core.CoreExpr{inner, &core.Var{CoreNode: core.CoreNode{NodeID: 6}, Name: "k"}}}
	lam := &core.Lambda{CoreNode: core.CoreNode{NodeID: 3}, Params: []string{"x"}, Body: eq}
	let := &core.Let{CoreNode: core.CoreNode{NodeID: 1}, Name: "x", Value: lit, Body: lam}
	cti := types.NewCoreTypeInfo()
	cti.Set(2, types.TInt)
	cti.Set(5, &types.TVar2{Name: "a", Kind: types.Star})
	cti.Set(6, &types.TVar2{Name: "a", Kind: types.Star})
	return let, inner, eq, cti
}

// VarResolver must not copy the outer let's int onto the lambda's x.
func TestVarResolverLambdaParamShadows(t *testing.T) {
	let, inner, _, cti := scopeProbe()
	NewVarResolver(cti).Resolve(&core.Program{Decls: []core.CoreExpr{let}})
	if got, _ := cti.Get(inner.ID()); got == types.TInt {
		t.Fatal("the lambda parameter x took the outer let's type int")
	}
}

// OpLowerer must not choose the int instance for the lambda's x == k.
func TestOpLowererLambdaParamShadows(t *testing.T) {
	let, _, _, cti := scopeProbe()
	l := NewOpLowerer(types.NewTypeEnv(), cti)
	if out := l.lowerExpr(let).String(); strings.Contains(out, "eq_Int") {
		t.Fatalf("== on the lambda parameter x was typed by the outer `let x = 1`: %s", out)
	}
	if _, ok := l.bindings["x"]; ok {
		t.Fatal("the local let x is still tracked after its body")
	}
}

// After an inner let's body the OUTER binding is visible again: a later read
// of x (still a type variable) gets the outer let's int. It used to be
// deleted, so the outer x lost its resolution after any inner `let x`.
func TestVarResolverRestoresOuterAfterInnerLet(t *testing.T) {
	tv := func() types.Type { return &types.TVar2{Name: "a", Kind: types.Star} }
	outerLit := &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(1)}
	innerLit := &core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.StringLit, Value: "s"}
	innerRead := &core.Var{CoreNode: core.CoreNode{NodeID: 5}, Name: "x"}
	laterRead := &core.Var{CoreNode: core.CoreNode{NodeID: 7}, Name: "x"}
	inner := &core.Let{CoreNode: core.CoreNode{NodeID: 3}, Name: "x", Value: innerLit, Body: innerRead}
	prog := &core.Let{CoreNode: core.CoreNode{NodeID: 1}, Name: "x", Value: outerLit,
		Body: &core.Let{CoreNode: core.CoreNode{NodeID: 6}, Name: "s", Value: inner, Body: laterRead}}
	cti := types.NewCoreTypeInfo()
	cti.Set(2, types.TInt)
	cti.Set(4, types.TString)
	cti.Set(5, tv())
	cti.Set(7, tv())
	NewVarResolver(cti).Resolve(&core.Program{Decls: []core.CoreExpr{prog}})
	if got, _ := cti.Get(innerRead.ID()); got != types.TString {
		t.Errorf("inner read of x: %v, want string", got)
	}
	if got, _ := cti.Get(laterRead.ID()); got != types.TInt {
		t.Errorf("read of the outer x after the inner let: %v, want int", got)
	}
}

// A body that calls a member of a mutually recursive local group must not
// resolve the name to an outer polymorphic lambda of the same name and be
// rewritten into a call of it.
func TestSpecializerLetRecGroupShadowsOuter(t *testing.T) {
	tv := &types.TVar2{Name: "a", Kind: types.Star}
	v := func(id uint64, n string) *core.Var { return &core.Var{CoreNode: core.CoreNode{NodeID: id}, Name: n} }
	outer := &core.Lambda{CoreNode: core.CoreNode{NodeID: 10}, Params: []string{"x"}, Body: v(11, "x")}
	a := &core.Lambda{CoreNode: core.CoreNode{NodeID: 20}, Params: []string{"n"},
		Body: &core.App{CoreNode: core.CoreNode{NodeID: 21}, Func: v(22, "b"), Args: []core.CoreExpr{v(23, "n")}}}
	b := &core.Lambda{CoreNode: core.CoreNode{NodeID: 30}, Params: []string{"m"},
		Body: &core.App{CoreNode: core.CoreNode{NodeID: 31}, Func: v(32, "a"), Args: []core.CoreExpr{v(33, "m")}}}
	call := &core.App{CoreNode: core.CoreNode{NodeID: 41}, Func: v(42, "a"),
		Args: []core.CoreExpr{&core.Lit{CoreNode: core.CoreNode{NodeID: 43}, Kind: core.IntLit, Value: int64(3)}}}
	group := &core.LetRec{CoreNode: core.CoreNode{NodeID: 40}, Bindings: []core.RecBinding{{Name: "a", Value: a}, {Name: "b", Value: b}}, Body: call}
	cti := types.NewCoreTypeInfo()
	cti.Set(10, &types.TFunc2{Params: []types.Type{tv}, Return: tv})
	cti.Set(43, types.TInt)
	s := NewSpecializer(&cti)
	out, err := s.specializeExpr(group, map[string]types.Type{}, map[string]core.CoreExpr{"a": outer})
	if err != nil {
		t.Fatal(err)
	}
	body := out.(*core.LetRec).Body.(*core.App)
	if _, ok := body.Func.(*core.Var); !ok {
		t.Fatalf("the call of group member a was rewritten into %s", body.Func)
	}
}

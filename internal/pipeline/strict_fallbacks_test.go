package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// --- helpers for building synthetic Core -----------------------------------

func strLit(s string) *core.Lit { return &core.Lit{Kind: core.StringLit, Value: s} }
func intLit(i int64) *core.Lit  { return &core.Lit{Kind: core.IntLit, Value: i} }
func boolLit(b bool) *core.Lit  { return &core.Lit{Kind: core.BoolLit, Value: b} }
func emptyList() *core.List     { return &core.List{} }
func varRef(n string) *core.Var { return &core.Var{Name: n} }

func joApp(arg core.CoreExpr) *core.App {
	return &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/json", Name: "jo"}},
		Args: []core.CoreExpr{arg},
	}
}

// --- resolveAtomic ----------------------------------------------------------

func TestResolveAtomic_ThroughLetBinding(t *testing.T) {
	env := map[string]core.CoreExpr{"t": emptyList()}
	got := resolveAtomic(varRef("t"), env, maxResolveDepth)
	if _, ok := got.(*core.List); !ok {
		t.Fatalf("expected Var{t} to resolve to a List, got %T", got)
	}
}

func TestResolveAtomic_Chained(t *testing.T) {
	// t1 = "", t2 = t1  => resolving t2 yields the "" literal.
	env := map[string]core.CoreExpr{
		"t1": strLit(""),
		"t2": varRef("t1"),
	}
	got := resolveAtomic(varRef("t2"), env, maxResolveDepth)
	lit, ok := got.(*core.Lit)
	if !ok || lit.Kind != core.StringLit || lit.Value.(string) != "" {
		t.Fatalf("expected chained resolve to empty string, got %#v", got)
	}
}

func TestResolveAtomic_UnboundVarUnchanged(t *testing.T) {
	env := map[string]core.CoreExpr{}
	got := resolveAtomic(varRef("x"), env, maxResolveDepth)
	if v, ok := got.(*core.Var); !ok || v.Name != "x" {
		t.Fatalf("expected unbound Var to be returned unchanged, got %#v", got)
	}
}

func TestResolveAtomic_CycleTerminates(t *testing.T) {
	// a = b, b = a — must not loop.
	env := map[string]core.CoreExpr{
		"a": varRef("b"),
		"b": varRef("a"),
	}
	_ = resolveAtomic(varRef("a"), env, maxResolveDepth) // just must return
}

// --- isEmptyValue: positives ------------------------------------------------

func TestIsEmptyValue_DirectEmptyString(t *testing.T) {
	if empty, _ := isEmptyValue(strLit(""), nil, maxResolveDepth); !empty {
		t.Fatal("expected empty string to be an empty value")
	}
}

func TestIsEmptyValue_ResolvedEmptyList(t *testing.T) {
	// The ANF case: Ok([]) => let t = [] in Ok(t); Args[0] is Var{t}.
	env := map[string]core.CoreExpr{"t": emptyList()}
	empty, kind := isEmptyValue(varRef("t"), env, maxResolveDepth)
	if !empty {
		t.Fatal("expected resolved empty list to be an empty value")
	}
	if kind != "empty list" {
		t.Errorf("unexpected kind: %q", kind)
	}
}

func TestIsEmptyValue_JoEmptyViaRegistry(t *testing.T) {
	// The motivating case: Ok(jo([])) => let t = jo([]) in Ok(t).
	env := map[string]core.CoreExpr{"t": joApp(emptyList())}
	empty, kind := isEmptyValue(varRef("t"), env, maxResolveDepth)
	if !empty {
		t.Fatal("expected jo([]) via registry to be an empty value")
	}
	if kind != "empty std/json.jo()" {
		t.Errorf("unexpected kind: %q", kind)
	}
}

func TestIsEmptyValue_JoEmptyViaResolvedArg(t *testing.T) {
	// jo(t) where t binds [] — the arg itself is an ANF Var.
	env := map[string]core.CoreExpr{"kvs": emptyList()}
	app := joApp(varRef("kvs"))
	empty, _ := isEmptyValue(app, env, maxResolveDepth)
	if !empty {
		t.Fatal("expected jo(t=[]) to resolve through the arg and flag")
	}
}

func TestIsEmptyValue_EmptyRecord(t *testing.T) {
	if empty, kind := isEmptyValue(&core.Record{Fields: map[string]core.CoreExpr{}}, nil, maxResolveDepth); !empty || kind != "empty record" {
		t.Fatalf("expected empty record, got empty=%v kind=%q", empty, kind)
	}
}

func TestIsEmptyValue_AllZeroRecord(t *testing.T) {
	rec := &core.Record{Fields: map[string]core.CoreExpr{
		"name":   strLit(""),
		"age":    intLit(0),
		"active": boolLit(false),
	}}
	empty, kind := isEmptyValue(rec, nil, maxResolveDepth)
	if !empty || kind != "all-zero record" {
		t.Fatalf("expected all-zero record, got empty=%v kind=%q", empty, kind)
	}
}

// --- isEmptyValue: negatives ------------------------------------------------

func TestIsEmptyValue_NonEmptyString(t *testing.T) {
	if empty, _ := isEmptyValue(strLit("real"), nil, maxResolveDepth); empty {
		t.Fatal("non-empty string must NOT flag")
	}
}

func TestIsEmptyValue_NonZeroScalarNotFlagged(t *testing.T) {
	// Bare Ok(0) is a legitimate success value; not flagged by isEmptyValue.
	if empty, _ := isEmptyValue(intLit(0), nil, maxResolveDepth); empty {
		t.Fatal("bare Ok(0) must NOT flag (legitimate success value)")
	}
}

func TestIsEmptyValue_MixedRecordNotFlagged(t *testing.T) {
	// {name: <realVar>, age: 0} — a real field means it is NOT all-zero.
	rec := &core.Record{Fields: map[string]core.CoreExpr{
		"name": varRef("realName"), // unbound → not a zero literal
		"age":  intLit(0),
	}}
	if empty, _ := isEmptyValue(rec, nil, maxResolveDepth); empty {
		t.Fatal("mixed record with a real field must NOT flag")
	}
}

func TestIsEmptyValue_JoNonEmptyNotFlagged(t *testing.T) {
	// jo([kv]) — non-empty arg list must NOT flag.
	app := joApp(&core.List{Elements: []core.CoreExpr{varRef("kv")}})
	if empty, _ := isEmptyValue(app, nil, maxResolveDepth); empty {
		t.Fatal("jo([kv]) must NOT flag")
	}
}

func TestIsEmptyValue_UserLocalJoNotFlagged(t *testing.T) {
	// A user-defined LOCAL `jo` is a plain core.Var head (never VarGlobal).
	// App{Func: Var{jo}, Args:[[]]} must NOT hit the registry.
	app := &core.App{Func: varRef("jo"), Args: []core.CoreExpr{emptyList()}}
	if empty, _ := isEmptyValue(app, nil, maxResolveDepth); empty {
		t.Fatal("user-local jo([]) (plain Var head) must NOT flag")
	}
}

func TestIsEmptyValue_NonEmptyListNotFlagged(t *testing.T) {
	lst := &core.List{Elements: []core.CoreExpr{varRef("x")}}
	if empty, _ := isEmptyValue(lst, nil, maxResolveDepth); empty {
		t.Fatal("non-empty list must NOT flag")
	}
}

// --- isZeroField ------------------------------------------------------------

func TestIsZeroField_Scalars(t *testing.T) {
	cases := []struct {
		name string
		v    core.CoreExpr
		want bool
	}{
		{"empty string", strLit(""), true},
		{"real string", strLit("x"), false},
		{"int zero", intLit(0), true},
		{"int nonzero", intLit(3), false},
		{"bool false", boolLit(false), true},
		{"bool true", boolLit(true), false},
		{"empty list", emptyList(), true},
		{"unbound var", varRef("y"), false},
	}
	for _, tc := range cases {
		if got := isZeroField(tc.v, nil, maxResolveDepth); got != tc.want {
			t.Errorf("%s: isZeroField = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// --- depth guard ------------------------------------------------------------

func TestIsEmptyValue_DepthBounded(t *testing.T) {
	// Deep chain of let bindings must terminate.
	env := map[string]core.CoreExpr{}
	prev := core.CoreExpr(emptyList())
	for i := 0; i < 200; i++ {
		name := "v" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		env[name] = prev
		prev = varRef(name)
	}
	// Just needs to return without hanging.
	_, _ = isEmptyValue(prev, env, maxResolveDepth)
}

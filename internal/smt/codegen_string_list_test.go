package smt

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// --- String builtin encoding tests ---

func TestEncodeStringBuiltin_ConcatString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "concat_String"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "a"},
			&core.Var{Name: "b"},
		},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.++ a b)" {
		t.Errorf("got %q, want %q", got, "(str.++ a b)")
	}
}

func TestEncodeStringBuiltin_EqString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(= a b)" {
		t.Errorf("got %q, want %q", got, "(= a b)")
	}
}

func TestEncodeStringBuiltin_LtString(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "lt_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.< a b)" {
		t.Errorf("got %q, want %q", got, "(str.< a b)")
	}
}

func TestEncodeStringBuiltin_GtString_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "gt_String"}},
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// gt(a,b) → (str.< b a) with flipped args
	if got != "(str.< b a)" {
		t.Errorf("got %q, want %q", got, "(str.< b a)")
	}
}

func TestEncodeStringBuiltin_StrLen(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_len"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.len s)" {
		t.Errorf("got %q, want %q", got, "(str.len s)")
	}
}

func TestEncodeStringBuiltin_StrFind_AppendsZero(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_find"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "t"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_find(s,t) → (str.indexof s t 0)
	if got != "(str.indexof s t 0)" {
		t.Errorf("got %q, want %q", got, "(str.indexof s t 0)")
	}
}

func TestEncodeStringBuiltin_StrSlice_SubstrMode(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_slice"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "s"},
			&core.Var{Name: "start"},
			&core.Var{Name: "end"},
		},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_slice(s, start, end) → (str.substr s start (- end start))
	want := "(str.substr s start (- end start))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeStringBuiltin_StartsWith_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_startsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "prefix"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_startsWith(s, prefix) → (str.prefixof prefix s)
	if got != "(str.prefixof prefix s)" {
		t.Errorf("got %q, want %q", got, "(str.prefixof prefix s)")
	}
}

func TestEncodeStringBuiltin_EndsWith_FlippedArgs(t *testing.T) {
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_endsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "suffix"}},
	}
	got, err := EncodeExpr(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// _str_endsWith(s, suffix) → (str.suffixof suffix s)
	if got != "(str.suffixof suffix s)" {
		t.Errorf("got %q, want %q", got, "(str.suffixof suffix s)")
	}
}

func TestEncodeIntrinsic_OpConcat(t *testing.T) {
	intr := &core.Intrinsic{
		Op:   core.OpConcat,
		Args: []core.CoreExpr{&core.Var{Name: "a"}, &core.Var{Name: "b"}},
	}
	got, err := EncodeExpr(intr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.++ a b)" {
		t.Errorf("got %q, want %q", got, "(str.++ a b)")
	}
}

// --- List encoding tests ---

func TestEncodeExpr_EmptyList(t *testing.T) {
	expr := &core.List{Elements: []core.CoreExpr{}}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(as seq.empty (Seq Int))" {
		t.Errorf("got %q, want %q", got, "(as seq.empty (Seq Int))")
	}
}

func TestEncodeExpr_SingleElementList(t *testing.T) {
	expr := &core.List{Elements: []core.CoreExpr{
		&core.Lit{Kind: core.IntLit, Value: int64(42)},
	}}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.unit 42)" {
		t.Errorf("got %q, want %q", got, "(seq.unit 42)")
	}
}

func TestEncodeExpr_MultiElementList(t *testing.T) {
	expr := &core.List{Elements: []core.CoreExpr{
		&core.Lit{Kind: core.IntLit, Value: int64(1)},
		&core.Lit{Kind: core.IntLit, Value: int64(2)},
		&core.Lit{Kind: core.IntLit, Value: int64(3)},
	}}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(seq.++ (seq.unit 1) (seq.unit 2) (seq.unit 3))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeExpr_ListConcat(t *testing.T) {
	// concat_List(xs, ys) → (seq.++ xs ys)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "concat_List"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}, &core.Var{Name: "ys"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.++ xs ys)" {
		t.Errorf("got %q, want %q", got, "(seq.++ xs ys)")
	}
}

func TestEncodeExpr_ListCons(t *testing.T) {
	// :: (cons) from std/list module → (seq.++ (seq.unit x) xs)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "::"}},
		Args: []core.CoreExpr{&core.Var{Name: "x"}, &core.Var{Name: "xs"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.++ (seq.unit x) xs)" {
		t.Errorf("got %q, want %q", got, "(seq.++ (seq.unit x) xs)")
	}
}

func TestEncodeExpr_ListLength(t *testing.T) {
	// _list_length(xs) → (seq.len xs)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.len xs)" {
		t.Errorf("got %q, want %q", got, "(seq.len xs)")
	}
}

func TestEncodeExpr_ListHead(t *testing.T) {
	// _list_head(xs) → (seq.nth xs 0)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_head"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.nth xs 0)" {
		t.Errorf("got %q, want %q", got, "(seq.nth xs 0)")
	}
}

func TestEncodeExpr_ListNth(t *testing.T) {
	// _list_nth(xs, i) → (seq.nth xs i)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_nth"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}, &core.Var{Name: "i"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.nth xs i)" {
		t.Errorf("got %q, want %q", got, "(seq.nth xs i)")
	}
}

// --- Stdlib SMT transparency tests ---

func TestEncodeStdlib_StringLength(t *testing.T) {
	// std/string.length(s) → (str.len s)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "length"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.len s)" {
		t.Errorf("got %q, want %q", got, "(str.len s)")
	}
}

func TestEncodeStdlib_StringStartsWith(t *testing.T) {
	// std/string.startsWith(s, prefix) → (str.prefixof prefix s) (FlipArgs)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "startsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "prefix"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.prefixof prefix s)" {
		t.Errorf("got %q, want %q", got, "(str.prefixof prefix s)")
	}
}

func TestEncodeStdlib_StringEndsWith(t *testing.T) {
	// std/string.endsWith(s, suffix) → (str.suffixof suffix s) (FlipArgs)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "endsWith"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "suffix"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.suffixof suffix s)" {
		t.Errorf("got %q, want %q", got, "(str.suffixof suffix s)")
	}
}

func TestEncodeStdlib_StringContains(t *testing.T) {
	// std/string.contains(s, sub) → (str.contains s sub)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "contains"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "sub"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.contains s sub)" {
		t.Errorf("got %q, want %q", got, "(str.contains s sub)")
	}
}

func TestEncodeStdlib_StringFind(t *testing.T) {
	// std/string.find(s, t) → (str.indexof s t 0) (AppendZero)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "find"}},
		Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "t"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(str.indexof s t 0)" {
		t.Errorf("got %q, want %q", got, "(str.indexof s t 0)")
	}
}

func TestEncodeStdlib_StringSubstring(t *testing.T) {
	// std/string.substring(s, start, end) → (str.substr s start (- end start))
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "substring"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "s"},
			&core.Var{Name: "start"},
			&core.Var{Name: "end"},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(str.substr s start (- end start))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- New list builtin encoding tests (M3_RECURSIVE_LIST_OPS) ---

func TestEncodeExpr_ListContains(t *testing.T) {
	// _list_contains(xs, elem) → (seq.contains xs (seq.unit elem))
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_contains"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}, &core.Var{Name: "elem"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(seq.contains xs (seq.unit elem))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeExpr_ListExtract(t *testing.T) {
	// _list_extract(xs, offset, length) → (seq.extract xs offset length)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_extract"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "xs"},
			&core.Var{Name: "offset"},
			&core.Var{Name: "length"},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(seq.extract xs offset length)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeExpr_ListContains_WithLiteral(t *testing.T) {
	// _list_contains(xs, 2) → (seq.contains xs (seq.unit 2))
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_contains"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "xs"},
			&core.Lit{Kind: core.IntLit, Value: int64(2)},
		},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(seq.contains xs (seq.unit 2))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeStdlib_ListContains(t *testing.T) {
	// std/list.contains(xs, elem) → (seq.contains xs (seq.unit elem))
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "contains"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}, &core.Var{Name: "elem"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(seq.contains xs (seq.unit elem))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeStdlib_ListLength(t *testing.T) {
	// std/list.length(xs) → (seq.len xs)
	expr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "length"}},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(seq.len xs)" {
		t.Errorf("got %q, want %q", got, "(seq.len xs)")
	}
}

func TestEncodeStdlib_CurriedStringSubstring(t *testing.T) {
	// Curried: App(App(VarGlobal(std/string.substring), [s, start]), [end])
	expr := &core.App{
		Func: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "substring"}},
			Args: []core.CoreExpr{&core.Var{Name: "s"}, &core.Var{Name: "start"}},
		},
		Args: []core.CoreExpr{&core.Var{Name: "end"}},
	}
	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(str.substr s start (- end start))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Numeric conversion builtin tests (E1+E2) ---

func TestEncodeApp_IntToFloat(t *testing.T) {
	// intToFloat(x) → (to_real x)
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "intToFloat"}},
		Args: []core.CoreExpr{&core.Var{Name: "x"}},
	}
	got, err := encodeApp(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(to_real x)" {
		t.Errorf("got %q, want %q", got, "(to_real x)")
	}
}

func TestEncodeApp_FloatToInt(t *testing.T) {
	// floatToInt(x) → (to_int x)
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "floatToInt"}},
		Args: []core.CoreExpr{&core.Var{Name: "x"}},
	}
	got, err := encodeApp(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(to_int x)" {
		t.Errorf("got %q, want %q", got, "(to_int x)")
	}
}

func TestEncodeApp_IntToFloat_Underscore(t *testing.T) {
	// _int_to_float(x) → (to_real x)
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_int_to_float"}},
		Args: []core.CoreExpr{&core.Var{Name: "n"}},
	}
	got, err := encodeApp(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(to_real n)" {
		t.Errorf("got %q, want %q", got, "(to_real n)")
	}
}

func TestEncodeApp_IntToFloat_WithLiteral(t *testing.T) {
	// intToFloat(42) → (to_real 42)
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "intToFloat"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(42)}},
	}
	got, err := encodeApp(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(to_real 42)" {
		t.Errorf("got %q, want %q", got, "(to_real 42)")
	}
}

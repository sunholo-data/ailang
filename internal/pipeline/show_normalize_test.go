package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-SMT-INTERP-SHOW: the interpolation desugar wraps every "${...}" hole in
// `show(...)` at PARSE time, before any type is known. For a hole that is already
// a string, `show` is the identity; for bool and int it has an exact encodable
// equivalent. ShowNormalizer removes it where the argument type says it is safe,
// so string-building functions land in the SMT-decidable fragment.
//
// The design's load-bearing correctness premise is `show(s: string) ≡ s` (design
// doc V1, verified against internal/builtins/show.go:118-120 and at runtime).

// compileForNormalize runs the full pipeline and returns the post-pass Core and
// its type info. The pass is wired into the pipeline, so what comes back is
// already normalized.
func compileForNormalize(t *testing.T, name, src string) (*core.Program, types.CoreTypeInfo) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	res, err := Run(Config{RelaxModules: true, NoCache: true}, Source{Code: src, Filename: p})
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("compile %s: %v", name, res.Errors)
	}
	return res.Artifacts.Core, res.Artifacts.CoreTI
}

// countShowCalls returns how many `$builtin.show` applications survive.
func countShowCalls(prog *core.Program) int {
	n := 0
	for _, d := range prog.Decls {
		walkCore(d, func(e core.CoreExpr) {
			if app, ok := e.(*core.App); ok {
				if vg, ok := app.Func.(*core.VarGlobal); ok &&
					vg.Ref.Module == "$builtin" && vg.Ref.Name == "show" {
					n++
				}
			}
		})
	}
	return n
}

// TestShowNormalize_StringHoleElided: `"${s}"` with s:string must leave NO show
// call behind — the whole point of the milestone.
func TestShowNormalize_StringHoleElided(t *testing.T) {
	prog, _ := compileForNormalize(t, "strhole", `module strhole
export func f(s: string) -> string ! {} { "X${s}Y" }
`)
	if got := countShowCalls(prog); got != 0 {
		t.Fatalf("string hole: %d show calls survived, want 0", got)
	}
}

// TestShowNormalize_BoolHoleRewritten: bool holes become If(x,"true","false"),
// and EVERY minted node carries a CoreTypeInfo entry — ValidateCoreTypeInfo
// requires one for every node it walks, Lit included (design doc V19).
func TestShowNormalize_BoolHoleRewritten(t *testing.T) {
	prog, ti := compileForNormalize(t, "boolhole", `module boolhole
export func f(b: bool) -> string ! {} { "v=${b}" }
`)
	if got := countShowCalls(prog); got != 0 {
		t.Fatalf("bool hole: %d show calls survived, want 0", got)
	}

	var found *core.If
	for _, d := range prog.Decls {
		walkCore(d, func(e core.CoreExpr) {
			ifn, ok := e.(*core.If)
			if !ok {
				return
			}
			th, tok := ifn.Then.(*core.Lit)
			el, eok := ifn.Else.(*core.Lit)
			if tok && eok && th.Value == "true" && el.Value == "false" {
				found = ifn
			}
		})
	}
	if found == nil {
		t.Fatal("bool hole: no If(x, \"true\", \"false\") in the normalized Core")
	}
	for _, n := range []core.CoreExpr{found, found.Then, found.Else} {
		ty, has := ti.Get(n.ID())
		if !has {
			t.Errorf("minted node %d (%T) has no CoreTypeInfo entry", n.ID(), n)
			continue
		}
		if !isTConNamed(ty, "string") {
			t.Errorf("minted node %d (%T): type %v, want string", n.ID(), n, ty)
		}
	}
}

// TestShowNormalize_UnsupportedTypeUntouched: a float hole is genuinely
// unencodable, so `show` must survive. This is the honest residue, NOT a bug.
func TestShowNormalize_UnsupportedTypeUntouched(t *testing.T) {
	prog, _ := compileForNormalize(t, "floathole", `module floathole
export func f(x: float) -> string ! {} { "v=${x}" }
`)
	if got := countShowCalls(prog); got != 1 {
		t.Fatalf("float hole: %d show calls, want exactly 1 (must be left alone)", got)
	}
}

// TestShowNormalize_ReusedArgKeepsItsNode: the argument is spliced in unchanged.
// Re-registering it would be wrong — it already has a valid ID and entry.
func TestShowNormalize_ReusedArgKeepsItsNode(t *testing.T) {
	prog, ti := compileForNormalize(t, "reuse", `module reuse
export func f(b: bool) -> string ! {} { "v=${b}" }
`)
	var cond core.CoreExpr
	for _, d := range prog.Decls {
		walkCore(d, func(e core.CoreExpr) {
			if ifn, ok := e.(*core.If); ok {
				if th, tok := ifn.Then.(*core.Lit); tok && th.Value == "true" {
					cond = ifn.Cond
				}
			}
		})
	}
	if cond == nil {
		t.Fatal("no rewritten If found")
	}
	ty, has := ti.Get(cond.ID())
	if !has {
		t.Fatalf("reused arg node %d lost its CoreTypeInfo entry", cond.ID())
	}
	if !isTConNamed(ty, "bool") {
		t.Fatalf("reused arg node %d: type %v, want bool (it must NOT be re-registered as string)", cond.ID(), ty)
	}
}

// TestShowNormalize_NestedShow: a user's explicit `show` inside a hole gives
// show(show(x)); with x:string both are the identity and both must go.
func TestShowNormalize_NestedShow(t *testing.T) {
	prog, _ := compileForNormalize(t, "nested", `module nested
export func f(s: string) -> string ! {} { "${show(s)}" }
`)
	if got := countShowCalls(prog); got != 0 {
		t.Fatalf("nested show: %d show calls survived, want 0", got)
	}
}

// TestShowNormalize_MissingTypeInfoIsHardError: a MISSING CoreTypeInfo entry is a
// broken compiler invariant (ValidateCoreTypeInfo calls such a miss a compiler
// bug), not a reason to degrade to "leave it alone". Degrading would surface as
// an ordinary encodability skip and mask the real fault — a silent fallback,
// which CLAUDE.md Principle 2 forbids.
func TestShowNormalize_MissingTypeInfoIsHardError(t *testing.T) {
	ti := types.NewCoreTypeInfo()
	arg := &core.Var{CoreNode: core.CoreNode{NodeID: 10}, Name: "x"}
	// deliberately NO ti.Set for node 10
	showFn := &core.VarGlobal{
		CoreNode: core.CoreNode{NodeID: 11},
		Ref:      core.GlobalRef{Module: "$builtin", Name: "show"},
	}
	app := &core.App{CoreNode: core.CoreNode{NodeID: 12}, Func: showFn, Args: []core.CoreExpr{arg}}
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				CoreNode: core.CoreNode{NodeID: 13},
				Bindings: []core.RecBinding{{Name: "victim", Value: app}},
				Body:     &core.Lit{CoreNode: core.CoreNode{NodeID: 14}, Kind: core.UnitLit},
			},
		},
		Meta: map[string]*core.DeclMeta{},
	}

	_, err := NewShowNormalizer(&ti).Normalize(prog)
	if err == nil {
		t.Fatal("missing CoreTypeInfo entry was absorbed silently; want a hard error")
	}
	for _, want := range []string{"victim", "10", "show"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q (it must name the function, the node and the pass)", err.Error(), want)
		}
	}
}

// isTConNamed reports whether ty is a type constructor with the given name,
// tolerating the "string"/"String" casing that circulates in the type system
// (see internal/types/typechecker_operators.go:528).
func isTConNamed(ty types.Type, name string) bool {
	con, ok := ty.(*types.TCon)
	if !ok {
		return false
	}
	return strings.EqualFold(con.Name, name)
}

// TestShowNormalize_ValueEquivalence is the safety net for the pass's
// load-bearing premise: `show(s: string) ≡ s`, exactly, for every string —
// including ones that would be quoted, escaped or truncated if `show` treated
// them as a nested value. `showValue` returns bare strings only at the top level
// (internal/builtins/show.go:118-120) and applies `truncateIfNeeded` (maxWidth
// 80) to composites, so a 100-char string is a real test of the claim, not a
// decorative one.
//
// The expectations are literal, not derived, so an elision that quietly changed
// the bytes fails here rather than corpus-wide.
func TestShowNormalize_ValueEquivalence(t *testing.T) {
	hundred := strings.Repeat("0123456789", 10)

	cases := []struct {
		name string
		expr string
		want string
	}{
		{"plain", `"${s}"`, "hi"},
		{"empty", `"${e}"`, ""},
		{"embedded_quote", `"${q}"`, `he said "hi"`},
		{"embedded_newline", `"${n}"`, "a\nb"},
		{"long_100_over_maxwidth", `"${long}"`, hundred},
		{"explicit_show_on_string", `"${show(s)}"`, "hi"},
		{"adjacent_holes", `"${s}${e}${s}"`, "hihi"},
		{"bool_true", `"${bt}"`, "true"},
		{"bool_false", `"${bf}"`, "false"},
		{"bool_in_context", `"v=${bt}!"`, "v=true!"},
		{"int_positive", `"${ip}"`, "42"},
		{"int_negative", `"${ineg}"`, "-5"},
		{"int_zero", `"${iz}"`, "0"},
		{"float_untouched", `"${fl}"`, "3.5"},
		{"mixed", `"${s}|${ip}|${bt}|${fl}"`, "hi|42|true|3.5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A BARE EXPRESSION (no module header) routes through runSingle and is
			// actually evaluated, so this also exercises the pipeline_single.go
			// wiring — the module path is covered by the Core-shape tests above.
			src := `let s = "hi" in
let e = "" in
let q = "he said \"hi\"" in
let n = "a\nb" in
let long = "` + hundred + `" in
let bt = true in
let bf = false in
let ip = 42 in
let ineg = -5 in
let iz = 0 in
let fl = 3.5 in
` + tc.expr + `
`
			dir := t.TempDir()
			p := filepath.Join(dir, "valeq.ail")
			if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			// The bare-expression path needs a global resolver for $builtin
			// references, exactly as the CLI wires one (main_run_exec.go:197).
			// Without it a surviving `show` — i.e. the float case — cannot be
			// evaluated, which would make this test silently stop covering the
			// residue rows.
			evaluator := eval.NewCoreEvaluator()
			res, err := Run(Config{
				RelaxModules:   true,
				NoCache:        true,
				Mode:           ModeEval,
				GlobalResolver: runtime.NewBuiltinOnlyResolver(runtime.NewBuiltinRegistry(evaluator)),
			}, Source{Code: src, Filename: p})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if len(res.Errors) > 0 {
				t.Fatalf("errors: %v", res.Errors)
			}
			sv, ok := res.Value.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected a string result, got %T (%v)", res.Value, res.Value)
			}
			got := sv.Value
			if got != tc.want {
				t.Errorf("interpolation output changed\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

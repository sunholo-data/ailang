package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// M-ELABORATOR-LEXICAL-SCOPE (#1467): local binders shadow imports, builtins,
// constructors and module aliases in expression position. Each case counts how
// often the GLOBAL resolution of a name survives in the elaborated core.

func elaborateWithGlobals(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src, "test.ail"))
	prog := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	elab := NewElaborator()
	elab.SetGlobalEnv(map[string]core.GlobalRef{
		"tick":  {Module: "pkg/a", Name: "tick"},
		"L.map": {Module: "std/list", Name: "map"},
	})
	elab.AddBuiltinsToGlobalEnv()
	elab.RegisterConstructor("Option", "None", 0, true, 1)
	elab.RegisterConstructor("Option", "Some", 1, true, 1)
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		t.Fatalf("elaborate error: %v", err)
	}
	if len(elab.scope) != 0 {
		t.Errorf("scope stack leaked %d frame(s)", len(elab.scope))
	}
	var b strings.Builder
	for _, d := range coreProg.Decls {
		b.WriteString(d.String())
		b.WriteString("\n")
	}
	return b.String()
}

func TestLexicalScope_BinderShadowsGlobals(t *testing.T) {
	tests := []struct {
		name, src, global string
		want              int // occurrences of the global resolution
	}{
		{"lambda_param", `\tick. tick + 1`, "pkg/a.tick", 0},
		{"lambda_param_record", `\tick. {tick: tick}`, "pkg/a.tick", 0},
		{"let_body", `let tick = 5 in tick`, "pkg/a.tick", 0},
		{"let_value_outside_scope", `let tick = tick(1) in tick`, "pkg/a.tick", 1},
		{"letrec_value_and_body", `letrec tick = \n. tick(n) in tick(1)`, "pkg/a.tick", 0},
		{"match_binder", `match 1 { tick => tick }`, "pkg/a.tick", 0},
		{"match_guard", `match 1 { tick if tick > 0 => tick, _ => 0 }`, "pkg/a.tick", 0},
		{"tuple_binder", `match (1, 2) { (tick, b) => tick + b }`, "pkg/a.tick", 0},
		{"list_tail_binder", `match [1] { [h, ...tick] => tick }`, "pkg/a.tick", 0},
		{"record_binder", `match {tick: 1} { {tick} => tick }`, "pkg/a.tick", 0},
		{"ctor_arg_binder", `match Some(1) { Some(tick) => tick, None => 0 }`, "pkg/a.tick", 0},
		{"block_statement_let", `{ let tick = 2; tick }`, "pkg/a.tick", 0},
		{"block_use_before_let", `{ let a = tick(1); let tick = 2; a + tick }`, "pkg/a.tick", 1},
		{"block_statement_letrec", `{ letrec tick = \n. tick(n); tick(1) }`, "pkg/a.tick", 0},
		{"no_binder_keeps_import", `\x. tick(x)`, "pkg/a.tick", 1},
		{"scope_closes_after_lambda", `(\tick. tick)(tick(1))`, "pkg/a.tick", 1},
		{"scope_closes_after_arm", `match 1 { tick => tick } + tick(1)`, "pkg/a.tick", 1},
		{"builtin_param", `\show. show`, "$builtin.show", 0},
		{"builtin_kept", `\x. show(x)`, "$builtin.show", 1},
		{"nullary_ctor_param", `\None. None`, "$adt.make_Option_None", 0},
		{"ctor_call_param", `\Some. Some(1)`, "$adt.make_Option_Some", 0},
		{"ctor_call_kept", `\x. Some(x)`, "$adt.make_Option_Some", 1},
		{"module_alias_local", `\L. L.map`, "std/list.map", 0},
		{"module_alias_kept", `\x. L.map`, "std/list.map", 1},
		{"func_literal_param", `let f = func(tick: int) -> int { tick + 1 } in f(1)`, "pkg/a.tick", 0},
		{"forall_var", `forall tick: 0..3 => tick > 0`, "pkg/a.tick", 0},
		{"pattern_ctor_unchanged", `\None. match Some(1) { None => 0, Some(v) => v }`, "None([])", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := elaborateWithGlobals(t, tt.src)
			if got := strings.Count(out, tt.global); got != tt.want {
				t.Errorf("%s: %q appears %d time(s), want %d\ncore: %s", tt.src, tt.global, got, tt.want, out)
			}
		})
	}
}

// A function declaration's parameters shadow imports in its body.
func TestLexicalScope_FuncDeclParams(t *testing.T) {
	src := "module m\nexport pure func f(tick: int) -> int = tick + 1\nexport pure func g(x: int) -> int = tick(x)\n"
	out := elaborateWithGlobals(t, src)
	if got := strings.Count(out, "pkg/a.tick"); got != 1 {
		t.Errorf("want exactly one import reference (from g), got %d\ncore: %s", got, out)
	}
}

// The module pipeline elaborates functions through ElaborateFile (SCC
// emitter + funcToLambda + contract elaboration): parameters shadow imports in
// the body and in requires/ensures contracts.
func TestLexicalScope_ElaborateFileParamsAndContracts(t *testing.T) {
	src := `module m
export func f(tick: int) -> int ! {}
requires { tick >= 0 }
ensures { result >= tick } {
  tick + 1
}
export pure func g(x: int) -> int = tick(x)
`
	p := parser.New(lexer.New(src, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	elab := NewElaboratorWithPath("m")
	elab.SetGlobalEnv(map[string]core.GlobalRef{"tick": {Module: "pkg/a", Name: "tick"}})
	coreProg, err := elab.ElaborateFile(file)
	if err != nil {
		t.Fatalf("elaborate error: %v", err)
	}
	if len(elab.scope) != 0 {
		t.Errorf("scope stack leaked %d frame(s)", len(elab.scope))
	}
	var b strings.Builder
	for _, d := range coreProg.Decls {
		b.WriteString(d.String() + "\n")
	}
	if got := strings.Count(b.String(), "pkg/a.tick"); got != 1 {
		t.Errorf("want exactly one import reference (from g), got %d\ncore: %s", got, b.String())
	}
	meta := coreProg.Meta["f"]
	if meta == nil || len(meta.Contracts) != 2 {
		t.Fatalf("want 2 contracts on f, got %#v", meta)
	}
	for _, c := range meta.Contracts {
		if s := c.Expr.String(); strings.Contains(s, "pkg/a.tick") {
			t.Errorf("contract %q resolved the parameter to the import: %s", c.Message, s)
		}
	}
}

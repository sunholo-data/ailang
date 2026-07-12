package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// elaborateForTest parses and elaborates a source string with std/string.split
// resolved to its VarGlobal (mirroring what the module pipeline does via
// SetGlobalEnv). It returns the elaborated Core program (pre-lowering).
func elaborateForTest(t *testing.T, src string) *core.Program {
	t.Helper()
	l := lexer.New(src, "test.ail")
	p := parser.New(l)
	f := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	el := elaborate.NewElaborator()
	el.AddBuiltinsToGlobalEnv()
	// Simulate `import std/string (split)`.
	el.MergeGlobalEnv(map[string]core.GlobalRef{
		"split": {Module: "std/string", Name: "split"},
	})
	prog, err := el.ElaborateFile(f)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}
	return prog
}

// wrapMain wraps a split expression body inside a minimal module so it parses.
func wrapMain(body string) string {
	return "module test/m\n" +
		"import std/string (split)\n" +
		"export func main() -> [string] {\n" +
		body + "\n}\n"
}

func TestDetectArgOrderWarnings_Split(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantWarn bool
	}{
		// Triggers: short literal delimiter as arg0, non-literal as arg1.
		{"comma_var", `let name = "a,b,c"; split(",", name)`, true},
		{"slash_var", `let path = "a/b/c"; split("/", path)`, true},
		{"newline_var", `let text = "a\nb"; split("\n", text)`, true},
		{"double_colon_var", `let q = "a::b"; split("::", q)`, true},

		// No false positives.
		{"correct_order", `let name = "a,b,c"; split(name, ",")`, false},
		{"long_first_arg", `split("hello world", " ")`, false},
		{"both_literals", `split(",", ",")`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := elaborateForTest(t, wrapMain(tc.body))
			warns := DetectArgOrderWarnings(prog)
			got := len(warns) > 0
			if got != tc.wantWarn {
				t.Fatalf("body %q: wantWarn=%v got=%v (warns=%d)", tc.body, tc.wantWarn, got, len(warns))
			}
			if tc.wantWarn {
				aw, ok := warns[0].(*ArgOrderWarning)
				if !ok {
					t.Fatalf("expected *ArgOrderWarning, got %T", warns[0])
				}
				// Warning satisfies elaborate.Warning and renders hint + note.
				var _ elaborate.Warning = aw
				s := aw.String()
				if !strings.Contains(s, "hint:") || !strings.Contains(s, "note:") {
					t.Errorf("warning missing hint/note: %q", s)
				}
				if !strings.Contains(s, "split") {
					t.Errorf("warning missing func name: %q", s)
				}
			}
		})
	}
}

// TestDetectArgOrderWarnings_BothVarsNoWarn covers split(a, b) where both are
// variables — the pass cannot tell, so it must not warn. Kept separate because
// it needs two bound vars.
func TestDetectArgOrderWarnings_BothVarsNoWarn(t *testing.T) {
	body := `let a = ","; let b = "x,y"; split(a, b)`
	prog := elaborateForTest(t, wrapMain(body))
	if warns := DetectArgOrderWarnings(prog); len(warns) != 0 {
		t.Fatalf("split(a, b) should not warn, got %d: %v", len(warns), warns)
	}
}

// TestDetectArgOrderWarnings_ModuleGuard proves non-vacuity: a user-defined
// local `split(x, y)` (NOT imported from std/string) must NOT trigger, because
// it elaborates to a plain *core.Var, not the std/string.split VarGlobal.
func TestDetectArgOrderWarnings_ModuleGuard(t *testing.T) {
	src := "module test/userdef\n" +
		"export func split(x: string, y: string) -> [string] { [x, y] }\n" +
		"export func main() -> [string] {\n" +
		"  let name = \"a/b/c\";\n" +
		"  split(\"/\", name)\n" +
		"}\n"

	// Elaborate WITHOUT the std/string import mapping: `split` is a local func.
	l := lexer.New(src, "test.ail")
	p := parser.New(l)
	f := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	el := elaborate.NewElaborator()
	el.AddBuiltinsToGlobalEnv()
	prog, err := el.ElaborateFile(f)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	// Sanity: assert the call really is a plain Var, not a std/string VarGlobal.
	if hasStdStringSplitVarGlobal(prog) {
		t.Fatal("test setup invalid: user-defined split resolved to std/string VarGlobal")
	}
	if warns := DetectArgOrderWarnings(prog); len(warns) != 0 {
		t.Fatalf("user-defined split must not warn, got %d: %v", len(warns), warns)
	}
}

// TestSplitArgWarning_Integration exercises the full module pipeline end-to-end:
// the reversed-split warning must be surfaced in result.Warnings AND must NOT
// block compilation (no error). A correct-order call must not warn.
func TestSplitArgWarning_Integration(t *testing.T) {
	dir := t.TempDir()

	writeMod := func(name, body string) string {
		p := filepath.Join(dir, name)
		src := "module " + strings.TrimSuffix(name, ".ail") + "\n" +
			"import std/string (split)\n" +
			"import std/list (length)\n" +
			"export func main() -> int {\n" +
			body + "\n}\n"
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		return p
	}

	countWarns := func(warns []elaborate.Warning) int {
		n := 0
		for _, w := range warns {
			if _, ok := w.(*ArgOrderWarning); ok {
				n++
			}
		}
		return n
	}

	t.Run("reversed_warns_non_blocking", func(t *testing.T) {
		p := writeMod("rev.ail", `  let name = "a/b/c";
  length(split("/", name))`)
		b, _ := os.ReadFile(p)
		cfg := Config{RelaxModules: true, NoCache: true}
		res, err := Run(cfg, Source{Code: string(b), Filename: p})
		if err != nil {
			t.Fatalf("pipeline error (warning must not block): %v", err)
		}
		if len(res.Errors) != 0 {
			t.Fatalf("unexpected errors: %v", res.Errors)
		}
		if got := countWarns(res.Warnings); got != 1 {
			t.Fatalf("expected 1 ArgOrderWarning, got %d (all=%d)", got, len(res.Warnings))
		}
	})

	t.Run("correct_order_no_warn", func(t *testing.T) {
		p := writeMod("ok.ail", `  let name = "a/b/c";
  length(split(name, "/"))`)
		b, _ := os.ReadFile(p)
		cfg := Config{RelaxModules: true, NoCache: true}
		res, err := Run(cfg, Source{Code: string(b), Filename: p})
		if err != nil {
			t.Fatalf("pipeline error: %v", err)
		}
		if got := countWarns(res.Warnings); got != 0 {
			t.Fatalf("expected 0 ArgOrderWarning, got %d", got)
		}
	})
}

// hasStdStringSplitVarGlobal reports whether any App in prog calls the
// std/string.split VarGlobal (used to validate module-guard test setup).
func hasStdStringSplitVarGlobal(prog *core.Program) bool {
	found := false
	var walk func(e core.CoreExpr)
	walk = func(e core.CoreExpr) {
		if e == nil || found {
			return
		}
		switch n := e.(type) {
		case *core.App:
			if vg, ok := n.Func.(*core.VarGlobal); ok {
				if vg.Ref.Module == "std/string" && vg.Ref.Name == "split" {
					found = true
					return
				}
			}
			walk(n.Func)
			for _, a := range n.Args {
				walk(a)
			}
		case *core.Lambda:
			walk(n.Body)
		case *core.Let:
			walk(n.Value)
			walk(n.Body)
		case *core.LetRec:
			for _, b := range n.Bindings {
				walk(b.Value)
			}
			walk(n.Body)
		case *core.If:
			walk(n.Cond)
			walk(n.Then)
			walk(n.Else)
		}
	}
	for _, d := range prog.Decls {
		walk(d)
	}
	return found
}

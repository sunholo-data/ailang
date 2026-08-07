package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// mustParse parses source into a *ast.Program, failing the test on any parse
// error. It is the shared front door for formatter tests.
func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(src, "test://fmt"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parse errors for %q:\n%v", src, errs)
	}
	if prog == nil || prog.File == nil {
		t.Fatalf("nil program/file for %q", src)
	}
	return prog
}

// formatSrc parses and formats source, failing on either error.
func formatSrc(t *testing.T, src string) string {
	t.Helper()
	prog := mustParse(t, src)
	out, err := Source(prog, Options{})
	if err != nil {
		t.Fatalf("Source(%q) error: %v", src, err)
	}
	return string(out)
}

// TestCanonicalLayout locks the canonical whitespace, sequence, and final-newline
// rules with exact golden output.
func TestCanonicalLayout(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "hello_world",
			src:  "module examples/hello\nimport std/io (println)\nexport func main() -> () ! {IO} = println(\"Hello, World!\")",
			// Header block is TIGHT: `module` and its imports are one contiguous
			// group, with the blank line falling before the first declaration.
			// That is what the teaching prompt writes (module→import contiguous in
			// 21 examples vs 2; import→import in 55 vs 0), and fmt disagreeing with
			// it fired the motoko fmt extension on nearly every file a model wrote.
			want: "module examples/hello\nimport std/io (println)\n\nexport func main() -> () ! {IO} = println(\"Hello, World!\")\n",
		},
		{
			name: "braced_newline_block",
			src:  "module m\nfunc f() { let a = 1; let b = 2; a + b }",
			want: "module m\n\nfunc f() {\n  let a = 1\n  let b = 2\n  a + b\n}\n",
		},
		{
			name: "single_expr_equation_body",
			src:  "module m\nfunc f() = x * 2",
			want: "module m\n\nfunc f() = x * 2\n",
		},
		{
			name: "explicit_let_in_stays_explicit",
			src:  "module m\nfunc f() = let a = 1 in let b = 2 in a + b",
			want: "module m\n\nfunc f() = let a = 1 in let b = 2 in a + b\n",
		},
		{
			name: "adt_deriving",
			src:  "module m\ntype Color = Red | Green | Blue deriving (Eq)",
			want: "module m\n\ntype Color = Red | Green | Blue deriving (Eq)\n",
		},
		{
			name: "adt_positional_fields",
			src:  "module m\ntype Shape = Circle(int) | Rectangle(int, int)",
			want: "module m\n\ntype Shape = Circle(int) | Rectangle(int, int)\n",
		},
		{
			name: "top_level_let",
			src:  "module m\nlet x: bool = let a = 1 in a == 1",
			want: "module m\n\nlet x: bool = let a = 1 in a == 1\n",
		},
		{
			name: "two_top_level_decls_blank_line",
			src:  "module m\nfunc a() = 1\nfunc b() = 2",
			want: "module m\n\nfunc a() = 1\n\nfunc b() = 2\n",
		},
		{
			name: "match_arms",
			src:  "module m\nfunc h(xs) = match xs { [] => 0, [x, ...rest] => x }",
			want: "module m\n\nfunc h(xs) = match xs {\n  [] => 0,\n  [x, ...rest] => x\n}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSrc(t, tc.src)
			if got != tc.want {
				t.Errorf("canonical layout mismatch\n--- got ---\n%q\n--- want ---\n%q", got, tc.want)
			}
		})
	}
}

// TestExactlyOneFinalNewline asserts the output always ends in exactly one LF.
func TestExactlyOneFinalNewline(t *testing.T) {
	got := formatSrc(t, "module m\nfunc f() = 1")
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("output must end with a newline, got %q", got)
	}
	if strings.HasSuffix(got, "\n\n") {
		t.Fatalf("output must end with exactly one newline, got %q", got)
	}
}

// TestNoSequenceSemicolonsInCommonCase confirms the canonical block form uses no
// semicolons when every statement begins with a statement-starter token.
func TestNoSequenceSemicolonsInCommonCase(t *testing.T) {
	got := formatSrc(t, "module m\nfunc f() { let a = 1; let b = 2; a + b }")
	if strings.Contains(got, ";") {
		t.Errorf("canonical block should contain no semicolons, got:\n%s", got)
	}
}

// TestSemicolonInsertedForUnsafeSeparator verifies a `;` is emitted before a
// statement that starts with a non-starter token, so the block re-parses
// correctly rather than gluing the two statements together. The AST is built
// directly because the parser would otherwise glue a source `f(x)\n()` at parse
// time (exactly the merge the `;` prevents on re-parse).
func TestSemicolonInsertedForUnsafeSeparator(t *testing.T) {
	call := &ast.FuncCall{
		Func: &ast.Identifier{Name: "println"},
		Args: []ast.Expr{&ast.Literal{Kind: ast.StringLit, Value: "hi"}},
	}
	unit := &ast.Literal{Kind: ast.UnitLit}
	d := &ast.FuncDecl{
		Name: "main",
		Body: &ast.Block{Exprs: []ast.Expr{call, unit}},
	}
	prog := &ast.Program{File: &ast.File{Module: &ast.ModuleDecl{Path: "m"}, Decls: []ast.Node{d}}}
	out, err := Source(prog, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, `println("hi");`) {
		t.Errorf("expected `;` before the bare-unit statement, got:\n%s", got)
	}
	// Re-parse must round-trip to a 2-statement block, not a glued call.
	reprog := mustParse(t, got)
	blk, ok := reprog.File.Funcs[0].Body.(*ast.Block)
	if !ok || len(blk.Exprs) != 2 {
		t.Errorf("expected 2-statement block after re-parse, got %T", reprog.File.Funcs[0].Body)
	}
}

// TestPrecedenceParens covers precedence-driven parenthesization: redundant
// source parens are dropped and required parens are reconstructed.
func TestPrecedenceParens(t *testing.T) {
	cases := []struct{ src, wantExpr string }{
		{"module m\nfunc f(x) = 1 + 2 * 3 - x", "1 + 2 * 3 - x"},
		{"module m\nfunc f() = (1 + 2) * 3", "(1 + 2) * 3"},
		{"module m\nfunc f(x) = ((x + 1) * 2) - 3", "(x + 1) * 2 - 3"},
		{"module m\nfunc f(a, b, c) = a && b || c", "a && b || c"},
		{"module m\nfunc f(a, b, c) = a && (b || c)", "a && (b || c)"},
		{"module m\nfunc f(a, b) = a :: b :: []", "a :: b :: []"},
	}
	for _, tc := range cases {
		t.Run(tc.wantExpr, func(t *testing.T) {
			got := formatSrc(t, tc.src)
			if !strings.Contains(got, tc.wantExpr) {
				t.Errorf("expected %q in output, got:\n%s", tc.wantExpr, got)
			}
		})
	}
}

// TestUnsupportedNodesError proves the printer fails loudly rather than falling
// back to a debug String() rendering. Each case constructs an AST directly.
func TestUnsupportedNodesError(t *testing.T) {
	cases := []struct {
		name    string
		program *ast.Program
	}{
		{
			name: "error_node",
			program: &ast.Program{File: &ast.File{
				Module: &ast.ModuleDecl{Path: "m"},
				Decls:  []ast.Node{&ast.Error{Msg: "boom"}},
			}},
		},
		{
			name:    "nil_program",
			program: nil,
		},
		{
			name:    "nil_file",
			program: &ast.Program{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Source(tc.program, Options{})
			if err == nil {
				t.Fatalf("expected an error for %s, got nil", tc.name)
			}
		})
	}
}

// TestErrorExprInsideDecl ensures an ast.Error nested in an expression position
// also errors rather than emitting `<error>`.
func TestErrorExprInsideDecl(t *testing.T) {
	prog := &ast.Program{File: &ast.File{
		Module: &ast.ModuleDecl{Path: "m"},
		Decls: []ast.Node{
			&ast.Let{Name: "x", Value: &ast.Error{Msg: "bad"}},
		},
	}}
	if _, err := Source(prog, Options{}); err == nil {
		t.Fatal("expected error for ast.Error in value position, got nil")
	}
}

// TestCustomIndent verifies the Options.Indent unit is honored.
func TestCustomIndent(t *testing.T) {
	prog := mustParse(t, "module m\nfunc f() { let a = 1; a }")
	out, err := Source(prog, Options{Indent: "\t"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\n\tlet a = 1\n") {
		t.Errorf("expected tab indentation, got:\n%q", out)
	}
}

// TestNoStringFallbackForCons is a regression guard: the cons operator desugars
// to FuncCall{`::`} which has NO prefix/call re-parse; it must be re-emitted
// infix, never as `::(a, b)`.
func TestConsRendersInfix(t *testing.T) {
	got := formatSrc(t, "module m\nfunc c(a, b) = a :: b")
	if strings.Contains(got, "::(") {
		t.Errorf("cons must render infix, not as a call; got:\n%s", got)
	}
	if !strings.Contains(got, "a :: b") {
		t.Errorf("expected `a :: b`, got:\n%s", got)
	}
}

// TestExamplesGoldens formats a small set of real example files against goldens
// captured under testdata/. Run `UPDATE_GOLDEN=1 go test` to regenerate.
func TestExamplesGoldens(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.golden")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Skip("no goldens present")
	}
	for _, golden := range matches {
		golden := golden
		name := strings.TrimSuffix(filepath.Base(golden), ".golden")
		t.Run(name, func(t *testing.T) {
			srcPath := strings.TrimSuffix(golden, ".golden") + ".ail"
			src, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read source %s: %v", srcPath, err)
			}
			got := formatSrc(t, string(src))
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden %s: %v", golden, err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

// TestConsPatternRendersInfix is the PATTERN half of TestConsRendersInfix.
//
// A cons pattern parses to ConstructorPattern{Name:"::"} just as a cons
// expression parses to FuncCall{"::"}. The expression half was fixed and
// guarded; the pattern half was not, so `h :: t` was re-emitted as `::(h, t)`.
//
// That is not cosmetic. `ailang prompt` — the teaching text every eval model
// receives — uses `h :: t` and never the prefix form, so fmt was telling models
// their correct pattern was non-canonical and handing them an untaught dialect.
func TestConsPatternRendersInfix(t *testing.T) {
	got := formatSrc(t, "module m\npure func f(xs: [int]) -> int =\n  match xs {\n    [] => 0,\n    h :: t => h + f(t)\n  }")
	if strings.Contains(got, "::(") {
		t.Errorf("cons PATTERN must render infix, not as a call; got:\n%s", got)
	}
	if !strings.Contains(got, "h :: t") {
		t.Errorf("expected `h :: t`, got:\n%s", got)
	}
}

// TestListTypeRendersSugar: `[T]` is normalized to TypeApp{Constructor:"list"}
// at parse time, which made fmt print `list[int]` for every `[int]` written.
// The teaching prompt uses `[int]` 64 times and `list[...]` zero times.
func TestListTypeRendersSugar(t *testing.T) {
	got := formatSrc(t, "module m\npure func f(xs: [int]) -> [string] = []")
	if strings.Contains(got, "list[") {
		t.Errorf("list type must render as [T] sugar, not list[T]; got:\n%s", got)
	}
	if !strings.Contains(got, "xs: [int]") || !strings.Contains(got, "-> [string]") {
		t.Errorf("expected `[int]` and `[string]`, got:\n%s", got)
	}
}

// TestFmtOutputMatchesTaughtDialect guards the invariant that actually matters:
// whatever fmt emits, a model was taught to write. Both spellings parse, so a
// type-check cannot catch this — only a dialect assertion can.
func TestFmtOutputMatchesTaughtDialect(t *testing.T) {
	src := "module m\npure func f(xs: [int]) -> int =\n  match xs {\n    [] => 0,\n    h :: t => h + f(t)\n  }"
	got := formatSrc(t, src)
	for _, untaught := range []string{"list[", "::("} {
		if strings.Contains(got, untaught) {
			t.Errorf("fmt emitted %q, which `ailang prompt` never teaches:\n%s", untaught, got)
		}
	}
	// Idempotence: formatting the output again must not reintroduce drift.
	again := formatSrc(t, got)
	if again != got {
		t.Errorf("fmt not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

package format

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// fmtExpr formats src (a whole module) and returns the output, failing on any
// format or re-parse error.
func fmtExpr(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src, "interp_test"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("input does not parse: %v", errs[0])
	}
	out, err := Source(prog, Options{Indent: "  "})
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	// Every formatter output must re-parse to the same tree.
	rp := parser.New(lexer.New(string(out), "interp_test"))
	reprog := rp.Parse()
	if errs := rp.Errors(); len(errs) > 0 {
		t.Fatalf("output does not re-parse: %v\n--- output ---\n%s", errs[0], out)
	}
	if d := cmp.Diff(prog.File, reprog.File, ignorePosSpan); d != "" {
		t.Fatalf("round-trip changed the AST:\n%s\n--- output ---\n%s", d, out)
	}
	return string(out)
}

// TestInterpolationRoundTripsAsWritten covers the shapes the teaching prompt uses.
// Each must come back out as an interpolation, not as a concat_String chain —
// the divergence that cost +62% output tokens (see interp.go).
func TestInterpolationRoundTripsAsWritten(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"single hole with text", `println("Ok(${intToStr(n)})")`, `"Ok(${intToStr(n)})"`},
		{"hole only, with prefix", `println("Err: ${e}")`, `"Err: ${e}"`},
		{"quoted text around hole", `println("Parse '42': ${v}")`, `"Parse '42': ${v}"`},
		{"adjacent holes", `println("${a}${b}")`, `"${a}${b}"`},
		{"escaped newlines between holes", `println("${a}\n${b}\n${c}")`, `"${a}\n${b}\n${c}"`},
		{"nested show", `println("x=${show(a)}, y=${show(b)}")`, `"x=${show(a)}, y=${show(b)}"`},
		{"arithmetic in hole", `println("next: ${a + 1}")`, `"next: ${a + 1}"`},
		{"multi-arg call in hole", `println("first: ${substring(s, 0, 1)}")`, `"first: ${substring(s, 0, 1)}"`},
		// The prompt explicitly teaches that nested braces work inside a hole.
		{"record literal in hole", `println("Point: ${show({x: 1, y: 2}.x)}")`, `"Point: ${show({ x: 1, y: 2 }.x)}"`},
		{"let-block in hole", `println("Squared: ${let sq = a * a in sq}")`, `"Squared: ${let sq = a * a in sq}"`},
		// A literal `${` is escaped by the lexer and must stay escaped.
		{"escaped interpolation marker", `println("Use \${var} here")`, `"Use \${var} here"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "module m\nexport func main() -> () ! {IO} {\n  let a = 1\n  let b = 2\n  let c = 3\n  let e = \"e\"\n  let n = 1\n  let s = \"s\"\n  let v = \"v\"\n  " + tc.body + "\n}\n"
			out := fmtExpr(t, src)
			if !strings.Contains(out, tc.want) {
				t.Errorf("want output to contain %s\ngot:\n%s", tc.want, out)
			}
			if strings.Contains(out, "concat_String") {
				t.Errorf("output still contains a concat_String chain — the interpolation was not re-sugared:\n%s", out)
			}
		})
	}
}

// TestHandWrittenConcatIsLeftAlone locks the guards in interp.go. Each of these
// is a concat_String the user wrote themselves; re-sugaring any of them would
// change what the source re-parses to, which is strictly worse than printing a
// chain.
func TestHandWrittenConcatIsLeftAlone(t *testing.T) {
	cases := []struct {
		name, body, reason string
	}{
		{"no show wrapper", `concat_String("a", s)`,
			"a bare operand is not a hole: `${s}` would add a show()"},
		{"all literals", `concat_String("a", "b")`,
			"no hole at all; `\"ab\"` would fuse two parts into one"},
		{"adjacent literals", `concat_String(concat_String("a", "b"), show(n))`,
			"`\"ab${n}\"` re-parses with ONE text part, not two"},
		{"right-nested chain", `concat_String(show(n), concat_String("a", show(n)))`,
			"flattening would re-associate the tree"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "module m\nexport func main() -> () ! {IO} {\n  let n = 1\n  let s = \"s\"\n  println(" + tc.body + ")\n}\n"
			// fmtExpr asserts the round-trip, which is the property that matters.
			out := fmtExpr(t, src)
			if !strings.Contains(out, "concat_String") {
				t.Errorf("re-sugared a hand-written chain (%s):\n%s", tc.reason, out)
			}
		})
	}
}

// TestBareShowIsNotAnInterpolation guards the degenerate desugaring: `"${x}"`
// with no surrounding text produces a BARE show(x) and no concat at all
// (parser_literals.go elides empty parts). Re-sugaring that would rewrite every
// hand-written show() call in the language into a string.
func TestBareShowIsNotAnInterpolation(t *testing.T) {
	out := fmtExpr(t, "module m\nexport func main() -> () ! {IO} {\n  let n = 1\n  println(show(n))\n}\n")
	if strings.Contains(out, `"${`) {
		t.Errorf("a bare show() was rewritten as an interpolation:\n%s", out)
	}
}

// TestTaughtSpellingsSurviveFormatting covers the other four dialect fixes that
// landed with M-FMT-DIALECT-ALIGNMENT. Each pair parses to an identical AST, so
// the formatter is free to pick — and must pick what `ailang prompt` teaches.
func TestTaughtSpellingsSurviveFormatting(t *testing.T) {
	cases := []struct {
		name, src, want, notWant string
	}{
		{
			"single-param function type keeps the bare arrow",
			"module m\nfunc twice(f: int -> int, x: int) -> int = f(f(x))\n",
			"f: int -> int", "(int) -> int",
		},
		{
			"function-typed parameter keeps its parens",
			"module m\nfunc ap(f: (int -> int) -> int, x: int) -> int = f(\\y. y)\n",
			"(int -> int) -> int", "int -> int -> int",
		},
		{
			"zero-argument call stays empty",
			"module m\nexport func main() -> () ! {Clock} {\n  let t = now()\n  ()\n}\n",
			"now()", "now(())",
		},
		{
			"record pattern uses the punning shorthand",
			"module m\nfunc f(p: {name: string, age: int}) -> string = match p {\n  {name, age} => name\n}\n",
			"{ name, age }", "name: name",
		},
		{
			"record pattern keeps an explicit rename",
			"module m\nfunc f(p: {host: string}) -> string = match p {\n  {host: h} => h\n}\n",
			"{ host: h }", "",
		},
		{
			"open record prints the ... sugar, not _r0",
			"module m\npure func getEmail(u: {email: string, ...}) -> string = u.email\n",
			"{ email: string, ... }", "_r0",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := fmtExpr(t, tc.src)
			if !strings.Contains(out, tc.want) {
				t.Errorf("want %q in output, got:\n%s", tc.want, out)
			}
			if tc.notWant != "" && strings.Contains(out, tc.notWant) {
				t.Errorf("output still contains the untaught spelling %q:\n%s", tc.notWant, out)
			}
		})
	}
}

// TestBlockStatementMayStartWithALiteral is the parser half of the fix.
//
// fmt drops `;` separators and relies on the R2 newline soft separator
// (peekStartsNewlineBlockStatement). That set omitted every literal, so a block
// whose last statement began with one did not parse — which is exactly what fmt
// emits once interpolation is re-sugared, and exactly what a model writes when
// taught both newline separators and `"${x}"`.
func TestBlockStatementMayStartWithALiteral(t *testing.T) {
	tails := []string{`"plain"`, `"${a} y"`, `42`, `3.5`, `true`, `false`}
	for _, tail := range tails {
		t.Run(tail, func(t *testing.T) {
			src := "module m\nexport func f() -> int {\n  let a = 1\n  " + tail + "\n}\n"
			p := parser.New(lexer.New(src, "interp_test"))
			p.Parse()
			if errs := p.Errors(); len(errs) > 0 {
				t.Errorf("newline-separated block ending in %s must parse: %v", tail, errs[0])
			}
		})
	}
}

// TestNewlineDoesNotSplitAContinuedExpression is the other side of that fix: a
// multi-line expression must still be ONE expression. Operators are not
// statement-starters, so the line break is not a separator here.
func TestNewlineDoesNotSplitAContinuedExpression(t *testing.T) {
	cases := map[string]string{
		"arithmetic continuation": "module m\nexport func f() -> int {\n  let a = 1\n  a\n  + 2\n}\n",
		"append continuation":     "module m\nexport func f() -> string {\n  let a = \"x\"\n  a\n  ++ \"y\"\n}\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			p := parser.New(lexer.New(src, "interp_test"))
			prog := p.Parse()
			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("must parse: %v", errs[0])
			}
			out, err := Source(prog, Options{Indent: "  "})
			if err != nil {
				t.Fatalf("format: %v", err)
			}
			// One joined expression, so the block has a single statement and the
			// operator survives in the output.
			if !strings.Contains(string(out), "+ 2") && !strings.Contains(string(out), `++ "y"`) {
				t.Errorf("continuation was split into separate statements:\n%s", out)
			}
		})
	}
}

package format

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
)

func chainBody(t *testing.T, src string) ast.Expr {
	t.Helper()
	prog := mustParse(t, src)
	if len(prog.File.Funcs) != 1 {
		t.Fatalf("got %d functions, want 1", len(prog.File.Funcs))
	}
	body, ok := prog.File.Funcs[0].Body.(*ast.Block)
	if !ok || len(body.Exprs) != 1 {
		t.Fatalf("function body = %T, want one-expression *ast.Block", prog.File.Funcs[0].Body)
	}
	return body.Exprs[0]
}

func TestWriterTracksRuneColumn(t *testing.T) {
	w := newWriter("é ")
	w.depth = 2
	w.write("café")
	if got, want := w.col, 8; got != want {
		t.Fatalf("column after indentation and write = %d, want %d", got, want)
	}
	w.hardline()
	if w.col != 0 {
		t.Fatalf("column after hardline = %d, want 0", w.col)
	}
	w.write("x")
	if got, want := w.col, 5; got != want {
		t.Fatalf("column on next indented line = %d, want %d", got, want)
	}
}

func TestResolveMaxWidthAndEntryPoints(t *testing.T) {
	if got := resolveMaxWidth(Options{}); got != defaultMaxWidth {
		t.Fatalf("zero MaxWidth resolved to %d, want %d", got, defaultMaxWidth)
	}
	if got := resolveMaxWidth(Options{MaxWidth: 88}); got != 88 {
		t.Fatalf("explicit MaxWidth resolved to %d, want 88", got)
	}

	src := "module m\nfunc f() = let a = 1 in a"
	prog := mustParse(t, src)
	plain, err := Source(prog, Options{MaxWidth: 88})
	if err != nil {
		t.Fatal(err)
	}
	withComments, err := SourceWithComments(prog, []byte(src), Options{MaxWidth: 88})
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != string(withComments) {
		t.Fatalf("entry points differ:\nSource: %q\nSourceWithComments: %q", plain, withComments)
	}
}

func TestInlineWidthAndCurrentColumn(t *testing.T) {
	chain := chainBody(t, "module m\nexport func main() -> () ! {IO} = let a = 1 in let b = 2 in let c = 3 in println(\"done\")")
	p := &printer{w: newWriter("  "), maxWidth: 60}
	if got := p.inlineWidth(chain); got != 54 {
		t.Fatalf("inlineWidth(chain) = %d, want 54", got)
	}
	if got := 31 + prefixEquationBody + p.inlineWidth(chain); got != 88 {
		t.Fatalf("full candidate line = %d, want 88", got)
	}
	if p.exceedsWidth(chain, prefixLetIn) {
		t.Fatal("chain should fit at column 0")
	}
	p.w.write(strings.Repeat("x", 40))
	if !p.exceedsWidth(chain, prefixLetIn) {
		t.Fatal("same chain should exceed width at column 40")
	}
}

func TestInlineWidthIgnoresParentWidthAndDepth(t *testing.T) {
	chain := chainBody(t, `module m
func f() = let first = "12345678901234567890" in let second = "12345678901234567890" in let third = "12345678901234567890" in let fourth = "12345678901234567890" in first + second + third + fourth`)
	widths := make([]int, 0, 4)
	for _, maxWidth := range []int{40, 120} {
		for _, depth := range []int{0, 4} {
			p := &printer{w: newWriter("  "), maxWidth: maxWidth}
			p.w.depth = depth
			widths = append(widths, p.inlineWidth(chain))
		}
	}
	for i, got := range widths[1:] {
		if got != widths[0] {
			t.Fatalf("measurement %d = %d, want invariant width %d", i+1, got, widths[0])
		}
	}
	if widths[0] <= defaultMaxWidth {
		t.Fatalf("long-chain width = %d, want > %d", widths[0], defaultMaxWidth)
	}
}

func TestMeasurementModePreventsNestedPrinter(t *testing.T) {
	chain := chainBody(t, "module m\nfunc f() = let a = let b = let c = 1 in c in b in a")
	oldHook := measurementPrinterHook
	t.Cleanup(func() { measurementPrinterHook = oldHook })
	maxDepth := 0
	measurementPrinterHook = func(depth int) {
		if depth >= 2 {
			t.Fatalf("measurement printer reached forbidden depth %d", depth)
		}
		maxDepth = max(maxDepth, depth)
	}
	p := &printer{w: newWriter("  "), maxWidth: 1}
	measurement := p.newMeasurementPrinter()
	if measurement.exceedsWidth(chain, prefixLetIn) {
		t.Fatal("measurement mode must unconditionally disable width predicates")
	}
	if maxDepth != 1 {
		t.Fatalf("maximum measurement depth = %d, want 1", maxDepth)
	}
}

func TestWidthUsesRunesNotBytes(t *testing.T) {
	ascii := chainBody(t, "module m\nfunc f() = let x = \"cafe\" in x")
	unicode := chainBody(t, "module m\nfunc f() = let x = \"café\" in x")
	p := &printer{w: newWriter("  "), maxWidth: 80}
	asciiWidth := p.inlineWidth(ascii)
	unicodeWidth := p.inlineWidth(unicode)
	if asciiWidth != unicodeWidth {
		t.Fatalf("equal-rune expressions measured differently: ASCII=%d Unicode=%d", asciiWidth, unicodeWidth)
	}
	if utf8.RuneCountInString("café") == len("café") {
		t.Fatal("test control is invalid: Unicode fixture has equal byte and rune lengths")
	}
	p.maxWidth = asciiWidth
	if p.exceedsWidth(ascii, 0) || p.exceedsWidth(unicode, 0) {
		t.Fatal("expressions exactly at the rune width boundary must stay inline")
	}
}

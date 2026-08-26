package format

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
)

func formatWithWidth(t *testing.T, src string, maxWidth int) string {
	t.Helper()
	prog := mustParse(t, src)
	out, err := Source(prog, Options{MaxWidth: maxWidth})
	if err != nil {
		t.Fatalf("Source(MaxWidth=%d): %v", maxWidth, err)
	}
	return string(out)
}

func longestRuneLine(s string) int {
	longest := 0
	for _, line := range strings.Split(s, "\n") {
		longest = max(longest, utf8.RuneCountInString(line))
	}
	return longest
}

func TestChainWidthBoundaryAtEverySite(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		split string
	}{
		{
			name:  "equation_body",
			src:   "module m\nfunc f() = let aaaaaaaaaa = 1 in let bbbbbbbbbb = 2 in aaaaaaaaaa + bbbbbbbbbb",
			split: " =\n  let",
		},
		{
			name:  "top_level_let_value",
			src:   "module m\nlet result = let aaaaaaaaaa = 1 in let bbbbbbbbbb = 2 in aaaaaaaaaa + bbbbbbbbbb",
			split: " =\n  let",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wide := formatWithWidth(t, tc.src, 1000)
			if !strings.Contains(wide, " in let ") {
				t.Fatalf("control did not produce an inline chain:\n%s", wide)
			}
			boundary := longestRuneLine(wide)
			atBoundary := formatWithWidth(t, tc.src, boundary)
			if atBoundary != wide || strings.Contains(atBoundary, tc.split) {
				t.Fatalf("line at MaxWidth=%d wrapped:\n%s", boundary, atBoundary)
			}
			overBoundary := formatWithWidth(t, tc.src, boundary-1)
			if !strings.Contains(overBoundary, tc.split) {
				t.Fatalf("line at MaxWidth+1 stayed inline (MaxWidth=%d):\n%s", boundary-1, overBoundary)
			}
		})
	}
}

func TestLetInWidthBoundary(t *testing.T) {
	chain := chainBody(t, "module m\nfunc f() = let aaaaaaaaaa = 1 in let bbbbbbbbbb = 2 in aaaaaaaaaa + bbbbbbbbbb")
	let, ok := chain.(*ast.Let)
	if !ok {
		t.Fatalf("chain = %T, want *ast.Let", chain)
	}
	probe := &printer{w: newWriter("  "), maxWidth: 1000}
	inlineWidth := probe.inlineWidth(let)
	const currentColumn = 2
	boundary := currentColumn + inlineWidth // prefixLetIn is specified as zero at this site.

	render := func(maxWidth int) string {
		p := &printer{w: newWriter("  "), maxWidth: maxWidth}
		p.w.write(strings.Repeat(" ", currentColumn))
		if err := p.letIn(let); err != nil {
			t.Fatal(err)
		}
		return p.w.string()
	}
	if got := render(boundary); !strings.Contains(got, " in let ") {
		t.Fatalf("let-in at MaxWidth=%d wrapped:\n%s", boundary, got)
	}
	if got := render(boundary - 1); strings.Contains(got, " in let ") {
		t.Fatalf("let-in at MaxWidth+1 stayed inline:\n%s", got)
	}
}

func TestLongChainCorpusTail(t *testing.T) {
	cases := []struct {
		path         string
		maxAllowed   int
		linesOver120 int
	}{
		{path: "../../examples/runnable/set_operations.ail", maxAllowed: 120, linesOver120: 0},
		{path: "../../examples/runnable/list_extremes.ail", maxAllowed: 150, linesOver120: 1000},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			src, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			prog := mustParse(t, string(src))
			out, err := SourceWithComments(prog, src, Options{})
			if err != nil {
				t.Fatal(err)
			}
			maxWidth, over120 := 0, 0
			for _, line := range strings.Split(string(out), "\n") {
				width := utf8.RuneCountInString(line)
				maxWidth = max(maxWidth, width)
				if width > 120 {
					over120++
				}
			}
			if maxWidth > tc.maxAllowed || over120 > tc.linesOver120 {
				t.Fatalf("formatted width: max=%d (> %d), lines>120=%d (> %d)", maxWidth, tc.maxAllowed, over120, tc.linesOver120)
			}
		})
	}
}

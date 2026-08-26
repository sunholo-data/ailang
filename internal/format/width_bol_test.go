package format

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// bolFixture builds a block whose SECOND statement is a `let .. in ..` that the
// emitter reaches at beginning-of-line (blockBraced does hardline -> indented ->
// blockStatement), so the width predicate is consulted while the indentation is
// still pending. depth controls how many braced levels wrap it.
func bolFixture(depth int) string {
	stmt := "let q = " + strings.Repeat("a", 54) + " in " + strings.Repeat("c", 54)
	body := "  let z = 0 in z\n  " + stmt
	for i := 0; i < depth-1; i++ {
		body = "  let z" + string(rune('a'+i)) + " = 0 in z" + string(rune('a'+i)) +
			"\n  if 1 == 1 then {\n" + indentAll(body) + "\n  } else { 0 }"
	}
	return "module m\n\nexport func f() -> int = {\n" + body + "\n}\n"
}

func indentAll(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func formatWith(t *testing.T, src string, opts Options) string {
	t.Helper()
	prog := mustParse(t, src)
	out, err := Source(prog, opts)
	if err != nil {
		t.Fatalf("Source(%+v): %v", opts, err)
	}
	return string(out)
}

func longestLine(s string) string {
	longest := ""
	for _, line := range strings.Split(s, "\n") {
		if utf8.RuneCountInString(line) > utf8.RuneCountInString(longest) {
			longest = line
		}
	}
	return longest
}

// TestLetInAtBOLCountsPendingIndent is the M1b regression. It drives the predicate
// through a REAL hardline()->atBOL->blockStatement path (no synthetic column fake),
// and derives its boundary from the emitter's own wide rendering rather than from a
// hardcoded padding, so it stays valid if defaultMaxWidth or the indent unit change.
func TestLetInAtBOLCountsPendingIndent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		depth  int
		indent string
	}{
		{name: "depth1", depth: 1, indent: ""},
		{name: "depth2", depth: 2, indent: ""},
		{name: "depth1_tab", depth: 1, indent: "\t"},
		{name: "depth1_wide_unicode_indent", depth: 1, indent: "││││"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := bolFixture(tc.depth)

			// Control: at a generous width the chain is emitted INLINE and is the
			// longest line. If either fails the fixture stopped exercising the path.
			wide := formatWith(t, src, Options{Indent: tc.indent, MaxWidth: 4000})
			longest := longestLine(wide)
			if !strings.Contains(longest, " in ") {
				t.Fatalf("fixture's longest line is not the inline let..in chain:\n%q", longest)
			}
			n := utf8.RuneCountInString(longest)
			if strings.TrimLeft(longest, " \t│") == longest {
				t.Fatalf("fixture chain is not indented, so the BOL path is not exercised:\n%q", longest)
			}

			// Killer: one rune below that line's width, the emitter MUST break the
			// chain. At HEAD the predicate reads col==0 at BOL and keeps it inline,
			// emitting a line of n runes under a MaxWidth of n-1.
			tight := formatWith(t, src, Options{Indent: tc.indent, MaxWidth: n - 1})
			if got := utf8.RuneCountInString(longestLine(tight)); got > n-1 {
				t.Fatalf("MaxWidth=%d emitted a %d-rune line:\n%s", n-1, got, tight)
			}

			// Opposite arm (kills OVER-counting, e.g. a byte-length or constant
			// stand-in for the pending indent): at exactly that width the chain must
			// still be emitted inline, byte-identical to the wide rendering.
			if at := formatWith(t, src, Options{Indent: tc.indent, MaxWidth: n}); at != wide {
				t.Fatalf("MaxWidth=%d wrapped a line that exactly fits:\n%s", n, at)
			}
		})
	}
}

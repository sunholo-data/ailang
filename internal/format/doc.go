package format

import (
	"strings"
	"unicode/utf8"
)

// The document builder is a small, intentionally imperative pretty-printer core.
// Rather than concatenating ad-hoc strings, printers emit into a *writer that
// tracks the current indentation depth and column, and renders hard newlines
// deterministically. The emitters use hard-line layout; width-sensitive choices
// select between existing layouts rather than relying on soft-line/group
// primitives.

// writer accumulates formatted output with indentation tracking.
type writer struct {
	buf    strings.Builder
	indent string // the per-level indentation unit (e.g. "  ")
	depth  int    // current indentation depth
	atBOL  bool   // true when the cursor is at the beginning of a line
	col    int    // current column, counted in runes
}

// newWriter creates a writer using the given indentation unit.
func newWriter(indent string) *writer {
	return &writer{indent: indent, atBOL: true}
}

// write appends literal text on the current line, emitting the pending
// indentation first if the cursor is at the beginning of a line. The text must
// not contain newlines; callers use hardline for line breaks.
func (w *writer) write(s string) {
	if s == "" {
		return
	}
	if w.atBOL {
		for i := 0; i < w.depth; i++ {
			w.buf.WriteString(w.indent)
			w.col += utf8.RuneCountInString(w.indent)
		}
		w.atBOL = false
	}
	w.buf.WriteString(s)
	w.col += utf8.RuneCountInString(s)
}

// hardline emits a mandatory line break. Trailing indentation is deferred until
// the next write, so blank lines contain no trailing whitespace.
func (w *writer) hardline() {
	w.buf.WriteByte('\n')
	w.atBOL = true
	w.col = 0
}

// blankline emits exactly one empty line (two consecutive newlines). It is a
// no-op if the cursor is already at the beginning of a fresh line preceded by a
// blank line boundary; callers control blank-line placement explicitly.
func (w *writer) blankline() {
	w.hardline()
	w.hardline()
}

// indented runs fn with the indentation depth increased by one level.
func (w *writer) indented(fn func()) {
	w.depth++
	fn()
	w.depth--
}

// string returns the accumulated output.
func (w *writer) string() string {
	return w.buf.String()
}

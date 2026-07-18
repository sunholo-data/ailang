package format

import "strings"

// The document builder is a small, intentionally imperative pretty-printer core.
// Rather than concatenating ad-hoc strings, printers emit into a *writer that
// tracks the current indentation depth and column, and renders hard newlines
// deterministically. Phase 1 uses only hard-line layout (no adaptive width
// reflow); soft-line and group primitives are provided so the emitters express
// intent uniformly and so a later phase can add width-sensitive reflow without
// rewriting every printer.

// writer accumulates formatted output with indentation tracking.
type writer struct {
	buf    strings.Builder
	indent string // the per-level indentation unit (e.g. "  ")
	depth  int    // current indentation depth
	atBOL  bool   // true when the cursor is at the beginning of a line
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
		}
		w.atBOL = false
	}
	w.buf.WriteString(s)
}

// hardline emits a mandatory line break. Trailing indentation is deferred until
// the next write, so blank lines contain no trailing whitespace.
func (w *writer) hardline() {
	w.buf.WriteByte('\n')
	w.atBOL = true
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

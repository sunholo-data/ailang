package eval

import (
	"fmt"
)

// ShowBounded renders v exactly as Show(v) would, but stops WRITING once
// maxBytes have been produced and only COUNTS the remainder. The result is byte
// for byte what a caller would get from rendering the whole value and then
// truncating it — prefix plus "…(+N bytes elided)" — without the whole value
// ever existing as one string.
//
// This is the walk the trace collector sites use (M-V1-MEMORY-FOOTPRINT M1),
// via ShowTraceBounded. The old shape was `truncate(v.String(), max)`: correct
// retention, unbounded peak, because rendering a growing accumulator
// materialises O(n) per call and O(n²) over a recursion before the cap
// discards all but 1 KB of it.
//
// Since M-V1-SHOW-TOTAL M4 a trace spells values exactly as `show` does
// (nested strings quoted, floats in show's spelling): it is the same renderer
// (show.go) with a byte budget. A budget <= 0 means unbounded.
func ShowBounded(v Value, maxBytes int) string {
	return showBoundedWith(v, maxBytes, inspectShow)
}

func showBoundedWith(v Value, maxBytes int, inspect func(Value) ShowNode[Value]) string {
	if maxBytes <= 0 {
		return RenderShow(v, inspect)
	}
	w := showWriter{buf: make([]byte, 0, min(maxBytes, 256)), limit: maxBytes}
	renderShow(&w, v, inspect)
	if w.over == 0 {
		return string(w.buf)
	}
	return ShowBoundedOverflow(w.buf, w.over)
}

// ShowBoundedOverflow formats a truncated rendering: prefix plus the elided
// byte count.
func ShowBoundedOverflow(prefix []byte, overflow int) string {
	return fmt.Sprintf("%s…(+%d bytes elided)", prefix, overflow)
}

// RenderedLen reports len(Show(v)) without building the string. Used for the
// redacted-values descriptor, which records only a byte count.
func RenderedLen(v Value) int {
	w := showWriter{limit: 0}
	renderShow(&w, v, inspectShow)
	return w.over
}

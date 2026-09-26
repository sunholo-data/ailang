package eval

import (
	"fmt"
	"sort"
	"strconv"
)

// ShowBounded renders v exactly as v.String() would, but stops WRITING once
// maxBytes have been produced and only COUNTS the remainder. The result is byte
// for byte what a caller would get from rendering the whole value and then
// truncating it — prefix plus "…(+N bytes elided)" — without the whole value
// ever existing as one string.
//
// This is the renderer the trace collector sites use (M-V1-MEMORY-FOOTPRINT M1).
// The old shape was `truncate(v.String(), max)`: correct retention, unbounded
// peak, because String() on a growing accumulator materialises O(n) per call
// and O(n²) over a recursion before the cap discards all but 1 KB of it.
//
// A budget <= 0 means unbounded and delegates to String(). Container String()
// methods keep their unbounded semantics: `show`, `println` and error messages
// are untouched by this — only trace rendering is bounded.
func ShowBounded(v Value, maxBytes int) string {
	if maxBytes <= 0 {
		return v.String()
	}
	w := boundedWriter{buf: make([]byte, 0, maxBytes), budget: maxBytes}
	w.render(v)
	if w.overflow == 0 {
		return string(w.buf)
	}
	return ShowBoundedOverflow(w.buf, w.overflow)
}

// ShowBoundedOverflow formats a truncated rendering: prefix plus the elided
// byte count.
func ShowBoundedOverflow(prefix []byte, overflow int) string {
	return fmt.Sprintf("%s…(+%d bytes elided)", prefix, overflow)
}

// RenderedLen reports len(v.String()) without building the string. Used for
// the redacted-values descriptor, which records only a byte count.
func RenderedLen(v Value) int {
	w := boundedWriter{countOnly: true}
	w.render(v)
	return w.overflow
}

// boundedWriter appends into buf until budget is spent, then counts.
type boundedWriter struct {
	buf       []byte
	budget    int
	overflow  int
	countOnly bool
	redact    bool // withhold credentials (ShowTraceBounded, trace_redact.go)
	scratch   [24]byte
}

func (w *boundedWriter) writeString(s string) {
	if w.countOnly {
		w.overflow += len(s)
		return
	}
	room := w.budget - len(w.buf)
	if room >= len(s) {
		w.buf = append(w.buf, s...)
		return
	}
	if room > 0 {
		w.buf = append(w.buf, s[:room]...)
		w.overflow += len(s) - room
		return
	}
	w.overflow += len(s)
}

func (w *boundedWriter) writeBytes(b []byte) {
	if w.countOnly {
		w.overflow += len(b)
		return
	}
	room := w.budget - len(w.buf)
	if room >= len(b) {
		w.buf = append(w.buf, b...)
		return
	}
	if room > 0 {
		w.buf = append(w.buf, b[:room]...)
		w.overflow += len(b) - room
		return
	}
	w.overflow += len(b)
}

func (w *boundedWriter) writeByte(c byte) {
	if !w.countOnly && len(w.buf) < w.budget {
		w.buf = append(w.buf, c)
		return
	}
	w.overflow++
}

// render mirrors each container's String() method element by element. Any
// value type not listed here — scalars, handles, closures — is rendered via
// its own String(), which is small by construction for those types.
func (w *boundedWriter) render(v Value) {
	switch val := v.(type) {
	case *IntValue:
		w.writeBytes(strconv.AppendInt(w.scratch[:0], int64(val.Value), 10))
	case *StringValue:
		if w.redact && isCredentialString(val.Value) {
			w.writeString(RedactedMarker)
			return
		}
		w.writeString(val.Value)
	case *ListValue:
		w.renderSeq("[", "]", val.Elements)
	case *ArrayValue:
		w.renderSeq("#[", "]", val.Elements)
	case *TupleValue:
		if w.redact && len(val.Elements) == 2 {
			if name, ok := val.Elements[0].(*StringValue); ok && IsSensitiveHeaderName(name.Value) {
				w.renderSeq("(", ")", []Value{name, &StringValue{Value: RedactedMarker}})
				return
			}
		}
		w.renderSeq("(", ")", val.Elements)
	case *TaggedValue:
		w.writeString(val.CtorName)
		if len(val.Fields) > 0 {
			w.renderSeq("(", ")", val.Fields)
		}
	case *RecordValue:
		keys := make([]string, 0, len(val.Fields))
		for k := range val.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		redactValue := false
		if w.redact {
			if name, ok := headerEntryName(val); ok && IsSensitiveHeaderName(name) {
				redactValue = true
			}
		}
		w.writeByte('{')
		for i, k := range keys {
			if i > 0 {
				w.writeString(", ")
			}
			w.writeString(k)
			w.writeString(": ")
			if w.redact && ((redactValue && k == "value") || IsSensitiveHeaderName(k)) {
				w.writeString(RedactedMarker)
				continue
			}
			w.render(val.Fields[k])
		}
		w.writeByte('}')
	case *MapValue:
		if len(val.Entries) == 0 {
			w.writeString("Map{}")
			return
		}
		keys := make([]string, 0, len(val.Entries))
		for k := range val.Entries {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		w.writeString("Map{")
		for i, k := range keys {
			if i > 0 {
				w.writeString(", ")
			}
			entry := val.Entries[k]
			w.render(entry.Key)
			w.writeString(": ")
			w.render(entry.Value)
		}
		w.writeByte('}')
	case *IndirectValue:
		if val.Cell.Init {
			w.render(val.Cell.Val)
			return
		}
		w.writeString("<uninitialized>")
	default:
		w.writeString(v.String())
	}
}

func (w *boundedWriter) renderSeq(open, close string, elems []Value) {
	w.writeString(open)
	for i, e := range elems {
		if i > 0 {
			w.writeString(", ")
		}
		w.render(e)
	}
	w.writeString(close)
}

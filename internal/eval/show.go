package eval

import (
	"encoding/hex"
	"math"
	"sort"
	"strconv"
	"unicode/utf8"
	"unsafe"

	"github.com/sunholo-data/ailang/internal/strutil"
)

// One renderer for `show`, shared by the `show` builtin (internal/builtins),
// the bytecode VM (internal/vm), string interpolation (the parser wraps every
// `${…}` hole in show), the display paths (REPL, `run --print`, test-failure
// messages) and the trace renderer (show_bounded.go). Each engine describes its
// own values as ShowNodes and RenderShow makes every formatting decision, so the
// two engines cannot disagree on spelling (#1453).
//
// The contract (M-V1-SHOW-TOTAL, design_docs/planned/v1_0_0/m-v1-show-total.md):
//
//   - total: every part of a value is rendered; there is no width, depth or
//     byte limit (the old 80-column / depth-3 elision is gone);
//   - deterministic: a pure function of the value, the same on every engine;
//   - injective per type: a string NESTED inside a container or constructor is
//     double-quoted and escaped (strutil.Quote); a TOP-LEVEL string is returned
//     unchanged, so `show(s) == s` and `"${s}"` keep working.
//
// Cost is O(size of output): the value is walked with an explicit stack (no Go
// recursion, so a million-deep value cannot overflow the goroutine stack) and
// written into one buffer. Bounded callers (showPreview, traces) use the same
// walk with a budget and only COUNT what lies past it.

// ShowKind says how RenderShow lays out a node.
type ShowKind uint8

const (
	// ShowText is a leaf written verbatim at every depth (bool, unit,
	// functions, markers).
	ShowText ShowKind = iota
	// ShowString is a string payload: bare at top level, quoted and escaped
	// when nested.
	ShowString
	// ShowInt is an integer leaf (Int).
	ShowInt
	// ShowFloat is a float leaf (Float); RenderShow spells it.
	ShowFloat
	// ShowBytes is a byte string (Bytes), rendered <bytes:HEX> in full.
	ShowBytes
	// ShowList, ShowArray and ShowTuple render Items in order.
	ShowList
	ShowArray
	ShowTuple
	// ShowRecord renders Names[i]: Items[i], in byte order of the name.
	ShowRecord
	// ShowCtor renders Text (the constructor name) applied to Items.
	ShowCtor
	// ShowMap renders Items as alternating key, value, in the order given.
	ShowMap
	// ShowSame renders Items[0] at the same depth (an indirection).
	ShowSame
)

// ShowNode is one engine value as RenderShow sees it. V is the engine's value
// type; Items hold the children, so an engine can hand over its own slice
// without boxing each element.
type ShowNode[V any] struct {
	Kind  ShowKind
	Text  string
	Int   int64
	Float float64
	Bytes []byte
	Items []V
	Names []string
}

// RenderShow renders v in full, using inspect to describe each value.
func RenderShow[V any](v V, inspect func(V) ShowNode[V]) string {
	w := showWriter{limit: -1}
	renderShow(&w, v, inspect)
	return unsafe.String(unsafe.SliceData(w.buf), len(w.buf)) // w.buf is never written again
}

// PreviewMarker formats the tail showPreview appends after a cut: the exact
// number of code points left out. Deliberately not valid AILANG or JSON.
func PreviewMarker(omitted int) string {
	return "…(+" + strconv.Itoa(omitted) + " chars)"
}

// RenderShowPreview renders a preview of v: its show rendering, except that a
// string is quoted and escaped at the top level too (as if nested). If that
// rendering has at most maxChars code points it is returned whole; otherwise
// its first maxChars code points followed by PreviewMarker(n), n being the
// exact number left out. It never builds more than maxChars code points of
// output: past the budget the walk only counts. maxChars <= 0 is a budget of 0.
//
// Quoting a top-level string is what makes a cut preview impossible to
// mistake for a complete one: a bare string can contain the marker text
// itself (showPreview("abc…(+3 chars)", 100) would equal
// showPreview("abcdef", 3)); a quoted one cannot, because an untruncated
// string preview always ends in its closing quote and the marker never does,
// and no other complete rendering contains "…" outside a quoted string.
func RenderShowPreview[V any](v V, inspect func(V) ShowNode[V], maxChars int) string {
	if maxChars < 0 {
		maxChars = 0
	}
	w := showWriter{limit: maxChars, runes: true, quoteTop: true, buf: make([]byte, 0, min(maxChars, 256))}
	renderShow(&w, v, inspect)
	if w.over == 0 {
		return string(w.buf)
	}
	return string(w.buf) + PreviewMarker(w.over)
}

// Show renders v the way the `show` builtin does.
func Show(v Value) string { return RenderShow(v, inspectShow) }

// ShowPreview renders v the way std/string.showPreview does.
func ShowPreview(v Value, maxChars int) string {
	return RenderShowPreview(v, inspectShow, maxChars)
}

// showWriter collects rendered text. limit < 0 is unbounded; otherwise the
// first limit units (bytes, or code points when runes is set) are kept and the
// rest only counted in over.
type showWriter struct {
	buf   []byte
	limit int
	runes bool
	used  int
	over  int
	// quoteTop quotes a top-level string as if it were nested (previews).
	quoteTop bool
	scratch  [64]byte
}

// WriteString implements io.StringWriter (strutil.QuoteTo writes through it).
func (w *showWriter) WriteString(s string) (int, error) {
	switch {
	case w.limit < 0:
		w.buf = append(growDoubling(w.buf, len(s)), s...)
	case !w.runes:
		room := w.limit - w.used
		switch {
		case room <= 0:
			w.over += len(s)
		case room >= len(s):
			w.buf = append(w.buf, s...)
			w.used += len(s)
		default:
			w.buf = append(w.buf, s[:room]...)
			w.used += room
			w.over += len(s) - room
		}
	default:
		room := w.limit - w.used
		if room <= 0 {
			w.over += utf8.RuneCountInString(s)
			break
		}
		// Keep up to room code points; an invalid byte counts as one, as
		// in std/string.length, so a cut never splits a valid sequence.
		i, n := 0, 0
		for i < len(s) && n < room {
			_, size := utf8.DecodeRuneInString(s[i:])
			i += size
			n++
		}
		w.buf = append(w.buf, s[:i]...)
		w.used += n
		if i < len(s) {
			w.over += utf8.RuneCountInString(s[i:])
		}
	}
	return len(s), nil
}

func (w *showWriter) writeBytes(b []byte) {
	if w.limit < 0 {
		w.buf = append(w.buf, b...)
		return
	}
	// Scratch-backed and short; the conversion does not escape.
	_, _ = w.WriteString(string(b))
}

// The render state is two stacks. closes holds one byte per open container
// (which bracket closes it), so a value nested a million deep costs a megabyte
// of state, not a million Go frames. frames holds only the containers that
// still have children to hand out: a container is popped from frames as soon
// as its LAST child is handed out, so a chain such as S(S(S(…))) or a list's
// final element keeps no frame at all. Every container in closes deeper than
// the innermost frame has therefore been fully rendered whenever a child
// finishes, and is closed then.
const (
	closeBracket byte = iota // ]
	closeParen               // )
	closeBrace               // }
)

var closeText = [...]string{closeBracket: "]", closeParen: ")", closeBrace: "}"}

// showFrame is a container that still has children to render.
type showFrame[V any] struct {
	items []V
	rec   *showRecordOrder // records only
	next  int32
	depth int32 // len(closes) when the container was opened
	kind  ShowKind
}

// showRecordOrder is a record's field names and the order to visit them in.
type showRecordOrder struct {
	names []string
	order []int
}

// renderShow walks v depth-first with an explicit stack, writing into w.
func renderShow[V any](w *showWriter, root V, inspect func(V) ShowNode[V]) {
	var frames []showFrame[V]
	var closes []byte
	v := root
	for {
		n := inspect(v)
		for n.Kind == ShowSame {
			n = inspect(n.Items[0])
		}
		switch n.Kind {
		case ShowString:
			if len(closes) > 0 || w.quoteTop {
				strutil.QuoteTo(w, n.Text)
			} else {
				_, _ = w.WriteString(n.Text)
			}
		case ShowInt:
			w.writeBytes(strconv.AppendInt(w.scratch[:0], n.Int, 10))
		case ShowFloat:
			w.writeBytes(appendShowFloat(w.scratch[:0], n.Float))
		case ShowBytes:
			writeShowBytes(w, n.Bytes)
		case ShowList:
			frames, closes = openContainer(w, frames, closes, n, "[", closeBracket)
		case ShowArray:
			frames, closes = openContainer(w, frames, closes, n, "#[", closeBracket)
		case ShowTuple:
			frames, closes = openContainer(w, frames, closes, n, "(", closeParen)
		case ShowRecord:
			frames, closes = openContainer(w, frames, closes, n, "{", closeBrace)
		case ShowMap:
			// Map{...} is deliberate debug notation: AILANG has no map literal.
			frames, closes = openContainer(w, frames, closes, n, "Map{", closeBrace)
		case ShowCtor:
			_, _ = w.WriteString(n.Text)
			if len(n.Items) > 0 {
				frames, closes = openContainer(w, frames, closes, n, "(", closeParen)
			}
		default: // ShowText
			_, _ = w.WriteString(n.Text)
		}

		// Close every container that has been fully rendered: all of them
		// above the innermost one that still has children (all of them when
		// none has).
		keep := 0
		if len(frames) > 0 {
			keep = int(frames[len(frames)-1].depth)
		}
		for len(closes) > keep {
			_, _ = w.WriteString(closeText[closes[len(closes)-1]])
			closes = closes[:len(closes)-1]
		}
		if len(frames) == 0 {
			return
		}
		v, frames = nextChild(w, frames)
	}
}

// openContainer writes a container's opening text and pushes it, or writes
// the whole empty container.
func openContainer[V any](w *showWriter, frames []showFrame[V], closes []byte, n ShowNode[V], open string, closer byte) ([]showFrame[V], []byte) {
	_, _ = w.WriteString(open)
	if len(n.Items) == 0 {
		_, _ = w.WriteString(closeText[closer])
		return frames, closes
	}
	closes = append(growDoubling(closes, 1), closer)
	f := showFrame[V]{kind: n.Kind, items: n.Items, depth: int32(len(closes))}
	if n.Kind == ShowRecord {
		rec := &showRecordOrder{names: n.Names, order: make([]int, len(n.Names))}
		for i := range rec.order {
			rec.order[i] = i
		}
		sort.Slice(rec.order, func(a, b int) bool { return n.Names[rec.order[a]] < n.Names[rec.order[b]] })
		f.rec = rec
	}
	return append(growDoubling(frames, 1), f), closes
}

// nextChild writes the separator before the next child of the innermost frame
// and returns that child. Handing out the last child pops the frame (its
// bracket stays on closes until the child is done).
func nextChild[V any](w *showWriter, frames []showFrame[V]) (V, []showFrame[V]) {
	f := &frames[len(frames)-1]
	i := int(f.next)
	f.next++
	child := f.items[i]
	switch f.kind {
	case ShowRecord:
		if i > 0 {
			_, _ = w.WriteString(", ")
		}
		j := f.rec.order[i]
		_, _ = w.WriteString(f.rec.names[j])
		_, _ = w.WriteString(": ")
		child = f.items[j]
	case ShowMap:
		if i%2 == 1 {
			_, _ = w.WriteString(": ")
		} else if i > 0 {
			_, _ = w.WriteString(", ")
		}
	default:
		if i > 0 {
			_, _ = w.WriteString(", ")
		}
	}
	if int(f.next) == len(f.items) {
		frames = frames[:len(frames)-1]
	}
	return child, frames
}

// growDoubling returns s with room for n more elements, doubling the capacity
// when it must grow. append's own policy grows large slices by 1.25x, which
// re-copies a multi-megabyte rendering about five times over; doubling keeps
// the total bytes allocated for a rendering within a small constant of its
// size (TestShowDeepAndWideAreLinear).
func growDoubling[T any](s []T, n int) []T {
	if cap(s)-len(s) >= n {
		return s
	}
	c := 2 * cap(s)
	if c < len(s)+n {
		c = len(s) + n
	}
	if c < 64 {
		c = 64
	}
	grown := make([]T, len(s), c)
	copy(grown, s)
	return grown
}

// appendShowFloat spells f: shortest round-trip digits without an exponent,
// ".0" appended to a whole number, -0.0 distinct from 0.0, NaN / Inf / -Inf.
func appendShowFloat(dst []byte, f float64) []byte {
	switch {
	case math.IsNaN(f):
		return append(dst, "NaN"...)
	case math.IsInf(f, 1):
		return append(dst, "Inf"...)
	case math.IsInf(f, -1):
		return append(dst, "-Inf"...)
	}
	start := len(dst)
	dst = strconv.AppendFloat(dst, f, 'f', -1, 64)
	for _, c := range dst[start:] {
		if c == '.' {
			return dst
		}
	}
	return append(dst, ".0"...)
}

// writeShowBytes renders b as <bytes:HEX>, lowercase, in full, encoding a
// chunk at a time so a bounded writer never sees the whole hex string.
func writeShowBytes(w *showWriter, b []byte) {
	_, _ = w.WriteString("<bytes:")
	for len(b) > 0 {
		chunk := b[:min(len(b), len(w.scratch)/2)]
		n := hex.Encode(w.scratch[:], chunk)
		w.writeBytes(w.scratch[:n])
		b = b[len(chunk):]
	}
	_, _ = w.WriteString(">")
}

// inspectShow describes an evaluator value for RenderShow.
func inspectShow(x Value) ShowNode[Value] {
	switch val := x.(type) {
	case *IntValue:
		return ShowNode[Value]{Kind: ShowInt, Int: int64(val.Value)}
	case *FloatValue:
		return ShowNode[Value]{Kind: ShowFloat, Float: val.Value}
	case *BoolValue:
		return ShowNode[Value]{Text: strconv.FormatBool(val.Value)}
	case *StringValue:
		return ShowNode[Value]{Kind: ShowString, Text: val.Value}
	case *ListValue:
		return ShowNode[Value]{Kind: ShowList, Items: val.Elements}
	case *ArrayValue:
		return ShowNode[Value]{Kind: ShowArray, Items: val.Elements()}
	case *TupleValue:
		return ShowNode[Value]{Kind: ShowTuple, Items: val.Elements}
	case *MapValue:
		// Encoded-key order: the order M.toList and M.keys use.
		keys := make([]string, 0, len(val.Entries))
		for key := range val.Entries {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]Value, 0, 2*len(keys))
		for _, key := range keys {
			items = append(items, val.Entries[key].Key, val.Entries[key].Value)
		}
		return ShowNode[Value]{Kind: ShowMap, Items: items}
	case *RecordValue:
		names := make([]string, 0, len(val.Fields))
		items := make([]Value, 0, len(val.Fields))
		for k, fv := range val.Fields {
			names = append(names, k)
			items = append(items, fv)
		}
		return ShowNode[Value]{Kind: ShowRecord, Names: names, Items: items}
	case *TaggedValue:
		return ShowNode[Value]{Kind: ShowCtor, Text: val.CtorName, Items: val.Fields}
	case *UnitValue:
		return ShowNode[Value]{Text: "()"}
	case *FunctionValue, *BuiltinFunction, *ConstructorClosure:
		return ShowNode[Value]{Text: "<function>"}
	case *BytesValue:
		return ShowNode[Value]{Kind: ShowBytes, Bytes: val.Value}
	case *IndirectValue:
		if val.Cell == nil || !val.Cell.Init || val.Cell.Val == nil {
			return ShowNode[Value]{Text: "<uninitialized>"}
		}
		return ShowNode[Value]{Kind: ShowSame, Items: []Value{val.Cell.Val}}
	case *ErrorValue:
		return ShowNode[Value]{Text: "Error: " + val.Message}
	case *tailCall:
		// Never escapes applyFunctionValue's loop; rendered only so a
		// violation of that invariant is visible rather than "<unknown>".
		return ShowNode[Value]{Text: val.String()}
	}
	return ShowNode[Value]{Text: "<unknown>"}
}

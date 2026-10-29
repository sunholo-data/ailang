// Package pcoll owns the canonical key grammar for AILANG Map and Set keys,
// shared by the evaluator and the bytecode VM (M-PERSISTENT-COLLECTIONS 1a).
//
// It is a stdlib-only leaf: it imports neither engine, so it never sees an
// eval.Value or a bytecode.Value. Each engine keeps a small walker over its own
// value type that drives a KeyWriter; the byte grammar lives only here. The two
// walkers are held together by a cross-engine golden test in internal/vm.
//
// # Grammar
//
// Top-level atoms keep the encodings that shipped in v0.11.0 byte for byte, so
// every existing map keeps its keys and its iteration order (Decision D2):
//
//	int     "i:" + decimal          (so int keys iterate lexically: 10 < 2 < 9)
//	string  "s:" + raw bytes
//	bool    "b:true" | "b:false"
//
// New kinds:
//
//	unit    "u:"
//	float   "f:" + 16 hex digits of the order-preserving bit transform; NaN is
//	        rejected and -0.0 is folded into +0.0 so key equality agrees with ==
//	tuple   "t:" + elem(e1) elem(e2) ...
//	record  "r:" + elem(name1) elem(v1) elem(name2) elem(v2) ...   (names sorted)
//	ctor    "c:" + elem(ctorName) elem(f1) elem(f2) ...
//
// where elem(x) = escape(x) + "\x00" and escape replaces each NUL with
// "\x00\xff". A terminator is never followed by 0xff (every element starts with
// a kind letter or an identifier), so the encoding is injective, and because
// the terminator sorts below every other byte, composite keys sort element by
// element, each element in the order it would have as a top-level key.
//
// An ADT constructor is identified by its constructor name alone. The VM's
// ADT objects carry no type or module name, and a well-typed map only ever
// holds constructors of one type, whose names are unique within it.
package pcoll

import (
	"fmt"
	"math"
	"strconv"
)

// SupportedKeyKinds is the human-readable domain named by UnsupportedKeyError.
const SupportedKeyKinds = "int, string, bool, float, unit, and tuples, records and constructors of those"

// UnsupportedKeyError reports a value that cannot be a Map or Set key, naming
// its type. Every map operation returns it; none answers false instead.
type UnsupportedKeyError struct{ Kind string }

func (e *UnsupportedKeyError) Error() string {
	return fmt.Sprintf("unsupported map key type: %s (supported: %s)", e.Kind, SupportedKeyKinds)
}

type frameKind uint8

const (
	frameTuple frameKind = iota
	frameRecord
	frameCtor
)

type frame struct {
	kind frameKind
	buf  []byte
	// records only: last field name written, and whether the next element
	// must be a field name.
	lastField string
	wantName  bool
	nfields   int
}

// KeyWriter builds one canonical key. Walkers call the atom methods and the
// Begin/End pairs in value order, then Key. The first error sticks.
type KeyWriter struct {
	top    []byte
	topSet bool
	stack  []frame
	err    error
}

// fail records the first error.
func (w *KeyWriter) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

// Fail records an error raised by a walker (for example a value that must be
// forced first and could not be).
func (w *KeyWriter) Fail(err error) { w.fail(err) }

// Unsupported records that a value of the named kind cannot be a key.
func (w *KeyWriter) Unsupported(kind string) { w.fail(&UnsupportedKeyError{Kind: kind}) }

// emit places one complete element encoding into the current position.
func (w *KeyWriter) emit(enc []byte) {
	if w.err != nil {
		return
	}
	if len(w.stack) == 0 {
		if w.topSet {
			w.fail(fmt.Errorf("pcoll: key writer received a second top-level value"))
			return
		}
		w.top, w.topSet = enc, true
		return
	}
	f := &w.stack[len(w.stack)-1]
	if f.kind == frameRecord {
		if f.wantName {
			w.fail(fmt.Errorf("pcoll: record value written before its field name"))
			return
		}
		f.wantName = true
	}
	f.buf = appendElem(f.buf, enc)
}

// appendElem appends escape(enc) + "\x00".
func appendElem(dst, enc []byte) []byte {
	for _, c := range enc {
		if c == 0 {
			dst = append(dst, 0, 0xff)
			continue
		}
		dst = append(dst, c)
	}
	return append(dst, 0)
}

// Int writes an int key.
func (w *KeyWriter) Int(n int64) {
	w.emit(strconv.AppendInt([]byte("i:"), n, 10))
}

// Str writes a string key.
func (w *KeyWriter) Str(s string) {
	w.emit(append([]byte("s:"), s...))
}

// Bool writes a bool key.
func (w *KeyWriter) Bool(b bool) {
	if b {
		w.emit([]byte("b:true"))
		return
	}
	w.emit([]byte("b:false"))
}

// Unit writes the unit key.
func (w *KeyWriter) Unit() { w.emit([]byte("u:")) }

// Float writes a float key. NaN is rejected: it is not equal to itself, so a
// NaN key could be inserted but never found. -0.0 is folded into +0.0 because
// the two compare equal.
func (w *KeyWriter) Float(f float64) {
	if math.IsNaN(f) {
		w.fail(fmt.Errorf("unsupported map key: NaN (NaN is not equal to itself, so it could never be looked up)"))
		return
	}
	if f == 0 {
		f = 0 // folds -0.0
	}
	bits := math.Float64bits(f)
	if bits&(1<<63) != 0 {
		bits = ^bits
	} else {
		bits |= 1 << 63
	}
	enc := []byte("f:")
	enc = append(enc, fmt.Sprintf("%016x", bits)...)
	w.emit(enc)
}

func (w *KeyWriter) begin(kind frameKind, prefix string) {
	if w.err != nil {
		return
	}
	w.stack = append(w.stack, frame{kind: kind, buf: append([]byte(nil), prefix...), wantName: kind == frameRecord})
}

// BeginTuple opens a tuple; its elements follow, then End.
func (w *KeyWriter) BeginTuple() { w.begin(frameTuple, "t:") }

// BeginRecord opens a record; Field(name) then the value follow for each
// field, in strictly increasing name order, then End.
func (w *KeyWriter) BeginRecord() { w.begin(frameRecord, "r:") }

// Field names the next record field. Names must be strictly increasing, so a
// walker that forgets to sort fails loudly instead of producing a second key
// for the same record.
func (w *KeyWriter) Field(name string) {
	if w.err != nil {
		return
	}
	if len(w.stack) == 0 || w.stack[len(w.stack)-1].kind != frameRecord {
		w.fail(fmt.Errorf("pcoll: Field(%q) outside a record", name))
		return
	}
	f := &w.stack[len(w.stack)-1]
	if !f.wantName {
		w.fail(fmt.Errorf("pcoll: record field %q written before the previous field's value", name))
		return
	}
	if f.nfields > 0 && name <= f.lastField {
		w.fail(fmt.Errorf("pcoll: record fields out of order: %q after %q", name, f.lastField))
		return
	}
	f.buf = appendElem(f.buf, []byte(name))
	f.lastField, f.wantName = name, false
	f.nfields++
}

// BeginCtor opens an ADT constructor value; its fields follow, then End.
func (w *KeyWriter) BeginCtor(ctor string) {
	w.begin(frameCtor, "c:")
	if w.err == nil {
		f := &w.stack[len(w.stack)-1]
		f.buf = appendElem(f.buf, []byte(ctor))
	}
}

// End closes the innermost Begin.
func (w *KeyWriter) End() {
	if w.err != nil {
		return
	}
	if len(w.stack) == 0 {
		w.fail(fmt.Errorf("pcoll: End without Begin"))
		return
	}
	f := w.stack[len(w.stack)-1]
	if f.kind == frameRecord && !f.wantName {
		w.fail(fmt.Errorf("pcoll: record field %q has no value", f.lastField))
		return
	}
	w.stack = w.stack[:len(w.stack)-1]
	w.emit(f.buf)
}

// Key returns the canonical key, or the first error recorded.
func (w *KeyWriter) Key() (string, error) {
	if w.err != nil {
		return "", w.err
	}
	if len(w.stack) != 0 {
		return "", fmt.Errorf("pcoll: %d unclosed Begin(s)", len(w.stack))
	}
	if !w.topSet {
		return "", fmt.Errorf("pcoll: empty key")
	}
	return string(w.top), nil
}

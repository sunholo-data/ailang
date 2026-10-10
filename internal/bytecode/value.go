package bytecode

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ValueTag identifies the runtime type of a Value.
//
// Tag identity is part of the in-memory representation, not the bytecode
// format. Numeric values may be reordered freely.
type ValueTag uint8

const (
	TagInt ValueTag = iota
	TagFloat
	TagBool
	TagUnit
	TagString
	TagList
	TagTuple
	TagRecord
	TagClosure
	TagADT
	TagArray
	TagBytes
)

// String returns the human-readable tag name. Used for error messages and
// debugging.
func (t ValueTag) String() string {
	switch t {
	case TagInt:
		return "Int"
	case TagFloat:
		return "Float"
	case TagBool:
		return "Bool"
	case TagUnit:
		return "Unit"
	case TagString:
		return "String"
	case TagList:
		return "List"
	case TagTuple:
		return "Tuple"
	case TagRecord:
		return "Record"
	case TagClosure:
		return "Closure"
	case TagADT:
		return "ADT"
	case TagArray:
		return "Array"
	case TagBytes:
		return "Bytes"
	}
	return fmt.Sprintf("UnknownTag(%d)", uint8(t))
}

// Value is the Phase 2B tagged-struct representation of an AILANG runtime
// value, per M-BYTECODE-VM §3.2.
//
// Primitives (Int, Float, Bool, Unit) are unboxed into the dedicated fields.
// Heap objects (String, List, Tuple, Record, Closure, ADT) live in Obj as
// pointers to the corresponding *Obj struct.
//
// This is intentionally larger than a NaN-boxed uint64 (~32 bytes vs 8). The
// design doc gates NaN-boxing on Phase 2D benchmark evidence — do not switch
// representation until we have data showing value dispatch is the bottleneck.
//
// Field order is load-bearing for size: Tag and Bool share one word, so a
// Value is 40 bytes, not the 48 that Tag, Int, Flt, Bool, Obj in declaration
// order pads to. Every list cons / list build copies whole Values, so this
// width is what the VM pays per element on copy-heavy shapes (#1501 Phase 3:
// consrepro allocated 3.0x the evaluator's bytes at 48 B).
type Value struct {
	Tag  ValueTag
	Bool bool
	Int  int64
	Flt  float64
	Obj  any // *StringObj, *ListObj, *TupleObj, *RecordObj, *ClosureObj, *ADTObj, *ArrayObj, *BytesObj
}

// --- Heap object types -------------------------------------------------------

// StringObj wraps an immutable string. Strings are reference-shared across
// VM/evaluator boundaries since they cannot be mutated.
type StringObj struct {
	S string
}

// ListObj is a slice-backed list. The design doc (§3.2) leaves the choice of
// cons-cell vs slice open; Phase 2B uses slices for cache locality and simple
// indexing. Revisit in Phase 2D if benchmarks demand.
type ListObj struct {
	Elems []Value
}

// ArrayObj is an immutable Array[a] (M-NUMERICS-VEC-ARRAY M4). Like the
// evaluator's ArrayValue it has two stores and exactly one is used: Floats for
// an all-float array (shared with the evaluator across the bridge, no copy),
// Elems for everything else. Neither slice is ever mutated after construction.
type ArrayObj struct {
	Elems  []Value
	Floats []float64
}

// Len returns the number of elements.
func (a *ArrayObj) Len() int {
	if a.Floats != nil {
		return len(a.Floats)
	}
	return len(a.Elems)
}

// At returns element i as a VM value. The caller checks bounds.
func (a *ArrayObj) At(i int) Value {
	if a.Floats != nil {
		return NewFloat(a.Floats[i])
	}
	return a.Elems[i]
}

// BytesObj is an immutable byte string. The VM has no bytes operations; it
// carries bytes between evaluator calls (M-NUMERICS-VEC-ARRAY M6), keeping
// the upload metadata so a round trip is lossless.
type BytesObj struct {
	B        []byte
	Filename string
	MimeType string
}

// TupleObj is a fixed-arity heterogeneous record-without-names.
type TupleObj struct {
	Elems []Value
}

// RecordField is a single field of a record. Records store fields in
// alphabetical order by Name (per §4.3 — A1 determinism).
type RecordField struct {
	Name  string
	Value Value
}

// RecordObj is an alphabetically-sorted set of named fields. The sort
// invariant is enforced by NewRecord; do not construct RecordObj directly.
type RecordObj struct {
	Fields []RecordField
}

// FuncPrototypeRef is a forward reference to a FuncPrototype defined in
// image.go. Defined here as an interface to avoid forcing a particular
// concrete type into the closure value.
//
// In practice the only implementer is *FuncPrototype, but using an interface
// keeps Value decoupled from the image format and lets tests substitute
// stub prototypes.
type FuncPrototypeRef interface {
	ProtoName() string
	NumRegisters() uint8
	NumParameters() uint8
}

// ClosureObj is a flat closure: the prototype plus a fixed array of captured
// values. Captures are copied at closure creation time (§3.3 — flat closures).
type ClosureObj struct {
	Proto    FuncPrototypeRef
	Captures []Value
}

// ADTObj is a tagged algebraic data type instance. Tag is a per-type local
// ordinal assigned during type elaboration; it is NOT globally unique. Type
// disambiguation is the compiler's responsibility (§4.3).
type ADTObj struct {
	Tag int
	// Ctor is the constructor name ("Some", "Circle"). It does not take part
	// in equality or dispatch — Tag does — but `show` needs it to render the
	// value the way the evaluator does (#1453).
	Ctor   string
	Fields []Value
}

// --- Constructors ------------------------------------------------------------

// NewInt constructs an Int value.
func NewInt(n int64) Value { return Value{Tag: TagInt, Int: n} }

// NewFloat constructs a Float value.
func NewFloat(f float64) Value { return Value{Tag: TagFloat, Flt: f} }

// NewBool constructs a Bool value.
func NewBool(b bool) Value { return Value{Tag: TagBool, Bool: b} }

// Unit returns the canonical unit value. Unit is a singleton from a semantic
// standpoint; the struct copy is cheap.
func Unit() Value { return Value{Tag: TagUnit} }

// NewString constructs a String value backed by a heap StringObj.
func NewString(s string) Value {
	return Value{Tag: TagString, Obj: &StringObj{S: s}}
}

// NewList constructs a List value from the given elements. The slice is
// retained as-is — callers must not mutate it after construction.
func NewList(elems []Value) Value {
	return Value{Tag: TagList, Obj: &ListObj{Elems: elems}}
}

// NewArray constructs a boxed Array value. The slice is retained as-is.
func NewArray(elems []Value) Value {
	return Value{Tag: TagArray, Obj: &ArrayObj{Elems: elems}}
}

// NewFloatArray constructs a packed Array[float] value. The slice is retained
// as-is and must not be mutated by anyone afterwards.
func NewFloatArray(xs []float64) Value {
	return Value{Tag: TagArray, Obj: &ArrayObj{Floats: xs}}
}

// AsArray returns the underlying ArrayObj. Panics if v is not an Array.
func (v Value) AsArray() *ArrayObj {
	if v.Tag != TagArray {
		panic(fmt.Sprintf("bytecode: AsArray called on %s", v.Tag))
	}
	return v.Obj.(*ArrayObj)
}

// NewBytes constructs a Bytes value. The slice is retained as-is and must
// not be mutated afterwards.
func NewBytes(b *BytesObj) Value {
	return Value{Tag: TagBytes, Obj: b}
}

// AsBytes returns the underlying BytesObj. Panics if v is not Bytes.
func (v Value) AsBytes() *BytesObj {
	if v.Tag != TagBytes {
		panic(fmt.Sprintf("bytecode: AsBytes called on %s", v.Tag))
	}
	return v.Obj.(*BytesObj)
}

// StdADTTag returns standard ADT constructor ordinals, in declaration order
// from std/option.ail, std/result.ail and std/terminal.ail. Host conversion and
// VM match compilation must use the same ordinals.
func StdADTTag(modulePath, typeName, ctor string) (int, bool) {
	switch {
	case modulePath == "std/option" && typeName == "Option":
		switch ctor {
		case "Some":
			return 0, true
		case "None":
			return 1, true
		}
	case modulePath == "std/result" && typeName == "Result":
		switch ctor {
		case "Ok":
			return 0, true
		case "Err":
			return 1, true
		}
	case modulePath == "std/terminal":
		var names []string
		switch typeName {
		case "TerminalSession":
			names = []string{"TerminalSession"}
		case "TerminalKey":
			names = []string{"Text", "Up", "Down", "Left", "Right", "Enter", "Escape", "Backspace", "Tab", "Home", "End", "PageUp", "PageDown", "Delete"}
		case "TerminalEvent":
			names = []string{"Key", "Resize", "EndOfInput", "Interrupted", "Idle"}
		case "TerminalError":
			names = []string{"Unsupported", "NotTTY", "Busy", "InvalidSession", "InvalidTimeout", "InputFailure", "QueryFailure", "CleanupFailure"}
		}
		for i, name := range names {
			if name == ctor {
				return i, true
			}
		}

	}
	return 0, false
}

// NewTuple constructs a Tuple value from the given elements.
func NewTuple(elems []Value) Value {
	return Value{Tag: TagTuple, Obj: &TupleObj{Elems: elems}}
}

// NewRecord constructs a Record value, sorting fields alphabetically by name.
// Duplicate field names cause a panic — the compiler must reject duplicates
// upstream.
func NewRecord(fields []RecordField) Value {
	sorted := make([]RecordField, len(fields))
	copy(sorted, fields)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Name == sorted[i-1].Name {
			panic(fmt.Sprintf("bytecode: duplicate record field %q", sorted[i].Name))
		}
	}
	return Value{Tag: TagRecord, Obj: &RecordObj{Fields: sorted}}
}

// NewClosure constructs a Closure value with flat-captured values.
func NewClosure(proto FuncPrototypeRef, captures []Value) Value {
	return Value{Tag: TagClosure, Obj: &ClosureObj{Proto: proto, Captures: captures}}
}

// NewADT constructs an ADT value with the given tag and fields.
func NewADT(tag int, ctor string, fields []Value) Value {
	return Value{Tag: TagADT, Obj: &ADTObj{Tag: tag, Ctor: ctor, Fields: fields}}
}

// --- Accessors (panic on tag mismatch — VM dispatch must check first) -------

// AsString returns the underlying string. Panics if v is not a String.
func (v Value) AsString() string {
	if v.Tag != TagString {
		panic(fmt.Sprintf("bytecode: AsString called on %s", v.Tag))
	}
	return v.Obj.(*StringObj).S
}

// AsList returns the underlying slice. Panics if v is not a List. The caller
// must not mutate the returned slice.
func (v Value) AsList() []Value {
	if v.Tag != TagList {
		panic(fmt.Sprintf("bytecode: AsList called on %s", v.Tag))
	}
	return v.Obj.(*ListObj).Elems
}

// AsTuple returns the underlying slice. Panics if v is not a Tuple.
func (v Value) AsTuple() []Value {
	if v.Tag != TagTuple {
		panic(fmt.Sprintf("bytecode: AsTuple called on %s", v.Tag))
	}
	return v.Obj.(*TupleObj).Elems
}

// AsRecord returns the underlying record fields (alphabetically sorted).
func (v Value) AsRecord() []RecordField {
	if v.Tag != TagRecord {
		panic(fmt.Sprintf("bytecode: AsRecord called on %s", v.Tag))
	}
	return v.Obj.(*RecordObj).Fields
}

// AsClosure returns the underlying closure object.
func (v Value) AsClosure() *ClosureObj {
	if v.Tag != TagClosure {
		panic(fmt.Sprintf("bytecode: AsClosure called on %s", v.Tag))
	}
	return v.Obj.(*ClosureObj)
}

// AsADT returns the underlying ADT object.
func (v Value) AsADT() *ADTObj {
	if v.Tag != TagADT {
		panic(fmt.Sprintf("bytecode: AsADT called on %s", v.Tag))
	}
	return v.Obj.(*ADTObj)
}

// --- Equality ---------------------------------------------------------------

// Equal reports whether two values are structurally equal under AILANG's
// canonical equality (§3.6). This is the equivalence relation used for
// constant-pool deduplication and for comparing test results between the
// VM and evaluator.
//
// Notes:
//   - Float NaN compares equal to NaN here (so NaN constants dedupe). The
//     `EQ` opcode at runtime uses IEEE semantics — NaN != NaN — and must NOT
//     route through this function.
//   - Records require both sides to be in alphabetical order, which is the
//     constructor's invariant.
//   - Closures compare by prototype identity and capture-by-capture equality.
//     Two closures wrapping the same prototype with different captures are
//     unequal.
func (v Value) Equal(other Value) bool {
	if v.Tag != other.Tag {
		return false
	}
	switch v.Tag {
	case TagInt:
		return v.Int == other.Int
	case TagFloat:
		// Treat NaN==NaN for dedup purposes; see doc comment.
		if math.IsNaN(v.Flt) && math.IsNaN(other.Flt) {
			return true
		}
		return v.Flt == other.Flt
	case TagBool:
		return v.Bool == other.Bool
	case TagUnit:
		return true
	case TagString:
		return v.Obj.(*StringObj).S == other.Obj.(*StringObj).S
	case TagList:
		a, b := v.Obj.(*ListObj).Elems, other.Obj.(*ListObj).Elems
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !a[i].Equal(b[i]) {
				return false
			}
		}
		return true
	case TagTuple:
		a, b := v.Obj.(*TupleObj).Elems, other.Obj.(*TupleObj).Elems
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !a[i].Equal(b[i]) {
				return false
			}
		}
		return true
	case TagBytes:
		return string(v.Obj.(*BytesObj).B) == string(other.Obj.(*BytesObj).B)
	case TagArray:
		a, b := v.Obj.(*ArrayObj), other.Obj.(*ArrayObj)
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !a.At(i).Equal(b.At(i)) {
				return false
			}
		}
		return true
	case TagRecord:
		a, b := v.Obj.(*RecordObj).Fields, other.Obj.(*RecordObj).Fields
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].Name != b[i].Name || !a[i].Value.Equal(b[i].Value) {
				return false
			}
		}
		return true
	case TagClosure:
		a, b := v.Obj.(*ClosureObj), other.Obj.(*ClosureObj)
		if a.Proto != b.Proto {
			return false
		}
		if len(a.Captures) != len(b.Captures) {
			return false
		}
		for i := range a.Captures {
			if !a.Captures[i].Equal(b.Captures[i]) {
				return false
			}
		}
		return true
	case TagADT:
		a, b := v.Obj.(*ADTObj), other.Obj.(*ADTObj)
		if a.Tag != b.Tag || len(a.Fields) != len(b.Fields) {
			return false
		}
		for i := range a.Fields {
			if !a.Fields[i].Equal(b.Fields[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// --- Debug formatting -------------------------------------------------------

// String renders a value for debugging. Output is intended to be readable, not
// machine-parseable, and not part of any wire format.
func (v Value) String() string {
	switch v.Tag {
	case TagInt:
		return fmt.Sprintf("%d", v.Int)
	case TagFloat:
		return fmt.Sprintf("%g", v.Flt)
	case TagBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case TagUnit:
		return "()"
	case TagString:
		return fmt.Sprintf("%q", v.Obj.(*StringObj).S)
	case TagList:
		var sb strings.Builder
		sb.WriteByte('[')
		for i, e := range v.Obj.(*ListObj).Elems {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(e.String())
		}
		sb.WriteByte(']')
		return sb.String()
	case TagBytes:
		return fmt.Sprintf("<bytes:%d>", len(v.Obj.(*BytesObj).B))
	case TagArray:
		a := v.Obj.(*ArrayObj)
		var sb strings.Builder
		sb.WriteString("#[")
		for i := 0; i < a.Len(); i++ {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(a.At(i).String())
		}
		sb.WriteByte(']')
		return sb.String()
	case TagTuple:
		var sb strings.Builder
		sb.WriteByte('(')
		for i, e := range v.Obj.(*TupleObj).Elems {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(e.String())
		}
		sb.WriteByte(')')
		return sb.String()
	case TagRecord:
		var sb strings.Builder
		sb.WriteByte('{')
		for i, f := range v.Obj.(*RecordObj).Fields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(f.Name)
			sb.WriteString(": ")
			sb.WriteString(f.Value.String())
		}
		sb.WriteByte('}')
		return sb.String()
	case TagClosure:
		c := v.Obj.(*ClosureObj)
		name := "<anon>"
		if c.Proto != nil {
			name = c.Proto.ProtoName()
		}
		return fmt.Sprintf("<closure %s/%d>", name, len(c.Captures))
	case TagADT:
		a := v.Obj.(*ADTObj)
		var sb strings.Builder
		fmt.Fprintf(&sb, "<adt#%d", a.Tag)
		for _, f := range a.Fields {
			sb.WriteByte(' ')
			sb.WriteString(f.String())
		}
		sb.WriteByte('>')
		return sb.String()
	}
	return fmt.Sprintf("<unknown tag %d>", v.Tag)
}

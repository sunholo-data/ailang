package eval

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/pcoll"
)

// MapKey returns the canonical key of v for Map and Set storage. It is the
// evaluator's walker over internal/pcoll's key grammar; the VM has the twin
// in bytecode.MapKey, and the cross-engine golden test in internal/vm keeps
// the two byte-identical. Distinct from String(), which is for display only.
//
// Encodable: int, string, bool, float (not NaN), unit, and tuples, records and
// ADT constructors of encodable values, recursively. Anything else (functions,
// lists, arrays, maps, bytes) is an error naming the type.
func MapKey(v Value) (string, error) {
	var w pcoll.KeyWriter
	writeMapKey(&w, v)
	return w.Key()
}

func writeMapKey(w *pcoll.KeyWriter, v Value) {
	switch k := v.(type) {
	case *IntValue:
		w.Int(int64(k.Value))
	case *StringValue:
		w.Str(k.Value)
	case *BoolValue:
		w.Bool(k.Value)
	case *FloatValue:
		w.Float(k.Value)
	case *UnitValue:
		w.Unit()
	case *TupleValue:
		w.BeginTuple()
		for _, e := range k.Elements {
			writeMapKey(w, e)
		}
		w.End()
	case *RecordValue:
		names := make([]string, 0, len(k.Fields))
		for name := range k.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		w.BeginRecord()
		for _, name := range names {
			w.Field(name)
			writeMapKey(w, k.Fields[name])
		}
		w.End()
	case *TaggedValue:
		w.BeginCtor(k.CtorName)
		for _, f := range k.Fields {
			writeMapKey(w, f)
		}
		w.End()
	case *IndirectValue:
		forced, err := k.Force()
		if err != nil {
			w.Fail(err)
			return
		}
		writeMapKey(w, forced)
	case *FunctionValue, *BuiltinFunction, *ConstructorClosure:
		w.Unsupported("function")
	case nil:
		w.Unsupported("<nil>")
	default:
		w.Unsupported(v.Type())
	}
}

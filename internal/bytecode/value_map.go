package bytecode

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/pcoll"
)

// MapEntry retains the original key alongside its canonical index.
type MapEntry struct{ Key, Value Value }

// MapObj is immutable after publication. Updates copy its entry table.
type MapObj struct{ Entries map[string]MapEntry }

// NewMap retains entries; callers must not mutate them after construction.
func NewMap(entries map[string]MapEntry) Value {
	return Value{Tag: TagMap, Obj: &MapObj{Entries: entries}}
}

func (v Value) AsMap() *MapObj {
	if v.Tag != TagMap {
		panic(fmt.Sprintf("bytecode: AsMap called on %s", v.Tag))
	}
	return v.Obj.(*MapObj)
}

// MapKey returns the canonical key of v: the VM's walker over internal/pcoll's
// key grammar, twin of eval.MapKey. The cross-engine golden test in
// internal/vm asserts the two produce byte-identical keys, which matters
// because maps cross the engine bridge with their canonical keys intact.
func MapKey(v Value) (string, error) {
	var w pcoll.KeyWriter
	writeMapKey(&w, v)
	return w.Key()
}

func writeMapKey(w *pcoll.KeyWriter, v Value) {
	switch v.Tag {
	case TagInt:
		w.Int(v.Int)
	case TagString:
		w.Str(v.AsString())
	case TagBool:
		w.Bool(v.Bool)
	case TagFloat:
		w.Float(v.Flt)
	case TagUnit:
		w.Unit()
	case TagTuple:
		w.BeginTuple()
		for _, e := range v.AsTuple() {
			writeMapKey(w, e)
		}
		w.End()
	case TagRecord:
		// RecordObj keeps fields sorted by name (NewRecord's invariant), the
		// order the grammar requires; Field rejects any other order loudly.
		w.BeginRecord()
		for _, f := range v.AsRecord() {
			w.Field(f.Name)
			writeMapKey(w, f.Value)
		}
		w.End()
	case TagADT:
		a := v.AsADT()
		w.BeginCtor(a.Ctor)
		for _, f := range a.Fields {
			writeMapKey(w, f)
		}
		w.End()
	case TagClosure:
		w.Unsupported("function")
	default:
		w.Unsupported(strings.ToLower(v.Tag.String()))
	}
}

func (m *MapObj) SortedKeys() []string {
	keys := make([]string, 0, len(m.Entries))
	for k := range m.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Lookup returns the value for key. An unencodable key is an error, never a
// silent miss (CLAUDE.md §2).
func (m *MapObj) Lookup(key Value) (Value, bool, error) {
	k, err := MapKey(key)
	if err != nil {
		return Value{}, false, err
	}
	entry, ok := m.Entries[k]
	return entry.Value, ok, nil
}

func (m *MapObj) Insert(key, val Value) (Value, error) {
	k, err := MapKey(key)
	if err != nil {
		return Value{}, err
	}
	entries := make(map[string]MapEntry, len(m.Entries)+1)
	for k, v := range m.Entries {
		entries[k] = v
	}
	entries[k] = MapEntry{Key: key, Value: val}
	return NewMap(entries), nil
}

func (m *MapObj) Remove(key Value) (Value, error) {
	k, err := MapKey(key)
	if err != nil {
		return Value{}, err
	}
	if _, ok := m.Entries[k]; !ok {
		return Value{Tag: TagMap, Obj: m}, nil
	}
	entries := make(map[string]MapEntry, len(m.Entries)-1)
	for old, v := range m.Entries {
		if old != k {
			entries[old] = v
		}
	}
	return NewMap(entries), nil
}

func (m *MapObj) Equal(other *MapObj, equal func(Value, Value) bool) bool {
	if len(m.Entries) != len(other.Entries) {
		return false
	}
	for k, v := range m.Entries {
		o, ok := other.Entries[k]
		if !ok || !equal(v.Value, o.Value) {
			return false
		}
	}
	return true
}

func (m *MapObj) String() string {
	parts := make([]string, 0, len(m.Entries))
	for _, k := range m.SortedKeys() {
		e := m.Entries[k]
		parts = append(parts, e.Key.String()+": "+e.Value.String())
	}
	return "Map{" + strings.Join(parts, ", ") + "}"
}

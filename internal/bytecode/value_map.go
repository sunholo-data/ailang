package bytecode

import (
	"fmt"
	"sort"
	"strings"
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

// MapKey matches eval.MapKey, including lexical integer iteration order.
func MapKey(v Value) (string, error) {
	switch v.Tag {
	case TagInt:
		return fmt.Sprintf("i:%d", v.Int), nil
	case TagString:
		return "s:" + v.AsString(), nil
	case TagBool:
		return fmt.Sprintf("b:%t", v.Bool), nil
	default:
		return "", fmt.Errorf("unsupported map key type: %s (only int, string, bool supported)", v.Tag)
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

func (m *MapObj) Lookup(key Value) (Value, bool) {
	k, err := MapKey(key)
	if err != nil {
		return Value{}, false
	}
	entry, ok := m.Entries[k]
	return entry.Value, ok
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

func (m *MapObj) Remove(key Value) Value {
	k, err := MapKey(key)
	if _, ok := m.Entries[k]; err != nil || !ok {
		return Value{Tag: TagMap, Obj: m}
	}
	entries := make(map[string]MapEntry, len(m.Entries)-1)
	for old, v := range m.Entries {
		if old != k {
			entries[old] = v
		}
	}
	return NewMap(entries)
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

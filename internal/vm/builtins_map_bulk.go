package vm

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/bytecode"
)

func builtinMapFromList(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_from_list", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("map_from_list: expected List, got %s", args[0].Tag)
	}
	entries := make(map[string]bytecode.MapEntry, len(args[0].AsList()))
	for i, e := range args[0].AsList() {
		if e.Tag != bytecode.TagTuple {
			return bytecode.Value{}, fmt.Errorf("map_from_list: element %d is not a tuple, got %s", i, e.Tag)
		}
		pair := e.AsTuple()
		if len(pair) != 2 {
			return bytecode.Value{}, fmt.Errorf("map_from_list: element %d is not a 2-tuple (has %d elements)", i, len(pair))
		}
		k, err := bytecode.MapKey(pair[0])
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("map_from_list: element %d: %w", i, err)
		}
		entries[k] = bytecode.MapEntry{Key: pair[0], Value: pair[1]}
	}
	return bytecode.NewMap(entries), nil
}

func mapProjection(name string, args []bytecode.Value, project func(bytecode.MapEntry) bytecode.Value) (bytecode.Value, error) {
	if err := arity(name, args, 1); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg(name, args)
	if err != nil {
		return bytecode.Value{}, err
	}
	values := make([]bytecode.Value, 0, len(m.Entries))
	for _, k := range m.SortedKeys() {
		values = append(values, project(m.Entries[k]))
	}
	return bytecode.NewList(values), nil
}

func builtinMapKeys(args []bytecode.Value) (bytecode.Value, error) {
	return mapProjection("map_keys", args, func(e bytecode.MapEntry) bytecode.Value { return e.Key })
}
func builtinMapValues(args []bytecode.Value) (bytecode.Value, error) {
	return mapProjection("map_values", args, func(e bytecode.MapEntry) bytecode.Value { return e.Value })
}
func builtinMapToList(args []bytecode.Value) (bytecode.Value, error) {
	return mapProjection("map_to_list", args, func(e bytecode.MapEntry) bytecode.Value { return bytecode.NewTuple([]bytecode.Value{e.Key, e.Value}) })
}

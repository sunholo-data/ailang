package vm

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/bytecode"
)

func mapArg(name string, args []bytecode.Value) (*bytecode.MapObj, error) {
	if args[0].Tag != bytecode.TagMap {
		return nil, fmt.Errorf("%s: expected Map, got %s", name, args[0].Tag)
	}
	return args[0].AsMap(), nil
}

func builtinMapEmpty(_ []bytecode.Value) (bytecode.Value, error) { return bytecode.NewMap(nil), nil }

func builtinMapInsert(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_insert", args, 3); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg("map_insert", args)
	if err != nil {
		return bytecode.Value{}, err
	}
	v, err := m.Insert(args[1], args[2])
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("map_insert: %w", err)
	}
	return v, nil
}

func builtinMapLookup(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_lookup", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg("map_lookup", args)
	if err != nil {
		return bytecode.Value{}, err
	}
	v, ok, err := m.Lookup(args[1])
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("map_lookup: %w", err)
	}
	if ok {
		return bytecode.NewADT(0, "Some", []bytecode.Value{v}), nil
	}
	return bytecode.NewADT(1, "None", nil), nil
}

func builtinMapMember(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_member", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg("map_member", args)
	if err != nil {
		return bytecode.Value{}, err
	}
	_, ok, err := m.Lookup(args[1])
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("map_member: %w", err)
	}
	return bytecode.NewBool(ok), nil
}

func builtinMapRemove(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_remove", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg("map_remove", args)
	if err != nil {
		return bytecode.Value{}, err
	}
	v, err := m.Remove(args[1])
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("map_remove: %w", err)
	}
	return v, nil
}

func builtinMapSize(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("map_size", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	m, err := mapArg("map_size", args)
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewInt(int64(len(m.Entries))), nil
}

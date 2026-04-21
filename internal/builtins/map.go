package builtins

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Map builtin functions for AILANG
// These provide O(1) lookup operations on immutable hash maps.
// Keys must be int, string, or bool. Updates are copy-on-write (O(n)).

func init() {
	registerMapEmpty()
	registerMapInsert()
	registerMapLookup()
	registerMapMember()
	registerMapRemove()
	registerMapSize()
	registerMapKeys()
	registerMapValues()
	registerMapFromList()
	registerMapToList()
}

// ============================================================================
// Helper functions
// ============================================================================

func mapMakeSome(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{val},
	}
}

func mapMakeNone() eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "None",
		Fields:     []eval.Value{},
	}
}

// sortedCanonicalKeys returns the canonical keys of a map in sorted order (A1 determinism)
func sortedCanonicalKeys(m *eval.MapValue) []string {
	keys := make([]string, 0, len(m.Entries))
	for k := range m.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ============================================================================
// _map_empty: (()) -> Map[k, v]  (S-CALL0: takes unit parameter)
// ============================================================================

func registerMapEmpty() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_empty",
		NumArgs: 1, // S-CALL0: zero-arg builtins take unit parameter
		IsPure:  true,
		Effect:  "",
		Type:    makeMapEmptyType,
		Impl:    mapEmptyImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create an empty map",
			LongDesc:    "Returns an empty map with no entries. O(1).",
			Params:      []ParamDoc{},
			Returns:     "Empty map",
			Since:       "v0.11.0",
			Stability:   StabilityExperimental,
			Tags:        []string{"map", "create"},
			Category:    "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_empty: %v", err))
	}
}

func makeMapEmptyType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	return T.Func(T.Unit()).Returns(T.Map(k, v)).Build()
}

func mapEmptyImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// args[0] is unit — ignore it
	return &eval.MapValue{Entries: make(map[string]*eval.MapEntry)}, nil
}

// ============================================================================
// _map_insert: (Map[k, v], k, v) -> Map[k, v]
// ============================================================================

func registerMapInsert() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_insert",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapInsertType,
		Impl:    mapInsertImpl,
		Metadata: &BuiltinMetadata{
			Description: "Insert a key-value pair into a map",
			LongDesc:    "Returns a new map with the key-value pair added or updated. O(n) copy-on-write.",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to insert into"},
				{Name: "key", Description: "Key (int, string, or bool)"},
				{Name: "val", Description: "Value to associate with key"},
			},
			Returns:   "New map with the entry added/updated",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "insert", "update"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_insert: %v", err))
	}
}

func makeMapInsertType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV, k, v).Returns(mapKV).Build()
}

func mapInsertImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_insert: expected Map, got %T", args[0])
	}
	result, err := m.Insert(args[1], args[2])
	if err != nil {
		return nil, fmt.Errorf("map_insert: %w", err)
	}
	return result, nil
}

// ============================================================================
// _map_lookup: (Map[k, v], k) -> Option[v]
// ============================================================================

func registerMapLookup() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_lookup",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapLookupType,
		Impl:    mapLookupImpl,
		Metadata: &BuiltinMetadata{
			Description: "Look up a key in a map",
			LongDesc:    "Returns Some(value) if the key exists, None otherwise. O(1).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to search"},
				{Name: "key", Description: "Key to look up"},
			},
			Returns:   "Some(value) if found, None otherwise",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "lookup", "search"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_lookup: %v", err))
	}
}

func makeMapLookupType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	optionV := T.App("Option", v)
	return T.Func(mapKV, k).Returns(optionV).Build()
}

func mapLookupImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_lookup: expected Map, got %T", args[0])
	}
	val, found := m.Lookup(args[1])
	if found {
		return mapMakeSome(val), nil
	}
	return mapMakeNone(), nil
}

// ============================================================================
// _map_member: (Map[k, v], k) -> bool
// ============================================================================

func registerMapMember() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_member",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapMemberType,
		Impl:    mapMemberImpl,
		Metadata: &BuiltinMetadata{
			Description: "Check if a key exists in a map",
			LongDesc:    "Returns true if the key is present in the map, false otherwise. O(1).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to check"},
				{Name: "key", Description: "Key to check for"},
			},
			Returns:   "true if key exists, false otherwise",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "member", "contains"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_member: %v", err))
	}
}

func makeMapMemberType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV, k).Returns(T.Bool()).Build()
}

func mapMemberImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_member: expected Map, got %T", args[0])
	}
	_, found := m.Lookup(args[1])
	return &eval.BoolValue{Value: found}, nil
}

// ============================================================================
// _map_remove: (Map[k, v], k) -> Map[k, v]
// ============================================================================

func registerMapRemove() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_remove",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapRemoveType,
		Impl:    mapRemoveImpl,
		Metadata: &BuiltinMetadata{
			Description: "Remove a key from a map",
			LongDesc:    "Returns a new map without the given key. O(n) copy-on-write.",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to remove from"},
				{Name: "key", Description: "Key to remove"},
			},
			Returns:   "New map without the key",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "remove", "delete"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_remove: %v", err))
	}
}

func makeMapRemoveType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV, k).Returns(mapKV).Build()
}

func mapRemoveImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_remove: expected Map, got %T", args[0])
	}
	return m.Remove(args[1]), nil
}

// ============================================================================
// _map_size: (Map[k, v]) -> int
// ============================================================================

func registerMapSize() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_size",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapSizeType,
		Impl:    mapSizeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the number of entries in a map",
			LongDesc:    "Returns the number of key-value pairs in the map. O(1).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to get size of"},
			},
			Returns:   "Number of entries",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "size", "length"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_size: %v", err))
	}
}

func makeMapSizeType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV).Returns(T.Int()).Build()
}

func mapSizeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_size: expected Map, got %T", args[0])
	}
	return &eval.IntValue{Value: m.Size()}, nil
}

// ============================================================================
// _map_keys: (Map[k, v]) -> [k]
// ============================================================================

func registerMapKeys() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_keys",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapKeysType,
		Impl:    mapKeysImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get all keys from a map",
			LongDesc:    "Returns a list of all keys in the map, sorted for determinism. O(n log n).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to get keys from"},
			},
			Returns:   "Sorted list of keys",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "keys", "iterate"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_keys: %v", err))
	}
}

func makeMapKeysType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV).Returns(T.List(k)).Build()
}

func mapKeysImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_keys: expected Map, got %T", args[0])
	}
	sortedKeys := sortedCanonicalKeys(m)
	elements := make([]eval.Value, len(sortedKeys))
	for i, k := range sortedKeys {
		elements[i] = m.Entries[k].Key
	}
	return &eval.ListValue{Elements: elements}, nil
}

// ============================================================================
// _map_values: (Map[k, v]) -> [v]
// ============================================================================

func registerMapValues() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_values",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapValuesType,
		Impl:    mapValuesImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get all values from a map",
			LongDesc:    "Returns a list of all values in the map, sorted by key for determinism. O(n log n).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to get values from"},
			},
			Returns:   "List of values sorted by key",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "values", "iterate"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_values: %v", err))
	}
}

func makeMapValuesType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	return T.Func(mapKV).Returns(T.List(v)).Build()
}

func mapValuesImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_values: expected Map, got %T", args[0])
	}
	sortedKeys := sortedCanonicalKeys(m)
	elements := make([]eval.Value, len(sortedKeys))
	for i, k := range sortedKeys {
		elements[i] = m.Entries[k].Value
	}
	return &eval.ListValue{Elements: elements}, nil
}

// ============================================================================
// _map_from_list: ([(k, v)]) -> Map[k, v]
// ============================================================================

func registerMapFromList() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_from_list",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapFromListType,
		Impl:    mapFromListImpl,
		Metadata: &BuiltinMetadata{
			Description: "Build a map from a list of key-value tuples",
			LongDesc:    "Creates a map from a list of (key, value) tuples. Built mutably internally for O(n) average performance, returns immutable result.",
			Params: []ParamDoc{
				{Name: "pairs", Description: "List of (key, value) tuples"},
			},
			Returns:   "New map containing all key-value pairs",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "create", "convert"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_from_list: %v", err))
	}
}

func makeMapFromListType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	pairType := &types.TTuple{Elements: []types.Type{k, v}}
	listOfPairs := T.List(pairType)
	mapKV := T.Map(k, v)
	return T.Func(listOfPairs).Returns(mapKV).Build()
}

func mapFromListImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("map_from_list: expected List, got %T", args[0])
	}
	// Build mutably inside this single call, then return immutable
	entries := make(map[string]*eval.MapEntry, len(list.Elements))
	for i, elem := range list.Elements {
		tuple, ok := elem.(*eval.TupleValue)
		if !ok {
			return nil, fmt.Errorf("map_from_list: element %d is not a tuple, got %T", i, elem)
		}
		if len(tuple.Elements) != 2 {
			return nil, fmt.Errorf("map_from_list: element %d is not a 2-tuple (has %d elements)", i, len(tuple.Elements))
		}
		canonKey, err := eval.MapKey(tuple.Elements[0])
		if err != nil {
			return nil, fmt.Errorf("map_from_list: element %d: %w", i, err)
		}
		entries[canonKey] = &eval.MapEntry{
			Key:   tuple.Elements[0],
			Value: tuple.Elements[1],
		}
	}
	return &eval.MapValue{Entries: entries}, nil
}

// ============================================================================
// _map_to_list: (Map[k, v]) -> [(k, v)]
// ============================================================================

func registerMapToList() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/map",
		Name:    "_map_to_list",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeMapToListType,
		Impl:    mapToListImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert a map to a list of key-value tuples",
			LongDesc:    "Returns a list of (key, value) tuples, sorted by key for determinism. O(n log n).",
			Params: []ParamDoc{
				{Name: "m", Description: "Map to convert"},
			},
			Returns:   "Sorted list of (key, value) tuples",
			Since:     "v0.11.0",
			Stability: StabilityExperimental,
			Tags:      []string{"map", "convert", "list"},
			Category:  "map",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register map_to_list: %v", err))
	}
}

func makeMapToListType() types.Type {
	T := types.NewBuilder()
	k := T.Var("k")
	v := T.Var("v")
	mapKV := T.Map(k, v)
	pairType := &types.TTuple{Elements: []types.Type{k, v}}
	listOfPairs := T.List(pairType)
	return T.Func(mapKV).Returns(listOfPairs).Build()
}

func mapToListImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	m, ok := args[0].(*eval.MapValue)
	if !ok {
		return nil, fmt.Errorf("map_to_list: expected Map, got %T", args[0])
	}
	sortedKeys := sortedCanonicalKeys(m)
	elements := make([]eval.Value, len(sortedKeys))
	for i, k := range sortedKeys {
		entry := m.Entries[k]
		elements[i] = &eval.TupleValue{Elements: []eval.Value{entry.Key, entry.Value}}
	}
	return &eval.ListValue{Elements: elements}, nil
}

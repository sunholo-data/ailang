package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-DOCPARSE-DX M2: Set Operations on lists
// member, dedup, intersect, union, difference

func init() {
	registerListMember()
	registerListDedup()
	registerListIntersect()
	registerListUnion()
	registerListDifference()
}

// registerListMember registers _list_member: check if element is in list
func registerListMember() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_member",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListMemberType,
		Impl:    listMemberImpl,
		Metadata: &BuiltinMetadata{
			Description: "Check if an element is a member of a list",
			Params: []ParamDoc{
				{Name: "elem", Description: "Element to search for"},
				{Name: "xs", Description: "List to search in"},
			},
			Returns:   "true if element is found, false otherwise",
			Examples:  []Example{{Code: `member(3, [1,2,3])`, Description: "Returns true"}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "set", "member", "contains"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_member: %v", err))
	}
}

func makeListMemberType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	return T.Func(a, T.List(a)).Returns(T.Bool()).Build()
}

func listMemberImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	elem := args[0]
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_member: expected List, got %T", args[1])
	}
	for _, v := range list.Elements {
		if valuesEqual(elem, v) {
			return &eval.BoolValue{Value: true}, nil
		}
	}
	return &eval.BoolValue{Value: false}, nil
}

// registerListDedup registers _list_dedup: remove duplicates preserving first occurrence
func registerListDedup() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_dedup",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeListDedupType,
		Impl:    listDedupImpl,
		Metadata: &BuiltinMetadata{
			Description: "Remove duplicate elements, preserving first occurrence order",
			Params: []ParamDoc{
				{Name: "xs", Description: "List to deduplicate"},
			},
			Returns:   "List with duplicates removed",
			Examples:  []Example{{Code: `dedup([1,2,1,3,2])`, Description: "Returns [1,2,3]"}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "set", "dedup", "unique"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_dedup: %v", err))
	}
}

func makeListDedupType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	return T.Func(T.List(a)).Returns(T.List(a)).Build()
}

func listDedupImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_dedup: expected List, got %T", args[0])
	}
	// M-HASH-COLLECTIONS Phase 1: O(n) via canonicalKey + Go map
	seen := make(map[string]bool, len(list.Elements))
	result := make([]eval.Value, 0, len(list.Elements))
	for _, v := range list.Elements {
		key := canonicalKey(v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return &eval.ListValue{Elements: result}, nil
}

// registerListIntersect registers _list_intersect: elements in both lists
func registerListIntersect() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_intersect",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListSetOpType,
		Impl:    listIntersectImpl,
		Metadata: &BuiltinMetadata{
			Description: "Return elements present in both lists (set intersection)",
			Params: []ParamDoc{
				{Name: "xs", Description: "First list"},
				{Name: "ys", Description: "Second list"},
			},
			Returns:   "List of elements in both xs and ys",
			Examples:  []Example{{Code: `intersect([1,2,3], [2,3,4])`, Description: "Returns [2,3]"}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "set", "intersect"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_intersect: %v", err))
	}
}

// makeListSetOpType is shared by intersect, union, difference: [a] -> [a] -> [a]
func makeListSetOpType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	return T.Func(T.List(a), T.List(a)).Returns(T.List(a)).Build()
}

func listIntersectImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list1, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_intersect: expected List, got %T", args[0])
	}
	list2, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_intersect: expected List, got %T", args[1])
	}
	// M-HASH-COLLECTIONS Phase 1: O(n+m) via canonicalKey + Go map
	set2 := make(map[string]bool, len(list2.Elements))
	for _, v := range list2.Elements {
		set2[canonicalKey(v)] = true
	}
	seen := make(map[string]bool)
	result := make([]eval.Value, 0)
	for _, v := range list1.Elements {
		key := canonicalKey(v)
		if set2[key] && !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return &eval.ListValue{Elements: result}, nil
}

// registerListUnion registers _list_union: elements in either list (no duplicates)
func registerListUnion() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_union",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListSetOpType,
		Impl:    listUnionImpl,
		Metadata: &BuiltinMetadata{
			Description: "Return union of two lists (no duplicates)",
			Params: []ParamDoc{
				{Name: "xs", Description: "First list"},
				{Name: "ys", Description: "Second list"},
			},
			Returns:   "List of unique elements from both lists",
			Examples:  []Example{{Code: `union([1,2], [2,3])`, Description: "Returns [1,2,3]"}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "set", "union"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_union: %v", err))
	}
}

func listUnionImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list1, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_union: expected List, got %T", args[0])
	}
	list2, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_union: expected List, got %T", args[1])
	}
	// M-HASH-COLLECTIONS Phase 1: O(n+m) via canonicalKey + Go map
	seen := make(map[string]bool, len(list1.Elements)+len(list2.Elements))
	result := make([]eval.Value, 0, len(list1.Elements)+len(list2.Elements))
	for _, v := range list1.Elements {
		key := canonicalKey(v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	for _, v := range list2.Elements {
		key := canonicalKey(v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return &eval.ListValue{Elements: result}, nil
}

// registerListDifference registers _list_difference: elements in first but not second
func registerListDifference() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_difference",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListSetOpType,
		Impl:    listDifferenceImpl,
		Metadata: &BuiltinMetadata{
			Description: "Return elements in first list but not in second (set difference)",
			Params: []ParamDoc{
				{Name: "xs", Description: "List to subtract from"},
				{Name: "ys", Description: "Elements to remove"},
			},
			Returns:   "Elements in xs that are not in ys",
			Examples:  []Example{{Code: `difference([1,2,3], [2])`, Description: "Returns [1,3]"}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "set", "difference", "subtract"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_difference: %v", err))
	}
}

func listDifferenceImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list1, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_difference: expected List, got %T", args[0])
	}
	list2, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_difference: expected List, got %T", args[1])
	}
	// M-HASH-COLLECTIONS Phase 1: O(n+m) via canonicalKey + Go map
	set2 := make(map[string]bool, len(list2.Elements))
	for _, v := range list2.Elements {
		set2[canonicalKey(v)] = true
	}
	result := make([]eval.Value, 0, len(list1.Elements))
	for _, v := range list1.Elements {
		if !set2[canonicalKey(v)] {
			result = append(result, v)
		}
	}
	return &eval.ListValue{Elements: result}, nil
}

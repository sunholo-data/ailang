package vm

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
)

// M-BYTECODE-STDLIB-BUILTINS M3: Pure list builtins wired to VM OpBuiltinCall.
//
// HOF builtins (map, filter, foldl) stay EvalOnly because the VM's BuiltinFunc
// signature cannot call back into VM closures. Only pure-data list operations
// are wired here. Map and bytes builtins are also deferred — the VM has no
// TagMap or TagBytes value type.

func builtinListNth(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_nth: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_nth: arg 0 must be list")
	}
	if args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__list_nth: arg 1 must be int")
	}
	elems := args[0].AsList()
	i := int(args[1].Int)
	if i < 0 || i >= len(elems) {
		return bytecode.Value{}, fmt.Errorf("__list_nth: index %d out of range [0,%d)", i, len(elems))
	}
	return elems[i], nil
}

func builtinListMember(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_member: expected 2 args, got %d", len(args))
	}
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_member: arg 1 must be list")
	}
	elem := args[0]
	for _, v := range args[1].AsList() {
		if elem.Equal(v) {
			return bytecode.NewBool(true), nil
		}
	}
	return bytecode.NewBool(false), nil
}

func builtinListDedup(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__list_dedup: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_dedup: expected list")
	}
	elems := args[0].AsList()
	result := make([]bytecode.Value, 0, len(elems))
	for _, e := range elems {
		found := false
		for _, r := range result {
			if e.Equal(r) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, e)
		}
	}
	return bytecode.NewList(result), nil
}

func builtinListDifference(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_difference: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList || args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_difference: expected lists")
	}
	xs := args[0].AsList()
	ys := args[1].AsList()
	result := make([]bytecode.Value, 0, len(xs))
	for _, x := range xs {
		found := false
		for _, y := range ys {
			if x.Equal(y) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, x)
		}
	}
	return bytecode.NewList(result), nil
}

func builtinListIntersect(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_intersect: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList || args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_intersect: expected lists")
	}
	xs := args[0].AsList()
	ys := args[1].AsList()
	result := make([]bytecode.Value, 0, len(xs))
	for _, x := range xs {
		for _, y := range ys {
			if x.Equal(y) {
				result = append(result, x)
				break
			}
		}
	}
	return bytecode.NewList(result), nil
}

func builtinListUnion(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_union: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList || args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_union: expected lists")
	}
	xs := args[0].AsList()
	ys := args[1].AsList()
	// Start with all of xs, then add unique elements from ys
	result := make([]bytecode.Value, len(xs), len(xs)+len(ys))
	copy(result, xs)
	for _, y := range ys {
		found := false
		for _, x := range result {
			if y.Equal(x) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, y)
		}
	}
	return bytecode.NewList(result), nil
}

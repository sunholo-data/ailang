package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// Native VM ports of the iterative std/list search/fold builtins (#1518,
// #1501). They mirror internal/builtins/list_iterative_search.go exactly;
// list_hof_parity_test.go pins the two engines together.

// listPredicateIndexVM returns the index of the first element of args[1] for
// which the closure args[0] returns true, or -1. Shared by any/findIndex.
func listPredicateIndexVM(name string, caller ClosureCaller, args []bytecode.Value) (int, error) {
	var argv callArgs // one buffer for every callback (#1501)
	if len(args) != 2 {
		return 0, fmt.Errorf("%s: expected 2 args, got %d", name, len(args))
	}
	if args[1].Tag != bytecode.TagList {
		return 0, fmt.Errorf("%s: arg 1 must be list, got %s", name, args[1].Tag)
	}
	for i, e := range args[1].AsList() {
		v, err := caller.CallClosure(args[0], argv.of1(e))
		if err != nil {
			return 0, fmt.Errorf("%s: callback error at index %d: %w", name, i, err)
		}
		if v.Tag != bytecode.TagBool {
			return 0, fmt.Errorf("%s: predicate must return bool, got %s at index %d", name, v.Tag, i)
		}
		if v.Bool {
			return i, nil
		}
	}
	return -1, nil
}

// hofBuiltinListAny implements __list_any: (a -> bool, [a]) -> bool.
func hofBuiltinListAny(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	i, err := listPredicateIndexVM("__list_any", caller, args)
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewBool(i >= 0), nil
}

// hofBuiltinListFindIndex implements __list_findIndex: (a -> bool, [a]) -> Option[int].
func hofBuiltinListFindIndex(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	i, err := listPredicateIndexVM("__list_findIndex", caller, args)
	if err != nil {
		return bytecode.Value{}, err
	}
	if i < 0 {
		return bytecode.NewADT(optionTagNone, "None", nil), nil
	}
	return bytecode.NewADT(optionTagSome, "Some", []bytecode.Value{bytecode.NewInt(int64(i))}), nil
}

// hofBuiltinListFoldr implements __list_foldr: ((a, b) -> b, b, [a]) -> b,
// combining right-to-left.
func hofBuiltinListFoldr(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	var argv callArgs // one buffer for every callback (#1501)
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__list_foldr: expected 3 args, got %d", len(args))
	}
	fn, acc := args[0], args[1]
	if args[2].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_foldr: arg 2 must be list, got %s", args[2].Tag)
	}
	elems := args[2].AsList()
	var err error
	for i := len(elems) - 1; i >= 0; i-- {
		acc, err = caller.CallClosure(fn, argv.of2(elems[i], acc))
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__list_foldr: callback error at index %d: %w", i, err)
		}
	}
	return acc, nil
}

// hofBuiltinListMapAccumL implements
// __list_mapAccumL: ((s, a) -> (b, s), s, [a]) -> ([b], s).
func hofBuiltinListMapAccumL(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	var argv callArgs // one buffer for every callback (#1501)
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__list_mapAccumL: expected 3 args, got %d", len(args))
	}
	fn, state := args[0], args[1]
	if args[2].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_mapAccumL: arg 2 must be list, got %s", args[2].Tag)
	}
	elems := args[2].AsList()
	out := make([]bytecode.Value, len(elems))
	for i, e := range elems {
		v, err := caller.CallClosure(fn, argv.of2(state, e))
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__list_mapAccumL: callback error at index %d: %w", i, err)
		}
		if v.Tag != bytecode.TagTuple || len(v.AsTuple()) != 2 {
			return bytecode.Value{}, fmt.Errorf("__list_mapAccumL: step must return a pair (output, state), got %s at index %d", v.Tag, i)
		}
		pair := v.AsTuple()
		out[i], state = pair[0], pair[1]
	}
	return bytecode.NewTuple([]bytecode.Value{bytecode.NewList(out), state}), nil
}

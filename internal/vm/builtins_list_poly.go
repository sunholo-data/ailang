package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// Native polymorphic list builtins (M-VM-PURE-BUILTIN-COVERAGE M3, #1447).
// These cannot go through the generic adapter: their elements may be ADTs or
// closures, which the VM↔evaluator converter cannot name. They only move
// Values around, so they run natively. Semantics mirror the evaluator Impls in
// internal/builtins (list.go, list_iterative_more.go) exactly; the parity test
// in builtins_list_poly_test.go holds them together.

func listArg(name string, args []bytecode.Value, i int) ([]bytecode.Value, error) {
	if args[i].Tag != bytecode.TagList {
		return nil, fmt.Errorf("%s: arg %d must be list, got %s", name, i, args[i].Tag)
	}
	return args[i].AsList(), nil
}

func intArg(name string, args []bytecode.Value, i int) (int, error) {
	if args[i].Tag != bytecode.TagInt {
		return 0, fmt.Errorf("%s: arg %d must be int, got %s", name, i, args[i].Tag)
	}
	return int(args[i].Int), nil
}

func arity(name string, args []bytecode.Value, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s: expected %d args, got %d", name, n, len(args))
	}
	return nil
}

// builtinListReverse implements __list_reverse: [a] -> [a].
func builtinListReverse(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_reverse", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_reverse", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	out := make([]bytecode.Value, len(xs))
	for i, x := range xs {
		out[len(xs)-1-i] = x
	}
	return bytecode.NewList(out), nil
}

// builtinListTake implements __list_take: (int, [a]) -> [a]. n<=0 gives [],
// n past the end gives the whole list.
func builtinListTake(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_take", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	n, err := intArg("__list_take", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_take", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	n = max(0, min(n, len(xs)))
	return bytecode.NewList(append([]bytecode.Value(nil), xs[:n]...)), nil
}

// builtinListDrop implements __list_drop: (int, [a]) -> [a]. n<=0 gives the
// whole list, n past the end gives [].
func builtinListDrop(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_drop", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	n, err := intArg("__list_drop", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_drop", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	n = max(0, min(n, len(xs)))
	return bytecode.NewList(append([]bytecode.Value{}, xs[n:]...)), nil
}

// builtinListZip implements __list_zip: ([a], [b]) -> [(a, b)], truncating to
// the shorter list.
func builtinListZip(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_zip", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_zip", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	ys, err := listArg("__list_zip", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	n := min(len(xs), len(ys))
	out := make([]bytecode.Value, n)
	for i := range n {
		out[i] = bytecode.NewTuple([]bytecode.Value{xs[i], ys[i]})
	}
	return bytecode.NewList(out), nil
}

// builtinListContains implements __list_contains: ([a], a) -> bool, using the
// VM's structural equality (the same runtimeEq as ==).
func builtinListContains(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_contains", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_contains", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	for _, x := range xs {
		if runtimeEq(x, args[1]) {
			return bytecode.NewBool(true), nil
		}
	}
	return bytecode.NewBool(false), nil
}

// builtinListHead implements __list_head: [a] -> a; an empty list is an error.
func builtinListHead(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_head", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_head", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	if len(xs) == 0 {
		return bytecode.Value{}, fmt.Errorf("_list_head: empty list")
	}
	return xs[0], nil
}

// builtinListExtract implements __list_extract: ([a], offset, length) -> [a],
// clamped like Z3's seq.extract: a negative offset reads from 0, a negative
// length or an offset at/after the end gives [].
func builtinListExtract(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("__list_extract", args, 3); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("__list_extract", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	offset, err := intArg("__list_extract", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	length, err := intArg("__list_extract", args, 2)
	if err != nil {
		return bytecode.Value{}, err
	}
	offset = max(0, min(offset, len(xs)))
	if length < 0 || offset >= len(xs) {
		return bytecode.NewList([]bytecode.Value{}), nil
	}
	end := min(offset+length, len(xs))
	return bytecode.NewList(append([]bytecode.Value(nil), xs[offset:end]...)), nil
}

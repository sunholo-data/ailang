package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// Native polymorphic std/array builtins. Like the list ports in
// builtins_list_poly.go these cannot use the generic adapter (elements may be
// ADTs or closures). Semantics, including which results use the packed float
// store, mirror internal/builtins/array.go and array_float.go; the parity test
// in builtins_array_poly_test.go holds them together.

// packedOrBoxed mirrors eval.NewArray: a non-empty all-float slice is packed.
func packedOrBoxed(elems []bytecode.Value) bytecode.Value {
	if len(elems) == 0 {
		return bytecode.NewArray(elems)
	}
	fs := make([]float64, len(elems))
	for i, e := range elems {
		if e.Tag != bytecode.TagFloat {
			return bytecode.NewArray(elems)
		}
		fs[i] = e.Flt
	}
	return bytecode.NewFloatArray(fs)
}

func arrayArg(name string, args []bytecode.Value, i int) (*bytecode.ArrayObj, error) {
	if args[i].Tag != bytecode.TagArray {
		return nil, fmt.Errorf("%s: expected Array, got %s", name, args[i].Tag)
	}
	return args[i].AsArray(), nil
}

// boxedElems returns the elements as a fresh slice the caller may modify.
func boxedElems(a *bytecode.ArrayObj) []bytecode.Value {
	out := make([]bytecode.Value, a.Len())
	for i := range out {
		out[i] = a.At(i)
	}
	return out
}

// builtinArrayEmpty implements __array_empty: () -> Array[a].
func builtinArrayEmpty(_ []bytecode.Value) (bytecode.Value, error) {
	return bytecode.NewArray([]bytecode.Value{}), nil
}

// builtinArrayMake implements __array_make: (int, a) -> Array[a].
func builtinArrayMake(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_make", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	n, err := intArg("array_make", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	if n < 0 {
		return bytecode.Value{}, fmt.Errorf("array_make: size cannot be negative, got %d", n)
	}
	if args[1].Tag == bytecode.TagFloat {
		fs := make([]float64, n)
		for i := range fs {
			fs[i] = args[1].Flt
		}
		if n == 0 {
			return bytecode.NewArray([]bytecode.Value{}), nil
		}
		return bytecode.NewFloatArray(fs), nil
	}
	elems := make([]bytecode.Value, n)
	for i := range elems {
		elems[i] = args[1]
	}
	return bytecode.NewArray(elems), nil
}

func arrayIndex(name string, args []bytecode.Value) (bytecode.Value, error) {
	if err := arity(name, args, 2); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg(name, args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	i, err := intArg(name, args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	if i < 0 || i >= a.Len() {
		return bytecode.Value{}, fmt.Errorf("%s: index %d out of bounds (array length: %d)", name, i, a.Len())
	}
	return a.At(i), nil
}

// builtinArrayGet implements __array_get: (Array[a], int) -> a.
func builtinArrayGet(args []bytecode.Value) (bytecode.Value, error) {
	return arrayIndex("array_get", args)
}

// builtinArrayUnsafeGet implements __array_unsafe_get (same contract as get).
func builtinArrayUnsafeGet(args []bytecode.Value) (bytecode.Value, error) {
	return arrayIndex("array_unsafe_get", args)
}

// builtinArrayLength implements __array_length: Array[a] -> int.
func builtinArrayLength(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_length", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg("array_length", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewInt(int64(a.Len())), nil
}

// builtinArraySet implements __array_set: (Array[a], int, a) -> Array[a],
// copy-on-write; out of bounds is an error.
func builtinArraySet(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_set", args, 3); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg("array_set", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	i, err := intArg("array_set", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	if i < 0 || i >= a.Len() {
		return bytecode.Value{}, fmt.Errorf("array_set: index %d out of bounds (array length: %d)", i, a.Len())
	}
	if a.Floats != nil && args[2].Tag == bytecode.TagFloat {
		fs := append([]float64(nil), a.Floats...)
		fs[i] = args[2].Flt
		return bytecode.NewFloatArray(fs), nil
	}
	elems := boxedElems(a)
	elems[i] = args[2]
	return packedOrBoxed(elems), nil
}

// builtinArrayFromList implements __array_from_list: [a] -> Array[a].
func builtinArrayFromList(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_from_list", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	xs, err := listArg("array_from_list", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	return packedOrBoxed(append([]bytecode.Value{}, xs...)), nil
}

// builtinArrayToList implements __array_to_list: Array[a] -> [a].
func builtinArrayToList(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_to_list", args, 1); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg("array_to_list", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewList(boxedElems(a)), nil
}

// builtinArrayAppend implements __array_append: (Array[a], a) -> Array[a].
func builtinArrayAppend(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("array_append", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg("array_append", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	if a.Floats != nil && args[1].Tag == bytecode.TagFloat {
		fs := append(append(make([]float64, 0, len(a.Floats)+1), a.Floats...), args[1].Flt)
		return bytecode.NewFloatArray(fs), nil
	}
	return packedOrBoxed(append(boxedElems(a), args[1])), nil
}

// builtinArrayUpdateMany implements __array_update_many:
// (Array[a], [(int, a)]) -> Array[a]; later writes win, out of bounds is an
// error, one copy.
func builtinArrayUpdateMany(args []bytecode.Value) (bytecode.Value, error) {
	if err := arity("updateMany", args, 2); err != nil {
		return bytecode.Value{}, err
	}
	a, err := arrayArg("updateMany", args, 0)
	if err != nil {
		return bytecode.Value{}, err
	}
	writes, err := listArg("updateMany", args, 1)
	if err != nil {
		return bytecode.Value{}, err
	}
	out := boxedElems(a)
	for _, w := range writes {
		if w.Tag != bytecode.TagTuple || len(w.AsTuple()) != 2 || w.AsTuple()[0].Tag != bytecode.TagInt {
			return bytecode.Value{}, fmt.Errorf("updateMany: each write must be an (int, value) pair, got %s", w.Tag)
		}
		i := int(w.AsTuple()[0].Int)
		if i < 0 || i >= len(out) {
			return bytecode.Value{}, fmt.Errorf("updateMany: index %d out of bounds (array length: %d)", i, len(out))
		}
		out[i] = w.AsTuple()[1]
	}
	return packedOrBoxed(out), nil
}

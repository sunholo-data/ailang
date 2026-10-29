package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// Suffix views share storage (#1827, M-VM-LIST-TAIL-SHARE). _list_tail used to
// copy the suffix, so every `x :: rest` walk was O(n^2) on the VM.

func bigInts(n int) bytecode.Value {
	vs := make([]bytecode.Value, n)
	for i := range vs {
		vs[i] = bytecode.NewInt(int64(i))
	}
	return bytecode.NewList(vs)
}

func TestListTailDoesNotCopy(t *testing.T) {
	// Allocation counts cannot see this bug (a copy is one make), so check
	// identity: the result must point into the source's backing array.
	xs := bigInts(10000)
	src := xs.AsList()
	cases := []struct {
		name  string
		fn    BuiltinFunc
		args  []bytecode.Value
		first *bytecode.Value
	}{
		{"_list_tail", builtinListTail, []bytecode.Value{xs, i64(1)}, &src[1]},
		{"__list_drop", builtinListDrop, []bytecode.Value{i64(1), xs}, &src[1]},
		{"__list_take", builtinListTake, []bytecode.Value{i64(9999), xs}, &src[0]},
	}
	for _, c := range cases {
		v, err := c.fn(c.args)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := v.AsList(); &got[0] != c.first {
			t.Errorf("%s copied the list; want a view sharing the source's storage", c.name)
		}
	}
}

func TestListTailClampsAndCapsCapacity(t *testing.T) {
	xs := ints(1, 2, 3)
	for _, tc := range []struct {
		n    int64
		want []int64
	}{
		{-1, []int64{1, 2, 3}}, {0, []int64{1, 2, 3}}, {1, []int64{2, 3}},
		{3, nil}, {4, nil},
	} {
		v, err := builtinListTail([]bytecode.Value{xs, i64(tc.n)})
		if err != nil {
			t.Fatalf("_list_tail(xs, %d): %v", tc.n, err)
		}
		got := v.AsList()
		if len(got) != len(tc.want) {
			t.Fatalf("_list_tail(xs, %d) = %d elems, want %d", tc.n, len(got), len(tc.want))
		}
		for i, w := range tc.want {
			if got[i].Int != w {
				t.Errorf("_list_tail(xs, %d)[%d] = %d, want %d", tc.n, i, got[i].Int, w)
			}
		}
		// cap == len: an append on the view reallocates instead of writing into xs.
		if cap(got) != len(got) {
			t.Errorf("_list_tail(xs, %d): cap %d > len %d, appends would alias the source", tc.n, cap(got), len(got))
		}
	}
	take, _ := builtinListTake([]bytecode.Value{i64(1), ints(1, 2, 3)})
	if l := take.AsList(); cap(l) != len(l) {
		t.Errorf("__list_take: cap %d > len %d, appends would overwrite the source", cap(l), len(l))
	}
}

// A sharing view must never let a later write leak into the source list.
func TestListTailViewIsolatedFromAppend(t *testing.T) {
	src := ints(1, 2, 3, 4)
	head, _ := builtinListTake([]bytecode.Value{i64(2), src})
	grown := append(head.AsList(), bytecode.NewInt(99))
	_ = grown
	if got := src.AsList()[2].Int; got != 3 {
		t.Fatalf("append on take(2, xs) overwrote xs[2]: got %d, want 3", got)
	}
}

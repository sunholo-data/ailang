package vm

import (
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// #1814: _list_tail binds `rest` in `x :: rest` patterns. It returns a view of
// the parent's backing array instead of a copy, so list recursion is linear.

func TestListTailClamps(t *testing.T) {
	xs := ints(1, 2, 3)
	cases := []struct {
		n    int64
		want bytecode.Value
	}{
		{-1, ints(1, 2, 3)},
		{0, ints(1, 2, 3)},
		{1, ints(2, 3)},
		{3, ints()},
		{4, ints()},
	}
	for _, c := range cases {
		got, err := builtinListTail([]bytecode.Value{xs, i64(c.n)})
		if err != nil {
			t.Fatalf("n=%d: %v", c.n, err)
		}
		if !got.Equal(c.want) {
			t.Errorf("n=%d: got %v, want %v", c.n, got.AsList(), c.want.AsList())
		}
	}
}

func TestListTailSharesParent(t *testing.T) {
	parent := ints(10, 20, 30, 40)
	got, err := builtinListTail([]bytecode.Value{parent, i64(1)})
	if err != nil {
		t.Fatal(err)
	}
	tail, elems := got.AsList(), parent.AsList()
	if &tail[0] != &elems[1] {
		t.Fatal("tail does not share the parent's backing array")
	}
	if cap(tail) != len(tail) {
		t.Fatalf("tail cap %d != len %d: an append could write into the parent", cap(tail), len(tail))
	}
}

// The view must stay safe when the parent has spare capacity: appending to
// the tail reallocates instead of writing into the parent's array.
func TestListTailAppendDoesNotWriteParent(t *testing.T) {
	backing := make([]bytecode.Value, 3, 8)
	for i := range backing {
		backing[i] = i64(int64(i + 1))
	}
	parent := bytecode.NewList(backing)
	got, err := builtinListTail([]bytecode.Value{parent, i64(1)})
	if err != nil {
		t.Fatal(err)
	}
	grown := append(got.AsList(), i64(99))
	if &grown[0] == &backing[1] {
		t.Fatal("append on the tail reused the parent's array")
	}
	if spare := backing[:4][3]; spare.Tag != 0 || spare.Int != 0 {
		t.Fatalf("parent's spare capacity was written: %v", spare)
	}
}

// Taking a tail of a large list allocates O(1), not O(n). Before #1814 this
// copied the whole remainder (~1.5 MB per call at n = 64k).
func TestListTailAllocationIsConstant(t *testing.T) {
	const n = 1 << 16
	vs := make([]bytecode.Value, n)
	for i := range vs {
		vs[i] = i64(int64(i))
	}
	parent := bytecode.NewList(vs)
	args := []bytecode.Value{parent, i64(1)}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for range 100 {
		if _, err := builtinListTail(args); err != nil {
			t.Fatal(err)
		}
	}
	runtime.ReadMemStats(&after)
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 64<<10 {
		t.Fatalf("100 tails of a %d-element list allocated %d bytes, want O(1) per call", n, delta)
	}
}

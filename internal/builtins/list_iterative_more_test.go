package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func intCmp() eval.Value {
	return goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.IntValue{Value: args[0].(*eval.IntValue).Value - args[1].(*eval.IntValue).Value}, nil
	})
}

func TestListSortBySorts(t *testing.T) {
	in := intList(3, 1, 2, 5, 4)
	got, err := listSortByImpl(newTestEffCtx(), []eval.Value{intCmp(), in})
	if err != nil {
		t.Fatal(err)
	}
	if g := toInts(t, got); !equalInts(g, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("got %v", g)
	}
	// The input list is not mutated.
	if g := toInts(t, in); !equalInts(g, []int{3, 1, 2, 5, 4}) {
		t.Fatalf("input mutated: %v", g)
	}
}

func TestListSortByComparatorErrors(t *testing.T) {
	boom := goFn(func([]eval.Value) (eval.Value, error) { return nil, errString("boom") })
	if _, err := listSortByImpl(newTestEffCtx(), []eval.Value{boom, intList(2, 1)}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("comparator error not propagated: %v", err)
	}
	notInt := goFn(func([]eval.Value) (eval.Value, error) { return &eval.BoolValue{Value: true}, nil })
	if _, err := listSortByImpl(newTestEffCtx(), []eval.Value{notInt, intList(2, 1)}); err == nil || !strings.Contains(err.Error(), "must return int") {
		t.Fatalf("non-int comparator result not rejected: %v", err)
	}
}

func TestListZipTruncatesToShorter(t *testing.T) {
	got, err := listZipImpl(nil, []eval.Value{intList(1, 2, 3), intList(10, 20)})
	if err != nil {
		t.Fatal(err)
	}
	l := got.(*eval.ListValue)
	if len(l.Elements) != 2 {
		t.Fatalf("len = %d, want 2", len(l.Elements))
	}
	pair := l.Elements[1].(*eval.TupleValue)
	if pair.Elements[0].(*eval.IntValue).Value != 2 || pair.Elements[1].(*eval.IntValue).Value != 20 {
		t.Fatalf("second pair = %v", pair)
	}
}

func TestListFlatMapConcatenatesInOrder(t *testing.T) {
	dup := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.ListValue{Elements: []eval.Value{args[0], args[0]}}, nil
	})
	got, err := listFlatMapImpl(newTestEffCtx(), []eval.Value{dup, intList(1, 2)})
	if err != nil {
		t.Fatal(err)
	}
	if g := toInts(t, got); !equalInts(g, []int{1, 1, 2, 2}) {
		t.Fatalf("got %v", g)
	}
	notList := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })
	if _, err := listFlatMapImpl(newTestEffCtx(), []eval.Value{notList, intList(1)}); err == nil {
		t.Fatal("non-list callback result not rejected")
	}
}

func TestStrRepeat(t *testing.T) {
	for _, c := range []struct {
		s    string
		n    int
		want string
	}{{"ab", 3, "ababab"}, {"x", 0, ""}, {"x", -2, ""}, {"", 5, ""}} {
		got, err := strRepeatImpl(nil, []eval.Value{&eval.StringValue{Value: c.s}, &eval.IntValue{Value: c.n}})
		if err != nil {
			t.Fatal(err)
		}
		if got.(*eval.StringValue).Value != c.want {
			t.Fatalf("repeat(%q, %d) = %q", c.s, c.n, got.(*eval.StringValue).Value)
		}
	}
	if _, err := strRepeatImpl(nil, []eval.Value{&eval.StringValue{Value: "ab"}, &eval.IntValue{Value: int(^uint(0) >> 1)}}); err == nil {
		t.Fatal("overflowing repeat not rejected")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

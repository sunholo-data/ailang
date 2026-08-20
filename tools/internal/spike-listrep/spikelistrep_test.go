package spikelistrep_test

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	spikelistrep "github.com/sunholo-data/ailang/tools/internal/spike-listrep"
)

func TestSliceListRoundTrip(t *testing.T) {
	testRoundTrip(t, spikelistrep.SliceFromSlice, spikelistrep.SliceEmpty, spikelistrep.SliceCons)
}

func TestConsListRoundTrip(t *testing.T) {
	testRoundTrip(t, spikelistrep.ConsFromSlice, spikelistrep.ConsEmpty, spikelistrep.ConsCons)
}

func testRoundTrip(
	t *testing.T,
	fromSlice func([]eval.Value) spikelistrep.List,
	empty func() spikelistrep.List,
	cons func(eval.Value, spikelistrep.List) spikelistrep.List,
) {
	t.Helper()
	one := &eval.IntValue{Value: 1}
	two := &eval.StringValue{Value: "two"}
	list := fromSlice([]eval.Value{one, two})

	if list.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", list.Len())
	}
	var iterated []eval.Value
	for element := range list.All() {
		iterated = append(iterated, element)
	}
	assertElements(t, iterated, []eval.Value{one, two})
	assertElements(t, list.ToSlice(), []eval.Value{one, two})

	head, tail, ok := list.Uncons()
	if !ok || head != one {
		t.Fatalf("Uncons() = (%v, _, %v), want (%v, _, true)", head, ok, one)
	}
	assertElements(t, tail.ToSlice(), []eval.Value{two})
	assertElements(t, list.DropPrefix(1).ToSlice(), []eval.Value{two})
	if got, ok := list.At(1); !ok || got != two {
		t.Fatalf("At(1) = (%v, %v), want (%v, true)", got, ok, two)
	}
	for _, index := range []int{-1, 2} {
		if got, ok := list.At(index); ok || got != nil {
			t.Fatalf("At(%d) = (%v, %v), want (nil, false)", index, got, ok)
		}
	}
	assertElements(t, cons(one, empty()).ToSlice(), []eval.Value{one})
}

func assertElements(t *testing.T, got, want []eval.Value) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("elements length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("element %d = %v, want %v", i, got[i], want[i])
		}
	}
}

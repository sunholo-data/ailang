package spikelistrep_test

import (
	"fmt"
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

func TestChunkListRoundTrip(t *testing.T) {
	for _, k := range []int{8, 32} {
		t.Run(fmt.Sprintf("K=%d", k), func(t *testing.T) {
			testRoundTrip(t,
				func(v []eval.Value) spikelistrep.List { return spikelistrep.ChunkFromSlice(k, v) },
				func() spikelistrep.List { return spikelistrep.ChunkEmpty(k) },
				func(v eval.Value, l spikelistrep.List) spikelistrep.List { return spikelistrep.ChunkCons(k, v, l) },
			)
		})
	}
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

// TestBranchingIndependence pins the property the ENTIRE B1 benchmark assumes and
// which no smoke test previously exercised: two prepends off ONE shared base must
// produce independent lists, and must leave the base untouched.
//
// B1 measures `m` prepends onto a single retained base. If a candidate silently
// shared mutable storage between branches, B1 would still produce a plausible
// ratio and AC-2 would still read green — the kill criterion would be measuring a
// representation that does not implement the semantics it is credited with.
//
// Kills the mutation "SliceCons appends into a shared scratch buffer instead of a
// freshly allocated one", which survives every other test in this package.
func TestBranchingIndependence(t *testing.T) {
	for _, arm := range benchmarkArms {
		t.Run("arm="+arm.name, func(t *testing.T) {
			baseElements := []eval.Value{
				&eval.IntValue{Value: 100},
				&eval.IntValue{Value: 200},
			}
			base := arm.fromSlice(baseElements)

			headA := &eval.IntValue{Value: 1}
			headB := &eval.IntValue{Value: 2}
			branchA := arm.cons(headA, base)
			branchB := arm.cons(headB, base)

			gotA, okA := branchA.At(0)
			gotB, okB := branchB.At(0)
			if !okA || !okB {
				t.Fatalf("At(0) not ok: A=%v B=%v", okA, okB)
			}
			if gotA != eval.Value(headA) {
				t.Errorf("branchA.At(0) = %v, want %v — branches share storage", gotA, headA)
			}
			if gotB != eval.Value(headB) {
				t.Errorf("branchB.At(0) = %v, want %v — branches share storage", gotB, headB)
			}
			if gotA == gotB {
				t.Errorf("branchA and branchB both see %v: the two prepends are not independent", gotA)
			}

			// The shared base must be unchanged by either prepend.
			if base.Len() != 2 {
				t.Errorf("base.Len() = %d after two prepends, want 2", base.Len())
			}
			assertElements(t, base.ToSlice(), baseElements)

			// Each branch must be base with exactly its own head in front.
			assertElements(t, branchA.ToSlice(), []eval.Value{headA, baseElements[0], baseElements[1]})
			assertElements(t, branchB.ToSlice(), []eval.Value{headB, baseElements[0], baseElements[1]})
		})
	}
}

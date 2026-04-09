package compiler

import "testing"

func TestRegAlloc_ScopeRecycling(t *testing.T) {
	alloc := newRegAlloc()
	scope := newScopeStack(alloc)

	// Pin a param at r0 (root scope, never freed).
	r0, _ := alloc.allocPinned()
	scope.bind("param", r0)

	// Push a nested scope and pin registers via bindScoped.
	scope.push()
	r1, _ := alloc.allocPinned()
	scope.bindScoped("a", r1)
	r2, _ := alloc.allocPinned()
	scope.bindScoped("b", r2)

	if r1 != 1 || r2 != 2 {
		t.Fatalf("expected r1=1, r2=2, got r1=%d, r2=%d", r1, r2)
	}

	// Pop scope — r1 and r2 should be returned to the free list.
	scope.pop()

	// allocTemp should reuse r1 then r2 (pop frees in reverse: r2 then r1,
	// LIFO free list returns r1 first, then r2).
	got1, _ := alloc.allocTemp()
	got2, _ := alloc.allocTemp()
	if got1 != r1 {
		t.Errorf("after pop, allocTemp should reuse r1=%d, got %d", r1, got1)
	}
	if got2 != r2 {
		t.Errorf("after pop, allocTemp should reuse r2=%d, got %d", r2, got2)
	}
}

func TestRegAlloc_ContigFromFreeList(t *testing.T) {
	alloc := newRegAlloc()

	// Allocate 5 registers, then free the middle 3.
	for i := 0; i < 5; i++ {
		alloc.allocPinned()
	}
	alloc.freeTemp(1)
	alloc.freeTemp(2)
	alloc.freeTemp(3)

	// allocContig(3) should find [1,2,3] in free list instead of bumping.
	base, err := alloc.allocContig(3)
	if err != nil {
		t.Fatalf("allocContig(3) failed: %v", err)
	}
	if base != 1 {
		t.Errorf("expected base=1 from free list, got %d", base)
	}

	// Next is still 5 (didn't bump).
	if alloc.next != 5 {
		t.Errorf("expected next=5 (no bump), got %d", alloc.next)
	}
}

func TestRegAlloc_ContigFallsBackToBump(t *testing.T) {
	alloc := newRegAlloc()

	// Allocate 5 registers, free non-adjacent ones.
	for i := 0; i < 5; i++ {
		alloc.allocPinned()
	}
	alloc.freeTemp(1)
	alloc.freeTemp(3) // [1, 3] are free but not contiguous for n=2

	// allocContig(2) should bump from next=5 since no adjacent pair in free list.
	base, err := alloc.allocContig(2)
	if err != nil {
		t.Fatalf("allocContig(2) failed: %v", err)
	}
	if base != 5 {
		t.Errorf("expected base=5 from bump, got %d", base)
	}
}

func TestRegAlloc_ScopeRecyclingEnablesContig(t *testing.T) {
	// Simulates the pattern in switch cases: each case pins registers in a
	// scoped binding, then the scope pops and frees them. The freed registers
	// should then be reusable by allocContig in subsequent cases.
	alloc := newRegAlloc()
	scope := newScopeStack(alloc)

	// Root: param in r0.
	r0, _ := alloc.allocPinned()
	scope.bind("param", r0)

	// Simulate 3 switch cases, each pinning 2 registers.
	for i := 0; i < 3; i++ {
		scope.push()
		ra, _ := alloc.allocPinned()
		scope.bindScoped("a", ra)
		rb, _ := alloc.allocPinned()
		scope.bindScoped("b", rb)
		scope.pop()
	}

	// Without recycling, next would be 7 (r0 + 3*2). With recycling, the
	// popped registers are freed and allocContig(3) should find them.
	base, err := alloc.allocContig(3)
	if err != nil {
		t.Fatalf("allocContig(3) failed: %v", err)
	}
	// Should reuse from free list (registers 1,2 from first case).
	if base > 4 {
		t.Errorf("expected base from free list (≤4), got %d", base)
	}
}

func TestSortUint8(t *testing.T) {
	tests := []struct {
		input    []uint8
		expected []uint8
	}{
		{[]uint8{3, 1, 2}, []uint8{1, 2, 3}},
		{[]uint8{5, 4, 3, 2, 1}, []uint8{1, 2, 3, 4, 5}},
		{[]uint8{1}, []uint8{1}},
		{[]uint8{}, []uint8{}},
	}
	for _, tt := range tests {
		s := make([]uint8, len(tt.input))
		copy(s, tt.input)
		sortUint8(s)
		for i := range s {
			if s[i] != tt.expected[i] {
				t.Errorf("sortUint8(%v) = %v, want %v", tt.input, s, tt.expected)
				break
			}
		}
	}
}

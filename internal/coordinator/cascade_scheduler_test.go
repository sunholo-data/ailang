package coordinator

import (
	"testing"

	"github.com/sunholo/ailang/internal/pkg"
)

func TestScheduleCascadeUpdate_LinearChain(t *testing.T) {
	// A → B → C (A published, B and C need updating in order)
	index := &pkg.RegistryIndex{
		Packages: []pkg.IndexEntry{
			{Name: "A", Dependencies: nil},
			{Name: "B", Dependencies: []string{"A"}},
			{Name: "C", Dependencies: []string{"B"}},
		},
	}

	order, err := ScheduleCascadeUpdate(index, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("expected 2 affected packages, got %d: %v", len(order), order)
	}
	// B must come before C
	if order[0] != "B" || order[1] != "C" {
		t.Errorf("expected [B, C], got %v", order)
	}
}

func TestScheduleCascadeUpdate_DiamondDeps(t *testing.T) {
	// A → B, A → C, B → D, C → D (diamond)
	index := &pkg.RegistryIndex{
		Packages: []pkg.IndexEntry{
			{Name: "A", Dependencies: nil},
			{Name: "B", Dependencies: []string{"A"}},
			{Name: "C", Dependencies: []string{"A"}},
			{Name: "D", Dependencies: []string{"B", "C"}},
		},
	}

	order, err := ScheduleCascadeUpdate(index, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 affected packages, got %d: %v", len(order), order)
	}
	// D must come after both B and C
	dIdx := -1
	for i, name := range order {
		if name == "D" {
			dIdx = i
		}
	}
	if dIdx != 2 {
		t.Errorf("expected D at position 2 (after B and C), got position %d in %v", dIdx, order)
	}
}

func TestScheduleCascadeUpdate_NoDependents(t *testing.T) {
	index := &pkg.RegistryIndex{
		Packages: []pkg.IndexEntry{
			{Name: "A", Dependencies: nil},
			{Name: "B", Dependencies: nil},
		},
	}

	order, err := ScheduleCascadeUpdate(index, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected 0 affected packages, got %d: %v", len(order), order)
	}
}

func TestScheduleCascadeUpdate_NonexistentPackage(t *testing.T) {
	index := &pkg.RegistryIndex{
		Packages: []pkg.IndexEntry{
			{Name: "A", Dependencies: nil},
		},
	}

	order, err := ScheduleCascadeUpdate(index, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order != nil {
		t.Errorf("expected nil for nonexistent package, got %v", order)
	}
}

func TestCascadeCircuitBreaker(t *testing.T) {
	cb := &CascadeCircuitBreaker{MaxFailures: 3, CorrelationID: "test-cascade"}

	if cb.IsBroken() {
		t.Error("should not be broken initially")
	}

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsBroken() {
		t.Error("should not be broken after 2 failures (threshold is 3)")
	}
	if cb.FailureCount() != 2 {
		t.Errorf("expected count 2, got %d", cb.FailureCount())
	}

	cb.RecordFailure()
	if !cb.IsBroken() {
		t.Error("should be broken after 3 failures")
	}

	// Success resets
	cb.RecordSuccess()
	if cb.IsBroken() {
		t.Error("should not be broken after success reset")
	}
	if cb.FailureCount() != 0 {
		t.Errorf("expected count 0 after reset, got %d", cb.FailureCount())
	}
}

package coordinator

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
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

// TestCascadeBudget_BasicFlow exercises the per-cascade cumulative cost
// gate added in M-PKG-AUTONOMOUS-CASCADE-SAFE M3. Charge accumulates,
// CanDispatch admits while there's headroom and rejects with a structured
// reason once the cap would be exceeded.
func TestCascadeBudget_BasicFlow(t *testing.T) {
	b := NewCascadeBudget(1.00, "cascade-1")

	if ok, _ := b.CanDispatch(0.30); !ok {
		t.Fatal("first task ($0.30 of $1.00) should be admitted")
	}
	b.Charge(0.30)

	if ok, _ := b.CanDispatch(0.40); !ok {
		t.Fatal("second task ($0.30 + est $0.40 = $0.70 of $1.00) should be admitted")
	}
	b.Charge(0.40)

	if ok, reason := b.CanDispatch(0.50); ok {
		t.Errorf("third task ($0.70 + est $0.50 = $1.20 of $1.00) should be rejected, got admitted")
	} else if reason == "" {
		t.Error("rejection reason should be non-empty")
	}

	if got := b.Used(); got != 0.70 {
		t.Errorf("Used() = $%.2f, want $0.70", got)
	}
	if got := b.AbortedCount(); got != 1 {
		t.Errorf("AbortedCount() = %d, want 1", got)
	}
}

// TestCascadeBudget_DefaultCap verifies a zero/negative cap falls back to
// the package-level default ($1.00) so callers can pass 0 to mean "use
// the default" without special-casing at every call site.
func TestCascadeBudget_DefaultCap(t *testing.T) {
	b := NewCascadeBudget(0, "cascade-default")
	if b.MaxCostUSD != 1.0 {
		t.Errorf("default cap: got $%.2f, want $1.00", b.MaxCostUSD)
	}

	b2 := NewCascadeBudget(-0.5, "cascade-neg")
	if b2.MaxCostUSD != 1.0 {
		t.Errorf("negative cap: got $%.2f, want $1.00", b2.MaxCostUSD)
	}
}

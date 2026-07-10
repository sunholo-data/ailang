package feedbackgate

import (
	"context"
	"testing"
	"time"
)

// fakeBudgetStore is an in-memory BudgetStore: it counts calls per day key.
type fakeBudgetStore struct {
	perDay map[string]int
}

func newFakeBudgetStore() *fakeBudgetStore {
	return &fakeBudgetStore{perDay: map[string]int{}}
}

func (f *fakeBudgetStore) IncrementDaily(_ context.Context, dayKey string, _ time.Time) (int, error) {
	f.perDay[dayKey]++
	return f.perDay[dayKey], nil
}

// TestBudgetUnderThenOver: with a $5/day cap at $0.0002/call the allowance is
// 25000 calls. Rather than loop 25k times, seed the store near the cap.
func TestBudgetUnderThenOver(t *testing.T) {
	store := newFakeBudgetStore()
	b := NewBudget(store)
	key := dayKey(time.Now())

	// Seed to exactly the cap - 1 so the next call is the 25000th (allowed),
	// and the one after is the 25001st (over).
	maxCalls := int(5.0 / classifierCallCostUSD) // 25000
	store.perDay[key] = maxCalls - 1

	ok, err := b.CheckAndReserve(context.Background(), 5.0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !ok {
		t.Fatalf("call at exactly the cap should be allowed")
	}
	ok2, _ := b.CheckAndReserve(context.Background(), 5.0)
	if ok2 {
		t.Fatalf("call over the cap should be rejected")
	}
}

// TestBudgetNilStoreAllows: a nil *Budget or nil store never caps.
func TestBudgetNilStoreAllows(t *testing.T) {
	var b *Budget
	ok, err := b.CheckAndReserve(context.Background(), 5.0)
	if err != nil || !ok {
		t.Fatalf("nil budget must allow: ok=%v err=%v", ok, err)
	}
	b2 := NewBudget(nil)
	ok2, _ := b2.CheckAndReserve(context.Background(), 5.0)
	if !ok2 {
		t.Fatal("nil store must allow")
	}
}

// TestClassifierBudgetForcesFile: over-budget classifier stage forces file and
// never calls the provider.
func TestClassifierBudgetForcesFile(t *testing.T) {
	store := newFakeBudgetStore()
	store.perDay[dayKey(time.Now())] = int(5.0/classifierCallCostUSD) + 5 // already over
	prov := &countingProvider{text: `{"is_genuine_feedback":true,"best_category":"bug","estimated_dispatch_value":"high"}`}

	cfg := FeedbackGateConfig{}.normalized()
	cfg.Classifier = NewClassifier(prov, DefaultPrompt(), NewBudget(store))

	v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionFile || v.Reason != ReasonBudgetExceeded {
		t.Fatalf("over-budget: got %q/%q, want file/classifier_budget_exceeded", v.Action, v.Reason)
	}
	if prov.calls != 0 {
		t.Fatalf("provider called %d times over budget, want 0", prov.calls)
	}
}

// TestClassifierBudgetUnderRuns: under budget, the classifier runs normally.
func TestClassifierBudgetUnderRuns(t *testing.T) {
	store := newFakeBudgetStore() // empty -> first call is under budget
	prov := &countingProvider{text: `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"high"}`}

	cfg := FeedbackGateConfig{}.normalized()
	cfg.Classifier = NewClassifier(prov, DefaultPrompt(), NewBudget(store))

	v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionDispatch {
		t.Fatalf("under budget: Action = %q, want dispatch", v.Action)
	}
	if prov.calls != 1 {
		t.Fatalf("provider called %d times, want 1", prov.calls)
	}
}

// TestBudgetDefaultOverridable checks the config default and override.
func TestBudgetDefaultOverridable(t *testing.T) {
	def := FeedbackGateConfig{}.normalized()
	if def.DailyBudgetUSD != 5.0 {
		t.Fatal("default daily budget should be $5")
	}
	custom := FeedbackGateConfig{DailyBudgetUSD: 20.0}.normalized()
	if custom.DailyBudgetUSD != 20.0 {
		t.Fatal("explicit daily budget should be preserved")
	}
}

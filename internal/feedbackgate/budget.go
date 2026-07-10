package feedbackgate

import (
	"context"
	"time"
)

// classifierCallCostUSD is the assumed cost of one Haiku classifier call,
// from the design doc (~$0.0002/call). Used to translate a daily budget in
// USD into an allowed call count. Not a live pricing lookup.
const classifierCallCostUSD = 0.0002

// BudgetStore tracks the classifier's daily call count. Mirrors the
// CooldownStore shape (a narrow single-method interface) so tests use an
// in-memory fake and the cloud wiring uses the same Firestore-backed adapter
// keyed by day.
type BudgetStore interface {
	// IncrementDaily records one classifier call for the given UTC day key and
	// returns the running count for that day (inclusive of this call).
	IncrementDaily(ctx context.Context, dayKey string, now time.Time) (count int, err error)
}

// Budget enforces the M5 daily classifier spend cap. On exceed, the classifier
// stage short-circuits to file (never dispatch) — worst case the inbox fills
// and a human triages; never a Sonnet flood.
type Budget struct {
	store BudgetStore
}

// NewBudget builds a Budget from a store. A nil store makes CheckAndReserve a
// permissive no-op (callers should treat nil *Budget as "no cap").
func NewBudget(store BudgetStore) *Budget {
	return &Budget{store: store}
}

// dayKey returns the UTC calendar-day key for now (YYYY-MM-DD).
func dayKey(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

// CheckAndReserve records one classifier call for today and reports whether it
// is within the dailyBudgetUSD cap. Returns true when the call is allowed
// (under budget), false when the cap is reached/exceeded. A nil store allows
// everything.
func (b *Budget) CheckAndReserve(ctx context.Context, dailyBudgetUSD float64) (bool, error) {
	if b == nil || b.store == nil {
		return true, nil
	}
	now := time.Now()
	count, err := b.store.IncrementDaily(ctx, dayKey(now), now)
	if err != nil {
		return false, err
	}
	maxCalls := int(dailyBudgetUSD / classifierCallCostUSD)
	// count is inclusive of the just-recorded call; allow up to maxCalls.
	return count <= maxCalls, nil
}

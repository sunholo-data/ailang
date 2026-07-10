package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
)

// This file provides the Firestore-backed implementations of the two
// feedbackgate injected dependencies constructed in the coordinator's cloud
// wiring (cmd/ailang/coordinator_lifecycle.go). It preserves layering: the
// firestore package depends on feedbackgate (for the interface types), never
// the reverse, and the coordinator stays firestore-free.
//
// Testability: no Firestore emulator is used (the repo convention — see
// CoordinatorStore / messaging.go). ALL window logic lives in the pure
// trimAndCount / appendCapped functions, which are exhaustively table-tested
// in feedbackgate_stores_test.go. The transaction wrappers are thin
// read-modify-write shells mirroring messaging.go:93.

// Compile-time assertions: the adapters MUST satisfy the feedbackgate
// interfaces. A loud regression guard if either interface drifts.
var (
	_ feedbackgate.CooldownStore = (*FeedbackGateCooldownStore)(nil)
	_ feedbackgate.BudgetStore   = (*FeedbackGateBudgetStore)(nil)
)

const (
	// collFeedbackGateCooldown holds one doc per contact key: a trimmed sliding
	// window of dispatch attempt timestamps.
	collFeedbackGateCooldown = "feedback_gate_cooldown"
	// collFeedbackGateBudget holds one doc per UTC day: the classifier call
	// count for that day.
	collFeedbackGateBudget = "feedback_gate_budget"

	// cooldownWindow is the trailing window trimAndCount retains. Attempts
	// strictly older than now-cooldownWindow are dropped. Matches the day
	// bound applyCooldown compares against (MaxDispatchPerDay).
	cooldownWindow = 24 * time.Hour
	// cooldownHourWindow is the trailing hour bound for the hourly count.
	cooldownHourWindow = time.Hour

	// cooldownSaturationCap bounds the timestamp array length per doc. Once a
	// contact's trimmed window holds this many attempts, we stop appending —
	// applyCooldown only tests > MaxDispatchPerHour/Day (3/10 default), so
	// precision above the cap is meaningless, and this bounds doc size under a
	// flood. 64 gives ample headroom above the day limit.
	cooldownSaturationCap = 64

	// cooldownTTL / budgetTTL set expires_at for Firestore TTL housekeeping.
	// The TTL POLICY itself is provisioned via terraform (documented handoff);
	// without it, stale docs are tiny and harmless (saturation-capped).
	cooldownTTL = 7 * 24 * time.Hour
	budgetTTL   = 3 * 24 * time.Hour
)

// --- Cooldown store ---

// FeedbackGateCooldownStore is the Firestore-backed feedbackgate.CooldownStore.
// It keeps a per-contact sliding window of dispatch-attempt timestamps in the
// feedback_gate_cooldown collection, keyed by a hash of the raw contact key.
type FeedbackGateCooldownStore struct {
	client *Client
}

// NewFeedbackGateCooldownStore builds the cooldown adapter from a Firestore
// client. Constructed only in cloud mode (see coordinator_lifecycle.go).
func NewFeedbackGateCooldownStore(client *Client) *FeedbackGateCooldownStore {
	return &FeedbackGateCooldownStore{client: client}
}

// cooldownDocID hashes the raw contact key (a |-joined string of arbitrary
// user text) into a safe, uniform, fixed-length Firestore document ID.
func cooldownDocID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:32]
}

// Increment records a dispatch attempt for key at now and returns the number of
// attempts within the trailing hour and day windows (inclusive of the
// just-recorded one, unless the saturation cap is hit). It runs a single
// read-modify-write transaction: read the current window, trim to 24h, append
// now (capped), and write back with a fresh TTL.
func (s *FeedbackGateCooldownStore) Increment(ctx context.Context, key string, now time.Time) (hourCount, dayCount int, err error) {
	ref := s.client.Doc(collFeedbackGateCooldown, cooldownDocID(key))

	txErr := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		attempts, err := readAttempts(tx, ref)
		if err != nil {
			// A transient read failure must ABORT the transaction so we never
			// commit a reset window over stored state. Returning it here fails
			// closed: RunTransaction aborts, Increment surfaces the error,
			// applyCooldown propagates, and the M4 wiring blocks with an audit.
			return err
		}

		kept, _ := appendCapped(attempts, now)
		_, hourCount, dayCount = trimAndCount(kept, now)

		return tx.Set(ref, map[string]interface{}{
			"key":        key,
			"attempts":   kept,
			"expires_at": now.Add(cooldownTTL),
		})
	})
	if txErr != nil {
		return 0, 0, txErr
	}
	return hourCount, dayCount, nil
}

// readAttempts reads the stored attempt timestamps from a doc snapshot inside a
// transaction. A missing doc yields an empty window with no error (the
// legitimate first attempt for the key). Any OTHER read error is returned so
// the caller can abort the transaction rather than commit a reset window —
// mirroring messaging.go's GetOrCreateThreadWithWorkspace, which returns
// non-Done iterator errors from its transaction fn. Failing closed here is a
// deliberate no-silent-fallback decision (CLAUDE.md Critical Principle 2):
// cooldown/budget protection must not silently reset under Firestore
// degradation.
func readAttempts(tx *firestore.Transaction, ref *firestore.DocumentRef) ([]time.Time, error) {
	snap, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil // first attempt for this key — legitimate empty window
		}
		return nil, err // transient/other read failure → abort, do not reset the window
	}
	data := snap.Data()
	raw, ok := data["attempts"].([]interface{})
	if !ok {
		// Deterministic corruption (attempts field absent or not a list), not a
		// transient failure: treat as an empty window defensively so a mangled
		// doc self-heals on the next write rather than wedging the gate.
		return nil, nil
	}
	out := make([]time.Time, 0, len(raw))
	for _, v := range raw {
		if t, ok := v.(time.Time); ok {
			out = append(out, t)
		}
		// Non-time entries are deterministic corruption, not transient
		// degradation: skipped defensively (they self-heal on the next write).
	}
	return out, nil
}

// trimAndCount drops attempts strictly older than cooldownWindow (24h) before
// now, then counts how many KEPT attempts fall within the trailing hour and day
// windows. The windows are half-open on the old side: an attempt exactly at
// now-24h (or now-1h) is KEPT/counted; strictly older is dropped. Pure: no I/O,
// no clock — now is passed in. This is the fully-tested core of the adapter.
func trimAndCount(attempts []time.Time, now time.Time) (kept []time.Time, hour, day int) {
	dayCutoff := now.Add(-cooldownWindow)
	hourCutoff := now.Add(-cooldownHourWindow)
	kept = make([]time.Time, 0, len(attempts))
	for _, a := range attempts {
		if a.Before(dayCutoff) {
			continue // strictly older than 24h → trimmed
		}
		kept = append(kept, a)
		day++
		if !a.Before(hourCutoff) {
			hour++
		}
	}
	return kept, hour, day
}

// appendCapped trims attempts to the 24h window, then appends now UNLESS the
// trimmed window already holds cooldownSaturationCap entries. Returns the new
// window and whether the append happened. Saturation bounds doc size under a
// flood; the counts are already well over MaxDispatchPerHour/Day by then, so
// the gate files regardless.
func appendCapped(attempts []time.Time, now time.Time) (kept []time.Time, appended bool) {
	kept, _, _ = trimAndCount(attempts, now)
	if len(kept) >= cooldownSaturationCap {
		return kept, false
	}
	kept = append(kept, now)
	return kept, true
}

// --- Budget store ---

// FeedbackGateBudgetStore is the Firestore-backed feedbackgate.BudgetStore. It
// keeps one counter doc per UTC day in the feedback_gate_budget collection,
// tracking the classifier's daily call count.
type FeedbackGateBudgetStore struct {
	client *Client
}

// NewFeedbackGateBudgetStore builds the budget adapter from a Firestore client.
func NewFeedbackGateBudgetStore(client *Client) *FeedbackGateBudgetStore {
	return &FeedbackGateBudgetStore{client: client}
}

// IncrementDaily records one classifier call for dayKey (YYYY-MM-DD, already
// UTC-normalized by feedbackgate.dayKey) and returns the running count for that
// day (inclusive of this call). Single read-modify-write transaction.
func (s *FeedbackGateBudgetStore) IncrementDaily(ctx context.Context, dayKey string, now time.Time) (count int, err error) {
	ref := s.client.Doc(collFeedbackGateBudget, dayKey)

	txErr := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		current, err := readCount(tx, ref)
		if err != nil {
			// Same fail-closed rationale as Increment: a transient read failure
			// must abort rather than reset today's budget counter to zero.
			return err
		}
		count = current + 1
		return tx.Set(ref, map[string]interface{}{
			"count":      count,
			"expires_at": now.Add(budgetTTL),
		})
	})
	if txErr != nil {
		return 0, txErr
	}
	return count, nil
}

// readCount reads the current daily count from a doc snapshot inside a
// transaction. A missing doc means zero calls so far today (returned with no
// error). Any OTHER read error is returned so the caller aborts the
// transaction rather than resetting the budget — the same fail-closed,
// no-silent-fallback decision as readAttempts (CLAUDE.md Critical Principle 2).
func readCount(tx *firestore.Transaction, ref *firestore.DocumentRef) (int, error) {
	snap, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, nil // no doc yet today — legitimate zero
		}
		return 0, err // transient/other read failure → abort, do not reset the counter
	}
	data := snap.Data()
	switch v := data["count"].(type) {
	case int64:
		return int(v), nil
	case int:
		return v, nil
	default:
		// Deterministic corruption (count field absent or wrong type), not a
		// transient failure: treat as zero defensively so a mangled doc
		// self-heals on the next write rather than wedging the budget gate.
		return 0, nil
	}
}

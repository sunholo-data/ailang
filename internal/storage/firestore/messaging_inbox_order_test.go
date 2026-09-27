package firestore

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// The sender filter orders and limits client-side (no composite index for
// from_agent); it must return what OrderBy(created_at desc).Limit(n) would.
// MU: drop the sort in newestFirst and the order check fails.
func TestNewestFirstMatchesServerOrdering(t *testing.T) {
	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	msgs := []messaging.InboxMessage{
		{ID: "old", CreatedAt: base},
		{ID: "newest", CreatedAt: base.Add(2 * time.Hour)},
		{ID: "mid", CreatedAt: base.Add(time.Hour)},
	}
	got := newestFirst(msgs, 2)
	if len(got) != 2 || got[0].ID != "newest" || got[1].ID != "mid" {
		t.Fatalf("got %v, want [newest mid]", []string{got[0].ID, got[1].ID})
	}
	if all := newestFirst(msgs, 0); len(all) != 3 {
		t.Errorf("limit 0 kept %d, want all 3", len(all))
	}
}

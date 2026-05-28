package coordinator

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// TestTriageRouterIntegration exercises the router against a real SQLite store:
// a categorized bug is promoted to design-doc-creator, a general note is held in
// place, and a duplicate is never forwarded.
func TestTriageRouterIntegration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "triage.db")
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	seed := func(category, from, title, dupOf string) {
		m := &messaging.InboxMessage{
			ToInbox:   "user",
			FromAgent: from,
			Category:  category,
			Title:     title,
			DupOf:     dupOf,
		}
		if err := store.InsertInboxMessage(m); err != nil {
			t.Fatalf("insert %q: %v", title, err)
		}
	}
	seed("bug", "cli", "real bug report", "")
	seed("general", "cli", "vague note", "")
	seed("bug", "cli", "duplicate bug", "some-original-id") // hidden by Collapsed

	router := NewTriageRouter(store, TriageConfig{IntakeInboxes: []string{"user"}}, nil)

	if promoted := router.tickOnce(context.Background()); promoted != 1 {
		t.Fatalf("expected 1 promotion, got %d", promoted)
	}

	// The bug now lives in design-doc-creator.
	ddc, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: "design-doc-creator"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ddc) != 1 || ddc[0].Title != "real bug report" {
		t.Fatalf("expected only 'real bug report' in design-doc-creator, got %v", inboxTitles(ddc))
	}

	// The general note stays held in the user inbox; the bug left it.
	userMsgs, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: "user", UnreadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	titles := inboxTitles(userMsgs)
	if !stringInSlice(titles, "vague note") {
		t.Errorf("general note should remain in user inbox, got %v", titles)
	}
	if stringInSlice(titles, "real bug report") {
		t.Errorf("promoted bug should have left the user inbox, got %v", titles)
	}
}

func inboxTitles(msgs []messaging.InboxMessage) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = m.Title
	}
	return out
}

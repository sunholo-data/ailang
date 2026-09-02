package main

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func msg(created time.Time, inbox, from, mtype, status, payload string) messaging.InboxMessage {
	return messaging.InboxMessage{
		CreatedAt: created, ToInbox: inbox, FromAgent: from,
		MessageType: mtype, Status: status, Payload: payload,
	}
}

func TestSummarizeActivityCountsOutcomesIncludingNoChanges(t *testing.T) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)
	msgs := []messaging.InboxMessage{
		msg(now.Add(-1*time.Hour), "docparse", "mission-v1", "notification", "unread", ""),
		msg(now.Add(-2*time.Hour), "docparse", "docparse", "completion", "read", `{"status":"no_changes"}`),
		msg(now.Add(-3*time.Hour), "docparse", "docparse", "completion", "read", `{"status":"completed"}`),
		msg(now.Add(-4*time.Hour), "pkg:x", "agent-x", "completion", "read", `{"status":"failed"}`),
		msg(now.Add(-5*time.Hour), "pkg:x", "agent-x", "completion", "read", `not json`),
		// Outside the window — must be excluded, not silently folded in.
		msg(now.Add(-48*time.Hour), "docparse", "old", "notification", "unread", ""),
	}

	a := summarizeActivity(msgs, since, 24)

	if a.Total != 5 {
		t.Fatalf("Total = %d, want 5 (the 48h-old message must be excluded)", a.Total)
	}
	// no_changes is the outcome that used to be invisible — it is the reason
	// this view exists at all.
	if a.ByOutcome["no_changes"] != 1 {
		t.Errorf("no_changes = %d, want 1", a.ByOutcome["no_changes"])
	}
	if a.ByOutcome["completed"] != 1 || a.ByOutcome["failed"] != 1 {
		t.Errorf("outcomes = %v", a.ByOutcome)
	}
	// A completion whose payload will not parse must be counted, not dropped:
	// silently discarding the rows you cannot read is how a summary lies.
	if a.ByOutcome["unparseable"] != 1 {
		t.Errorf("unparseable = %d, want 1 — an unreadable completion must still be counted", a.ByOutcome["unparseable"])
	}
	if a.ByType["completion"] != 4 || a.ByType["notification"] != 1 {
		t.Errorf("by type = %v", a.ByType)
	}
	if a.ByInbox["docparse"] != 3 || a.ByInbox["pkg:x"] != 2 {
		t.Errorf("by inbox = %v", a.ByInbox)
	}
}

func TestSummarizeActivityTracksUnreadAndItsAge(t *testing.T) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)
	oldest := now.Add(-10 * time.Hour)
	msgs := []messaging.InboxMessage{
		msg(now.Add(-1*time.Hour), "a", "s", "notification", "unread", ""),
		msg(oldest, "a", "s", "notification", "unread", ""),
		msg(now.Add(-2*time.Hour), "a", "s", "notification", "read", ""),
	}

	a := summarizeActivity(msgs, since, 24)
	if a.Unread != 2 {
		t.Errorf("Unread = %d, want 2", a.Unread)
	}
	if a.OldestUnread != oldest.UTC().Format(time.RFC3339) {
		t.Errorf("OldestUnread = %q, want the oldest of the two", a.OldestUnread)
	}
}

func TestSummarizeActivityEmptyWindowIsNotAnError(t *testing.T) {
	a := summarizeActivity(nil, time.Now().Add(-time.Hour), 1)
	if a.Total != 0 || a.Unread != 0 || a.OldestUnread != "" {
		t.Errorf("an empty window must summarize cleanly, got %+v", a)
	}
}

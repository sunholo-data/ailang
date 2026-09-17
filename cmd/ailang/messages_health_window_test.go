package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseHealthWindow(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want time.Duration
		bad  bool
	}{
		{"", 24 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"0.5d", 12 * time.Hour, false},
		{"0", 0, false},   // judge everything
		{"all", 0, false}, // same, spelled the way people say it
		{"yesterday", 0, true},
		{"-3h", 0, true},
	} {
		got, err := parseHealthWindow(tc.in)
		if tc.bad {
			if err == nil {
				t.Errorf("parseHealthWindow(%q) = %v, want an error naming the accepted forms", tc.in, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("parseHealthWindow(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
}

// The bug this window exists for: fifteen undelivered handoffs, none newer than
// 35 hours, made the banner read DEGRADED while the plane had dispatched
// everything it received all day. Old debt and a live outage must not print the
// same.
func TestSummarizeHealth_BacklogIsNotALiveFault(t *testing.T) {
	msgs := []healthMsg{
		{Inbox: "sprint-planner", Bucket: bucketRoutable, Age: 35 * time.Hour},
		{Inbox: "sprint-planner", Bucket: bucketRoutable, Age: 9 * 24 * time.Hour},
		{Inbox: "ailang", Bucket: bucketUnroutable, Age: 61 * time.Hour},
		{Inbox: "ailang-core", Bucket: bucketResult, Age: 2 * time.Hour},
		{Inbox: "user", Bucket: bucketTriage, Age: 30 * time.Hour},
	}
	s := summarizeHealth(msgs, 24*time.Hour)
	if s.LiveFaults() != 0 {
		t.Errorf("LiveFaults = %d, want 0: nothing arrived in the window and stalled", s.LiveFaults())
	}
	if s.Backlog() != 3 {
		t.Errorf("Backlog = %d, want 3", s.Backlog())
	}
	if s.Routable.Total != 2 || s.Routable.New != 0 || s.Routable.Backlog != 2 {
		t.Errorf("routable = %+v, want total 2 / new 0 / backlog 2", s.Routable)
	}
	if s.Unread != 5 {
		t.Errorf("unread total = %d, want 5", s.Unread)
	}

	// The same data with no window is the all-time judgement, and there
	// everything counts as live — that is what --since 0 asks for.
	all := summarizeHealth(msgs, 0)
	if all.LiveFaults() != 3 {
		t.Errorf("--since 0: LiveFaults = %d, want 3", all.LiveFaults())
	}
	if all.Backlog() != 0 {
		t.Errorf("--since 0: Backlog = %d, want 0 — with no window nothing is 'older'", all.Backlog())
	}
}

func TestSummarizeHealth_NewWorkSurfacesAboveOldDebt(t *testing.T) {
	msgs := []healthMsg{
		// Nine days of backlog on one inbox...
		{Inbox: "sprint-planner", Bucket: bucketRoutable, Age: 9 * 24 * time.Hour},
		{Inbox: "sprint-planner", Bucket: bucketRoutable, Age: 8 * 24 * time.Hour},
		{Inbox: "sprint-planner", Bucket: bucketRoutable, Age: 7 * 24 * time.Hour},
		// ...and ONE message that arrived an hour ago and went nowhere.
		{Inbox: "design-doc-creator", Bucket: bucketRoutable, Age: time.Hour},
	}
	s := summarizeHealth(msgs, 24*time.Hour)
	if s.LiveFaults() != 1 {
		t.Fatalf("LiveFaults = %d, want 1", s.LiveFaults())
	}
	if got := s.Routable.Rows[0].Inbox; got != "design-doc-creator" {
		t.Errorf("first row = %q, want design-doc-creator: today's fault must not sort below the backlog", got)
	}
	// The oldest age per row is what says "this is debt" at a glance.
	for _, r := range s.Routable.Rows {
		if r.Inbox == "sprint-planner" && r.Oldest != 9*24*time.Hour {
			t.Errorf("sprint-planner oldest = %v, want 9d", r.Oldest)
		}
	}
}

func TestHumanAge(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{0, "-"},
		{30 * time.Minute, "30m"},
		{90 * time.Minute, "1.5h"},
		{40 * time.Hour, "40.0h"},
		{9 * 24 * time.Hour, "9.0d"},
	} {
		if got := humanAge(tc.in); got != tc.want {
			t.Errorf("humanAge(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if humanWindow(0) != "all time" {
		t.Error("a zero window must read as all time, not 'last 0h'")
	}
}

func TestStripANSIAndBriefError(t *testing.T) {
	if got := stripANSI(red("BROKEN") + " pubsub disabled"); got != "BROKEN pubsub disabled" {
		t.Errorf("stripANSI = %q — a JSON consumer greps for the word, not the escape", got)
	}
	long := "rpc error: code = FailedPrecondition desc = The query requires an index. " + strings.Repeat("x", 300)
	if got := briefError(long); len(got) > 120 {
		t.Errorf("briefError left %d chars — one backend error must not take over the report", len(got))
	}
	if got := briefError("first\nsecond"); got != "first" {
		t.Errorf("briefError = %q, want the first line", got)
	}
}

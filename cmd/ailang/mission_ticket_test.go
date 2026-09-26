package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/mission"
)

func newTicketTestStore(t *testing.T) *messaging.Store {
	t.Helper()
	s, err := messaging.OpenStore(filepath.Join(t.TempDir(), "inbox.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func runTicket(t *testing.T, s ticketStore, sub string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := missionTicketWithStore(sub, args, s, &out)
	return strings.TrimSpace(out.String()), err
}

func fileTicket(t *testing.T, s ticketStore, missionName, iter, sig string) {
	t.Helper()
	if _, err := runTicket(t, s, "file", "--mission", missionName, "--iteration", iter,
		"--signature", sig, "--evidence", "measured: "+sig); err != nil {
		t.Fatalf("file %s/%s: %v", missionName, sig, err)
	}
}

func TestMissionTicketRoundTrip(t *testing.T) {
	s := newTicketTestStore(t)
	fileTicket(t, s, "world", "192", "stall:gate-3")
	fileTicket(t, s, "v1", "400", "stall:gate-3")
	fileTicket(t, s, "docs", "12", "probe-timeout:codex")

	if got, _ := runTicket(t, s, "open", "--count"); got != "2" {
		t.Fatalf("open --count = %q, want 2 signatures", got)
	}
	// Listing must NOT mark read — a second read sees the same open work.
	if got, _ := runTicket(t, s, "open", "--count"); got != "2" {
		t.Fatalf("open marked tickets read: second count = %q", got)
	}
	js, err := runTicket(t, s, "open", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var groups []mission.OpenSignature
	if err := json.Unmarshal([]byte(js), &groups); err != nil {
		t.Fatalf("open --json: %v\n%s", err, js)
	}
	if groups[0].Signature != "stall:gate-3" || groups[0].SlotsLost != 2 {
		t.Fatalf("top signature = %+v", groups[0])
	}

	msg, err := runTicket(t, s, "resolve", "stall:gate-3", "--resolution", "opus before pi", "--sha", "d68e51e8e")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(msg, "2 occurrence(s)") {
		t.Fatalf("resolve output = %q", msg)
	}
	if got, _ := runTicket(t, s, "open", "--count"); got != "1" {
		t.Fatalf("after resolve open --count = %q, want 1", got)
	}
	for _, inbox := range []string{"mission-world", "mission-v1"} {
		replies, err := s.ListInboxMessages(messaging.InboxListOptions{Inbox: inbox, UnreadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(replies) != 1 || replies[0].Category != mission.ResolvedCategory ||
			!strings.Contains(replies[0].Payload, "d68e51e8e") {
			t.Fatalf("%s did not get exactly one resolution reply: %+v", inbox, replies)
		}
	}
	if replies, _ := s.ListInboxMessages(messaging.InboxListOptions{Inbox: "mission-docs"}); len(replies) != 0 {
		t.Fatalf("docs filed a different signature and must not be told: %+v", replies)
	}
}

func TestMissionTicketRefusals(t *testing.T) {
	s := newTicketTestStore(t)
	if _, err := runTicket(t, s, "file", "--mission", "fleet", "--iteration", "1", "--signature", "x", "--evidence", "y"); err == nil {
		t.Fatal("fleet must not be able to file a ticket")
	}
	if _, err := runTicket(t, s, "file", "--mission", "v1", "--iteration", "1", "--signature", "x"); err == nil {
		t.Fatal("a ticket without evidence must be refused")
	}
	if _, err := runTicket(t, s, "resolve", "nothing-open", "--resolution", "r"); err == nil {
		t.Fatal("resolving an unknown signature must fail")
	}
	fileTicket(t, s, "v1", "1", "x")
	if _, err := runTicket(t, s, "resolve", "x"); err == nil {
		t.Fatal("resolve without --resolution must fail (the filer unparks on it)")
	}
	if got, _ := runTicket(t, s, "open", "--count"); got != "1" {
		t.Fatalf("a refused resolve must leave the ticket open, count = %q", got)
	}
}

func TestMissionTicketUnparseableStillCountsOpen(t *testing.T) {
	s := newTicketTestStore(t)
	if err := s.InsertInboxMessage(&messaging.InboxMessage{
		FromAgent: "mission-v1", ToInbox: mission.FleetInbox, MessageType: "request", Category: mission.TicketCategory,
		Title: "[harness] hand-written", Payload: "not json",
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := runTicket(t, s, "open", "--count"); got != "1" {
		t.Fatalf("an unparseable ticket must count as open work, got %q", got)
	}
}

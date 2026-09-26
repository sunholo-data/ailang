package mission

import (
	"strings"
	"testing"
	"time"
)

func validTicket() Ticket {
	return Ticket{
		Mission:     "world",
		Iteration:   192,
		Signature:   "stall:gate-3:pi-openrouter-executor",
		SlotVerdict: "KILLED_at=gate-3 rc=143",
		Blocking:    BlockingNone,
		Evidence:    "STALL: claude 16486 made NO PROGRESS across 5 samples",
		Workaround:  "none",
	}
}

func TestTicketValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Ticket)
		wantErr string
	}{
		{"valid", func(*Ticket) {}, ""},
		{"no mission", func(tk *Ticket) { tk.Mission = "" }, "mission"},
		{"fleet cannot file", func(tk *Ticket) { tk.Mission = "fleet" }, "fleet"},
		{"no signature", func(tk *Ticket) { tk.Signature = "  " }, "signature"},
		{"signature with separator", func(tk *Ticket) { tk.Signature = "a · b" }, "signature"},
		{"bad blocking", func(tk *Ticket) { tk.Blocking = "maybe" }, "blocking"},
		{"no evidence", func(tk *Ticket) { tk.Evidence = "" }, "evidence"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tk := validTicket()
			c.mutate(&tk)
			err := tk.Validate()
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("want error mentioning %q, got %v", c.wantErr, err)
			}
		})
	}
}

func TestTicketEvidenceTruncated(t *testing.T) {
	tk := validTicket()
	tk.Evidence = strings.Repeat("x", TicketEvidenceMax+500)
	tk.Normalize()
	if len(tk.Evidence) > TicketEvidenceMax+len(ticketTruncMarker) {
		t.Fatalf("evidence not truncated: %d bytes", len(tk.Evidence))
	}
	if !strings.HasSuffix(tk.Evidence, ticketTruncMarker) {
		t.Fatalf("truncated evidence must say so")
	}
}

func TestTicketTitleAndCorrelationStable(t *testing.T) {
	tk := validTicket()
	if got, want := tk.Title(), "[harness] stall:gate-3:pi-openrouter-executor · world#192"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	if got := tk.CorrelationID(); got != "harness:stall:gate-3:pi-openrouter-executor" {
		t.Fatalf("correlation = %q", got)
	}
	// A title is per OCCURRENCE: the same signature from another iteration must differ,
	// or the inbox's per-title dedupe would swallow the second occurrence.
	other := validTicket()
	other.Iteration = 193
	if other.Title() == tk.Title() {
		t.Fatal("two occurrences of one signature must have distinct titles")
	}
}

func TestParseTicketRoundTrip(t *testing.T) {
	tk := validTicket()
	body, err := tk.Payload()
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseTicket(body)
	if err != nil {
		t.Fatal(err)
	}
	if back.Signature != tk.Signature || back.Mission != tk.Mission || back.Iteration != tk.Iteration {
		t.Fatalf("round trip lost fields: %+v", back)
	}
}

func TestGroupOpenRanksByOccurrencesThenAge(t *testing.T) {
	t0 := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	occ := func(sig, mission string, iter int, at time.Duration) TicketOccurrence {
		tk := validTicket()
		tk.Signature, tk.Mission, tk.Iteration = sig, mission, iter
		return TicketOccurrence{ID: sig + mission, Ticket: tk, CreatedAt: t0.Add(at)}
	}
	got := GroupOpen([]TicketOccurrence{
		occ("young-single", "v1", 1, 3*time.Hour),
		occ("hot", "world", 1, 2*time.Hour),
		occ("old-single", "docs", 1, 0),
		occ("hot", "v1", 2, 4*time.Hour),
		occ("hot", "world", 2, 5*time.Hour),
	})
	if len(got) != 3 {
		t.Fatalf("want 3 signatures, got %d", len(got))
	}
	if got[0].Signature != "hot" || got[0].SlotsLost != 3 {
		t.Fatalf("most-occurring signature must rank first: %+v", got[0])
	}
	if strings.Join(got[0].Missions, ",") != "v1,world" {
		t.Fatalf("missions must be distinct and sorted: %v", got[0].Missions)
	}
	if got[1].Signature != "old-single" || got[2].Signature != "young-single" {
		t.Fatalf("ties break oldest-first: %s, %s", got[1].Signature, got[2].Signature)
	}
	if !got[0].FirstSeen.Equal(t0.Add(2 * time.Hour)) {
		t.Fatalf("first seen = %v", got[0].FirstSeen)
	}
}

func TestGroupOpenBlockingIsWorstOccurrence(t *testing.T) {
	a, b := validTicket(), validTicket()
	b.Iteration, b.Blocking = 193, BlockingAll
	got := GroupOpen([]TicketOccurrence{{ID: "1", Ticket: a}, {ID: "2", Ticket: b}})
	if got[0].Blocking != BlockingAll {
		t.Fatalf("group blocking must be the worst occurrence, got %q", got[0].Blocking)
	}
}

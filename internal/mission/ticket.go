package mission

// Harness tickets (M-HARNESS-MISSION-LOOP). A product loop that hits a defect in the
// loop harness does not fix it: it files a ticket to the mission-fleet inbox and goes
// back to product work. The fleet loop works those tickets and nothing else.
//
// One message per OCCURRENCE, grouped by signature. The inbox dedupes on title, so a
// per-signature title would swallow every repeat — and the repeat count is exactly the
// number the fleet ranks by (product slots this defect has cost).

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// FleetInbox is where harness tickets go. Declared triage (read by the fleet loop at
// Gate 0, never dispatched by the coordinator).
const FleetInbox = "mission-fleet"

// FleetMission is the one mission that works tickets and may therefore not file them.
const FleetMission = "fleet"

// Ticket messages are store type "request" and the fleet's replies "response" (the store
// constrains message_type to a fixed set); the CATEGORY carries what they are.
const (
	TicketCategory   = "harness-friction"
	ResolvedCategory = "harness-resolved"
)

// TicketEvidenceMax bounds the evidence field; logs go in by excerpt, not whole.
const TicketEvidenceMax = 2048

const ticketTruncMarker = "\n[… truncated]"

// Blocking says how much of the filing loop's work the defect stops.
const (
	BlockingNone = "none" // costs time or a degraded lane; work continues
	BlockingItem = "item" // the current row cannot proceed; others can
	BlockingAll  = "all"  // nothing admissible remains; the loop yields
)

var blockingRank = map[string]int{BlockingNone: 0, BlockingItem: 1, BlockingAll: 2}

// Ticket is one occurrence of a harness defect, as filed by a product loop.
type Ticket struct {
	Mission     string `json:"mission"`
	Iteration   int    `json:"iteration"`
	FireStarted string `json:"fire_started,omitempty"`
	Signature   string `json:"signature"`
	SlotVerdict string `json:"slot_verdict,omitempty"`
	Blocking    string `json:"blocking"`
	Evidence    string `json:"evidence"`
	Workaround  string `json:"workaround,omitempty"`
}

// Normalize trims fields and bounds the evidence. Call before Validate.
func (t *Ticket) Normalize() {
	t.Mission = strings.TrimSpace(t.Mission)
	t.Signature = strings.TrimSpace(t.Signature)
	t.Blocking = strings.TrimSpace(t.Blocking)
	if t.Blocking == "" {
		t.Blocking = BlockingNone
	}
	if t.Workaround == "" {
		t.Workaround = "none"
	}
	if len(t.Evidence) > TicketEvidenceMax {
		t.Evidence = t.Evidence[:TicketEvidenceMax] + ticketTruncMarker
	}
}

// Validate refuses a ticket the fleet could not act on or group.
func (t Ticket) Validate() error {
	switch {
	case t.Mission == "":
		return fmt.Errorf("ticket: mission is required")
	case t.Mission == FleetMission:
		return fmt.Errorf("ticket: the fleet mission works tickets; it does not file them")
	case strings.TrimSpace(t.Signature) == "":
		return fmt.Errorf("ticket: signature is required (a stable key such as stall:gate-3:<lane>)")
	case strings.Contains(t.Signature, "·"):
		return fmt.Errorf("ticket: signature may not contain '·' (the title separator)")
	case strings.TrimSpace(t.Evidence) == "":
		return fmt.Errorf("ticket: evidence is required — a ticket without a measurement is a guess")
	}
	if _, ok := blockingRank[t.Blocking]; !ok {
		return fmt.Errorf("ticket: blocking must be none, item or all (got %q)", t.Blocking)
	}
	return nil
}

// Title is unique per occurrence (mission + iteration), so the inbox's title dedupe
// never swallows a repeat.
func (t Ticket) Title() string {
	return fmt.Sprintf("[harness] %s · %s#%d", t.Signature, t.Mission, t.Iteration)
}

// CorrelationID groups every occurrence of one signature.
func (t Ticket) CorrelationID() string { return "harness:" + t.Signature }

// Payload is the ticket as the message body.
func (t Ticket) Payload() (string, error) {
	b, err := json.Marshal(t)
	return string(b), err
}

// ParseTicket reads a message body back into a ticket.
func ParseTicket(payload string) (Ticket, error) {
	var t Ticket
	if err := json.Unmarshal([]byte(payload), &t); err != nil {
		return Ticket{}, fmt.Errorf("ticket: payload is not a ticket: %w", err)
	}
	return t, nil
}

// TicketOccurrence is a filed ticket as read back from the inbox.
type TicketOccurrence struct {
	ID        string
	Ticket    Ticket
	CreatedAt time.Time
}

// OpenSignature is one defect with every open occurrence of it.
type OpenSignature struct {
	Signature string    `json:"signature"`
	SlotsLost int       `json:"slots_lost"`
	Blocking  string    `json:"blocking"`
	Missions  []string  `json:"missions"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	IDs       []string  `json:"message_ids"`
	Latest    Ticket    `json:"latest"`
}

// GroupOpen folds open occurrences into signatures, ranked by slots lost (the cost to
// product loops), then oldest first so a quiet defect is not starved forever.
func GroupOpen(occ []TicketOccurrence) []OpenSignature {
	bySig := map[string]*OpenSignature{}
	missions := map[string]map[string]bool{}
	for _, o := range occ {
		g, ok := bySig[o.Ticket.Signature]
		if !ok {
			g = &OpenSignature{Signature: o.Ticket.Signature, Blocking: BlockingNone, FirstSeen: o.CreatedAt, LastSeen: o.CreatedAt, Latest: o.Ticket}
			bySig[o.Ticket.Signature] = g
			missions[o.Ticket.Signature] = map[string]bool{}
		}
		g.SlotsLost++
		g.IDs = append(g.IDs, o.ID)
		missions[o.Ticket.Signature][o.Ticket.Mission] = true
		if blockingRank[o.Ticket.Blocking] > blockingRank[g.Blocking] {
			g.Blocking = o.Ticket.Blocking
		}
		if o.CreatedAt.Before(g.FirstSeen) {
			g.FirstSeen = o.CreatedAt
		}
		if !o.CreatedAt.Before(g.LastSeen) {
			g.LastSeen, g.Latest = o.CreatedAt, o.Ticket
		}
	}
	out := make([]OpenSignature, 0, len(bySig))
	for sig, g := range bySig {
		for m := range missions[sig] {
			g.Missions = append(g.Missions, m)
		}
		sort.Strings(g.Missions)
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SlotsLost != out[j].SlotsLost {
			return out[i].SlotsLost > out[j].SlotsLost
		}
		if !out[i].FirstSeen.Equal(out[j].FirstSeen) {
			return out[i].FirstSeen.Before(out[j].FirstSeen)
		}
		return out[i].Signature < out[j].Signature
	})
	return out
}

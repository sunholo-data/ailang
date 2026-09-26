package main

// `ailang mission ticket` — harness tickets (M-HARNESS-MISSION-LOOP).
//
// A product loop that hits a defect in the loop harness FILES it here instead of fixing it,
// and goes back to product work. The fleet loop reads open tickets, fixes the top one, and
// RESOLVES it: one reply per filing mission, then every occurrence is marked read.
//
// "Open" means UNREAD in the mission-fleet inbox. That only holds if nothing marks a ticket
// read except `resolve` — so `open` lists without marking, and the fleet charter forbids
// `ailang messages read` on this inbox.

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/mission"
)

// ticketStore is the slice of the message store tickets need.
type ticketStore interface {
	InsertInboxMessage(msg *messaging.InboxMessage) error
	ListInboxMessages(opts messaging.InboxListOptions) ([]messaging.InboxMessage, error)
	MarkInboxMessageRead(id string) error
}

func missionTicket(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printMissionTicketHelp(os.Stdout)
		return nil
	}
	sub := args[0]
	switch sub {
	case "file", "open", "resolve":
	default:
		return fmt.Errorf("unknown ticket subcommand %q (want: file, open, resolve)", sub)
	}
	// The guard runs FORCED: until mission-fleet is declared on the plane a normal send
	// would be refused, and a loop that cannot file a ticket falls back to fixing the
	// harness itself — the failure this command exists to end. It still prints its warning.
	if sub == "file" {
		guardSendInbox(mission.FleetInbox, true)
	}
	store, err := openStore()
	if err != nil {
		return err
	}
	defer store.Close()
	return missionTicketWithStore(sub, args[1:], store, os.Stdout)
}

func missionTicketWithStore(sub string, args []string, store ticketStore, out io.Writer) error {
	switch sub {
	case "file":
		return ticketFile(args, store, out)
	case "open":
		return ticketOpen(args, store, out)
	case "resolve":
		return ticketResolve(args, store, out)
	}
	return fmt.Errorf("unknown ticket subcommand %q", sub)
}

func ticketFile(args []string, store ticketStore, out io.Writer) error {
	fs := flag.NewFlagSet("mission ticket file", flag.ContinueOnError)
	var t mission.Ticket
	fs.StringVar(&t.Mission, "mission", config.MissionName(), "Filing mission (default $MISSION_NAME)")
	fs.IntVar(&t.Iteration, "iteration", 0, "Iteration number that hit the defect")
	fs.StringVar(&t.FireStarted, "fire-started", "", "Fire start time (RFC3339), if known")
	fs.StringVar(&t.Signature, "signature", "", "Stable dedupe key, e.g. stall:gate-3:pi-openrouter-executor")
	fs.StringVar(&t.SlotVerdict, "slot-verdict", "", "Line from mission-<name>-slot-verdicts.log")
	fs.StringVar(&t.Blocking, "blocking", mission.BlockingNone, "none | item | all")
	fs.StringVar(&t.Evidence, "evidence", "", "Measured evidence (bounded to 2 KB)")
	evidenceFile := fs.String("evidence-file", "", "Read evidence from this file instead")
	fs.StringVar(&t.Workaround, "workaround", "", "The one reversible local workaround applied, or none")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *evidenceFile != "" {
		b, err := os.ReadFile(*evidenceFile)
		if err != nil {
			return fmt.Errorf("ticket: read evidence: %w", err)
		}
		t.Evidence = string(b)
	}
	t.Normalize()
	if err := t.Validate(); err != nil {
		return err
	}
	payload, err := t.Payload()
	if err != nil {
		return err
	}
	msg := &messaging.InboxMessage{
		FromAgent:     "mission-" + t.Mission,
		ToInbox:       mission.FleetInbox,
		MessageType:   "request",
		Title:         t.Title(),
		Payload:       payload,
		CorrelationID: t.CorrelationID(),
		Category:      mission.TicketCategory,
	}
	if err := store.InsertInboxMessage(msg); err != nil {
		return fmt.Errorf("ticket: file to %s: %w", mission.FleetInbox, err)
	}
	fmt.Fprintf(out, "filed %s → %s (id %s)\n", t.Title(), mission.FleetInbox, msg.ID)
	return nil
}

// openTickets reads unread ticket messages WITHOUT marking them read.
func openTickets(store ticketStore) ([]mission.TicketOccurrence, error) {
	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: mission.FleetInbox, UnreadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("ticket: list %s: %w", mission.FleetInbox, err)
	}
	var occ []mission.TicketOccurrence
	for _, m := range msgs {
		if m.Category != mission.TicketCategory {
			continue
		}
		t, perr := mission.ParseTicket(m.Payload)
		if perr != nil || t.Signature == "" {
			// Never drop an unreadable ticket silently: it still counts as open work, under a
			// signature that names the message so the fleet can look at it.
			t = mission.Ticket{Mission: "unknown", Signature: "unparseable:" + m.ID, Blocking: mission.BlockingNone,
				Evidence: "payload did not parse as a ticket: " + m.Title}
		}
		occ = append(occ, mission.TicketOccurrence{ID: m.ID, Ticket: t, CreatedAt: m.CreatedAt})
	}
	return occ, nil
}

func ticketOpen(args []string, store ticketStore, out io.Writer) error {
	fs := flag.NewFlagSet("mission ticket open", flag.ContinueOnError)
	count := fs.Bool("count", false, "Print only the number of open signatures")
	asJSON := fs.Bool("json", false, "Emit open signatures as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	occ, err := openTickets(store)
	if err != nil {
		return err
	}
	groups := mission.GroupOpen(occ)
	switch {
	case *count:
		fmt.Fprintln(out, len(groups))
	case *asJSON:
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(groups)
	default:
		if len(groups) == 0 {
			fmt.Fprintln(out, "no open harness tickets")
			return nil
		}
		for _, g := range groups {
			fmt.Fprintf(out, "%-3d %-6s %s  (%s; first %s)\n", g.SlotsLost, g.Blocking, g.Signature,
				strings.Join(g.Missions, ","), g.FirstSeen.Format("2006-01-02 15:04"))
		}
	}
	return nil
}

func ticketResolve(args []string, store ticketStore, out io.Writer) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: ailang mission ticket resolve <signature> --resolution TEXT [--sha SHA]")
	}
	sig := args[0]
	fs := flag.NewFlagSet("mission ticket resolve", flag.ContinueOnError)
	resolution := fs.String("resolution", "", "What was done (required)")
	sha := fs.String("sha", "", "Commit that carries the fix")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*resolution) == "" {
		return fmt.Errorf("ticket resolve: --resolution is required — the filing loop unparks on it")
	}
	occ, err := openTickets(store)
	if err != nil {
		return err
	}
	var hit *mission.OpenSignature
	groups := mission.GroupOpen(occ)
	for i := range groups {
		if groups[i].Signature == sig {
			hit = &groups[i]
		}
	}
	if hit == nil {
		return fmt.Errorf("ticket resolve: no open ticket with signature %q", sig)
	}
	body, _ := json.Marshal(map[string]any{"signature": sig, "resolution": *resolution, "sha": *sha, "slots_lost": hit.SlotsLost})
	missions := append([]string(nil), hit.Missions...)
	sort.Strings(missions)
	// Reply BEFORE marking read: a crash between the two leaves the ticket open (re-resolvable),
	// never closed with nobody told.
	for _, m := range missions {
		if m == "unknown" {
			continue
		}
		reply := &messaging.InboxMessage{
			FromAgent: "mission-" + mission.FleetMission, ToInbox: "mission-" + m,
			MessageType: "response", Category: mission.ResolvedCategory,
			Title:         fmt.Sprintf("[harness-resolved] %s · %s", sig, m),
			Payload:       string(body),
			CorrelationID: "harness:" + sig,
		}
		if err := store.InsertInboxMessage(reply); err != nil {
			return fmt.Errorf("ticket resolve: reply to mission-%s: %w (nothing marked read)", m, err)
		}
	}
	for _, id := range hit.IDs {
		if err := store.MarkInboxMessageRead(id); err != nil {
			return fmt.Errorf("ticket resolve: mark %s read: %w (replies already sent; re-run to finish)", id, err)
		}
	}
	fmt.Fprintf(out, "resolved %s: %d occurrence(s), replied to %s\n", sig, len(hit.IDs), strings.Join(missions, ","))
	return nil
}

func printMissionTicketHelp(w io.Writer) {
	fmt.Fprint(w, `ailang mission ticket — harness tickets (product loops file, the fleet loop resolves)

  ailang mission ticket file --iteration N --signature KEY --evidence TEXT|--evidence-file F
      [--mission NAME (default $MISSION_NAME)] [--blocking none|item|all]
      [--slot-verdict LINE] [--workaround TEXT] [--fire-started RFC3339]
                         file one occurrence to the mission-fleet inbox
  ailang mission ticket open [--count|--json]
                         open signatures ranked by slots lost; never marks read
  ailang mission ticket resolve KEY --resolution TEXT [--sha SHA]
                         reply to each filing mission, then mark every occurrence read
`)
}

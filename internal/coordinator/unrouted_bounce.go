package coordinator

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Telling a sender their message went nowhere.
//
// A send to an inbox no agent serves is accepted, stored, and skipped with a
// line in a log nobody reads. Measured over 2026-09-08..10: fifteen messages —
// daneel's AILANG asks to `ailang-core`, clients-planner's std/regex and
// std/datetime reports to `pkg:sunholo/ailang`, one to `pkg:sunholo/email_parse`
// which is one underscore away from the real `pkg:sunholo/email` — all filed,
// none dispatched, none noticed for two days. The plane was healthy the whole
// time; every one of those reached the coordinator within milliseconds.
//
// It is a DISCOVERY problem, not a routing one (Mark, 2026-09-10): nothing tells
// a sender which inbox names exist or what sending to one does. The bounce
// closes the half of that a machine can close — you find out immediately, from
// the component that actually knows the registry.
//
// WHY THE COORDINATOR AND NOT `messages send`: the sender does not have the
// authoritative registry. A laptop's ~/.ailang/config.yaml holds 2 agents while
// the cloud registry holds 35, so a send-side check would refuse correct sends
// and pass wrong ones. The coordinator IS the registry's reader.
//
// LOOP SAFETY — this is the part that has bitten before. One legitimate message
// once produced 591 self-addressed notifications and 59 job executions in 96
// minutes (2026-08-31 backstop sweep incident), so a component that generates
// messages in response to messages carries the burden of proof:
//
//   - a bounce is NEVER sent for a bounce (bounceMessageType, checked first);
//   - a bounce is never sent for a DECLARED triage inbox — those are filed for a
//     human on purpose, and saying so every time would be noise, not signal;
//   - the bounce goes to the SENDER, and is skipped when there is no sender to
//     answer, so it cannot fan out to a default inbox;
//   - each message can produce at most one bounce, because the cloud adapter's
//     ListUnread DRAINS its Pub/Sub buffer rather than re-reading Firestore, and
//     the backstop sweep already refuses unrouted inboxes
//     (backstop_sweep.go:148). Both paths were checked before this was written.

// bounceMessageType marks a bounce so the dispatcher can recognise its own
// output. Load-bearing: a bounce lands in the sender's inbox, that inbox may
// itself be unrouted, and without this marker the reply would bounce the bounce.
const bounceMessageType = "inbox_unrouted_notice"

// isUnroutedBounce reports whether a message is one of our own bounces.
func isUnroutedBounce(msg *Message) bool {
	if msg == nil {
		return false
	}
	if strings.EqualFold(msg.Kind, bounceMessageType) {
		return true
	}
	// Belt and braces for a store that did not round-trip the type: the title is
	// generated here and nowhere else.
	return strings.HasPrefix(msg.Title, bounceTitlePrefix)
}

const bounceTitlePrefix = "UNDELIVERED: "

// SuggestInbox returns the closest registered inbox to a mistyped one, or "".
//
// Exported so `ailang messages inboxes` offers the SAME suggestion the bounce
// does — two near-miss rules that disagree would send a confused sender two ways.
//
// Aimed at the typo case specifically — `pkg:sunholo/email_parse` for
// `pkg:sunholo/email`. The threshold is deliberately tight: a wrong suggestion
// sends someone to the wrong agent, which is worse than no suggestion, so a
// distant name gets the full directory instead.
func SuggestInbox(wanted string, known []string) string {
	wanted = strings.ToLower(strings.TrimSpace(wanted))
	if wanted == "" {
		return ""
	}
	best, bestDist := "", -1
	for _, k := range known {
		lk := strings.ToLower(k)
		if strings.ContainsRune(lk, '*') {
			continue // a pattern is not a name someone can be told to use
		}

		// A DECORATED name: the sender took a real inbox and appended to it —
		// `pkg:sunholo/email_parse` for `pkg:sunholo/email`. Edit distance alone
		// misses this (six insertions), and it is the case actually measured.
		//
		// The direction is load-bearing and NOT symmetric. Accept only when the
		// registered name is a prefix of what was typed. The reverse —
		// `pkg:sunholo/ailang` typed, `pkg:sunholo/ailang_parse` registered — is
		// a DIFFERENT product, not a typo, and suggesting it would route a
		// language report to the document parser. That exact pair was sent on
		// 2026-09-09, so the wrong answer here has a name.
		if len(wanted) > len(lk) && strings.HasPrefix(wanted, lk) {
			if sep := wanted[len(lk)]; sep == '_' || sep == '-' || sep == '/' || sep == '.' {
				if bestDist < 0 || 0 < bestDist {
					best, bestDist = k, 0
				}
				continue
			}
		}

		d := levenshtein(wanted, lk)
		// Otherwise accept only genuinely close names: at most a quarter of the
		// length, and never more than 4 edits.
		limit := len(wanted) / 4
		if limit > 4 {
			limit = 4
		}
		if limit < 1 {
			limit = 1
		}
		if d <= limit && (bestDist < 0 || d < bestDist) {
			best, bestDist = k, d
		}
	}
	return best
}

// levenshtein is the standard edit distance, two rows.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// bounceBody composes the notice. It has to answer three things the sender
// cannot see: that nothing happened, why, and what to do instead.
func bounceBody(original *Message, inbox, suggestion string, routable []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your message was FILED BUT NOT DISPATCHED.\n\n")
	fmt.Fprintf(&b, "  inbox:   %q — no agent serves this, and it is not declared human-triage\n", inbox)
	fmt.Fprintf(&b, "  title:   %s\n", original.Title)
	fmt.Fprintf(&b, "  id:      %s\n\n", original.ID)

	if suggestion != "" {
		fmt.Fprintf(&b, "Did you mean %q? It is one of the registered inboxes and is a close\n", suggestion)
		fmt.Fprintf(&b, "match for what you sent to.\n\n")
	}

	b.WriteString("The message is stored and readable, but no task was created and no agent\n")
	b.WriteString("will act on it. Resend to a registered inbox, or ask the operator to\n")
	b.WriteString("register this one.\n\n")

	b.WriteString("See what each inbox does:  ailang messages inboxes\n")

	if len(routable) > 0 {
		b.WriteString("\nInboxes that dispatch to an agent right now:\n")
		for i, r := range routable {
			if i == 12 {
				fmt.Fprintf(&b, "  … and %d more (ailang messages inboxes)\n", len(routable)-12)
				break
			}
			fmt.Fprintf(&b, "  %s\n", r)
		}
	}
	return b.String()
}

// bounceUnroutedMessage tells the sender their message went nowhere.
//
// Returns whether a bounce was actually sent, so the caller can log the
// difference between "told them" and "could not".
func (d *Daemon) bounceUnroutedMessage(msg *Message, inbox string) bool {
	if msg == nil || d.msgStore == nil || d.agentRegistry == nil {
		return false
	}

	// Guard 1: never answer our own bounce.
	if isUnroutedBounce(msg) {
		return false
	}

	// Guard 2: a declared triage inbox is doing exactly what it should.
	if d.agentRegistry.IsTriageOnly(inbox) {
		return false
	}

	// Guard 3: no sender, nobody to tell. Never fall back to a default inbox —
	// that is how a notice becomes work for whoever owns the default.
	sender := strings.TrimSpace(msg.From)
	if sender == "" {
		return false
	}

	known := d.agentRegistry.ListInboxes()
	routable := make([]string, 0, len(known))
	for _, k := range known {
		if !strings.ContainsRune(k, '*') {
			routable = append(routable, k)
		}
	}

	notice := &messaging.InboxMessage{
		FromAgent:   "coordinator",
		ToInbox:     sender,
		MessageType: bounceMessageType,
		Title:       bounceTitlePrefix + msg.Title,
		Payload:     bounceBody(msg, inbox, SuggestInbox(inbox, known), routable),
		Status:      "unread",
	}
	if err := d.msgStore.InsertInboxMessage(notice); err != nil {
		d.logger.Printf("could not notify %s that %q is unrouted: %v", sender, inbox, err)
		return false
	}
	return true
}

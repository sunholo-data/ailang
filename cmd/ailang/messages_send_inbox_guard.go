package main

// Reject an unknown inbox AT SEND TIME.
//
// A send to an inbox nothing serves succeeds. The row is written, the CLI prints
// a green tick, a bounce notice is filed somewhere nobody reads, and the work
// never happens. `messages inboxes` has been able to say "did you mean
// daneel-design?" for weeks — but only AFTERWARDS, to whoever thought to look.
// The suggestion existed at the wrong end of the pipe.
//
// Measured 2026-09-13: five design requests went to `daneel-design-ailang`
// instead of `daneel-design`. All five were filed, all five bounced, and each
// bounce was itself addressed to `ailang`, which is also unregistered — a bounce
// that bounced. Nothing was lost only because a human noticed twenty minutes
// later and resent them by hand.
//
// THE FALLBACK IS THE DANGEROUS PART, so it is explicit here. resolveInboxRegistry
// falls back to this machine's ~/.ailang/config.yaml when GCS is unreadable, and
// that file may hold two agents. Refusing a send against a two-agent registry
// would block every legitimate message on any laptop without credentials — a
// guard that fails closed on its own blind spot. So: refuse only when we are
// judging against the registry the PLANE dispatches from, and otherwise warn and
// send. The check reports which of the two happened.

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/storage"
)

// inboxGuardVerdict is the decision, separated from the I/O so it is testable.
type inboxGuardVerdict struct {
	Allow   bool
	Message string
}

// checkSendInbox decides whether a send to inbox may proceed.
//
// authoritative says the registry came from the plane being written to. When it
// did not, an unknown inbox is a fact about OUR VIEW, not about the plane.
func checkSendInbox(reg *coordinator.AgentRegistry, inbox string, authoritative, force bool) inboxGuardVerdict {
	if reg == nil {
		return inboxGuardVerdict{Allow: true}
	}
	if reg.GetAgentForInbox(inbox) != nil || reg.IsTriageOnly(inbox) {
		return inboxGuardVerdict{Allow: true}
	}

	detail := fmt.Sprintf("no agent serves %q and it is not declared human-triage", inbox)
	if s := coordinator.SuggestInbox(inbox, reg.ListInboxes()); s != "" {
		detail += fmt.Sprintf(" — did you mean %q?", s)
	}

	if !authoritative {
		return inboxGuardVerdict{
			Allow: true,
			Message: detail + "\n  (judged against THIS MACHINE's config, which is not the plane's — sending anyway;\n" +
				"   run `ailang messages inboxes` where GCS is readable to check properly)",
		}
	}
	return inboxGuardVerdict{
		Allow: force,
		Message: detail + "\n  A send here is FILED, NOT DISPATCHED: it is stored, it bounces, and no agent\n" +
			"  ever acts on it.\n" +
			"    ailang messages inboxes         what each inbox does\n" +
			"    ailang coordinator agents       the agents on the live plane\n" +
			"  Use --force to send anyway (e.g. a deliberate probe).",
	}
}

// guardSendInbox applies checkSendInbox and exits 1 on refusal.
func guardSendInbox(inbox string, force bool) {
	reg, source, err := resolveInboxRegistry("")
	if err != nil {
		// No registry at all: warn, do not block. The send path predates this
		// check and must not become dependent on it.
		fmt.Fprintf(os.Stderr, "%s could not load a registry to check the inbox (%v) — sending unchecked\n", yellow("!"), err)
		return
	}
	// Authoritative only when the registry came from the same plane we write to.
	mode, _ := messagesTarget()
	authoritative := mode != storage.ModeGCP || strings.HasPrefix(source, "gs://")

	v := checkSendInbox(reg, inbox, authoritative, force)
	if v.Message == "" {
		return
	}
	if v.Allow {
		fmt.Fprintf(os.Stderr, "%s %s\n", yellow("!"), v.Message)
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s %s\n\n", red("Refusing to send:"), v.Message)
	os.Exit(1)
}

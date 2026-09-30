package main

import (
	"fmt"
	"os"
	"time"
)

// messagesOneShotDeadline bounds the one-shot `ailang messages` subcommands.
//
// The gcp message store runs every Firestore call on context.Background(), so with
// the network to firestore.googleapis.com blocked the client retries forever and the
// command never returns. Measured 2026-09-29: inside the pi mission sandbox (whose
// allowlist has no Google endpoints) `ailang messages list --unread` hung until the
// runner killed it at 600s — on all three World iter-208 executor runs and three
// more in iter-207 — and the runner banked each one as the MODEL's `stream_dead`.
// Reproduced attended with HTTPS_PROXY pointed at a closed port: still running at
// 45s, against 1s for the unproxied control.
//
// The bound lives here and not in the store because the same store serves the
// coordinator and dashboard, whose aggregate scans are legitimately long.
const messagesOneShotDeadline = 90 * time.Second

// oneShotDeadline returns the wall-clock bound for subCmd, or 0 for a subcommand
// that is long-running by design (watch, the interactive browser) or does bulk
// work whose duration scales with the store (cleanup, dedupe, GitHub sync).
func oneShotDeadline(subCmd string) time.Duration {
	switch subCmd {
	case "list", "ls", "search", "read", "ack", "unack", "send", "reply",
		"forward", "fwd", "inboxes", "renotify":
		return messagesOneShotDeadline
	}
	return 0
}

// armMessagesDeadline exits the process with a diagnosis if subCmd outlives its
// bound. Exit 124 is timeout(1)'s convention, so a caller can tell "the store did
// not answer" from an ordinary failure (1) or a configuration error (2).
func armMessagesDeadline(subCmd, store string) {
	d := oneShotDeadline(subCmd)
	if d <= 0 {
		return
	}
	time.AfterFunc(d, func() {
		fmt.Fprintf(os.Stderr, "%s `ailang messages %s` got no answer from the %s message store in %s. "+
			"Is its network blocked (a sandbox, a proxy, no connectivity)? "+
			"Set AILANG_STORAGE_MESSAGING=local to read this machine's store instead.\n",
			red("Error:"), subCmd, store, d)
		os.Exit(124)
	})
}

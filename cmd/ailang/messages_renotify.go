package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// `ailang messages renotify <id>` — publish the Pub/Sub notification for a
// message that is ALREADY in the store.
//
// A send publishes its notification only when the SENDER's config carries a
// pubsub section. A sender without one (a LaunchAgent, another user account)
// writes the message and nothing ever dispatches it — it sits unread on a
// DISPATCHES inbox for good ("filed, not dispatched"). Measured 2026-09-17:
// Daneel's first live `daneel ask` (inbox_1789619843359_e0f06d06) sat four
// hours on daneel-executor. Re-sending is not a fix: the sender's own records
// (and every completion title) key on the original message id.
//
// Refuses a message that is not unread — a read message was dispatched or
// acked, and a second notification would run it twice.
func runMessagesRenotify(args []string) {
	fs := flag.NewFlagSet("messages renotify", flag.ExitOnError)
	force := fs.Bool("force", false, "Notify even if the message is not unread (it may run twice)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang messages renotify <message-id>")
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	msg, err := store.GetInboxMessage(msgID)
	if err != nil || msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message %s not found (%v)\n", red("Error"), msgID, err)
		os.Exit(1)
	}
	if err := renotifyGuard(msg.Status, *force); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s %v\n", red("Error"), msgID, err)
		os.Exit(1)
	}

	cfg, err := messaging.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: messaging config unreadable: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if cfg == nil || cfg.PubSub == nil || !cfg.PubSub.Enabled {
		fmt.Fprintf(os.Stderr, "%s: this machine's config has no enabled pubsub section — nothing can be published from here\n", red("Error"))
		os.Exit(1)
	}
	notifier, err := messaging.NewPubSubNotifier(notifyConfigForStore(cfg.PubSub))
	if err != nil || notifier == nil {
		fmt.Fprintf(os.Stderr, "%s: Pub/Sub notifier: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer notifier.Close()
	if err := notifier.Notify(context.Background(), msg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: Pub/Sub notify failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Pub/Sub notification published for %s → %s (%q)\n", green("✓"), msgID, msg.ToInbox, msg.Title)
}

// renotifyGuard: only an unread message may be re-notified without --force. A
// read message was dispatched or acked; a second notification runs it twice.
func renotifyGuard(status string, force bool) error {
	if status == messaging.InboxStatusUnread || force {
		return nil
	}
	return fmt.Errorf("is %s, not unread — it was dispatched or acked; --force notifies anyway (it may run twice)", status)
}

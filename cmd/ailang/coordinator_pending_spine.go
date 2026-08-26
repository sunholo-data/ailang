package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// printApprovalsInboxPending lists unread approval_request messages from the
// `approvals` inbox on the CANONICAL message store (honors
// AILANG_MESSAGES_STORE), with provenance per row. Best-effort: a store error
// is printed, never silently dropped — an unreachable spine must not read as
// an empty one.
func printApprovalsInboxPending() {
	msgStore, err := openStore()
	if err != nil {
		fmt.Println(yellow("⚠"), "approvals inbox unavailable:", err)
		return
	}
	defer func() { _ = msgStore.Close() }()

	msgs, err := msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "approvals", UnreadOnly: true, Limit: 50})
	if err != nil {
		fmt.Println(yellow("⚠"), "approvals inbox unavailable:", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	fmt.Println()
	fmt.Println(cyan("Decision spine — unread in the `approvals` inbox:"))
	for _, m := range msgs {
		fmt.Printf("  ● %-22s %s  (from %s, %s)\n", m.ID, m.Title, m.FromAgent, m.CreatedAt.Format("01-02 15:04"))
	}
	fmt.Println("  Resolve via dashboard, `ailang coordinator approve <id>`, or ack the message when handled elsewhere.")
}

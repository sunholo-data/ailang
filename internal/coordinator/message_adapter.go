package coordinator

import (
	"github.com/sunholo/ailang/internal/messaging"
)

// InboxMessageAdapter adapts messaging.Store (inbox_messages table) to the coordinator's MessageStore interface
type InboxMessageAdapter struct {
	store *messaging.Store
	inbox string // Target inbox (e.g., "user", "coordinator")
}

// NewInboxMessageAdapter creates a new adapter that watches a specific inbox
func NewInboxMessageAdapter(store *messaging.Store, inbox string) *InboxMessageAdapter {
	if inbox == "" {
		inbox = "user"
	}
	return &InboxMessageAdapter{
		store: store,
		inbox: inbox,
	}
}

// ListUnread returns unread messages for the configured inbox
func (a *InboxMessageAdapter) ListUnread() ([]*Message, error) {
	// Use the inbox_messages table via ListInboxMessages
	msgs, err := a.store.ListInboxMessages(messaging.InboxListOptions{
		Inbox:      a.inbox,
		UnreadOnly: true,
		Collapsed:  true, // Hide duplicates
		Limit:      100,
	})
	if err != nil {
		return nil, err
	}

	// Convert to coordinator Message type
	result := make([]*Message, 0, len(msgs))
	for _, m := range msgs {
		// Extract GitHub issue number if linked
		githubIssue := 0
		if m.GitHubIssue != nil {
			githubIssue = *m.GitHubIssue
		}
		result = append(result, &Message{
			ID:           m.ID,
			From:         m.FromAgent,
			Title:        m.Title,
			Content:      m.Payload,
			Type:         m.Category,     // bug, feature, general
			Kind:         m.MessageType,  // directive, question
			Priority:     "",             // Will be classified by analyzer
			GithubIssue:  githubIssue,    // M-COORD-GITHUB-AUTO-ROUTING
			ParentTaskID: m.ParentTaskID, // M-TASK-HIERARCHY: propagate from inbox message
			Iteration:    m.Iteration,    // M-TASK-HIERARCHY: propagate iteration for feedback loops
			CreatedAt:    m.CreatedAt,
		})
	}

	return result, nil
}

// MarkAsRead marks a message as read
func (a *InboxMessageAdapter) MarkAsRead(id string) error {
	return a.store.MarkInboxMessageRead(id)
}

// Compile-time check that InboxMessageAdapter implements MessageStore
var _ MessageStore = (*InboxMessageAdapter)(nil)

// OpenDefaultInboxAdapter opens the default collaboration database and creates an adapter
func OpenDefaultInboxAdapter(targetInbox string) (*InboxMessageAdapter, *messaging.Store, error) {
	dbPath := messaging.GetDefaultDatabasePath()
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return nil, nil, err
	}

	if targetInbox == "" {
		targetInbox = "user"
	}

	adapter := NewInboxMessageAdapter(store, targetInbox)
	return adapter, store, nil
}

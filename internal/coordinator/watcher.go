package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Message represents a message from the messaging system
type Message struct {
	ID           string
	From         string
	Title        string
	Content      string
	Inbox        string // Target inbox (set by PubSubInboxAdapter from message attributes, M-CLOUD-E2E)
	Type         string // bug, feature, task, etc. (category)
	Kind         string // directive, question (message type)
	Source       string // Pub/Sub topic the message arrived on: "cascade" = authoritative bump (M-PKG-AUTONOMOUS-CASCADE-SAFE M1), anything else = treat as public-routed
	Priority     string // high, medium, low
	GithubIssue  int    // Linked GitHub issue number (M-COORD-GITHUB-AUTO-ROUTING)
	GithubRepo   string // GitHub repo (owner/repo) for issue operations (M-COORD-GITHUB-CLOSE-ON-MERGE)
	ParentTaskID string // Parent task ID for hierarchy tracking (M-TASK-HIERARCHY)
	Iteration    int    // Iteration number for feedback loops (M-TASK-HIERARCHY)
	ChainID      string // ExecutionChain ID for unified hierarchy (M-CHAINS-SIMPLIFY)
	CreatedAt    time.Time

	// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope fields, populated
	// from the Pub/Sub data field when Source=="cascade". Empty strings/false
	// when the message wasn't a cascade or the publisher used the legacy
	// pre-envelope cascade publish.
	RootPackage       string // vendor/name@version
	RootChangeClass   string // "A" content-only, "B" additive, "C" interface change
	FromVersion       string
	ToVersion         string
	FromInterfaceHash string
	ToInterfaceHash   string
	FromContentHash   string
	ToContentHash     string
	EffectsWidened    bool
	PrevEffectCeiling []string
	NewEffectCeiling  []string
}

// MessageStore is the interface for accessing messages
type MessageStore interface {
	ListUnread() ([]*Message, error)
	MarkAsRead(id string) error
}

// MessageWatcher polls for unread messages and emits tasks
type MessageWatcher struct {
	store        MessageStore
	pollInterval time.Duration
	seenMsgIDs   map[string]bool
	tasksChan    chan *Task
	mu           sync.RWMutex
}

// NewMessageWatcher creates a new message watcher
func NewMessageWatcher(store MessageStore, pollInterval time.Duration) *MessageWatcher {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}

	return &MessageWatcher{
		store:        store,
		pollInterval: pollInterval,
		seenMsgIDs:   make(map[string]bool),
		tasksChan:    make(chan *Task, 100),
	}
}

// Start begins watching for messages
func (w *MessageWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Initial poll
	if err := w.poll(); err != nil {
		return fmt.Errorf("initial poll failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			close(w.tasksChan)
			return nil
		case <-ticker.C:
			if err := w.poll(); err != nil {
				// Log but don't stop
				fmt.Printf("poll error: %v\n", err)
			}
		}
	}
}

// Tasks returns the channel of discovered tasks
func (w *MessageWatcher) Tasks() <-chan *Task {
	return w.tasksChan
}

// poll checks for new unread messages
func (w *MessageWatcher) poll() error {
	messages, err := w.store.ListUnread()
	if err != nil {
		return fmt.Errorf("failed to list unread messages: %w", err)
	}

	for _, msg := range messages {
		if w.hasSeenMessage(msg.ID) {
			continue
		}

		// Convert message to task
		task := w.messageToTask(msg)

		// Send to channel (non-blocking)
		select {
		case w.tasksChan <- task:
			w.markSeen(msg.ID)
		default:
			// Channel full, will retry next poll
		}
	}

	return nil
}

// messageToTask converts a message to a task
func (w *MessageWatcher) messageToTask(msg *Message) *Task {
	priority := classifyPriority(msg)

	// Use the message's kind directly if set
	// Otherwise, infer from category type
	kind := msg.Kind
	if kind == "" {
		// Fallback: infer kind from category type
		// "question" or "research" categories get read-only mode
		if msg.Type == "question" || msg.Type == "research" {
			kind = "question"
		} else {
			kind = "directive"
		}
	}

	return &Task{
		ID:           fmt.Sprintf("task-%s", msg.ID),
		Title:        extractTitle(msg),
		Content:      msg.Content,
		Kind:         kind,
		Priority:     priority,
		MessageID:    msg.ID,
		ParentTaskID: msg.ParentTaskID, // M-TASK-HIERARCHY: propagate from message
		CreatedAt:    msg.CreatedAt,
	}
}

// hasSeenMessage checks if a message has already been processed
func (w *MessageWatcher) hasSeenMessage(id string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.seenMsgIDs[id]
}

// markSeen marks a message as seen
func (w *MessageWatcher) markSeen(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seenMsgIDs[id] = true
}

// ClearSeen clears the seen messages (useful for testing)
func (w *MessageWatcher) ClearSeen() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seenMsgIDs = make(map[string]bool)
}

// SeenCount returns the number of seen messages
func (w *MessageWatcher) SeenCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.seenMsgIDs)
}

// classifyPriority determines task priority from message metadata
func classifyPriority(msg *Message) int {
	// Check explicit priority
	switch msg.Priority {
	case "high", "urgent", "critical":
		return 1
	case "medium", "normal":
		return 5
	case "low":
		return 10
	}

	// Check type
	switch msg.Type {
	case "bug", "error", "crash":
		return 2
	case "feature", "enhancement":
		return 5
	case "docs", "documentation":
		return 8
	case "research", "question":
		return 9
	}

	// Default medium priority
	return 5
}

// extractTitle extracts a title from the message
func extractTitle(msg *Message) string {
	if msg.Title != "" {
		return msg.Title
	}

	// Use first line of content as title
	content := msg.Content
	for i, c := range content {
		if c == '\n' {
			return content[:i]
		}
	}

	// Truncate if too long
	if len(content) > 100 {
		return content[:100] + "..."
	}

	return content
}

// MockMessageStore is a mock implementation for testing
type MockMessageStore struct {
	messages []*Message
	readIDs  map[string]bool
	mu       sync.RWMutex
}

// NewMockMessageStore creates a new mock message store
func NewMockMessageStore() *MockMessageStore {
	return &MockMessageStore{
		messages: make([]*Message, 0),
		readIDs:  make(map[string]bool),
	}
}

// AddMessage adds a message to the mock store
func (m *MockMessageStore) AddMessage(msg *Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// ListUnread returns unread messages
func (m *MockMessageStore) ListUnread() ([]*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unread := make([]*Message, 0)
	for _, msg := range m.messages {
		if !m.readIDs[msg.ID] {
			unread = append(unread, msg)
		}
	}
	return unread, nil
}

// MarkAsRead marks a message as read
func (m *MockMessageStore) MarkAsRead(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readIDs[id] = true
	return nil
}

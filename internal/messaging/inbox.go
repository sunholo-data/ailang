package messaging

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/builtins"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InboxMessage represents a message in the unified inbox system
type InboxMessage struct {
	ID                 string     `json:"id"`
	MessageID          string     `json:"message_id"`
	CorrelationID      string     `json:"correlation_id,omitempty"`
	FromAgent          string     `json:"from_agent"`
	ToInbox            string     `json:"to_inbox"`
	MessageType        string     `json:"message_type"`
	Title              string     `json:"title"`
	Payload            string     `json:"payload,omitempty"`
	Category           string     `json:"category,omitempty"`             // bug, feature, general (for GitHub sync)
	GitHubIssue        *int       `json:"github_issue,omitempty"`         // GitHub issue number
	GitHubRepo         string     `json:"github_repo,omitempty"`          // GitHub repo (owner/repo)
	Simhash            *int64     `json:"simhash,omitempty"`              // SimHash for semantic search (v1.2.0)
	DupOf              string     `json:"dup_of,omitempty"`               // ID of message this is a duplicate of (v1.2.0)
	Embedding          string     `json:"embedding,omitempty"`            // JSON-encoded float32 array (v1.3.0)
	EmbeddingModel     string     `json:"embedding_model,omitempty"`      // e.g., "ollama:nomic-embed-text" (v1.3.0)
	EmbeddingUpdatedAt *int64     `json:"embedding_updated_at,omitempty"` // Unix millis (v1.3.0)
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	ReadAt             *time.Time `json:"read_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// Inbox message statuses
const (
	InboxStatusUnread   = "unread"
	InboxStatusRead     = "read"
	InboxStatusArchived = "archived"
	InboxStatusDeleted  = "deleted"
)

// Inbox message types
const (
	InboxTypeNotification = "notification"
	InboxTypeRequest      = "request"
	InboxTypeResponse     = "response"
)

// Message categories (for GitHub sync and coordinator routing)
const (
	CategoryBug      = "bug"
	CategoryFeature  = "feature"
	CategoryGeneral  = "general"
	CategoryDocs     = "docs"
	CategoryResearch = "research"
	CategoryRefactor = "refactor"
	CategoryTest     = "test"
)

// InboxListOptions specifies filters for listing inbox messages
type InboxListOptions struct {
	Inbox       string // Filter by inbox name (empty = all)
	Status      string // Filter by status (empty = all)
	UnreadOnly  bool   // Only unread messages
	FromAgent   string // Filter by sender
	Limit       int    // Max results (0 = default 50)
	IncludeRead bool   // Include read messages (default: true unless UnreadOnly)
	Collapsed   bool   // Hide messages where dup_of IS NOT NULL (semantic dedup)
	DupOf       string // Only messages that are duplicates of this ID
}

// InsertInboxMessage adds a new message to the inbox
func (s *Store) InsertInboxMessage(msg *InboxMessage) error {
	// Start span for message send operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.send",
		trace.WithAttributes(
			attribute.String("message.to_inbox", msg.ToInbox),
			attribute.String("message.from_agent", msg.FromAgent),
			attribute.String("message.type", msg.MessageType),
			attribute.String("message.category", msg.Category),
		),
	)
	defer span.End()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.MessageID == "" {
		msg.MessageID = fmt.Sprintf("msg_%s_%s", time.Now().Format("20060102_150405"), msg.ID[:8])
	}
	if msg.Status == "" {
		msg.Status = InboxStatusUnread
	}
	if msg.MessageType == "" {
		msg.MessageType = InboxTypeNotification
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	var readAt, expiresAt *string
	if msg.ReadAt != nil {
		t := msg.ReadAt.Format(time.RFC3339)
		readAt = &t
	}
	if msg.ExpiresAt != nil {
		t := msg.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}

	// Handle nullable category and github fields
	var category, githubRepo *string
	if msg.Category != "" {
		category = &msg.Category
	}
	if msg.GitHubRepo != "" {
		githubRepo = &msg.GitHubRepo
	}

	// Compute simhash from title + payload for semantic search
	var simhash *int64
	if msg.Simhash != nil {
		simhash = msg.Simhash
	} else {
		searchText := msg.Title
		if msg.Payload != "" {
			searchText += " " + msg.Payload
		}
		hash := builtins.SimHash(searchText)
		simhash = &hash
		msg.Simhash = simhash // Store back in msg for caller
	}

	// Handle nullable dup_of field
	var dupOf *string
	if msg.DupOf != "" {
		dupOf = &msg.DupOf
	}

	_, err := s.db.Exec(`
		INSERT INTO inbox_messages (id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.MessageID, msg.CorrelationID, msg.FromAgent, msg.ToInbox, msg.MessageType, msg.Title, msg.Payload, category, msg.GitHubIssue, githubRepo, simhash, dupOf, msg.Status, msg.CreatedAt.Format(time.RFC3339), readAt, expiresAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to insert message")
	} else {
		span.SetAttributes(attribute.String("message.id", msg.ID))
		span.SetStatus(codes.Ok, "message sent")
	}

	return err
}

// ListInboxMessages returns messages matching the given options
func (s *Store) ListInboxMessages(opts InboxListOptions) ([]InboxMessage, error) {
	// Start span for message list operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.list",
		trace.WithAttributes(
			attribute.String("list.inbox", opts.Inbox),
			attribute.Bool("list.unread_only", opts.UnreadOnly),
			attribute.Bool("list.collapsed", opts.Collapsed),
			attribute.Int("list.limit", opts.Limit),
		),
	)
	defer span.End()

	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at FROM inbox_messages WHERE 1=1`
	args := []interface{}{}

	if opts.Inbox != "" {
		query += " AND to_inbox = ?"
		args = append(args, opts.Inbox)
	}

	if opts.UnreadOnly {
		query += " AND status = ?"
		args = append(args, InboxStatusUnread)
	} else if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	} else if !opts.IncludeRead {
		// Default: exclude deleted/archived unless specifically requested
		query += " AND status NOT IN (?, ?)"
		args = append(args, InboxStatusDeleted, InboxStatusArchived)
	}

	if opts.FromAgent != "" {
		query += " AND from_agent = ?"
		args = append(args, opts.FromAgent)
	}

	// Collapsed mode: hide duplicates (messages with dup_of set)
	if opts.Collapsed {
		query += " AND (dup_of IS NULL OR dup_of = '')"
	}

	// Filter for duplicates of a specific message
	if opts.DupOf != "" {
		query += " AND dup_of = ?"
		args = append(args, opts.DupOf)
	}

	query += " ORDER BY created_at DESC"

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query messages")
		return nil, err
	}
	defer rows.Close()

	var messages []InboxMessage
	for rows.Next() {
		var msg InboxMessage
		var correlationID, payload, category, githubRepo, dupOf sql.NullString
		var githubIssue, simhash sql.NullInt64
		var readAt, expiresAt sql.NullString
		var createdAt string

		err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &simhash, &dupOf, &msg.Status, &createdAt, &readAt, &expiresAt)
		if err != nil {
			return nil, err
		}

		msg.CorrelationID = correlationID.String
		msg.Payload = payload.String
		msg.Category = category.String
		msg.GitHubRepo = githubRepo.String
		msg.DupOf = dupOf.String
		if githubIssue.Valid {
			issueNum := int(githubIssue.Int64)
			msg.GitHubIssue = &issueNum
		}
		if simhash.Valid {
			hash := simhash.Int64
			msg.Simhash = &hash
		}

		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			msg.CreatedAt = t
		}
		if readAt.Valid {
			if t, err := time.Parse(time.RFC3339, readAt.String); err == nil {
				msg.ReadAt = &t
			}
		}
		if expiresAt.Valid {
			if t, err := time.Parse(time.RFC3339, expiresAt.String); err == nil {
				msg.ExpiresAt = &t
			}
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error iterating messages")
		return messages, err
	}

	span.SetAttributes(attribute.Int("list.result_count", len(messages)))
	span.SetStatus(codes.Ok, "messages listed")
	return messages, nil
}

// GetInboxMessage returns a single message by ID (UUID or message_id)
func (s *Store) GetInboxMessage(id string) (*InboxMessage, error) {
	// Start span for message read operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.read",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	row := s.db.QueryRow(`
		SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE id = ? OR message_id = ?
	`, id, id)

	var msg InboxMessage
	var correlationID, payload, category, githubRepo, dupOf sql.NullString
	var githubIssue, simhash sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string

	err := row.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &simhash, &dupOf, &msg.Status, &createdAt, &readAt, &expiresAt)
	if err == sql.ErrNoRows {
		span.SetStatus(codes.Ok, "message not found")
		return nil, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read message")
		return nil, err
	}

	msg.CorrelationID = correlationID.String
	msg.Payload = payload.String
	msg.Category = category.String
	msg.GitHubRepo = githubRepo.String
	msg.DupOf = dupOf.String
	if githubIssue.Valid {
		issueNum := int(githubIssue.Int64)
		msg.GitHubIssue = &issueNum
	}
	if simhash.Valid {
		hash := simhash.Int64
		msg.Simhash = &hash
	}

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		msg.CreatedAt = t
	}
	if readAt.Valid {
		if t, err := time.Parse(time.RFC3339, readAt.String); err == nil {
			msg.ReadAt = &t
		}
	}
	if expiresAt.Valid {
		if t, err := time.Parse(time.RFC3339, expiresAt.String); err == nil {
			msg.ExpiresAt = &t
		}
	}

	span.SetAttributes(
		attribute.String("message.from_agent", msg.FromAgent),
		attribute.String("message.to_inbox", msg.ToInbox),
		attribute.String("message.type", msg.MessageType),
	)
	span.SetStatus(codes.Ok, "message read")
	return &msg, nil
}

// MarkInboxMessageRead marks a message as read
func (s *Store) MarkInboxMessageRead(id string) error {
	// Start span for message ack operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.ack",
		trace.WithAttributes(
			attribute.String("message.id", id),
			attribute.String("message.new_status", InboxStatusRead),
		),
	)
	defer span.End()

	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE inbox_messages SET status = ?, read_at = ? WHERE (id = ? OR message_id = ?) AND status = ?
	`, InboxStatusRead, now, id, id, InboxStatusUnread)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		err := fmt.Errorf("message not found or already read")
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int64("messages.affected", affected))
	span.SetStatus(codes.Ok, "message acknowledged")
	return nil
}

// MarkInboxMessageUnread marks a message as unread
func (s *Store) MarkInboxMessageUnread(id string) error {
	// Start span for message unack operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.unack",
		trace.WithAttributes(
			attribute.String("message.id", id),
			attribute.String("message.new_status", InboxStatusUnread),
		),
	)
	defer span.End()

	result, err := s.db.Exec(`
		UPDATE inbox_messages SET status = ?, read_at = NULL WHERE id = ? OR message_id = ?
	`, InboxStatusUnread, id, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		err := fmt.Errorf("message not found")
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int64("messages.affected", affected))
	span.SetStatus(codes.Ok, "message unacknowledged")
	return nil
}

// MarkAllInboxMessagesRead marks all messages in an inbox as read
func (s *Store) MarkAllInboxMessagesRead(inbox string) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	var result sql.Result
	var err error

	if inbox == "" {
		result, err = s.db.Exec(`
			UPDATE inbox_messages SET status = ?, read_at = ? WHERE status = ?
		`, InboxStatusRead, now, InboxStatusUnread)
	} else {
		result, err = s.db.Exec(`
			UPDATE inbox_messages SET status = ?, read_at = ? WHERE to_inbox = ? AND status = ?
		`, InboxStatusRead, now, inbox, InboxStatusUnread)
	}

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ForwardInboxMessage moves a message to a different inbox
func (s *Store) ForwardInboxMessage(id string, toInbox string) error {
	result, err := s.db.Exec(`
		UPDATE inbox_messages SET to_inbox = ? WHERE id = ? OR message_id = ?
	`, toInbox, id, id)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

// InboxMessageExistsByGitHub checks if a message with the given GitHub issue already exists
func (s *Store) InboxMessageExistsByGitHub(repo string, issueNumber int) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM inbox_messages
		WHERE github_repo = ? AND github_issue_number = ?
	`, repo, issueNumber).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateInboxMessageGitHub updates the GitHub issue number and repo for a message
func (s *Store) UpdateInboxMessageGitHub(messageID string, issueNumber int, repo string) error {
	var repoPtr *string
	if repo != "" {
		repoPtr = &repo
	}

	result, err := s.db.Exec(`
		UPDATE inbox_messages
		SET github_issue_number = ?, github_repo = COALESCE(?, github_repo)
		WHERE message_id = ?
	`, issueNumber, repoPtr, messageID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("message not found: %s", messageID)
	}
	return nil
}

// CleanupInboxMessages removes old messages
func (s *Store) CleanupInboxMessages(olderThan time.Duration, expiredOnly bool) (int64, error) {
	// Start span for message cleanup operation
	ctx := context.Background()
	_, span := messagingTracer.Start(ctx, "messages.cleanup",
		trace.WithAttributes(
			attribute.Bool("cleanup.expired_only", expiredOnly),
			attribute.Int64("cleanup.older_than_ms", olderThan.Milliseconds()),
		),
	)
	defer span.End()

	var result sql.Result
	var err error

	if expiredOnly {
		now := time.Now().Format(time.RFC3339)
		result, err = s.db.Exec(`
			DELETE FROM inbox_messages WHERE expires_at IS NOT NULL AND expires_at < ?
		`, now)
	} else {
		cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
		result, err = s.db.Exec(`
			DELETE FROM inbox_messages WHERE created_at < ?
		`, cutoff)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	affected, _ := result.RowsAffected()
	span.SetAttributes(attribute.Int64("cleanup.deleted_count", affected))
	span.SetStatus(codes.Ok, "cleanup complete")
	return affected, nil
}

// CountInboxMessagesByStatus returns counts of messages by status
func (s *Store) CountInboxMessagesByStatus(inbox string) (map[string]int64, error) {
	query := `SELECT status, COUNT(*) as count FROM inbox_messages`
	args := []interface{}{}

	if inbox != "" {
		query += " WHERE to_inbox = ?"
		args = append(args, inbox)
	}
	query += " GROUP BY status"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

package messaging

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InboxMessage represents a message in the unified inbox system
type InboxMessage struct {
	ID            string     `json:"id"`
	MessageID     string     `json:"message_id"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	FromAgent     string     `json:"from_agent"`
	ToInbox       string     `json:"to_inbox"`
	MessageType   string     `json:"message_type"`
	Title         string     `json:"title"`
	Payload       string     `json:"payload,omitempty"`
	Category      string     `json:"category,omitempty"`     // bug, feature, general (for GitHub sync)
	GitHubIssue   *int       `json:"github_issue,omitempty"` // GitHub issue number
	GitHubRepo    string     `json:"github_repo,omitempty"`  // GitHub repo (owner/repo)
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
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

// Message categories (for GitHub sync)
const (
	CategoryBug     = "bug"
	CategoryFeature = "feature"
	CategoryGeneral = "general"
)

// InboxListOptions specifies filters for listing inbox messages
type InboxListOptions struct {
	Inbox       string // Filter by inbox name (empty = all)
	Status      string // Filter by status (empty = all)
	UnreadOnly  bool   // Only unread messages
	FromAgent   string // Filter by sender
	Limit       int    // Max results (0 = default 50)
	IncludeRead bool   // Include read messages (default: true unless UnreadOnly)
}

// InsertInboxMessage adds a new message to the inbox
func (s *Store) InsertInboxMessage(msg *InboxMessage) error {
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

	_, err := s.db.Exec(`
		INSERT INTO inbox_messages (id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, status, created_at, read_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.MessageID, msg.CorrelationID, msg.FromAgent, msg.ToInbox, msg.MessageType, msg.Title, msg.Payload, category, msg.GitHubIssue, githubRepo, msg.Status, msg.CreatedAt.Format(time.RFC3339), readAt, expiresAt)

	return err
}

// ListInboxMessages returns messages matching the given options
func (s *Store) ListInboxMessages(opts InboxListOptions) ([]InboxMessage, error) {
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, status, created_at, read_at, expires_at FROM inbox_messages WHERE 1=1`
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

	query += " ORDER BY created_at DESC"

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []InboxMessage
	for rows.Next() {
		var msg InboxMessage
		var correlationID, payload, category, githubRepo sql.NullString
		var githubIssue sql.NullInt64
		var readAt, expiresAt sql.NullString
		var createdAt string

		err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &msg.Status, &createdAt, &readAt, &expiresAt)
		if err != nil {
			return nil, err
		}

		msg.CorrelationID = correlationID.String
		msg.Payload = payload.String
		msg.Category = category.String
		msg.GitHubRepo = githubRepo.String
		if githubIssue.Valid {
			issueNum := int(githubIssue.Int64)
			msg.GitHubIssue = &issueNum
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

	return messages, rows.Err()
}

// GetInboxMessage returns a single message by ID (UUID or message_id)
func (s *Store) GetInboxMessage(id string) (*InboxMessage, error) {
	row := s.db.QueryRow(`
		SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE id = ? OR message_id = ?
	`, id, id)

	var msg InboxMessage
	var correlationID, payload, category, githubRepo sql.NullString
	var githubIssue sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string

	err := row.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &msg.Status, &createdAt, &readAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	msg.CorrelationID = correlationID.String
	msg.Payload = payload.String
	msg.Category = category.String
	msg.GitHubRepo = githubRepo.String
	if githubIssue.Valid {
		issueNum := int(githubIssue.Int64)
		msg.GitHubIssue = &issueNum
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

	return &msg, nil
}

// MarkInboxMessageRead marks a message as read
func (s *Store) MarkInboxMessageRead(id string) error {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(`
		UPDATE inbox_messages SET status = ?, read_at = ? WHERE (id = ? OR message_id = ?) AND status = ?
	`, InboxStatusRead, now, id, id, InboxStatusUnread)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("message not found or already read")
	}
	return nil
}

// MarkInboxMessageUnread marks a message as unread
func (s *Store) MarkInboxMessageUnread(id string) error {
	result, err := s.db.Exec(`
		UPDATE inbox_messages SET status = ?, read_at = NULL WHERE id = ? OR message_id = ?
	`, InboxStatusUnread, id, id)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("message not found")
	}
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
		return 0, err
	}

	return result.RowsAffected()
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

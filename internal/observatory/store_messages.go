// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"fmt"
	"time"
)

// MessageListOptions configures message listing.
type MessageListOptions struct {
	Inbox     string
	Status    MessageStatus
	TaskID    string
	FromAgent string
	Limit     int
	Offset    int
}

// CreateMessage inserts a new message.
func (s *Store) CreateMessage(m *Message) error {
	// Convert empty strings to NULL for foreign key columns
	var taskID, replyToID interface{}
	if m.TaskID != "" {
		taskID = m.TaskID
	}
	if m.ReplyToID != "" {
		replyToID = m.ReplyToID
	}

	// Convert zero to NULL for github_issue_number
	var ghIssue interface{}
	if m.GitHubIssueNumber != 0 {
		ghIssue = m.GitHubIssueNumber
	}

	_, err := s.db.Exec(`
		INSERT INTO messages (id, task_id, inbox, from_agent, title, content,
		                      message_type, status, priority, github_issue_number,
		                      github_repo, correlation_id, reply_to_id, created_at,
		                      read_at, archived_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, taskID, m.Inbox, m.FromAgent, m.Title, m.Content,
		m.MessageType, m.Status, m.Priority, ghIssue,
		m.GitHubRepo, m.CorrelationID, replyToID, m.CreatedAt,
		m.ReadAt, m.ArchivedAt, m.ContentHash)
	return err
}

// GetMessage retrieves a message by ID.
func (s *Store) GetMessage(id string) (*Message, error) {
	m := &Message{}
	var taskID, correlationID, replyToID, contentHash, ghRepo sql.NullString
	var ghIssue sql.NullInt64
	var readAt, archivedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, task_id, inbox, from_agent, title, content,
		       message_type, status, priority, github_issue_number,
		       github_repo, correlation_id, reply_to_id, created_at,
		       read_at, archived_at, content_hash
		FROM messages WHERE id = ?
	`, id).Scan(&m.ID, &taskID, &m.Inbox, &m.FromAgent, &m.Title, &m.Content,
		&m.MessageType, &m.Status, &m.Priority, &ghIssue,
		&ghRepo, &correlationID, &replyToID, &m.CreatedAt,
		&readAt, &archivedAt, &contentHash)
	if err != nil {
		return nil, err
	}

	if taskID.Valid {
		m.TaskID = taskID.String
	}
	if correlationID.Valid {
		m.CorrelationID = correlationID.String
	}
	if replyToID.Valid {
		m.ReplyToID = replyToID.String
	}
	if contentHash.Valid {
		m.ContentHash = contentHash.String
	}
	if ghRepo.Valid {
		m.GitHubRepo = ghRepo.String
	}
	if ghIssue.Valid {
		m.GitHubIssueNumber = int(ghIssue.Int64)
	}
	if readAt.Valid {
		m.ReadAt = &readAt.Time
	}
	if archivedAt.Valid {
		m.ArchivedAt = &archivedAt.Time
	}

	return m, nil
}

// ListMessages returns messages with optional filtering.
func (s *Store) ListMessages(opts MessageListOptions) ([]*Message, error) {
	query := `
		SELECT id, task_id, inbox, from_agent, title, content,
		       message_type, status, priority, github_issue_number,
		       github_repo, correlation_id, reply_to_id, created_at,
		       read_at, archived_at, content_hash
		FROM messages WHERE 1=1
	`
	var args []interface{}

	if opts.Inbox != "" {
		query += " AND inbox = ?"
		args = append(args, opts.Inbox)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.FromAgent != "" {
		query += " AND from_agent = ?"
		args = append(args, opts.FromAgent)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		m := &Message{}
		var taskID, correlationID, replyToID, contentHash, ghRepo sql.NullString
		var ghIssue sql.NullInt64
		var readAt, archivedAt sql.NullTime

		if err := rows.Scan(&m.ID, &taskID, &m.Inbox, &m.FromAgent, &m.Title, &m.Content,
			&m.MessageType, &m.Status, &m.Priority, &ghIssue,
			&ghRepo, &correlationID, &replyToID, &m.CreatedAt,
			&readAt, &archivedAt, &contentHash); err != nil {
			return nil, err
		}

		if taskID.Valid {
			m.TaskID = taskID.String
		}
		if correlationID.Valid {
			m.CorrelationID = correlationID.String
		}
		if replyToID.Valid {
			m.ReplyToID = replyToID.String
		}
		if contentHash.Valid {
			m.ContentHash = contentHash.String
		}
		if ghRepo.Valid {
			m.GitHubRepo = ghRepo.String
		}
		if ghIssue.Valid {
			m.GitHubIssueNumber = int(ghIssue.Int64)
		}
		if readAt.Valid {
			m.ReadAt = &readAt.Time
		}
		if archivedAt.Valid {
			m.ArchivedAt = &archivedAt.Time
		}

		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// UpdateMessage updates an existing message.
func (s *Store) UpdateMessage(m *Message) error {
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, read_at = ?, archived_at = ?
		WHERE id = ?
	`, m.Status, m.ReadAt, m.ArchivedAt, m.ID)
	return err
}

// DeleteMessage removes a message by ID.
func (s *Store) DeleteMessage(id string) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

// MarkMessageRead marks a message as read.
func (s *Store) MarkMessageRead(id string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, read_at = ? WHERE id = ?
	`, MessageStatusRead, now, id)
	return err
}

// MarkMessageArchived marks a message as archived.
func (s *Store) MarkMessageArchived(id string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, archived_at = ? WHERE id = ?
	`, MessageStatusArchived, now, id)
	return err
}

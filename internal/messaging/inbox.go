package messaging

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/telemetry"
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
	ParentTaskID       string     `json:"parent_task_id,omitempty"`       // Parent task for hierarchy (v1.5.0, M-UNIFIED-AI-CONTROL-PLANE)
	ChainID            string     `json:"chain_id,omitempty"`             // Execution chain ID for unified hierarchy (M-CHAINS-SIMPLIFY)
	Iteration          int        `json:"iteration,omitempty"`            // Iteration number for feedback loops (M-TASK-HIERARCHY)
	Envelope           *Envelope  `json:"envelope,omitempty"`             // Multi-aspect semantic embeddings (v1.8.0, M-SEMANTIC-ENVELOPE)
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
	Limit       int    // Max results (0 = no limit)
	IncludeRead bool   // Include read messages (default: true unless UnreadOnly)
	Collapsed   bool   // Hide messages where dup_of IS NOT NULL (semantic dedup)
	DupOf       string // Only messages that are duplicates of this ID
	StartDate   string // Filter messages created >= this date (YYYY-MM-DD)
	EndDate     string // Filter messages created <= this date (YYYY-MM-DD)
}

// InsertInboxMessage adds a new message to the inbox.
// For trace context propagation, use InsertInboxMessageWithContext instead.
func (s *Store) InsertInboxMessage(msg *InboxMessage) error {
	return s.InsertInboxMessageWithContext(context.Background(), msg)
}

// InsertInboxMessageWithContext adds a new message to the inbox with trace context propagation.
// Use this when you want the messages.send span to be a child of the caller's trace.
func (s *Store) InsertInboxMessageWithContext(ctx context.Context, msg *InboxMessage) error {
	// Start span for message send operation
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.send",
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

	// Handle nullable parent_task_id field (v1.5.0, M-UNIFIED-AI-CONTROL-PLANE)
	var parentTaskID *string
	if msg.ParentTaskID != "" {
		parentTaskID = &msg.ParentTaskID
	}

	// Handle nullable chain_id field (v1.7.0, M-CHAINS-SIMPLIFY)
	var chainID *string
	if msg.ChainID != "" {
		chainID = &msg.ChainID
	}

	// Serialize envelope (v1.8.0, M-SEMANTIC-ENVELOPE)
	var envelopeJSON *string
	if msg.Envelope != nil && !msg.Envelope.IsEmpty() {
		s := msg.Envelope.ToJSON()
		envelopeJSON = &s
	}

	_, err := s.db.Exec(`
		INSERT INTO inbox_messages (id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, parent_task_id, chain_id, envelope, status, created_at, read_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.MessageID, msg.CorrelationID, msg.FromAgent, msg.ToInbox, msg.MessageType, msg.Title, msg.Payload, category, msg.GitHubIssue, githubRepo, simhash, dupOf, parentTaskID, chainID, envelopeJSON, msg.Status, msg.CreatedAt.Format(time.RFC3339), readAt, expiresAt)

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
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.list",
		trace.WithAttributes(
			attribute.String("list.inbox", opts.Inbox),
			attribute.Bool("list.unread_only", opts.UnreadOnly),
			attribute.Bool("list.collapsed", opts.Collapsed),
			attribute.Int("list.limit", opts.Limit),
		),
	)
	defer span.End()

	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, parent_task_id, chain_id, envelope, status, created_at, read_at, expires_at FROM inbox_messages WHERE 1=1`
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

	// Date range filtering
	if opts.StartDate != "" {
		query += " AND created_at >= ?"
		args = append(args, opts.StartDate+" 00:00:00")
	}
	if opts.EndDate != "" {
		query += " AND created_at <= ?"
		args = append(args, opts.EndDate+" 23:59:59")
	}

	query += " ORDER BY created_at DESC"

	// Only apply LIMIT if opts.Limit > 0 (0 = no limit)
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

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
		var correlationID, payload, category, githubRepo, dupOf, parentTaskID, chainID, envelopeJSON sql.NullString
		var githubIssue, simhash sql.NullInt64
		var readAt, expiresAt sql.NullString
		var createdAt string

		err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &simhash, &dupOf, &parentTaskID, &chainID, &envelopeJSON, &msg.Status, &createdAt, &readAt, &expiresAt)
		if err != nil {
			return nil, err
		}

		msg.CorrelationID = correlationID.String
		msg.Payload = payload.String
		msg.Category = category.String
		msg.GitHubRepo = githubRepo.String
		msg.DupOf = dupOf.String
		msg.ParentTaskID = parentTaskID.String
		msg.ChainID = chainID.String
		if envelopeJSON.Valid && envelopeJSON.String != "" && envelopeJSON.String != "{}" {
			msg.Envelope = EnvelopeFromJSON(envelopeJSON.String)
		}
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
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.read",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	row := s.db.QueryRow(`
		SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, parent_task_id, chain_id, envelope, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE id = ? OR message_id = ?
	`, id, id)

	var msg InboxMessage
	var correlationID, payload, category, githubRepo, dupOf, parentTaskID, chainID, envelopeJSON sql.NullString
	var githubIssue, simhash sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string

	err := row.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox, &msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo, &simhash, &dupOf, &parentTaskID, &chainID, &envelopeJSON, &msg.Status, &createdAt, &readAt, &expiresAt)
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
	msg.ParentTaskID = parentTaskID.String
	msg.ChainID = chainID.String
	if envelopeJSON.Valid && envelopeJSON.String != "" && envelopeJSON.String != "{}" {
		msg.Envelope = EnvelopeFromJSON(envelopeJSON.String)
	}
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

// FindMessageByPrefix resolves a short ID prefix to a full message ID using SQL.
// Returns error if no match or multiple matches (ambiguous prefix).
func (s *Store) FindMessageByPrefix(prefix string) (string, error) {
	rows, err := s.db.Query(
		`SELECT id FROM inbox_messages WHERE id LIKE ? LIMIT 2`,
		prefix+"%",
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matches []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		matches = append(matches, id)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no message found with prefix '%s'", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix '%s' matches multiple messages, use a longer prefix", prefix)
	}
}

// MarkInboxMessageRead marks a message as read
func (s *Store) MarkInboxMessageRead(id string) error {
	// Start span for message ack operation
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.ack",
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
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.unack",
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

// InboxMessageExistsByTitle checks if a message with the same title already exists in the inbox
// Returns the existing message ID if found, or empty string if no duplicate
func (s *Store) InboxMessageExistsByTitle(inbox string, title string) (string, error) {
	var existingID string
	err := s.db.QueryRow(`
		SELECT id FROM inbox_messages
		WHERE to_inbox = ? AND title = ? AND status NOT IN (?, ?)
		ORDER BY created_at DESC
		LIMIT 1
	`, inbox, title, InboxStatusDeleted, InboxStatusArchived).Scan(&existingID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	return existingID, nil
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
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.cleanup",
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

// ===== Observed Topology Queries =====

// MessageFlowEdge represents an edge between agents based on actual message handoffs
type MessageFlowEdge struct {
	FromAgent    string `json:"from_agent"`
	ToInbox      string `json:"to_inbox"`
	MessageCount int    `json:"message_count"`
	LastActivity string `json:"last_activity,omitempty"`
}

// ActiveAgent represents an agent that has sent or received messages
type ActiveAgent struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	MessagesSent int    `json:"messages_sent"`
	MessagesRecv int    `json:"messages_recv"`
	LastActivity string `json:"last_activity"`
}

// GetMessageFlowEdges returns edges derived from actual from_agent → to_inbox message flows
func (s *Store) GetMessageFlowEdges() ([]MessageFlowEdge, error) {
	rows, err := s.db.Query(`
		SELECT
			from_agent,
			to_inbox,
			COUNT(*) as message_count,
			MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE from_agent IS NOT NULL AND from_agent != ''
		  AND to_inbox IS NOT NULL AND to_inbox != ''
		GROUP BY from_agent, to_inbox
		ORDER BY message_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []MessageFlowEdge
	for rows.Next() {
		var edge MessageFlowEdge
		if err := rows.Scan(&edge.FromAgent, &edge.ToInbox, &edge.MessageCount, &edge.LastActivity); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

// GetActiveAgents returns agents that have sent or received messages
func (s *Store) GetActiveAgents() ([]ActiveAgent, error) {
	// Get agents who have sent messages
	sentQuery := `
		SELECT from_agent as id, COUNT(*) as count, MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE from_agent IS NOT NULL AND from_agent != ''
		GROUP BY from_agent
	`

	// Get agents who have received messages (using to_inbox as agent id)
	recvQuery := `
		SELECT to_inbox as id, COUNT(*) as count, MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE to_inbox IS NOT NULL AND to_inbox != ''
		GROUP BY to_inbox
	`

	// Combine both using a union and aggregate
	combinedQuery := `
		SELECT
			id,
			id as label,
			COALESCE(SUM(sent_count), 0) as messages_sent,
			COALESCE(SUM(recv_count), 0) as messages_recv,
			MAX(last_activity) as last_activity
		FROM (
			SELECT id, count as sent_count, 0 as recv_count, last_activity FROM (` + sentQuery + `)
			UNION ALL
			SELECT id, 0 as sent_count, count as recv_count, last_activity FROM (` + recvQuery + `)
		)
		GROUP BY id
		ORDER BY messages_sent + messages_recv DESC
	`

	rows, err := s.db.Query(combinedQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []ActiveAgent
	for rows.Next() {
		var agent ActiveAgent
		if err := rows.Scan(&agent.ID, &agent.Label, &agent.MessagesSent, &agent.MessagesRecv, &agent.LastActivity); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

package messaging

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SearchOptions configures semantic search parameters
type SearchOptions struct {
	Query     string   // Natural language query
	Threshold float64  // Minimum similarity (0.0-1.0), default 0.70
	Limit     int      // Max results, default 20
	MaxScan   int      // Max messages to scan, default 1000
	Inbox     string   // Filter by inbox (optional)
	UseNeural bool     // Use embedding search via Ollama
	Embedder  Embedder // Optional embedder instance (created if nil and UseNeural=true)
}

// SearchHit represents a search result with similarity score
type SearchHit struct {
	Message   InboxMessage `json:"message"`
	Score     float64      `json:"score"`      // Similarity score (0.0-1.0)
	ScoreKind string       `json:"score_kind"` // "simhash" or "embedding"
}

// SemanticSearch finds messages similar to the query using SimHash or embeddings
func (s *Store) SemanticSearch(opts SearchOptions) ([]SearchHit, error) {
	// Start span for search operation
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, messagingTracer, "messages.search",
		trace.WithAttributes(
			attribute.String("search.query", opts.Query),
			attribute.Bool("search.use_neural", opts.UseNeural),
			attribute.Float64("search.threshold", opts.Threshold),
			attribute.Int("search.limit", opts.Limit),
			attribute.Int("search.max_scan", opts.MaxScan),
			attribute.String("search.inbox", opts.Inbox),
		),
	)
	defer span.End()

	// Apply defaults
	if opts.Threshold <= 0 {
		opts.Threshold = 0.70
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.MaxScan <= 0 {
		opts.MaxScan = 1000
	}

	// Use neural search if requested
	if opts.UseNeural {
		results, err := s.neuralSearch(opts)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "neural search failed")
			return nil, err
		}
		span.SetAttributes(attribute.Int("search.result_count", len(results)))
		span.SetStatus(codes.Ok, "search completed")
		return results, nil
	}

	// Compute query simhash
	queryHash := builtins.SimHash(opts.Query)

	// Build query for messages
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE simhash IS NOT NULL`
	args := []interface{}{}

	if opts.Inbox != "" {
		query += " AND to_inbox = ?"
		args = append(args, opts.Inbox)
	}

	// Exclude deleted messages
	query += " AND status != ?"
	args = append(args, InboxStatusDeleted)

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, opts.MaxScan)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		msg, err := scanInboxMessage(rows)
		if err != nil {
			return nil, err
		}

		// Skip messages without simhash
		if msg.Simhash == nil {
			continue
		}

		// Compute similarity: 1.0 - (hamming_distance / 64.0)
		distance := builtins.HammingDistance(queryHash, *msg.Simhash)
		score := 1.0 - (float64(distance) / 64.0)

		// Apply threshold
		if score >= opts.Threshold {
			hits = append(hits, SearchHit{
				Message:   msg,
				Score:     score,
				ScoreKind: "simhash",
			})
		}
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error iterating search results")
		return nil, err
	}

	// Sort by score descending, then by message_id ascending for determinism
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Message.MessageID < hits[j].Message.MessageID
	})

	// Apply limit
	if len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}

	span.SetAttributes(attribute.Int("search.result_count", len(hits)))
	span.SetStatus(codes.Ok, "search completed")
	return hits, nil
}

// FindSimilar finds messages similar to a given message by ID
func (s *Store) FindSimilar(msgID string, threshold float64, limit int) ([]SearchHit, error) {
	// Get the reference message
	msg, err := s.GetInboxMessage(msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil // Message not found
	}

	// If message has no simhash, compute it from content
	var queryHash int64
	if msg.Simhash != nil {
		queryHash = *msg.Simhash
	} else {
		searchText := msg.Title
		if msg.Payload != "" {
			searchText += " " + msg.Payload
		}
		queryHash = builtins.SimHash(searchText)
	}

	// Apply defaults
	if threshold <= 0 {
		threshold = 0.70
	}
	if limit <= 0 {
		limit = 20
	}

	// Query for similar messages (excluding the reference message)
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE simhash IS NOT NULL
		AND message_id != ?
		AND status != ?`
	args := []interface{}{msgID, InboxStatusDeleted}

	// Optionally filter by same inbox
	if msg.ToInbox != "" {
		query += " AND to_inbox = ?"
		args = append(args, msg.ToInbox)
	}

	query += " ORDER BY created_at DESC LIMIT 1000"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		candidate, err := scanInboxMessage(rows)
		if err != nil {
			return nil, err
		}

		if candidate.Simhash == nil {
			continue
		}

		// Compute similarity
		distance := builtins.HammingDistance(queryHash, *candidate.Simhash)
		score := 1.0 - (float64(distance) / 64.0)

		if score >= threshold {
			hits = append(hits, SearchHit{
				Message:   candidate,
				Score:     score,
				ScoreKind: "simhash",
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort by score descending, then by message_id ascending for determinism
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Message.MessageID < hits[j].Message.MessageID
	})

	// Apply limit
	if len(hits) > limit {
		hits = hits[:limit]
	}

	return hits, nil
}

// scanInboxMessage scans a row into an InboxMessage
func scanInboxMessage(rows *sql.Rows) (InboxMessage, error) {
	var msg InboxMessage
	var correlationID, payload, category, githubRepo, dupOf sql.NullString
	var githubIssue, simhash sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string

	err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox,
		&msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo,
		&simhash, &dupOf, &msg.Status, &createdAt, &readAt, &expiresAt)
	if err != nil {
		return msg, err
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

	return msg, nil
}

// DuplicateGroup represents a cluster of near-duplicate messages
type DuplicateGroup struct {
	Representative InboxMessage   `json:"representative"` // Oldest message in group (kept)
	Duplicates     []InboxMessage `json:"duplicates"`     // Similar messages (to be marked)
	MinScore       float64        `json:"min_score"`      // Minimum similarity in group
	ScoreKind      string         `json:"score_kind"`     // "simhash" or "embedding"
}

// FindDuplicates identifies clusters of near-duplicate messages
func (s *Store) FindDuplicates(inbox string, threshold float64) ([]DuplicateGroup, error) {
	if threshold <= 0 {
		threshold = 0.95 // Default high threshold for deduplication
	}

	// Build query for messages with simhash
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload, category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE simhash IS NOT NULL
		AND dup_of IS NULL
		AND status != ?`
	args := []interface{}{InboxStatusDeleted}

	if inbox != "" {
		query += " AND to_inbox = ?"
		args = append(args, inbox)
	}

	// Order by created_at ASC so oldest messages come first (will be representatives)
	query += " ORDER BY created_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Load all messages
	var messages []InboxMessage
	for rows.Next() {
		msg, err := scanInboxMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build clusters using union-find approach
	// processed[i] tracks if message i has been assigned to a group
	processed := make([]bool, len(messages))
	var groups []DuplicateGroup

	for i := range messages {
		if processed[i] {
			continue
		}

		// Start a new group with message i as representative (it's the oldest due to ORDER BY)
		group := DuplicateGroup{
			Representative: messages[i],
			MinScore:       1.0,
			ScoreKind:      "simhash",
		}
		processed[i] = true

		// Find all similar messages
		for j := i + 1; j < len(messages); j++ {
			if processed[j] {
				continue
			}

			// Compute similarity
			if messages[i].Simhash == nil || messages[j].Simhash == nil {
				continue
			}
			distance := builtins.HammingDistance(*messages[i].Simhash, *messages[j].Simhash)
			score := 1.0 - (float64(distance) / 64.0)

			if score >= threshold {
				group.Duplicates = append(group.Duplicates, messages[j])
				if score < group.MinScore {
					group.MinScore = score
				}
				processed[j] = true
			}
		}

		// Only add groups that have duplicates
		if len(group.Duplicates) > 0 {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// ApplyDuplicates marks duplicate messages by setting dup_of and status
func (s *Store) ApplyDuplicates(groups []DuplicateGroup, runID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`UPDATE inbox_messages SET dup_of = ?, status = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, group := range groups {
		for _, dup := range group.Duplicates {
			_, err := stmt.Exec(group.Representative.ID, InboxStatusRead, dup.ID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ClearDuplicateMarker clears the dup_of field for a message (undo deduplication)
func (s *Store) ClearDuplicateMarker(msgID string) error {
	result, err := s.db.Exec(`UPDATE inbox_messages SET dup_of = NULL WHERE id = ? OR message_id = ?`, msgID, msgID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// neuralSearch performs semantic search using embeddings via Ollama
func (s *Store) neuralSearch(opts SearchOptions) ([]SearchHit, error) {
	// Create embedder if not provided
	embedder := opts.Embedder
	if embedder == nil {
		cfg := LoadEmbedConfigFromEnv()
		var err error
		embedder, err = NewOllamaEmbedder(cfg.Ollama)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedder: %w (is Ollama running?)", err)
		}
	}

	// Compute query embedding
	queryEmbedding, err := embedder.Embed(opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Get candidate messages
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload,
		category, github_issue_number, github_repo, simhash, dup_of, status, created_at, read_at, expires_at,
		embedding, embedding_model, embedding_updated_at
		FROM inbox_messages
		WHERE status != ?`
	args := []interface{}{InboxStatusDeleted}

	if opts.Inbox != "" {
		query += " AND to_inbox = ?"
		args = append(args, opts.Inbox)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, opts.MaxScan)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	var needsEmbedding []InboxMessage // Messages that need embeddings computed

	for rows.Next() {
		msg, embedding, err := scanInboxMessageWithEmbedding(rows)
		if err != nil {
			return nil, err
		}

		if embedding != nil {
			// Has embedding - compute similarity
			score := CosineSimilarity(queryEmbedding, embedding)
			if score >= opts.Threshold {
				hits = append(hits, SearchHit{
					Message:   msg,
					Score:     score,
					ScoreKind: "embedding",
				})
			}
		} else {
			// Needs embedding - queue for lazy computation
			needsEmbedding = append(needsEmbedding, msg)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Lazy-embed messages that don't have embeddings (bounded)
	maxLazyEmbed := 50 // Limit lazy embedding to avoid long delays
	if len(needsEmbedding) > maxLazyEmbed {
		needsEmbedding = needsEmbedding[:maxLazyEmbed]
	}

	for _, msg := range needsEmbedding {
		searchText := msg.Title
		if msg.Payload != "" {
			searchText += " " + msg.Payload
		}

		msgEmbedding, err := embedder.Embed(searchText)
		if err != nil {
			continue // Skip on error, don't fail entire search
		}

		// Store embedding for future use (ignore errors - non-critical)
		_ = s.UpdateMessageEmbedding(msg.ID, msgEmbedding, embedder.ModelName())

		// Compute similarity
		score := CosineSimilarity(queryEmbedding, msgEmbedding)
		if score >= opts.Threshold {
			hits = append(hits, SearchHit{
				Message:   msg,
				Score:     score,
				ScoreKind: "embedding",
			})
		}
	}

	// Sort by score descending, then by message_id ascending for determinism
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Message.MessageID < hits[j].Message.MessageID
	})

	// Apply limit
	if len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}

	return hits, nil
}

// scanInboxMessageWithEmbedding scans a row including embedding fields
func scanInboxMessageWithEmbedding(rows *sql.Rows) (InboxMessage, []float32, error) {
	var msg InboxMessage
	var correlationID, payload, category, githubRepo, dupOf sql.NullString
	var githubIssue, simhash sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string
	var embeddingJSON, embeddingModel sql.NullString
	var embeddingUpdatedAt sql.NullInt64

	err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox,
		&msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo,
		&simhash, &dupOf, &msg.Status, &createdAt, &readAt, &expiresAt,
		&embeddingJSON, &embeddingModel, &embeddingUpdatedAt)
	if err != nil {
		return msg, nil, err
	}

	msg.CorrelationID = correlationID.String
	msg.Payload = payload.String
	msg.Category = category.String
	msg.GitHubRepo = githubRepo.String
	msg.DupOf = dupOf.String
	msg.Embedding = embeddingJSON.String
	msg.EmbeddingModel = embeddingModel.String
	if embeddingUpdatedAt.Valid {
		ts := embeddingUpdatedAt.Int64
		msg.EmbeddingUpdatedAt = &ts
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

	// Parse embedding if present
	var embedding []float32
	if embeddingJSON.Valid && embeddingJSON.String != "" {
		embedding, _ = EmbeddingFromJSON(embeddingJSON.String)
	}

	return msg, embedding, nil
}

// UpdateMessageEmbedding stores an embedding for a message
func (s *Store) UpdateMessageEmbedding(msgID string, embedding []float32, model string) error {
	embJSON := EmbeddingToJSON(embedding)
	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		UPDATE inbox_messages
		SET embedding = ?, embedding_model = ?, embedding_updated_at = ?
		WHERE id = ? OR message_id = ?
	`, embJSON, model, now, msgID, msgID)

	return err
}

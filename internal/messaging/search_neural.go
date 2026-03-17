package messaging

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

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

// SearchByEnvelope finds messages by comparing the query embedding against a specific
// envelope slot across all messages. Returns results sorted by similarity score.
func (s *Store) SearchByEnvelope(opts SearchOptions) ([]SearchHit, error) {
	if opts.EnvelopeSpace == "" {
		return nil, fmt.Errorf("EnvelopeSpace is required for envelope search")
	}
	if err := ValidateSlot(opts.EnvelopeSpace); err != nil {
		return nil, err
	}

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

	// Get or create embedder
	embedder := opts.Embedder
	if embedder == nil {
		cfg := LoadEmbedConfigFromEnv()
		var err error
		embedder, err = NewEmbedderFromConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedder: %w", err)
		}
		if embedder == nil {
			return nil, fmt.Errorf("no embedder available (provider is 'none')")
		}
	}

	// Embed the query
	queryVec, err := embedder.Embed(opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Fetch messages with envelopes
	query := `SELECT id, message_id, correlation_id, from_agent, to_inbox, message_type, title, payload,
		category, github_issue_number, github_repo, simhash, dup_of, parent_task_id, chain_id, envelope,
		status, created_at, read_at, expires_at
		FROM inbox_messages
		WHERE status != ? AND envelope IS NOT NULL AND envelope != '{}'`
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
	for rows.Next() {
		msg, err := scanInboxMessageFull(rows)
		if err != nil {
			return nil, err
		}

		if msg.Envelope == nil {
			continue
		}

		slotVec := msg.Envelope.GetVector(opts.EnvelopeSpace)
		if slotVec == nil {
			continue
		}

		score := CosineSimilarity(queryVec, slotVec)
		if score >= opts.Threshold {
			hits = append(hits, SearchHit{
				Message:   msg,
				Score:     score,
				ScoreKind: "envelope:" + opts.EnvelopeSpace,
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

	if len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}

	return hits, nil
}

// scanInboxMessageFull scans a row with all columns including envelope.
func scanInboxMessageFull(rows *sql.Rows) (InboxMessage, error) {
	var msg InboxMessage
	var correlationID, payload, category, githubRepo, dupOf, parentTaskID, chainID, envelopeJSON sql.NullString
	var githubIssue, simhash sql.NullInt64
	var readAt, expiresAt sql.NullString
	var createdAt string

	err := rows.Scan(&msg.ID, &msg.MessageID, &correlationID, &msg.FromAgent, &msg.ToInbox,
		&msg.MessageType, &msg.Title, &payload, &category, &githubIssue, &githubRepo,
		&simhash, &dupOf, &parentTaskID, &chainID, &envelopeJSON,
		&msg.Status, &createdAt, &readAt, &expiresAt)
	if err != nil {
		return msg, err
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

	return msg, nil
}

// UpdateMessageEnvelope merges new envelope slots into a message's existing envelope.
// Existing slots are preserved unless overwrite is true.
func (s *Store) UpdateMessageEnvelope(msgID string, env *Envelope, overwrite bool) error {
	if env == nil || env.IsEmpty() {
		return nil
	}

	// Read existing envelope
	var existingJSON sql.NullString
	err := s.db.QueryRow(`SELECT envelope FROM inbox_messages WHERE id = ? OR message_id = ?`, msgID, msgID).Scan(&existingJSON)
	if err != nil {
		return fmt.Errorf("failed to read existing envelope: %w", err)
	}

	existing := EnvelopeFromJSON(existingJSON.String)
	if overwrite {
		existing.MergeOverwrite(env)
	} else {
		existing.Merge(env)
	}

	_, err = s.db.Exec(`UPDATE inbox_messages SET envelope = ? WHERE id = ? OR message_id = ?`,
		existing.ToJSON(), msgID, msgID)
	return err
}

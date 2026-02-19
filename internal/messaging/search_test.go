package messaging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSemanticSearch_FindsSimilarMessages(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert test messages
	messages := []struct {
		title   string
		payload string
	}{
		{"Parser error on multiline strings", "The parser fails when parsing multiline strings with escapes"},
		{"Parsing fails for heredoc syntax", "Multiline heredocs cause parser to crash"},
		{"Unrelated message about types", "Type inference is working correctly"},
		{"Another parser bug", "Parser crashes on nested expressions"},
	}

	for _, m := range messages {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     m.title,
			Payload:   m.payload,
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Search for parser-related messages
	hits, err := store.SemanticSearch(SearchOptions{
		Query:     "parser multiline",
		Threshold: 0.5,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SemanticSearch failed: %v", err)
	}

	// Should find at least 2 parser-related messages
	if len(hits) < 2 {
		t.Errorf("expected at least 2 hits, got %d", len(hits))
	}

	// Verify all hits have score >= threshold
	for _, hit := range hits {
		if hit.Score < 0.5 {
			t.Errorf("hit score %f is below threshold 0.5", hit.Score)
		}
		if hit.ScoreKind != "simhash" {
			t.Errorf("expected score_kind 'simhash', got '%s'", hit.ScoreKind)
		}
	}
}

func TestSemanticSearch_ThresholdFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert diverse messages
	titles := []string{
		"Parser error in lexer",
		"Type checker bug",
		"Database connection issue",
		"Memory leak in evaluator",
	}

	for _, title := range titles {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     title,
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// High threshold should return fewer results
	highHits, err := store.SemanticSearch(SearchOptions{
		Query:     "parser lexer",
		Threshold: 0.9,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SemanticSearch failed: %v", err)
	}

	// Low threshold should return more results
	lowHits, err := store.SemanticSearch(SearchOptions{
		Query:     "parser lexer",
		Threshold: 0.3,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SemanticSearch failed: %v", err)
	}

	// Low threshold should have >= high threshold results
	if len(lowHits) < len(highHits) {
		t.Errorf("low threshold (%d hits) should have >= high threshold (%d hits)",
			len(lowHits), len(highHits))
	}
}

func TestSemanticSearch_DeterministicOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert test messages
	for i := 0; i < 5; i++ {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     "Test message about parsing",
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Run same search twice
	opts := SearchOptions{
		Query:     "parsing",
		Threshold: 0.3,
		Limit:     10,
	}

	hits1, err := store.SemanticSearch(opts)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}

	hits2, err := store.SemanticSearch(opts)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}

	// Results should be identical
	if len(hits1) != len(hits2) {
		t.Fatalf("result count differs: %d vs %d", len(hits1), len(hits2))
	}

	for i := range hits1 {
		if hits1[i].Message.MessageID != hits2[i].Message.MessageID {
			t.Errorf("result %d differs: %s vs %s",
				i, hits1[i].Message.MessageID, hits2[i].Message.MessageID)
		}
	}
}

func TestFindSimilar_ByMessageID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert similar messages
	msg1 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Parser error on ADT patterns",
		Payload:   "The parser fails when parsing algebraic data types",
	}
	if err := store.InsertInboxMessage(msg1); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	msg2 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "ADT parsing causes crash",
		Payload:   "Algebraic data type parsing fails with stack overflow",
	}
	if err := store.InsertInboxMessage(msg2); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Insert unrelated message
	msg3 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Database connection timeout",
		Payload:   "Connection to PostgreSQL times out after 30 seconds",
	}
	if err := store.InsertInboxMessage(msg3); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Find similar to msg1
	hits, err := store.FindSimilar(msg1.MessageID, 0.5, 10)
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	// Should find msg2 as similar
	if len(hits) == 0 {
		t.Error("expected at least 1 similar message")
	}

	// Should not include the original message
	for _, hit := range hits {
		if hit.Message.MessageID == msg1.MessageID {
			t.Error("FindSimilar should not include the original message")
		}
	}
}

func TestFindSimilar_MessageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Find similar to non-existent message
	hits, err := store.FindSimilar("nonexistent_id", 0.5, 10)
	if err != nil {
		t.Fatalf("FindSimilar failed: %v", err)
	}

	// Should return nil (not found)
	if hits != nil {
		t.Errorf("expected nil for non-existent message, got %d hits", len(hits))
	}
}

func TestSemanticSearch_InboxFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert messages to different inboxes
	msg1 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "inbox_a",
		Title:     "Parser bug report",
	}
	if err := store.InsertInboxMessage(msg1); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	msg2 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "inbox_b",
		Title:     "Parser bug report",
	}
	if err := store.InsertInboxMessage(msg2); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Search only inbox_a
	hits, err := store.SemanticSearch(SearchOptions{
		Query:     "parser bug",
		Threshold: 0.3,
		Inbox:     "inbox_a",
	})
	if err != nil {
		t.Fatalf("SemanticSearch failed: %v", err)
	}

	// Should only find message from inbox_a
	for _, hit := range hits {
		if hit.Message.ToInbox != "inbox_a" {
			t.Errorf("expected inbox_a, got %s", hit.Message.ToInbox)
		}
	}
}

func TestFindDuplicates_ClustersCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert nearly identical messages
	for i := 0; i < 3; i++ {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     "Parser error on multiline strings",
			Payload:   "The parser fails when parsing multiline strings",
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Insert a different message
	msg := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Database connection timeout",
		Payload:   "PostgreSQL connection times out",
	}
	if err := store.InsertInboxMessage(msg); err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Find duplicates with high threshold
	groups, err := store.FindDuplicates("test_inbox", 0.90)
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	// Should have 1 group (the 3 similar parser messages)
	if len(groups) != 1 {
		t.Errorf("expected 1 duplicate group, got %d", len(groups))
	}

	if len(groups) > 0 {
		// Group should have 1 representative and 2 duplicates
		if len(groups[0].Duplicates) != 2 {
			t.Errorf("expected 2 duplicates, got %d", len(groups[0].Duplicates))
		}
		if groups[0].ScoreKind != "simhash" {
			t.Errorf("expected score_kind 'simhash', got '%s'", groups[0].ScoreKind)
		}
	}
}

func TestApplyDuplicates_SetsDupOf(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert duplicate messages
	for i := 0; i < 3; i++ {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     "Same title for duplicates",
			Payload:   "Same payload for duplicates",
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Find duplicates
	groups, err := store.FindDuplicates("test_inbox", 0.90)
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}

	if len(groups) == 0 {
		t.Fatal("expected at least 1 duplicate group")
	}

	// Apply duplicates
	if err := store.ApplyDuplicates(groups, "test_run"); err != nil {
		t.Fatalf("ApplyDuplicates failed: %v", err)
	}

	// Verify dup_of is set on duplicates
	for _, dup := range groups[0].Duplicates {
		msg, err := store.GetInboxMessage(dup.ID)
		if err != nil {
			t.Fatalf("failed to get message: %v", err)
		}
		if msg.DupOf == "" {
			t.Errorf("expected dup_of to be set on message %s", dup.ID)
		}
		if msg.DupOf != groups[0].Representative.ID {
			t.Errorf("expected dup_of to be %s, got %s", groups[0].Representative.ID, msg.DupOf)
		}
	}
}

func TestListWithCollapsed_HidesDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert duplicate messages
	for i := 0; i < 3; i++ {
		msg := &InboxMessage{
			FromAgent: "test",
			ToInbox:   "test_inbox",
			Title:     "Duplicate message",
			Payload:   "Same content",
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Find and apply duplicates
	groups, err := store.FindDuplicates("test_inbox", 0.90)
	if err != nil {
		t.Fatalf("FindDuplicates failed: %v", err)
	}
	if err := store.ApplyDuplicates(groups, "test_run"); err != nil {
		t.Fatalf("ApplyDuplicates failed: %v", err)
	}

	// List without collapsed - should see all 3
	allMsgs, err := store.ListInboxMessages(InboxListOptions{
		Inbox:       "test_inbox",
		IncludeRead: true,
	})
	if err != nil {
		t.Fatalf("ListInboxMessages failed: %v", err)
	}
	if len(allMsgs) != 3 {
		t.Errorf("expected 3 messages without collapsed, got %d", len(allMsgs))
	}

	// List with collapsed - should see only 1 (representative)
	collapsedMsgs, err := store.ListInboxMessages(InboxListOptions{
		Inbox:       "test_inbox",
		IncludeRead: true,
		Collapsed:   true,
	})
	if err != nil {
		t.Fatalf("ListInboxMessages with collapsed failed: %v", err)
	}
	if len(collapsedMsgs) != 1 {
		t.Errorf("expected 1 message with collapsed, got %d", len(collapsedMsgs))
	}
}

func TestSearchByEnvelope_DifferentSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert messages with envelopes containing different slot vectors
	// Message 1: parser bug (code=parser, intent=fix)
	msg1 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Fix parser crash",
		Payload:   "Parser crashes on nested records",
		Envelope: func() *Envelope {
			e := NewEnvelope()
			e.Set(SlotCode, []float32{1.0, 0.0, 0.0, 0.0}, "mock:test")
			e.Set(SlotIntent, []float32{0.0, 1.0, 0.0, 0.0}, "mock:test")
			return e
		}(),
	}

	// Message 2: parser feature (code=parser, intent=feature)
	msg2 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Add parser record support",
		Payload:   "Need to parse record expressions",
		Envelope: func() *Envelope {
			e := NewEnvelope()
			e.Set(SlotCode, []float32{0.9, 0.1, 0.0, 0.0}, "mock:test")
			e.Set(SlotIntent, []float32{0.0, 0.0, 1.0, 0.0}, "mock:test")
			return e
		}(),
	}

	// Message 3: types bug (code=types, intent=fix)
	msg3 := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "test_inbox",
		Title:     "Fix type unification",
		Payload:   "Unifier crashes on recursive types",
		Envelope: func() *Envelope {
			e := NewEnvelope()
			e.Set(SlotCode, []float32{0.0, 0.0, 1.0, 0.0}, "mock:test")
			e.Set(SlotIntent, []float32{0.0, 0.8, 0.2, 0.0}, "mock:test")
			return e
		}(),
	}

	for _, msg := range []*InboxMessage{msg1, msg2, msg3} {
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Create a mock embedder that returns specific vectors based on query
	mockEmb := &deterministicMockEmbedder{
		dim: 4,
		responses: map[string][]float32{
			"parser code":    {1.0, 0.0, 0.0, 0.0}, // Should match msg1 and msg2 on code
			"fix bug intent": {0.0, 1.0, 0.0, 0.0}, // Should match msg1 and msg3 on intent
		},
	}

	// Search code space for "parser code" — should find msg1 and msg2
	codeHits, err := store.SearchByEnvelope(SearchOptions{
		Query:         "parser code",
		EnvelopeSpace: SlotCode,
		Threshold:     0.5,
		Limit:         10,
		Embedder:      mockEmb,
	})
	if err != nil {
		t.Fatalf("SearchByEnvelope code failed: %v", err)
	}

	if len(codeHits) < 1 {
		t.Error("expected at least 1 code-space hit for parser query")
	}
	for _, hit := range codeHits {
		if hit.ScoreKind != "envelope:code" {
			t.Errorf("expected score_kind 'envelope:code', got %q", hit.ScoreKind)
		}
	}

	// Search intent space for "fix bug intent" — should find msg1 and msg3
	intentHits, err := store.SearchByEnvelope(SearchOptions{
		Query:         "fix bug intent",
		EnvelopeSpace: SlotIntent,
		Threshold:     0.5,
		Limit:         10,
		Embedder:      mockEmb,
	})
	if err != nil {
		t.Fatalf("SearchByEnvelope intent failed: %v", err)
	}

	if len(intentHits) < 1 {
		t.Error("expected at least 1 intent-space hit for fix query")
	}
	for _, hit := range intentHits {
		if hit.ScoreKind != "envelope:intent" {
			t.Errorf("expected score_kind 'envelope:intent', got %q", hit.ScoreKind)
		}
	}

	// Verify different spaces return different result sets
	// Code space should NOT match msg3 (types code), intent space should NOT match msg2 (feature intent)
	codeIDs := make(map[string]bool)
	for _, h := range codeHits {
		codeIDs[h.Message.ID] = true
	}
	intentIDs := make(map[string]bool)
	for _, h := range intentHits {
		intentIDs[h.Message.ID] = true
	}

	// At least one ID should differ between the two result sets
	if len(codeHits) > 0 && len(intentHits) > 0 {
		allSame := true
		for id := range codeIDs {
			if !intentIDs[id] {
				allSame = false
				break
			}
		}
		for id := range intentIDs {
			if !codeIDs[id] {
				allSame = false
				break
			}
		}
		if allSame && len(codeIDs) == len(intentIDs) {
			t.Error("code and intent searches should return different result sets")
		}
	}
}

func TestSearchByEnvelope_InvalidSlot(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.SearchByEnvelope(SearchOptions{
		Query:         "test",
		EnvelopeSpace: "bogus",
		Embedder:      &mockEmbedder{dim: 4},
	})
	if err == nil {
		t.Error("SearchByEnvelope with invalid slot should error")
	}
}

func TestSearchByEnvelope_EmptySpace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.SearchByEnvelope(SearchOptions{
		Query:    "test",
		Embedder: &mockEmbedder{dim: 4},
	})
	if err == nil {
		t.Error("SearchByEnvelope without EnvelopeSpace should error")
	}
}

// deterministicMockEmbedder returns pre-configured vectors for known queries
type deterministicMockEmbedder struct {
	dim       int
	responses map[string][]float32
}

func (m *deterministicMockEmbedder) Embed(text string) ([]float32, error) {
	if vec, ok := m.responses[text]; ok {
		return vec, nil
	}
	// Default: return zero vector
	return make([]float32, m.dim), nil
}

func (m *deterministicMockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		v, _ := m.Embed(t)
		results[i] = v
	}
	return results, nil
}

func (m *deterministicMockEmbedder) Dimension() int    { return m.dim }
func (m *deterministicMockEmbedder) ModelName() string { return "mock:deterministic" }

func TestUpdateMessageEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Insert message without envelope
	msg := &InboxMessage{
		FromAgent: "test",
		ToInbox:   "inbox",
		Title:     "Test message",
	}
	if err := store.InsertInboxMessage(msg); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Add intent slot
	env1 := NewEnvelope()
	env1.Set(SlotIntent, []float32{1.0, 2.0}, "model-a")
	if err := store.UpdateMessageEnvelope(msg.ID, env1, false); err != nil {
		t.Fatalf("update envelope failed: %v", err)
	}

	// Read back
	got, err := store.GetInboxMessage(msg.ID)
	if err != nil {
		t.Fatalf("get message failed: %v", err)
	}
	if got.Envelope == nil {
		t.Fatal("envelope should be populated after update")
	}
	if got.Envelope.Get(SlotIntent) == nil {
		t.Error("intent slot should be set")
	}

	// Add resolution slot (non-overwrite — should preserve intent)
	env2 := NewEnvelope()
	env2.Set(SlotResolution, []float32{3.0, 4.0}, "model-b")
	env2.Set(SlotIntent, []float32{9.0, 9.0}, "model-b") // Should NOT overwrite
	if err := store.UpdateMessageEnvelope(msg.ID, env2, false); err != nil {
		t.Fatalf("update envelope 2 failed: %v", err)
	}

	got2, _ := store.GetInboxMessage(msg.ID)
	if got2.Envelope.Get(SlotResolution) == nil {
		t.Error("resolution slot should be added")
	}
	if got2.Envelope.Get(SlotIntent).Vector[0] != 1.0 {
		t.Error("intent should not be overwritten in non-overwrite mode")
	}

	// Overwrite mode
	env3 := NewEnvelope()
	env3.Set(SlotIntent, []float32{5.0, 6.0}, "model-c")
	if err := store.UpdateMessageEnvelope(msg.ID, env3, true); err != nil {
		t.Fatalf("overwrite envelope failed: %v", err)
	}

	got3, _ := store.GetInboxMessage(msg.ID)
	if got3.Envelope.Get(SlotIntent).Vector[0] != 5.0 {
		t.Error("intent should be overwritten in overwrite mode")
	}
	// Resolution should still be there (only intent was in env3)
	if got3.Envelope.Get(SlotResolution) == nil {
		t.Error("resolution should still exist after overwrite of intent only")
	}
}

// Helper to create store (uses NewStore from store.go)
func init() {
	// Ensure temp directory exists for tests
	os.MkdirAll(os.TempDir(), 0755)
}

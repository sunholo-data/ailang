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

// Helper to create store (uses NewStore from store.go)
func init() {
	// Ensure temp directory exists for tests
	os.MkdirAll(os.TempDir(), 0755)
}

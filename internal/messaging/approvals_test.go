package messaging

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create a thread first
	thread, err := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create approval request
	effectDelta := &EffectDelta{
		CapType:     "FS",
		Paths:       []string{"src/", "docs/"},
		BudgetDelta: 0.50,
	}

	approval, err := store.CreateApproval(thread.ID, "agent1", effectDelta, "Read source files", "low", 0.50)
	if err != nil {
		t.Fatalf("CreateApproval failed: %v", err)
	}

	if approval.ID == "" {
		t.Error("Expected approval ID to be set")
	}
	if approval.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", approval.Status)
	}
	if approval.Proposal != "Read source files" {
		t.Errorf("Expected proposal 'Read source files', got %s", approval.Proposal)
	}

	// Verify effect delta JSON
	var delta EffectDelta
	if err := json.Unmarshal([]byte(approval.EffectDeltaJSON), &delta); err != nil {
		t.Fatalf("Failed to unmarshal effect delta: %v", err)
	}
	if delta.CapType != "FS" {
		t.Errorf("Expected cap_type 'FS', got %s", delta.CapType)
	}
	if len(delta.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(delta.Paths))
	}
}

func TestGetApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create thread and approval
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "IO", Paths: []string{}, BudgetDelta: 1.0}
	created, _ := store.CreateApproval(thread.ID, "agent1", effectDelta, "Test proposal", "medium", 1.0)

	// Retrieve approval
	approval, err := store.GetApproval(created.ID)
	if err != nil {
		t.Fatalf("GetApproval failed: %v", err)
	}

	if approval.ID != created.ID {
		t.Errorf("Expected approval ID %s, got %s", created.ID, approval.ID)
	}
	if approval.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", approval.Status)
	}
}

func TestGetApprovalNonExistent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetApproval("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent approval, got nil")
	}
}

func TestGetApprovalsByStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create thread and multiple approvals
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{}, BudgetDelta: 0.5}

	// Create 3 pending approvals
	for i := 0; i < 3; i++ {
		_, err := store.CreateApproval(thread.ID, "agent1", effectDelta, "Test proposal", "low", 0.5)
		if err != nil {
			t.Fatalf("Failed to create approval: %v", err)
		}
	}

	// Retrieve pending approvals
	approvals, err := store.GetApprovalsByStatus("pending", 0)
	if err != nil {
		t.Fatalf("GetApprovalsByStatus failed: %v", err)
	}

	if len(approvals) != 3 {
		t.Errorf("Expected 3 pending approvals, got %d", len(approvals))
	}

	// Test with limit
	limitedApprovals, err := store.GetApprovalsByStatus("pending", 2)
	if err != nil {
		t.Fatalf("GetApprovalsByStatus with limit failed: %v", err)
	}

	if len(limitedApprovals) != 2 {
		t.Errorf("Expected 2 approvals with limit, got %d", len(limitedApprovals))
	}
}

func TestApproveApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create thread and approval
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{"src/"}, BudgetDelta: 0.5}
	approval, _ := store.CreateApproval(thread.ID, "agent1", effectDelta, "Test proposal", "low", 0.5)

	// Approve the approval
	err = store.ApproveApproval(approval.ID, "user1", "Looks good", 24*time.Hour)
	if err != nil {
		t.Fatalf("ApproveApproval failed: %v", err)
	}

	// Verify approval status updated
	updated, err := store.GetApproval(approval.ID)
	if err != nil {
		t.Fatalf("GetApproval failed: %v", err)
	}

	if updated.Status != "approved" {
		t.Errorf("Expected status 'approved', got %s", updated.Status)
	}
	if updated.ReviewedBy != "user1" {
		t.Errorf("Expected reviewed_by 'user1', got %s", updated.ReviewedBy)
	}
	if updated.ReviewNotes != "Looks good" {
		t.Errorf("Expected review_notes 'Looks good', got %s", updated.ReviewNotes)
	}
	if updated.CapabilityToken == "" {
		t.Error("Expected capability token to be generated")
	}
}

func TestApproveApprovalAlreadyProcessed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create and approve
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "IO", Paths: []string{}, BudgetDelta: 1.0}
	approval, _ := store.CreateApproval(thread.ID, "agent1", effectDelta, "Test", "low", 1.0)
	_ = store.ApproveApproval(approval.ID, "user1", "OK", 24*time.Hour)

	// Try to approve again
	err = store.ApproveApproval(approval.ID, "user2", "Also OK", 24*time.Hour)
	if err == nil {
		t.Error("Expected error when approving already approved approval")
	}
}

func TestRejectApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// Create thread and approval
	thread, _ := store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "Net", Paths: []string{}, BudgetDelta: 5.0}
	approval, _ := store.CreateApproval(thread.ID, "agent1", effectDelta, "Expensive operation", "high", 5.0)

	// Reject the approval
	err = store.RejectApproval(approval.ID, "user1", "Too expensive")
	if err != nil {
		t.Fatalf("RejectApproval failed: %v", err)
	}

	// Verify approval status updated
	updated, err := store.GetApproval(approval.ID)
	if err != nil {
		t.Fatalf("GetApproval failed: %v", err)
	}

	if updated.Status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", updated.Status)
	}
	if updated.ReviewedBy != "user1" {
		t.Errorf("Expected reviewed_by 'user1', got %s", updated.ReviewedBy)
	}
	if updated.ReviewNotes != "Too expensive" {
		t.Errorf("Expected review_notes 'Too expensive', got %s", updated.ReviewNotes)
	}
	if updated.CapabilityToken != "" {
		t.Error("Expected no capability token for rejected approval")
	}
}

func TestGenerateAndVerifyCapabilityToken(t *testing.T) {
	threadID := "thread_123"
	instanceID := "agent1"
	approvalID := "approval_456"
	effectJSON := `{"cap_type":"FS","paths":["src/"],"budget_delta":0.5}`

	// Generate token
	tokenString, expiresAt, err := generateCapabilityToken(threadID, instanceID, approvalID, effectJSON, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if tokenString == "" {
		t.Error("Expected non-empty token string")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("Expected expiry time to be in the future")
	}

	// Verify token
	token, err := VerifyCapabilityToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	if token.ThreadID != threadID {
		t.Errorf("Expected thread_id %s, got %s", threadID, token.ThreadID)
	}
	if token.InstanceID != instanceID {
		t.Errorf("Expected instance_id %s, got %s", instanceID, token.InstanceID)
	}
	if token.ApprovalID != approvalID {
		t.Errorf("Expected approval_id %s, got %s", approvalID, token.ApprovalID)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	threadID := "thread_123"
	instanceID := "agent1"
	approvalID := "approval_456"
	effectJSON := `{"cap_type":"FS","paths":["src/"],"budget_delta":0.5}`

	// Generate token with negative duration (expired immediately)
	tokenString, _, err := generateCapabilityToken(threadID, instanceID, approvalID, effectJSON, -1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to verify expired token
	_, err = VerifyCapabilityToken(tokenString)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	// Test with invalid base64
	_, err := VerifyCapabilityToken("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}

	// Test with valid base64 but invalid JSON
	invalidToken := "aW52YWxpZCBqc29u" // "invalid json" in base64
	_, err = VerifyCapabilityToken(invalidToken)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	threadID := "thread_123"
	instanceID := "agent1"
	approvalID := "approval_456"
	effectJSON := `{"cap_type":"FS","paths":["src/"],"budget_delta":0.5}`

	// Generate valid token
	tokenString, _, err := generateCapabilityToken(threadID, instanceID, approvalID, effectJSON, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Decode and modify the token
	tokenBytes, _ := base64.StdEncoding.DecodeString(tokenString)
	var token CapabilityToken
	_ = json.Unmarshal(tokenBytes, &token)

	// Tamper with the token (change instance_id)
	token.InstanceID = "evil_agent"
	tamperedBytes, _ := json.Marshal(token)
	tamperedTokenString := base64.StdEncoding.EncodeToString(tamperedBytes)

	// Try to verify tampered token
	_, err = VerifyCapabilityToken(tamperedTokenString)
	if err == nil {
		t.Error("Expected error for tampered token, got nil")
	}
}

func TestTokenSecretFromEnv(t *testing.T) {
	// Set custom secret
	customSecret := "my-custom-secret-key"
	os.Setenv("AILANG_TOKEN_SECRET", customSecret)
	defer os.Unsetenv("AILANG_TOKEN_SECRET")

	secret := getSigningSecret()
	if string(secret) != customSecret {
		t.Errorf("Expected secret %s, got %s", customSecret, string(secret))
	}
}

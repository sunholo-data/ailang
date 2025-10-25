package agentprotocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMessageSigner_SignAndVerify(t *testing.T) {
	tempDir := t.TempDir()
	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Create test message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-123",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload: map[string]interface{}{
			"message": "Hello, world!",
		},
	}

	// Sign message
	if err := signer.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Verify signature fields are set
	if env.Signature == "" {
		t.Error("Signature is empty")
	}
	if env.KID == "" {
		t.Error("KID is empty")
	}
	if env.SignatureAlg != "hmac-sha256" {
		t.Errorf("SignatureAlg = %s, want hmac-sha256", env.SignatureAlg)
	}

	// Verify message
	if err := signer.VerifyMessage(env); err != nil {
		t.Errorf("VerifyMessage failed: %v", err)
	}
}

func TestMessageSigner_VerifyTamperedMessage(t *testing.T) {
	tempDir := t.TempDir()
	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Create and sign message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-456",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload: map[string]interface{}{
			"message": "Original message",
		},
	}

	if err := signer.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Tamper with message
	env.Payload["message"] = "Tampered message"

	// Verification should fail
	err = signer.VerifyMessage(env)
	if err == nil {
		t.Error("Expected verification to fail for tampered message")
	}
	if err != nil && err.Error() != "signature verification failed" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestMessageSigner_VerifyMissingSignature(t *testing.T) {
	tempDir := t.TempDir()
	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Create unsigned message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-789",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	// Verification should fail
	err = signer.VerifyMessage(env)
	if err == nil {
		t.Error("Expected verification to fail for unsigned message")
	}
}

func TestMessageSigner_KeyPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create first signer
	signer1, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	kid1 := signer1.GetKeyID()

	// Create second signer (should load same key)
	signer2, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	kid2 := signer2.GetKeyID()

	// Key IDs should match
	if kid1 != kid2 {
		t.Errorf("Key ID mismatch: %s != %s", kid1, kid2)
	}

	// Sign with first signer, verify with second
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-persistence",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if err := signer1.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	if err := signer2.VerifyMessage(env); err != nil {
		t.Errorf("VerifyMessage failed with second signer: %v", err)
	}
}

func TestMessageSigner_KeyRotation(t *testing.T) {
	tempDir := t.TempDir()

	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Get original key ID
	originalKID := signer.GetKeyID()

	// Rotate key
	if err := signer.RotateKey(); err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	// Get new key ID
	newKID := signer.GetKeyID()

	// Key IDs should be different
	if originalKID == newKID {
		t.Error("Key ID should change after rotation")
	}

	// Sign message with new key
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-rotation",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if err := signer.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Verify KID matches new key
	if env.KID != newKID {
		t.Errorf("Message KID = %s, want %s", env.KID, newKID)
	}

	// Verification should succeed
	if err := signer.VerifyMessage(env); err != nil {
		t.Errorf("VerifyMessage failed: %v", err)
	}
}

func TestMessageSigner_KeyFileFormat(t *testing.T) {
	tempDir := t.TempDir()

	_, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Read key file
	keyPath := filepath.Join(tempDir, "signing_key.json")
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	// Parse key
	var key SigningKey
	if err := json.Unmarshal(data, &key); err != nil {
		t.Fatalf("Failed to unmarshal key: %v", err)
	}

	// Verify key structure
	if key.KID == "" {
		t.Error("Key ID is empty")
	}
	if key.Algorithm != "hmac-sha256" {
		t.Errorf("Algorithm = %s, want hmac-sha256", key.Algorithm)
	}
	if len(key.Secret) != 32 {
		t.Errorf("Secret length = %d, want 32", len(key.Secret))
	}
	if key.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// Verify KID format (should be "key-" followed by hex)
	if len(key.KID) < 5 || key.KID[:4] != "key-" {
		t.Errorf("Invalid KID format: %s", key.KID)
	}
}

func TestMessageSigner_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	_, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Check key file permissions
	keyPath := filepath.Join(tempDir, "signing_key.json")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	// Permissions should be 0600 (owner read/write only)
	expectedPerms := os.FileMode(0600)
	actualPerms := info.Mode().Perm()

	if actualPerms != expectedPerms {
		t.Errorf("Key file permissions = %o, want %o", actualPerms, expectedPerms)
	}
}

func TestMessageSigner_CanonicalRepresentation(t *testing.T) {
	tempDir := t.TempDir()
	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Create message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-canonical",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	// Get canonical representation
	canonical, err := signer.canonicalRepresentation(env)
	if err != nil {
		t.Fatalf("canonicalRepresentation failed: %v", err)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(canonical, &parsed); err != nil {
		t.Fatalf("Canonical representation is not valid JSON: %v", err)
	}

	// Should not include signature fields
	if _, ok := parsed["signature"]; ok {
		t.Error("Canonical representation includes signature field")
	}
	if _, ok := parsed["signature_alg"]; ok {
		t.Error("Canonical representation includes signature_alg field")
	}
	if _, ok := parsed["kid"]; ok {
		t.Error("Canonical representation includes kid field")
	}

	// Should include other fields
	if parsed["message_id"] != env.MessageID {
		t.Error("Canonical representation missing message_id")
	}
	if parsed["from_agent"] != env.FromAgent {
		t.Error("Canonical representation missing from_agent")
	}
}

func TestMessageSigner_DeterministicSignature(t *testing.T) {
	tempDir := t.TempDir()
	signer, err := NewMessageSigner(tempDir)
	if err != nil {
		t.Fatalf("NewMessageSigner failed: %v", err)
	}

	// Create message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-deterministic",
		CorrelationID:   "cycle-test",
		TraceID:         "trace-test",
		Timestamp:       "2025-10-25T12:00:00Z", // Fixed timestamp
		TTLSeconds:      300,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		PayloadSchema:   "test.v1",
		Payload: map[string]interface{}{
			"message": "Test",
		},
	}

	// Sign message
	if err := signer.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}
	signature1 := env.Signature

	// Clear signature and sign again
	env.Signature = ""
	env.SignatureAlg = ""
	env.KID = ""

	if err := signer.SignMessage(env); err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}
	signature2 := env.Signature

	// Signatures should be identical (deterministic)
	if signature1 != signature2 {
		t.Errorf("Signatures differ:\n%s\n%s", signature1, signature2)
	}
}

package agentprotocol

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactStore_StoreAndRetrieve(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	// Test data
	content := []byte("# Test Design Doc\n\nThis is a test document.")
	originalPath := "design_docs/planned/M-TEST-123.md"
	mimeType := "text/markdown"

	// Store artifact
	hash, err := store.StoreArtifact(originalPath, content, mimeType)
	if err != nil {
		t.Fatalf("StoreArtifact failed: %v", err)
	}

	// Verify hash format
	if len(hash) != 71 { // "sha256:" + 64 hex chars
		t.Errorf("Invalid hash format: %s (length %d)", hash, len(hash))
	}
	if hash[:7] != "sha256:" {
		t.Errorf("Hash doesn't start with 'sha256:': %s", hash)
	}

	// Retrieve artifact
	retrievedContent, metadata, err := store.RetrieveArtifact(hash)
	if err != nil {
		t.Fatalf("RetrieveArtifact failed: %v", err)
	}

	// Verify content matches
	if string(retrievedContent) != string(content) {
		t.Errorf("Content mismatch:\nGot: %s\nWant: %s", retrievedContent, content)
	}

	// Verify metadata
	if metadata.OriginalPath != originalPath {
		t.Errorf("Original path mismatch: got %s, want %s", metadata.OriginalPath, originalPath)
	}
	if metadata.MimeType != mimeType {
		t.Errorf("MIME type mismatch: got %s, want %s", metadata.MimeType, mimeType)
	}
	if metadata.Size != int64(len(content)) {
		t.Errorf("Size mismatch: got %d, want %d", metadata.Size, len(content))
	}
	if metadata.Hash != hash {
		t.Errorf("Hash mismatch: got %s, want %s", metadata.Hash, hash)
	}
}

func TestArtifactStore_Deduplication(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	content := []byte("Test content for deduplication")

	// Store the same content twice
	hash1, err := store.StoreArtifact("path1.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("First StoreArtifact failed: %v", err)
	}

	hash2, err := store.StoreArtifact("path2.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("Second StoreArtifact failed: %v", err)
	}

	// Hashes should be identical (content-addressed)
	if hash1 != hash2 {
		t.Errorf("Deduplication failed: hash1=%s, hash2=%s", hash1, hash2)
	}

	// Verify only one content file exists
	hashHex := hash1[7:] // Remove "sha256:" prefix
	contentPath := filepath.Join(tempDir, "artifacts", "sha256", hashHex, "content")

	info, err := os.Stat(contentPath)
	if err != nil {
		t.Fatalf("Content file doesn't exist: %v", err)
	}
	if info.Size() != int64(len(content)) {
		t.Errorf("Content file size mismatch: got %d, want %d", info.Size(), len(content))
	}
}

func TestArtifactStore_StoreFromFile(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.md")
	content := []byte("# Test File\n\nThis is a test.")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Store from file
	hash, err := store.StoreArtifactFromFile(testFile, "")
	if err != nil {
		t.Fatalf("StoreArtifactFromFile failed: %v", err)
	}

	// Retrieve and verify
	retrievedContent, metadata, err := store.RetrieveArtifact(hash)
	if err != nil {
		t.Fatalf("RetrieveArtifact failed: %v", err)
	}

	if string(retrievedContent) != string(content) {
		t.Errorf("Content mismatch")
	}

	// MIME type should be auto-detected as text/markdown
	if metadata.MimeType != "text/markdown" {
		t.Errorf("MIME type mismatch: got %s, want text/markdown", metadata.MimeType)
	}
}

func TestArtifactStore_HashVerification(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	content := []byte("Original content")
	hash, err := store.StoreArtifact("test.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("StoreArtifact failed: %v", err)
	}

	// Corrupt the content file
	hashHex := hash[7:]
	contentPath := filepath.Join(tempDir, "artifacts", "sha256", hashHex, "content")
	corrupted := []byte("Corrupted content")
	if err := os.WriteFile(contentPath, corrupted, 0644); err != nil {
		t.Fatalf("Failed to corrupt content: %v", err)
	}

	// Retrieve should fail with hash mismatch
	_, _, err = store.RetrieveArtifact(hash)
	if err == nil {
		t.Error("Expected error for hash mismatch, got nil")
	}
	if err != nil && err.Error()[:13] != "hash mismatch" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestArtifactStore_ArtifactExists(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	content := []byte("Test content")
	hash, err := store.StoreArtifact("test.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("StoreArtifact failed: %v", err)
	}

	// Check existence
	exists, err := store.ArtifactExists(hash)
	if err != nil {
		t.Fatalf("ArtifactExists failed: %v", err)
	}
	if !exists {
		t.Error("Artifact should exist")
	}

	// Check non-existent artifact
	fakeHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	exists, err = store.ArtifactExists(fakeHash)
	if err != nil {
		t.Fatalf("ArtifactExists failed for fake hash: %v", err)
	}
	if exists {
		t.Error("Fake artifact should not exist")
	}
}

func TestArtifactStore_ListArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	// Store multiple artifacts
	hashes := make([]string, 3)
	for i := 0; i < 3; i++ {
		content := []byte(string(rune('A' + i)))
		hash, err := store.StoreArtifact("test.txt", content, "text/plain")
		if err != nil {
			t.Fatalf("StoreArtifact failed: %v", err)
		}
		hashes[i] = hash
	}

	// List artifacts
	listed, err := store.ListArtifacts()
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("Expected 3 artifacts, got %d", len(listed))
	}

	// Verify all hashes are present
	hashSet := make(map[string]bool)
	for _, h := range listed {
		hashSet[h] = true
	}
	for _, h := range hashes {
		if !hashSet[h] {
			t.Errorf("Hash %s not in list", h)
		}
	}
}

func TestArtifactStore_CopyToFile(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	content := []byte("Test content for copy")
	hash, err := store.StoreArtifact("original.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("StoreArtifact failed: %v", err)
	}

	// Copy to new location
	destPath := filepath.Join(tempDir, "restored.txt")
	if err := store.CopyArtifactToFile(hash, destPath); err != nil {
		t.Fatalf("CopyArtifactToFile failed: %v", err)
	}

	// Verify destination file
	restored, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restored) != string(content) {
		t.Errorf("Content mismatch after copy")
	}
}

func TestArtifactStore_DeleteArtifact(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)

	content := []byte("Test content")
	hash, err := store.StoreArtifact("test.txt", content, "text/plain")
	if err != nil {
		t.Fatalf("StoreArtifact failed: %v", err)
	}

	// Delete artifact
	if err := store.DeleteArtifact(hash); err != nil {
		t.Fatalf("DeleteArtifact failed: %v", err)
	}

	// Verify it's gone
	exists, err := store.ArtifactExists(hash)
	if err != nil {
		t.Fatalf("ArtifactExists failed: %v", err)
	}
	if exists {
		t.Error("Artifact should not exist after deletion")
	}
}

func TestComputeHash(t *testing.T) {
	content := []byte("Test content")
	hash := ComputeHash(content)

	// Verify format
	if len(hash) != 71 {
		t.Errorf("Invalid hash length: %d", len(hash))
	}
	if hash[:7] != "sha256:" {
		t.Errorf("Hash doesn't start with 'sha256:'")
	}

	// Verify consistency
	hash2 := ComputeHash(content)
	if hash != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Verify different content gives different hash
	hash3 := ComputeHash([]byte("Different content"))
	if hash == hash3 {
		t.Error("Different content should give different hash")
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path      string
		shouldErr bool
	}{
		{"design_docs/planned/M-TEST.md", false},
		{"../etc/passwd", true},
		{"foo/../bar", true},
		{"/absolute/path", false}, // Absolute paths are allowed
		{"normal/path.txt", false},
	}

	for _, tt := range tests {
		err := validatePath(tt.path)
		if tt.shouldErr && err == nil {
			t.Errorf("Expected error for path %s, got nil", tt.path)
		}
		if !tt.shouldErr && err != nil {
			t.Errorf("Unexpected error for path %s: %v", tt.path, err)
		}
	}
}

func TestGuessMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"test.md", "text/markdown"},
		{"test.json", "application/json"},
		{"test.yaml", "application/yaml"},
		{"test.go", "text/x-go"},
		{"test.ail", "text/x-ailang"},
		{"test.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := guessMimeType(tt.path)
		if got != tt.expected {
			t.Errorf("guessMimeType(%s) = %s, want %s", tt.path, got, tt.expected)
		}
	}
}

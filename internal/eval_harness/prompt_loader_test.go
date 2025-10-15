package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewPromptLoader(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"test-v1": {
				File:        "prompts/test.md",
				Hash:        "abc123",
				Description: "Test prompt",
				Created:     "2025-01-01",
				Tags:        []string{"test"},
				Notes:       "Test notes",
			},
		},
		Active: "test-v1",
		Notes:  []string{"Test registry"},
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading
	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	if loader.registry.SchemaVersion != "1.0" {
		t.Errorf("Expected schema version 1.0, got %s", loader.registry.SchemaVersion)
	}

	if loader.registry.Active != "test-v1" {
		t.Errorf("Expected active version test-v1, got %s", loader.registry.Active)
	}

	if len(loader.registry.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(loader.registry.Versions))
	}
}

func TestLoadPrompt_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test prompt file
	promptContent := "# Test Prompt\n\nThis is a test prompt."
	promptPath := filepath.Join(promptsDir, "test.md")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute hash
	hash, err := ComputePromptHash(promptPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"test-v1": {
				File: "prompts/test.md",
				Hash: hash,
			},
		},
		Active: "test-v1",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load prompt
	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	content, err := loader.LoadPrompt("test-v1")
	if err != nil {
		t.Fatalf("Failed to load prompt: %v", err)
	}

	if content != promptContent {
		t.Errorf("Expected content %q, got %q", promptContent, content)
	}
}

func TestLoadPrompt_HashMismatch(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test prompt file
	promptContent := "# Test Prompt\n\nThis is a test prompt."
	promptPath := filepath.Join(promptsDir, "test.md")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create registry with WRONG hash
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"test-v1": {
				File: "prompts/test.md",
				Hash: "wronghash123", // Intentionally wrong
			},
		},
		Active: "test-v1",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load prompt should fail
	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = loader.LoadPrompt("test-v1")
	if err == nil {
		t.Fatal("Expected hash mismatch error, got nil")
	}

	if !contains(err.Error(), "hash mismatch") {
		t.Errorf("Expected 'hash mismatch' error, got: %v", err)
	}
}

func TestLoadPrompt_PlaceholderHash(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test prompt file
	promptContent := "# Test Prompt\n\nThis is a test prompt."
	promptPath := filepath.Join(promptsDir, "test.md")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create registry with PLACEHOLDER hash (should skip verification)
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"test-v1": {
				File: "prompts/test.md",
				Hash: "PLACEHOLDER",
			},
		},
		Active: "test-v1",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load prompt should succeed despite hash mismatch
	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	content, err := loader.LoadPrompt("test-v1")
	if err != nil {
		t.Fatalf("Failed to load prompt with PLACEHOLDER hash: %v", err)
	}

	if content != promptContent {
		t.Errorf("Expected content %q, got %q", promptContent, content)
	}
}

func TestLoadPrompt_NotFound(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create empty registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions:      map[string]PromptVersion{},
		Active:        "",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	// Try to load non-existent version
	_, err = loader.LoadPrompt("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent version, got nil")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetActivePrompt(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test prompt file
	promptContent := "# Active Prompt\n\nThis is the active prompt."
	promptPath := filepath.Join(promptsDir, "active.md")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compute hash
	hash, err := ComputePromptHash(promptPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"active-v1": {
				File: "prompts/active.md",
				Hash: hash,
			},
		},
		Active: "active-v1",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load active prompt
	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	content, err := loader.GetActivePrompt()
	if err != nil {
		t.Fatalf("Failed to load active prompt: %v", err)
	}

	if content != promptContent {
		t.Errorf("Expected content %q, got %q", promptContent, content)
	}
}

func TestGetVersion(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"test-v1": {
				File:        "prompts/test.md",
				Hash:        "abc123",
				Description: "Test prompt",
				Created:     "2025-01-01",
				Tags:        []string{"test", "experimental"},
				Notes:       "Test notes",
			},
		},
		Active: "test-v1",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	// Get version metadata
	version, err := loader.GetVersion("test-v1")
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	if version.Description != "Test prompt" {
		t.Errorf("Expected description 'Test prompt', got %q", version.Description)
	}

	if len(version.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(version.Tags))
	}
}

func TestListVersions(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create registry with multiple versions
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"v1": {File: "prompts/v1.md", Hash: "hash1"},
			"v2": {File: "prompts/v2.md", Hash: "hash2"},
			"v3": {File: "prompts/v3.md", Hash: "hash3"},
		},
		Active: "v2",
	}

	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	loader, err := NewPromptLoader(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	// List all versions
	versions := loader.ListVersions()
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	if _, exists := versions["v1"]; !exists {
		t.Error("Expected v1 to exist")
	}
	if _, exists := versions["v2"]; !exists {
		t.Error("Expected v2 to exist")
	}
	if _, exists := versions["v3"]; !exists {
		t.Error("Expected v3 to exist")
	}
}

func TestComputePromptHash(t *testing.T) {
	// Create temporary file
	tmpFile := filepath.Join(t.TempDir(), "test.md")
	content := "# Test\nContent"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hash1, err := ComputePromptHash(tmpFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	// Compute again - should be deterministic
	hash2, err := ComputePromptHash(tmpFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Expected deterministic hash, got %s and %s", hash1, hash2)
	}

	// Different content should produce different hash
	if err := os.WriteFile(tmpFile, []byte("different content"), 0644); err != nil {
		t.Fatal(err)
	}

	hash3, err := ComputePromptHash(tmpFile)
	if err != nil {
		t.Fatalf("Failed to compute hash: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Expected different hash for different content")
	}
}

// Helper function for substring matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

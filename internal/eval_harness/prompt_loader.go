package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PromptVersion represents metadata about a prompt version
type PromptVersion struct {
	File        string   `json:"file"`
	Hash        string   `json:"hash"`
	Description string   `json:"description"`
	Created     string   `json:"created"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes"`
}

// PromptRegistry contains all registered prompt versions
type PromptRegistry struct {
	SchemaVersion string                   `json:"schema_version"`
	Versions      map[string]PromptVersion `json:"versions"`
	Active        string                   `json:"active"`
	Notes         []string                 `json:"notes"`
}

// PromptLoader loads and verifies prompt versions
type PromptLoader struct {
	registry *PromptRegistry
	rootDir  string // Root directory for resolving relative paths
}

// NewPromptLoader creates a loader from versions.json
func NewPromptLoader(registryPath string) (*PromptLoader, error) {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry PromptRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	// Determine root directory from registry path
	rootDir := filepath.Dir(registryPath)
	if filepath.Base(rootDir) == "prompts" {
		rootDir = filepath.Dir(rootDir) // Go up one level to project root
	}

	return &PromptLoader{
		registry: &registry,
		rootDir:  rootDir,
	}, nil
}

// LoadPrompt loads a prompt by version ID with hash verification
func (l *PromptLoader) LoadPrompt(versionID string) (string, error) {
	version, exists := l.registry.Versions[versionID]
	if !exists {
		return "", fmt.Errorf("prompt version %q not found in registry", versionID)
	}

	// Resolve file path relative to root directory
	promptPath := filepath.Join(l.rootDir, version.File)

	// Read prompt content
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt %q: %w", version.File, err)
	}

	// Verify hash (skip if placeholder)
	if version.Hash != "PLACEHOLDER" {
		actualHash := computeSHA256(content)
		if actualHash != version.Hash {
			// Truncate hashes for error message (safely handle short hashes)
			expectedPreview := version.Hash
			if len(expectedPreview) > 16 {
				expectedPreview = expectedPreview[:16] + "..."
			}
			actualPreview := actualHash
			if len(actualPreview) > 16 {
				actualPreview = actualPreview[:16] + "..."
			}
			return "", fmt.Errorf("hash mismatch for %q: expected %s, got %s (file may have been modified)",
				versionID, expectedPreview, actualPreview)
		}
	}

	return string(content), nil
}

// GetActivePrompt loads the active prompt version
func (l *PromptLoader) GetActivePrompt() (string, error) {
	if l.registry.Active == "" {
		return "", fmt.Errorf("no active prompt version specified in registry")
	}
	return l.LoadPrompt(l.registry.Active)
}

// GetVersion returns metadata for a specific version
func (l *PromptLoader) GetVersion(versionID string) (*PromptVersion, error) {
	version, exists := l.registry.Versions[versionID]
	if !exists {
		return nil, fmt.Errorf("prompt version %q not found", versionID)
	}
	return &version, nil
}

// ListVersions returns all available prompt versions
func (l *PromptLoader) ListVersions() map[string]PromptVersion {
	return l.registry.Versions
}

// computeSHA256 calculates the SHA256 hash of content
func computeSHA256(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// ComputePromptHash is a helper to compute hash for a prompt file (for updating registry)
func ComputePromptHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return computeSHA256(content), nil
}

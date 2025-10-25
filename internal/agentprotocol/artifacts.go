package agentprotocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArtifactMetadata describes a stored artifact.
type ArtifactMetadata struct {
	// OriginalPath is the path where the content came from
	OriginalPath string `json:"original_path"`

	// MimeType is the content type
	MimeType string `json:"mime_type"`

	// Size is the content size in bytes
	Size int64 `json:"size"`

	// Hash is the SHA256 hash (same as directory name)
	Hash string `json:"hash"`

	// CreatedAt is when the artifact was stored
	CreatedAt time.Time `json:"created_at"`
}

// ArtifactStore manages content-addressed artifact storage.
//
// Directory structure:
//
//	.ailang/state/artifacts/sha256/<hash>/
//	  ├── content        # The actual file content
//	  └── metadata.json  # ArtifactMetadata
type ArtifactStore struct {
	stateDir string // Root directory (.ailang/state)
}

// NewArtifactStore creates a new artifact store.
func NewArtifactStore(stateDir string) *ArtifactStore {
	return &ArtifactStore{
		stateDir: stateDir,
	}
}

// StoreArtifact stores content in the artifact store and returns its hash.
//
// The content is addressed by its SHA256 hash, enabling deduplication.
// If the same content is stored twice, only one copy is kept.
//
// Parameters:
//   - originalPath: The original file path (for metadata)
//   - content: The file content to store
//   - mimeType: The content type
//
// Returns:
//   - hash: The SHA256 hash in format "sha256:abc123..."
//   - error: Any error that occurred
func (s *ArtifactStore) StoreArtifact(originalPath string, content []byte, mimeType string) (string, error) {
	// Sanitize path to prevent directory traversal
	if err := validatePath(originalPath); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])
	hashStr := "sha256:" + hashHex

	// Create artifact directory
	artifactDir := filepath.Join(s.stateDir, "artifacts", "sha256", hashHex)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Write content (if not already exists)
	contentPath := filepath.Join(artifactDir, "content")
	if _, err := os.Stat(contentPath); os.IsNotExist(err) {
		// Content doesn't exist yet, write it
		if err := os.WriteFile(contentPath, content, 0644); err != nil {
			return "", fmt.Errorf("failed to write content: %w", err)
		}
	}

	// Write metadata (always update to track latest access)
	metadata := &ArtifactMetadata{
		OriginalPath: originalPath,
		MimeType:     mimeType,
		Size:         int64(len(content)),
		Hash:         hashStr,
		CreatedAt:    time.Now().UTC(),
	}

	metadataPath := filepath.Join(artifactDir, "metadata.json")
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	return hashStr, nil
}

// StoreArtifactFromFile reads a file and stores it as an artifact.
//
// This is a convenience wrapper around StoreArtifact that reads from disk.
//
// Parameters:
//   - filePath: Path to the file to store
//   - mimeType: The content type (if empty, guessed from extension)
//
// Returns:
//   - hash: The SHA256 hash in format "sha256:abc123..."
//   - error: Any error that occurred
func (s *ArtifactStore) StoreArtifactFromFile(filePath string, mimeType string) (string, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Guess MIME type if not provided
	if mimeType == "" {
		mimeType = guessMimeType(filePath)
	}

	return s.StoreArtifact(filePath, content, mimeType)
}

// RetrieveArtifact retrieves content and metadata for a given hash.
//
// Parameters:
//   - hash: The SHA256 hash in format "sha256:abc123..."
//
// Returns:
//   - content: The stored content
//   - metadata: The artifact metadata
//   - error: Any error that occurred
func (s *ArtifactStore) RetrieveArtifact(hash string) ([]byte, *ArtifactMetadata, error) {
	// Extract hex hash
	hashHex, err := extractHashHex(hash)
	if err != nil {
		return nil, nil, err
	}

	// Build paths
	artifactDir := filepath.Join(s.stateDir, "artifacts", "sha256", hashHex)
	contentPath := filepath.Join(artifactDir, "content")
	metadataPath := filepath.Join(artifactDir, "metadata.json")

	// Read content
	content, err := os.ReadFile(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("artifact not found: %s", hash)
		}
		return nil, nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Verify hash
	computedHash := sha256.Sum256(content)
	computedHashHex := hex.EncodeToString(computedHash[:])
	if computedHashHex != hashHex {
		return nil, nil, fmt.Errorf("hash mismatch: expected %s, got %s", hashHex, computedHashHex)
	}

	// Read metadata
	metadataJSON, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata ArtifactMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return content, &metadata, nil
}

// ArtifactExists checks if an artifact exists for a given hash.
func (s *ArtifactStore) ArtifactExists(hash string) (bool, error) {
	hashHex, err := extractHashHex(hash)
	if err != nil {
		return false, err
	}

	contentPath := filepath.Join(s.stateDir, "artifacts", "sha256", hashHex, "content")
	_, err = os.Stat(contentPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListArtifacts returns a list of all stored artifact hashes.
func (s *ArtifactStore) ListArtifacts() ([]string, error) {
	artifactsDir := filepath.Join(s.stateDir, "artifacts", "sha256")

	// Check if directory exists
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		return nil, nil // No artifacts yet
	}

	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifacts directory: %w", err)
	}

	var hashes []string
	for _, entry := range entries {
		if entry.IsDir() {
			hashes = append(hashes, "sha256:"+entry.Name())
		}
	}

	return hashes, nil
}

// DeleteArtifact removes an artifact from storage.
//
// Use with caution - this permanently deletes content.
func (s *ArtifactStore) DeleteArtifact(hash string) error {
	hashHex, err := extractHashHex(hash)
	if err != nil {
		return err
	}

	artifactDir := filepath.Join(s.stateDir, "artifacts", "sha256", hashHex)
	return os.RemoveAll(artifactDir)
}

// CopyArtifactToFile copies an artifact's content to a file.
//
// This is useful for restoring files from the artifact store.
func (s *ArtifactStore) CopyArtifactToFile(hash string, destPath string) error {
	content, _, err := s.RetrieveArtifact(hash)
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write content to destination
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// extractHashHex extracts the hex hash from a "sha256:..." string.
func extractHashHex(hash string) (string, error) {
	if !strings.HasPrefix(hash, "sha256:") {
		return "", fmt.Errorf("invalid hash format (must start with 'sha256:'): %s", hash)
	}

	hashHex := strings.TrimPrefix(hash, "sha256:")
	if len(hashHex) != 64 {
		return "", fmt.Errorf("invalid SHA256 hash length: %d (expected 64)", len(hashHex))
	}

	return hashHex, nil
}

// validatePath checks that a path doesn't contain directory traversal attacks.
func validatePath(path string) error {
	// Reject paths containing ".."
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains '..': %s", path)
	}

	// Check for absolute paths (we only store relative paths in metadata)
	// Note: We allow absolute paths but this is just for validation
	_ = filepath.IsAbs(path) // Intentionally not used - just checking path structure

	return nil
}

// guessMimeType attempts to guess MIME type from file extension.
func guessMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	mimeTypes := map[string]string{
		".md":   "text/markdown",
		".txt":  "text/plain",
		".json": "application/json",
		".yaml": "application/yaml",
		".yml":  "application/yaml",
		".go":   "text/x-go",
		".py":   "text/x-python",
		".js":   "text/javascript",
		".ts":   "text/typescript",
		".html": "text/html",
		".css":  "text/css",
		".xml":  "application/xml",
		".sh":   "application/x-sh",
		".ail":  "text/x-ailang",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	return "application/octet-stream" // Default
}

// ComputeHash computes the SHA256 hash of content without storing it.
// This is useful for verifying content or checking if it's already stored.
func ComputeHash(content []byte) string {
	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])
	return "sha256:" + hashHex
}

// ComputeFileHash computes the SHA256 hash of a file.
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	hashBytes := h.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)
	return "sha256:" + hashHex, nil
}

package prompt

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// embeddedPrompts is set by SetEmbeddedFS (called from main)
var embeddedPrompts fs.FS

// SetEmbeddedFS sets the embedded filesystem for prompts
// This should be called from main() with the embedded FS
func SetEmbeddedFS(efs fs.FS) {
	embeddedPrompts = efs
}

// VersionMetadata represents metadata for a prompt version
type VersionMetadata struct {
	File        string   `json:"file"`
	Hash        string   `json:"hash"`
	Description string   `json:"description"`
	Created     string   `json:"created"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes"`
}

// VersionsManifest represents the versions.json file structure
type VersionsManifest struct {
	SchemaVersion string                     `json:"schema_version"`
	Versions      map[string]VersionMetadata `json:"versions"`
	Active        string                     `json:"active"`
	Notes         []string                   `json:"notes"`
}

// LoadPrompt loads a prompt by version string.
// If version is empty or "latest", uses the active version from versions.json.
// Returns the prompt content as a string.
func LoadPrompt(version string) (string, error) {
	// Load versions.json
	manifest, err := loadVersionsManifest()
	if err != nil {
		return "", fmt.Errorf("failed to load versions manifest: %w", err)
	}

	// Resolve version
	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = manifest.Active
	}

	// Look up version metadata
	metadata, ok := manifest.Versions[targetVersion]
	if !ok {
		return "", fmt.Errorf("version %q not found in versions.json", targetVersion)
	}

	var content []byte

	// Try embedded FS first (works anywhere, bundled in binary)
	if embeddedPrompts != nil {
		var embErr error
		content, embErr = fs.ReadFile(embeddedPrompts, metadata.File)
		if embErr == nil {
			return string(content), nil
		}
	}

	// Fallback to disk (for development - allows editing without rebuild)
	projectRoot := findProjectRoot()
	promptPath := filepath.Join(projectRoot, metadata.File)
	content, err = os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s (tried embedded and disk): %w", metadata.File, err)
	}

	return string(content), nil
}

// LoadPromptWithVersion loads a prompt by version string and returns the resolved version ID.
// If version is empty or "latest", the active version is used and its ID is returned.
// Returns (content, versionUsed, error).
func LoadPromptWithVersion(version string) (string, string, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return "", "", fmt.Errorf("failed to load versions manifest: %w", err)
	}
	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = manifest.Active
	}
	content, err := LoadPrompt(targetVersion)
	if err != nil {
		return "", "", err
	}
	return content, targetVersion, nil
}

// loadVersionsManifest loads the versions.json file
func loadVersionsManifest() (*VersionsManifest, error) {
	var data []byte
	var err error

	// Try embedded FS first (works anywhere, bundled in binary)
	if embeddedPrompts != nil {
		data, err = fs.ReadFile(embeddedPrompts, "prompts/versions.json")
		if err == nil {
			var manifest VersionsManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("failed to parse versions.json: %w", err)
			}
			return &manifest, nil
		}
	}

	// Fallback to disk (for development - allows editing without rebuild)
	projectRoot := findProjectRoot()
	versionFile := filepath.Join(projectRoot, "prompts", "versions.json")
	data, err = os.ReadFile(versionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read versions.json (tried embedded and disk): %w", err)
	}

	var manifest VersionsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse versions.json: %w", err)
	}

	return &manifest, nil
}

// findProjectRoot finds the project root by looking for marker files
func findProjectRoot() string {
	// Look for markers like go.mod, .git
	markers := []string{"go.mod", ".git", "prompts"}

	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		// Check for markers
		for _, marker := range markers {
			path := filepath.Join(dir, marker)
			if _, err := os.Stat(path); err == nil {
				return dir
			}
		}

		// Move up
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Default to current directory
	return "."
}

// GetActiveVersion returns the active version from versions.json
func GetActiveVersion() (string, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return "", err
	}
	return manifest.Active, nil
}

// ListVersions returns all available prompt versions
func ListVersions() ([]string, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(manifest.Versions))
	for version := range manifest.Versions {
		versions = append(versions, version)
	}
	return versions, nil
}

// GetVersionMetadata returns metadata for a specific version
func GetVersionMetadata(version string) (*VersionMetadata, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return nil, err
	}

	// Resolve version
	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = manifest.Active
	}

	metadata, ok := manifest.Versions[targetVersion]
	if !ok {
		return nil, fmt.Errorf("version %q not found in versions.json", targetVersion)
	}

	return &metadata, nil
}

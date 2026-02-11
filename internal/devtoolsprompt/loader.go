package devtoolsprompt

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// embeddedPrompts is set by SetEmbeddedFS (called from main)
// This is the same embedded FS as the syntax prompts — devtools lives under prompts/devtools/
var embeddedPrompts fs.FS

// SetEmbeddedFS sets the embedded filesystem for devtools prompts.
// This should be called from main() with the same embedded FS used for syntax prompts.
func SetEmbeddedFS(efs fs.FS) {
	embeddedPrompts = efs
}

// VersionMetadata represents metadata for a devtools prompt version
type VersionMetadata struct {
	File        string   `json:"file"`
	Hash        string   `json:"hash"`
	Description string   `json:"description"`
	Created     string   `json:"created"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes"`
}

// VersionsManifest represents the devtools versions.json file structure
type VersionsManifest struct {
	SchemaVersion string                     `json:"schema_version"`
	Versions      map[string]VersionMetadata `json:"versions"`
	Active        string                     `json:"active"`
	Notes         []string                   `json:"notes"`
}

// LoadPrompt loads a devtools prompt by version string.
// If version is empty or "latest", uses the active version from versions.json.
func LoadPrompt(version string) (string, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return "", fmt.Errorf("failed to load devtools versions manifest: %w", err)
	}

	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = manifest.Active
	}

	metadata, ok := manifest.Versions[targetVersion]
	if !ok {
		return "", fmt.Errorf("version %q not found in devtools versions.json", targetVersion)
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

	// Fallback to disk (for development — allows editing without rebuild)
	projectRoot := findProjectRoot()
	promptPath := filepath.Join(projectRoot, metadata.File)
	content, err = os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read devtools prompt file %s (tried embedded and disk): %w", metadata.File, err)
	}

	return string(content), nil
}

// loadVersionsManifest loads the devtools versions.json file
func loadVersionsManifest() (*VersionsManifest, error) {
	var data []byte
	var err error

	// Try embedded FS first
	if embeddedPrompts != nil {
		data, err = fs.ReadFile(embeddedPrompts, "prompts/devtools/versions.json")
		if err == nil {
			var manifest VersionsManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("failed to parse devtools versions.json: %w", err)
			}
			return &manifest, nil
		}
	}

	// Fallback to disk
	projectRoot := findProjectRoot()
	versionFile := filepath.Join(projectRoot, "prompts", "devtools", "versions.json")
	data, err = os.ReadFile(versionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read devtools versions.json (tried embedded and disk): %w", err)
	}

	var manifest VersionsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse devtools versions.json: %w", err)
	}

	return &manifest, nil
}

// findProjectRoot finds the project root by looking for marker files
func findProjectRoot() string {
	markers := []string{"go.mod", ".git", "prompts"}

	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		for _, marker := range markers {
			path := filepath.Join(dir, marker)
			if _, err := os.Stat(path); err == nil {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "."
}

// GetActiveVersion returns the active version from devtools versions.json
func GetActiveVersion() (string, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return "", err
	}
	return manifest.Active, nil
}

// ListVersions returns all available devtools prompt versions
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

// GetVersionMetadata returns metadata for a specific devtools prompt version
func GetVersionMetadata(version string) (*VersionMetadata, error) {
	manifest, err := loadVersionsManifest()
	if err != nil {
		return nil, err
	}

	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = manifest.Active
	}

	metadata, ok := manifest.Versions[targetVersion]
	if !ok {
		return nil, fmt.Errorf("version %q not found in devtools versions.json", targetVersion)
	}

	return &metadata, nil
}

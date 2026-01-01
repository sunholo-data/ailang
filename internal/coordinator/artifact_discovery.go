package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ArtifactDiscovery provides deterministic artifact discovery using git diff.
// This is preferred over parsing output markers, as git accurately tracks all file changes.
type ArtifactDiscovery struct {
	WorktreePath string   // Path to git worktree
	Patterns     []string // Glob patterns to filter files (e.g., "*.md", "design_docs/**")
}

// NewArtifactDiscovery creates a new artifact discovery instance.
func NewArtifactDiscovery(worktreePath string, patterns []string) *ArtifactDiscovery {
	return &ArtifactDiscovery{
		WorktreePath: worktreePath,
		Patterns:     patterns,
	}
}

// DiscoverChangedFiles returns files that were created or modified in the worktree.
// Uses git diff to compare against HEAD (or the base branch).
// Filters results by the configured patterns.
func (ad *ArtifactDiscovery) DiscoverChangedFiles() ([]string, error) {
	if ad.WorktreePath == "" {
		return nil, nil
	}

	// Get list of all changed files (staged + unstaged + untracked)
	changedFiles, err := ad.getChangedFiles()
	if err != nil {
		return nil, err
	}

	// Filter by patterns
	if len(ad.Patterns) == 0 {
		return changedFiles, nil
	}

	var matched []string
	for _, file := range changedFiles {
		if ad.matchesAnyPattern(file) {
			matched = append(matched, file)
		}
	}

	return matched, nil
}

// getChangedFiles returns all files that have changed in the worktree.
func (ad *ArtifactDiscovery) getChangedFiles() ([]string, error) {
	var allFiles []string

	// Get staged and unstaged changes
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = ad.WorktreePath
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, f := range files {
			if f != "" {
				allFiles = append(allFiles, f)
			}
		}
	}

	// Get staged changes (in case HEAD doesn't exist yet)
	cmd = exec.Command("git", "diff", "--name-only", "--cached")
	cmd.Dir = ad.WorktreePath
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, f := range files {
			if f != "" && !sliceContains(allFiles, f) {
				allFiles = append(allFiles, f)
			}
		}
	}

	// Get untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = ad.WorktreePath
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, f := range files {
			if f != "" && !sliceContains(allFiles, f) {
				allFiles = append(allFiles, f)
			}
		}
	}

	return allFiles, nil
}

// matchesAnyPattern checks if the file matches any of the configured patterns.
func (ad *ArtifactDiscovery) matchesAnyPattern(file string) bool {
	for _, pattern := range ad.Patterns {
		if matchGlob(pattern, file) {
			return true
		}
	}
	return false
}

// matchGlob performs glob matching with support for ** (recursive).
func matchGlob(pattern, file string) bool {
	// Handle ** for recursive matching
	if strings.Contains(pattern, "**") {
		// Convert ** pattern to check
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check if file starts with prefix (or prefix is empty)
			if prefix != "" && !strings.HasPrefix(file, prefix+"/") && file != prefix {
				return false
			}

			// Check if file ends with suffix pattern
			if suffix != "" {
				// Use filepath.Match for the suffix
				fileName := filepath.Base(file)
				if matched, _ := filepath.Match(suffix, fileName); matched {
					return true
				}
				// Also try matching the full relative path after prefix
				relativePath := file
				if prefix != "" {
					relativePath = strings.TrimPrefix(file, prefix+"/")
				}
				if matched, _ := filepath.Match(suffix, relativePath); matched {
					return true
				}
				// Try matching just the extension pattern
				if strings.HasPrefix(suffix, "*.") {
					ext := strings.TrimPrefix(suffix, "*")
					if strings.HasSuffix(file, ext) {
						return true
					}
				}
			} else {
				// Just ** with prefix - match anything under that directory
				return true
			}
		}
	}

	// Simple glob matching
	if matched, _ := filepath.Match(pattern, file); matched {
		return true
	}

	// Also try matching just the filename
	if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
		return true
	}

	return false
}

// ReadArtifactContent reads the content of an artifact file from the worktree.
func (ad *ArtifactDiscovery) ReadArtifactContent(relativePath string) (string, error) {
	fullPath := filepath.Join(ad.WorktreePath, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// DiscoverArtifacts returns a map of file paths to their content.
// Only includes files matching the patterns that are under maxSize bytes.
func (ad *ArtifactDiscovery) DiscoverArtifacts(maxSize int64) (map[string]string, error) {
	files, err := ad.DiscoverChangedFiles()
	if err != nil {
		return nil, err
	}

	artifacts := make(map[string]string)
	for _, file := range files {
		fullPath := filepath.Join(ad.WorktreePath, file)

		// Check file size
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files we can't stat
		}
		if maxSize > 0 && info.Size() > maxSize {
			continue // Skip files that are too large
		}

		// Read content
		content, err := ad.ReadArtifactContent(file)
		if err != nil {
			continue // Skip files we can't read
		}

		artifacts[file] = content
	}

	return artifacts, nil
}

// sliceContains checks if a slice contains a string.
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

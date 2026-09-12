package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workspace configuration: which path belongs to which workspace, and how that
// answer becomes the dashboard's SQL access filter.
//
// Split out of agent_config.go when it crossed the 800-line ceiling
// (make check-file-sizes). This is the self-contained half — nothing here is
// read by the agent registry.

// WorkspacesConfig contains workspace-related configuration for access control.
type WorkspacesConfig struct {
	Mappings         []WorkspaceMapping `yaml:"mappings" json:"mappings"`
	DefaultWorkspace string             `yaml:"default_workspace" json:"default_workspace"`
	DeriveFromPath   bool               `yaml:"derive_from_path" json:"derive_from_path"` // If true, derive workspace from path when not matched
}

// WorkspaceMapping defines a path pattern to workspace ID mapping.
type WorkspaceMapping struct {
	Pattern   string `yaml:"pattern" json:"pattern"`     // Glob pattern (e.g., "*/dev/sunholo/ailang")
	Workspace string `yaml:"workspace" json:"workspace"` // Workspace ID (e.g., "sunholo-data/ailang")
}

// LoadWorkspacesConfig loads workspace configuration from ~/.ailang/config.yaml.
// Returns a default configuration if no config is set.
func LoadWorkspacesConfig() *WorkspacesConfig {
	configPath := defaultConfigPath()
	if configPath == "" {
		return DefaultWorkspacesConfig()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultWorkspacesConfig()
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return DefaultWorkspacesConfig()
	}

	if config.Workspaces == nil {
		return DefaultWorkspacesConfig()
	}

	// Apply defaults
	if config.Workspaces.DefaultWorkspace == "" {
		config.Workspaces.DefaultWorkspace = "public"
	}

	return config.Workspaces
}

// DefaultWorkspacesConfig returns minimal default workspace mappings.
// Only internal patterns are hardcoded - user projects should be in config.
// For unmapped paths, DeriveWorkspaceFromPath() extracts workspace ID from path.
func DefaultWorkspacesConfig() *WorkspacesConfig {
	return &WorkspacesConfig{
		Mappings: []WorkspaceMapping{
			// Internal patterns only - user project mappings go in config
			{Pattern: "*/.eval_workspace/*", Workspace: "eval_workspace"},
			{Pattern: "*/worktrees/*", Workspace: "coordinator_worktrees"},
		},
		DefaultWorkspace: "", // Empty means use derived workspace
		DeriveFromPath:   true,
	}
}

// DeriveWorkspaceFromPath extracts a workspace ID from a file path.
// Uses the last two meaningful path segments (parent/basename) as the workspace ID.
// This is portable across different directory structures.
//
// Examples:
//   - /Users/mark/dev/sunholo/ailang -> sunholo/ailang
//   - /home/user/projects/rockwool/ROCKGAP -> rockwool/ROCKGAP
//   - /path/to/TwilightGame -> to/TwilightGame (or just TwilightGame if only 1 meaningful segment)
//   - /tmp/foo -> foo
func DeriveWorkspaceFromPath(path string) string {
	if path == "" || path == "unknown" {
		return "unknown"
	}

	// Clean path and split into components
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))

	// Collect meaningful path segments (skip empty, hidden, and common temp dirs)
	var meaningful []string
	for _, p := range parts {
		if p == "" || p == "tmp" || p == "var" || p == "folders" || strings.HasPrefix(p, ".") {
			continue
		}
		// Also skip common home/user path segments
		if p == "Users" || p == "home" || p == "mark" {
			continue
		}
		meaningful = append(meaningful, p)
	}

	if len(meaningful) == 0 {
		return "unknown"
	}

	// Use last two segments as org/repo (or just last one if only one exists)
	if len(meaningful) >= 2 {
		return meaningful[len(meaningful)-2] + "/" + meaningful[len(meaningful)-1]
	}
	return meaningful[len(meaningful)-1]
}

// BuildWorkspaceMappingSQL generates a SQL CASE statement for mapping file paths to workspace IDs.
// Used by the analytics backend to map process.cwd values to Firestore workspace IDs.
//
// When DeriveFromPath is true, unmapped paths are derived using SQL expressions:
// - Paths with /dev/{org}/{repo} return "org/repo"
// - Other paths return "unknown"
func (c *WorkspacesConfig) BuildWorkspaceMappingSQL(cwdColumn string) string {
	if c == nil {
		return "'unknown'"
	}

	var cases []string
	for _, m := range c.Mappings {
		// Convert glob pattern to SQL LIKE pattern:
		// - "*" at start/end becomes "%"
		// - Internal "*" becomes "%"
		pattern := m.Pattern
		pattern = strings.ReplaceAll(pattern, "*", "%")

		cases = append(cases, fmt.Sprintf(
			"WHEN %s LIKE '%s' THEN '%s'",
			cwdColumn, pattern, m.Workspace))
	}

	// ELSE clause: either use default or derive from path
	if c.DeriveFromPath {
		// For unmapped paths, use the raw cwd value as workspace.
		// The config mappings handle the main workspaces.
		// Label formatting will make raw paths human-readable.
		deriveSql := fmt.Sprintf(`ELSE %s`, cwdColumn)
		cases = append(cases, deriveSql)
	} else {
		defaultWs := c.DefaultWorkspace
		if defaultWs == "" {
			defaultWs = "unknown"
		}
		cases = append(cases, fmt.Sprintf("ELSE '%s'", defaultWs))
	}

	return "CASE\n" + strings.Join(cases, "\n") + "\nEND"
}

// GetWorkspaceLabel returns a human-friendly label for a workspace ID.
// For internal workspaces, returns predefined labels.
// For user workspaces (org/repo format), returns a formatted label.
// For raw paths (from DeriveFromPath fallback), derives and formats the label.
func (c *WorkspacesConfig) GetWorkspaceLabel(workspaceID string) string {
	// Internal workspace labels
	internalLabels := map[string]string{
		"eval_workspace":        "Eval Benchmarks",
		"coordinator_worktrees": "Coordinator Tasks",
		"unknown":               "No Workspace",
	}
	if label, ok := internalLabels[workspaceID]; ok {
		return label
	}

	// Check if this looks like a raw file path (starts with / or has more than 2 slashes)
	if strings.HasPrefix(workspaceID, "/") || strings.Count(workspaceID, "/") > 1 {
		// Derive workspace from path and use that
		derived := DeriveWorkspaceFromPath(workspaceID)
		if parts := strings.Split(derived, "/"); len(parts) == 2 {
			return formatWorkspaceLabel(parts[1])
		}
		return formatWorkspaceLabel(derived)
	}

	// For org/repo format, make the repo name the label
	if parts := strings.Split(workspaceID, "/"); len(parts) == 2 {
		return formatWorkspaceLabel(parts[1])
	}

	// For single-segment workspace, format it nicely
	return formatWorkspaceLabel(workspaceID)
}

// GetPathPatternsForWorkspace returns SQL LIKE patterns that match a workspace ID.
// Used for filtering spans by workspace when the filter is an org/repo ID.
// Returns nil if no matching mappings found (caller should use fallback).
//
// Example: For workspace "MarkEdmondson1234/TwilightGame" with mapping
// {Pattern: "*/TwilightGame*", Workspace: "MarkEdmondson1234/TwilightGame"},
// returns ["%/TwilightGame%"] as the pattern to match process.cwd values.
func (c *WorkspacesConfig) GetPathPatternsForWorkspace(workspaceID string) []string {
	if c == nil || workspaceID == "" {
		return nil
	}

	var patterns []string
	for _, m := range c.Mappings {
		if m.Workspace == workspaceID {
			// Convert glob pattern to SQL LIKE pattern
			pattern := strings.ReplaceAll(m.Pattern, "*", "%")
			patterns = append(patterns, pattern)
		}
	}

	// If no explicit mapping, try to derive patterns from the workspace ID
	// For org/repo format, generate patterns that would match the repo name
	if len(patterns) == 0 && c.DeriveFromPath {
		// Extract the repo name from org/repo format
		parts := strings.Split(workspaceID, "/")
		if len(parts) >= 1 {
			repoName := parts[len(parts)-1]
			// Generate pattern that matches paths ending with this repo name
			patterns = append(patterns, "%/"+repoName)
			patterns = append(patterns, "%/"+repoName+"/%")
		}
	}

	return patterns
}

// formatWorkspaceLabel converts a workspace name to a human-readable label.
// Examples: "ailang" -> "Ailang", "ROCKGAP" -> "ROCKGAP", "stapledons_voyage" -> "Stapledons Voyage"
//
//	"TwilightGame" -> "Twilight Game" (splits camel case)
func formatWorkspaceLabel(name string) string {
	if name == "" {
		return "Unknown"
	}

	// If all uppercase, keep it (like ROCKGAP, ROCKGPT)
	if strings.ToUpper(name) == name && len(name) > 1 {
		return name
	}

	// Insert spaces before uppercase letters (camel case -> spaces)
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous char was lowercase (camel case boundary)
			prev := rune(name[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune(' ')
			}
		}
		result.WriteRune(r)
	}
	name = result.String()

	// Replace underscores/dashes with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Title case each word
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

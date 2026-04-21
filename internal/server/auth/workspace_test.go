package auth

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestMatchPathPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		// Exact matches
		{"exact match", "/Users/mark/dev/sunholo/ailang", "/Users/mark/dev/sunholo/ailang", true},
		{"exact no match", "/Users/mark/dev/sunholo/ailang", "/Users/mark/dev/sunholo/other", false},

		// Single wildcard (*)
		{"single wildcard matches dir", "*/ailang", "/Users/mark/dev/sunholo/ailang", false}, // * matches ONE directory
		{"single wildcard in middle", "/Users/*/dev/sunholo/ailang", "/Users/mark/dev/sunholo/ailang", true},
		{"single wildcard no match", "/Users/*/dev/sunholo/ailang", "/Users/mark/code/sunholo/ailang", false},

		// Double wildcard (**)
		{"double wildcard matches multiple", "**/ailang", "/Users/mark/dev/sunholo/ailang", true},
		{"double wildcard at start", "**/sunholo/ailang", "/Users/mark/dev/sunholo/ailang", true},
		{"double wildcard at end", "/Users/mark/**", "/Users/mark/dev/sunholo/ailang", true},
		{"double wildcard in middle", "/Users/**/ailang", "/Users/mark/dev/sunholo/ailang", true},

		// Pattern with wildcards in directory names
		// Note: Single * matches ONE path component, so path length must match
		{"wildcard in dirname", "**/dev/sunholo/ailang", "/any/path/dev/sunholo/ailang", true},
		{"wildcard prefix in dirname", "**/dev/sunholo/*", "/any/path/dev/sunholo/ailang", true},

		// Edge cases
		{"empty path", "*/ailang", "", false},
		{"empty pattern", "", "/Users/mark", false},
		{"root path", "/", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPathPattern(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPathPattern(%q, %q) = %v, want %v",
					tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		s        string
		expected bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"abc", "abc", true},
		{"abc", "abcd", false},
		{"a*c", "abc", true},
		{"a*c", "ac", true},
		{"a*c", "abdc", true},
		{"*bc", "abc", true},
		{"ab*", "abc", true},
		{"a*b*c", "aXbYc", true},
		{"a*b*c", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.s, func(t *testing.T) {
			result := matchWildcard(tt.pattern, tt.s)
			if result != tt.expected {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v",
					tt.pattern, tt.s, result, tt.expected)
			}
		})
	}
}

func TestWorkspaceService_GetWorkspaceByPath(t *testing.T) {
	config := &WorkspacesConfig{
		Mappings: []coordinator.WorkspaceMapping{
			// Use ** to match any leading path components
			{Pattern: "**/dev/sunholo/ailang", Workspace: "sunholo-data/ailang"},
			{Pattern: "**/dev/sunholo/stapledons_voyage", Workspace: "sunholo-data/stapledons_voyage"},
			{Pattern: "**/dev/TwilightGame", Workspace: "MarkEdmondson1234/TwilightGame"},
		},
		DefaultWorkspace: "public",
	}

	ws := NewWorkspaceService(nil, config)

	tests := []struct {
		path     string
		expected string
	}{
		{"/Users/mark/dev/sunholo/ailang", "sunholo-data/ailang"},
		{"/Users/mark/dev/sunholo/stapledons_voyage", "sunholo-data/stapledons_voyage"},
		{"/Users/mark/dev/TwilightGame", "MarkEdmondson1234/TwilightGame"},
		{"/Users/mark/dev/unknown/project", "public"},
		{"/home/user/dev/sunholo/ailang", "sunholo-data/ailang"}, // Different user path
		{"", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ws.GetWorkspaceByPath(tt.path)
			if result != tt.expected {
				t.Errorf("GetWorkspaceByPath(%q) = %q, want %q",
					tt.path, result, tt.expected)
			}
		})
	}
}

func TestWorkspaceService_GetWorkspaceByPath_NoConfig(t *testing.T) {
	// Test with nil config
	ws := NewWorkspaceService(nil, nil)
	result := ws.GetWorkspaceByPath("/any/path")
	if result != "public" {
		t.Errorf("GetWorkspaceByPath with nil config = %q, want %q", result, "public")
	}

	// Test with empty config
	ws2 := NewWorkspaceService(nil, &WorkspacesConfig{})
	result2 := ws2.GetWorkspaceByPath("/any/path")
	if result2 != "public" {
		t.Errorf("GetWorkspaceByPath with empty config = %q, want %q", result2, "public")
	}
}

func TestWorkspaceService_CacheStats(t *testing.T) {
	ws := NewWorkspaceService(nil, nil)

	workspaces, access := ws.CacheStats()
	if workspaces != 0 || access != 0 {
		t.Errorf("Initial cache should be empty, got workspaces=%d, access=%d",
			workspaces, access)
	}
}

func TestWorkspaceService_InvalidateCache(t *testing.T) {
	ws := NewWorkspaceService(nil, nil)

	// Manually add something to cache
	ws.mutex.Lock()
	ws.cache.workspaces["test"] = &cachedWorkspace{}
	ws.cache.access["test:test"] = &cachedWorkspaceAccess{}
	ws.mutex.Unlock()

	w, a := ws.CacheStats()
	if w != 1 || a != 1 {
		t.Errorf("Cache should have 1 item each, got workspaces=%d, access=%d", w, a)
	}

	ws.InvalidateCache()

	w, a = ws.CacheStats()
	if w != 0 || a != 0 {
		t.Errorf("Cache should be empty after invalidate, got workspaces=%d, access=%d", w, a)
	}
}

func TestWorkspaceAccess_Validation(t *testing.T) {
	// Test that WorkspaceAccess struct has expected fields
	access := WorkspaceAccess{
		Email:       "test@example.com",
		WorkspaceID: "sunholo-data/ailang",
		Role:        "Viewer",
	}

	if access.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", access.Email, "test@example.com")
	}
	if access.WorkspaceID != "sunholo-data/ailang" {
		t.Errorf("WorkspaceID = %q, want %q", access.WorkspaceID, "sunholo-data/ailang")
	}
	if access.Role != "Viewer" {
		t.Errorf("Role = %q, want %q", access.Role, "Viewer")
	}
}

func TestWorkspace_Struct(t *testing.T) {
	// Test that Workspace struct has expected fields
	workspace := Workspace{
		ID:         "sunholo-data/ailang",
		Name:       "AILANG",
		GitHubRepo: "sunholo-data/ailang",
		IsPublic:   true,
		PathPatterns: []string{
			"*/dev/sunholo/ailang",
		},
	}

	if workspace.ID != "sunholo-data/ailang" {
		t.Errorf("ID = %q, want %q", workspace.ID, "sunholo-data/ailang")
	}
	if !workspace.IsPublic {
		t.Error("IsPublic should be true")
	}
	if len(workspace.PathPatterns) != 1 {
		t.Errorf("PathPatterns length = %d, want 1", len(workspace.PathPatterns))
	}
}

func TestAccessibleWorkspace_Struct(t *testing.T) {
	// Test that AccessibleWorkspace includes both Workspace and Role
	aw := AccessibleWorkspace{
		Workspace: Workspace{
			ID:       "sunholo-data/ailang",
			Name:     "AILANG",
			IsPublic: true,
		},
		Role: "Approver",
	}

	if aw.ID != "sunholo-data/ailang" {
		t.Errorf("ID = %q, want %q", aw.ID, "sunholo-data/ailang")
	}
	if aw.Role != "Approver" {
		t.Errorf("Role = %q, want %q", aw.Role, "Approver")
	}
}

// Integration tests would require a Firestore emulator or mock.
// The following tests verify the logic without Firestore.

func TestMatchParts_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		pattern  []string
		path     []string
		expected bool
	}{
		{"empty both", []string{}, []string{}, true},
		{"empty pattern", []string{}, []string{"a"}, false},
		{"empty path", []string{"a"}, []string{}, false},
		{"single match", []string{"a"}, []string{"a"}, true},
		{"single wildcard", []string{"*"}, []string{"a"}, true},
		{"double wildcard only", []string{"**"}, []string{"a", "b", "c"}, true},
		{"double wildcard empty", []string{"**"}, []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchParts(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchParts(%v, %v) = %v, want %v",
					tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

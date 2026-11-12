package auth

import (
	"testing"
)

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

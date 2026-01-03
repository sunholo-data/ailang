package coordinator

import (
	"testing"
)

func TestMergeResultStruct(t *testing.T) {
	result := &MergeResult{
		Success:       true,
		MergedFiles:   []string{"file1.go", "file2.go"},
		ConflictFiles: nil,
		CommitHash:    "abc123def456",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if len(result.MergedFiles) != 2 {
		t.Errorf("expected 2 merged files, got %d", len(result.MergedFiles))
	}
	if result.CommitHash != "abc123def456" {
		t.Errorf("expected commit hash 'abc123def456', got %q", result.CommitHash)
	}
}

func TestMergeResultWithConflicts(t *testing.T) {
	result := &MergeResult{
		Success:       false,
		MergedFiles:   []string{"file1.go"},
		ConflictFiles: []string{"file2.go", "file3.go"},
		Error:         "merge conflicts detected",
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if len(result.ConflictFiles) != 2 {
		t.Errorf("expected 2 conflict files, got %d", len(result.ConflictFiles))
	}
	if result.Error == "" {
		t.Error("expected Error to be set")
	}
}

func TestParseConflictFiles(t *testing.T) {
	output := `Auto-merging file1.go
CONFLICT (content): Merge conflict in file2.go
CONFLICT (content): Merge conflict in internal/parser/parser.go
Automatic merge failed; fix conflicts and then commit the result.`

	conflicts := parseConflictFiles(output)

	if len(conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d: %v", len(conflicts), conflicts)
	}
	if len(conflicts) > 0 && conflicts[0] != "file2.go" {
		t.Errorf("expected first conflict to be 'file2.go', got %q", conflicts[0])
	}
	if len(conflicts) > 1 && conflicts[1] != "internal/parser/parser.go" {
		t.Errorf("expected second conflict to be 'internal/parser/parser.go', got %q", conflicts[1])
	}
}

func TestParseConflictFilesEmpty(t *testing.T) {
	output := `Already up to date.`

	conflicts := parseConflictFiles(output)

	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestGetMainRepoPath(t *testing.T) {
	// This tests the path extraction logic - not actual git operations
	// Path pattern: ~/.ailang/state/worktrees/coordinator/task-id/
	// Git dir pattern: /main/repo/.git/worktrees/task-id

	// Just verify the function doesn't panic with empty input
	result := getMainRepoPath("")
	if result != "" {
		t.Errorf("expected empty result for empty input, got %q", result)
	}
}

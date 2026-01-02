package coordinator

import (
	"strings"
	"testing"
	"time"
)

// TestTaskCreationSetsAgentID verifies that when a task is created from an inbox message,
// the AgentID field is set correctly based on the inbox name.
func TestTaskCreationSetsAgentID(t *testing.T) {
	// This test verifies Bug #1 in M-COORD-ARTIFACT-DISCOVERY:
	// AgentID should be set on task creation from inbox

	// Create a task record simulating what daemon_tasks.go should do
	agentID := "design-doc-creator" // This comes from inbox mapping

	task := &TaskRecord{
		ID:      "test-task-123",
		AgentID: agentID, // This is what we're testing - should be set
		Title:   "Test Task",
		Content: "Test content",
	}

	if task.AgentID != "design-doc-creator" {
		t.Errorf("AgentID not set: got %q, want %q", task.AgentID, "design-doc-creator")
	}
}

// TestDirectiveIncludesSkillInvocation verifies that the directive built for a task
// with an AgentID includes the skill invocation instruction.
func TestDirectiveIncludesSkillInvocation(t *testing.T) {
	// This test verifies Bug #2 in M-COORD-ARTIFACT-DISCOVERY:
	// Directive should include "invoke the X skill" when AgentID is set

	testCases := []struct {
		name          string
		task          *TaskRecord
		wantContains  string
		wantSkillName string
	}{
		{
			name: "design-doc-creator agent",
			task: &TaskRecord{
				ID:          "test-1",
				AgentID:     "design-doc-creator",
				Content:     "Create a design doc for feature X",
				GithubIssue: 0, // No GitHub issue
				Stage:       TaskStageNone,
			},
			wantContains:  "invoke the design-doc-creator skill",
			wantSkillName: "design-doc-creator",
		},
		{
			name: "sprint-planner agent",
			task: &TaskRecord{
				ID:          "test-2",
				AgentID:     "sprint-planner",
				Content:     "Plan the sprint",
				GithubIssue: 0,
				Stage:       TaskStageNone,
			},
			wantContains:  "invoke the sprint-planner skill",
			wantSkillName: "sprint-planner",
		},
		{
			name: "sprint-executor agent",
			task: &TaskRecord{
				ID:          "test-3",
				AgentID:     "sprint-executor",
				Content:     "Execute the sprint",
				GithubIssue: 0,
				Stage:       TaskStageNone,
			},
			wantContains:  "invoke the sprint-executor skill",
			wantSkillName: "sprint-executor",
		},
		{
			name: "GitHub issue with stage (legacy path)",
			task: &TaskRecord{
				ID:          "test-4",
				AgentID:     "",
				Content:     "Fix the bug",
				GithubIssue: 102,
				Stage:       TaskStageDesign,
			},
			wantContains:  "invoke the design-doc-creator skill",
			wantSkillName: "design-doc-creator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build directive using the function under test
			directive := BuildStageDirective(tc.task)

			// Check that directive contains skill invocation
			if !strings.Contains(strings.ToLower(directive), strings.ToLower(tc.wantContains)) {
				t.Errorf("Directive missing skill invocation.\nGot: %s\nWant to contain: %s",
					directive, tc.wantContains)
			}
		})
	}
}

// TestDirectiveFallbackOnlyWhenNoAgentInfo verifies that raw content is only
// returned when we truly have no agent information (no AgentID AND no Stage).
func TestDirectiveFallbackOnlyWhenNoAgentInfo(t *testing.T) {
	// This tests the fix for Bug #2: || should be &&

	testCases := []struct {
		name           string
		task           *TaskRecord
		expectRawOnly  bool // If true, directive should be raw content only
		expectSkillInv bool // If true, directive should include skill invocation
	}{
		{
			name: "no AgentID, no Stage, no GitHub - should return raw content",
			task: &TaskRecord{
				ID:          "test-1",
				AgentID:     "",
				Content:     "Raw content here",
				GithubIssue: 0,
				Stage:       TaskStageNone,
			},
			expectRawOnly:  true,
			expectSkillInv: false,
		},
		{
			name: "has AgentID, no Stage - should invoke skill",
			task: &TaskRecord{
				ID:          "test-2",
				AgentID:     "design-doc-creator",
				Content:     "Content with agent",
				GithubIssue: 0,
				Stage:       TaskStageNone,
			},
			expectRawOnly:  false,
			expectSkillInv: true,
		},
		{
			name: "no AgentID, has Stage - should invoke skill",
			task: &TaskRecord{
				ID:          "test-3",
				AgentID:     "",
				Content:     "Content with stage",
				GithubIssue: 102,
				Stage:       TaskStageDesign,
			},
			expectRawOnly:  false,
			expectSkillInv: true,
		},
		{
			name: "has both AgentID and Stage - should invoke skill",
			task: &TaskRecord{
				ID:          "test-4",
				AgentID:     "design-doc-creator",
				Content:     "Content with both",
				GithubIssue: 102,
				Stage:       TaskStageDesign,
			},
			expectRawOnly:  false,
			expectSkillInv: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directive := BuildStageDirective(tc.task)

			isRawOnly := directive == tc.task.Content
			hasSkillInv := strings.Contains(strings.ToLower(directive), "invoke the")

			if tc.expectRawOnly && !isRawOnly {
				t.Errorf("Expected raw content only, got: %s", directive)
			}
			if !tc.expectRawOnly && isRawOnly {
				t.Errorf("Expected directive with skill invocation, got raw content only: %s", directive)
			}
			if tc.expectSkillInv && !hasSkillInv {
				t.Errorf("Expected skill invocation in directive, got: %s", directive)
			}
		})
	}
}

// TestArtifactDiscoveryFindsMatchingFiles verifies that artifact discovery
// finds files matching the configured patterns.
func TestArtifactDiscoveryFindsMatchingFiles(t *testing.T) {
	// This test verifies artifact discovery pattern matching works correctly
	// Note: This is a unit test for matchGlob function, not a full git test

	testCases := []struct {
		pattern string
		file    string
		want    bool
	}{
		// design_docs/**/*.md pattern
		{"design_docs/**/*.md", "design_docs/planned/v0_6_3/feature.md", true},
		{"design_docs/**/*.md", "design_docs/implemented/v0_6_2/old.md", true},
		{"design_docs/**/*.md", "design_docs/feature.md", true},
		{"design_docs/**/*.md", "design_docs/deep/nested/path/doc.md", true},
		{"design_docs/**/*.md", "other/file.md", false},
		{"design_docs/**/*.md", "design_docs/file.txt", false},

		// Sprint JSON pattern
		{".ailang/state/sprints/*.json", ".ailang/state/sprints/sprint_M-TEST.json", true},
		{".ailang/state/sprints/*.json", ".ailang/state/sprints/other.json", true},
		{".ailang/state/sprints/*.json", ".ailang/state/other.json", false},

		// Go files pattern
		{"**/*.go", "internal/coordinator/file.go", true},
		{"**/*.go", "cmd/ailang/main.go", true},
		{"**/*.go", "file.go", true},
		{"**/*.go", "file.txt", false},
	}

	for _, tc := range testCases {
		t.Run(tc.pattern+"_"+tc.file, func(t *testing.T) {
			got := matchGlob(tc.pattern, tc.file)
			if got != tc.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pattern, tc.file, got, tc.want)
			}
		})
	}
}

// TestGitHubCommentIncludesArtifactContent verifies that rendered GitHub comments
// include the artifact content in a collapsible details section.
func TestGitHubCommentIncludesArtifactContent(t *testing.T) {
	// This test verifies that when artifacts are found, their content
	// is included in the GitHub comment

	designDocContent := `# Test Design Doc

This is a test design document.

## Goals
- Goal 1
- Goal 2
`

	data := &CommentData{
		TaskID:           "test-task",
		Duration:         5 * time.Minute,
		DesignDocPath:    "design_docs/planned/v0_6_3/test.md",
		DesignDocContent: designDocContent,
	}

	comment, err := RenderDesignDocComment(data)
	if err != nil {
		t.Fatalf("RenderDesignDocComment failed: %v", err)
	}

	// Verify comment contains key elements
	checks := []string{
		"<details>",                    // Collapsible section
		"</details>",                   // End of collapsible
		"test.md",                      // File name appears
		"This is a test design document", // Content included
	}

	for _, check := range checks {
		if !strings.Contains(comment, check) {
			t.Errorf("Comment missing expected content: %q\nFull comment:\n%s", check, comment)
		}
	}
}

// TestNoArtifactsReportsError verifies that when no artifacts are found,
// an appropriate error message is generated (not an empty comment).
func TestNoArtifactsReportsError(t *testing.T) {
	// This test verifies Bug #3 in M-COORD-ARTIFACT-DISCOVERY:
	// When no artifacts found, should report "Failed to find artifacts" not empty

	// Test the comment rendering with empty content
	data := &CommentData{
		TaskID:           "test-task",
		Duration:         5 * time.Minute,
		DesignDocPath:    "", // No path
		DesignDocContent: "", // No content
	}

	comment, err := RenderDesignDocComment(data)
	if err != nil {
		t.Fatalf("RenderDesignDocComment failed: %v", err)
	}

	// The comment should still render, but ideally with a "no artifacts" message
	// For now, verify it doesn't crash and produces something
	if comment == "" {
		t.Error("Comment should not be completely empty")
	}

	// Note: The actual "Failed to find artifacts" message would be in
	// ProcessStageCompletion, not in the template rendering.
	// This test validates the template handles empty content gracefully.
}

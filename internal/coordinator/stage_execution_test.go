package coordinator

import (
	"strings"
	"testing"
)

func TestBuildStageDirective_NonGitHubTask(t *testing.T) {
	task := &TaskRecord{
		Content:     "Fix the bug",
		GithubIssue: 0, // No GitHub issue
		Stage:       TaskStageNone,
	}

	result := BuildStageDirective(task)
	if result != "Fix the bug" {
		t.Errorf("Expected original content for non-GitHub task, got: %s", result)
	}
}

func TestBuildStageDirective_DesignStage(t *testing.T) {
	task := &TaskRecord{
		Content:     "Add a new feature",
		GithubIssue: 42,
		Stage:       TaskStageDesign,
	}

	result := BuildStageDirective(task)

	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Design directive should reference the GitHub issue number")
	}
	if !strings.Contains(result, "design-doc-creator") {
		t.Error("Design directive should mention design-doc-creator skill")
	}
	if !strings.Contains(result, "DESIGN_DOC_PATH:") {
		t.Error("Design directive should include output format for design doc path")
	}
}

func TestBuildStageDirective_SprintStage(t *testing.T) {
	task := &TaskRecord{
		Content:     "Add a new feature",
		GithubIssue: 42,
		Stage:       TaskStageSprint,
	}

	result := BuildStageDirective(task)

	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Sprint directive should reference the GitHub issue number")
	}
	if !strings.Contains(result, "sprint-planner") {
		t.Error("Sprint directive should mention sprint-planner skill")
	}
	if !strings.Contains(result, "SPRINT_PLAN_PATH:") {
		t.Error("Sprint directive should include output format for sprint plan path")
	}
}

func TestBuildStageDirective_ImplementationStage(t *testing.T) {
	task := &TaskRecord{
		Content:     "Add a new feature",
		GithubIssue: 42,
		Stage:       TaskStageImplementation,
	}

	result := BuildStageDirective(task)

	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Implementation directive should reference the GitHub issue number")
	}
	if !strings.Contains(result, "sprint-executor") {
		t.Error("Implementation directive should mention sprint-executor skill")
	}
	if !strings.Contains(result, "IMPLEMENTATION_COMPLETE:") {
		t.Error("Implementation directive should include output format for completion")
	}
}

func TestParseStageOutput_DesignDoc(t *testing.T) {
	output := `Created design document.
DESIGN_DOC_PATH: design_docs/planned/v0_6_3/fix-parser-bug.md
Done.`

	result := ParseStageOutput(output, TaskStageDesign)

	if result.DesignDocPath != "design_docs/planned/v0_6_3/fix-parser-bug.md" {
		t.Errorf("Expected design doc path, got: %s", result.DesignDocPath)
	}
}

func TestParseStageOutput_SprintPlan(t *testing.T) {
	output := `Created sprint plan.
SPRINT_PLAN_PATH: design_docs/planned/v0_6_3/fix-parser-bug-sprint-plan.md
Ready for review.`

	result := ParseStageOutput(output, TaskStageSprint)

	if result.SprintPlanPath != "design_docs/planned/v0_6_3/fix-parser-bug-sprint-plan.md" {
		t.Errorf("Expected sprint plan path, got: %s", result.SprintPlanPath)
	}
}

func TestParseStageOutput_Implementation(t *testing.T) {
	output := `Implementation complete.
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: fix-parser-bug
FILES_CREATED: internal/parser/helper.go, internal/parser/helper_test.go
FILES_MODIFIED: internal/parser/parser.go`

	result := ParseStageOutput(output, TaskStageImplementation)

	if result.BranchName != "fix-parser-bug" {
		t.Errorf("Expected branch name, got: %s", result.BranchName)
	}
	if len(result.FilesCreated) != 2 {
		t.Errorf("Expected 2 files created, got: %v", result.FilesCreated)
	}
	if len(result.FilesModified) != 1 {
		t.Errorf("Expected 1 file modified, got: %v", result.FilesModified)
	}
}

func TestParseStageOutput_ImplementationNotComplete(t *testing.T) {
	output := `Still working on implementation...`

	result := ParseStageOutput(output, TaskStageImplementation)

	if result.BranchName != "" {
		t.Error("Should not extract branch name if IMPLEMENTATION_COMPLETE not present")
	}
}

func TestParseStageOutput_MissingPath(t *testing.T) {
	output := `Something happened but no path was output.`

	result := ParseStageOutput(output, TaskStageDesign)

	if result.DesignDocPath != "" {
		t.Error("Should return empty path if not found in output")
	}
}

func TestExtractList_EmptyValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"FILES_CREATED: none", 0},
		{"FILES_CREATED: None", 0},
		{"FILES_CREATED: ", 0},
		{"FILES_CREATED: a, b, c", 3},
	}

	for _, tc := range tests {
		result := extractList(tc.input, `FILES_CREATED:\s*(.+)`)
		if len(result) != tc.expected {
			t.Errorf("For input %q, expected %d items, got %d", tc.input, tc.expected, len(result))
		}
	}
}

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

// Tests for BuildDirectiveFromConfig (M-COORD-GENERIC-WORKFLOWS M2)

func TestBuildDirectiveFromConfig_SkillType(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-123",
		Content:     "Create a design document",
		GithubIssue: 42,
		Stage:       TaskStageDesign,
	}
	agent := &AgentConfig{
		ID: "design-agent",
		Invoke: &InvokeConfig{
			Type: "skill",
			Name: "design-doc-creator",
		},
		OutputMarkers: []string{"DESIGN_DOC_PATH:"},
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Skill directive should reference the GitHub issue number")
	}
	if !strings.Contains(result, "design-doc-creator") {
		t.Error("Skill directive should mention the skill name")
	}
	if !strings.Contains(result, "DESIGN_DOC_PATH:") {
		t.Error("Skill directive should include output markers")
	}
}

func TestBuildDirectiveFromConfig_AgentType(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-456",
		Content:     "Plan the sprint",
		GithubIssue: 42,
		Stage:       TaskStageSprint,
	}
	agent := &AgentConfig{
		ID: "sprint-agent",
		Invoke: &InvokeConfig{
			Type: "agent",
			Name: "sprint-planner",
		},
		OutputMarkers: []string{"SPRINT_PLAN_PATH:"},
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Agent directive should reference the GitHub issue number")
	}
	if !strings.Contains(result, "sprint-planner") {
		t.Error("Agent directive should mention the target agent")
	}
	if !strings.Contains(result, "Hand off") {
		t.Error("Agent directive should mention handoff")
	}
}

func TestBuildDirectiveFromConfig_PromptType(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-789",
		Content:     "Analyze this code",
		GithubIssue: 42,
		Stage:       TaskStageDesign,
	}
	agent := &AgentConfig{
		ID: "custom-agent",
		Invoke: &InvokeConfig{
			Type:     "prompt",
			Template: "Task {{.TaskID}} for issue #{{.GithubIssue}}: {{.Content}}",
		},
		OutputMarkers: []string{"RESULT:"},
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "task-789") {
		t.Errorf("Prompt directive should expand TaskID, got: %s", result)
	}
	if !strings.Contains(result, "#42") {
		t.Error("Prompt directive should expand GithubIssue")
	}
	if !strings.Contains(result, "Analyze this code") {
		t.Error("Prompt directive should expand Content")
	}
}

func TestBuildDirectiveFromConfig_NoGitHubIssue(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-abc",
		Content:     "Internal task",
		GithubIssue: 0,
		Stage:       TaskStageNone,
	}
	agent := &AgentConfig{
		ID: "internal-agent",
		Invoke: &InvokeConfig{
			Type: "skill",
			Name: "research-tool",
		},
	}

	result := BuildDirectiveFromConfig(task, agent)

	if strings.Contains(result, "GitHub issue #0") {
		t.Error("Should not mention GitHub issue when issue number is 0")
	}
	if !strings.Contains(result, "Internal task") {
		t.Error("Should include task content")
	}
	if !strings.Contains(result, "research-tool") {
		t.Error("Should mention skill name")
	}
}

func TestBuildDirectiveFromConfig_FallbackToLegacy(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-legacy",
		Content:     "Fix the bug",
		GithubIssue: 42,
		Stage:       TaskStageDesign,
	}

	// No agent - should fall back to legacy
	result1 := BuildDirectiveFromConfig(task, nil)
	if !strings.Contains(result1, "design-doc-creator") {
		t.Error("Should fall back to legacy design directive when agent is nil")
	}

	// Agent without InvokeConfig - should fall back to legacy
	agent := &AgentConfig{ID: "legacy-agent"}
	result2 := BuildDirectiveFromConfig(task, agent)
	if !strings.Contains(result2, "design-doc-creator") {
		t.Error("Should fall back to legacy design directive when InvokeConfig is nil")
	}
}

func TestBuildDirectiveFromConfig_UnknownType(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-unknown",
		Content:     "Do something",
		GithubIssue: 42,
		Stage:       TaskStageDesign,
	}
	agent := &AgentConfig{
		ID: "unknown-agent",
		Invoke: &InvokeConfig{
			Type: "unknown-type",
			Name: "some-name",
		},
	}

	result := BuildDirectiveFromConfig(task, agent)

	// Should fall back to legacy
	if !strings.Contains(result, "design-doc-creator") {
		t.Error("Unknown invoke type should fall back to legacy directive")
	}
}

func TestBuildDirectiveFromConfig_MultipleMarkers(t *testing.T) {
	task := &TaskRecord{
		ID:          "task-multi",
		Content:     "Create implementation",
		GithubIssue: 42,
		Stage:       TaskStageImplementation,
	}
	agent := &AgentConfig{
		ID: "impl-agent",
		Invoke: &InvokeConfig{
			Type: "skill",
			Name: "sprint-executor",
		},
		OutputMarkers: []string{"IMPLEMENTATION_COMPLETE:", "BRANCH_NAME:", "FILES_CREATED:"},
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "IMPLEMENTATION_COMPLETE:") {
		t.Error("Should include first marker")
	}
	if !strings.Contains(result, "BRANCH_NAME:") {
		t.Error("Should include second marker")
	}
	if !strings.Contains(result, "FILES_CREATED:") {
		t.Error("Should include third marker")
	}
}

// Test that agents without explicit InvokeConfig use effective defaults
func TestBuildDirectiveFromConfig_UsesEffectiveDefaults(t *testing.T) {
	task := &TaskRecord{
		ID:          "test-task",
		GithubIssue: 42,
		Content:     "Test task content",
		Stage:       TaskStageSprint,
	}

	// Agent without explicit InvokeConfig (common in real configs)
	agent := &AgentConfig{
		ID:    "sprint-planner",
		Label: "Sprint Planner",
		// No Invoke, OutputMarkers, or Approval set - uses defaults
	}

	result := BuildDirectiveFromConfig(task, agent)

	// Should use default InvokeConfig for sprint-planner (skill type)
	if !strings.Contains(result, "sprint-planner") {
		t.Error("Should use default skill name 'sprint-planner'")
	}

	// Should use default OutputMarkers for sprint-planner
	if !strings.Contains(result, "SPRINT_PLAN_PATH:") {
		t.Error("Should include default marker SPRINT_PLAN_PATH:")
	}

	// Should include GitHub issue reference
	if !strings.Contains(result, "GitHub issue #42") {
		t.Error("Should include GitHub issue reference")
	}
}

// Test design-doc-creator defaults
func TestBuildDirectiveFromConfig_DesignDocCreatorDefaults(t *testing.T) {
	task := &TaskRecord{
		ID:          "test-task",
		GithubIssue: 123,
		Content:     "Create a design doc",
		Stage:       TaskStageDesign,
	}

	agent := &AgentConfig{
		ID:    "design-doc-creator",
		Label: "Design Doc Creator",
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "design-doc-creator") {
		t.Error("Should use default skill name 'design-doc-creator'")
	}
	if !strings.Contains(result, "DESIGN_DOC_PATH:") {
		t.Error("Should include default marker DESIGN_DOC_PATH:")
	}
}

// Test sprint-executor defaults
func TestBuildDirectiveFromConfig_SprintExecutorDefaults(t *testing.T) {
	task := &TaskRecord{
		ID:          "test-task",
		GithubIssue: 456,
		Content:     "Implement the feature",
		Stage:       TaskStageImplementation,
	}

	agent := &AgentConfig{
		ID:    "sprint-executor",
		Label: "Sprint Executor",
	}

	result := BuildDirectiveFromConfig(task, agent)

	if !strings.Contains(result, "sprint-executor") {
		t.Error("Should use default skill name 'sprint-executor'")
	}
	if !strings.Contains(result, "IMPLEMENTATION_COMPLETE:") {
		t.Error("Should include default marker IMPLEMENTATION_COMPLETE:")
	}
	if !strings.Contains(result, "BRANCH_NAME:") {
		t.Error("Should include default marker BRANCH_NAME:")
	}
}

func TestFormatOutputMarkers(t *testing.T) {
	tests := []struct {
		markers  []string
		expected string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"PATH:"}, "- PATH: <value>\n"},
		{[]string{"A:", "B:"}, "- A: <value>\n- B: <value>\n"},
	}

	for _, tc := range tests {
		result := formatOutputMarkers(tc.markers)
		if result != tc.expected {
			t.Errorf("For markers %v, expected %q, got %q", tc.markers, tc.expected, result)
		}
	}
}

// Tests for ParseOutputMarkers (M-COORD-GENERIC-WORKFLOWS M3)

func TestParseOutputMarkers_SingleMarker(t *testing.T) {
	output := `Task completed successfully.
DESIGN_DOC_PATH: design_docs/planned/v0_6_3/feature.md
Done.`

	markers := []string{"DESIGN_DOC_PATH:"}
	result := ParseOutputMarkers(output, markers)

	if len(result) != 1 {
		t.Errorf("Expected 1 marker, got %d", len(result))
	}
	if result["DESIGN_DOC_PATH:"] != "design_docs/planned/v0_6_3/feature.md" {
		t.Errorf("Wrong value: %s", result["DESIGN_DOC_PATH:"])
	}
}

func TestParseOutputMarkers_MultipleMarkers(t *testing.T) {
	output := `Implementation finished.
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: fix-parser-bug
FILES_CREATED: parser.go, parser_test.go`

	markers := []string{"IMPLEMENTATION_COMPLETE:", "BRANCH_NAME:", "FILES_CREATED:"}
	result := ParseOutputMarkers(output, markers)

	if len(result) != 3 {
		t.Errorf("Expected 3 markers, got %d", len(result))
	}
	if result["IMPLEMENTATION_COMPLETE:"] != "true" {
		t.Errorf("Wrong IMPLEMENTATION_COMPLETE: %s", result["IMPLEMENTATION_COMPLETE:"])
	}
	if result["BRANCH_NAME:"] != "fix-parser-bug" {
		t.Errorf("Wrong BRANCH_NAME: %s", result["BRANCH_NAME:"])
	}
	if result["FILES_CREATED:"] != "parser.go, parser_test.go" {
		t.Errorf("Wrong FILES_CREATED: %s", result["FILES_CREATED:"])
	}
}

func TestParseOutputMarkers_MarkerWithoutColon(t *testing.T) {
	output := `RESULT: success`

	// Should work with or without trailing colon in marker spec
	markers := []string{"RESULT"}
	result := ParseOutputMarkers(output, markers)

	if result["RESULT"] != "success" {
		t.Errorf("Expected 'success', got %q", result["RESULT"])
	}
}

func TestParseOutputMarkers_NoSpace(t *testing.T) {
	output := `RESULT:success`

	markers := []string{"RESULT:"}
	result := ParseOutputMarkers(output, markers)

	if result["RESULT:"] != "success" {
		t.Errorf("Expected 'success', got %q", result["RESULT:"])
	}
}

func TestParseOutputMarkers_EmptyMarkers(t *testing.T) {
	output := `RESULT: value`

	result := ParseOutputMarkers(output, nil)
	if len(result) != 0 {
		t.Error("Expected empty map for nil markers")
	}

	result = ParseOutputMarkers(output, []string{})
	if len(result) != 0 {
		t.Error("Expected empty map for empty markers slice")
	}
}

func TestParseOutputMarkers_MarkerNotFound(t *testing.T) {
	output := `Some random output without markers.`

	markers := []string{"NONEXISTENT:"}
	result := ParseOutputMarkers(output, markers)

	if len(result) != 0 {
		t.Errorf("Expected empty map for missing marker, got %v", result)
	}
}

func TestParseOutputMarkers_CustomMarkers(t *testing.T) {
	// Test with completely custom markers that aren't built-in
	output := `Workflow completed.
CUSTOM_STATUS: approved
USER_FEEDBACK: looks good to me
NEXT_ACTION: merge`

	markers := []string{"CUSTOM_STATUS:", "USER_FEEDBACK:", "NEXT_ACTION:"}
	result := ParseOutputMarkers(output, markers)

	if len(result) != 3 {
		t.Errorf("Expected 3 markers, got %d", len(result))
	}
	if result["CUSTOM_STATUS:"] != "approved" {
		t.Error("Wrong CUSTOM_STATUS")
	}
	if result["USER_FEEDBACK:"] != "looks good to me" {
		t.Error("Wrong USER_FEEDBACK")
	}
	if result["NEXT_ACTION:"] != "merge" {
		t.Error("Wrong NEXT_ACTION")
	}
}

func TestSplitMarkerValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"none", 0},
		{"None", 0},
		{"single.go", 1},
		{"a.go, b.go", 2},
		{"a.go,b.go,c.go", 3},
		{" a.go , b.go , c.go ", 3},
	}

	for _, tc := range tests {
		result := SplitMarkerValues(tc.input)
		if len(result) != tc.expected {
			t.Errorf("For input %q, expected %d values, got %d: %v", tc.input, tc.expected, len(result), result)
		}
	}
}

func TestHasMarkerValue(t *testing.T) {
	output := `IMPLEMENTATION_COMPLETE: true
STATUS: success`

	if !HasMarkerValue(output, "IMPLEMENTATION_COMPLETE:", "true") {
		t.Error("Should find IMPLEMENTATION_COMPLETE: true")
	}
	if !HasMarkerValue(output, "IMPLEMENTATION_COMPLETE:", "TRUE") {
		t.Error("Should be case-insensitive")
	}
	if HasMarkerValue(output, "IMPLEMENTATION_COMPLETE:", "false") {
		t.Error("Should not match wrong value")
	}
	if HasMarkerValue(output, "NONEXISTENT:", "value") {
		t.Error("Should not match nonexistent marker")
	}
}

func TestExtractMarkerValue(t *testing.T) {
	tests := []struct {
		output   string
		marker   string
		expected string
	}{
		{"PATH: /tmp/test.md", "PATH:", "/tmp/test.md"},
		{"PATH:/tmp/test.md", "PATH:", "/tmp/test.md"},
		{"PATH:  /tmp/test.md  \n", "PATH:", "/tmp/test.md"},
		{"No marker here", "PATH:", ""},
		{"MULTI: a, b, c", "MULTI:", "a, b, c"},
	}

	for _, tc := range tests {
		result := extractMarkerValue(tc.output, tc.marker)
		if result != tc.expected {
			t.Errorf("For output %q, marker %q: expected %q, got %q", tc.output, tc.marker, tc.expected, result)
		}
	}
}

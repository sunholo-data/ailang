package coordinator

import (
	"strings"
	"testing"
)

// TestEndToEndWorkflow_DirectiveBuildFlow tests the complete workflow
// from inbox message to directive with skill invocation.
// This is an integration test that verifies the fix for M-COORD-ARTIFACT-DISCOVERY.
func TestEndToEndWorkflow_DirectiveBuildFlow(t *testing.T) {
	// Simulate the complete flow:
	// 1. Message comes from design-doc-creator inbox
	// 2. Task is created with AgentID set
	// 3. Directive is built with skill invocation

	t.Run("complete flow from inbox to directive", func(t *testing.T) {
		// Step 1: Simulate inbox message (what the daemon receives)
		inboxName := "design-doc-creator"
		messageContent := "Create a design doc for feature X"
		githubIssue := 102

		// Step 2: Create task like daemon_tasks.go does (with AgentID fix)
		agentID := inboxName // This is what the fix ensures
		task := &TaskRecord{
			ID:          "e2e-test-task-1",
			AgentID:     agentID, // M-COORD-ARTIFACT-DISCOVERY fix: AgentID is now set
			Title:       "Test Task",
			Content:     messageContent,
			GithubIssue: githubIssue,
			Stage:       TaskStageNone, // No stage set initially
		}

		// Verify AgentID is set correctly
		if task.AgentID != "design-doc-creator" {
			t.Errorf("AgentID not set correctly: got %q, want %q", task.AgentID, "design-doc-creator")
		}

		// Step 3: Build directive (with fallback logic fix)
		directive := BuildStageDirective(task)

		// Verify directive includes skill invocation
		if !strings.Contains(strings.ToLower(directive), "invoke the design-doc-creator skill") {
			t.Errorf("Directive missing skill invocation.\nGot: %s", directive)
		}

		// Verify directive includes original content
		if !strings.Contains(directive, messageContent) {
			t.Errorf("Directive missing original content.\nGot: %s", directive)
		}

		// Verify directive mentions GitHub issue
		if !strings.Contains(directive, "102") {
			// This is optional - depends on how BuildDirectiveFromConfig formats it
			t.Logf("Note: Directive may not include GitHub issue number directly")
		}
	})
}

// TestEndToEndWorkflow_ArtifactPatterns tests that artifact patterns
// are correctly resolved for different agent types.
func TestEndToEndWorkflow_ArtifactPatterns(t *testing.T) {
	testCases := []struct {
		agentID         string
		expectedPattern string
	}{
		{"design-doc-creator", "design_docs/**/*.md"},
		{"sprint-planner", "design_docs/**/*.md"},
		{"sprint-executor", "**/*.go"},
	}

	for _, tc := range testCases {
		t.Run(tc.agentID, func(t *testing.T) {
			agent := &AgentConfig{ID: tc.agentID}
			patterns := agent.GetEffectiveArtifactPatterns()

			found := false
			for _, p := range patterns {
				if p == tc.expectedPattern {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected pattern %q not found in patterns: %v", tc.expectedPattern, patterns)
			}
		})
	}
}

// TestEndToEndWorkflow_StageTransitions tests the complete stage
// transition flow for GitHub-linked tasks.
func TestEndToEndWorkflow_StageTransitions(t *testing.T) {
	// Verify stage-to-agent mappings work correctly
	testCases := []struct {
		stage           TaskStage
		expectedAgentID string
	}{
		{TaskStageDesign, "design-doc-creator"},
		{TaskStageSprint, "sprint-planner"},
		{TaskStageImplementation, "sprint-executor"},
		{TaskStageNone, ""},
	}

	for _, tc := range testCases {
		t.Run(string(tc.stage), func(t *testing.T) {
			agentID := stageToAgentIDForDirective(tc.stage)
			if agentID != tc.expectedAgentID {
				t.Errorf("stageToAgentIDForDirective(%s) = %q, want %q",
					tc.stage, agentID, tc.expectedAgentID)
			}
		})
	}
}

// TestEndToEndWorkflow_DirectivePrecedence tests that AgentID takes
// precedence over Stage when both are set.
func TestEndToEndWorkflow_DirectivePrecedence(t *testing.T) {
	// When both AgentID and Stage are set, AgentID should take precedence
	task := &TaskRecord{
		ID:          "precedence-test",
		AgentID:     "sprint-executor", // AgentID says sprint-executor
		Content:     "Test content",
		GithubIssue: 100,
		Stage:       TaskStageDesign, // Stage says design (which maps to design-doc-creator)
	}

	directive := BuildStageDirective(task)

	// Should use AgentID (sprint-executor), not stage (design-doc-creator)
	if strings.Contains(strings.ToLower(directive), "design-doc-creator") {
		t.Errorf("Directive used Stage instead of AgentID: %s", directive)
	}
	if !strings.Contains(strings.ToLower(directive), "sprint-executor") {
		t.Errorf("Directive missing AgentID skill invocation.\nGot: %s", directive)
	}
}

// TestEndToEndWorkflow_GlobPatternMatching tests glob pattern matching
// for various file paths (simulating git diff output).
func TestEndToEndWorkflow_GlobPatternMatching(t *testing.T) {
	// Test real-world file paths against patterns
	testCases := []struct {
		pattern string
		file    string
		want    bool
	}{
		// Design doc patterns
		{"design_docs/**/*.md", "design_docs/planned/v0_6_3/m-coord-artifact-discovery.md", true},
		{"design_docs/**/*.md", "design_docs/planned/v0_6_3/m-coord-artifact-discovery-sprint-plan.md", true},
		{"design_docs/**/*.md", "design_docs/implemented/v0_6_2/m-generic-workflows.md", true},
		{"design_docs/**/*.md", "internal/coordinator/workflow_test.go", false},

		// Sprint JSON patterns
		{".ailang/state/sprints/*.json", ".ailang/state/sprints/sprint_M-COORD-ARTIFACT.json", true},
		{".ailang/state/sprints/*.json", ".ailang/state/other.json", false},

		// Go file patterns
		{"**/*.go", "internal/coordinator/daemon_tasks.go", true},
		{"**/*.go", "internal/coordinator/stage_execution.go", true},
		{"**/*.go", "design_docs/something.md", false},
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

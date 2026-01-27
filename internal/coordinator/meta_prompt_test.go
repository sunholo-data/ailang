package coordinator

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_IncludesMetaPrompt(t *testing.T) {
	prompt := BuildSystemPrompt(TaskTypeBugFix, nil)

	// Must include critical rules
	if !strings.Contains(prompt, "INVOKE YOUR SKILL") {
		t.Error("system prompt missing INVOKE YOUR SKILL rule")
	}
	if !strings.Contains(prompt, "MAKE CODE CHANGES") {
		t.Error("system prompt missing MAKE CODE CHANGES rule")
	}
	if !strings.Contains(prompt, "COMMIT YOUR WORK") {
		t.Error("system prompt missing COMMIT YOUR WORK rule")
	}
}

func TestBuildSystemPrompt_TaskTypeContext(t *testing.T) {
	tests := []struct {
		taskType TaskType
		contains string
	}{
		{TaskTypeBugFix, "BUG FIX"},
		{TaskTypeFeature, "FEATURE IMPLEMENTATION"},
		{TaskTypeRefactor, "REFACTORING"},
		{TaskTypeTest, "TESTING"},
		{TaskTypeDocs, "DOCUMENTATION"},
		{TaskTypeResearch, "RESEARCH"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			prompt := BuildSystemPrompt(tt.taskType, nil)
			if !strings.Contains(prompt, tt.contains) {
				t.Errorf("system prompt for %s missing %q", tt.taskType, tt.contains)
			}
		})
	}
}

func TestBuildSystemPrompt_UnknownTaskType(t *testing.T) {
	prompt := BuildSystemPrompt(TaskTypeUnknown, nil)

	// Should still have meta-prompt but no TASK CONTEXT line
	if !strings.Contains(prompt, "INVOKE YOUR SKILL") {
		t.Error("system prompt missing meta-prompt for unknown task type")
	}
	if strings.Contains(prompt, "TASK CONTEXT:") {
		t.Error("system prompt should not have TASK CONTEXT for unknown type")
	}
}

func TestBuildSystemPrompt_WithAgentSystemPrompt(t *testing.T) {
	agent := &AgentConfig{
		ID:           "test-agent",
		SystemPrompt: "Always use Go 1.22 features.",
	}

	prompt := BuildSystemPrompt(TaskTypeBugFix, agent)

	if !strings.Contains(prompt, "Always use Go 1.22 features.") {
		t.Error("system prompt missing agent-specific instructions")
	}
	if !strings.Contains(prompt, "AGENT-SPECIFIC INSTRUCTIONS") {
		t.Error("system prompt missing AGENT-SPECIFIC INSTRUCTIONS header")
	}
}

func TestBuildSystemPrompt_NilAgentConfig(t *testing.T) {
	// Should not panic with nil agent config
	prompt := BuildSystemPrompt(TaskTypeFeature, nil)
	if prompt == "" {
		t.Error("system prompt should not be empty with nil agent config")
	}
}

func TestBuildSystemPrompt_EmptyAgentSystemPrompt(t *testing.T) {
	agent := &AgentConfig{
		ID:           "test-agent",
		SystemPrompt: "", // Empty - should not add agent section
	}

	prompt := BuildSystemPrompt(TaskTypeBugFix, agent)

	if strings.Contains(prompt, "AGENT-SPECIFIC INSTRUCTIONS") {
		t.Error("system prompt should not have agent section when SystemPrompt is empty")
	}
}

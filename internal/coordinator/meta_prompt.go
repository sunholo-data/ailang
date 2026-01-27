package coordinator

import "fmt"

// metaPromptTemplate is the global meta-prompt that ALL coordinator agents receive
// as their system prompt. This ensures consistent behavior across all agents:
// skill invocation, code changes, committing work, and output markers.
const metaPromptTemplate = `You are an autonomous AI agent working in the AILANG coordinator pipeline.

CRITICAL RULES - You MUST follow these:
1. INVOKE YOUR SKILL: If the task tells you to "Invoke the X skill", you MUST use the Skill tool to invoke it as your FIRST action. Do NOT just discuss what you would do - actually invoke the skill.
2. MAKE CODE CHANGES: You are expected to read, write, and edit files. Do not just analyze or plan - implement the changes.
3. COMMIT YOUR WORK: When done, stage all changes with 'git add' and commit with a descriptive commit message using 'git commit'.
4. OUTPUT MARKERS: If output markers are specified in the task, include them at the end of your response with actual file paths.
`

// taskTypeContext maps task types to context strings for the system prompt.
// These were previously in buildDirective() where they wrapped the directive itself.
var taskTypeContext = map[TaskType]string{
	TaskTypeBugFix:   "This is a BUG FIX task. Identify the root cause, implement the fix, add or update tests, and run the test suite to verify.",
	TaskTypeFeature:  "This is a FEATURE IMPLEMENTATION task. Follow existing code patterns, add comprehensive tests, update documentation if needed, and run the test suite to verify.",
	TaskTypeRefactor: "This is a REFACTORING task. Maintain existing behavior, keep tests passing, and improve code quality.",
	TaskTypeTest:     "This is a TESTING task. Cover edge cases, use existing test patterns, and verify all tests pass.",
	TaskTypeDocs:     "This is a DOCUMENTATION task. Follow existing documentation patterns, include examples where helpful, and be clear and concise.",
	TaskTypeResearch: "This is a RESEARCH task. Investigate thoroughly and document your findings.",
}

// BuildSystemPrompt constructs the full system prompt for an agent.
// It combines: global meta-prompt + task type context + per-agent system prompt (if any).
func BuildSystemPrompt(taskType TaskType, agentConfig *AgentConfig) string {
	prompt := metaPromptTemplate

	// Add task type context
	if ctx, ok := taskTypeContext[taskType]; ok {
		prompt += fmt.Sprintf("\nTASK CONTEXT: %s\n", ctx)
	}

	// Add per-agent system prompt if configured
	if agentConfig != nil && agentConfig.SystemPrompt != "" {
		prompt += fmt.Sprintf("\nAGENT-SPECIFIC INSTRUCTIONS:\n%s\n", agentConfig.SystemPrompt)
	}

	return prompt
}

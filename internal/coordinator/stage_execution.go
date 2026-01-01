package coordinator

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// StageExecutionResult contains extracted artifacts from stage execution
type StageExecutionResult struct {
	// Design doc stage
	DesignDocPath string

	// Sprint plan stage
	SprintPlanPath string

	// Implementation stage
	BranchName    string
	FilesCreated  []string
	FilesModified []string

	// Common fields
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int
}

// BuildStageDirective creates a stage-appropriate directive for GitHub-linked tasks.
// This is the key integration point - tasks at different stages get different prompts
// that invoke the appropriate skills.
//
// If an AgentConfig is provided with InvokeConfig, it uses the config-driven approach.
// Otherwise, it falls back to the legacy hardcoded directives.
func BuildStageDirective(task *TaskRecord) string {
	// For non-GitHub tasks, use original content as-is
	if task.GithubIssue == 0 || task.Stage == TaskStageNone {
		return task.Content
	}

	// Build stage-specific directive that invokes the appropriate skill
	switch task.Stage {
	case TaskStageDesign:
		return buildDesignDirective(task)
	case TaskStageSprint:
		return buildSprintDirective(task)
	case TaskStageImplementation:
		return buildImplementationDirective(task)
	default:
		return task.Content
	}
}

// BuildDirectiveFromConfig creates a directive based on agent configuration.
// This is the config-driven approach that replaces hardcoded skill names.
//
// Supports three invoke types:
//   - "skill": Invokes a Claude Code skill by name (e.g., "/design-doc-creator")
//   - "agent": Sends a message to another agent (e.g., "sprint-planner")
//   - "prompt": Uses a custom template with variable substitution
//
// Template variables available for prompt type:
//   - {{.TaskID}}: The task ID
//   - {{.GithubIssue}}: The GitHub issue number
//   - {{.Content}}: The original task content
//   - {{.Stage}}: The current task stage
//   - {{.OutputMarkers}}: Comma-separated list of expected output markers
func BuildDirectiveFromConfig(task *TaskRecord, agent *AgentConfig) string {
	if agent == nil || agent.Invoke == nil {
		// Fall back to legacy behavior
		return BuildStageDirective(task)
	}

	invoke := agent.Invoke

	switch invoke.Type {
	case "skill":
		return buildSkillDirective(task, agent)
	case "agent":
		return buildAgentHandoffDirective(task, agent)
	case "prompt":
		return buildTemplateDirective(task, agent)
	default:
		// Unknown type, fall back to legacy
		return BuildStageDirective(task)
	}
}

// buildSkillDirective creates a directive that invokes a skill by name.
func buildSkillDirective(task *TaskRecord, agent *AgentConfig) string {
	skillName := agent.Invoke.Name
	markers := formatOutputMarkers(agent.OutputMarkers)

	var sb strings.Builder
	if task.GithubIssue > 0 {
		sb.WriteString(fmt.Sprintf("GitHub issue #%d: %s\n\n", task.GithubIssue, task.Content))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n\n", task.Content))
	}

	sb.WriteString(fmt.Sprintf("Invoke the %s skill to complete this task.\n", skillName))

	if len(agent.OutputMarkers) > 0 {
		sb.WriteString(fmt.Sprintf("\n**REQUIRED**: After completion, output these markers:\n%s", markers))
	}

	return sb.String()
}

// buildAgentHandoffDirective creates a directive that hands off to another agent.
func buildAgentHandoffDirective(task *TaskRecord, agent *AgentConfig) string {
	targetAgent := agent.Invoke.Name
	markers := formatOutputMarkers(agent.OutputMarkers)

	var sb strings.Builder
	if task.GithubIssue > 0 {
		sb.WriteString(fmt.Sprintf("GitHub issue #%d: %s\n\n", task.GithubIssue, task.Content))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n\n", task.Content))
	}

	sb.WriteString(fmt.Sprintf("Hand off this task to the %s agent for processing.\n", targetAgent))

	if len(agent.OutputMarkers) > 0 {
		sb.WriteString(fmt.Sprintf("\n**REQUIRED**: After completion, output these markers:\n%s", markers))
	}

	return sb.String()
}

// buildTemplateDirective creates a directive from a custom template.
// Template uses Go text/template syntax with {{.Field}} placeholders.
func buildTemplateDirective(task *TaskRecord, agent *AgentConfig) string {
	template := agent.Invoke.Template
	if template == "" {
		return task.Content
	}

	// Simple variable substitution (not full Go template to avoid complexity)
	result := template
	result = strings.ReplaceAll(result, "{{.TaskID}}", task.ID)
	result = strings.ReplaceAll(result, "{{.GithubIssue}}", fmt.Sprintf("%d", task.GithubIssue))
	result = strings.ReplaceAll(result, "{{.Content}}", task.Content)
	result = strings.ReplaceAll(result, "{{.Stage}}", string(task.Stage))
	result = strings.ReplaceAll(result, "{{.OutputMarkers}}", strings.Join(agent.OutputMarkers, ", "))

	return result
}

// formatOutputMarkers formats the output markers for display in directives.
func formatOutputMarkers(markers []string) string {
	if len(markers) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, marker := range markers {
		sb.WriteString(fmt.Sprintf("- %s <value>\n", marker))
	}
	return sb.String()
}

// ParseOutputMarkers extracts values for configured markers from execution output.
// This is the config-driven approach that replaces the hardcoded ParseStageOutput.
//
// It searches for each marker in the output and extracts the value that follows.
// Markers should be in the format "MARKER_NAME:" (with or without trailing colon).
//
// Returns a map of marker name -> extracted value.
// Returns empty map if no markers are found or markers slice is empty.
//
// For multi-value markers (comma-separated), the full comma-separated string is returned.
// Use splitMarkerValues() to split into individual values if needed.
func ParseOutputMarkers(output string, markers []string) map[string]string {
	if len(markers) == 0 {
		return make(map[string]string)
	}

	result := make(map[string]string)
	for _, marker := range markers {
		value := extractMarkerValue(output, marker)
		if value != "" {
			result[marker] = value
		}
	}
	return result
}

// extractMarkerValue extracts the value for a single marker from output.
// The marker can be specified with or without trailing colon.
// Handles both "MARKER: value" and "MARKER:value" formats.
func extractMarkerValue(output, marker string) string {
	// Normalize marker to not have trailing colon for pattern building
	markerBase := strings.TrimSuffix(marker, ":")

	// Build pattern to match "MARKER:" followed by value
	// Captures everything until end of line
	pattern := fmt.Sprintf(`%s:\s*(.+?)(?:\n|$)`, regexp.QuoteMeta(markerBase))
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// SplitMarkerValues splits a comma-separated marker value into individual values.
// Useful for markers like "FILES_CREATED: file1.go, file2.go, file3.go".
// Returns nil if value is empty or "none"/"None".
func SplitMarkerValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "none" || value == "None" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// HasMarkerValue checks if a specific marker value is present in output.
// Useful for boolean markers like "IMPLEMENTATION_COMPLETE: true".
func HasMarkerValue(output, marker, expectedValue string) bool {
	markers := ParseOutputMarkers(output, []string{marker})
	if val, ok := markers[marker]; ok {
		return strings.EqualFold(strings.TrimSpace(val), expectedValue)
	}
	return false
}

// buildDesignDirective creates a directive that invokes design-doc-creator skill.
//
// Deprecated: Use BuildDirectiveFromConfig with InvokeConfig{Type: "skill", Name: "design-doc-creator"}
// This legacy function is retained for backwards compatibility with agents that don't have InvokeConfig.
func buildDesignDirective(task *TaskRecord) string {
	return fmt.Sprintf(`GitHub issue #%d: %s

Invoke the design-doc-creator skill to create a design document for this request.

**REQUIRED**: After creating the design doc, output exactly this line:
DESIGN_DOC_PATH: design_docs/planned/<version>/<name>.md

This path will be posted to GitHub for review.`, task.GithubIssue, task.Content)
}

// buildSprintDirective creates a directive that invokes sprint-planner skill.
//
// Deprecated: Use BuildDirectiveFromConfig with InvokeConfig{Type: "skill", Name: "sprint-planner"}
// This legacy function is retained for backwards compatibility with agents that don't have InvokeConfig.
func buildSprintDirective(task *TaskRecord) string {
	return fmt.Sprintf(`GitHub issue #%d: %s

Design document approved. Invoke the sprint-planner skill to create a sprint plan.

When done, output: SPRINT_PLAN_PATH: <path-to-created-plan>`, task.GithubIssue, task.Content)
}

// buildImplementationDirective creates a directive that invokes sprint-executor skill.
//
// Deprecated: Use BuildDirectiveFromConfig with InvokeConfig{Type: "skill", Name: "sprint-executor"}
// This legacy function is retained for backwards compatibility with agents that don't have InvokeConfig.
func buildImplementationDirective(task *TaskRecord) string {
	return fmt.Sprintf(`GitHub issue #%d: %s

Sprint plan approved. Invoke the sprint-executor skill to implement the plan.

When done, output:
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: <branch-name>
FILES_CREATED: <files>
FILES_MODIFIED: <files>`, task.GithubIssue, task.Content)
}

// ParseStageOutput extracts structured artifacts from execution output
func ParseStageOutput(output string, stage TaskStage) *StageExecutionResult {
	result := &StageExecutionResult{}

	switch stage {
	case TaskStageDesign:
		result.DesignDocPath = extractPath(output, `DESIGN_DOC_PATH:\s*(.+\.md)`)
	case TaskStageSprint:
		result.SprintPlanPath = extractPath(output, `SPRINT_PLAN_PATH:\s*(.+\.md)`)
	case TaskStageImplementation:
		if strings.Contains(output, "IMPLEMENTATION_COMPLETE: true") ||
			strings.Contains(output, "IMPLEMENTATION_COMPLETE:true") {
			result.BranchName = extractPath(output, `BRANCH_NAME:\s*(\S+)`)
			result.FilesCreated = extractList(output, `FILES_CREATED:\s*(.+)`)
			result.FilesModified = extractList(output, `FILES_MODIFIED:\s*(.+)`)
		}
	}

	return result
}

// extractPath extracts a single path from output using regex
func extractPath(output, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractList extracts a comma-separated list from output
func extractList(output, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		raw := strings.TrimSpace(matches[1])
		if raw == "" || raw == "none" || raw == "None" {
			return nil
		}
		parts := strings.Split(raw, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}
	return nil
}

// ProcessStageCompletion handles the completion of a stage for GitHub-linked tasks.
// It calls the appropriate TaskChain callback and handles stage transitions.
func (d *Daemon) ProcessStageCompletion(ctx context.Context, task *TaskRecord, execResult *ExecuteResult) error {
	if task.GithubIssue == 0 || task.Stage == TaskStageNone || d.taskChain == nil {
		return nil // Not a GitHub-linked pipeline task
	}

	// Parse the output to extract artifacts
	stageResult := ParseStageOutput(execResult.Output, task.Stage)
	stageResult.Duration = execResult.Duration
	stageResult.Cost = execResult.Cost
	stageResult.TokensUsed = execResult.TokensUsed
	stageResult.InputTokens = execResult.InputTokens
	stageResult.OutputTokens = execResult.OutputTokens

	d.logger.Printf("Processing stage completion for task %s (stage: %s, issue: #%d)",
		task.ID, task.Stage, task.GithubIssue)

	switch task.Stage {
	case TaskStageDesign:
		if stageResult.DesignDocPath == "" {
			d.logger.Printf("Warning: No design doc path found in output for task %s", task.ID)
			// Still notify GitHub but without the path
		}
		return d.taskChain.OnDesignDocComplete(ctx, task.ID, &DesignDocResult{
			Path:         stageResult.DesignDocPath,
			Duration:     stageResult.Duration,
			Cost:         stageResult.Cost,
			TokensUsed:   stageResult.TokensUsed,
			InputTokens:  stageResult.InputTokens,
			OutputTokens: stageResult.OutputTokens,
		})

	case TaskStageSprint:
		if stageResult.SprintPlanPath == "" {
			d.logger.Printf("Warning: No sprint plan path found in output for task %s", task.ID)
		}
		return d.taskChain.OnSprintPlanComplete(ctx, task.ID, &SprintPlanResult{
			Path:         stageResult.SprintPlanPath,
			Duration:     stageResult.Duration,
			Cost:         stageResult.Cost,
			TokensUsed:   stageResult.TokensUsed,
			InputTokens:  stageResult.InputTokens,
			OutputTokens: stageResult.OutputTokens,
		})

	case TaskStageImplementation:
		return d.taskChain.OnImplementationComplete(ctx, task.ID, &ImplementResult{
			BranchName:    stageResult.BranchName,
			WorktreePath:  task.WorktreePath,
			Duration:      stageResult.Duration,
			Cost:          stageResult.Cost,
			TokensUsed:    stageResult.TokensUsed,
			InputTokens:   stageResult.InputTokens,
			OutputTokens:  stageResult.OutputTokens,
			FilesCreated:  stageResult.FilesCreated,
			FilesModified: stageResult.FilesModified,
		})
	}

	return nil
}

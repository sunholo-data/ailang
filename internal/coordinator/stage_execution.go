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
// This function uses the config-driven approach by creating a temporary AgentConfig
// with the stage-appropriate agent ID, which triggers the effective defaults.
func BuildStageDirective(task *TaskRecord) string {
	// M-COORD-ARTIFACT-DISCOVERY: Only fall back to raw content if we truly have no agent info
	// (BOTH AgentID empty AND Stage is None). Previously used || which was wrong.
	if task.AgentID == "" && task.Stage == TaskStageNone {
		return task.Content
	}

	// Use AgentID if available (from inbox), otherwise derive from Stage (legacy path)
	agentID := task.AgentID
	if agentID == "" {
		agentID = stageToAgentIDForDirective(task.Stage)
	}

	if agentID == "" {
		return task.Content
	}

	// Use config-driven approach with effective defaults
	agent := &AgentConfig{ID: agentID}
	return BuildDirectiveFromConfig(task, agent)
}

// stageToAgentIDForDirective maps task stages to agent IDs for directive building.
func stageToAgentIDForDirective(stage TaskStage) string {
	switch stage {
	case TaskStageDesign:
		return "design-doc-creator"
	case TaskStageSprint:
		return "sprint-planner"
	case TaskStageImplementation:
		return "sprint-executor"
	default:
		return ""
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
	if agent == nil {
		// No agent config, fall back to legacy behavior
		return BuildStageDirective(task)
	}

	// Use effective config (includes defaults for known agents)
	invoke := agent.GetEffectiveInvokeConfig()
	if invoke == nil {
		// Unknown agent with no config, fall back to legacy
		return BuildStageDirective(task)
	}

	switch invoke.Type {
	case "skill":
		return buildSkillDirectiveWithConfig(task, agent, invoke)
	case "agent":
		return buildAgentHandoffDirectiveWithConfig(task, agent, invoke)
	case "prompt":
		return buildTemplateDirective(task, agent)
	default:
		// Unknown type, fall back to legacy
		return BuildStageDirective(task)
	}
}

// buildSkillDirectiveWithConfig creates a directive that invokes a skill by name.
// Uses effective config (explicit or defaults) for skill name and output markers.
func buildSkillDirectiveWithConfig(task *TaskRecord, agent *AgentConfig, invoke *InvokeConfig) string {
	skillName := invoke.Name
	effectiveMarkers := agent.GetEffectiveOutputMarkers()

	var sb strings.Builder
	if task.GithubIssue > 0 {
		sb.WriteString(fmt.Sprintf("GitHub issue #%d: %s\n\n", task.GithubIssue, task.Content))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n\n", task.Content))
	}

	sb.WriteString(fmt.Sprintf("Invoke the %s skill to complete this task.\n", skillName))

	// Add output markers as a prominent suffix - these are CRITICAL for coordinator to track artifacts
	if len(effectiveMarkers) > 0 {
		sb.WriteString("\n---\n\n")
		sb.WriteString("## CRITICAL: Coordinator Output Requirements\n\n")
		sb.WriteString("**You MUST include these exact markers at the END of your response.**\n")
		sb.WriteString("The coordinator parses these to track artifacts and post updates to GitHub.\n\n")
		sb.WriteString("```\n")
		for _, marker := range effectiveMarkers {
			sb.WriteString(fmt.Sprintf("%s <path-to-file>\n", marker))
		}
		sb.WriteString("```\n\n")
		sb.WriteString("**Example format:**\n")
		sb.WriteString("```\n")
		for _, marker := range effectiveMarkers {
			// Show example with backticks for markdown
			sb.WriteString(fmt.Sprintf("**%s** `design_docs/planned/v0_6_3/example.md`\n", marker))
		}
		sb.WriteString("```\n")
	}

	return sb.String()
}

// buildAgentHandoffDirectiveWithConfig creates a directive that hands off to another agent.
// Uses effective config (explicit or defaults) for agent name and output markers.
func buildAgentHandoffDirectiveWithConfig(task *TaskRecord, agent *AgentConfig, invoke *InvokeConfig) string {
	targetAgent := invoke.Name
	effectiveMarkers := agent.GetEffectiveOutputMarkers()

	var sb strings.Builder
	if task.GithubIssue > 0 {
		sb.WriteString(fmt.Sprintf("GitHub issue #%d: %s\n\n", task.GithubIssue, task.Content))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n\n", task.Content))
	}

	sb.WriteString(fmt.Sprintf("Hand off this task to the %s agent for processing.\n", targetAgent))

	// Add output markers as a prominent suffix - these are CRITICAL for coordinator to track artifacts
	if len(effectiveMarkers) > 0 {
		sb.WriteString("\n---\n\n")
		sb.WriteString("## CRITICAL: Coordinator Output Requirements\n\n")
		sb.WriteString("**You MUST include these exact markers at the END of your response.**\n")
		sb.WriteString("The coordinator parses these to track artifacts and post updates to GitHub.\n\n")
		sb.WriteString("```\n")
		for _, marker := range effectiveMarkers {
			sb.WriteString(fmt.Sprintf("%s <path-to-file>\n", marker))
		}
		sb.WriteString("```\n\n")
		sb.WriteString("**Example format:**\n")
		sb.WriteString("```\n")
		for _, marker := range effectiveMarkers {
			sb.WriteString(fmt.Sprintf("**%s** `design_docs/planned/v0_6_3/example.md`\n", marker))
		}
		sb.WriteString("```\n")
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
// Handles both plain text and markdown-formatted markers:
//   - MARKER: value
//   - MARKER:value
//   - **MARKER**: value
//   - **MARKER**: `value`
func extractMarkerValue(output, marker string) string {
	// Normalize marker to not have trailing colon for pattern building
	markerBase := strings.TrimSuffix(marker, ":")

	// Build pattern that handles optional ** (bold) and optional backticks
	// Pattern: **?MARKER**?: `?value`?
	pattern := fmt.Sprintf(`\*{0,2}%s\*{0,2}:\s*`+"`?"+`(.+?)`+"`?"+`(?:\n|$)`, regexp.QuoteMeta(markerBase))
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		value := strings.TrimSpace(matches[1])
		// Remove any remaining backticks
		value = strings.Trim(value, "`")
		return value
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

// ParseStageOutput extracts structured artifacts from execution output.
// Handles both plain text markers and markdown-formatted markers:
//   - DESIGN_DOC_PATH: path.md
//   - **DESIGN_DOC_PATH**: `path.md`
func ParseStageOutput(output string, stage TaskStage) *StageExecutionResult {
	result := &StageExecutionResult{}

	switch stage {
	case TaskStageDesign:
		// Handle: DESIGN_DOC_PATH: path.md OR **DESIGN_DOC_PATH**: `path.md`
		result.DesignDocPath = extractPathWithMarkdown(output, "DESIGN_DOC_PATH")
	case TaskStageSprint:
		// Handle: SPRINT_PLAN_PATH: path.md OR **SPRINT_PLAN_PATH**: `path.md`
		result.SprintPlanPath = extractPathWithMarkdown(output, "SPRINT_PLAN_PATH")
	case TaskStageImplementation:
		if strings.Contains(output, "IMPLEMENTATION_COMPLETE: true") ||
			strings.Contains(output, "IMPLEMENTATION_COMPLETE:true") ||
			strings.Contains(output, "**IMPLEMENTATION_COMPLETE**: true") ||
			strings.Contains(output, "**IMPLEMENTATION_COMPLETE**:true") {
			result.BranchName = extractPathWithMarkdown(output, "BRANCH_NAME")
			result.FilesCreated = extractListWithMarkdown(output, "FILES_CREATED")
			result.FilesModified = extractListWithMarkdown(output, "FILES_MODIFIED")
		}
	}

	return result
}

// extractPathWithMarkdown extracts a path value that may have markdown formatting.
// Handles: MARKER: value, **MARKER**: value, MARKER: `value`, **MARKER**: `value`
func extractPathWithMarkdown(output, marker string) string {
	// Pattern handles optional ** (bold) around marker and optional backticks around value
	pattern := fmt.Sprintf(`\*{0,2}%s\*{0,2}:\s*`+"`?"+`([^`+"`"+`\n]+?)`+"`?"+`(?:\s|$|\n)`, regexp.QuoteMeta(marker))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractListWithMarkdown extracts a comma-separated list that may have markdown formatting.
// Uses a pattern that matches until end of line (unlike extractPathWithMarkdown which stops at whitespace).
func extractListWithMarkdown(output, marker string) []string {
	// Pattern matches until end of line for comma-separated lists
	pattern := fmt.Sprintf(`\*{0,2}%s\*{0,2}:\s*(.+?)(?:\n|$)`, regexp.QuoteMeta(marker))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil
	}

	value := strings.TrimSpace(matches[1])
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}

	// Handle backtick-wrapped items: `file1.go`, `file2.go`
	value = strings.ReplaceAll(value, "`", "")

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
//
// Artifact discovery strategy (in order of preference):
// 1. Git diff + artifact patterns (deterministic, reliable)
// 2. Output markers (fallback for backwards compatibility)
func (d *Daemon) ProcessStageCompletion(ctx context.Context, task *TaskRecord, execResult *ExecuteResult) error {
	if task.GithubIssue == 0 || task.Stage == TaskStageNone || d.taskChain == nil {
		return nil // Not a GitHub-linked pipeline task
	}

	d.logger.Printf("Processing stage completion for task %s (stage: %s, issue: #%d, agent: %s)",
		task.ID, task.Stage, task.GithubIssue, task.AgentID)

	// Get agent config for artifact patterns
	// Prefer task.AgentID if set, otherwise fall back to stage-to-agent mapping
	agentID := task.AgentID
	if agentID == "" {
		agentID = stageToAgentIDForDirective(task.Stage)
	}
	agent := &AgentConfig{ID: agentID}
	patterns := agent.GetEffectiveArtifactPatterns()
	d.logger.Printf("Using artifact patterns for agent %s: %v", agentID, patterns)

	// Primary: Discover artifacts via git diff (deterministic)
	var discoveredArtifacts []string
	if task.WorktreePath != "" && len(patterns) > 0 {
		discovery := NewArtifactDiscovery(task.WorktreePath, patterns)
		artifacts, err := discovery.DiscoverChangedFiles()
		if err != nil {
			d.logger.Printf("Warning: Failed to discover artifacts via git diff: %v", err)
		} else {
			discoveredArtifacts = artifacts
			d.logger.Printf("Discovered %d artifacts via git diff: %v", len(artifacts), artifacts)
		}
	}

	// Fallback: Parse output markers (for backwards compatibility)
	stageResult := ParseStageOutput(execResult.Output, task.Stage)
	stageResult.Duration = execResult.Duration
	stageResult.Cost = execResult.Cost
	stageResult.TokensUsed = execResult.TokensUsed
	stageResult.InputTokens = execResult.InputTokens
	stageResult.OutputTokens = execResult.OutputTokens

	switch task.Stage {
	case TaskStageDesign:
		// M-COORD-ARTIFACT-DISCOVERY: Prefer git-discovered artifacts over output markers
		// This is more reliable than parsing agent output for markers like DESIGN_DOC_PATH:
		designDocPath := ""
		if len(discoveredArtifacts) > 0 {
			// Use first .md file discovered via git diff
			for _, artifact := range discoveredArtifacts {
				if strings.HasSuffix(artifact, ".md") {
					designDocPath = artifact
					d.logger.Printf("Using git-discovered design doc: %s", designDocPath)
					break
				}
			}
		}
		// Fallback to output markers if git diff found nothing
		if designDocPath == "" && stageResult.DesignDocPath != "" && !strings.Contains(stageResult.DesignDocPath, "*") {
			designDocPath = stageResult.DesignDocPath
			d.logger.Printf("Using marker-based design doc path: %s", designDocPath)
		}
		d.logger.Printf("Design stage: final path=%q, discovered=%d artifacts", designDocPath, len(discoveredArtifacts))
		if designDocPath == "" {
			d.logger.Printf("Warning: No design doc path found for task %s", task.ID)
		}
		return d.taskChain.OnDesignDocComplete(ctx, task.ID, &DesignDocResult{
			Path:         designDocPath,
			Duration:     stageResult.Duration,
			Cost:         stageResult.Cost,
			TokensUsed:   stageResult.TokensUsed,
			InputTokens:  stageResult.InputTokens,
			OutputTokens: stageResult.OutputTokens,
		})

	case TaskStageSprint:
		// Use git-discovered artifact if no path from markers
		sprintPlanPath := stageResult.SprintPlanPath
		// M-COORD-ARTIFACT-DISCOVERY: Also treat glob patterns as invalid paths
		isValidPath := sprintPlanPath != "" && !strings.Contains(sprintPlanPath, "*")
		if !isValidPath && len(discoveredArtifacts) > 0 {
			// Find first sprint plan .md file
			for _, artifact := range discoveredArtifacts {
				if strings.HasPrefix(artifact, "design_docs/") && strings.HasSuffix(artifact, ".md") &&
					strings.Contains(artifact, "sprint") {
					sprintPlanPath = artifact
					d.logger.Printf("Using git-discovered sprint plan: %s", sprintPlanPath)
					break
				}
			}
			// Fallback: any .md in design_docs/
			if sprintPlanPath == "" {
				for _, artifact := range discoveredArtifacts {
					if strings.HasPrefix(artifact, "design_docs/") && strings.HasSuffix(artifact, ".md") {
						sprintPlanPath = artifact
						d.logger.Printf("Using git-discovered artifact as sprint plan: %s", sprintPlanPath)
						break
					}
				}
			}
		}
		if sprintPlanPath == "" {
			d.logger.Printf("Warning: No sprint plan path found for task %s (neither git diff nor markers)", task.ID)
		}
		return d.taskChain.OnSprintPlanComplete(ctx, task.ID, &SprintPlanResult{
			Path:         sprintPlanPath,
			Duration:     stageResult.Duration,
			Cost:         stageResult.Cost,
			TokensUsed:   stageResult.TokensUsed,
			InputTokens:  stageResult.InputTokens,
			OutputTokens: stageResult.OutputTokens,
		})

	case TaskStageImplementation:
		// Use git-discovered files if no list from markers
		filesCreated := stageResult.FilesCreated
		filesModified := stageResult.FilesModified
		if len(filesCreated) == 0 && len(filesModified) == 0 && len(discoveredArtifacts) > 0 {
			// All discovered artifacts are effectively "created or modified"
			filesModified = discoveredArtifacts
			d.logger.Printf("Using git-discovered files as modified: %v", filesModified)
		}
		return d.taskChain.OnImplementationComplete(ctx, task.ID, &ImplementResult{
			BranchName:    stageResult.BranchName,
			WorktreePath:  task.WorktreePath,
			Duration:      stageResult.Duration,
			Cost:          stageResult.Cost,
			TokensUsed:    stageResult.TokensUsed,
			InputTokens:   stageResult.InputTokens,
			OutputTokens:  stageResult.OutputTokens,
			FilesCreated:  filesCreated,
			FilesModified: filesModified,
		})
	}

	return nil
}

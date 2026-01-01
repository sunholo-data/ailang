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

// buildDesignDirective creates a directive that invokes design-doc-creator skill
func buildDesignDirective(task *TaskRecord) string {
	return fmt.Sprintf(`You are working on GitHub issue #%d.

## Task
%s

## Instructions
1. Run the design-doc-creator skill to create a design document for this feature/bug
2. Analyze the issue requirements carefully
3. Create a comprehensive design document following AILANG conventions
4. Output the path to the created design doc in this format:
   DESIGN_DOC_PATH: design_docs/planned/<version>/<doc-name>.md

## Important
- Follow existing design doc patterns in design_docs/implemented/
- Include acceptance criteria
- Estimate effort and complexity
- The design doc will be posted to GitHub for human review
`, task.GithubIssue, task.Content)
}

// buildSprintDirective creates a directive that invokes sprint-planner skill
func buildSprintDirective(task *TaskRecord) string {
	return fmt.Sprintf(`You are continuing work on GitHub issue #%d.
The design document has been approved. Now create a sprint plan.

## Original Request
%s

## Instructions
1. Run the sprint-planner skill to create a sprint plan
2. Read the approved design document
3. Analyze recent velocity and estimate realistic milestones
4. Create a detailed sprint plan with day-by-day tasks
5. Output the path to the created sprint plan in this format:
   SPRINT_PLAN_PATH: design_docs/planned/<version>/<doc-name>-sprint-plan.md

## Important
- Use actual velocity data from CHANGELOG.md
- Include clear acceptance criteria for each milestone
- Create the JSON progress file for multi-session execution
- The sprint plan will be posted to GitHub for human review
`, task.GithubIssue, task.Content)
}

// buildImplementationDirective creates a directive that invokes sprint-executor skill
func buildImplementationDirective(task *TaskRecord) string {
	return fmt.Sprintf(`You are implementing GitHub issue #%d.
The design document and sprint plan have been approved. Now execute the implementation.

## Original Request
%s

## Instructions
1. Run the sprint-executor skill to implement the approved plan
2. Follow test-driven development (TDD)
3. Run tests and linting after each milestone
4. Update CHANGELOG.md progressively
5. When complete, output a summary in this format:
   IMPLEMENTATION_COMPLETE: true
   BRANCH_NAME: <branch-name>
   FILES_CREATED: <comma-separated list>
   FILES_MODIFIED: <comma-separated list>

## Important
- All tests must pass before marking complete
- All linting must pass
- Document any deviations from the plan
- The implementation summary will be posted to GitHub for merge review
`, task.GithubIssue, task.Content)
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

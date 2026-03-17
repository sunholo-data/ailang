package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// Generic Agent Completion Handler (M-GENERIC-PIPELINE)
// =============================================================================
// These handlers support config-driven agent pipelines, replacing the need for
// hardcoded stage-to-agent mappings. Any agent with an ApprovalConfig will
// automatically get the right GitHub labels and comments.

// AgentResult contains the unified result of any agent completion.
// This struct supports all agent types (design-doc-creator, sprint-planner,
// sprint-executor, or any custom agent).
type AgentResult struct {
	// Artifact paths (set whichever is relevant)
	ArtifactPath    string   // Primary artifact (design doc, sprint plan, etc.)
	ArtifactContent string   // Optional: content for GitHub comment display
	AllArtifacts    []string // All discovered artifacts from git diff

	// Execution metrics
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int

	// Implementation-specific (for sprint-executor type agents)
	BranchName    string
	WorktreePath  string
	FilesCreated  []string
	FilesModified []string
}

// OnAgentComplete is the unified handler for any agent completion.
// It uses the agent's ApprovalConfig to determine the appropriate GitHub labels
// and comment template, eliminating the need for hardcoded stage handlers.
//
// This handler:
// 1. Stores the artifact path for later use
// 2. Reads artifact content from worktree (if applicable)
// 3. Posts completion comment to GitHub
// 4. Adds needs-approval label from agent config
//
// For agents without ApprovalConfig, this is a no-op (agent handles its own workflow).
func (tc *TaskChain) OnAgentComplete(ctx context.Context, taskID, agentID string, result *AgentResult, registry *AgentRegistry) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get agent config for approval workflow
	var agent *AgentConfig
	if registry != nil {
		agent = registry.GetAgentByID(agentID)
	}
	if agent == nil {
		// Create temporary config to get effective defaults
		agent = &AgentConfig{ID: agentID}
	}

	approval := agent.GetEffectiveApprovalConfig()
	if approval == nil {
		// No approval workflow configured - agent handles its own completion
		log.Printf("[TaskChain] Agent %s has no approval config, skipping GitHub workflow", agentID)
		return nil
	}

	// Store artifact path for later use (in merge comment, etc.)
	if result.ArtifactPath != "" {
		// Use stage-specific storage for backwards compatibility
		switch agentID {
		case "design-doc-creator":
			if err := tc.store.SetTaskDesignDocPath(ctx, taskID, result.ArtifactPath); err != nil {
				log.Printf("[TaskChain] Warning: Failed to store design doc path: %v", err)
			}
		case "sprint-planner":
			if err := tc.store.SetTaskSprintPlanPath(ctx, taskID, result.ArtifactPath); err != nil {
				log.Printf("[TaskChain] Warning: Failed to store sprint plan path: %v", err)
			}
		}
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// Handle case where no artifact found but files were changed
	if result.ArtifactPath == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No primary artifact found for task %s, but %d files discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			fileList := formatFileList(result.AllArtifacts, 20)
			infoComment := fmt.Sprintf(
				"## 📁 Files Changed\n\n"+
					"**Task:** %s\n"+
					"**Agent:** %s\n"+
					"**Duration:** %v\n\n"+
					"**%d files discovered** (no primary artifact for inline display):\n\n%s\n"+
					"Review changes in the dashboard diff viewer.",
				taskID, agentID, result.Duration, len(result.AllArtifacts), fileList,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, infoComment); err != nil {
				log.Printf("[TaskChain] Failed to post files comment: %v", err)
			}
		}
		// Continue with approval workflow
	} else if result.ArtifactPath == "" && len(result.AllArtifacts) == 0 {
		// No files changed at all - this is a failure
		log.Printf("[TaskChain] No artifacts found for task %s, posting failure message", taskID)
		if tc.poster != nil {
			failureComment := fmt.Sprintf(
				"## ❌ No Changes Detected\n\n"+
					"**Task:** %s\n"+
					"**Agent:** %s\n"+
					"**Duration:** %v\n\n"+
					"No files were created or modified. The agent may have:\n"+
					"- Encountered an error\n"+
					"- Not made any changes\n\n"+
					"Check the agent output for details.",
				taskID, agentID, result.Duration,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, failureComment); err != nil {
				log.Printf("[TaskChain] Failed to post failure comment: %v", err)
			}
		}
		return fmt.Errorf("no artifacts found for task %s", taskID)
	}

	// Read artifact content from worktree for display
	artifactContent := result.ArtifactContent
	if artifactContent == "" && result.ArtifactPath != "" && task.WorktreePath != "" {
		fullPath := filepath.Join(task.WorktreePath, result.ArtifactPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[TaskChain] Warning: Could not read artifact at %s: %v", fullPath, err)
		} else {
			artifactContent = string(content)
		}
	}

	// Render completion comment using template based on agent ID
	comment, err := tc.renderAgentCompleteComment(agentID, taskID, result, artifactContent, approval)
	if err != nil {
		return fmt.Errorf("failed to render completion comment: %w", err)
	}

	if tc.poster != nil {
		// Post the comment
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-approval label from config
		if approval.NeedsLabel != "" {
			if err := tc.poster.AddLabelInRepo(task.GithubRepo, task.GithubIssue, approval.NeedsLabel); err != nil {
				log.Printf("[TaskChain] Failed to add label %s: %v", approval.NeedsLabel, err)
			}
		}
	}

	return nil
}

// renderAgentCompleteComment selects and renders the appropriate comment template.
// Uses the agent's GithubCommentTemplate to select from predefined templates,
// or falls back to a generic template for unknown agents.
func (tc *TaskChain) renderAgentCompleteComment(agentID, taskID string, result *AgentResult, content string, approval *ApprovalConfig) (string, error) {
	// Build CommentData for template rendering
	data := &CommentData{
		TaskID:       taskID,
		Duration:     result.Duration,
		Cost:         result.Cost,
		TokensUsed:   result.TokensUsed,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
	}

	// Select template based on agent's comment template or agent ID
	templateName := approval.GithubCommentTemplate
	if templateName == "" {
		// Fall back to agent ID for template selection
		templateName = agentID
	}

	switch templateName {
	case "design_doc", "design-doc-creator":
		data.DesignDocPath = result.ArtifactPath
		data.DesignDocContent = content
		return RenderDesignDocComment(data)

	case "sprint_plan", "sprint-planner":
		data.SprintPlanPath = result.ArtifactPath
		data.SprintPlanContent = content
		return RenderSprintPlanComment(data)

	case "implementation", "sprint-executor":
		data.BranchName = result.BranchName
		data.WorktreePath = result.WorktreePath
		data.FilesCreated = result.FilesCreated
		data.FilesModified = result.FilesModified
		return RenderImplementCompleteComment(data)

	default:
		// Generic completion comment for unknown agents
		return tc.renderGenericCompleteComment(agentID, taskID, result, content, approval)
	}
}

// renderGenericCompleteComment renders a generic completion comment for agents
// without a predefined template. This supports custom agents added via config.
func (tc *TaskChain) renderGenericCompleteComment(agentID, taskID string, result *AgentResult, content string, approval *ApprovalConfig) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**✅ Agent Complete: %s**\n\n", agentID))

	// Summary table
	sb.WriteString("### Summary\n\n")
	sb.WriteString("| Field | Value |\n")
	sb.WriteString("|-------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Task ID** | `%s` |\n", taskID))
	sb.WriteString(fmt.Sprintf("| **Agent** | %s |\n", agentID))
	sb.WriteString(fmt.Sprintf("| **Duration** | %v |\n", result.Duration))
	if result.Cost > 0 {
		sb.WriteString(fmt.Sprintf("| **Cost** | $%.4f |\n", result.Cost))
	}
	if result.TokensUsed > 0 {
		sb.WriteString(fmt.Sprintf("| **Tokens** | %d (%d in / %d out) |\n",
			result.TokensUsed, result.InputTokens, result.OutputTokens))
	}

	// Artifact content if available
	if result.ArtifactPath != "" && content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(fmt.Sprintf("<details>\n<summary><strong>📄 Artifact: %s</strong> (click to expand)</summary>\n\n", result.ArtifactPath))
		sb.WriteString(content)
		sb.WriteString("\n\n</details>\n")
	}

	// Files list if no content
	if content == "" && len(result.AllArtifacts) > 0 {
		sb.WriteString("\n### Files Changed\n\n")
		for i, f := range result.AllArtifacts {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("- ... and %d more files\n", len(result.AllArtifacts)-20))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	// Next steps
	sb.WriteString("\n---\n\n")
	sb.WriteString("### Next Steps\n\n")
	sb.WriteString("1. **Review the output** above\n")
	if approval.ApprovedLabel != "" {
		sb.WriteString(fmt.Sprintf("2. **Add the `%s` label** to approve and continue\n", approval.ApprovedLabel))
	}
	sb.WriteString("3. **Add the `needs-revision` label** if changes are needed\n")

	return sb.String(), nil
}

// formatFileList formats a list of files for display, limiting to maxFiles.
func formatFileList(files []string, maxFiles int) string {
	var sb strings.Builder
	for i, f := range files {
		if i >= maxFiles {
			sb.WriteString(fmt.Sprintf("- ... and %d more files\n", len(files)-maxFiles))
			break
		}
		sb.WriteString(fmt.Sprintf("- `%s`\n", f))
	}
	return sb.String()
}

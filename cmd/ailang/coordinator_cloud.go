package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/pubsub"
)

// coordinatorExecuteJob is the entry point for Cloud Run Jobs.
// It reads task configuration from environment variables, executes the task
// using an AI executor (Claude or Gemini), and publishes completion to Pub/Sub.
//
// Required environment variables:
//
//	AILANG_TASK_ID      - Task ID to execute
//	AILANG_AGENT_ID     - Agent ID (e.g., "sprint-executor")
//	AILANG_WORKSPACE    - Workspace identifier
//	AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT - GCP project for Pub/Sub
//
// Optional:
//
//	AILANG_PROVIDER      - Executor provider (default: "claude")
//	AILANG_REPO_URL      - Git repository URL to clone
//	AILANG_BRANCH        - Branch to work on (default: "dev")
//	AILANG_DIRECTIVE     - Task directive/prompt
//	AILANG_TOPIC_PREFIX  - Topic prefix (default: "ailang")
func coordinatorExecuteJob(args []string) error {
	// Parse flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printExecuteJobHelp()
			return nil
		}
	}

	// Read required environment variables
	taskID := os.Getenv("AILANG_TASK_ID")
	if taskID == "" {
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}

	agentID := os.Getenv("AILANG_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}

	workspace := os.Getenv("AILANG_WORKSPACE")
	if workspace == "" {
		workspace = "default"
	}

	projectID := os.Getenv("AILANG_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		return fmt.Errorf("AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required")
	}

	// Optional configuration
	provider := os.Getenv("AILANG_PROVIDER")
	if provider == "" {
		provider = "claude"
	}

	repoURL := os.Getenv("AILANG_REPO_URL")
	branch := os.Getenv("AILANG_BRANCH")
	if branch == "" {
		branch = "dev"
	}
	directive := os.Getenv("AILANG_DIRECTIVE")

	prefix := os.Getenv("AILANG_TOPIC_PREFIX")
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}

	fmt.Printf("execute-job: starting task %s (agent=%s, workspace=%s)\n", taskID, agentID, workspace)

	// Initialize Pub/Sub client for completion publishing
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID, prefix)
	if err != nil {
		return fmt.Errorf("failed to create Pub/Sub client: %w", err)
	}
	defer client.Close()

	publisher := pubsub.NewPublisher(client)
	defer publisher.Stop()

	// Execute the task
	branchName, execErr := executeCloudTask(ctx, taskID, agentID, repoURL, branch, directive, provider)

	// Publish completion (success or failure)
	completion := pubsub.TaskCompletion{
		TaskID:  taskID,
		AgentID: agentID,
	}

	if execErr != nil {
		completion.Status = "failed"
		completion.ErrorMsg = execErr.Error()
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		completion.Status = "completed"
		completion.BranchName = branchName
		fmt.Printf("execute-job: task %s completed (branch=%s)\n", taskID, branchName)
	}

	if pubErr := publisher.PublishCompletion(ctx, completion, workspace); pubErr != nil {
		fmt.Printf("execute-job: WARNING failed to publish completion: %v\n", pubErr)
		// Still return the original error if task failed
		if execErr != nil {
			return execErr
		}
		return fmt.Errorf("task completed but failed to publish completion: %w", pubErr)
	}

	fmt.Printf("execute-job: completion published for task %s\n", taskID)
	return execErr
}

// executeCloudTask runs the AI executor in a cloned repository.
// Returns the branch name with changes, or error.
func executeCloudTask(ctx context.Context, taskID, agentID, repoURL, baseBranch, directive, provider string) (string, error) {
	workDir := fmt.Sprintf("/workspace/%s", taskID)

	// Step 1: Clone the repository (if URL provided)
	if repoURL != "" {
		fmt.Printf("execute-job: cloning %s (branch=%s)\n", repoURL, baseBranch)
		cloneCmd := exec.CommandContext(ctx, "git", "clone", "--branch", baseBranch, "--depth", "1", repoURL, workDir)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return "", fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		// No repo URL — use /workspace as working directory (mounted volume)
		workDir = "/workspace"
	}

	// Step 2: Create task branch
	branchName := fmt.Sprintf("coordinator/%s", taskID)
	fmt.Printf("execute-job: creating branch %s\n", branchName)

	checkoutCmd := exec.CommandContext(ctx, "git", "-C", workDir, "checkout", "-b", branchName)
	checkoutCmd.Stdout = os.Stdout
	checkoutCmd.Stderr = os.Stderr
	if err := checkoutCmd.Run(); err != nil {
		return "", fmt.Errorf("git checkout -b failed: %w", err)
	}

	// Step 3: Run the AI executor
	if directive == "" {
		directive = fmt.Sprintf("Execute task %s as agent %s", taskID, agentID)
	}

	fmt.Printf("execute-job: running %s executor\n", provider)
	execErr := runExecutor(ctx, workDir, provider, directive, taskID)
	if execErr != nil {
		return branchName, fmt.Errorf("executor failed: %w", execErr)
	}

	// Step 4: Check if there are changes to push
	statusCmd := exec.CommandContext(ctx, "git", "-C", workDir, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return branchName, fmt.Errorf("git status failed: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		fmt.Println("execute-job: no changes to commit")
		return branchName, nil
	}

	// Step 5: Stage, commit, and push
	addCmd := exec.CommandContext(ctx, "git", "-C", workDir, "add", "-A")
	if err := addCmd.Run(); err != nil {
		return branchName, fmt.Errorf("git add failed: %w", err)
	}

	commitMsg := fmt.Sprintf("Task %s: %s\n\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>",
		taskID, directive, agentID, time.Now().UTC().Format(time.RFC3339))

	commitCmd := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "-m", commitMsg)
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return branchName, fmt.Errorf("git commit failed: %w", err)
	}

	if repoURL != "" {
		pushCmd := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", branchName)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return branchName, fmt.Errorf("git push failed: %w", err)
		}
		fmt.Printf("execute-job: pushed branch %s\n", branchName)
	}

	return branchName, nil
}

// runExecutor invokes the AI executor CLI tool.
func runExecutor(ctx context.Context, workDir, provider, directive, taskID string) error {
	var cmd *exec.Cmd

	switch provider {
	case "claude":
		cmd = exec.CommandContext(ctx, "claude",
			"-p", directive,
			"--output-format", "json",
		)
	case "gemini":
		cmd = exec.CommandContext(ctx, "gemini",
			"-p", directive,
		)
	default:
		return fmt.Errorf("unsupported executor provider: %s", provider)
	}

	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass task context via environment
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AILANG_TASK_ID=%s", taskID),
	)

	return cmd.Run()
}

func printExecuteJobHelp() {
	fmt.Println("Usage: ailang coordinator execute-job")
	fmt.Println("")
	fmt.Println("Execute a task in a Cloud Run Job container. Configuration is read")
	fmt.Println("from environment variables set by Eventarc/Pub/Sub trigger.")
	fmt.Println("")
	fmt.Println("Required Environment Variables:")
	fmt.Println("  AILANG_TASK_ID          Task ID to execute")
	fmt.Println("  AILANG_AGENT_ID         Agent ID (e.g., sprint-executor)")
	fmt.Println("  AILANG_CLOUD_PROJECT    GCP project for Pub/Sub")
	fmt.Println("")
	fmt.Println("Optional Environment Variables:")
	fmt.Println("  AILANG_WORKSPACE        Workspace identifier (default: default)")
	fmt.Println("  AILANG_PROVIDER         Executor: claude or gemini (default: claude)")
	fmt.Println("  AILANG_REPO_URL         Git repo URL to clone")
	fmt.Println("  AILANG_BRANCH           Base branch (default: dev)")
	fmt.Println("  AILANG_DIRECTIVE        Task prompt/directive")
	fmt.Println("  AILANG_TOPIC_PREFIX     Topic prefix (default: ailang)")
}

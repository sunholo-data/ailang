package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sunholo/ailang/internal/executor"
	"github.com/sunholo/ailang/internal/pubsub"
)

// coordinatorExecuteJob is the entry point for Cloud Run Jobs.
// It reads task configuration from environment variables, executes the task
// using an AI executor (Claude or Gemini), and publishes completion to Pub/Sub.
//
// RELIABILITY: This function guarantees a completion message is always published.
// A defer guard with recover() catches panics and early exits. For failures before
// Pub/Sub is initialized, structured logs go to stderr for Cloud Logging.
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
//	AILANG_BRANCH        - Base branch to clone (default: "dev")
//	AILANG_PUSH_BRANCH   - Push directly to this branch (skip coordinator/ branch creation)
//	AILANG_DIRECTIVE     - Task directive/prompt
//	AILANG_TOPIC_PREFIX  - Topic prefix (default: "ailang")
//	AILANG_PLUGIN_REPO   - Git URL for shared skills plugin (cloned as --plugin-dir)
func coordinatorExecuteJob(args []string) error {
	// Parse flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printExecuteJobHelp()
			return nil
		}
	}

	// Read ALL environment variables upfront (before any early returns).
	taskID := os.Getenv("AILANG_TASK_ID")
	agentID := os.Getenv("AILANG_AGENT_ID")
	workspace := os.Getenv("AILANG_WORKSPACE")
	if workspace == "" {
		workspace = "default"
	}
	projectID := os.Getenv("AILANG_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
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

	// Initialize Pub/Sub client as early as possible so the defer guard can use it.
	// If Pub/Sub init itself fails, we fall back to stderr logging.
	ctx := context.Background()
	var publisher *pubsub.Publisher
	var completionSent atomic.Bool

	if projectID != "" {
		client, err := pubsub.NewClient(ctx, projectID, prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|error=pubsub_init: %v\n",
				taskID, agentID, err)
		} else {
			defer client.Close()
			publisher = pubsub.NewPublisher(client)
			defer publisher.Stop()
		}
	}

	// publishCompletion is a closure that publishes (or logs to stderr as fallback).
	publishCompletion := func(status, errMsg, branchName string) {
		if completionSent.Swap(true) {
			return // Already sent — prevent double-publish.
		}
		completion := pubsub.TaskCompletion{
			TaskID:     taskID,
			AgentID:    agentID,
			Status:     status,
			ErrorMsg:   errMsg,
			BranchName: branchName,
		}
		if publisher != nil {
			pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if pubErr := publisher.PublishCompletion(pubCtx, completion, workspace); pubErr != nil {
				fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|status=%s|error=publish: %v\n",
					taskID, agentID, status, pubErr)
			} else {
				fmt.Printf("execute-job: completion published for task %s (status=%s)\n", taskID, status)
			}
		} else {
			// No Pub/Sub available — structured stderr log for Cloud Logging.
			fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|status=%s|error=%s\n",
				taskID, agentID, status, errMsg)
		}
	}

	// Defer guard: catches panics and any exit path that forgot to publish.
	defer func() {
		if r := recover(); r != nil {
			publishCompletion("failed", fmt.Sprintf("panic: %v", r), "")
		} else if !completionSent.Load() {
			// Should not happen — means we returned without publishing.
			publishCompletion("failed", "unknown: exited without publishing completion", "")
		}
	}()

	// Validate required env vars (after Pub/Sub init so failures are reported).
	if taskID == "" {
		publishCompletion("failed", "AILANG_TASK_ID environment variable is required", "")
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}
	if agentID == "" {
		publishCompletion("failed", "AILANG_AGENT_ID environment variable is required", "")
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}
	if projectID == "" {
		publishCompletion("failed", "AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required", "")
		return fmt.Errorf("AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required")
	}

	// Read plugin repo for shared skills (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	pluginRepo := os.Getenv("AILANG_PLUGIN_REPO")

	fmt.Printf("execute-job: starting task %s (agent=%s, workspace=%s)\n", taskID, agentID, workspace)

	// Execute the task
	branchName, execErr := executeCloudTask(ctx, taskID, agentID, repoURL, branch, directive, provider, pluginRepo)

	// Publish completion (success or failure)
	if execErr != nil {
		publishCompletion("failed", execErr.Error(), branchName)
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		publishCompletion("completed", "", branchName)
		fmt.Printf("execute-job: task %s completed (branch=%s)\n", taskID, branchName)
	}

	return execErr
}

// executeCloudTask runs the AI executor in a cloned repository.
// Returns the branch name with changes, or error.
//
// When AILANG_PUSH_BRANCH is set, the agent works directly on the cloned branch
// and pushes to that branch (no coordinator/{taskID} branch creation). This is
// used for skip_approval agents like website-builder that push directly to main.
func executeCloudTask(ctx context.Context, taskID, agentID, repoURL, baseBranch, directive, provider, pluginRepo string) (string, error) {
	workDir := fmt.Sprintf("/workspace/%s", taskID)
	pushBranch := os.Getenv("AILANG_PUSH_BRANCH")

	// Step 0: Clone shared skills plugin if configured (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	pluginDir := ""
	if pluginRepo != "" {
		pluginDir = filepath.Join("/plugins", taskID, "ailang_bootstrap")
		fmt.Printf("execute-job: cloning plugin repo %s\n", pluginRepo)
		pluginCloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", pluginRepo, pluginDir)
		pluginCloneCmd.Stdout = os.Stdout
		pluginCloneCmd.Stderr = os.Stderr
		if err := pluginCloneCmd.Run(); err != nil {
			// Best effort — agent can still work without plugin skills
			fmt.Fprintf(os.Stderr, "warning: plugin clone failed (continuing without plugin skills): %v\n", err)
			pluginDir = ""
		}
	}

	// Step 1: Clone the repository (required in cloud mode)
	if repoURL == "" {
		return "", fmt.Errorf("AILANG_REPO_URL is required: set workspace to GitHub org/repo (e.g., sunholo-data/ailang) in agent config")
	}
	fmt.Printf("execute-job: cloning %s (branch=%s)\n", repoURL, baseBranch)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--branch", baseBranch, "--depth", "1", repoURL, workDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}

	// Step 1.5: Inject AGENTS.md from plugin if repo doesn't have one (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	// This gives the agent cross-platform instructions without requiring every repo to include AGENTS.md.
	if pluginDir != "" {
		injectAgentsMD(pluginDir, workDir)
	}

	// Step 2: Create task branch (skip if direct push mode)
	var branchName string
	if pushBranch != "" {
		// Direct push mode: work on the cloned branch, push to pushBranch
		branchName = pushBranch
		fmt.Printf("execute-job: direct push mode — working on %s (no coordinator branch)\n", pushBranch)
	} else {
		// Standard mode: create coordinator/{taskID} branch
		branchName = fmt.Sprintf("coordinator/%s", taskID)
		fmt.Printf("execute-job: creating branch %s\n", branchName)

		checkoutCmd := exec.CommandContext(ctx, "git", "-C", workDir, "checkout", "-b", branchName)
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return "", fmt.Errorf("git checkout -b failed: %w", err)
		}
	}

	// Step 3: Run the AI executor
	if directive == "" {
		directive = fmt.Sprintf("Execute task %s as agent %s", taskID, agentID)
	}

	fmt.Printf("execute-job: running %s executor\n", provider)
	execErr := runExecutor(ctx, workDir, provider, directive, taskID, pluginDir)
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
// Uses executor.BuildEnvironment to match the local executor's environment setup,
// ensuring AILANG_STDLIB_PATH, trace context, correlation IDs, and telemetry vars
// are all set consistently between local and cloud execution.
func runExecutor(ctx context.Context, workDir, provider, directive, taskID, pluginDir string) error {
	var cmd *exec.Cmd

	// Build an executor.Task so BuildEnvironment has the same context as local executors
	task := &executor.Task{
		ID:        taskID,
		Directive: directive,
		Workspace: workDir,
	}
	if pluginDir != "" {
		task.PluginDirs = []string{pluginDir}
	}

	switch provider {
	case "claude":
		args := []string{
			"-p", directive,
			"--output-format", "json",
			// Cloud containers always need --dangerously-skip-permissions because
			// --permission-mode bypassPermissions does NOT bypass settings.json
			// permission.allow lists. Without this, Write tool calls get denied.
			"--dangerously-skip-permissions",
		}
		// Pass plugin directory if available (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
		if pluginDir != "" {
			args = append(args, "--plugin-dir", pluginDir)
		}
		cmd = exec.CommandContext(ctx, "claude", args...)
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

	// Use BuildEnvironment for consistent env between local and cloud executors.
	// This sets AILANG_STDLIB_PATH, PWD, TRACEPARENT, correlation IDs, OTEL vars, etc.
	cmd.Env = executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:                  task,
		SessionID:             taskID,
		Context:               ctx,
		EnableClaudeTelemetry: provider == "claude",
		EnableGeminiTelemetry: provider == "gemini",
	})

	return cmd.Run()
}

// injectAgentsMD copies AGENTS.md from the plugin directory into the workspace
// if the workspace doesn't already have one. This gives agents cross-platform
// instructions without requiring every repo to include AGENTS.md.
func injectAgentsMD(pluginDir, workDir string) {
	src := filepath.Join(pluginDir, "AGENTS.md")
	dst := filepath.Join(workDir, "AGENTS.md")

	// Don't overwrite if repo already has AGENTS.md
	if _, err := os.Stat(dst); err == nil {
		fmt.Printf("execute-job: AGENTS.md already exists in repo, skipping injection\n")
		return
	}

	// Check if plugin has AGENTS.md
	srcData, err := os.ReadFile(src)
	if err != nil {
		// Plugin doesn't have AGENTS.md — that's OK
		return
	}

	if err := os.WriteFile(dst, srcData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to inject AGENTS.md: %v\n", err)
		return
	}
	fmt.Printf("execute-job: injected AGENTS.md from plugin into workspace\n")
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
	fmt.Println("  AILANG_PUSH_BRANCH      Push directly to this branch (skip coordinator/ branch)")
	fmt.Println("  AILANG_DIRECTIVE        Task prompt/directive")
	fmt.Println("  AILANG_TOPIC_PREFIX     Topic prefix (default: ailang)")
	fmt.Println("  AILANG_PLUGIN_REPO      Git URL for shared skills plugin (--plugin-dir)")
}

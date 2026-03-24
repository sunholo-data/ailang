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

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/executor"
	// Import to trigger init() registration — same as local coordinator (provider_executor.go)
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
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
// The executor uses the same infrastructure as the local coordinator (via
// executor.GlobalFactory) for full parity: stream-JSON parsing, token extraction,
// OTEL spans, session tracking, idle/hard timeouts, and cost calculation.
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
//	AILANG_MAX_COST_USD  - Per-task cost budget in USD (0 = unlimited) from budget config
//	AILANG_MODEL         - AI model override (e.g., "sonnet", "opus") from agent config
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
	// The optional execResult carries metrics from the executor for parity with local.
	// changedFiles lists files created/modified by the agent (discovered via git diff).
	publishCompletion := func(status, errMsg, branchName string, execResult *executor.Result, changedFiles []string) {
		if completionSent.Swap(true) {
			return // Already sent — prevent double-publish.
		}
		completion := pubsub.TaskCompletion{
			TaskID:       taskID,
			AgentID:      agentID,
			Status:       status,
			ErrorMsg:     errMsg,
			BranchName:   branchName,
			ChangedFiles: changedFiles,
		}
		// Populate executor metrics when available (same data as local coordinator)
		if execResult != nil {
			completion.SessionID = execResult.SessionID
			completion.NumTurns = execResult.NumTurns
			completion.ToolCallCount = execResult.ToolCallCount
			completion.InputTokens = execResult.InputTokens
			completion.OutputTokens = execResult.OutputTokens
			completion.CostUSD = execResult.CostUSD
			completion.DurationMS = execResult.DurationMS
		}
		if publisher != nil {
			pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if pubErr := publisher.PublishCompletion(pubCtx, completion, workspace); pubErr != nil {
				fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|status=%s|error=publish: %v\n",
					taskID, agentID, status, pubErr)
			} else {
				fmt.Printf("execute-job: completion published for task %s (status=%s, turns=%d, tokens=%d+%d, cost=$%.4f)\n",
					taskID, status, completion.NumTurns, completion.InputTokens, completion.OutputTokens, completion.CostUSD)
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
			publishCompletion("failed", fmt.Sprintf("panic: %v", r), "", nil, nil)
		} else if !completionSent.Load() {
			// Should not happen — means we returned without publishing.
			publishCompletion("failed", "unknown: exited without publishing completion", "", nil, nil)
		}
	}()

	// Validate required env vars (after Pub/Sub init so failures are reported).
	if taskID == "" {
		publishCompletion("failed", "AILANG_TASK_ID environment variable is required", "", nil, nil)
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}
	if agentID == "" {
		publishCompletion("failed", "AILANG_AGENT_ID environment variable is required", "", nil, nil)
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}
	if projectID == "" {
		publishCompletion("failed", "AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required", "", nil, nil)
		return fmt.Errorf("AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required")
	}

	// Read plugin repo for shared skills (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	pluginRepo := os.Getenv("AILANG_PLUGIN_REPO")

	// Read model override from agent config (passed via AILANG_MODEL env var).
	// Without this, the executor defaults to "haiku" which is too weak for coding tasks.
	model := os.Getenv("AILANG_MODEL")

	// Read timeout from agent config (passed via AILANG_TIMEOUT env var, M-CLOUD-OAUTH).
	// Without this, the executor defaults to 5m which is too short for complex tasks.
	timeoutStr := os.Getenv("AILANG_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "30m" // Reasonable default for cloud tasks
	}

	fmt.Printf("execute-job: starting task %s (agent=%s, workspace=%s, model=%s, timeout=%s)\n", taskID, agentID, workspace, model, timeoutStr)

	// Execute the task
	branchName, execResult, changedFiles, execErr := executeCloudTask(ctx, taskID, agentID, repoURL, branch, directive, provider, pluginRepo, model, timeoutStr)

	// Publish completion with executor metrics (success or failure)
	if execErr != nil {
		publishCompletion("failed", execErr.Error(), branchName, execResult, nil)
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		publishCompletion("completed", "", branchName, execResult, changedFiles)
		fmt.Printf("execute-job: task %s completed (branch=%s, files=%d)\n", taskID, branchName, len(changedFiles))
	}

	return execErr
}

// executeCloudTask runs the AI executor in a cloned repository.
// Returns the branch name with changes, executor result with metrics, or error.
//
// When AILANG_PUSH_BRANCH is set, the agent works directly on the cloned branch
// and pushes to that branch (no coordinator/{taskID} branch creation). This is
// used for skip_approval agents like website-builder that push directly to main.
func executeCloudTask(ctx context.Context, taskID, agentID, repoURL, baseBranch, directive, provider, pluginRepo, model, timeoutStr string) (string, *executor.Result, []string, error) {
	workDir := fmt.Sprintf("/workspace/%s", taskID)
	pushBranch := os.Getenv("AILANG_PUSH_BRANCH")

	// When push branch is set, clone that branch instead of baseBranch.
	// This handles repos where the default branch differs from "dev"
	// (e.g., sunholo-websites uses "main"). The push branch is the branch
	// that actually exists in the remote and where we want to commit.
	if pushBranch != "" && baseBranch != pushBranch {
		fmt.Printf("execute-job: overriding clone branch %s → %s (push branch)\n", baseBranch, pushBranch)
		baseBranch = pushBranch
	}

	// Step -1: Configure git credentials from GITHUB_TOKEN.
	// Cloud containers don't have a credential helper — git can't authenticate HTTPS
	// requests without this. GITHUB_TOKEN is provided via Secret Manager.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		credHelper := fmt.Sprintf("!f() { echo username=x-access-token; echo \"password=%s\"; }; f", token)
		credCmd := exec.CommandContext(ctx, "git", "config", "--global", "credential.helper", credHelper)
		if err := credCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to configure git credentials: %v\n", err)
		}
	}

	// Step 0: Resolve shared skills plugin directory (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	pluginDir := ""
	if pluginRepo != "" {
		pluginDir = filepath.Join("/plugins", taskID, "ailang_bootstrap")
		fmt.Printf("execute-job: cloning plugin repo %s\n", pluginRepo)
		pluginCloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", pluginRepo, pluginDir)
		pluginCloneCmd.Stdout = os.Stdout
		pluginCloneCmd.Stderr = os.Stderr
		if err := pluginCloneCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: plugin clone failed: %v\n", err)
			pluginDir = ""
		}
	}
	// Fall back to pre-baked plugin from Docker image build
	if pluginDir == "" {
		if info, err := os.Stat("/plugins/ailang_bootstrap"); err == nil && info.IsDir() {
			pluginDir = "/plugins/ailang_bootstrap"
			fmt.Printf("execute-job: using pre-baked plugin at %s\n", pluginDir)
		}
	}

	// Step 1: Clone the repository (required in cloud mode)
	if repoURL == "" {
		return "", nil, nil, fmt.Errorf("AILANG_REPO_URL is required: set workspace to GitHub org/repo (e.g., sunholo-data/ailang) in agent config")
	}
	fmt.Printf("execute-job: cloning %s (branch=%s)\n", repoURL, baseBranch)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--branch", baseBranch, "--depth", "1", repoURL, workDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return "", nil, nil, fmt.Errorf("git clone failed: %w", err)
	}

	// M-HARNESS-COMMIT-CONTRACT: Capture clone point for artifact discovery.
	clonePointCmd := exec.CommandContext(ctx, "git", "-C", workDir, "rev-parse", "HEAD")
	clonePointOutput, _ := clonePointCmd.Output()
	clonePoint := strings.TrimSpace(string(clonePointOutput))

	// Step 1.5: Inject AGENTS.md from plugin if repo doesn't have one
	if pluginDir != "" {
		injectAgentsMD(pluginDir, workDir)
	}

	// Step 2: Create task branch (skip if direct push mode)
	var branchName string
	if pushBranch != "" {
		branchName = pushBranch
		fmt.Printf("execute-job: direct push mode — working on %s (no coordinator branch)\n", pushBranch)
	} else {
		branchName = fmt.Sprintf("coordinator/%s", taskID)
		fmt.Printf("execute-job: creating branch %s\n", branchName)

		checkoutCmd := exec.CommandContext(ctx, "git", "-C", workDir, "checkout", "-b", branchName)
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
		if err := checkoutCmd.Run(); err != nil {
			return "", nil, nil, fmt.Errorf("git checkout -b failed: %w", err)
		}
	}

	// Step 3: Run the AI executor
	if directive == "" {
		directive = fmt.Sprintf("Execute task %s as agent %s", taskID, agentID)
	}

	// M-GIT-GUARDRAILS: Default to guardrails if not set per-agent or via Terraform.
	if os.Getenv("AILANG_GIT_MODE") == "" {
		os.Setenv("AILANG_GIT_MODE", "guardrails")
	}

	// M-PKG-AUTONOMOUS-UPDATES: Scope executor to monorepo subdirectory if set.
	execWorkDir := workDir
	if subdir := os.Getenv("AILANG_SUBDIRECTORY"); subdir != "" {
		execWorkDir = filepath.Join(workDir, subdir)
		fmt.Printf("execute-job: scoped to subdirectory %s (within %s)\n", subdir, workDir)
	}

	fmt.Printf("execute-job: running %s executor (unified path)\n", provider)
	execResult, execErr := runExecutor(ctx, execWorkDir, provider, directive, taskID, pluginDir, model, timeoutStr)
	if execErr != nil {
		return branchName, execResult, nil, fmt.Errorf("executor failed: %w", execErr)
	}

	// Log executor metrics
	if execResult != nil {
		fmt.Printf("execute-job: executor completed (turns=%d, tools=%d, tokens=%d+%d, cost=$%.4f, session=%s)\n",
			execResult.NumTurns, execResult.ToolCallCount,
			execResult.InputTokens, execResult.OutputTokens,
			execResult.CostUSD, execResult.SessionID)
	}

	// Step 4: Check if there are uncommitted changes to stage+commit
	statusCmd := exec.CommandContext(ctx, "git", "-C", workDir, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return branchName, execResult, nil, fmt.Errorf("git status failed: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		// Step 5a: Stage, commit uncommitted changes
		addCmd := exec.CommandContext(ctx, "git", "-C", workDir, "add", "-A")
		if err := addCmd.Run(); err != nil {
			return branchName, execResult, nil, fmt.Errorf("git add failed: %w", err)
		}

		// M-HARNESS-COMMIT-CONTRACT: Use structured commit message when site metadata available.
		var commitMsg string
		siteSlug := os.Getenv("AILANG_SITE_SLUG")
		briefID := os.Getenv("AILANG_BRIEF_ID")
		if siteSlug != "" {
			subject := fmt.Sprintf("Build: %s", siteSlug)
			if briefID != "" {
				subject += fmt.Sprintf(" [briefId=%s]", briefID)
			}
			commitMsg = fmt.Sprintf("%s\n\nTask: %s\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>",
				subject, taskID, agentID, time.Now().UTC().Format(time.RFC3339))
		} else {
			commitMsg = fmt.Sprintf("Task %s: %s\n\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>",
				taskID, directive, agentID, time.Now().UTC().Format(time.RFC3339))
		}

		commitCmd := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "-m", commitMsg)
		commitCmd.Stdout = os.Stdout
		commitCmd.Stderr = os.Stderr
		if err := commitCmd.Run(); err != nil {
			return branchName, execResult, nil, fmt.Errorf("git commit failed: %w", err)
		}
	} else {
		fmt.Println("execute-job: no uncommitted changes (agent may have committed directly)")
	}

	// Step 5b: Check for ANY unpushed commits
	logCmd := exec.CommandContext(ctx, "git", "-C", workDir, "log", fmt.Sprintf("origin/%s..HEAD", branchName), "--oneline")
	logOutput, err := logCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: git log origin/%s..HEAD failed (shallow clone?): %v\n", branchName, err)
		revCmd := exec.CommandContext(ctx, "git", "-C", workDir, "rev-list", "--count", fmt.Sprintf("origin/%s..HEAD", branchName))
		revOutput, revErr := revCmd.Output()
		if revErr == nil && strings.TrimSpace(string(revOutput)) != "0" {
			logOutput = revOutput
		}
	}

	if len(strings.TrimSpace(string(logOutput))) == 0 {
		fmt.Println("execute-job: no commits to push")
		changedFiles := discoverChangedFilesFromCommit(workDir, clonePoint)
		return branchName, execResult, changedFiles, nil
	}

	fmt.Printf("execute-job: unpushed commits:\n%s", string(logOutput))

	// Step 5c: Push all commits
	if repoURL != "" {
		pushCmd := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", branchName)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return branchName, execResult, nil, fmt.Errorf("git push failed: %w", err)
		}
		fmt.Printf("execute-job: pushed branch %s\n", branchName)
	}

	// Step 6: Discover changed files for the completion message.
	changedFiles := discoverChangedFilesFromCommit(workDir, clonePoint)
	return branchName, execResult, changedFiles, nil
}

// discoverChangedFilesFromCommit uses ArtifactDiscovery to find files created/modified by the agent.
func discoverChangedFilesFromCommit(workDir, clonePoint string) []string {
	ad := coordinator.NewArtifactDiscovery(workDir, nil)
	if clonePoint != "" {
		ad.WithBaseCommit(clonePoint)
	}
	files, err := ad.DiscoverChangedFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: warning: artifact discovery failed: %v\n", err)
		return nil
	}
	if len(files) > 0 {
		fmt.Printf("execute-job: discovered %d changed files\n", len(files))
	}
	return files
}

// injectAgentsMD copies AGENTS.md from the plugin directory into the workspace
// if the workspace doesn't already have one.
func injectAgentsMD(pluginDir, workDir string) {
	src := filepath.Join(pluginDir, "AGENTS.md")
	dst := filepath.Join(workDir, "AGENTS.md")

	if _, err := os.Stat(dst); err == nil {
		fmt.Printf("execute-job: AGENTS.md already exists in repo, skipping injection\n")
		return
	}

	srcData, err := os.ReadFile(src)
	if err != nil {
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
	fmt.Println("  AILANG_MODEL            AI model override (e.g., sonnet, opus)")
	fmt.Println("  AILANG_MAX_COST_USD     Per-task cost budget in USD (0 = unlimited)")
}

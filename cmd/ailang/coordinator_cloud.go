package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	// Import to trigger init() registration — same as local coordinator (provider_executor.go)
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
	_ "github.com/sunholo-data/ailang/internal/executor/gemini"
	"github.com/sunholo-data/ailang/internal/pubsub"
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
	// artifactPath is the GCS path prefix where raw artifacts were uploaded (may be empty).
	publishCompletion := func(status, errMsg, branchName string, execResult *executor.Result, changedFiles []string, artifactPath string) {
		if completionSent.Swap(true) {
			return // Already sent — prevent double-publish.
		}
		completion := pubsub.TaskCompletion{
			TaskID:          taskID,
			AgentID:         agentID,
			Status:          status,
			ErrorMsg:        errMsg,
			BranchName:      branchName,
			ChangedFiles:    changedFiles,
			ArtifactGCSPath: artifactPath,
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
			completion.CacheReadTokens = execResult.CacheReadInputTokens
			completion.CacheCreationTokens = execResult.CacheCreationInputTokens
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
			publishCompletion("failed", fmt.Sprintf("panic: %v", r), "", nil, nil, "")
		} else if !completionSent.Load() {
			// Should not happen — means we returned without publishing.
			publishCompletion("failed", "unknown: exited without publishing completion", "", nil, nil, "")
		}
	}()

	// Validate required env vars (after Pub/Sub init so failures are reported).
	if taskID == "" {
		publishCompletion("failed", "AILANG_TASK_ID environment variable is required", "", nil, nil, "")
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}
	if agentID == "" {
		publishCompletion("failed", "AILANG_AGENT_ID environment variable is required", "", nil, nil, "")
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}
	if projectID == "" {
		publishCompletion("failed", "AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required", "", nil, nil, "")
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

	// Write artifact files to the GCS-mounted directory (/artifacts/tasks/{taskID}/).
	// The artifact bucket is mounted read-write at /artifacts via Cloud Run volume mount.
	// Files written here land directly in GCS — no upload client needed.
	artifactPath := writeTaskArtifacts(taskID, execResult)

	// Publish completion with executor metrics (success or failure)
	if execErr != nil {
		publishCompletion("failed", execErr.Error(), branchName, execResult, nil, artifactPath)
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		publishCompletion("completed", "", branchName, execResult, changedFiles, artifactPath)
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

	// Direct Claude Code session storage into the GCS-mounted artifact directory.
	// CLAUDE_CONFIG_DIR overrides ~/.claude — session JSONL is written directly to GCS
	// without any upload step. Path: /artifacts/tasks/{taskID}/claude/projects/{path}/{sid}.jsonl
	claudeConfigDir := filepath.Join("/artifacts", "tasks", taskID, "claude")
	if err := os.MkdirAll(claudeConfigDir, 0755); err == nil {
		os.Setenv("CLAUDE_CONFIG_DIR", claudeConfigDir)
		fmt.Printf("execute-job: CLAUDE_CONFIG_DIR=%s (session JSONL → GCS)\n", claudeConfigDir)
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

	// Step 5b: Check for unpushed commits.
	// If origin/{branchName} doesn't exist yet (new coordinator branch), git log
	// exits 128 — treat that as "entire branch is unpushed" and proceed to push.
	logCmd := exec.CommandContext(ctx, "git", "-C", workDir, "log", fmt.Sprintf("origin/%s..HEAD", branchName), "--oneline")
	logOutput, err := logCmd.Output()
	newBranch := false
	if err != nil {
		// Check whether the remote ref simply doesn't exist yet.
		lsRemote := exec.CommandContext(ctx, "git", "-C", workDir, "ls-remote", "--exit-code", "origin", branchName)
		if lsRemote.Run() != nil {
			// Remote ref absent — whole branch needs pushing.
			newBranch = true
			fmt.Printf("execute-job: branch %s not on remote yet — will push\n", branchName)
		} else {
			fmt.Fprintf(os.Stderr, "execute-job: git log origin/%s..HEAD failed: %v\n", branchName, err)
		}
	}

	if !newBranch && len(strings.TrimSpace(string(logOutput))) == 0 {
		fmt.Println("execute-job: no commits to push")
		changedFiles := discoverChangedFilesFromCommit(workDir, clonePoint)
		return branchName, execResult, changedFiles, nil
	}

	fmt.Printf("execute-job: unpushed commits:\n%s", string(logOutput))

	// Step 5c: Push all commits.
	// Shallow clones can't push new branches — unshallow first so the remote
	// has full history context for the new branch ref.
	if repoURL != "" {
		unshallowCmd := exec.CommandContext(ctx, "git", "-C", workDir, "fetch", "--unshallow")
		unshallowCmd.Stdout = os.Stdout
		unshallowCmd.Stderr = os.Stderr
		if err := unshallowCmd.Run(); err != nil {
			// Already a full clone or network issue — log and continue
			fmt.Fprintf(os.Stderr, "execute-job: fetch --unshallow skipped: %v\n", err)
		}
		pushCmd := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", branchName)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return branchName, execResult, nil, fmt.Errorf("git push failed: %w", err)
		}
		fmt.Printf("execute-job: pushed branch %s\n", branchName)

		// Step 5d: Open a deterministic PR (M-PKG-AUTONOMOUS-CASCADE-SAFE follow-up).
		// Always-PR is the design — no autonomous merge for v1. Doing this in the
		// wrapper (vs the agent) means the agent doesn't need to know `gh` syntax,
		// and we get a consistent PR title/body across every agent run.
		// Best-effort: failures don't fail the task (branch is already pushed).
		openCascadePullRequest(ctx, workDir, branchName, taskID, agentID, baseBranch)
	}

	// Step 6: Discover changed files for the completion message.
	changedFiles := discoverChangedFilesFromCommit(workDir, clonePoint)
	return branchName, execResult, changedFiles, nil
}

// openCascadePullRequest opens a PR via `gh pr create` after the wrapper has
// pushed the agent's branch. Best-effort — if gh isn't available or the PR
// already exists, we log and continue (the task itself succeeded; the PR is
// the surfacing mechanism for human review).
//
// Cascade context (root_package, root_version) is surfaced via env vars set
// by the dispatcher; if absent, we open a generic agent-task PR instead.
func openCascadePullRequest(ctx context.Context, workDir, branchName, taskID, agentID, baseBranch string) {
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Detect cascade vs generic agent task from env (set by dispatcher when
	// the inbound message had source=cascade + root_package attribute).
	rootPackage := os.Getenv("AILANG_CASCADE_ROOT_PACKAGE")
	title := fmt.Sprintf("[agent] %s: %s", agentID, taskID)
	bodyLines := []string{
		fmt.Sprintf("Autonomous task `%s` completed and pushed by agent `%s`.", taskID, agentID),
		"",
		"This PR was opened deterministically by the AILANG coordinator wrapper.",
		"",
		fmt.Sprintf("View execution chain: `ailang chains view %s`", taskID),
	}
	labels := []string{"agent-task"}

	if rootPackage != "" {
		title = fmt.Sprintf("[cascade] bump %s (%s)", rootPackage, taskID)
		bodyLines = []string{
			fmt.Sprintf("Cascade-driven dependency update from `%s`.", rootPackage),
			"",
			fmt.Sprintf("Triggered by autonomous task `%s` (agent `%s`).", taskID, agentID),
			"",
			"This PR was opened deterministically by the AILANG coordinator wrapper.",
			"Always-PR is the v1 design — no autonomous merge.",
			"",
			fmt.Sprintf("View execution chain: `ailang chains view %s`", taskID),
		}
		labels = []string{"cascade", "agent-task"}
	}

	args := []string{"pr", "create",
		"--title", title,
		"--body", strings.Join(bodyLines, "\n"),
		"--base", baseBranch,
		"--head", branchName,
	}
	for _, l := range labels {
		args = append(args, "--label", l)
	}

	prCmd := exec.CommandContext(ctx, "gh", args...)
	prCmd.Dir = workDir
	out, err := prCmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		// Common case: gh CLI missing or PR already exists for this branch.
		// Don't fail the task — the branch push already succeeded.
		fmt.Fprintf(os.Stderr, "execute-job: gh pr create skipped: %v (output: %s)\n", err, outStr)
		return
	}
	fmt.Printf("execute-job: opened PR: %s\n", outStr)
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

// writeTaskArtifacts writes execution artifacts to the GCS-mounted artifact directory.
//
// The artifact bucket is mounted read-write at /artifacts via Cloud Run volume mount
// (configured in Terraform). Files written here go directly to GCS — no upload client needed.
//
// Writes:
//   - /artifacts/tasks/{taskID}/transcript.txt — plain-text turn summary
//   - /artifacts/tasks/{taskID}/metrics.json   — extended metrics (cache tokens, files)
//   - /artifacts/tasks/{taskID}/session.jsonl  — Claude Code JSONL history (copied from CLAUDE_CONFIG_DIR)
//
// The JSONL copy is necessary because gcsfuse uses "legacy staged writes" for files
// that Claude appends to incrementally — the staged write may not flush before the
// container exits. Explicitly re-writing the file via os.WriteFile guarantees flush.
//
// Returns the GCS path prefix ("tasks/{taskID}") for linking from Firestore.
// Failures are non-fatal and logged to stderr.
func writeTaskArtifacts(taskID string, result *executor.Result) string {
	artifactDir := filepath.Join("/artifacts", "tasks", taskID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: warning: could not create artifact dir %s: %v\n", artifactDir, err)
		return ""
	}

	prefix := fmt.Sprintf("tasks/%s", taskID)

	if result == nil {
		return prefix
	}

	// 1. Plain-text transcript
	if result.Transcript != "" {
		p := filepath.Join(artifactDir, "transcript.txt")
		if err := os.WriteFile(p, []byte(result.Transcript), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write transcript.txt: %v\n", err)
		}
	}

	// 2. Extended metrics (cache tokens, files — not in TaskCompletion message)
	metrics := map[string]any{
		"task_id":               taskID,
		"session_id":            result.SessionID,
		"num_turns":             result.NumTurns,
		"tool_call_count":       result.ToolCallCount,
		"input_tokens":          result.InputTokens,
		"output_tokens":         result.OutputTokens,
		"cache_read_tokens":     result.CacheReadInputTokens,
		"cache_creation_tokens": result.CacheCreationInputTokens,
		"cost_usd":              result.CostUSD,
		"duration_ms":           result.DurationMS,
		"files_created":         result.FilesCreated,
		"files_modified":        result.FilesModified,
		"written_at":            time.Now().UTC().Format(time.RFC3339),
	}
	if metricsJSON, err := json.Marshal(metrics); err == nil {
		p := filepath.Join(artifactDir, "metrics.json")
		if err := os.WriteFile(p, metricsJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write metrics.json: %v\n", err)
		}
	}

	// 3. Copy Claude Code session JSONL to session.jsonl.
	// Claude writes the JSONL to CLAUDE_CONFIG_DIR/projects/{path}/{sessionID}.jsonl via gcsfuse.
	// gcsfuse uses legacy staged writes for incrementally-appended files, which may not flush
	// before the container exits. Re-writing via os.WriteFile guarantees the data reaches GCS.
	if result.SessionID != "" {
		claudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
		if claudeConfigDir == "" {
			claudeConfigDir = filepath.Join("/artifacts", "tasks", taskID, "claude")
		}
		jsonlPath := findSessionJSONL(claudeConfigDir, result.SessionID)
		if jsonlPath != "" {
			if data, err := os.ReadFile(jsonlPath); err == nil && len(data) > 0 {
				dst := filepath.Join(artifactDir, "session.jsonl")
				if err := os.WriteFile(dst, data, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write session.jsonl: %v\n", err)
				} else {
					fmt.Printf("execute-job: session.jsonl written (%d bytes)\n", len(data))
				}
			}
		}
	}

	fmt.Printf("execute-job: artifacts written to /artifacts/%s\n", prefix)
	return prefix
}

// findSessionJSONL searches claudeConfigDir for a JSONL file matching sessionID.
// Claude Code writes to: {claudeConfigDir}/projects/{encoded-path}/{sessionID}.jsonl
func findSessionJSONL(claudeConfigDir, sessionID string) string {
	target := sessionID + ".jsonl"
	var found string
	filepath.WalkDir(claudeConfigDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
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

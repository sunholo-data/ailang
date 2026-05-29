package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	// Import to trigger init() registration — same as local coordinator (provider_executor.go)
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
	_ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// coordinatorExecuteJob is the entry point for Cloud Run Jobs.
// It reads task configuration from environment variables, executes the task
// using an AI executor (Claude — Gemini CLI was retired in v0.22.0
// per M-MANAGED-AGENTS), and publishes completion to Pub/Sub.
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

	// Step 1.5: Inject AGENTS.md from plugin if repo doesn't have one.
	// M-PKG-CASCADE-DETERMINISTIC-FIRST: skip injection for cascade tasks —
	// AGENTS.md is generic agent guidance and just clutters the cascade PR
	// (which has a single deterministic toml bump as its diff). Cascade
	// tasks are detected via AILANG_CASCADE_ROOT_PACKAGE.
	if pluginDir != "" && os.Getenv("AILANG_CASCADE_ROOT_PACKAGE") == "" {
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

	// M-PKG-AUTONOMOUS-UPDATES: Scope executor to monorepo subdirectory if set.
	execWorkDir := workDir
	if subdir := os.Getenv("AILANG_SUBDIRECTORY"); subdir != "" {
		execWorkDir = filepath.Join(workDir, subdir)
		fmt.Printf("execute-job: scoped to subdirectory %s (within %s)\n", subdir, workDir)
	}

	// M-PKG-CASCADE-DETERMINISTIC-FIRST: try deterministic bump before AI.
	// For class A (content-only) and class B (additive) cascades, the toml
	// bump + lock + check + test is pure computation — no interpretation
	// needed. We can complete the entire workflow in <10s with no AI cost
	// instead of spinning up a 3min haiku/sonnet run.
	var execResult *executor.Result
	var execErr error
	deterministicSuccess := false
	if rootPackage := os.Getenv("AILANG_CASCADE_ROOT_PACKAGE"); rootPackage != "" {
		changeClass := os.Getenv("AILANG_CASCADE_CHANGE_CLASS")
		toVersion := os.Getenv("AILANG_CASCADE_TO_VERSION")
		path := classifyDispatchPath(changeClass)
		fmt.Printf("execute-job: cascade detected — root=%s, change_class=%s, dispatch_path=%s\n",
			rootPackage, changeClass, path)

		if path == DispatchDeterministic {
			bumpErr := deterministicCascadeBump(ctx, execWorkDir, rootPackage, toVersion, taskID, agentID)
			if bumpErr == nil {
				deterministicSuccess = true
				execResult = &executor.Result{
					NumTurns:      0,
					ToolCallCount: 0,
					InputTokens:   0,
					OutputTokens:  0,
					CostUSD:       0,
					SessionID:     "deterministic-bump",
				}
				fmt.Printf("execute-job: ✓ deterministic cascade bump succeeded — skipping AI executor\n")
			} else {
				fmt.Fprintf(os.Stderr, "execute-job: deterministic bump failed (%v) — falling through to AI executor\n", bumpErr)
			}
		}
	}

	// Step 3: Run the AI executor (unless deterministic bump already succeeded)
	if !deterministicSuccess {
		if directive == "" {
			directive = fmt.Sprintf("Execute task %s as agent %s", taskID, agentID)
		}

		// M-GIT-GUARDRAILS: Default to guardrails if not set per-agent or via Terraform.
		if os.Getenv("AILANG_GIT_MODE") == "" {
			os.Setenv("AILANG_GIT_MODE", "guardrails")
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
		execResult, execErr = runExecutor(ctx, execWorkDir, provider, directive, taskID, pluginDir, model, timeoutStr)
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
		// Co-author resolves to the actual model the executor invoked (AILANG_MODEL env var,
		// set by the dispatcher from the agent config). Falls back to "AILANG cascade wrapper"
		// when there's no model — e.g., the deterministic bump path that doesn't invoke AI.
		var commitMsg string
		coAuthor := "AILANG cascade wrapper <noreply@sunholo.com>"
		if m := os.Getenv("AILANG_MODEL"); m != "" {
			coAuthor = fmt.Sprintf("Claude (%s) <noreply@anthropic.com>", m)
		}
		siteSlug := os.Getenv("AILANG_SITE_SLUG")
		briefID := os.Getenv("AILANG_BRIEF_ID")
		if siteSlug != "" {
			subject := fmt.Sprintf("Build: %s", siteSlug)
			if briefID != "" {
				subject += fmt.Sprintf(" [briefId=%s]", briefID)
			}
			commitMsg = fmt.Sprintf("%s\n\nTask: %s\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: %s",
				subject, taskID, agentID, time.Now().UTC().Format(time.RFC3339), coAuthor)
		} else {
			commitMsg = fmt.Sprintf("Task %s: %s\n\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: %s",
				taskID, directive, agentID, time.Now().UTC().Format(time.RFC3339), coAuthor)
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

// DispatchPath classifies how a cascade task should be handled.
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
type DispatchPath string

const (
	// DispatchDeterministic — wrapper applies the toml bump + lock + check + test
	// and only commits if all green. No AI agent is invoked.
	DispatchDeterministic DispatchPath = "deterministic"
	// DispatchAI — interface change with removed exports or widened effects;
	// the AI agent is invoked with full hash context to repair consumers.
	DispatchAI DispatchPath = "ai"
)

// classifyDispatchPath chooses how to handle a cascade task based on the
// change_class from the publisher. This mirrors classifyChange in
// internal/messaging/pkg_events.go but lives wrapper-side so the cloud job
// can decide without re-invoking the classifier.
//
// Mapping:
//
//	A (content-only)  → Deterministic
//	B (additive)      → Deterministic (consumer code keeps working — no exports removed, no effects widened)
//	C (interface)     → AI (something was removed OR effects widened — needs interpretation)
//	(unknown / empty) → AI (conservative default)
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func classifyDispatchPath(changeClass string) DispatchPath {
	switch changeClass {
	case "A", "B":
		return DispatchDeterministic
	default:
		return DispatchAI
	}
}

// deterministicCascadeBump performs the routine cascade work without invoking
// an AI agent: edit ailang.toml to point the dependency at the new version,
// regenerate ailang.lock, run check + test. If any step fails the function
// returns the error and the wrapper falls back to the AI escalation path.
//
// On success it stages and commits the changes; the wrapper's existing
// push + open-PR steps then run unchanged. The commit message tags the work
// as deterministic so the PR shows clearly that no AI was involved.
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func deterministicCascadeBump(ctx context.Context, workDir, rootPackage, toVersion, taskID, agentID string) error {
	// rootPackage format: "vendor/name@x.y.z" — strip the version since we
	// already have it as toVersion (and the toml maps to vendor/name only).
	parts := strings.SplitN(rootPackage, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid root package format %q (expected vendor/name@version)", rootPackage)
	}
	pkgName := parts[0]
	if toVersion == "" {
		toVersion = parts[1]
	}

	// 1. Read ailang.toml from the consumer package directory.
	tomlPath := filepath.Join(workDir, "ailang.toml")
	contentBytes, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("read ailang.toml at %s: %w", tomlPath, err)
	}

	// 2. Find and replace the dep version. ailang.toml uses TOML syntax
	// like:  "vendor/name" = "0.1.2"
	// We match the line with the package name and rewrite the version.
	pattern := fmt.Sprintf(`("%s"\s*=\s*)"[^"]+"`, regexp.QuoteMeta(pkgName))
	re := regexp.MustCompile(pattern)
	replacement := fmt.Sprintf(`${1}"%s"`, toVersion)
	newContent := re.ReplaceAllString(string(contentBytes), replacement)
	if newContent == string(contentBytes) {
		return fmt.Errorf("dep %q not found in ailang.toml at %s (or already at %s)", pkgName, tomlPath, toVersion)
	}
	if err := os.WriteFile(tomlPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write ailang.toml: %w", err)
	}
	fmt.Printf("execute-job: deterministic bump %s → %s in %s\n", pkgName, toVersion, tomlPath)

	// 3. Regenerate ailang.lock against the bumped dep.
	lockCmd := exec.CommandContext(ctx, "ailang", "lock")
	lockCmd.Dir = workDir
	if out, lockErr := lockCmd.CombinedOutput(); lockErr != nil {
		return fmt.Errorf("ailang lock failed: %w (output: %s)", lockErr, strings.TrimSpace(string(out)))
	}
	fmt.Println("execute-job: deterministic ailang lock regenerated")

	// 4. ailang check — this MUST pass for a class A or B bump. If it fails,
	// the change_class was misclassified at publish time or there's a real
	// breakage we didn't anticipate; either way, escalate to AI.
	checkCmd := exec.CommandContext(ctx, "ailang", "check", "--package", ".")
	checkCmd.Dir = workDir
	if out, checkErr := checkCmd.CombinedOutput(); checkErr != nil {
		return fmt.Errorf("ailang check failed (escalating to AI): %w (output: %s)", checkErr, strings.TrimSpace(string(out)))
	}
	fmt.Println("execute-job: deterministic ailang check passed")

	// 5. ailang test — only run if a *_test.ail exists in the package.
	testFiles, _ := filepath.Glob(filepath.Join(workDir, "*_test.ail"))
	if len(testFiles) > 0 {
		testCmd := exec.CommandContext(ctx, "ailang", "test", "--package", ".")
		testCmd.Dir = workDir
		if out, testErr := testCmd.CombinedOutput(); testErr != nil {
			return fmt.Errorf("ailang test failed (escalating to AI): %w (output: %s)", testErr, strings.TrimSpace(string(out)))
		}
		fmt.Printf("execute-job: deterministic ailang test passed (%d test files)\n", len(testFiles))
	}

	// 6. Stage + commit. The wrapper's existing Step 4 detects this commit
	// and the existing Step 5 does the push + PR-open.
	addCmd := exec.CommandContext(ctx, "git", "-C", workDir, "add", "-A")
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	commitMsg := fmt.Sprintf(
		"[cascade] bump %s to %s\n\n"+
			"Deterministic cascade bump by AILANG coordinator wrapper.\n"+
			"No AI agent was invoked — change_class permitted direct\n"+
			"toml bump + lock + check + test. All steps passed.\n\n"+
			"Task: %s\n"+
			"Agent: %s\n"+
			"Timestamp: %s",
		pkgName, toVersion, taskID, agentID, time.Now().UTC().Format(time.RFC3339),
	)
	commitCmd := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "-m", commitMsg)
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}

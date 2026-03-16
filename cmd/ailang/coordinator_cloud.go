package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/executor"
	// Import to trigger init() registration — same as local coordinator (provider_executor.go)
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
	"github.com/sunholo/ailang/internal/pubsub"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	publishCompletion := func(status, errMsg, branchName string, execResult *executor.Result) {
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
			publishCompletion("failed", fmt.Sprintf("panic: %v", r), "", nil)
		} else if !completionSent.Load() {
			// Should not happen — means we returned without publishing.
			publishCompletion("failed", "unknown: exited without publishing completion", "", nil)
		}
	}()

	// Validate required env vars (after Pub/Sub init so failures are reported).
	if taskID == "" {
		publishCompletion("failed", "AILANG_TASK_ID environment variable is required", "", nil)
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}
	if agentID == "" {
		publishCompletion("failed", "AILANG_AGENT_ID environment variable is required", "", nil)
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}
	if projectID == "" {
		publishCompletion("failed", "AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT is required", "", nil)
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
	branchName, execResult, execErr := executeCloudTask(ctx, taskID, agentID, repoURL, branch, directive, provider, pluginRepo, model, timeoutStr)

	// Publish completion with executor metrics (success or failure)
	if execErr != nil {
		publishCompletion("failed", execErr.Error(), branchName, execResult)
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		publishCompletion("completed", "", branchName, execResult)
		fmt.Printf("execute-job: task %s completed (branch=%s)\n", taskID, branchName)
	}

	return execErr
}

// executeCloudTask runs the AI executor in a cloned repository.
// Returns the branch name with changes, executor result with metrics, or error.
//
// When AILANG_PUSH_BRANCH is set, the agent works directly on the cloned branch
// and pushes to that branch (no coordinator/{taskID} branch creation). This is
// used for skip_approval agents like website-builder that push directly to main.
func executeCloudTask(ctx context.Context, taskID, agentID, repoURL, baseBranch, directive, provider, pluginRepo, model, timeoutStr string) (string, *executor.Result, error) {
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
	// Priority: 1) Clone from AILANG_PLUGIN_REPO if set, 2) Use pre-baked /plugins/ailang_bootstrap
	// The Docker image pre-clones the plugin at build time (Dockerfile.agent line 54),
	// so it's usually available without needing AILANG_PLUGIN_REPO at runtime.
	pluginDir := ""
	if pluginRepo != "" {
		pluginDir = filepath.Join("/plugins", taskID, "ailang_bootstrap")
		fmt.Printf("execute-job: cloning plugin repo %s\n", pluginRepo)
		pluginCloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", pluginRepo, pluginDir)
		pluginCloneCmd.Stdout = os.Stdout
		pluginCloneCmd.Stderr = os.Stderr
		if err := pluginCloneCmd.Run(); err != nil {
			// Best effort — fall through to check pre-baked plugin
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
		return "", nil, fmt.Errorf("AILANG_REPO_URL is required: set workspace to GitHub org/repo (e.g., sunholo-data/ailang) in agent config")
	}
	fmt.Printf("execute-job: cloning %s (branch=%s)\n", repoURL, baseBranch)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--branch", baseBranch, "--depth", "1", repoURL, workDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return "", nil, fmt.Errorf("git clone failed: %w", err)
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
			return "", nil, fmt.Errorf("git checkout -b failed: %w", err)
		}
	}

	// Step 3: Run the AI executor using the same infrastructure as local coordinator.
	// This gives us: stream-JSON parsing, token/cost extraction, OTEL spans,
	// session tracking, idle/hard timeouts, and proper executor.Result population.
	if directive == "" {
		directive = fmt.Sprintf("Execute task %s as agent %s", taskID, agentID)
	}

	// M-GIT-GUARDRAILS: Default to guardrails if not set per-agent or via Terraform.
	// Ensures local coordinator runs and test environments also get git guardrails.
	if os.Getenv("AILANG_GIT_MODE") == "" {
		os.Setenv("AILANG_GIT_MODE", "guardrails")
	}

	fmt.Printf("execute-job: running %s executor (unified path)\n", provider)
	execResult, execErr := runExecutor(ctx, workDir, provider, directive, taskID, pluginDir, model, timeoutStr)
	if execErr != nil {
		return branchName, execResult, fmt.Errorf("executor failed: %w", execErr)
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
		return branchName, execResult, fmt.Errorf("git status failed: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		// Step 5a: Stage, commit uncommitted changes
		addCmd := exec.CommandContext(ctx, "git", "-C", workDir, "add", "-A")
		if err := addCmd.Run(); err != nil {
			return branchName, execResult, fmt.Errorf("git add failed: %w", err)
		}

		commitMsg := fmt.Sprintf("Task %s: %s\n\nAgent: %s\nTimestamp: %s\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>",
			taskID, directive, agentID, time.Now().UTC().Format(time.RFC3339))

		commitCmd := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "-m", commitMsg)
		commitCmd.Stdout = os.Stdout
		commitCmd.Stderr = os.Stderr
		if err := commitCmd.Run(); err != nil {
			return branchName, execResult, fmt.Errorf("git commit failed: %w", err)
		}
	} else {
		fmt.Println("execute-job: no uncommitted changes (agent may have committed directly)")
	}

	// Step 5b: Check for ANY unpushed commits (handles both executor commits
	// and commits made by the agent via Bash tool during its session).
	// Without this, agent-committed changes are lost because git status is clean.
	logCmd := exec.CommandContext(ctx, "git", "-C", workDir, "log", fmt.Sprintf("origin/%s..HEAD", branchName), "--oneline")
	logOutput, err := logCmd.Output()
	if err != nil {
		// If the ref doesn't exist (shallow clone), fall back to checking rev-list
		fmt.Fprintf(os.Stderr, "execute-job: git log origin/%s..HEAD failed (shallow clone?): %v\n", branchName, err)
		// Use rev-list as fallback — counts commits ahead of origin
		revCmd := exec.CommandContext(ctx, "git", "-C", workDir, "rev-list", "--count", fmt.Sprintf("origin/%s..HEAD", branchName))
		revOutput, revErr := revCmd.Output()
		if revErr == nil && strings.TrimSpace(string(revOutput)) != "0" {
			logOutput = revOutput // Non-zero means there are unpushed commits
		}
	}

	if len(strings.TrimSpace(string(logOutput))) == 0 {
		fmt.Println("execute-job: no commits to push")
		return branchName, execResult, nil
	}

	fmt.Printf("execute-job: unpushed commits:\n%s", string(logOutput))

	// Step 5c: Push all commits (executor's and agent's)
	if repoURL != "" {
		pushCmd := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", branchName)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return branchName, execResult, fmt.Errorf("git push failed: %w", err)
		}
		fmt.Printf("execute-job: pushed branch %s\n", branchName)
	}

	return branchName, execResult, nil
}

// runExecutor uses the unified executor infrastructure (same as local coordinator).
// Instead of shelling out to raw CLI commands, it uses executor.GlobalFactory() to get
// the registered executor and calls ExecuteStreaming() — giving us stream-JSON parsing,
// token extraction, OTEL spans, session tracking, and a full executor.Result.
func runExecutor(ctx context.Context, workDir, provider, directive, taskID, pluginDir, model, timeoutStr string) (*executor.Result, error) {
	// M-CLOUD-PROGRESS-TRACKING M4: Extract trace context from env (injected by dispatcher).
	// This links Cloud Run Job spans to the coordinator's dispatch span in Cloud Trace.
	ctx = telemetry.ExtractTraceContext(ctx)
	tracer := telemetry.Tracer("cloud_job")
	ctx, span := tracer.Start(ctx, "cloud_job.execute",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("provider", provider),
			attribute.String("agent.id", os.Getenv("AILANG_AGENT_ID")),
		),
	)
	defer span.End()

	// Get executor from global factory (same as local coordinator's provider_executor.go)
	exec, err := executor.GlobalFactory().GetExecutor(provider)
	if err != nil {
		return nil, fmt.Errorf("get %s executor: %w", provider, err)
	}

	// Parse timeout from agent config (M-CLOUD-OAUTH)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: invalid timeout %q, using 30m default: %v\n", timeoutStr, err)
		timeout = 30 * time.Minute
	}

	// Build executor task (matches local coordinator's ExecutorProvider.Execute)
	task := &executor.Task{
		ID:        taskID,
		Directive: directive,
		Workspace: workDir,
		Model:     model,   // From AILANG_MODEL env var (agent config) — empty means executor default
		Timeout:   timeout, // From AILANG_TIMEOUT env var — overrides executor default (5m)
		Metadata:  make(map[string]string),
	}
	if pluginDir != "" {
		task.PluginDirs = []string{pluginDir}
	}

	// Create PubSubBroadcaster for live progress streaming (M-CLOUD-PROGRESS-TRACKING).
	// Reuses the same GCP project/prefix env vars as the completion publisher.
	var broadcaster *coordinator.PubSubBroadcaster
	evtProjectID := os.Getenv("AILANG_CLOUD_PROJECT")
	if evtProjectID == "" {
		evtProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	evtPrefix := os.Getenv("AILANG_TOPIC_PREFIX")
	if evtPrefix == "" {
		evtPrefix = pubsub.DefaultTopicPrefix
	}
	if evtProjectID != "" {
		evtClient, evtErr := pubsub.NewClient(ctx, evtProjectID, evtPrefix)
		if evtErr == nil {
			evtPublisher := pubsub.NewPublisher(evtClient)
			broadcaster = coordinator.NewPubSubBroadcaster(
				evtPublisher,
				workDir,
				log.New(os.Stderr, "[cloud-events] ", log.LstdFlags),
			)
			defer evtClient.Close()
			defer evtPublisher.Stop()
		}
	}

	// M-CLOUD-PROGRESS-TRACKING M3: Parse per-task cost budget from env var.
	var maxCostUSD float64
	if maxCostStr := os.Getenv("AILANG_MAX_COST_USD"); maxCostStr != "" {
		if parsed, parseErr := fmt.Sscanf(maxCostStr, "%f", &maxCostUSD); parsed != 1 || parseErr != nil {
			fmt.Fprintf(os.Stderr, "execute-job: invalid AILANG_MAX_COST_USD=%q, ignoring\n", maxCostStr)
			maxCostUSD = 0
		}
	}

	// Create cancellable context for budget enforcement.
	execCtx, execCancel := context.WithCancel(ctx)
	defer execCancel()

	handler := &cloudEventHandler{
		taskID:      taskID,
		agentID:     os.Getenv("AILANG_AGENT_ID"),
		workspace:   workDir,
		broadcaster: broadcaster,
		maxCostUSD:  maxCostUSD,
		cancel:      execCancel,
	}

	// Execute with streaming using CloudEventHandler for Cloud Logging + Pub/Sub visibility.
	result, err := exec.ExecuteStreaming(execCtx, task, handler)
	if err != nil {
		return nil, fmt.Errorf("%s execution failed: %w", provider, err)
	}

	// Check executor-reported failure (non-fatal error from CLI)
	if result != nil && !result.Success && result.Error != "" {
		return result, fmt.Errorf("%s task failed: %s", provider, result.Error)
	}

	return result, nil
}

// cloudEventHandler logs streaming events to stderr for Cloud Logging visibility
// AND broadcasts them to Pub/Sub for dashboard live progress (M-CLOUD-PROGRESS-TRACKING).
type cloudEventHandler struct {
	taskID      string
	agentID     string
	workspace   string
	broadcaster *coordinator.PubSubBroadcaster // nil if Pub/Sub not available

	// Rate limiting for text broadcasts (avoid flooding Pub/Sub)
	mu            sync.Mutex
	lastBroadcast time.Time

	// Turn tracking
	currentTurn int

	// Budget enforcement (M-CLOUD-PROGRESS-TRACKING M3)
	maxCostUSD float64
	cancel     context.CancelFunc
}

func (h *cloudEventHandler) OnTurnStart(turnNum int) {
	h.currentTurn = turnNum
	fmt.Fprintf(os.Stderr, "claude-stream: [turn %d] started\n", turnNum)
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		StreamType: websocket.TaskStreamTurnStart,
		TurnNum:    turnNum,
		AgentID:    h.agentID,
		Workspace:  h.workspace,
	})
}

func (h *cloudEventHandler) OnText(text string) {
	// Log text snippets (truncated to avoid flooding logs)
	displayText := text
	if len(displayText) > 200 {
		displayText = displayText[:200] + "..."
	}
	trimmed := strings.TrimSpace(displayText)
	if trimmed != "" {
		fmt.Fprintf(os.Stderr, "claude-stream: %s\n", trimmed)
	}
	// Rate-limit text broadcasts to max 1 per 500ms
	h.mu.Lock()
	shouldBroadcast := time.Since(h.lastBroadcast) >= 500*time.Millisecond
	if shouldBroadcast {
		h.lastBroadcast = time.Now()
	}
	h.mu.Unlock()
	if shouldBroadcast && strings.TrimSpace(text) != "" {
		broadcastText := text
		if len(broadcastText) > 500 {
			broadcastText = broadcastText[:500] + "..."
		}
		h.broadcast(&websocket.TaskStreamEvent{
			TaskID:     h.taskID,
			StreamType: websocket.TaskStreamText,
			Text:       broadcastText,
			TurnNum:    h.currentTurn,
			AgentID:    h.agentID,
		})
	}
}

func (h *cloudEventHandler) OnToolUse(toolName string, input string) {
	summary := extractToolSummary(toolName, input)
	fmt.Fprintf(os.Stderr, "claude-stream: [tool] %s: %s\n", toolName, summary)
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		StreamType: websocket.TaskStreamToolUse,
		ToolName:   toolName,
		ToolInput:  summary,
		TurnNum:    h.currentTurn,
		AgentID:    h.agentID,
	})
}

// extractToolSummary pulls the most diagnostic field from a tool's JSON input.
// For Bash: the command. For Write/Read: the file path. For Edit: old→new summary.
func extractToolSummary(toolName, input string) string {
	if input == "" || input == "{}" {
		return "(no input)"
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		// Not JSON — return truncated raw input
		if len(input) > 500 {
			return input[:500] + "..."
		}
		return input
	}
	switch toolName {
	case "Bash":
		if cmd, ok := m["command"].(string); ok {
			if len(cmd) > 500 {
				cmd = cmd[:500] + "..."
			}
			return cmd
		}
	case "Write":
		if fp, ok := m["file_path"].(string); ok {
			return fmt.Sprintf("→ %s", fp)
		}
	case "Read":
		if fp, ok := m["file_path"].(string); ok {
			return fp
		}
	case "Edit":
		fp, _ := m["file_path"].(string)
		old, _ := m["old_string"].(string)
		if len(old) > 100 {
			old = old[:100] + "..."
		}
		return fmt.Sprintf("%s (replacing %q)", fp, old)
	case "Glob":
		if pat, ok := m["pattern"].(string); ok {
			return pat
		}
	case "Grep":
		if pat, ok := m["pattern"].(string); ok {
			return fmt.Sprintf("/%s/", pat)
		}
	}
	// Fallback: truncated JSON
	if len(input) > 500 {
		return input[:500] + "..."
	}
	return input
}

func (h *cloudEventHandler) OnToolResult(toolName string, output string) {
	displayOutput := output
	if len(displayOutput) > 200 {
		displayOutput = displayOutput[:200] + "..."
	}
	fmt.Fprintf(os.Stderr, "claude-stream: [tool-result] %s: %s\n", toolName, displayOutput)
	broadcastOutput := output
	if len(broadcastOutput) > 500 {
		broadcastOutput = broadcastOutput[:500] + "..."
	}
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		StreamType: websocket.TaskStreamToolResult,
		ToolName:   toolName,
		ToolOutput: broadcastOutput,
		TurnNum:    h.currentTurn,
		AgentID:    h.agentID,
	})
}

func (h *cloudEventHandler) OnTurnEnd(turnNum int) {
	fmt.Fprintf(os.Stderr, "claude-stream: [turn %d] ended\n", turnNum)
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		StreamType: websocket.TaskStreamTurnEnd,
		TurnNum:    turnNum,
		AgentID:    h.agentID,
		Workspace:  h.workspace,
	})
}

func (h *cloudEventHandler) OnError(err error) {
	fmt.Fprintf(os.Stderr, "claude-stream: [error] %v\n", err)
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		StreamType: websocket.TaskStreamError,
		ErrorMsg:   err.Error(),
		TurnNum:    h.currentTurn,
		AgentID:    h.agentID,
	})
}

// OnMetrics receives final execution metrics from the executor (cost, tokens, turns).
// Implements executor.MetricsHandler optional interface.
func (h *cloudEventHandler) OnMetrics(metrics executor.ExecutionMetrics) {
	fmt.Fprintf(os.Stderr, "claude-stream: [metrics] turns=%d, tokens=%d+%d, cost=$%.4f\n",
		metrics.NumTurns, metrics.InputTokens, metrics.OutputTokens, metrics.CostUSD)
	status := "completed"
	if !metrics.Success {
		status = "failed"
	}
	h.broadcast(&websocket.TaskStreamEvent{
		TaskID:      h.taskID,
		StreamType:  websocket.TaskStreamStatus,
		Status:      status,
		TurnNum:     metrics.NumTurns,
		TokensIn:    metrics.InputTokens,
		TokensOut:   metrics.OutputTokens,
		Cost:        metrics.CostUSD,
		DurationSec: metrics.DurationMS / 1000,
		AgentID:     h.agentID,
		Workspace:   h.workspace,
	})

	// M-CLOUD-PROGRESS-TRACKING M3: Check cost budget and abort if exceeded.
	if h.maxCostUSD > 0 && metrics.CostUSD > h.maxCostUSD {
		fmt.Fprintf(os.Stderr, "claude-stream: [BUDGET] cost $%.4f exceeds limit $%.4f — aborting\n",
			metrics.CostUSD, h.maxCostUSD)
		h.broadcast(&websocket.TaskStreamEvent{
			TaskID:     h.taskID,
			StreamType: websocket.TaskStreamError,
			ErrorMsg:   fmt.Sprintf("cost budget exceeded ($%.2f > $%.2f limit)", metrics.CostUSD, h.maxCostUSD),
			AgentID:    h.agentID,
		})
		if h.cancel != nil {
			h.cancel()
		}
	}
}

// broadcast sends an event to Pub/Sub if a broadcaster is configured.
// Fire-and-forget: failures are logged but don't affect execution.
func (h *cloudEventHandler) broadcast(event *websocket.TaskStreamEvent) {
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(event)
	}
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
	fmt.Println("  AILANG_MODEL            AI model override (e.g., sonnet, opus)")
	fmt.Println("  AILANG_MAX_COST_USD     Per-task cost budget in USD (0 = unlimited)")
}

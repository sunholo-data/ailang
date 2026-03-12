// Package claude provides an Executor implementation for Claude Code CLI.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var claudeTracer = telemetry.Tracer("executor.claude")

// ClaudeExecutor executes tasks using Claude Code CLI
type ClaudeExecutor struct {
	claudePath     string
	nvmBinDir      string // NVM node bin directory to prepend to PATH (empty if not using NVM)
	model          string
	allowedTools   []string
	permissionMode string
	timeoutSeconds int
}

// New creates a new ClaudeExecutor
func New(cfg *executor.Config) (*ClaudeExecutor, error) {
	claudePath := cfg.ClaudePath
	if claudePath == "" {
		claudePath = "claude"
	}

	// Binary resolution cascade:
	// 1. Explicit config path (non-empty, not "claude") — use as-is
	// 2. VSCode native binary (Mach-O/ELF, no Node.js dependency)
	// 3. System PATH lookup
	// 4. NVM scan (newest version first, with proper semver sort)
	//
	// The native binary is preferred over NVM because it eliminates the
	// Node.js dependency and the PATH injection hack for #!/usr/bin/env node.
	var nvmBinDir string
	if claudePath == "claude" {
		if nativePath := executor.FindNativeBinary("claude"); nativePath != "" {
			// Native binary found — no Node needed, no nvmBinDir
			claudePath = nativePath
		} else if _, err := exec.LookPath("claude"); err != nil {
			// Not in PATH — fall back to NVM scan
			if nvmPath := executor.FindNVMBinary("claude"); nvmPath != "" {
				claudePath = nvmPath
				nvmBinDir = executor.FindNVMNodeBinDir("claude")
			}
		}
	}

	model := cfg.ClaudeModel
	if model == "" {
		model = "haiku"
	}

	tools := cfg.ClaudeTools
	if len(tools) == 0 {
		tools = []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob"}
	}

	permission := cfg.ClaudePermission
	if permission == "" {
		permission = "bypassPermissions"
	}

	return &ClaudeExecutor{
		claudePath:     claudePath,
		nvmBinDir:      nvmBinDir,
		model:          model,
		allowedTools:   tools,
		permissionMode: permission,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier
func (e *ClaudeExecutor) Name() string {
	return "claude"
}

// Execute runs a task and returns the result
func (e *ClaudeExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks
func (e *ClaudeExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	// Start OTEL span for Claude execution
	ctx, span := telemetry.StartSpan(ctx, claudeTracer, "claude.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "claude"),
			attribute.String("executor.model", e.model),
			attribute.String("task.workspace", task.Workspace),
			attribute.String("task.directive", telemetry.Truncate(task.Directive, 500)),
		),
	)
	defer span.End()

	// Update handler's context so child spans (turns, tools) are properly nested
	// under this claude.execute span rather than being siblings
	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}

	// Use task ID as session ID for unified hierarchy tracking
	// This allows: 1) consistent span correlation, 2) Claude session resume using task ID
	// IMPORTANT: Claude Code CLI requires session IDs to be valid UUIDs
	sessionID := task.ID
	if sessionID == "" {
		// Fallback if task has no ID (shouldn't happen in normal use)
		sessionID = uuid.New().String()
	} else if !isValidUUID(sessionID) {
		// Task ID is not a valid UUID (e.g., "exec-test", "task-123")
		// Generate a UUID for Claude's session, but keep task.ID for hierarchy tracking
		sessionID = uuid.New().String()
	}
	span.SetAttributes(attribute.String("session.id", sessionID))
	span.SetAttributes(attribute.Int("task.iteration", task.Iteration))

	// Write OAuth credentials file from env var for cloud auth (M-CLOUD-OAUTH).
	// Claude Code reads ~/.claude/.credentials.json for authentication.
	// In cloud containers, the token is passed via CLAUDE_CODE_OAUTH_TOKEN env var
	// from Secret Manager. This bridges the gap: env var → file → Claude reads it.
	// IMPORTANT: The env var alone causes Claude to exit(1) with zero output.
	// The file-based approach is what works (same as local interactive auth).
	if err := writeCredentialsFile(); err != nil {
		fmt.Fprintf(os.Stderr, "claude-auth: warning: %v\n", err)
	}

	// Install per-agent third-party plugins before execution (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	// This is best-effort — failures don't prevent task execution.
	if task.Plugins != nil {
		e.installPlugins(ctx, task.Plugins, task.Workspace)
	}

	// Build command arguments
	// Note: We do NOT pass --settings here. Claude Code will load project's
	// .claude/settings.json naturally, which has the telemetry hooks configured.
	// This ensures coordinator worktree sessions use the same hooks as interactive sessions.
	args := []string{
		"-p", task.Directive,
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--verbose",
		"--model", e.getModel(task),
	}

	// Permission handling: Cloud environments may have restrictive settings.json files
	// with permission.allow lists that override --permission-mode. For cloud tasks,
	// use --dangerously-skip-permissions to bypass all checks (M-CLOUD-PLUGIN-SKILLS, v0.9.1).
	// For local tasks, use the configured permission mode (default: bypassPermissions).
	isCloudTask := isCloudWorkspace(task.Workspace)
	if isCloudTask && e.permissionMode == "bypassPermissions" {
		// Cloud mode: use dangerously-skip-permissions to bypass restrictive settings.json
		args = append(args, "--dangerously-skip-permissions")
	} else {
		// Local mode or custom permission mode: use permission-mode flag
		args = append(args, "--permission-mode", e.permissionMode)
	}

	// Session handling: use --resume for iterations > 1 with existing session
	// --resume <sessionId> continues existing session with full context
	// --session-id <uuid> starts a new session with that ID
	if task.Iteration > 1 && task.ResumeSessionID != "" {
		args = append(args, "--resume", task.ResumeSessionID)
		span.SetAttributes(attribute.Bool("session.resumed", true))
	} else {
		args = append(args, "--session-id", sessionID)
	}

	if task.SystemPrompt != "" {
		args = append([]string{"--system-prompt", task.SystemPrompt}, args...)
	}

	if task.Workspace != "" {
		args = append(args, "--add-dir", task.Workspace)
	}

	// Effort level (Claude Code 2.1.47+: low/medium/high)
	if task.Effort != "" {
		args = append(args, "--effort", task.Effort)
	}

	// Plugin directories (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	for _, dir := range task.PluginDirs {
		args = append(args, "--plugin-dir", dir)
	}

	// Use task-specific tools if specified, otherwise fall back to executor config.
	// IMPORTANT: Skip --allowedTools in cloud mode with --dangerously-skip-permissions.
	// The --allowedTools flag restricts tool loading and interacts badly with
	// --dangerously-skip-permissions in headless/Docker mode, causing 0-tool-call
	// regressions (Claude receives the prompt but can't initialize tools properly).
	// See: commit 30e388f6 regression analysis.
	if !isCloudTask {
		tools := e.allowedTools
		if len(task.AllowedTools) > 0 {
			tools = task.AllowedTools
		}
		if len(tools) > 0 {
			args = append(args, "--allowedTools", strings.Join(tools, ","))
		}
	}

	cmd := exec.CommandContext(ctx, e.claudePath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	// Log the full command for debugging (helps diagnose cloud executor issues)
	fmt.Fprintf(os.Stderr, "claude-executor: %s %s\n", e.claudePath, strings.Join(args, " "))

	// Set up environment using shared builder (M-UNIFIED-AI-CONTROL-PLANE)
	cmd.Env = executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:                  task,
		SessionID:             sessionID,
		Context:               ctx,
		EnableClaudeTelemetry: true,
	})

	// Strip CLAUDE_CODE_OAUTH_TOKEN from subprocess environment (M-CLOUD-OAUTH).
	// Credentials are written to ~/.claude/.credentials.json above.
	// The env var causes Claude Code to crash (exit 1, 0 turns, no stderr).
	cmd.Env = executor.RemoveEnvVar(cmd.Env, "CLAUDE_CODE_OAUTH_TOKEN")

	// Prepend NVM node bin directory to PATH so the correct Node version
	// is found by the claude shebang (#!/usr/bin/env node).
	// Without this, `env node` resolves to the system/default Node which
	// may be incompatible with the installed Claude Code version.
	// Note: Not needed when using the native binary (nvmBinDir is empty).
	if e.nvmBinDir != "" {
		for i, v := range cmd.Env {
			if strings.HasPrefix(v, "PATH=") {
				cmd.Env[i] = "PATH=" + e.nvmBinDir + ":" + v[5:]
				break
			}
		}
	}

	// NOTE: CLAUDECODE env var is stripped centrally in BuildEnvironment()
	// so it applies to ALL executors (Claude, Gemini, future ones).

	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Set up timeout
	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Parse streaming output
	done := make(chan error, 1)
	var finalResult *claudeHeadlessResult
	var transcriptBuf strings.Builder
	var turnNum int
	var toolCallCount int
	var permissionDeniedCount int        // Track permission denied events (M-CLOUD-PLUGIN-SKILLS)
	var turnSpan trace.Span              // Track current turn's OTEL span
	var currentToolName string           // Accumulates tool name across streaming events
	var currentToolInput strings.Builder // Accumulates input_json_delta chunks

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Increase buffer size to 1MB to handle large JSON output from Claude Code
		// (e.g., tool results with large file contents or base64-encoded images)
		const maxScannerBuffer = 1024 * 1024 // 1MB
		stdoutScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)
		stderrScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

		// Read stderr in background — log to os.Stderr for Cloud Logging visibility.
		// Previously discarded, making it impossible to diagnose tool-use regressions.
		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				fmt.Fprintf(os.Stderr, "claude-stderr: %s\n", line)
			}
		}()

		// Parse NDJSON from stdout
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			if line == "" {
				continue
			}

			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			eventType, _ := event["type"].(string)

			switch eventType {
			case "stream_event":
				streamEvent, _ := event["event"].(map[string]interface{})
				if streamEvent == nil {
					continue
				}

				streamType, _ := streamEvent["type"].(string)
				switch streamType {
				case "message_start":
					turnNum++
					// End previous turn span if still open
					if turnSpan != nil {
						turnSpan.End()
					}
					// Start new turn span as child of claude.execute
					_, turnSpan = telemetry.StartSpan(ctx, claudeTracer, "exec.turn",
						trace.WithAttributes(
							attribute.Int("turn.number", turnNum),
							attribute.String("session.id", sessionID),
						),
					)
					handler.OnTurnStart(turnNum)
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

				case "content_block_start":
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						if blockType, _ := contentBlock["type"].(string); blockType == "tool_use" {
							toolCallCount++
							currentToolName, _ = contentBlock["name"].(string)
							currentToolInput.Reset()
							transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", currentToolName))
						}
					}

				case "content_block_delta":
					delta, _ := streamEvent["delta"].(map[string]interface{})
					if delta != nil {
						deltaType, _ := delta["type"].(string)
						switch deltaType {
						case "text_delta":
							text, _ := delta["text"].(string)
							handler.OnText(text)
							transcriptBuf.WriteString(text)
						case "input_json_delta":
							// Accumulate tool input chunks — emitted at content_block_stop
							if partial, ok := delta["partial_json"].(string); ok {
								currentToolInput.WriteString(partial)
							}
						}
					}

				case "content_block_stop":
					// Emit tool use with complete accumulated input.
					// Claude streams input via input_json_delta after content_block_start,
					// so we must wait until stop to have the full JSON.
					if currentToolName != "" {
						handler.OnToolUse(currentToolName, currentToolInput.String())
						currentToolName = ""
						currentToolInput.Reset()
					}

				case "permission_denied":
					// Permission denied event (M-CLOUD-PLUGIN-SKILLS)
					// This occurs when Claude tries to use a tool that's not in the allow list
					permissionDeniedCount++
					toolName, _ := streamEvent["tool"].(string)
					handler.OnError(fmt.Errorf("permission denied: tool %q not allowed", toolName))
					transcriptBuf.WriteString(fmt.Sprintf("[PERMISSION DENIED] %s\n", toolName))

				case "message_stop":
					handler.OnTurnEnd(turnNum)
					// End turn span
					if turnSpan != nil {
						turnSpan.End()
						turnSpan = nil
					}
				}

			case "result":
				if err := json.Unmarshal([]byte(line), &finalResult); err != nil {
					// Cleanup turn span before early return
					if turnSpan != nil {
						turnSpan.End()
					}
					done <- fmt.Errorf("failed to parse final result: %w", err)
					return
				}
			}
		}

		// Cleanup any open turn span
		if turnSpan != nil {
			turnSpan.End()
		}

		if err := stdoutScanner.Err(); err != nil {
			done <- fmt.Errorf("stdout scanner error: %w", err)
			return
		}

		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case <-timer.C:
		_ = cmd.Process.Kill()
		timeoutErr := fmt.Errorf("timeout after %v", timeout)
		handler.OnError(timeoutErr)
		span.RecordError(timeoutErr)
		span.SetStatus(codes.Error, "timeout")
		span.SetAttributes(
			attribute.Int("task.turns", turnNum),
			attribute.Bool("task.success", false),
		)
		return &executor.Result{
			Success:       false,
			Error:         fmt.Sprintf("timeout after %v", timeout),
			DurationMS:    int(time.Since(startTime).Milliseconds()),
			NumTurns:      turnNum,
			ToolCallCount: toolCallCount,
			SessionID:     sessionID,
			Transcript:    transcriptBuf.String(),
		}, nil

	case err := <-done:
		duration := time.Since(startTime)

		if err != nil {
			handler.OnError(err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", false),
			)
			return &executor.Result{
				Success:       false,
				Error:         err.Error(),
				DurationMS:    int(duration.Milliseconds()),
				NumTurns:      turnNum,
				ToolCallCount: toolCallCount,
				SessionID:     sessionID,
				Transcript:    transcriptBuf.String(),
			}, nil
		}

		if finalResult == nil {
			span.SetStatus(codes.Ok, "session completed")
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", true),
				attribute.Int("task.duration_ms", int(duration.Milliseconds())),
			)
			return &executor.Result{
				Success:       true,
				Output:        "Session completed",
				DurationMS:    int(duration.Milliseconds()),
				NumTurns:      turnNum,
				ToolCallCount: toolCallCount,
				SessionID:     sessionID,
				Transcript:    transcriptBuf.String(),
			}, nil
		}

		success := !finalResult.IsError && finalResult.Subtype == "success"

		// Detect permission failure: if we had tool calls but they were all denied,
		// treat as failure even if Claude says "success" (M-CLOUD-PLUGIN-SKILLS)
		if success && toolCallCount > 0 && permissionDeniedCount >= toolCallCount {
			success = false
		}

		span.SetAttributes(
			attribute.Int("task.turns", finalResult.NumTurns),
			attribute.Bool("task.success", success),
			attribute.Int("task.duration_ms", finalResult.DurationMS),
			attribute.Int64("task.tokens_in", int64(finalResult.Usage.InputTokens)),
			attribute.Int64("task.tokens_out", int64(finalResult.Usage.OutputTokens)),
			attribute.Float64("task.cost_usd", finalResult.TotalCostUSD),
			attribute.Int("permission.denied_count", permissionDeniedCount),
		)

		errorMsg := getErrorMessage(finalResult)
		if permissionDeniedCount > 0 && errorMsg == "" {
			errorMsg = fmt.Sprintf("permission denied: %d tool calls were blocked", permissionDeniedCount)
		}

		if !success {
			span.SetStatus(codes.Error, errorMsg)
		} else {
			span.SetStatus(codes.Ok, "task completed successfully")
		}

		return &executor.Result{
			Success:                  success,
			Output:                   finalResult.Result,
			Error:                    errorMsg,
			DurationMS:               finalResult.DurationMS,
			NumTurns:                 finalResult.NumTurns,
			ToolCallCount:            toolCallCount,
			CostUSD:                  finalResult.TotalCostUSD,
			InputTokens:              finalResult.Usage.InputTokens,
			OutputTokens:             finalResult.Usage.OutputTokens,
			CacheReadInputTokens:     finalResult.Usage.CacheReadInputTokens,
			CacheCreationInputTokens: finalResult.Usage.CacheCreationInputTokens,
			SessionID:                sessionID,
			Transcript:               transcriptBuf.String(),
		}, nil
	}
}

// Capabilities returns the list of features this executor supports
func (e *ClaudeExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapToolControl,
		executor.CapSessionResume,
		executor.CapLocalWorkspace,
	}
}

// CostModel returns pricing information for cost calculations
func (e *ClaudeExecutor) CostModel() *executor.CostModel {
	// Default to Haiku pricing
	return &executor.CostModel{
		ProviderName:    "anthropic",
		InputTokenCost:  0.001,  // $1.00 per 1M
		OutputTokenCost: 0.005,  // $5.00 per 1M
		CacheReadCost:   0.0001, // $0.10 per 1M
	}
}

// HealthCheck verifies the executor is configured and accessible
func (e *ClaudeExecutor) HealthCheck(ctx context.Context) error {
	// Check if claude binary exists
	_, err := exec.LookPath(e.claudePath)
	if err != nil {
		// Also try the resolved path directly (LookPath fails for absolute paths on some systems)
		if _, statErr := os.Stat(e.claudePath); statErr != nil {
			return fmt.Errorf("claude CLI not found: %w (install from https://claude.ai/code)", err)
		}
	}
	return nil
}

// Close releases any resources held by the executor
func (e *ClaudeExecutor) Close() error {
	return nil
}

func (e *ClaudeExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// claudeHeadlessResult matches Claude CLI output structure
type claudeHeadlessResult struct {
	Type         string      `json:"type"`
	Subtype      string      `json:"subtype"`
	IsError      bool        `json:"is_error"`
	Result       string      `json:"result"`
	NumTurns     int         `json:"num_turns"`
	DurationMS   int         `json:"duration_ms"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	SessionID    string      `json:"session_id"`
	Usage        claudeUsage `json:"usage"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func getErrorMessage(result *claudeHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" || result.Subtype == "timeout" {
		return result.Result
	}
	return ""
}

// isCloudWorkspace detects if a workspace path is in a cloud container.
// Cloud paths typically start with /workspace/ (Cloud Run, Google Cloud Container).
// This detection ensures we use appropriate permission handling for cloud environments.
func isCloudWorkspace(workspace string) bool {
	// Cloud container paths typically start with /workspace/
	// This convention is used by Cloud Run, Pub/Sub-triggered containers, and GCP workspaces
	return strings.HasPrefix(workspace, "/workspace/")
}

// isValidUUID checks if a string is a valid UUID format
// Claude Code CLI requires session IDs to be valid UUIDs
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// installPlugins registers marketplaces and installs third-party plugins.
// This runs before task execution. Best-effort: failures are logged but don't block execution.
func (e *ClaudeExecutor) installPlugins(ctx context.Context, plugins *executor.PluginsConfig, workspace string) {
	if plugins == nil {
		return
	}

	for _, mkt := range plugins.Marketplaces {
		cmd := exec.CommandContext(ctx, e.claudePath, "plugin", "marketplace", "add", mkt)
		if workspace != "" {
			cmd.Dir = workspace
		}
		if e.nvmBinDir != "" {
			cmd.Env = os.Environ()
			for i, v := range cmd.Env {
				if strings.HasPrefix(v, "PATH=") {
					cmd.Env[i] = "PATH=" + e.nvmBinDir + ":" + v[5:]
					break
				}
			}
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to add marketplace %s: %v (%s)\n", mkt, err, strings.TrimSpace(string(output)))
		}
	}

	for _, plugin := range plugins.Install {
		cmd := exec.CommandContext(ctx, e.claudePath, "plugin", "install", plugin)
		if workspace != "" {
			cmd.Dir = workspace
		}
		if e.nvmBinDir != "" {
			cmd.Env = os.Environ()
			for i, v := range cmd.Env {
				if strings.HasPrefix(v, "PATH=") {
					cmd.Env[i] = "PATH=" + e.nvmBinDir + ":" + v[5:]
					break
				}
			}
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to install plugin %s: %v (%s)\n", plugin, err, strings.TrimSpace(string(output)))
		}
	}
}

// writeCredentialsFile writes ~/.claude/.credentials.json from the
// CLAUDE_CODE_OAUTH_TOKEN environment variable (M-CLOUD-OAUTH).
//
// Claude Code authenticates locally via ~/.claude/.credentials.json.
// In cloud containers (Cloud Run Jobs), the OAuth token is injected as an
// env var from Secret Manager. This function bridges the two:
//
//	env var (inner):  {"accessToken":"...","refreshToken":"...","expiresAt":...}
//	file (wrapper):   {"claudeAiOauth":{"accessToken":"...","refreshToken":"...","expiresAt":...}}
//
// Returns nil if CLAUDE_CODE_OAUTH_TOKEN is not set (no-op for local dev).
func writeCredentialsFile() error {
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if token == "" {
		return nil
	}

	// Validate the token is valid JSON
	var inner json.RawMessage
	if err := json.Unmarshal([]byte(token), &inner); err != nil {
		return fmt.Errorf("CLAUDE_CODE_OAUTH_TOKEN is not valid JSON: %w", err)
	}

	// Wrap in the credentials file format that Claude Code expects
	wrapper := map[string]json.RawMessage{
		"claudeAiOauth": inner,
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to ~/.claude/.credentials.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		return fmt.Errorf("failed to create .claude dir: %w", err)
	}

	credPath := filepath.Join(claudeDir, ".credentials.json")
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "claude-auth: wrote credentials to %s (%d bytes)\n", credPath, len(data))
	return nil
}

// Register registers the Claude executor with the global factory
func Register() {
	executor.GlobalFactory().Register("claude", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}

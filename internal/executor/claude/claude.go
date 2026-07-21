// Package claude provides an Executor implementation for Claude Code CLI.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
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

	// M-CLOUD-DUAL-AUTH: Branch on auth mode.
	// "apikey" mode: ANTHROPIC_API_KEY is already in env (set by cloud dispatcher).
	// Claude Code reads it natively — no credentials file needed.
	// Default (OAuth) mode: write credentials file from CLAUDE_CODE_OAUTH_TOKEN.
	authMode := os.Getenv("AILANG_AUTH_MODE")
	if authMode == "apikey" {
		// Decrypt KMS-encrypted API key if present (ENC: prefix).
		if err := decryptAPIKeyIfNeeded(ctx); err != nil {
			return nil, fmt.Errorf("claude-auth: %w", err)
		}
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			return nil, fmt.Errorf("AILANG_AUTH_MODE=apikey but ANTHROPIC_API_KEY not set")
		}
		fmt.Fprintf(os.Stderr, "claude-auth: using ANTHROPIC_API_KEY (pay-per-token mode)\n")
	} else {
		// Write OAuth credentials file from env var for cloud auth (M-CLOUD-OAUTH).
		// Claude Code reads ~/.claude/.credentials.json for authentication.
		// In cloud containers, the token is passed via CLAUDE_CODE_OAUTH_TOKEN env var
		// from Secret Manager. This bridges the gap: env var → file → Claude reads it.
		// IMPORTANT: The env var alone causes Claude to exit(1) with zero output.
		// The file-based approach is what works (same as local interactive auth).
		if err := writeCredentialsFile(); err != nil {
			fmt.Fprintf(os.Stderr, "claude-auth: warning: %v\n", err)
		}
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
		GCPProject:            task.GCPProject,
		GCPLocation:           task.GCPLocation,
	})

	// Strip CLAUDE_CODE_OAUTH_TOKEN from subprocess environment (M-CLOUD-OAUTH).
	// Credentials are written to ~/.claude/.credentials.json above.
	// The env var causes Claude Code to crash (exit 1, 0 turns, no stderr).
	cmd.Env = executor.RemoveEnvVar(cmd.Env, "CLAUDE_CODE_OAUTH_TOKEN")

	// M-CLOUD-DUAL-AUTH: In OAuth mode, strip ANTHROPIC_API_KEY to prevent conflicts.
	// In apikey mode, keep it — Claude Code reads it natively.
	if authMode != "apikey" {
		cmd.Env = executor.RemoveEnvVar(cmd.Env, "ANTHROPIC_API_KEY")
	}

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
	var permissionDeniedCount int
	var turnSpan trace.Span
	var currentToolName string
	var currentToolInput strings.Builder

	// M-EVAL-COST-AND-SPEED-BUDGETS: speed instrumentation.
	// firstAttemptMs records ms from task start to the first time the agent
	// commits a candidate solution (first Write/Edit tool call OR first
	// assistant text if no tool calls). costKilled signals cost-budget
	// breach mid-stream (budget.Add() returned exceeded == true).
	var firstAttemptMs int64 = -1
	var firstStreamEventAt time.Time
	var costKilled bool
	// runningInputTokens / runningOutputTokens track the cumulative usage
	// reported in message_delta events; we feed deltas into Budget.Add().
	// Claude emits cumulative output_tokens in message_delta and the full
	// usage block only at the terminal "result" event.
	var runningInputTokens, runningOutputTokens int

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

			// Hook-reality capture (M-EVAL-FMT-WEAKMODEL-AB / M2b RESOLVED): the
			// LANDED format_ail.sh PostToolUse hook exits 0 to the CLI and, on
			// success, prints "✓ Formatted <file>" to stderr. Claude Code SWALLOWS
			// an exit-0 hook's stderr in --output-format stream-json mode — the
			// marker appears on NEITHER this stdout stream NOR the CLI's own
			// stderr — so a stdout stream-scan here was structurally always empty.
			// Verified live (two haiku ON-arm smoke runs banked 0 events despite a
			// .ail file being written). The capture point therefore moved
			// OUT-OF-BAND: the hook appends a JSONL event to
			// <workspace>/.claude/fmt_hook_events.jsonl, which the eval harness
			// reads post-run (ReadFmtHookSink). There is intentionally NO
			// per-stream-line hook dispatch here anymore; the RawStreamLineHandler
			// interface remains available as a generic extension point.

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
					if firstStreamEventAt.IsZero() {
						firstStreamEventAt = time.Now()
					}
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						if blockType, _ := contentBlock["type"].(string); blockType == "tool_use" {
							toolCallCount++
							currentToolName, _ = contentBlock["name"].(string)
							currentToolInput.Reset()
							transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", currentToolName))
							// M-EVAL-COST-AND-SPEED-BUDGETS: first Write/Edit = first solution attempt.
							if firstAttemptMs < 0 && (currentToolName == "Write" || currentToolName == "Edit") {
								firstAttemptMs = time.Since(startTime).Milliseconds()
							}
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

				case "message_delta":
					// M-EVAL-COST-AND-SPEED-BUDGETS: incremental cost tally.
					// Claude emits cumulative usage in message_delta.usage; convert
					// cumulative→delta and feed to Budget.Add(). Out-of-order or
					// duplicate values are guarded by max(running, new).
					if usage, ok := streamEvent["usage"].(map[string]interface{}); ok {
						newIn := intFromAny(usage["input_tokens"])
						newOut := intFromAny(usage["output_tokens"])
						deltaIn := newIn - runningInputTokens
						deltaOut := newOut - runningOutputTokens
						if deltaIn < 0 {
							deltaIn = 0
						}
						if deltaOut < 0 {
							deltaOut = 0
						}
						runningInputTokens = newIn
						runningOutputTokens = newOut
						if task.Budget != nil && (deltaIn > 0 || deltaOut > 0) {
							if _, exceeded := task.Budget.Add(deltaIn, deltaOut); exceeded {
								costKilled = true
								_ = cmd.Process.Kill()
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
				// M-EVAL-COST-AND-SPEED-BUDGETS: reconcile cumulative usage from final result.
				// message_delta deltas may under-count cache tokens; the result event has
				// the canonical totals. Add only the residual to keep Budget.Current accurate.
				if task.Budget != nil && finalResult != nil {
					residualIn := finalResult.Usage.InputTokens - runningInputTokens
					residualOut := finalResult.Usage.OutputTokens - runningOutputTokens
					if residualIn < 0 {
						residualIn = 0
					}
					if residualOut < 0 {
						residualOut = 0
					}
					if residualIn > 0 || residualOut > 0 {
						_, _ = task.Budget.Add(residualIn, residualOut)
					}
					runningInputTokens = finalResult.Usage.InputTokens
					runningOutputTokens = finalResult.Usage.OutputTokens
				}
				// Notify MetricsHandler with cost/token data (M-CLOUD-PROGRESS-TRACKING).
				// This lets cloud handlers broadcast metrics before the executor returns.
				if mh, ok := handler.(executor.MetricsHandler); ok && finalResult != nil {
					mh.OnMetrics(executor.ExecutionMetrics{
						NumTurns:     finalResult.NumTurns,
						InputTokens:  finalResult.Usage.InputTokens,
						OutputTokens: finalResult.Usage.OutputTokens,
						CostUSD:      finalResult.TotalCostUSD,
						DurationMS:   finalResult.DurationMS,
						SessionID:    finalResult.SessionID,
						Success:      !finalResult.IsError,
					})
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
			Success:        false,
			Error:          fmt.Sprintf("timeout after %v", timeout),
			DurationMS:     int(time.Since(startTime).Milliseconds()),
			NumTurns:       turnNum,
			ToolCallCount:  toolCallCount,
			SessionID:      sessionID,
			Transcript:     transcriptBuf.String(),
			InputTokens:    runningInputTokens,
			OutputTokens:   runningOutputTokens,
			CostKilledAt:   task.Budget.KilledAt(),
			FirstAttemptMs: firstAttemptMs,
			SuccessAtMs:    -1,
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
			errMsg := err.Error()
			if costKilled {
				errMsg = fmt.Sprintf("cost budget exceeded ($%.4f) — %s", task.Budget.KilledAt(), errMsg)
			}
			return &executor.Result{
				Success:        false,
				Error:          errMsg,
				DurationMS:     int(duration.Milliseconds()),
				NumTurns:       turnNum,
				ToolCallCount:  toolCallCount,
				SessionID:      sessionID,
				Transcript:     transcriptBuf.String(),
				InputTokens:    runningInputTokens,
				OutputTokens:   runningOutputTokens,
				CostKilledAt:   task.Budget.KilledAt(),
				FirstAttemptMs: firstAttemptMs,
				SuccessAtMs:    -1,
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
				Success:        true,
				Output:         "Session completed",
				DurationMS:     int(duration.Milliseconds()),
				NumTurns:       turnNum,
				ToolCallCount:  toolCallCount,
				SessionID:      sessionID,
				Transcript:     transcriptBuf.String(),
				InputTokens:    runningInputTokens,
				OutputTokens:   runningOutputTokens,
				CostKilledAt:   task.Budget.KilledAt(),
				FirstAttemptMs: firstAttemptMs,
				SuccessAtMs:    -1,
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
		if costKilled && errorMsg == "" {
			errorMsg = fmt.Sprintf("cost budget exceeded ($%.4f)", task.Budget.KilledAt())
			success = false
		}

		if !success {
			span.SetStatus(codes.Error, errorMsg)
		} else {
			span.SetStatus(codes.Ok, "task completed successfully")
		}

		// M-EVAL-COST-AND-SPEED-BUDGETS: speed metrics.
		// TokensPerSec = output_tokens / generation_seconds (first event → end).
		var tokensPerSec float64
		if !firstStreamEventAt.IsZero() && finalResult.Usage.OutputTokens > 0 {
			genSec := time.Since(firstStreamEventAt).Seconds()
			if genSec > 0 {
				tokensPerSec = float64(finalResult.Usage.OutputTokens) / genSec
			}
		}
		if firstAttemptMs < 0 {
			firstAttemptMs = -1
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
			CostKilledAt:             task.Budget.KilledAt(),
			FirstAttemptMs:           firstAttemptMs,
			SuccessAtMs:              -1,
			TokensPerSec:             tokensPerSec,
			FinishReason:             normalizeClaudeFinishReason(finalResult.Subtype),
		}, nil
	}
}

// normalizeClaudeFinishReason maps the Claude Code headless result subtype onto
// the executor.Result.FinishReason vocabulary.
//
// The subtype was already parsed and used for the success boolean, but the
// reason for a failure was discarded — so a run that burned through its turn
// budget banked identically to one that failed on the task itself.
// "error_max_turns" maps to "max_turns", which CategorizeAgentError already
// aliases to step_exhausted.
func normalizeClaudeFinishReason(subtype string) string {
	switch subtype {
	case "":
		return ""
	case "success":
		return "stop"
	case "error_max_turns":
		return "max_turns"
	case "error_during_execution":
		return "error"
	default:
		return strings.ToLower(subtype)
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

// Register registers the Claude executor with the global factory
func Register() {
	executor.GlobalFactory().Register("claude", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}

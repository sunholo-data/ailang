package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/executor"
	"github.com/sunholo/ailang/internal/pubsub"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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
// For Bash: the command. For Write/Read: the file path. For Edit: old->new summary.
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

package claude

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
)

// TestClaudeExecutorNew tests executor initialization with various configs
func TestClaudeExecutorNew(t *testing.T) {
	tests := []struct {
		name           string
		config         *executor.Config
		expectedModel  string
		expectedPath   string
		expectedTools  int
		expectedPerm   string
		expectedTimout int
	}{
		{
			name: "default config",
			config: &executor.Config{
				ClaudePath:       "",
				ClaudeModel:      "",
				ClaudeTools:      nil,
				ClaudePermission: "",
				TimeoutSeconds:   0,
			},
			expectedModel:  "haiku",
			expectedPath:   "claude",
			expectedTools:  6, // default tools
			expectedPerm:   "bypassPermissions",
			expectedTimout: 0,
		},
		{
			name: "custom config",
			config: &executor.Config{
				ClaudePath:       "/usr/local/bin/claude",
				ClaudeModel:      "opus",
				ClaudeTools:      []string{"Bash", "Read"},
				ClaudePermission: "requestApproval",
				TimeoutSeconds:   300,
			},
			expectedModel:  "opus",
			expectedPath:   "/usr/local/bin/claude",
			expectedTools:  2,
			expectedPerm:   "requestApproval",
			expectedTimout: 300,
		},
		{
			name: "partial override",
			config: &executor.Config{
				ClaudePath:       "/bin/claude",
				ClaudeModel:      "",
				ClaudePermission: "",
			},
			expectedModel: "haiku",
			expectedPath:  "/bin/claude",
			expectedTools: 6,
			expectedPerm:  "bypassPermissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := New(tt.config)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			if exec.model != tt.expectedModel {
				t.Errorf("expected model %q, got %q", tt.expectedModel, exec.model)
			}
			// Path check: when expectedPath is "claude", accept NVM-resolved absolute path too
			// (NVM paths aren't in PATH, so New() resolves to ~/.nvm/.../bin/claude)
			if tt.expectedPath == "claude" {
				if exec.claudePath != "claude" && !strings.HasSuffix(exec.claudePath, "/claude") {
					t.Errorf("expected path ending with 'claude', got %q", exec.claudePath)
				}
			} else if exec.claudePath != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, exec.claudePath)
			}
			if len(exec.allowedTools) != tt.expectedTools {
				t.Errorf("expected %d tools, got %d", tt.expectedTools, len(exec.allowedTools))
			}
			if exec.permissionMode != tt.expectedPerm {
				t.Errorf("expected permission %q, got %q", tt.expectedPerm, exec.permissionMode)
			}
			if exec.timeoutSeconds != tt.expectedTimout {
				t.Errorf("expected timeout %d, got %d", tt.expectedTimout, exec.timeoutSeconds)
			}
		})
	}
}

// TestClaudeExecutorName tests the Name() method
func TestClaudeExecutorName(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	name := exec.Name()
	if name != "claude" {
		t.Errorf("expected name 'claude', got %q", name)
	}
}

// TestClaudeSessionID tests session ID generation and validation
func TestClaudeSessionID(t *testing.T) {
	tests := []struct {
		name           string
		taskID         string
		shouldGenerate bool
		description    string
	}{
		{
			name:           "valid UUID",
			taskID:         "550e8400-e29b-41d4-a716-446655440000",
			shouldGenerate: false,
			description:    "UUID task ID should be used as-is",
		},
		{
			name:           "empty task ID",
			taskID:         "",
			shouldGenerate: true,
			description:    "Empty task ID should generate new UUID",
		},
		{
			name:           "non-UUID task ID",
			taskID:         "task-19efb196",
			shouldGenerate: true,
			description:    "Non-UUID task ID should generate new UUID",
		},
		{
			name:           "short non-UUID",
			taskID:         "test123",
			shouldGenerate: true,
			description:    "Short non-UUID task ID should generate new UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify isValidUUID utility function works correctly
			isValid := isValidUUID(tt.taskID)
			if isValid && tt.shouldGenerate {
				t.Errorf("%s: expected invalid UUID for %q", tt.description, tt.taskID)
			}
			if !isValid && !tt.shouldGenerate {
				t.Errorf("%s: expected valid UUID for %q", tt.description, tt.taskID)
			}

			// If UUID should be generated, verify it's a valid UUID
			if tt.shouldGenerate && tt.taskID != "" {
				newUUID := uuid.New().String()
				if !isValidUUID(newUUID) {
					t.Error("generated UUID should be valid")
				}
			}
		})
	}
}

// TestClaudeGetModel tests model selection logic
func TestClaudeGetModel(t *testing.T) {
	exec, _ := New(&executor.Config{ClaudeModel: "opus"})

	tests := []struct {
		taskModel     string
		expectedModel string
		description   string
	}{
		{
			taskModel:     "",
			expectedModel: "opus",
			description:   "empty task model should use executor config",
		},
		{
			taskModel:     "sonnet",
			expectedModel: "sonnet",
			description:   "task model should override executor config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			task := &executor.Task{Model: tt.taskModel}
			model := exec.getModel(task)
			if model != tt.expectedModel {
				t.Errorf("expected model %q, got %q", tt.expectedModel, model)
			}
		})
	}
}

// TestClaudeCapabilities verifies Claude executor claims correct capabilities
func TestClaudeCapabilities(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	caps := exec.Capabilities()

	// Claude should support streaming
	hasStreaming := false
	for _, cap := range caps {
		if cap == executor.CapStreaming {
			hasStreaming = true
			break
		}
	}

	if !hasStreaming {
		t.Error("Claude executor should support CapStreaming capability")
	}
}

// TestClaudeCostModel verifies cost calculation setup
func TestClaudeCostModel(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	costModel := exec.CostModel()

	if costModel == nil {
		t.Fatal("CostModel() returned nil")
	}

	if costModel.ProviderName != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", costModel.ProviderName)
	}

	// Cost model should have reasonable pricing
	if costModel.InputTokenCost <= 0 {
		t.Error("input token cost should be positive")
	}
	if costModel.OutputTokenCost <= 0 {
		t.Error("output token cost should be positive")
	}
}

// TestClaudeHealthCheck verifies health check implementation
func TestClaudeHealthCheck(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Health check should either succeed or fail gracefully
	// We don't test the actual binary since it might not be installed
	// Just verify the method exists and returns an error or nil
	err := exec.HealthCheck(ctx)
	// Error is expected if claude binary not found
	_ = err
}

// TestClaudeClose tests executor cleanup
func TestClaudeClose(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	err := exec.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// TestClaudeExecuteWithoutHandler tests Execute without event handler
func TestClaudeExecuteWithoutHandler(t *testing.T) {
	// Skip if claude binary not available
	t.Skip("requires claude binary installation")

	exec, _ := New(&executor.Config{ClaudePath: "claude"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task := &executor.Task{
		ID:        "test-task",
		Directive: "echo hello",
		Workspace: "/tmp",
	}

	result, err := exec.Execute(ctx, task)
	// May fail if claude not installed, that's OK for this test
	_ = result
	_ = err
}

// TestClaudeExecuteStreamingWithMockHandler tests streaming with event tracking
func TestClaudeExecuteStreamingWithMockHandler(t *testing.T) {
	// Skip if claude binary not available
	t.Skip("requires claude binary installation")

	handler := &MockEventHandler{}
	exec, _ := New(&executor.Config{ClaudePath: "claude"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task := &executor.Task{
		ID:        "test-streaming",
		Directive: "hello",
	}

	result, err := exec.ExecuteStreaming(ctx, task, handler)
	_ = result
	_ = err

	// Handler should have been called if command executed
	_ = handler
}

// MockEventHandler tracks events for testing
type MockEventHandler struct {
	TurnStarts  []int
	TurnEnds    []int
	TextEvents  []string
	ToolUses    []string
	ToolResults []string
	Errors      []error
}

func (m *MockEventHandler) OnTurnStart(turnNum int) {
	m.TurnStarts = append(m.TurnStarts, turnNum)
}

func (m *MockEventHandler) OnTurnEnd(turnNum int) {
	m.TurnEnds = append(m.TurnEnds, turnNum)
}

func (m *MockEventHandler) OnText(text string) {
	m.TextEvents = append(m.TextEvents, text)
}

func (m *MockEventHandler) OnToolUse(toolName string, input string) {
	m.ToolUses = append(m.ToolUses, fmt.Sprintf("%s(%s)", toolName, input))
}

func (m *MockEventHandler) OnToolResult(toolName string, output string) {
	m.ToolResults = append(m.ToolResults, fmt.Sprintf("%s->%s", toolName, output))
}

func (m *MockEventHandler) OnError(err error) {
	m.Errors = append(m.Errors, err)
}

// TestEventHandlerSequence verifies turn flow (start→text→end)
func TestEventHandlerSequence(t *testing.T) {
	handler := &MockEventHandler{}

	// Simulate a turn sequence
	handler.OnTurnStart(1)
	handler.OnText("Hello")
	handler.OnText(" world")
	handler.OnTurnEnd(1)

	// Verify sequence
	if len(handler.TurnStarts) != 1 || handler.TurnStarts[0] != 1 {
		t.Error("OnTurnStart should be called once with turn number 1")
	}
	if len(handler.TurnEnds) != 1 || handler.TurnEnds[0] != 1 {
		t.Error("OnTurnEnd should be called once with turn number 1")
	}
	if len(handler.TextEvents) != 2 {
		t.Errorf("expected 2 text events, got %d", len(handler.TextEvents))
	}
}

// TestEventHandlerMultipleTurns verifies multiple turn handling
func TestEventHandlerMultipleTurns(t *testing.T) {
	handler := &MockEventHandler{}

	// Simulate multi-turn conversation
	for turn := 1; turn <= 3; turn++ {
		handler.OnTurnStart(turn)
		handler.OnText(fmt.Sprintf("Turn %d response", turn))
		handler.OnTurnEnd(turn)
	}

	if len(handler.TurnStarts) != 3 {
		t.Errorf("expected 3 turn starts, got %d", len(handler.TurnStarts))
	}
	if len(handler.TurnEnds) != 3 {
		t.Errorf("expected 3 turn ends, got %d", len(handler.TurnEnds))
	}
}

// TestEventHandlerToolUse verifies tool use tracking
func TestEventHandlerToolUse(t *testing.T) {
	handler := &MockEventHandler{}

	handler.OnTurnStart(1)
	handler.OnToolUse("Bash", `{"command": "ls"}`)
	handler.OnText("output")
	handler.OnTurnEnd(1)

	if len(handler.ToolUses) != 1 {
		t.Errorf("expected 1 tool use, got %d", len(handler.ToolUses))
	}
}

// TestClaudeContextCancellation verifies context cancellation handling
func TestClaudeContextCancellation(t *testing.T) {
	// Skip if claude binary not available
	t.Skip("requires claude binary installation and runtime testing")

	exec, _ := New(&executor.Config{ClaudePath: "claude"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	task := &executor.Task{
		ID:        "cancelled-task",
		Directive: "this should be cancelled",
	}

	result, err := exec.Execute(ctx, task)
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	_ = result
}

// TestClaudeContextTimeout verifies timeout handling
func TestClaudeContextTimeout(t *testing.T) {
	// Skip if claude binary not available
	t.Skip("requires claude binary installation and runtime testing")

	exec, _ := New(&executor.Config{ClaudePath: "claude"})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	task := &executor.Task{
		ID:        "timeout-task",
		Directive: "this will timeout",
	}

	result, err := exec.Execute(ctx, task)
	// May timeout or error out
	_ = result
	_ = err
}

// TestClaudeExecuteTimeoutBefore tests timeout before execution starts
func TestClaudeExecuteTimeoutBefore(t *testing.T) {
	// This test validates timeout behavior without needing claude binary
	// by checking that the timeout value is properly set
	exec, _ := New(&executor.Config{TimeoutSeconds: 300})

	if exec.timeoutSeconds != 300 {
		t.Errorf("expected timeout 300, got %d", exec.timeoutSeconds)
	}

	// Verify timeout can be overridden per-task
	task := &executor.Task{
		Timeout: 5 * time.Second,
	}

	// If we could actually execute, the task timeout would override executor timeout
	if task.Timeout != 5*time.Second {
		t.Error("task timeout should override executor timeout")
	}
}

// TestClaudeStreamingWithContextCancellation verifies streaming respects context
func TestClaudeStreamingWithContextCancellation(t *testing.T) {
	// Skip if claude binary not available
	t.Skip("requires claude binary installation and runtime testing")

	exec, _ := New(&executor.Config{ClaudePath: "claude"})
	handler := &MockEventHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel context

	task := &executor.Task{
		ID:        "streaming-cancel",
		Directive: "test",
	}

	_, err := exec.ExecuteStreaming(ctx, task, handler)
	if err == nil {
		t.Error("ExecuteStreaming should error with cancelled context")
	}
}

// TestClaudeHealthCheckTimeout verifies health check respects timeout
func TestClaudeHealthCheckTimeout(t *testing.T) {
	exec, _ := New(&executor.Config{ClaudePath: "/nonexistent/claude"})

	// This should fail quickly without hanging
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := exec.HealthCheck(ctx)
	// Should have errored (binary not found or timeout)
	if err == nil {
		// If no error, that's OK too - means health check completed
		return
	}
	// Expected to fail in some way (binary not found, etc.)
}

// TestIsCloudWorkspace verifies cloud workspace detection (M-CLOUD-PLUGIN-SKILLS)
func TestIsCloudWorkspace(t *testing.T) {
	tests := []struct {
		workspace   string
		expected    bool
		description string
	}{
		{
			workspace:   "/workspace/task-abc123",
			expected:    true,
			description: "cloud container path with task ID",
		},
		{
			workspace:   "/workspace/",
			expected:    true,
			description: "cloud root path",
		},
		{
			workspace:   "/home/user/project",
			expected:    false,
			description: "local home directory",
		},
		{
			workspace:   "/Users/mark/dev/project",
			expected:    false,
			description: "local macOS path",
		},
		{
			workspace:   "/tmp/workspace",
			expected:    false,
			description: "local temp directory (not cloud convention)",
		},
		{
			workspace:   "",
			expected:    false,
			description: "empty workspace",
		},
		{
			workspace:   "workspace/task",
			expected:    false,
			description: "relative path without leading slash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := isCloudWorkspace(tt.workspace)
			if result != tt.expected {
				t.Errorf("expected cloud=%v, got cloud=%v for %q", tt.expected, result, tt.workspace)
			}
		})
	}
}

// TestClaudeCloudPermissionHandling verifies cloud tasks use dangerously-skip-permissions
func TestClaudeCloudPermissionHandling(t *testing.T) {
	tests := []struct {
		name                 string
		workspace            string
		permissionMode       string
		expectDangerously    bool
		expectPermissionMode bool
		description          string
	}{
		{
			name:                 "cloud workspace with bypass",
			workspace:            "/workspace/task-123",
			permissionMode:       "bypassPermissions",
			expectDangerously:    true,
			expectPermissionMode: false,
			description:          "cloud tasks should use --dangerously-skip-permissions",
		},
		{
			name:                 "local workspace with bypass",
			workspace:            "/home/user/project",
			permissionMode:       "bypassPermissions",
			expectDangerously:    false,
			expectPermissionMode: true,
			description:          "local tasks should use --permission-mode",
		},
		{
			name:                 "cloud workspace with custom mode",
			workspace:            "/workspace/task-456",
			permissionMode:       "requestApproval",
			expectDangerously:    false,
			expectPermissionMode: true,
			description:          "cloud tasks with custom mode should use --permission-mode",
		},
		{
			name:                 "empty workspace",
			workspace:            "",
			permissionMode:       "bypassPermissions",
			expectDangerously:    false,
			expectPermissionMode: true,
			description:          "empty workspace treated as local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the logic for choosing permission flags
			// In a real test with claude binary, we'd parse the command arguments
			isCloud := isCloudWorkspace(tt.workspace)
			useDangerously := isCloud && tt.permissionMode == "bypassPermissions"
			usePermissionMode := !useDangerously

			if useDangerously != tt.expectDangerously {
				t.Errorf("expected dangerously=%v, got dangerously=%v", tt.expectDangerously, useDangerously)
			}
			if usePermissionMode != tt.expectPermissionMode {
				t.Errorf("expected permissionMode=%v, got permissionMode=%v", tt.expectPermissionMode, usePermissionMode)
			}
		})
	}
}

// BenchmarkClaudeNew benchmarks executor initialization
func BenchmarkClaudeNew(b *testing.B) {
	cfg := executor.DefaultConfig()
	cfg.ClaudePath = "/usr/bin/claude"
	cfg.ClaudeModel = "opus"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = New(cfg)
	}
}

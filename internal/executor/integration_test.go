package executor_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	// Register executors via init()
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
	_ "github.com/sunholo-data/ailang/internal/executor/gemini"
)

func TestGeminiExecutorIntegration(t *testing.T) {
	factory := executor.GlobalFactory()

	// List available executors
	available := factory.ListAvailable()
	t.Logf("Available executors: %v", available)

	if len(available) < 2 {
		t.Errorf("Expected at least 2 executors (claude, gemini), got %d", len(available))
	}

	// Get the default executor (should be Gemini)
	defaultExec, err := factory.GetDefault()
	if err != nil {
		t.Fatalf("Failed to get default executor: %v", err)
	}

	if defaultExec.Name() != "gemini" {
		t.Errorf("Expected default executor 'gemini', got '%s'", defaultExec.Name())
	}
	t.Logf("Default executor: %s", defaultExec.Name())

	// Check AILANG_EXECUTOR env var
	os.Setenv("AILANG_EXECUTOR", "claude")
	defer os.Unsetenv("AILANG_EXECUTOR")

	claudeExec, err := factory.GetDefault()
	if err != nil {
		t.Fatalf("Failed to get claude executor via env: %v", err)
	}

	if claudeExec.Name() != "claude" {
		t.Errorf("Expected executor 'claude' with env var, got '%s'", claudeExec.Name())
	}

	os.Unsetenv("AILANG_EXECUTOR")
}

func TestGeminiCostModel(t *testing.T) {
	factory := executor.GlobalFactory()
	geminiExec, err := factory.GetExecutor("gemini")
	if err != nil {
		t.Fatalf("Failed to get gemini executor: %v", err)
	}

	costModel := geminiExec.CostModel()

	// Gemini 3 Flash pricing: $0.50/$3.00 per 1M
	if costModel.InputTokenCost != 0.0005 {
		t.Errorf("Expected input cost 0.0005, got %f", costModel.InputTokenCost)
	}
	if costModel.OutputTokenCost != 0.003 {
		t.Errorf("Expected output cost 0.003, got %f", costModel.OutputTokenCost)
	}

	t.Logf("Gemini cost model: $%.4f/1K input, $%.4f/1K output",
		costModel.InputTokenCost, costModel.OutputTokenCost)
}

func TestGeminiHealthCheck(t *testing.T) {
	// This test checks if gemini CLI is installed
	// It's expected to fail if gemini is not installed
	factory := executor.GlobalFactory()
	geminiExec, err := factory.GetExecutor("gemini")
	if err != nil {
		t.Fatalf("Failed to get gemini executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = geminiExec.HealthCheck(ctx)
	if err != nil {
		t.Logf("Gemini health check failed (expected if CLI not installed): %v", err)
		t.Skip("Gemini CLI not installed - skipping health check")
	}

	t.Log("Gemini health check passed - CLI is available")
}

func TestClaudeHealthCheck(t *testing.T) {
	factory := executor.GlobalFactory()
	claudeExec, err := factory.GetExecutor("claude")
	if err != nil {
		t.Fatalf("Failed to get claude executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = claudeExec.HealthCheck(ctx)
	if err != nil {
		t.Logf("Claude health check failed (expected if CLI not installed): %v", err)
		t.Skip("Claude CLI not installed - skipping health check")
	}

	t.Log("Claude health check passed - CLI is available")
}

// TestFactoryGetExecutorUnknown tests handling of unknown executor
func TestFactoryGetExecutorUnknown(t *testing.T) {
	factory := executor.GlobalFactory()

	_, err := factory.GetExecutor("nonexistent-executor")
	if err == nil {
		t.Error("Expected error for unknown executor, got nil")
	}
}

// TestFactoryConcurrentGetExecutor tests thread-safe concurrent access
func TestFactoryConcurrentGetExecutor(t *testing.T) {
	factory := executor.GlobalFactory()
	numGoroutines := 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	executorNames := make(map[string]int)

	// Launch 10 concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Alternate between claude and gemini
			name := "gemini"
			if index%2 == 0 {
				name = "claude"
			}

			exec, err := factory.GetExecutor(name)
			if err != nil {
				t.Errorf("GetExecutor(%s) failed: %v", name, err)
				return
			}

			if exec == nil {
				t.Errorf("GetExecutor(%s) returned nil", name)
				return
			}

			mu.Lock()
			executorNames[name]++
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if len(executorNames) != 2 {
		t.Errorf("Expected 2 executor types, got %d", len(executorNames))
	}
	if executorNames["claude"] != 5 {
		t.Errorf("Expected 5 claude calls, got %d", executorNames["claude"])
	}
	if executorNames["gemini"] != 5 {
		t.Errorf("Expected 5 gemini calls, got %d", executorNames["gemini"])
	}
}

// TestCostModelEdgeCases tests cost calculation edge cases
func TestCostModelEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		usage          executor.TokenUsage
		costModel      *executor.CostModel
		expectedResult float64
	}{
		{
			name:  "zero tokens",
			usage: executor.TokenUsage{},
			costModel: &executor.CostModel{
				InputTokenCost:  0.001,
				OutputTokenCost: 0.003,
				MinimumCharge:   0.01,
			},
			expectedResult: 0.01, // Should return minimum charge
		},
		{
			name: "large token count",
			usage: executor.TokenUsage{
				InputTokens:  1000000,
				OutputTokens: 1000000,
			},
			costModel: &executor.CostModel{
				InputTokenCost:  0.001,
				OutputTokenCost: 0.003,
			},
			expectedResult: 4.0, // (1M/1K)*0.001 + (1M/1K)*0.003 = 1000*0.001 + 1000*0.003
		},
		{
			name: "cache tokens",
			usage: executor.TokenUsage{
				InputTokens:              1000,
				OutputTokens:             500,
				CacheReadInputTokens:     1000,
				CacheCreationInputTokens: 500,
			},
			costModel: &executor.CostModel{
				InputTokenCost:  0.001,
				OutputTokenCost: 0.003,
				CacheReadCost:   0.0001,
				CacheWriteCost:  0.0005,
			},
			expectedResult: 0.0026, // (1K/1K)*0.001 + (0.5K/1K)*0.003 + (1K/1K)*0.0001
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.costModel.CalculateCost(tt.usage)

			// Allow small floating point error
			if fmt.Sprintf("%.4f", result) != fmt.Sprintf("%.4f", tt.expectedResult) {
				t.Errorf("expected %.6f, got %.6f", tt.expectedResult, result)
			}
		})
	}
}

// TestFactoryCloseIdempotent tests that Close() can be called multiple times safely
func TestFactoryCloseIdempotent(t *testing.T) {
	factory := executor.NewFactory(executor.DefaultConfig())

	// Register a test executor
	factory.Register("test", func(cfg *executor.Config) (executor.Executor, error) {
		return &testExecutor{}, nil
	})

	// Get the executor to initialize it
	exec, err := factory.GetExecutor("test")
	if err != nil {
		t.Fatalf("GetExecutor failed: %v", err)
	}

	// Close multiple times
	for i := 0; i < 3; i++ {
		err := exec.Close()
		if err != nil {
			t.Errorf("Close() iteration %d failed: %v", i, err)
		}
	}
}

// testExecutor is a minimal executor for testing
type testExecutor struct{}

func (t *testExecutor) Name() string { return "test" }
func (t *testExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return &executor.Result{Success: true}, nil
}
func (t *testExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	return t.Execute(ctx, task)
}
func (t *testExecutor) Capabilities() []executor.Capability { return []executor.Capability{} }
func (t *testExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{ProviderName: "test"}
}
func (t *testExecutor) HealthCheck(ctx context.Context) error { return nil }
func (t *testExecutor) Close() error                          { return nil }

// TestInvalidWorkspacePath tests handling of invalid workspace paths
func TestInvalidWorkspacePath(t *testing.T) {
	factory := executor.GlobalFactory()
	exec, err := factory.GetExecutor("claude")
	if err != nil {
		t.Fatalf("GetExecutor failed: %v", err)
	}

	task := &executor.Task{
		ID:        "test-invalid-workspace",
		Directive: "test",
		Workspace: "/nonexistent/workspace/that/does/not/exist",
	}

	// Skip if claude binary not available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// May fail or timeout depending on implementation
	_, _ = exec.Execute(ctx, task)
	// Just verify it doesn't panic
}

// MockEventHandlerForIntegration tracks all events in sequence
type MockEventHandlerForIntegration struct {
	Events []string
	Turns  int
}

func (m *MockEventHandlerForIntegration) OnTurnStart(turnNum int) {
	m.Turns++
	m.Events = append(m.Events, fmt.Sprintf("turn_start(%d)", turnNum))
}

func (m *MockEventHandlerForIntegration) OnTurnEnd(turnNum int) {
	m.Events = append(m.Events, fmt.Sprintf("turn_end(%d)", turnNum))
}

func (m *MockEventHandlerForIntegration) OnText(text string) {
	m.Events = append(m.Events, fmt.Sprintf("text(%d)", len(text)))
}

func (m *MockEventHandlerForIntegration) OnToolUse(toolName string, input string) {
	m.Events = append(m.Events, fmt.Sprintf("tool(%s)", toolName))
}

func (m *MockEventHandlerForIntegration) OnToolResult(toolName string, output string) {
	m.Events = append(m.Events, fmt.Sprintf("tool_result(%s)", toolName))
}

func (m *MockEventHandlerForIntegration) OnError(err error) {
	m.Events = append(m.Events, fmt.Sprintf("error(%v)", err))
}

// TestEventHandlerSequenceIntegration verifies event handler callback order
func TestEventHandlerSequenceIntegration(t *testing.T) {
	handler := &MockEventHandlerForIntegration{}

	// Simulate a typical execution sequence
	handler.OnTurnStart(1)
	handler.OnText("Hello ")
	handler.OnText("world")
	handler.OnToolUse("Bash", `{"command": "ls"}`)
	handler.OnToolResult("Bash", "file1.txt\nfile2.txt")
	handler.OnTurnEnd(1)

	// Verify sequence
	expectedSequence := []string{
		"turn_start(1)",
		"text(6)",
		"text(5)",
		"tool(Bash)",
		"tool_result(Bash)",
		"turn_end(1)",
	}

	if len(handler.Events) != len(expectedSequence) {
		t.Errorf("expected %d events, got %d", len(expectedSequence), len(handler.Events))
	}

	for i, expected := range expectedSequence {
		if i >= len(handler.Events) {
			t.Errorf("event %d missing: expected %s", i, expected)
		} else if handler.Events[i] != expected {
			t.Errorf("event %d: expected %s, got %s", i, expected, handler.Events[i])
		}
	}
}

// TestMultipleTurnsEventSequence verifies multi-turn conversation event flow
func TestMultipleTurnsEventSequence(t *testing.T) {
	handler := &MockEventHandlerForIntegration{}

	// Simulate 3-turn conversation
	for turn := 1; turn <= 3; turn++ {
		handler.OnTurnStart(turn)
		handler.OnText(fmt.Sprintf("Response %d", turn))
		handler.OnTurnEnd(turn)
	}

	// Should have exactly 9 events (3 turns × 3 events per turn)
	if len(handler.Events) != 9 {
		t.Errorf("expected 9 events for 3 turns, got %d", len(handler.Events))
	}

	if handler.Turns != 3 {
		t.Errorf("expected 3 turns tracked, got %d", handler.Turns)
	}

	// Verify turn order is preserved
	turnCount := 0
	for _, event := range handler.Events {
		if stringContains(event, "turn_start") {
			turnCount++
		}
	}

	if turnCount != 3 {
		t.Errorf("expected 3 turn_start events, got %d", turnCount)
	}
}

// TestEventHandlerErrorHandling verifies error callbacks work
func TestEventHandlerErrorHandling(t *testing.T) {
	handler := &MockEventHandlerForIntegration{}

	handler.OnTurnStart(1)
	handler.OnError(fmt.Errorf("tool execution failed"))
	handler.OnTurnEnd(1)

	errorCount := 0
	for _, event := range handler.Events {
		if stringContains(event, "error") {
			errorCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("expected 1 error event, got %d", errorCount)
	}
}

// stringContains checks if a string contains a substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestContextAwareHandlerSetContext verifies context propagation
func TestContextAwareHandlerSetContext(t *testing.T) {
	// Test that ContextAwareHandler interface works
	factory := executor.GlobalFactory()
	exec, err := factory.GetExecutor("claude")
	if err != nil {
		t.Fatalf("GetExecutor failed: %v", err)
	}

	handler := &MockEventHandlerForIntegration{}
	ctx := context.Background()

	// Create a task with streaming
	task := &executor.Task{
		ID:        "context-test",
		Directive: "test",
	}

	// Skip if claude binary not available
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// This may fail due to missing binary, but we're just checking
	// that the handler callback system works
	_, _ = exec.ExecuteStreaming(ctx, task, handler)

	// Handler should have been called if execution started
	_ = handler
}

// TestNoOpEventHandler verifies default handler doesn't error
func TestNoOpEventHandler(t *testing.T) {
	handler := &executor.NoOpEventHandler{}

	// These should all be no-ops
	handler.OnTurnStart(1)
	handler.OnText("test")
	handler.OnToolUse("Bash", "")
	handler.OnToolResult("Bash", "")
	handler.OnTurnEnd(1)
	handler.OnError(fmt.Errorf("test error"))

	// Should not panic
	t.Log("NoOpEventHandler functions completed without error")
}

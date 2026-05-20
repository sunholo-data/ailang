package executor

import (
	"context"
	"testing"
)

// MockExecutor for testing
type MockExecutor struct {
	name string
}

func (m *MockExecutor) Name() string { return m.name }
func (m *MockExecutor) Execute(ctx context.Context, task *Task) (*Result, error) {
	return &Result{Success: true, Output: "mock result"}, nil
}
func (m *MockExecutor) ExecuteStreaming(ctx context.Context, task *Task, handler EventHandler) (*Result, error) {
	return m.Execute(ctx, task)
}
func (m *MockExecutor) Capabilities() []Capability { return []Capability{CapStreaming} }
func (m *MockExecutor) CostModel() *CostModel {
	return &CostModel{ProviderName: m.name, InputTokenCost: 0.001}
}
func (m *MockExecutor) HealthCheck(ctx context.Context) error { return nil }
func (m *MockExecutor) Close() error                          { return nil }

func TestFactoryRegisterAndGet(t *testing.T) {
	factory := NewFactory(DefaultConfig())

	// Register a mock executor
	factory.Register("mock", func(cfg *Config) (Executor, error) {
		return &MockExecutor{name: "mock"}, nil
	})

	// Get the executor
	exec, err := factory.GetExecutor("mock")
	if err != nil {
		t.Fatalf("GetExecutor failed: %v", err)
	}

	if exec.Name() != "mock" {
		t.Errorf("expected name 'mock', got '%s'", exec.Name())
	}

	// Getting again should return same instance
	exec2, err := factory.GetExecutor("mock")
	if err != nil {
		t.Fatalf("GetExecutor second call failed: %v", err)
	}

	if exec != exec2 {
		t.Error("expected same executor instance")
	}
}

func TestFactoryUnknownExecutor(t *testing.T) {
	factory := NewFactory(DefaultConfig())

	_, err := factory.GetExecutor("unknown")
	if err == nil {
		t.Error("expected error for unknown executor")
	}
}

func TestFactoryListAvailable(t *testing.T) {
	factory := NewFactory(DefaultConfig())

	factory.Register("exec1", func(cfg *Config) (Executor, error) {
		return &MockExecutor{name: "exec1"}, nil
	})
	factory.Register("exec2", func(cfg *Config) (Executor, error) {
		return &MockExecutor{name: "exec2"}, nil
	})

	available := factory.ListAvailable()
	if len(available) != 2 {
		t.Errorf("expected 2 available, got %d", len(available))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Gemini CLI was retired in v0.22.0 (M-MANAGED-AGENTS); claude is the new default.
	if cfg.DefaultExecutor != "claude" {
		t.Errorf("expected default executor 'claude', got '%s'", cfg.DefaultExecutor)
	}
	if cfg.ClaudeModel != "haiku" {
		t.Errorf("expected claude model 'haiku', got '%s'", cfg.ClaudeModel)
	}
}

func TestCostModelCalculation(t *testing.T) {
	model := &CostModel{
		InputTokenCost:  0.001,  // $1 per 1M
		OutputTokenCost: 0.003,  // $3 per 1M
		CacheReadCost:   0.0001, // $0.1 per 1M
	}

	usage := TokenUsage{
		InputTokens:          1000,
		OutputTokens:         500,
		CacheReadInputTokens: 2000,
	}

	cost := model.CalculateCost(usage)

	// 1K input * $0.001 = $0.001
	// 0.5K output * $0.003 = $0.0015
	// 2K cache * $0.0001 = $0.0002
	// Total = $0.0027
	expected := 0.0027
	if cost != expected {
		t.Errorf("expected cost %.4f, got %.4f", expected, cost)
	}
}

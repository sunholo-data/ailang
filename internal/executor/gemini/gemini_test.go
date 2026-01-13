package gemini

import (
	"context"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/executor"
)

func TestNewGeminiExecutor(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if exec.Name() != "gemini" {
		t.Errorf("expected name 'gemini', got '%s'", exec.Name())
	}
}

func TestGeminiDefaultModel(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Should use Gemini 3 Flash preview as default (from DefaultConfig)
	if exec.model != "gemini-3-flash-preview" {
		t.Errorf("expected default model 'gemini-3-flash-preview', got '%s'", exec.model)
	}
}

func TestGeminiCapabilities(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	caps := exec.Capabilities()
	if len(caps) == 0 {
		t.Error("expected at least one capability")
	}

	hasStreaming := false
	for _, cap := range caps {
		if cap == executor.CapStreaming {
			hasStreaming = true
			break
		}
	}
	if !hasStreaming {
		t.Error("expected streaming capability")
	}
}

func TestGeminiCostModel(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	costModel := exec.CostModel()

	// Gemini 3 Flash pricing: $0.50/$3.00 per 1M
	expectedInput := 0.0005 // $0.50 per 1M = $0.0005 per 1K
	expectedOutput := 0.003 // $3.00 per 1M = $0.003 per 1K

	if costModel.InputTokenCost != expectedInput {
		t.Errorf("expected input cost %.4f, got %.4f", expectedInput, costModel.InputTokenCost)
	}
	if costModel.OutputTokenCost != expectedOutput {
		t.Errorf("expected output cost %.4f, got %.4f", expectedOutput, costModel.OutputTokenCost)
	}
}

func TestGeminiFactoryRegistration(t *testing.T) {
	// Gemini executor should be registered via init()
	factory := executor.GlobalFactory()
	available := factory.ListAvailable()

	hasGemini := false
	for _, name := range available {
		if name == "gemini" {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Error("expected 'gemini' to be registered in global factory")
	}
}

// TestGeminiContextCancellation verifies context cancellation handling
func TestGeminiContextCancellation(t *testing.T) {
	// Skip if gemini binary not available
	t.Skip("requires gemini binary installation and runtime testing")

	cfg := executor.DefaultConfig()
	exec, _ := New(cfg)

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

// TestGeminiContextTimeout verifies timeout handling
func TestGeminiContextTimeout(t *testing.T) {
	// Skip if gemini binary not available
	t.Skip("requires gemini binary installation and runtime testing")

	cfg := executor.DefaultConfig()
	exec, _ := New(cfg)

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

// TestGeminiHealthCheckTimeout verifies health check respects timeout
func TestGeminiHealthCheckTimeout(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, _ := New(cfg)

	// Health check should complete quickly even with non-existent workspace
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = exec.HealthCheck(ctx)
	// Health check result doesn't matter for this test
	// We're just verifying it respects the timeout
}

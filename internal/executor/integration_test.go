package executor_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/executor"
	// Register executors via init()
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
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

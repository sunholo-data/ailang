package eval_harness

import "testing"

func TestDefaultAgentConfig(t *testing.T) {
	config := DefaultAgentConfig()

	// MaxConcurrent removed 2026-05-23 — was dead code that caused user confusion.
	// The real dispatch semaphore is the -parallel flag, handled in eval_parallel.go.
	if config.RequestsPerSecond != 1 {
		t.Errorf("Expected RequestsPerSecond=1, got %d", config.RequestsPerSecond)
	}
	if config.TimeoutSeconds != 300 {
		t.Errorf("Expected TimeoutSeconds=300, got %d", config.TimeoutSeconds)
	}
	if len(config.AllowedTools) != 5 {
		t.Errorf("Expected 5 allowed tools, got %d", len(config.AllowedTools))
	}
}

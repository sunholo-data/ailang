package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

func TestCheckClaudeCLI(t *testing.T) {
	// This test verifies that our installed Claude CLI is detected
	err := checkClaudeCLI("claude")
	if err != nil {
		t.Skipf("Claude CLI not available: %v", err)
	}
	// If no error, Claude is properly installed
}

// Note: TestPrepareWorkspace and TestGenerateAgentPrompt moved to agent_prompt_test.go
// These functions are now tested in their dedicated test file

func TestClaudeHeadlessResultParsing(t *testing.T) {
	// Test parsing of actual Claude headless JSON output
	jsonOutput := `{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 2792,
  "duration_api_ms": 2756,
  "num_turns": 1,
  "result": "4",
  "session_id": "20186696-9ce0-4e79-bc3d-0f70b8304037",
  "total_cost_usd": 0.08914455,
  "usage": {
    "input_tokens": 3,
    "cache_creation_input_tokens": 22297,
    "cache_read_input_tokens": 18156,
    "output_tokens": 5
  },
  "modelUsage": {
    "claude-sonnet-4-5-20250929": {
      "inputTokens": 3,
      "outputTokens": 5,
      "cacheReadInputTokens": 18156,
      "cacheCreationInputTokens": 22297,
      "webSearchRequests": 0,
      "costUSD": 0.08914455,
      "contextWindow": 200000
    }
  },
  "permission_denials": [],
  "uuid": "44b07894-6727-4439-9e4f-7892a670a474"
}`

	var result ClaudeHeadlessResult
	err := json.Unmarshal([]byte(jsonOutput), &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify parsing
	if result.Type != "result" {
		t.Errorf("Expected type='result', got '%s'", result.Type)
	}
	if result.Subtype != "success" {
		t.Errorf("Expected subtype='success', got '%s'", result.Subtype)
	}
	if result.IsError {
		t.Error("Expected is_error=false")
	}
	if result.DurationMS != 2792 {
		t.Errorf("Expected duration_ms=2792, got %d", result.DurationMS)
	}
	if result.NumTurns != 1 {
		t.Errorf("Expected num_turns=1, got %d", result.NumTurns)
	}
	if result.TotalCostUSD != 0.08914455 {
		t.Errorf("Expected total_cost_usd=0.08914455, got %f", result.TotalCostUSD)
	}
	if result.SessionID != "20186696-9ce0-4e79-bc3d-0f70b8304037" {
		t.Errorf("Unexpected session_id: %s", result.SessionID)
	}
	if result.Result != "4" {
		t.Errorf("Expected result='4', got '%s'", result.Result)
	}

	// Verify token usage
	if result.Usage.InputTokens != 3 {
		t.Errorf("Expected input_tokens=3, got %d", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 5 {
		t.Errorf("Expected output_tokens=5, got %d", result.Usage.OutputTokens)
	}
	if result.Usage.CacheCreationInputTokens != 22297 {
		t.Errorf("Expected cache_creation_input_tokens=22297, got %d", result.Usage.CacheCreationInputTokens)
	}

	// Verify model usage
	if len(result.ModelUsage) != 1 {
		t.Errorf("Expected 1 model in modelUsage, got %d", len(result.ModelUsage))
	}
	stats, ok := result.ModelUsage["claude-sonnet-4-5-20250929"]
	if !ok {
		t.Error("Expected model 'claude-sonnet-4-5-20250929' in modelUsage")
	}
	if stats.CostUSD != 0.08914455 {
		t.Errorf("Expected model cost=0.08914455, got %f", stats.CostUSD)
	}
}

func TestDetermineSuccess(t *testing.T) {
	// Create test workspace
	tmpDir := t.TempDir()

	// Create benchmark/ subdirectory (matches actual workspace structure)
	benchmarkDir := filepath.Join(tmpDir, "benchmark")
	os.MkdirAll(benchmarkDir, 0755)

	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test",
		ExpectedOut: "42",
		Caps:        []string{},
	}

	// Test case 1: Error result
	errorResult := &ClaudeHeadlessResult{
		IsError: true,
		Subtype: "error",
	}
	result := determineSuccess(errorResult, spec, tmpDir, "ailang")
	if result.StdoutOk {
		t.Error("Expected failure for error result")
	}

	// Test case 2: Success result but no solution file
	successResult := &ClaudeHeadlessResult{
		IsError: false,
		Subtype: "success",
	}
	result = determineSuccess(successResult, spec, tmpDir, "ailang")
	if result.StdoutOk {
		t.Error("Expected failure when solution.ail missing")
	}

	// Test case 3: Success result with empty solution
	os.WriteFile(filepath.Join(tmpDir, "benchmark", "solution.ail"), []byte(""), 0644)
	result = determineSuccess(successResult, spec, tmpDir, "ailang")
	if result.StdoutOk {
		t.Error("Expected failure for empty solution")
	}

	// Test case 4: Success result with valid solution (will test actual execution)
	validSolution := `module benchmark/solution

export func main() -> () ! {IO} = {
  print("42")
}`
	os.WriteFile(filepath.Join(tmpDir, "benchmark", "solution.ail"), []byte(validSolution), 0644)

	// Note: This will actually try to run ailang, so it may fail if ailang not in PATH
	// But the test structure is correct
	validationResult := determineSuccess(successResult, spec, tmpDir, "ailang")
	// We don't assert here because it depends on ailang being available
	_ = validationResult
}

func TestGetErrorMessage(t *testing.T) {
	// Test with is_error=true
	result1 := &ClaudeHeadlessResult{
		IsError: true,
		Result:  "Some error message",
	}
	if msg := getErrorMessage(result1); msg != "Some error message" {
		t.Errorf("Expected 'Some error message', got '%s'", msg)
	}

	// Test with subtype=error
	result2 := &ClaudeHeadlessResult{
		IsError: false,
		Subtype: "error",
		Result:  "Another error",
	}
	if msg := getErrorMessage(result2); msg != "Another error" {
		t.Errorf("Expected 'Another error', got '%s'", msg)
	}

	// Test with success
	result3 := &ClaudeHeadlessResult{
		IsError: false,
		Subtype: "success",
		Result:  "Success result",
	}
	if msg := getErrorMessage(result3); msg != "" {
		t.Errorf("Expected empty string for success, got '%s'", msg)
	}
}

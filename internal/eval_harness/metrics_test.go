package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name      string
		compileOk bool
		runtimeOk bool
		stdoutOk  bool
		expected  string
	}{
		{"all ok", true, true, true, ErrorCategoryNone},
		{"compile failed", false, true, true, ErrorCategoryCompile},
		{"runtime failed", true, false, true, ErrorCategoryRuntime},
		{"output wrong", true, true, false, ErrorCategoryLogic},
		{"compile and runtime failed", false, false, false, ErrorCategoryCompile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.compileOk, tt.runtimeOk, tt.stdoutOk)
			if result != tt.expected {
				t.Errorf("CategorizeError() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestMetricsLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewMetricsLogger(tmpDir)

	metrics := &RunMetrics{
		ID:            "test",
		Lang:          "python",
		Model:         "gpt-4",
		Seed:          42,
		InputTokens:   50,
		OutputTokens:  50,
		TotalTokens:   100,
		CostUSD:       0.003,
		CompileOk:     true,
		RuntimeOk:     true,
		StdoutOk:      true,
		DurationMs:    150,
		CompileMs:     50,
		ExecuteMs:     100,
		ErrorCategory: ErrorCategoryNone,
		Timestamp:     time.Now(),
	}

	// Log metrics
	if err := logger.Log(metrics); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	// Read and parse file
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var loaded RunMetrics
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if loaded.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", loaded.ID)
	}

	if loaded.TotalTokens != 100 {
		t.Errorf("Expected total tokens 100, got %d", loaded.TotalTokens)
	}

	if loaded.InputTokens != 50 {
		t.Errorf("Expected input tokens 50, got %d", loaded.InputTokens)
	}

	if loaded.OutputTokens != 50 {
		t.Errorf("Expected output tokens 50, got %d", loaded.OutputTokens)
	}
}

// TestRunMetrics_AgentToolCalls verifies the agent_tool_calls field
// (M-MOTOKO-OBS-TOOLCALLS) round-trips through JSON for agent rows and is
// omitted for standard rows (omitempty). Tool-call count is the signal that
// distinguishes agentic runs (>0) from the degenerate "0 tool calls" failure.
func TestRunMetrics_AgentToolCalls(t *testing.T) {
	// Agent row with tool calls: field is serialized and reloads intact.
	agent := &RunMetrics{ID: "agent_run", EvalMode: EvalModeAgent, AgentTurns: 4, AgentToolCalls: 3}
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"agent_tool_calls":3`) {
		t.Errorf("agent_tool_calls not serialized; got: %s", data)
	}
	var reloaded RunMetrics
	if err := json.Unmarshal(data, &reloaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if reloaded.AgentToolCalls != 3 {
		t.Errorf("AgentToolCalls = %d, want 3", reloaded.AgentToolCalls)
	}

	// Standard row (no tool calls): omitempty drops the field entirely.
	std := &RunMetrics{ID: "std_run", EvalMode: EvalModeStandard}
	sdata, _ := json.Marshal(std)
	if strings.Contains(string(sdata), "agent_tool_calls") {
		t.Errorf("agent_tool_calls should be omitted for standard rows; got: %s", sdata)
	}
}

func TestNewRunMetrics(t *testing.T) {
	metrics := NewRunMetrics("test", "python", "gpt-4", 42)

	if metrics.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", metrics.ID)
	}

	if metrics.Lang != "python" {
		t.Errorf("Expected lang 'python', got '%s'", metrics.Lang)
	}

	if metrics.Seed != 42 {
		t.Errorf("Expected seed 42, got %d", metrics.Seed)
	}

	// Timestamp should be recent (within 1 second)
	if time.Since(metrics.Timestamp) > time.Second {
		t.Error("Timestamp is not recent")
	}
}

// M-BRAIN-MICRORAG: NewRunMetrics must auto-populate MicroragState from env
// so eval result files always record the effective state.
func TestNewRunMetrics_MicroragState(t *testing.T) {
	t.Setenv("AILANG_MICRORAG_ENABLED", "0")
	m := NewRunMetrics("t", "ailang", "m", 1)
	if m.MicroragState != "off" {
		t.Errorf("env=0 → state should be 'off', got %q", m.MicroragState)
	}

	t.Setenv("AILANG_MICRORAG_ENABLED", "1")
	m = NewRunMetrics("t", "ailang", "m", 1)
	if m.MicroragState != "on" {
		t.Errorf("env=1 → state should be 'on', got %q", m.MicroragState)
	}
}

// M-BRAIN-MICRORAG: MetricsLogger.Log() must backstop MicroragState for
// direct struct-literal call sites (agent path) that bypass NewRunMetrics.
func TestMetricsLogger_BackstopsMicroragState(t *testing.T) {
	t.Setenv("AILANG_MICRORAG_ENABLED", "1")
	logger := NewMetricsLogger(t.TempDir())
	m := &RunMetrics{ID: "t", Lang: "ailang", Model: "m", EvalMode: EvalModeAgent}
	if err := logger.Log(m); err != nil {
		t.Fatalf("log failed: %v", err)
	}
	if m.MicroragState != "on" {
		t.Errorf("logger should backstop MicroragState from env; got %q", m.MicroragState)
	}

	// Caller-set value must win over backstop.
	t.Setenv("AILANG_MICRORAG_ENABLED", "0")
	m2 := &RunMetrics{ID: "t2", Lang: "ailang", Model: "m", EvalMode: EvalModeAgent, MicroragState: "on"}
	if err := logger.Log(m2); err != nil {
		t.Fatalf("log failed: %v", err)
	}
	if m2.MicroragState != "on" {
		t.Errorf("explicit caller value must win; got %q", m2.MicroragState)
	}
}

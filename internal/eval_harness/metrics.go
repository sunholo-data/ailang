package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunMetrics captures the results of a single benchmark run
type RunMetrics struct {
	ID            string    `json:"id"`
	Lang          string    `json:"lang"`
	Model         string    `json:"model"`
	Seed          int64     `json:"seed"`
	InputTokens   int       `json:"input_tokens"`  // Prompt tokens (recorded but not primary metric)
	OutputTokens  int       `json:"output_tokens"` // Generated code tokens (PRIMARY METRIC)
	TotalTokens   int       `json:"total_tokens"`  // Total for billing
	CostUSD       float64   `json:"cost_usd"`
	CompileOk     bool      `json:"compile_ok"`
	RuntimeOk     bool      `json:"runtime_ok"`
	StdoutOk      bool      `json:"stdout_ok"`
	DurationMs    int64     `json:"duration_ms"`    // Total time (startup + compile + execution)
	CompileMs     int64     `json:"compile_ms"`     // Time spent in compilation (if separate)
	ExecuteMs     int64     `json:"execute_ms"`     // Time spent in execution (if measurable)
	ErrorCategory string    `json:"error_category"` // compile_error | runtime_error | logic_error | none
	Stderr        string    `json:"stderr,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Code          string    `json:"code,omitempty"` // Generated code (optional, for debugging)
}

// ErrorCategory constants
const (
	ErrorCategoryNone    = "none"
	ErrorCategoryCompile = "compile_error"
	ErrorCategoryRuntime = "runtime_error"
	ErrorCategoryLogic   = "logic_error"
)

// CategorizeError determines the error category based on execution results
func CategorizeError(compileOk, runtimeOk, stdoutOk bool) string {
	switch {
	case !compileOk:
		return ErrorCategoryCompile
	case !runtimeOk:
		return ErrorCategoryRuntime
	case !stdoutOk:
		return ErrorCategoryLogic
	default:
		return ErrorCategoryNone
	}
}

// MetricsLogger handles writing metrics to JSON files
type MetricsLogger struct {
	outputDir string
}

// NewMetricsLogger creates a new metrics logger
func NewMetricsLogger(outputDir string) *MetricsLogger {
	return &MetricsLogger{
		outputDir: outputDir,
	}
}

// Log writes a RunMetrics to a JSON file
func (l *MetricsLogger) Log(m *RunMetrics) error {
	// Ensure output directory exists
	if err := os.MkdirAll(l.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename: <id>_<lang>_<model>_<timestamp>.json
	filename := fmt.Sprintf("%s_%s_%s_%d.json",
		m.ID,
		m.Lang,
		m.Model,
		m.Timestamp.Unix(),
	)
	path := filepath.Join(l.outputDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// CalculateCost estimates the cost in USD based on model and token count
func CalculateCost(model string, tokens int) float64 {
	// Token pricing (as of 2025, per 1K tokens)
	// These are approximate rates and should be updated
	rates := map[string]float64{
		"gpt-4":         0.03,  // $0.03 per 1K tokens (input)
		"gpt-4-turbo":   0.01,  // $0.01 per 1K tokens
		"gpt-3.5-turbo": 0.001, // $0.001 per 1K tokens
		"claude-3":      0.015, // Anthropic pricing
		"claude-2":      0.01,
	}

	rate, ok := rates[model]
	if !ok {
		// Default to GPT-4 pricing if unknown
		rate = 0.03
	}

	return float64(tokens) / 1000.0 * rate
}

// NewRunMetrics creates a new RunMetrics with timestamp and error category
func NewRunMetrics(id, lang, model string, seed int64) *RunMetrics {
	return &RunMetrics{
		ID:        id,
		Lang:      lang,
		Model:     model,
		Seed:      seed,
		Timestamp: time.Now(),
	}
}

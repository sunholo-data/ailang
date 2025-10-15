package eval_analysis

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ValidationResult represents the outcome of validating a fix
type ValidationResult struct {
	BenchmarkID     string
	BaselineVersion string
	BaselineStatus  bool   // Was it passing in baseline?
	NewStatus       bool   // Is it passing now?
	BaselineError   string // Error category in baseline
	NewError        string // Error category now
	Outcome         ValidationOutcome
	Message         string
}

// ValidationOutcome categorizes the validation result
type ValidationOutcome string

const (
	OutcomeFixed        ValidationOutcome = "fixed"         // Was failing, now passing
	OutcomeBroken       ValidationOutcome = "broken"        // Was passing, now failing
	OutcomeStillFailing ValidationOutcome = "still_failing" // Still broken
	OutcomeStillPassing ValidationOutcome = "still_passing" // Still working
)

// ValidateFix runs a specific benchmark and compares it to the baseline
func ValidateFix(benchmarkID, baselineVersion string) (*ValidationResult, error) {
	// Load baseline
	var baseline *Baseline
	var err error

	if baselineVersion == "" {
		// Use latest baseline
		baseline, err = GetLatestBaseline()
		if err != nil {
			return nil, fmt.Errorf("no baseline found: %w", err)
		}
		baselineVersion = baseline.Version
	} else {
		baseline, err = LoadBaselineByVersion(baselineVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to load baseline %s: %w", baselineVersion, err)
		}
	}

	// Find benchmark in baseline
	var baselineResult *BenchmarkResult
	for _, r := range baseline.Results {
		if r.ID == benchmarkID {
			baselineResult = r
			break
		}
	}

	if baselineResult == nil {
		return nil, fmt.Errorf("benchmark %s not found in baseline %s", benchmarkID, baselineVersion)
	}

	// Run benchmark with current code
	newResult, err := runBenchmark(benchmarkID)
	if err != nil {
		return nil, fmt.Errorf("failed to run benchmark: %w", err)
	}

	// Compare results
	result := &ValidationResult{
		BenchmarkID:     benchmarkID,
		BaselineVersion: baselineVersion,
		BaselineStatus:  baselineResult.StdoutOk,
		NewStatus:       newResult.StdoutOk,
		BaselineError:   baselineResult.ErrorCategory,
		NewError:        newResult.ErrorCategory,
	}

	// Determine outcome
	if !result.BaselineStatus && result.NewStatus {
		result.Outcome = OutcomeFixed
		result.Message = "✓ FIX VALIDATED: Benchmark now passing!"
	} else if result.BaselineStatus && !result.NewStatus {
		result.Outcome = OutcomeBroken
		result.Message = "✗ REGRESSION: Benchmark was passing, now failing!"
	} else if !result.BaselineStatus && !result.NewStatus {
		result.Outcome = OutcomeStillFailing
		if result.BaselineError != result.NewError {
			result.Message = fmt.Sprintf("⚠ STILL FAILING: Error changed from %s to %s", result.BaselineError, result.NewError)
		} else {
			result.Message = "⚠ STILL FAILING: No improvement"
		}
	} else {
		result.Outcome = OutcomeStillPassing
		result.Message = "ℹ NO CHANGE: Benchmark still passing"
	}

	return result, nil
}

// runBenchmark executes a single benchmark and returns the result
func runBenchmark(benchmarkID string) (*BenchmarkResult, error) {
	// Create temporary output directory
	tmpDir := fmt.Sprintf("eval_results/validation/%s_%d", benchmarkID, time.Now().Unix())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Run benchmark using ailang eval command
	cmd := exec.Command("bin/ailang", "eval", "--benchmark", benchmarkID, "--output", tmpDir, "--self-repair")
	_ = cmd.Run() // Ignore error - we want to capture benchmark failures in result JSON

	// Find and load the result file
	pattern := filepath.Join(tmpDir, fmt.Sprintf("%s_*.json", benchmarkID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find result file: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no result file generated for benchmark %s", benchmarkID)
	}

	// Load the most recent result
	result, err := LoadResult(matches[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load result: %w", err)
	}

	return result, nil
}

// FormatValidationResult produces a human-readable validation report
func FormatValidationResult(result *ValidationResult, useColor bool) string {
	var output string

	// Header
	output += colorize("═══════════════════════════════════════════════\n", colorCyan, useColor)
	output += colorize(fmt.Sprintf("  Validating Fix: %s\n", result.BenchmarkID), colorBold, useColor)
	output += colorize("═══════════════════════════════════════════════\n", colorCyan, useColor)
	output += "\n"

	// Baseline status
	output += colorize("Baseline Status:\n", colorBold, useColor)
	output += fmt.Sprintf("  Version: %s\n", result.BaselineVersion)
	if result.BaselineStatus {
		output += colorize("  Status:  ✓ Passing\n", colorGreen, useColor)
	} else {
		output += colorize(fmt.Sprintf("  Status:  ✗ Failing (%s)\n", result.BaselineError), colorRed, useColor)
	}
	output += "\n"

	// New status
	output += colorize("Current Status:\n", colorBold, useColor)
	if result.NewStatus {
		output += colorize("  Status:  ✓ Passing\n", colorGreen, useColor)
	} else {
		output += colorize(fmt.Sprintf("  Status:  ✗ Failing (%s)\n", result.NewError), colorRed, useColor)
	}
	output += "\n"

	// Outcome
	output += "═══════════════════════════════════════════════\n"
	switch result.Outcome {
	case OutcomeFixed:
		output += colorize(result.Message+"\n", colorGreen, useColor)
	case OutcomeBroken:
		output += colorize(result.Message+"\n", colorRed, useColor)
	case OutcomeStillFailing:
		output += colorize(result.Message+"\n", colorYellow, useColor)
	case OutcomeStillPassing:
		output += colorize(result.Message+"\n", colorCyan, useColor)
	}
	output += "\n"

	return output
}

package testing

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OutputFormat specifies the format for test output.
type OutputFormat string

const (
	FormatHuman OutputFormat = "human" // Human-readable colored output
	FormatJSON  OutputFormat = "json"  // Machine-readable JSON
)

// Reporter handles formatting and outputting test results.
type Reporter struct {
	format OutputFormat
	writer io.Writer
	colors bool // Enable ANSI colors for human output
}

// NewReporter creates a new test reporter.
func NewReporter(format OutputFormat, writer io.Writer, colors bool) *Reporter {
	return &Reporter{
		format: format,
		writer: writer,
		colors: colors,
	}
}

// Report outputs a test suite result in the configured format.
func (r *Reporter) Report(result *SuiteResult) error {
	switch r.format {
	case FormatJSON:
		return r.reportJSON(result)
	case FormatHuman:
		return r.reportHuman(result)
	default:
		return fmt.Errorf("unknown output format: %s", r.format)
	}
}

// reportJSON outputs results as JSON.
func (r *Reporter) reportJSON(result *SuiteResult) error {
	// Create a JSON-friendly structure
	output := map[string]interface{}{
		"module_path":    result.ModulePath,
		"total_tests":    result.TotalTests,
		"passed_tests":   result.PassedTests,
		"failed_tests":   result.FailedTests,
		"skipped_tests":  result.SkippedTests,
		"total_duration": result.TotalDuration.String(),
		"success":        result.Success(),
		"tests":          r.formatTestsJSON(result.Tests),
		"properties":     r.formatPropertiesJSON(result.Properties),
	}

	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatTestsJSON converts test results to JSON-friendly format.
func (r *Reporter) formatTestsJSON(tests []TestResult) []map[string]interface{} {
	output := make([]map[string]interface{}, len(tests))
	for i, test := range tests {
		output[i] = map[string]interface{}{
			"name":     test.Name,
			"status":   string(test.Status),
			"duration": test.Duration.String(),
			"location": test.Location,
		}
		if test.Error != "" {
			output[i]["error"] = test.Error
		}
	}
	return output
}

// formatPropertiesJSON converts property results to JSON-friendly format.
func (r *Reporter) formatPropertiesJSON(properties []PropertyResult) []map[string]interface{} {
	output := make([]map[string]interface{}, len(properties))
	for i, prop := range properties {
		output[i] = map[string]interface{}{
			"name":      prop.Name,
			"status":    string(prop.Status),
			"duration":  prop.Duration.String(),
			"tests_run": prop.TestsRun,
			"location":  prop.Location,
		}
		if prop.Error != "" {
			output[i]["error"] = prop.Error
		}
		if prop.FailingInput != "" {
			output[i]["failing_input"] = prop.FailingInput
		}
	}
	return output
}

// reportHuman outputs results in human-readable format with optional colors.
func (r *Reporter) reportHuman(result *SuiteResult) error {
	// Header
	fmt.Fprintf(r.writer, "\n%s\n", r.color(colorBold, "Test Results"))
	fmt.Fprintf(r.writer, "Module: %s\n\n", result.ModulePath)

	// Test results
	if len(result.Tests) > 0 {
		fmt.Fprintf(r.writer, "%s\n", r.color(colorBold, "Tests:"))
		for _, test := range result.Tests {
			r.reportTestHuman(test)
		}
		fmt.Fprintln(r.writer)
	}

	// Property results
	if len(result.Properties) > 0 {
		fmt.Fprintf(r.writer, "%s\n", r.color(colorBold, "Properties:"))
		for _, prop := range result.Properties {
			r.reportPropertyHuman(prop)
		}
		fmt.Fprintln(r.writer)
	}

	// Summary
	r.reportSummaryHuman(result)

	return nil
}

// reportTestHuman outputs a single test result in human-readable format.
func (r *Reporter) reportTestHuman(test TestResult) {
	statusIcon := r.statusIcon(test.Status)
	statusColor := r.statusColor(test.Status)

	fmt.Fprintf(r.writer, "  %s %s", r.color(statusColor, statusIcon), test.Name)

	if test.Duration > 0 {
		fmt.Fprintf(r.writer, " (%v)", test.Duration)
	}

	fmt.Fprintln(r.writer)

	// Show error/reason for failed and skipped tests
	if (test.Status == StatusFail || test.Status == StatusSkip) && test.Error != "" {
		errorLines := strings.Split(test.Error, "\n")
		lineColor := colorRed
		if test.Status == StatusSkip {
			lineColor = colorYellow
		}
		for _, line := range errorLines {
			fmt.Fprintf(r.writer, "      %s\n", r.color(lineColor, line))
		}
	}

	// Show location
	if test.Location != "" {
		fmt.Fprintf(r.writer, "      at %s\n", r.color(colorDim, test.Location))
	}
}

// reportPropertyHuman outputs a single property result in human-readable format.
func (r *Reporter) reportPropertyHuman(prop PropertyResult) {
	statusIcon := r.statusIcon(prop.Status)
	statusColor := r.statusColor(prop.Status)

	fmt.Fprintf(r.writer, "  %s %s", r.color(statusColor, statusIcon), prop.Name)

	if prop.TestsRun > 0 {
		fmt.Fprintf(r.writer, " (%d cases, %v)", prop.TestsRun, prop.Duration)
	} else if prop.Duration > 0 {
		fmt.Fprintf(r.writer, " (%v)", prop.Duration)
	}

	fmt.Fprintln(r.writer)

	// Show error details if property failed
	if prop.Status == StatusFail {
		if prop.FailingInput != "" {
			fmt.Fprintf(r.writer, "      %s: %s\n",
				r.color(colorRed, "Failing input"), prop.FailingInput)
		}
		if prop.Error != "" {
			errorLines := strings.Split(prop.Error, "\n")
			for _, line := range errorLines {
				fmt.Fprintf(r.writer, "      %s\n", r.color(colorRed, line))
			}
		}
	}

	// Show location
	if prop.Location != "" {
		fmt.Fprintf(r.writer, "      at %s\n", r.color(colorDim, prop.Location))
	}
}

// reportSummaryHuman outputs the summary section in human-readable format.
func (r *Reporter) reportSummaryHuman(result *SuiteResult) {
	fmt.Fprintf(r.writer, "%s\n", strings.Repeat("─", 50))

	switch {
	case result.AllSkipped():
		// run==0, skipped>0: nothing actually executed — this is a hard failure
		fmt.Fprintf(r.writer, "%s %s\n",
			r.color(colorYellow, "⚠"),
			r.color(colorBold, fmt.Sprintf("NO TESTS RAN (%d skipped)", result.SkippedTests)))
	case result.Success():
		// ran>0 && failed==0 (may include skips)
		fmt.Fprintf(r.writer, "%s %s\n",
			r.color(colorGreen, "✓"),
			r.color(colorBold, "All tests passed!"))
	default:
		fmt.Fprintf(r.writer, "%s %s\n",
			r.color(colorRed, "✗"),
			r.color(colorBold, "Some tests failed"))
	}

	fmt.Fprintf(r.writer, "\n%s\n", result.Summary())

	// Detailed breakdown
	fmt.Fprintf(r.writer, "  %s %d\n", r.color(colorGreen, "✓ Passed:"), result.PassedTests)
	fmt.Fprintf(r.writer, "  %s %d\n", r.color(colorRed, "✗ Failed:"), result.FailedTests)
	if result.SkippedTests > 0 {
		fmt.Fprintf(r.writer, "  %s %d\n", r.color(colorYellow, "⊘ Skipped:"), result.SkippedTests)
	}
	fmt.Fprintln(r.writer)
}

// statusIcon returns an icon for a test status.
func (r *Reporter) statusIcon(status TestStatus) string {
	switch status {
	case StatusPass:
		return "✓"
	case StatusFail:
		return "✗"
	case StatusSkip:
		return "⊘"
	default:
		return "?"
	}
}

// statusColor returns a color for a test status.
func (r *Reporter) statusColor(status TestStatus) ansiColor {
	switch status {
	case StatusPass:
		return colorGreen
	case StatusFail:
		return colorRed
	case StatusSkip:
		return colorYellow
	default:
		return colorReset
	}
}

// ANSI color codes
type ansiColor string

const (
	colorReset  ansiColor = "\033[0m"
	colorRed    ansiColor = "\033[31m"
	colorGreen  ansiColor = "\033[32m"
	colorYellow ansiColor = "\033[33m"
	colorBold   ansiColor = "\033[1m"
	colorDim    ansiColor = "\033[2m"
)

// color applies ANSI color codes if colors are enabled.
func (r *Reporter) color(c ansiColor, text string) string {
	if !r.colors {
		return text
	}
	return string(c) + text + string(colorReset)
}

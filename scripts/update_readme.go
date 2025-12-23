//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/scripts/internal/reporttypes"
)

func main() {
	// Read the verification report
	reportFile, err := os.Open("examples_report.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading examples report: %v\n", err)
		os.Exit(1)
	}
	defer reportFile.Close()

	var report reporttypes.VerificationReport
	if err := json.NewDecoder(reportFile).Decode(&report); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	// Generate compact status (just badge + summary with link)
	statusContent := generateCompactStatus(report)

	// Read current README
	readmeContent, err := os.ReadFile("README.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading README: %v\n", err)
		os.Exit(1)
	}

	// Update README with new status
	updatedContent := updateReadmeStatus(string(readmeContent), statusContent)

	// Write updated README
	if err := os.WriteFile("README.md", []byte(updatedContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing README: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("README updated successfully")
}

// generateCompactStatus creates a compact status for README (badge + summary + link)
func generateCompactStatus(report reporttypes.VerificationReport) string {
	var sb strings.Builder

	// Add examples badge
	sb.WriteString("![Examples](https://img.shields.io/badge/examples-")
	if report.Failed == 0 {
		sb.WriteString(fmt.Sprintf("%d%%20passing-brightgreen", report.Passed))
	} else {
		sb.WriteString(fmt.Sprintf("%d%%20passing%%20%d%%20failing-", report.Passed, report.Failed))
		// Choose color based on percentage
		percentage := float64(report.Passed) / float64(report.TotalExamples) * 100
		if percentage >= 80 {
			sb.WriteString("green")
		} else if percentage >= 60 {
			sb.WriteString("orange")
		} else {
			sb.WriteString("red")
		}
	}
	sb.WriteString(".svg)\n")

	// Add license badge on same line
	sb.WriteString("![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)\n\n")

	// Add summary with link to full status
	percentage := float64(report.Passed) / float64(report.TotalExamples) * 100
	sb.WriteString(fmt.Sprintf("**%d/%d examples passing (%.0f%%)** | [Full status](https://ailang.sunholo.com/docs/examples)\n",
		report.Passed, report.TotalExamples, percentage))

	return sb.String()
}

func updateReadmeStatus(content, statusContent string) string {
	// Look for markers in README
	startMarker := "<!-- EXAMPLES_STATUS_START -->"
	endMarker := "<!-- EXAMPLES_STATUS_END -->"

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)

	if startIdx == -1 || endIdx == -1 {
		// Markers not found, add them after the main title
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				// Found main title, insert after it
				newLines := append(lines[:i+1],
					"",
					startMarker,
					statusContent,
					endMarker,
				)
				newLines = append(newLines, lines[i+1:]...)
				return strings.Join(newLines, "\n")
			}
		}
		// No main title found, prepend
		return startMarker + "\n" + statusContent + "\n" + endMarker + "\n\n" + content
	}

	// Replace content between markers
	before := content[:startIdx+len(startMarker)]
	after := content[endIdx:]
	return before + "\n" + statusContent + after
}

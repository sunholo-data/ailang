// Package display provides shared text formatting utilities for CLI and server.
// This package consolidates display logic to ensure consistency across all interfaces.
package display

import (
	"strings"
)

// WrapText wraps text at the specified width, preserving existing newlines.
// If a single word is longer than width, it won't be broken.
func WrapText(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		// Handle each line
		if len(line) <= width {
			result.WriteString(line)
			continue
		}
		// Wrap long lines
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine + "\n")
				currentLine = word
			}
		}
		if currentLine != "" {
			result.WriteString(currentLine)
		}
	}
	return result.String()
}

// Truncate truncates a string to maxLen characters, adding "..." if truncated.
// If maxLen is 0 or negative, returns the original string.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// TruncateID truncates an ID string (like UUIDs) to a short form.
// Default length is 12 characters if not specified.
func TruncateID(id string, length ...int) string {
	maxLen := 12
	if len(length) > 0 && length[0] > 0 {
		maxLen = length[0]
	}
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen] + "..."
}

// TruncateFirstLine returns the first line of text, truncated to maxLen.
// Useful for prompts and multi-line strings.
func TruncateFirstLine(text string, maxLen int) string {
	lines := strings.Split(text, "\n")
	firstLine := strings.TrimSpace(lines[0])
	return Truncate(firstLine, maxLen)
}

// TruncateOutput truncates output text and trims whitespace.
// Convenience function for tool outputs and results.
func TruncateOutput(output string, maxLen int) string {
	output = strings.TrimSpace(output)
	return Truncate(output, maxLen)
}

// WordWrapIndent wraps text at width and adds a prefix indent to each line.
func WordWrapIndent(text string, width int, indent string) string {
	wrapped := WrapText(text, width-len(indent))
	lines := strings.Split(wrapped, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(indent)
		result.WriteString(line)
	}
	return result.String()
}

// SanitizeNewlines replaces newlines with spaces for single-line display.
func SanitizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " ")
}

// TruncateMiddle truncates a string by removing the middle portion.
// Useful for file paths: "/Users/mark/.../file.go"
func TruncateMiddle(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 5 {
		return "..."
	}
	// Keep start and end portions
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}

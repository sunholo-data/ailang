package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// getDefaultStateDir returns the default state directory for agent protocol
func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ailang/state"
	}
	return filepath.Join(home, ".ailang", "state")
}

// getStatusColor returns a colored string for agent status
func getStatusColor(status string) string {
	switch status {
	case "active":
		return green(status)
	case "paused":
		return yellow(status)
	case "error":
		return red(status)
	default:
		return status
	}
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs ago", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm ago", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh ago", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd ago", d.Hours()/24)
	}
}

// indentText indents every line of text by the specified number of spaces
func indentText(text string, spaces int) string {
	lines := strings.Split(text, "\n")
	indent := strings.Repeat(" ", spaces)
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// pluralize returns "s" if count != 1, empty string otherwise
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

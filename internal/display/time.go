package display

import (
	"fmt"
	"time"
)

// FormatAge formats a time as a human-readable relative age string.
// Examples: "just now", "5m ago", "2h ago", "3d ago", "Jan 15"
func FormatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	} else if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	// For older dates, show month and day
	return t.Format("Jan 2")
}

// FormatDuration formats a duration in milliseconds for display.
// Returns "-" for zero, "123ms" for <1s, "1.5s" for >=1s.
func FormatDuration(ms float64) string {
	if ms == 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	if ms < 60000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	// For longer durations, show minutes
	return fmt.Sprintf("%.1fm", ms/60000)
}

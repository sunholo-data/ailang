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

// FormatDurationFromDuration formats a time.Duration for display.
func FormatDurationFromDuration(d time.Duration) string {
	return FormatDuration(float64(d.Milliseconds()))
}

// FormatTimestamp formats a timestamp for log-style display.
// Examples: "15:04:05" for today, "Jan 2 15:04" for this year, "2024-01-15 15:04" otherwise.
func FormatTimestamp(t time.Time) string {
	now := time.Now()

	// Same day: just time
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04:05")
	}

	// Same year: month, day, time
	if t.Year() == now.Year() {
		return t.Format("Jan 2 15:04")
	}

	// Different year: full date
	return t.Format("2006-01-02 15:04")
}

// FormatTimestampShort returns a compact timestamp for tables.
func FormatTimestampShort(t time.Time) string {
	now := time.Now()

	// Same day: just time
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}

	// Same year: month/day
	if t.Year() == now.Year() {
		return t.Format("01/02")
	}

	// Different year
	return t.Format("06/01/02")
}

// FormatTimeRange formats a time range (start to end) concisely.
func FormatTimeRange(start, end time.Time) string {
	if end.IsZero() {
		return fmt.Sprintf("%s - ongoing", FormatTimestamp(start))
	}
	duration := end.Sub(start)
	return fmt.Sprintf("%s (%s)", FormatTimestamp(start), FormatDurationFromDuration(duration))
}

// FormatRelativeTime formats a time relative to now, with direction.
// Examples: "in 5m", "5m ago"
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	if t.After(now) {
		d := t.Sub(now)
		if d < time.Minute {
			return "soon"
		} else if d < time.Hour {
			return fmt.Sprintf("in %dm", int(d.Minutes()))
		} else if d < 24*time.Hour {
			return fmt.Sprintf("in %dh", int(d.Hours()))
		}
		return fmt.Sprintf("in %dd", int(d.Hours()/24))
	}
	return FormatAge(t)
}

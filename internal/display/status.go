package display

// StatusDisplay contains display information for a status value.
// This struct can be embedded in API responses for consistent rendering.
type StatusDisplay struct {
	// Label is the human-readable status text (e.g., "Running", "Completed")
	Label string `json:"label"`
	// Color is a hex color code (e.g., "#00bcd4" for cyan)
	Color string `json:"color"`
	// Icon is a unicode icon character (e.g., "▶", "✓", "✗")
	Icon string `json:"icon"`
	// Severity indicates the status category for styling
	Severity StatusSeverity `json:"severity"`
}

// StatusSeverity indicates the general category of a status.
type StatusSeverity string

const (
	SeverityInfo    StatusSeverity = "info"
	SeveritySuccess StatusSeverity = "success"
	SeverityWarning StatusSeverity = "warning"
	SeverityError   StatusSeverity = "error"
	SeverityMuted   StatusSeverity = "muted"
)

// Color constants for consistent theming across CLI and frontend.
const (
	ColorCyan    = "#00bcd4"
	ColorGreen   = "#4caf50"
	ColorYellow  = "#ff9800"
	ColorRed     = "#f44336"
	ColorMagenta = "#9c27b0"
	ColorGray    = "#9e9e9e"
	ColorBlue    = "#2196f3"
)

// TaskStatusDisplay returns display information for coordinator task statuses.
// The status parameter should be the task status string (e.g., "pending", "running").
func TaskStatusDisplay(status string) StatusDisplay {
	switch status {
	case "pending":
		return StatusDisplay{
			Label:    "Pending",
			Color:    ColorYellow,
			Icon:     "○",
			Severity: SeverityInfo,
		}
	case "queued":
		return StatusDisplay{
			Label:    "Queued",
			Color:    ColorYellow,
			Icon:     "◎",
			Severity: SeverityInfo,
		}
	case "running":
		return StatusDisplay{
			Label:    "Running",
			Color:    ColorCyan,
			Icon:     "▶",
			Severity: SeverityInfo,
		}
	case "pending_approval":
		return StatusDisplay{
			Label:    "Awaiting Approval",
			Color:    ColorMagenta,
			Icon:     "⏳",
			Severity: SeverityWarning,
		}
	case "completed":
		return StatusDisplay{
			Label:    "Completed",
			Color:    ColorGreen,
			Icon:     "✓",
			Severity: SeveritySuccess,
		}
	case "failed":
		return StatusDisplay{
			Label:    "Failed",
			Color:    ColorRed,
			Icon:     "✗",
			Severity: SeverityError,
		}
	case "rejected":
		return StatusDisplay{
			Label:    "Rejected",
			Color:    ColorRed,
			Icon:     "⊘",
			Severity: SeverityError,
		}
	case "cancelled":
		return StatusDisplay{
			Label:    "Cancelled",
			Color:    ColorGray,
			Icon:     "⊗",
			Severity: SeverityMuted,
		}
	case "duplicate":
		return StatusDisplay{
			Label:    "Duplicate",
			Color:    ColorGray,
			Icon:     "⊜",
			Severity: SeverityMuted,
		}
	default:
		return StatusDisplay{
			Label:    status,
			Color:    ColorGray,
			Icon:     "?",
			Severity: SeverityMuted,
		}
	}
}

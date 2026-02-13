// Package observatory provides a unified observability platform for AILANG.
package observatory

// ===== Control Plane Types =====

// ControlPlaneFilter defines filter parameters for Control Plane queries
type ControlPlaneFilter struct {
	SourceType string // eval, coordinator, direct_api, local, other
	Provider   string // claude, gemini, openai, etc.
	Model      string // claude-sonnet-4-5, gemini-2-5-pro, etc.
	Workspace  string // workspace ID
	StartDate  string // YYYY-MM-DD format for time range filter (inclusive)
	EndDate    string // YYYY-MM-DD format for time range filter (inclusive)
}

// IsEmpty returns true if no filters are set
func (f *ControlPlaneFilter) IsEmpty() bool {
	return f.SourceType == "" && f.Provider == "" && f.Model == "" && f.Workspace == "" && f.StartDate == "" && f.EndDate == ""
}

// HasTimeRange returns true if time range filter is set
func (f *ControlPlaneFilter) HasTimeRange() bool {
	return f.StartDate != "" || f.EndDate != ""
}

// BreakdownItem represents a single item in a breakdown aggregation
type BreakdownItem struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	SpanCount  int     `json:"span_count"`
	TaskCount  int     `json:"task_count,omitempty"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMs int64   `json:"duration_ms"` // Total execution time in ms

	// Cache metrics
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheSavingsUSD     float64 `json:"cache_savings_usd"`
}

// HeatmapDataPoint represents activity data for a single day
type HeatmapDataPoint struct {
	Date        string  `json:"date"`         // YYYY-MM-DD
	SpanCount   int     `json:"span_count"`   // Number of spans
	TaskCount   int     `json:"task_count"`   // Number of distinct tasks
	Cost        float64 `json:"cost"`         // Total cost USD
	SuccessRate float64 `json:"success_rate"` // 0.0 to 1.0
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
}

// WorkspaceMapping provides workspace path-to-ID and ID-to-label mappings.
// This interface allows the observatory package to use workspace config without importing coordinator.
type WorkspaceMapping interface {
	// BuildWorkspaceMappingSQL returns a SQL CASE statement mapping cwdColumn values to workspace IDs.
	BuildWorkspaceMappingSQL(cwdColumn string) string
	// GetWorkspaceLabel returns a human-friendly label for a workspace ID.
	GetWorkspaceLabel(workspaceID string) string
	// GetPathPatternsForWorkspace returns SQL LIKE patterns that match a workspace ID.
	// Used for reverse mapping (workspace ID → path patterns) when filtering spans.
	GetPathPatternsForWorkspace(workspaceID string) []string
}

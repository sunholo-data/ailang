package server

import (
	"github.com/sunholo/ailang/internal/observatory"
)

// ============================================================================
// Task Evolution API Types
// ============================================================================

// TaskEvolutionPoint represents a single point in time for task metrics
type TaskEvolutionPoint struct {
	X               int     `json:"x"`                 // Normalized index (0 = start)
	Timestamp       string  `json:"timestamp"`         // ISO8601
	Cost            float64 `json:"cost"`              // Cumulative cost
	Tokens          int64   `json:"tokens"`            // Cumulative tokens (in + out)
	TokensIn        int64   `json:"tokens_in"`         // Cumulative input tokens
	TokensOut       int64   `json:"tokens_out"`        // Cumulative output tokens
	Turns           int     `json:"turns"`             // Cumulative turn count
	Spans           int     `json:"spans"`             // Cumulative span count
	DeltaCost       float64 `json:"delta_cost"`        // Cost change since last point
	DeltaSpans      int     `json:"delta_spans"`       // Span count change since last point
	DurationMs      int64   `json:"duration_ms"`       // Cumulative execution time in ms
	DeltaDurationMs int64   `json:"delta_duration_ms"` // Duration of this span in ms
	ElapsedMs       int64   `json:"elapsed_ms"`        // Wall clock time since task start in ms
}

// TaskEvolutionTask represents a single task's evolution data
type TaskEvolutionTask struct {
	TaskID    string               `json:"task_id"`
	Title     string               `json:"title,omitempty"`
	Provider  string               `json:"provider,omitempty"`
	Model     string               `json:"model,omitempty"`
	StartTime string               `json:"start_time"`
	EndTime   string               `json:"end_time,omitempty"`
	Status    string               `json:"status,omitempty"`
	Points    []TaskEvolutionPoint `json:"points"`
}

// TaskEvolutionResponse is the response for GET /api/controlplane/task-evolution
type TaskEvolutionResponse struct {
	Tasks      []TaskEvolutionTask `json:"tasks"`
	CliCommand string              `json:"cli_command"`
}

// ============================================================================
// Usage Time Series API Types
// ============================================================================

// UsageTimeSeriesPoint represents aggregated metrics for a time bucket
type UsageTimeSeriesPoint struct {
	Bucket      string             `json:"bucket"`                 // ISO8601 bucket start
	BucketEnd   string             `json:"bucket_end"`             // ISO8601 bucket end
	Cost        float64            `json:"cost"`                   // Total cost in bucket
	Tokens      int64              `json:"tokens"`                 // Total tokens
	TokensIn    int64              `json:"tokens_in"`              // Total input tokens
	TokensOut   int64              `json:"tokens_out"`             // Total output tokens
	Turns       int                `json:"turns"`                  // Turn count (api_request spans)
	Spans       int                `json:"spans"`                  // Total span count
	TaskCount   int                `json:"task_count"`             // Distinct tasks
	DurationMs  int64              `json:"duration_ms"`            // Total duration in bucket (ms)
	ByDimension map[string]float64 `json:"by_dimension,omitempty"` // Split by dimension
}

// UsageTimeSeriesResponse is the response for GET /api/controlplane/usage-timeseries
type UsageTimeSeriesResponse struct {
	Points     []UsageTimeSeriesPoint `json:"points"`
	Interval   string                 `json:"interval"`           // hour, day, week
	Metric     string                 `json:"metric"`             // Primary metric shown
	SplitBy    string                 `json:"split_by,omitempty"` // Dimension for split
	TotalCost  float64                `json:"total_cost"`
	CliCommand string                 `json:"cli_command"`
}

// ============================================================================
// Token Distribution API Types
// ============================================================================

// TokenDistributionBucket represents a bucket in the token histogram
type TokenDistributionBucket struct {
	Label     string  `json:"label"`      // e.g., "0-1K", "1K-5K"
	Min       int64   `json:"min"`        // Minimum tokens (inclusive)
	Max       int64   `json:"max"`        // Maximum tokens (exclusive, -1 for no limit)
	TaskCount int     `json:"task_count"` // Number of tasks in bucket
	SpanCount int     `json:"span_count"` // Number of spans in bucket
	TotalCost float64 `json:"total_cost"` // Total cost in bucket
}

// TokenDistributionResponse is the response for GET /api/controlplane/token-distribution
type TokenDistributionResponse struct {
	Buckets    []TokenDistributionBucket `json:"buckets"`
	TotalTasks int                       `json:"total_tasks"`
	TotalCost  float64                   `json:"total_cost"`
	CliCommand string                    `json:"cli_command"`
}

// ============================================================================
// Outliers Analysis Types
// ============================================================================

// OutliersAnalysisResponse is the response for GET /api/controlplane/outliers
type OutliersAnalysisResponse struct {
	TaskID       string                         `json:"task_id"`
	TaskTitle    string                         `json:"task_title"`
	SpanCount    int                            `json:"span_count"`
	Threshold    float64                        `json:"threshold"`
	Stats        []*observatory.TaskMetricStats `json:"stats"`
	Outliers     []*observatory.SpanOutlier     `json:"outliers"`
	RateOfChange *observatory.RateAnalysis      `json:"rate_of_change,omitempty"`
	CliCommand   string                         `json:"cli_command"`
	AnalyzedAt   string                         `json:"analyzed_at"`
}

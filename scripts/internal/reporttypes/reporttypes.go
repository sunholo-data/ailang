// Package reporttypes provides common types for example verification scripts
package reporttypes

import "time"

// ExampleResult represents the result of running a single example
type ExampleResult struct {
	File        string        `json:"file"`
	Status      string        `json:"status"` // "passed", "failed", "skipped"
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	Output      string        `json:"output,omitempty"`
	TraceStatus string        `json:"trace_status,omitempty"` // "match", "mismatch", "new", "error"
	TraceEvents int           `json:"trace_events,omitempty"` // Number of trace events captured
	TraceScore  float64       `json:"trace_score,omitempty"`  // Quality score 0.0-1.0
}

// TraceSummary aggregates trace verification results across all examples
type TraceSummary struct {
	Traced     int     `json:"traced"`     // Examples that produced traces
	Matches    int     `json:"matches"`    // Traces matching baselines
	Mismatches int     `json:"mismatches"` // Traces differing from baselines
	NewTraces  int     `json:"new_traces"` // No baseline to compare against
	Errors     int     `json:"errors"`     // Trace capture/replay errors
	AvgScore   float64 `json:"avg_score"`  // Average quality score
}

// VerificationReport represents the complete report of example verification
type VerificationReport struct {
	Timestamp     time.Time       `json:"timestamp"`
	TotalExamples int             `json:"total_examples"`
	Passed        int             `json:"passed"`
	Failed        int             `json:"failed"`
	Skipped       int             `json:"skipped"`
	TraceSummary  *TraceSummary   `json:"trace_summary,omitempty"`
	Results       []ExampleResult `json:"results"`
}

// Package reporttypes provides common types for example verification scripts
package reporttypes

import "time"

// ExampleResult represents the result of running a single example
type ExampleResult struct {
	File     string        `json:"file"`
	Status   string        `json:"status"`      // "passed", "failed", "skipped"
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`
}

// VerificationReport represents the complete report of example verification
type VerificationReport struct {
	Timestamp     time.Time       `json:"timestamp"`
	TotalExamples int             `json:"total_examples"`
	Passed        int             `json:"passed"`
	Failed        int             `json:"failed"`
	Skipped       int             `json:"skipped"`
	Results       []ExampleResult `json:"results"`
}
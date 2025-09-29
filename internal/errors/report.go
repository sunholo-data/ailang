package errors

import (
	"encoding/json"
	"errors"

	"github.com/sunholo/ailang/internal/ast"
)

// Report is the canonical structured error type for AILANG
// All error builders should return *Report, which can be wrapped as ReportError
type Report struct {
	Schema  string         `json:"schema"`         // Always "ailang.error/v1"
	Code    string         `json:"code"`           // Error code (IMP010, LDR001, etc.)
	Phase   string         `json:"phase"`          // Phase: "parser", "loader", "link", "typecheck", etc.
	Message string         `json:"message"`        // Human-readable message
	Span    *ast.Span      `json:"span,omitempty"` // Source location (optional)
	Data    map[string]any `json:"data,omitempty"` // Structured data (sorted keys)
	Fix     *Fix           `json:"fix,omitempty"`  // Suggested fix (optional)
}

// ReportError wraps a Report as an error
// This allows structured reports to survive errors.As() unwrapping
type ReportError struct {
	Rep *Report
}

// Error implements the error interface
func (e *ReportError) Error() string {
	if e.Rep == nil {
		return "unknown error"
	}
	return e.Rep.Code + ": " + e.Rep.Message
}

// AsReport attempts to extract a Report from an error chain
// Returns the Report and true if found, nil and false otherwise
func AsReport(err error) (*Report, bool) {
	var re *ReportError
	if errors.As(err, &re) {
		return re.Rep, true
	}
	return nil, false
}

// WrapReport wraps a Report as a ReportError
// Call sites should return errors.WrapReport(report) to preserve structure
func WrapReport(r *Report) error {
	if r == nil {
		return nil
	}
	return &ReportError{Rep: r}
}

// ToJSON converts a Report to JSON (deterministic, sorted keys)
func (r *Report) ToJSON(compact bool) (string, error) {
	var data []byte
	var err error

	if compact {
		data, err = json.Marshal(r)
	} else {
		data, err = json.MarshalIndent(r, "", "  ")
	}

	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewGeneric creates a generic error report for runtime errors
func NewGeneric(phase string, err error) *Report {
	return &Report{
		Schema:  "ailang.error/v1",
		Code:    "RUNTIME",
		Phase:   phase,
		Message: err.Error(),
		Data:    map[string]any{},
	}
}

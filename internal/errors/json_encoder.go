// Package errors provides structured error encoding for AI-first error reporting.
package errors

import (
	"fmt"
	"github.com/sunholo/ailang/internal/schema"
)

// Error codes taxonomy
const (
	// Type checking errors (TC###)
	TC001 = "TC001" // Type mismatch
	TC002 = "TC002" // Unbound variable
	TC003 = "TC003" // Constraint solving failed
	TC004 = "TC004" // Occurs check failed
	TC005 = "TC005" // Kind mismatch
	TC006 = "TC006" // Missing type annotation
	TC007 = "TC007" // Defaulting ambiguity

	// Elaboration errors (ELB###)
	ELB001 = "ELB001" // Invalid AST structure
	ELB002 = "ELB002" // Dictionary resolution failed
	ELB003 = "ELB003" // ANF transformation error
	ELB004 = "ELB004" // Pattern match exhaustiveness

	// Linking errors (LNK###)
	LNK001 = "LNK001" // Missing dictionary instance
	LNK002 = "LNK002" // Ambiguous instance
	LNK003 = "LNK003" // Module not found
	LNK004 = "LNK004" // Circular dependency

	// Runtime errors (RT###)
	RT001 = "RT001" // Division by zero
	RT002 = "RT002" // Pattern match failure
	RT003 = "RT003" // Index out of bounds
	RT004 = "RT004" // Null pointer
	RT005 = "RT005" // Stack overflow
	RT006 = "RT006" // Type assertion failed
)

// Fix represents a suggested fix with confidence score
type Fix struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

// Encoded represents a structured error in JSON format
type Encoded struct {
	Schema     string      `json:"schema"`
	SID        string      `json:"sid"`
	Phase      string      `json:"phase"`
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Fix        Fix         `json:"fix"`
	Context    interface{} `json:"context,omitempty"`
	SourceSpan string      `json:"source_span,omitempty"`
	Meta       interface{} `json:"meta,omitempty"`
}

// NewTypecheck creates a type checking error
func NewTypecheck(sid, code, msg string, ctx interface{}) Encoded {
	if sid == "" {
		sid = "unknown"
	}
	return Encoded{
		Schema:  schema.ErrorV1,
		SID:     sid,
		Phase:   "typecheck",
		Code:    code,
		Message: msg,
		Fix:     Fix{Suggestion: "", Confidence: 0.0},
		Context: ctx,
	}
}

// NewElaboration creates an elaboration error
func NewElaboration(sid, code, msg string, ctx interface{}) Encoded {
	if sid == "" {
		sid = "unknown"
	}
	return Encoded{
		Schema:  schema.ErrorV1,
		SID:     sid,
		Phase:   "elaboration",
		Code:    code,
		Message: msg,
		Fix:     Fix{Suggestion: "", Confidence: 0.0},
		Context: ctx,
	}
}

// NewLinking creates a linking error
func NewLinking(sid, code, msg string, ctx interface{}) Encoded {
	if sid == "" {
		sid = "unknown"
	}
	return Encoded{
		Schema:  schema.ErrorV1,
		SID:     sid,
		Phase:   "linking",
		Code:    code,
		Message: msg,
		Fix:     Fix{Suggestion: "", Confidence: 0.0},
		Context: ctx,
	}
}

// NewRuntime creates a runtime error
func NewRuntime(sid, code, msg string, ctx interface{}) Encoded {
	if sid == "" {
		sid = "unknown"
	}
	return Encoded{
		Schema:  schema.ErrorV1,
		SID:     sid,
		Phase:   "runtime",
		Code:    code,
		Message: msg,
		Fix:     Fix{Suggestion: "", Confidence: 0.0},
		Context: ctx,
	}
}

// WithFix adds a fix suggestion to the error
func (e Encoded) WithFix(suggestion string, confidence float64) Encoded {
	e.Fix = Fix{
		Suggestion: suggestion,
		Confidence: confidence,
	}
	return e
}

// WithSourceSpan adds source location to the error
func (e Encoded) WithSourceSpan(span string) Encoded {
	e.SourceSpan = span
	return e
}

// WithMeta adds metadata to the error
func (e Encoded) WithMeta(meta interface{}) Encoded {
	e.Meta = meta
	return e
}

// ToJSON converts the error to deterministic JSON
func (e Encoded) ToJSON() ([]byte, error) {
	data, err := schema.MarshalDeterministic(e)
	if err != nil {
		// Fallback if encoding fails
		fallback := Encoded{
			Schema:  schema.ErrorV1,
			Message: "encoding failed",
			Meta:    map[string]string{"original_error": err.Error()},
		}
		return schema.MarshalDeterministic(fallback)
	}
	return schema.FormatJSON(data)
}

// ErrorContext provides structured context for errors
type ErrorContext struct {
	Constraints []string `json:"constraints,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
	TraceSlice  string   `json:"trace_slice,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// SafeEncodeError safely encodes any error, never panics
func SafeEncodeError(err error, phase string) []byte {
	if err == nil {
		return nil
	}

	// Try to extract more information if it's a known error type
	encoded := Encoded{
		Schema:  schema.ErrorV1,
		SID:     "unknown",
		Phase:   phase,
		Code:    "ERR000",
		Message: err.Error(),
		Fix:     Fix{Suggestion: "", Confidence: 0.0},
	}

	data, _ := encoded.ToJSON()
	return data
}

// FormatSourceSpan formats file position as "file:line:col"
func FormatSourceSpan(file string, line, col int) string {
	return fmt.Sprintf("%s:%d:%d", file, line, col)
}
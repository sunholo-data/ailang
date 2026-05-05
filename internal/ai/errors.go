package ai

import (
	"errors"
	"fmt"
	"strings"
)

// Error codes used in AIError.Code. The vocabulary is a superset of the
// codes already in use by the streaming surface (cmd/ailang/configdriven_streaming.go's
// wrapErrAsAIError) — the new entries (RateLimit, ContextLength,
// SchemaValidation, ToolsNotSupported, ModelNotFound, Internal) are added by
// M-AI-TOOL-LOOP for the non-streaming Step / callResult / callJsonResult
// paths.
//
// Pre-existing codes (M-AI-CALL-STREAM-HELPER, v0.15.1):
//
//	AuthFailed, Timeout, ConnectionFailed, BudgetExhausted,
//	ProviderNotFound, CapabilityNotSupported, ProtocolError, ModelNotAllowed
const (
	CodeAuthFailed             = "AuthFailed"
	CodeTimeout                = "Timeout"
	CodeConnectionFailed       = "ConnectionFailed"
	CodeBudgetExhausted        = "BudgetExhausted"
	CodeProviderNotFound       = "ProviderNotFound"
	CodeCapabilityNotSupported = "CapabilityNotSupported"
	CodeProtocolError          = "ProtocolError"
	CodeModelNotAllowed        = "ModelNotAllowed"

	// New for M-AI-TOOL-LOOP (non-streaming + tool-loop).
	CodeRateLimit         = "RateLimit"
	CodeContextLength     = "ContextLength"
	CodeSchemaValidation  = "SchemaValidation"
	CodeToolsNotSupported = "ToolsNotSupported"
	CodeModelNotFound     = "ModelNotFound"
	CodeInternal          = "Internal"
)

// AIError is the canonical typed error returned by Provider.Step (and consumed
// by the new _ai_call_result / _ai_call_json_result / _ai_step builtins).
//
// Shape mirrors std/ai/streaming.AIError exactly: { code, message, retryable }.
// Provider and statusCode are intentionally NOT included — they were
// considered and deferred when the AIError record shape was locked in v0.15.0.
// Add them here only if a downstream consumer (motoko_agent, eval harness)
// reports a concrete need. Record-shape extension is additive and safe.
type AIError struct {
	Code      string // one of the Code* constants above
	Message   string // human-readable, may include provider name verbatim
	Retryable bool   // caller's retry hint
}

// Error implements the error interface so AIError can flow through Go error
// returns and be unwrapped by callers that prefer to keep the typed shape.
func (e *AIError) Error() string {
	if e == nil {
		return "<nil AIError>"
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAIError constructs an AIError; convenience to avoid &ai.AIError{...}
// at every call site.
func NewAIError(code, message string, retryable bool) *AIError {
	return &AIError{Code: code, Message: message, Retryable: retryable}
}

// IsRetryable returns the canonical retryable hint for an AIError code.
// Single source of truth — adapters and the AILANG-side wrapErrAsAIError
// (cmd/ailang/configdriven_streaming.go) both call this.
//
// The convention: transient/network/load conditions are retryable;
// configuration/auth/schema mismatches are not. Unknown codes default to
// retryable=true (conservative) so adapters that emit a new code without
// updating this table still surface as recoverable.
func IsRetryable(code string) bool {
	switch code {
	case CodeTimeout,
		CodeConnectionFailed,
		CodeRateLimit,
		CodeInternal:
		return true
	case CodeAuthFailed,
		CodeBudgetExhausted,
		CodeProviderNotFound,
		CodeCapabilityNotSupported,
		CodeProtocolError,
		CodeModelNotAllowed,
		CodeContextLength,
		CodeSchemaValidation,
		CodeToolsNotSupported,
		CodeModelNotFound:
		return false
	}
	// Unknown code — default to retryable so adapters that emit a custom
	// code don't accidentally surface as fatal.
	return true
}

// ClassifyHTTPError maps an HTTP status + provider response body into an
// AIError with a normalized code. Adapters call this from their Step
// implementations after parsing a non-2xx response.
//
// The body is scanned for substrings that disambiguate codes within a
// status class — e.g. HTTP 400 may be a context-length overflow ("context
// length exceeded") OR a schema validation failure ("does not match
// schema") OR plain bad request. The match is case-insensitive and only
// looks for high-confidence signals; ambiguous bodies fall through to
// the status-code default.
func ClassifyHTTPError(provider string, statusCode int, body string) *AIError {
	bodyLower := strings.ToLower(body)
	msg := strings.TrimSpace(body)
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d from %s", statusCode, provider)
	}

	switch {
	case statusCode == 401, statusCode == 403:
		return NewAIError(CodeAuthFailed, msg, false)
	case statusCode == 404:
		return NewAIError(CodeModelNotFound, msg, false)
	case statusCode == 429:
		return NewAIError(CodeRateLimit, msg, true)
	case statusCode == 400:
		// 400 disambiguation: context length and schema validation are
		// common, non-retryable shapes. Otherwise plain bad request.
		if strings.Contains(bodyLower, "context length") ||
			strings.Contains(bodyLower, "context_length") ||
			strings.Contains(bodyLower, "maximum context") ||
			strings.Contains(bodyLower, "too many tokens") {
			return NewAIError(CodeContextLength, msg, false)
		}
		if strings.Contains(bodyLower, "schema") &&
			(strings.Contains(bodyLower, "does not match") ||
				strings.Contains(bodyLower, "invalid") ||
				strings.Contains(bodyLower, "validation")) {
			return NewAIError(CodeSchemaValidation, msg, false)
		}
		return NewAIError(CodeProtocolError, msg, false)
	case statusCode >= 500 && statusCode < 600:
		return NewAIError(CodeInternal, msg, true)
	}
	// Anything else (3xx redirects, 1xx informational, undefined) — treat
	// as protocol error and let the caller decide.
	return NewAIError(CodeProtocolError, msg, false)
}

// ClassifyError maps a non-HTTP Go error into an AIError. Useful for
// transport/timeout/cancel errors that surface before any HTTP response
// is received.
func ClassifyError(err error) *AIError {
	if err == nil {
		return nil
	}
	// Allow callers to thread an existing AIError through unchanged.
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr
	}
	msg := err.Error()
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "context deadline exceeded"),
		strings.Contains(msgLower, "timeout"),
		strings.Contains(msgLower, "timed out"):
		return NewAIError(CodeTimeout, msg, true)
	case strings.Contains(msgLower, "context canceled"):
		return NewAIError(CodeTimeout, msg, true)
	case strings.Contains(msgLower, "connection refused"),
		strings.Contains(msgLower, "connection reset"),
		strings.Contains(msgLower, "no such host"),
		strings.Contains(msgLower, "tls"),
		strings.Contains(msgLower, "eof"):
		return NewAIError(CodeConnectionFailed, msg, true)
	}
	return NewAIError(CodeInternal, msg, true)
}

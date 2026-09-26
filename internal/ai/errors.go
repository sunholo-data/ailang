package ai

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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

	// CodeModelNoVision (M-STD-AI-VISION-INPUT, v0.30.0) is returned when a
	// message carries image parts but the selected model/provider cannot accept
	// vision input — a clear typed error rather than silently dropping the image.
	CodeModelNoVision = "ModelNoVision"
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

	// wrapped is an optional underlying sentinel error (unexported so it does
	// not affect the JSON/record wire shape, which stays {code, message,
	// retryable}). Set by the reasoning resolver so callers can errors.Is the
	// typed reasoning sentinels while errors.As still yields *AIError. nil for
	// all pre-existing AIError constructions.
	wrapped error
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

// Unwrap returns the optional underlying sentinel error so callers can match
// wrapped reasoning sentinels with errors.Is. Returns nil for AIErrors that
// were not constructed with a wrapped sentinel (all pre-existing call sites).
func (e *AIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.wrapped
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
		CodeModelNotFound,
		CodeModelNoVision:
		return false
	}
	// Unknown code — default to retryable so adapters that emit a custom
	// code don't accidentally surface as fatal.
	return true
}

// ClassifyHTTPError maps an HTTP status + provider response body into an
// AIError with a normalized code. Adapters call this from their Step
// implementations after parsing a non-2xx response; DoJSON calls it for
// every client; ClassifyError calls it for a *ProviderError that carries a
// status.
//
// The body is scanned for substrings that disambiguate codes within a
// status class — e.g. HTTP 400 may be a context-length overflow ("context
// length exceeded") OR a schema validation failure ("does not match
// schema") OR plain bad request, and HTTP 429 may be a transient rate limit
// OR a spent quota window (Ollama Cloud returns exhaustion as 429 with type
// "api_error"; only the message separates them — see IsQuotaExhausted). The
// match is case-insensitive and only looks for high-confidence signals;
// ambiguous bodies fall through to the status-code default.
func ClassifyHTTPError(provider string, statusCode int, body string) *AIError {
	bodyLower := strings.ToLower(body)
	msg := strings.TrimSpace(body)
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d from %s", statusCode, provider)
	}

	switch {
	case statusCode == 401, statusCode == 403:
		return NewAIError(CodeAuthFailed, msg, false)
	case statusCode == 402:
		// Payment Required: OpenRouter's out-of-credits answer. Not a rate
		// limit — no wait makes it succeed.
		return NewAIError(CodeBudgetExhausted, msg, false)
	case statusCode == 404:
		return NewAIError(CodeModelNotFound, msg, false)
	case statusCode == 429:
		if IsQuotaExhausted(bodyLower) {
			// A spent session/weekly/monthly window does not clear on retry;
			// retrying burns the rest of the run against a bucket that cannot
			// recover (measured 2026-08-26, M-OLLAMA-CLOUD V22).
			return NewAIError(CodeBudgetExhausted, msg, false)
		}
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

// IsQuotaExhausted reports whether a lower-cased provider message describes a
// spent quota window rather than a transient rate limit. The two share HTTP
// status 429 at Ollama Cloud (type "api_error" — captured verbatim 2026-08-26
// by deliberately exhausting a session window, M-OLLAMA-CLOUD V22), so only the
// message can tell them apart; getting it wrong retries into a bucket that
// does not refill for hours (session) or days (weekly).
func IsQuotaExhausted(msgLower string) bool {
	for _, sig := range quotaExhaustionSignals {
		if strings.Contains(msgLower, sig) {
			return true
		}
	}
	return false
}

var quotaExhaustionSignals = []string{
	// Ollama Cloud (V22, verbatim). Both windows.
	"session usage limit",
	"weekly usage limit",
	"usage limit, upgrade",
	// Other providers.
	"key limit exceeded",
	"monthly limit",
	"insufficient_quota",
	"insufficient quota",
	"quota exceeded",
	"billing",
}

// httpStatusWord matches a transient/server status code as a standalone
// token, so "1500 tokens" is not read as a 500 and "id 4290" is not a 429.
var httpStatusWord = regexp.MustCompile(`(^|[^0-9])(429|500|502|503|504)([^0-9]|$)`)

func hasStatusWord(msgLower string, codes ...string) bool {
	m := httpStatusWord.FindStringSubmatch(msgLower)
	if m == nil {
		return false
	}
	for _, c := range codes {
		if m[2] == c {
			return true
		}
	}
	return false
}

// ClassifyError maps any Go error into an AIError. It is the ONE classifier:
// a *AIError passes through unchanged; a *ProviderError carrying an HTTP
// status goes through ClassifyHTTPError (so a config-driven provider's 429 is
// CodeRateLimit, not CodeInternal); everything else is matched on its
// message — quota exhaustion, rate limits, timeouts, connection failures and
// server errors, in that order. An unrecognised error is CodeInternal with
// Retryable=true, the conservative default for AILANG-level callers that
// must not treat a new adapter code as fatal; retry LOOPS use ShouldRetry
// instead, which refuses to spend on an error nobody recognised.
func ClassifyError(err error) *AIError {
	e, _ := classify(err)
	return e
}

// ShouldRetry is the retry-loop predicate shared by the eval harness and the
// coordinator (M-V1-SIMPLIFY-S3 M4; it replaces two substring matchers that
// disagreed with each other and with this file). True only for an error
// classified into a KNOWN transient code — rate limit, timeout, connection
// failure, server error — and never for quota exhaustion, which arrives as a
// 429 but does not clear on retry. Unknown errors are not retried: a retry
// loop spends money, and "we could not tell what went wrong" is not a reason
// to spend it three more times.
func ShouldRetry(err error) bool {
	e, known := classify(err)
	return e != nil && known && e.Retryable
}

// classify returns the AIError plus whether it came from a recognised signal
// (false only for the CodeInternal catch-all).
func classify(err error) (*AIError, bool) {
	if err == nil {
		return nil, false
	}
	// Allow callers to thread an existing AIError through unchanged.
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr, true
	}
	var perr *ProviderError
	if errors.As(err, &perr) && perr.StatusCode > 0 {
		return ClassifyHTTPError(perr.Provider, perr.StatusCode, perr.Message), true
	}
	msg := err.Error()
	msgLower := strings.ToLower(msg)
	switch {
	case IsQuotaExhausted(msgLower):
		// Checked BEFORE the 429 rule: exhaustion arrives as a 429.
		return NewAIError(CodeBudgetExhausted, msg, false), true
	case hasStatusWord(msgLower, "429"),
		strings.Contains(msgLower, "rate limit"),
		strings.Contains(msgLower, "too many requests"):
		return NewAIError(CodeRateLimit, msg, true), true
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, context.Canceled),
		strings.Contains(msgLower, "context deadline exceeded"),
		strings.Contains(msgLower, "timeout"),
		strings.Contains(msgLower, "timed out"),
		strings.Contains(msgLower, "context canceled"):
		return NewAIError(CodeTimeout, msg, true), true
	case strings.Contains(msgLower, "connection"),
		strings.Contains(msgLower, "no such host"),
		strings.Contains(msgLower, "network"),
		strings.Contains(msgLower, "tls"),
		strings.Contains(msgLower, "eof"):
		return NewAIError(CodeConnectionFailed, msg, true), true
	case hasStatusWord(msgLower, "500", "502", "503", "504"),
		strings.Contains(msgLower, "internal server error"),
		strings.Contains(msgLower, "bad gateway"),
		strings.Contains(msgLower, "service unavailable"),
		strings.Contains(msgLower, "gateway timeout"):
		return NewAIError(CodeInternal, msg, true), true
	}
	return NewAIError(CodeInternal, msg, true), false
}

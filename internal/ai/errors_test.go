package ai

import (
	"errors"
	"testing"
)

func TestIsRetryable_Codes(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		// Pre-existing v0.15.x codes (regression coverage)
		{CodeTimeout, true},
		{CodeConnectionFailed, true},
		{CodeAuthFailed, false},
		{CodeBudgetExhausted, false},
		{CodeProviderNotFound, false},
		{CodeCapabilityNotSupported, false},
		{CodeProtocolError, false},
		{CodeModelNotAllowed, false},

		// New for M-AI-TOOL-LOOP
		{CodeRateLimit, true},
		{CodeContextLength, false},
		{CodeSchemaValidation, false},
		{CodeToolsNotSupported, false},
		{CodeModelNotFound, false},
		{CodeInternal, true},

		// Unknown code — conservative default
		{"SomeFutureCode", true},
		{"", true},
	}
	for _, tc := range cases {
		if got := IsRetryable(tc.code); got != tc.want {
			t.Errorf("IsRetryable(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestAIError_Error(t *testing.T) {
	e := NewAIError(CodeRateLimit, "throttled by upstream", true)
	want := "[RateLimit] throttled by upstream"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAIError_Error_Nil(t *testing.T) {
	var e *AIError
	if got := e.Error(); got != "<nil AIError>" {
		t.Errorf("nil AIError.Error() = %q, want sentinel", got)
	}
}

func TestAIError_ImplementsError(t *testing.T) {
	var _ error = (*AIError)(nil)
	// Round-trip through errors.As
	original := NewAIError(CodeAuthFailed, "bad key", false)
	var wrapped error = original
	var unwrapped *AIError
	if !errors.As(wrapped, &unwrapped) {
		t.Fatal("errors.As did not unwrap *AIError")
	}
	if unwrapped.Code != CodeAuthFailed {
		t.Errorf("unwrapped.Code = %q, want %q", unwrapped.Code, CodeAuthFailed)
	}
}

func TestClassifyHTTPError_StatusCodes(t *testing.T) {
	cases := []struct {
		name        string
		statusCode  int
		body        string
		wantCode    string
		wantRetry   bool
		msgContains string
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401, body: "invalid api key",
			wantCode: CodeAuthFailed, wantRetry: false,
		},
		{
			name:       "403 forbidden",
			statusCode: 403, body: "forbidden",
			wantCode: CodeAuthFailed, wantRetry: false,
		},
		{
			name:       "404 model not found",
			statusCode: 404, body: "model 'claude-omega-99' not found",
			wantCode: CodeModelNotFound, wantRetry: false,
		},
		{
			name:       "429 rate limit",
			statusCode: 429, body: "too many requests",
			wantCode: CodeRateLimit, wantRetry: true,
		},
		{
			name:       "400 context length exceeded",
			statusCode: 400, body: `{"error":"context length exceeded: 100000 > 32000"}`,
			wantCode: CodeContextLength, wantRetry: false,
		},
		{
			name:       "400 maximum context",
			statusCode: 400, body: "Maximum context window exceeded",
			wantCode: CodeContextLength, wantRetry: false,
		},
		{
			name:       "400 too many tokens",
			statusCode: 400, body: "request has too many tokens",
			wantCode: CodeContextLength, wantRetry: false,
		},
		{
			name:       "400 schema validation",
			statusCode: 400, body: `{"error":"response does not match schema"}`,
			wantCode: CodeSchemaValidation, wantRetry: false,
		},
		{
			name:       "400 schema invalid",
			statusCode: 400, body: "invalid schema for response_format",
			wantCode: CodeSchemaValidation, wantRetry: false,
		},
		{
			name:       "400 plain bad request",
			statusCode: 400, body: "missing required field model",
			wantCode: CodeProtocolError, wantRetry: false,
		},
		{
			name:       "500 server error",
			statusCode: 500, body: "upstream server error",
			wantCode: CodeInternal, wantRetry: true,
		},
		{
			name:       "503 service unavailable",
			statusCode: 503, body: "service unavailable",
			wantCode: CodeInternal, wantRetry: true,
		},
		{
			name:       "302 redirect (treated as protocol error)",
			statusCode: 302, body: "found",
			wantCode: CodeProtocolError, wantRetry: false,
		},
		{
			name:       "empty body falls back to status string",
			statusCode: 500, body: "",
			wantCode: CodeInternal, wantRetry: true, msgContains: "HTTP 500 from anthropic",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyHTTPError("anthropic", tc.statusCode, tc.body)
			if got.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tc.wantCode)
			}
			if got.Retryable != tc.wantRetry {
				t.Errorf("Retryable = %v, want %v", got.Retryable, tc.wantRetry)
			}
			if tc.msgContains != "" && !contains(got.Message, tc.msgContains) {
				t.Errorf("Message = %q, want to contain %q", got.Message, tc.msgContains)
			}
		})
	}
}

func TestClassifyHTTPError_CaseInsensitiveBody(t *testing.T) {
	// Substring matching on body should be case-insensitive for the
	// disambiguation signals (context length, schema).
	got := ClassifyHTTPError("openai", 400, "CONTEXT LENGTH EXCEEDED")
	if got.Code != CodeContextLength {
		t.Errorf("upper-case body: Code = %q, want %q", got.Code, CodeContextLength)
	}
}

func TestClassifyError_Timeouts(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{errors.New("context deadline exceeded"), CodeTimeout},
		{errors.New("operation timed out"), CodeTimeout},
		{errors.New("context canceled"), CodeTimeout},
		{errors.New("Post: net/http: timeout"), CodeTimeout},
	}
	for _, tc := range cases {
		got := ClassifyError(tc.err)
		if got.Code != tc.wantCode {
			t.Errorf("ClassifyError(%v).Code = %q, want %q", tc.err, got.Code, tc.wantCode)
		}
		if !got.Retryable {
			t.Errorf("ClassifyError(%v).Retryable = false, want true", tc.err)
		}
	}
}

func TestClassifyError_Connection(t *testing.T) {
	cases := []error{
		errors.New("dial tcp: connection refused"),
		errors.New("read tcp: connection reset by peer"),
		errors.New("dial tcp: lookup host: no such host"),
		errors.New("tls: handshake failure"),
		errors.New("read: unexpected EOF"),
	}
	for _, err := range cases {
		got := ClassifyError(err)
		if got.Code != CodeConnectionFailed {
			t.Errorf("ClassifyError(%v).Code = %q, want %q", err, got.Code, CodeConnectionFailed)
		}
		if !got.Retryable {
			t.Errorf("ClassifyError(%v).Retryable = false, want true", err)
		}
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	// Unknown errors fall through to Internal+retryable (conservative).
	got := ClassifyError(errors.New("something completely unexpected"))
	if got.Code != CodeInternal {
		t.Errorf("Code = %q, want %q", got.Code, CodeInternal)
	}
	if !got.Retryable {
		t.Error("Retryable = false, want true (conservative default)")
	}
}

func TestClassifyError_PassesThroughAIError(t *testing.T) {
	// If the error is already an *AIError, ClassifyError should return it
	// unchanged rather than wrapping it as CodeInternal.
	original := NewAIError(CodeAuthFailed, "bad key", false)
	got := ClassifyError(original)
	if got != original {
		t.Errorf("ClassifyError(*AIError) = %v, want %v (verbatim)", got, original)
	}
}

func TestClassifyError_Nil(t *testing.T) {
	if got := ClassifyError(nil); got != nil {
		t.Errorf("ClassifyError(nil) = %v, want nil", got)
	}
}

// contains is a tiny inline helper to avoid pulling strings into the test file.
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

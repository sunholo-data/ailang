package ai

import (
	"context"
	"errors"
	"testing"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	name      string
	response  *Response
	err       error
	lastReq   *Request
	callCount int
}

func (m *mockProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	m.lastReq = req
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

// Step is a stub for the M-AI-TOOL-LOOP Provider interface extension.
// Routes to Generate so existing tests that only exercise single-shot
// paths keep working unchanged.
func (m *mockProvider) Step(ctx context.Context, req *Request) (*Response, error) {
	return m.Generate(ctx, req)
}

func TestProviderError(t *testing.T) {
	tests := []struct {
		name       string
		err        *ProviderError
		wantString string
	}{
		{
			name: "with status code",
			err: &ProviderError{
				Provider:   "openai",
				StatusCode: 429,
				Message:    "rate limit exceeded",
			},
			wantString: "openai error (429): rate limit exceeded",
		},
		{
			name: "without status code",
			err: &ProviderError{
				Provider: "anthropic",
				Message:  "invalid api key",
			},
			wantString: "anthropic error: invalid api key",
		},
		{
			name: "with underlying error",
			err: &ProviderError{
				Provider:   "gemini",
				StatusCode: 500,
				Message:    "internal error",
				Err:        errors.New("connection refused"),
			},
			wantString: "gemini error (500): internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantString {
				t.Errorf("Error() = %q, want %q", got, tt.wantString)
			}
		})
	}
}

func TestProviderErrorUnwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &ProviderError{
		Provider: "test",
		Message:  "wrapper",
		Err:      underlying,
	}

	if unwrapped := err.Unwrap(); unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestNewProviderError(t *testing.T) {
	underlying := errors.New("network error")
	err := NewProviderError("openai", 503, "service unavailable", underlying)

	if err.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", err.Provider, "openai")
	}
	if err.StatusCode != 503 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 503)
	}
	if err.Message != "service unavailable" {
		t.Errorf("Message = %q, want %q", err.Message, "service unavailable")
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestGuessProvider(t *testing.T) {
	tests := []struct {
		model string
		want  ProviderType
	}{
		// OpenAI models
		{"gpt-5", ProviderOpenAI},
		{"gpt-5-mini", ProviderOpenAI},
		{"gpt-5.1", ProviderOpenAI},
		{"o1-preview", ProviderOpenAI},
		{"o3-mini", ProviderOpenAI},
		{"codex-max", ProviderOpenAI},

		// Anthropic models
		{"claude-sonnet-4-5", ProviderAnthropic},
		{"claude-haiku-4-5", ProviderAnthropic},
		{"claude-3-opus", ProviderAnthropic},

		// Google models
		{"gemini-2.5-flash", ProviderGoogle},
		{"gemini-2.5-pro", ProviderGoogle},

		// Unknown
		{"unknown-model", ""},
		{"llama-70b", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := GuessProvider(tt.model)
			if got != tt.want {
				t.Errorf("GuessProvider(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestProviderFromString(t *testing.T) {
	tests := []struct {
		input string
		want  ProviderType
	}{
		{"openai", ProviderOpenAI},
		{"OPENAI", ProviderOpenAI},
		{"OpenAI", ProviderOpenAI},
		{"anthropic", ProviderAnthropic},
		{"ANTHROPIC", ProviderAnthropic},
		{"google", ProviderGoogle},
		{"gemini", ProviderGoogle},
		{"vertex", ProviderGoogle},
		{"custom", ProviderType("custom")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ProviderFromString(tt.input)
			if got != tt.want {
				t.Errorf("ProviderFromString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandler_Call(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		response: &Response{
			Text:         "Hello, world!",
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
			Model:        "test-model",
		},
	}

	handler := NewHandler(mock, "test-model",
		WithSystemPrompt("You are helpful."),
		WithMaxTokens(2048),
	)

	result, err := handler.Call("Hi there")
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if result != "Hello, world!" {
		t.Errorf("Call() = %q, want %q", result, "Hello, world!")
	}

	// Verify request was built correctly
	if mock.lastReq.Model != "test-model" {
		t.Errorf("Model = %q, want %q", mock.lastReq.Model, "test-model")
	}
	if mock.lastReq.SystemPrompt != "You are helpful." {
		t.Errorf("SystemPrompt = %q, want %q", mock.lastReq.SystemPrompt, "You are helpful.")
	}
	if mock.lastReq.UserPrompt != "Hi there" {
		t.Errorf("UserPrompt = %q, want %q", mock.lastReq.UserPrompt, "Hi there")
	}
	if mock.lastReq.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want %d", mock.lastReq.MaxTokens, 2048)
	}
}

func TestHandler_CallError(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		err:  errors.New("api error"),
	}

	handler := NewHandler(mock, "test-model")

	_, err := handler.Call("Hi")
	if err == nil {
		t.Fatal("Call() expected error, got nil")
	}
	if err.Error() != "api error" {
		t.Errorf("Call() error = %v, want %v", err, "api error")
	}
}

func TestHandler_GenerateWithDetails(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		response: &Response{
			Text:         "Generated text",
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
			ReasonTokens: 20,
			Model:        "test-model",
		},
	}

	handler := NewHandler(mock, "test-model")

	resp, err := handler.GenerateWithDetails(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("GenerateWithDetails() error = %v", err)
	}

	if resp.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want %d", resp.InputTokens, 100)
	}
	if resp.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want %d", resp.OutputTokens, 50)
	}
	if resp.ReasonTokens != 20 {
		t.Errorf("ReasonTokens = %d, want %d", resp.ReasonTokens, 20)
	}
}

func TestHandler_DefaultMaxTokens(t *testing.T) {
	mock := &mockProvider{
		name:     "test",
		response: &Response{Text: "ok"},
	}

	// Create handler without WithMaxTokens option
	handler := NewHandler(mock, "test-model")
	handler.Call("test")

	if mock.lastReq.MaxTokens != 4096 {
		t.Errorf("default MaxTokens = %d, want %d", mock.lastReq.MaxTokens, 4096)
	}
}

func TestHandler_ProviderAndModel(t *testing.T) {
	mock := &mockProvider{name: "mock-provider"}
	handler := NewHandler(mock, "test-model")

	if handler.Provider() != mock {
		t.Error("Provider() did not return expected provider")
	}
	if handler.Model() != "test-model" {
		t.Errorf("Model() = %q, want %q", handler.Model(), "test-model")
	}
}

func TestRequest_Fields(t *testing.T) {
	req := &Request{
		Model:        "gpt-5",
		SystemPrompt: "Be helpful",
		UserPrompt:   "Hello",
		MaxTokens:    1000,
		Temperature:  0.7,
		Options: map[string]any{
			"top_p": 0.9,
		},
	}

	if req.Model != "gpt-5" {
		t.Errorf("Model = %q, want %q", req.Model, "gpt-5")
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want %f", req.Temperature, 0.7)
	}
	if req.Options["top_p"] != 0.9 {
		t.Errorf("Options[top_p] = %v, want %v", req.Options["top_p"], 0.9)
	}
}

func TestResponse_Fields(t *testing.T) {
	resp := &Response{
		Text:         "Generated content",
		InputTokens:  50,
		OutputTokens: 100,
		TotalTokens:  150,
		ReasonTokens: 25,
		Model:        "gpt-5",
	}

	if resp.Text != "Generated content" {
		t.Errorf("Text = %q, want %q", resp.Text, "Generated content")
	}
	if resp.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want %d", resp.TotalTokens, 150)
	}
	if resp.ReasonTokens != 25 {
		t.Errorf("ReasonTokens = %d, want %d", resp.ReasonTokens, 25)
	}
}

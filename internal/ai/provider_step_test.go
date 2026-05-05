package ai_test

// This test verifies the Provider.Step interface contract introduced by
// M-AI-TOOL-LOOP (v0.17.0): every concrete provider in internal/ai/ must
// implement Step, and the stub implementations must return a typed
// *ai.AIError (not nil, not a generic error) so the AILANG-side
// builtins can assert on Code/Retryable.

import (
	"context"
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// stepProvider matches the slice of the Provider interface this test cares
// about. It exists so we can iterate over heterogeneous client types
// without forcing each concrete client to expose the full Provider value.
type stepProvider interface {
	Step(ctx context.Context, req *ai.Request) (*ai.Response, error)
	Name() string
}

func TestProviderStep_AllStubsReturnAIError(t *testing.T) {
	// Note: ollama is NOT in this list — M4 wired its Step to delegate to
	// Generate when len(req.Tools) == 0 (instead of returning a stub),
	// so it has its own targeted tests in internal/ai/ollama/step_test.go.
	// The remaining providers (configdriven only, since M2-M4 landed real
	// impls in anthropic/gemini/openai/openrouter) are still stubs in
	// terms of "tools rejected, no real impl yet" — though configdriven
	// goes through CodeToolsNotSupported when tools present, and a
	// CodeInternal stub when tools absent.
	providers := []struct {
		name string
		p    stepProvider
	}{
		{"configdriven", configdriven.New(&pkg.AIProviderSpec{Name: "test"})},
	}

	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			req := &ai.Request{Model: "any-model"}
			resp, err := tc.p.Step(context.Background(), req)

			if resp != nil {
				t.Errorf("stub Step returned non-nil response: %+v", resp)
			}
			if err == nil {
				t.Fatal("stub Step returned nil error; expected *ai.AIError")
			}

			var aiErr *ai.AIError
			if !errors.As(err, &aiErr) {
				t.Fatalf("stub Step returned non-AIError: %T %v", err, err)
			}
			if aiErr.Code == "" {
				t.Errorf("AIError.Code is empty")
			}
			// Stubs are not retryable — they're configuration / not-yet-impl
			// failures, not transient ones. (Ollama with no tools currently
			// also returns an Internal stub for the no-tools path; M4 wires
			// it to Generate.)
			if aiErr.Retryable {
				t.Errorf("stub AIError marked retryable; expected false (Code=%s)", aiErr.Code)
			}
		})
	}
}

// Note: TestProviderStep_OllamaToolsNotSupported was removed — the
// equivalent assertion lives in-package at
// internal/ai/ollama/step_test.go::TestStep_ToolsRejected.

func TestProviderStep_ConfigdrivenToolsNotSupported(t *testing.T) {
	// Same shape as the ollama test — config-driven providers also
	// reject tools at the boundary in v1.
	p := configdriven.New(&pkg.AIProviderSpec{Name: "test"})
	req := &ai.Request{
		Model: "any",
		Tools: []ai.ToolSchema{{Name: "noop", Description: "n/a", Parameters: "{}"}},
	}
	_, err := p.Step(context.Background(), req)
	if err == nil {
		t.Fatal("configdriven Step with tools returned nil error; expected ToolsNotSupported")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("configdriven Step error not *AIError: %T", err)
	}
	if aiErr.Code != ai.CodeToolsNotSupported {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeToolsNotSupported)
	}
}

// TestRequest_NewFields_Default verifies that legacy callers who don't set
// the new Messages/Tools fields keep working — these are zero-valued slices
// (nil), which existing Generate paths must continue to ignore.
func TestRequest_NewFields_Default(t *testing.T) {
	req := &ai.Request{Model: "gpt-5", SystemPrompt: "be brief", UserPrompt: "hi"}
	if req.Messages != nil {
		t.Errorf("Request.Messages default = %v, want nil", req.Messages)
	}
	if req.Tools != nil {
		t.Errorf("Request.Tools default = %v, want nil", req.Tools)
	}
}

// TestResponse_NewFields_Default verifies the symmetric case on Response.
func TestResponse_NewFields_Default(t *testing.T) {
	resp := &ai.Response{Text: "hi", InputTokens: 1, OutputTokens: 1}
	if resp.ToolCalls != nil {
		t.Errorf("Response.ToolCalls default = %v, want nil", resp.ToolCalls)
	}
	if resp.FinishReason != "" {
		t.Errorf("Response.FinishReason default = %q, want \"\"", resp.FinishReason)
	}
}

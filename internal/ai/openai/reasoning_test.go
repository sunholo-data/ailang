package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func captureBody(t *testing.T, captured *string, respJSON any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*captured = string(b)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respJSON)
	}))
}

// TestOpenAI_Responses_Golden_PreservesImplicitMedium (AC14 + design table):
// with no reasoning control, the Responses body preserves its historical
// implicit reasoning.effort:"medium" block exactly.
func TestOpenAI_Responses_Golden_PreservesImplicitMedium(t *testing.T) {
	var body string
	resp := responsesResponse{
		Output: []responsesOutputItem{{Type: "message", Role: "assistant", Content: []responsesContent{{Type: "output_text", Text: "ok"}}}},
	}
	srv := captureBody(t, &body, resp)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithAPIType(APIResponses))
	_, err := c.Generate(context.Background(), &ai.Request{Model: "codex", UserPrompt: "hi"})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if !strings.Contains(body, `"reasoning":{"effort":"medium"}`) {
		t.Fatalf("Responses body must preserve implicit medium block, got %s", body)
	}
}

// TestOpenAI_Chat_Golden_NoReasoningField (AC14): with no reasoning control the
// Chat body carries NO reasoning_effort key (byte-identical; omitempty).
func TestOpenAI_Chat_Golden_NoReasoningField(t *testing.T) {
	var body string
	resp := chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: "ok"}, FinishReason: "stop"}}}
	srv := captureBody(t, &body, resp)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithAPIType(APIChatCompletions))
	_, err := c.Generate(context.Background(), &ai.Request{Model: "gpt-4", UserPrompt: "hi"})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if strings.Contains(body, "reasoning_effort") {
		t.Fatalf("unset Chat body leaked reasoning_effort: %s", body)
	}
}

// TestOpenAI_Off_Rejected (AC3): OpenAI "off" is rejected, never mapped to
// "minimal" or omitted, and no request is dispatched.
func TestOpenAI_Off_Rejected(t *testing.T) {
	for _, apiType := range []APIType{APIResponses, APIChatCompletions} {
		hit := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
		c := NewClient("k", WithBaseURL(srv.URL), WithAPIType(apiType))
		_, err := c.Generate(context.Background(), &ai.Request{Model: "gpt-x", UserPrompt: "hi", ReasoningEffort: "off"})
		if !errors.Is(err, ai.ErrUnsupportedReasoningEffort) {
			t.Fatalf("apiType %s: off must be ErrUnsupportedReasoningEffort, got %v", apiType, err)
		}
		if hit {
			t.Fatalf("apiType %s: request dispatched for rejected off", apiType)
		}
		srv.Close()
	}
}

// TestOpenAI_Chat_NativeField_Applied (AC12): a resolved qualitative effort
// becomes the native top-level reasoning_effort field. Synthesizes the decision
// directly (capability table ships empty), matching what the resolver hands to
// BuildChatStepRequest for a registered model.
func TestOpenAI_Chat_NativeField_Applied(t *testing.T) {
	req := &ai.Request{Model: "gpt-x", Messages: []ai.Message{{Role: "user", Content: "hi"}}}
	dec := ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "high"}
	apiReq, aiErr := BuildChatStepRequest(req, dec)
	if aiErr != nil {
		t.Fatalf("BuildChatStepRequest error = %v", aiErr)
	}
	if apiReq.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", apiReq.ReasoningEffort)
	}
	raw, _ := json.Marshal(apiReq)
	if !strings.Contains(string(raw), `"reasoning_effort":"high"`) {
		t.Fatalf("native reasoning_effort not on wire: %s", raw)
	}

	// None => field omitted (byte-identical).
	apiReq2, _ := BuildChatStepRequest(req, ai.ReasoningDecision{})
	raw2, _ := json.Marshal(apiReq2)
	if strings.Contains(string(raw2), "reasoning_effort") {
		t.Fatalf("None decision must omit reasoning_effort: %s", raw2)
	}
}

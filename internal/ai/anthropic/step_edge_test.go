package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestStep_DefaultMaxTokensAndTemperature(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:       "claude-sonnet-4-5",
		Messages:    []ai.Message{{Role: "user", Content: "hi"}},
		Temperature: 0.7,
		// MaxTokens unset → must default to 4096
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if sent.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096 (default)", sent.MaxTokens)
	}
	if sent.Temperature == nil {
		t.Errorf("Temperature missing in body, want 0.7 emitted")
	} else if *sent.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", *sent.Temperature)
	}
}

func TestStep_ToolsSerialization(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Tools: []ai.ToolSchema{
			{
				Name:        "read_doc",
				Description: "Read a doc",
				Parameters:  `{"type":"object","properties":{"name":{"type":"string"}}}`,
			},
			{
				Name: "no_schema",
				// Empty Parameters → adapter must default to {"type":"object"}
			},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(sent.Tools))
	}
	if sent.Tools[0].Name != "read_doc" || sent.Tools[0].Description != "Read a doc" {
		t.Errorf("tools[0] = %+v", sent.Tools[0])
	}
	// input_schema must be a JSON object on the wire.
	if len(sent.Tools[0].InputSchema) == 0 || sent.Tools[0].InputSchema[0] != '{' {
		t.Errorf("tools[0].input_schema not an object: %s", string(sent.Tools[0].InputSchema))
	}
	// Empty Parameters → default {"type":"object"}.
	var defSchema map[string]any
	if err := json.Unmarshal(sent.Tools[1].InputSchema, &defSchema); err != nil {
		t.Fatalf("tools[1].input_schema not JSON: %v", err)
	}
	if defSchema["type"] != "object" {
		t.Errorf("tools[1].input_schema = %v, want type=object default", defSchema)
	}
}

func TestStep_InvalidToolArguments(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "id1", Name: "x", Arguments: "not-json"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed arguments, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_InvalidToolParameters(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Tools: []ai.ToolSchema{
			{Name: "bad", Parameters: "not-json"},
		},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed tool parameters, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_UnknownRole(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "moderator", Content: "no such role"}},
	})
	if err == nil {
		t.Fatal("Step() expected error for unknown role, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_EmptyToolCallArguments(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "go"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "tid", Name: "noargs", Arguments: ""},
				},
			},
			{Role: "tool", ToolCallID: "tid", Content: "done"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	asst := sent.Messages[1]
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(asst.Content, &blocks); err != nil {
		t.Fatalf("assistant content not array: %v", err)
	}
	for _, b := range blocks {
		if b.Type == "tool_use" {
			if string(b.Input) != "{}" {
				t.Errorf("empty Arguments → input = %s, want {}", string(b.Input))
			}
		}
	}
}

func TestStep_StopReason_Stop_Sequence_And_Unknown(t *testing.T) {
	tests := []struct {
		stopReason string
		wantFinish string
	}{
		{"stop_sequence", "stop"},
		{"weird_unrecognized_value", "error"},
	}
	for _, tt := range tests {
		t.Run(tt.stopReason, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := messagesResponse{
					Content: []contentBlock{
						{Type: "text", Text: "x"},
					},
					StopReason: tt.stopReason,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			resp, err := client.Step(context.Background(), &ai.Request{
				Model:    "claude-sonnet-4-5",
				Messages: []ai.Message{{Role: "user", Content: "x"}},
			})
			if err != nil {
				t.Fatalf("Step() error = %v", err)
			}
			if resp.FinishReason != tt.wantFinish {
				t.Errorf("FinishReason = %q, want %q", resp.FinishReason, tt.wantFinish)
			}
		})
	}
}

func TestStep_UserWithToolCallID_AsToolResult(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			// User role with non-empty ToolCallID — must be emitted as a
			// tool_result content array (same shape as Role="tool").
			{Role: "user", ToolCallID: "toolu_user_route", Content: "result-payload"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(sent.Messages))
	}
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(sent.Messages[0].Content, &blocks); err != nil {
		t.Fatalf("user-with-ToolCallID content not array: %v (%s)",
			err, string(sent.Messages[0].Content))
	}
	if len(blocks) != 1 || blocks[0].Type != "tool_result" {
		t.Fatalf("blocks = %+v, want a single tool_result block", blocks)
	}
	if blocks[0].ToolUseID != "toolu_user_route" {
		t.Errorf("tool_use_id = %q, want toolu_user_route", blocks[0].ToolUseID)
	}
}

func TestStep_AssistantPlainTextNoTools(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "hi"},
			// Assistant with no ToolCalls — must emit content as plain string.
			{Role: "assistant", Content: "I am thinking..."},
			{Role: "user", Content: "go on"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(sent.Messages))
	}
	asst := sent.Messages[1]
	if asst.Role != "assistant" {
		t.Fatalf("messages[1] role = %q, want assistant", asst.Role)
	}
	// Content must be a JSON string (starts with `"`), not an array.
	if len(asst.Content) == 0 || asst.Content[0] != '"' {
		t.Errorf("plain assistant content not a JSON string: %s", string(asst.Content))
	}
	var asString string
	if err := json.Unmarshal(asst.Content, &asString); err != nil {
		t.Errorf("assistant content not a string: %v", err)
	} else if asString != "I am thinking..." {
		t.Errorf("assistant content = %q, want %q", asString, "I am thinking...")
	}
}

func TestStep_MultipleTextBlocksJoined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: "part one"},
				{Type: "text", Text: "part two"},
			},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "part one\npart two" {
		t.Errorf("Text = %q, want %q (newline-joined)", resp.Text, "part one\npart two")
	}
}

func TestStep_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed JSON, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeProtocolError)
	}
}

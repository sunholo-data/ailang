package openai

import (
	"encoding/json"
	"testing"
)

func TestChatStepUsageAbsenceIsObservable(t *testing.T) {
	const (
		absentUsageBody = `{
			"choices":[{"index":0,"message":{"role":"assistant","content":null},"finish_reason":"stop"}]
		}`
		presentUsageBody = `{
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}
		}`
		toolCallBody = `{
			"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}],
			"usage":{"prompt_tokens":13,"completion_tokens":5,"total_tokens":18}
		}`
		zeroUsageBody = `{
			"choices":[{"index":0,"message":{"role":"assistant","content":null},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}
		}`
	)

	tests := []struct {
		name       string
		body       string
		usageNil   bool
		wantUsage  ChatStepUsage
		checkUsage bool
	}{
		{
			name:     "failing shape omits usage",
			body:     absentUsageBody,
			usageNil: true,
		},
		{
			name:       "healthy completion reports usage",
			body:       presentUsageBody,
			wantUsage:  ChatStepUsage{PromptTokens: 11, CompletionTokens: 7, TotalTokens: 18},
			checkUsage: true,
		},
		{
			// A future policy must not use content:null as its signal: legitimate
			// tool-call responses have null content and still report usage.
			name:       "legitimate tool call reports usage",
			body:       toolCallBody,
			wantUsage:  ChatStepUsage{PromptTokens: 13, CompletionTokens: 5, TotalTokens: 18},
			checkUsage: true,
		},
		{
			name:       "present usage may be all zero",
			body:       zeroUsageBody,
			wantUsage:  ChatStepUsage{},
			checkUsage: true,
		},
	}

	parsed := make(map[string]ChatStepResponse, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw ChatStepResponse
			if err := json.Unmarshal([]byte(tt.body), &raw); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			parsed[tt.name] = raw

			if tt.usageNil {
				if raw.Usage != nil {
					t.Fatalf("Usage = %#v, want nil", raw.Usage)
				}
				return
			}
			if raw.Usage == nil {
				t.Fatal("Usage = nil, want present usage block")
			}
			if tt.checkUsage && *raw.Usage != tt.wantUsage {
				t.Fatalf("Usage = %#v, want %#v", *raw.Usage, tt.wantUsage)
			}
		})
	}

	t.Run("omitted and present-zero usage are distinguishable", func(t *testing.T) {
		absent := parsed["failing shape omits usage"]
		presentZero := parsed["present usage may be all zero"]
		if absent.Usage != nil || presentZero.Usage == nil {
			t.Fatalf("usage presence collapsed: absent.Usage=%#v, presentZero.Usage=%#v", absent.Usage, presentZero.Usage)
		}
		if *presentZero.Usage != (ChatStepUsage{}) {
			t.Fatalf("presentZero.Usage = %#v, want all-zero usage", *presentZero.Usage)
		}
	})

	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "omitted usage", body: absentUsageBody},
		{name: "present zero usage", body: zeroUsageBody},
	} {
		t.Run("parser behavior unchanged with "+tt.name, func(t *testing.T) {
			got, aiErr := ParseChatStepResponse([]byte(tt.body), "requested-model")
			if aiErr != nil {
				t.Fatalf("ParseChatStepResponse() error = %v", aiErr)
			}
			if got.Text != "" || len(got.ToolCalls) != 0 || got.InputTokens != 0 || got.OutputTokens != 0 || got.TotalTokens != 0 {
				t.Fatalf("ParseChatStepResponse() = Text %q, %d tool calls, tokens (%d, %d, %d); want empty response with zero tokens",
					got.Text, len(got.ToolCalls), got.InputTokens, got.OutputTokens, got.TotalTokens)
			}
		})
	}
}

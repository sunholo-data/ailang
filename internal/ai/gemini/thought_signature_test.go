package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Gemini 3 signs each functionCall part and answers 400 "Function call is
// missing a thought_signature" unless the next turn sends it back on the same
// part (measured 2026-11-11, gemini-3.8-flash on Vertex, from a std/ai
// runTools loop). The adapter carries it in the opaque tool-call ID.

const testSig = "CiQBVthbYr+/sig=="

func TestThoughtSignatureRoundTrip(t *testing.T) {
	var bodies []string
	turn := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		w.Header().Set("Content-Type", "application/json")
		if turn == 0 {
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"ailang_check","args":{"code":"x"}},"thoughtSignature":"` + testSig + `"},{"functionCall":{"name":"ailang_run","args":{"code":"y"}}}]},"finishReason":"STOP"}]}`))
		} else {
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"done"}]},"finishReason":"STOP"}]}`))
		}
		turn++
	}))
	defer srv.Close()
	client := NewClient("test-key", WithBaseURL(srv.URL))

	msgs := []ai.Message{{Role: "user", Content: "go"}}
	resp, err := client.Step(context.Background(), &ai.Request{Model: "gemini-3-8-flash", Messages: msgs})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].ID != "0_0:"+testSig || resp.ToolCalls[1].ID != "0_1" {
		t.Fatalf("IDs = %q, %q: want the signature only on the signed part", resp.ToolCalls[0].ID, resp.ToolCalls[1].ID)
	}

	// The next turn sends the assistant message and both results back.
	msgs = append(msgs, ai.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
	for _, tc := range resp.ToolCalls {
		msgs = append(msgs, ai.Message{Role: "tool", ToolCallID: tc.ID, Content: "{}"})
	}
	if _, err := client.Step(context.Background(), &ai.Request{Model: "gemini-3-8-flash", Messages: msgs}); err != nil {
		t.Fatal(err)
	}
	var req struct {
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				FunctionCall     *struct{ Name string } `json:"functionCall"`
				FunctionResponse *struct{ Name string } `json:"functionResponse"`
				ThoughtSignature string                 `json:"thoughtSignature"`
			} `json:"parts"`
		} `json:"contents"`
	}
	if err := json.Unmarshal([]byte(bodies[1]), &req); err != nil {
		t.Fatal(err)
	}
	var sigs, calls, results []string
	for _, c := range req.Contents {
		for _, p := range c.Parts {
			if p.FunctionCall != nil {
				calls = append(calls, p.FunctionCall.Name)
				sigs = append(sigs, p.ThoughtSignature)
			}
			if p.FunctionResponse != nil {
				results = append(results, p.FunctionResponse.Name)
			}
		}
	}
	if strings.Join(calls, ",") != "ailang_check,ailang_run" || sigs[0] != testSig || sigs[1] != "" {
		t.Fatalf("functionCall parts %v with signatures %q: want the signature back on ailang_check only", calls, sigs)
	}
	// The results still find their calls through the full ID.
	if strings.Join(results, ",") != "ailang_check,ailang_run" {
		t.Fatalf("functionResponse names = %v", results)
	}
	if strings.Contains(bodies[1], "0_0:") {
		t.Fatal("the adapter's ID leaked onto the wire")
	}
}

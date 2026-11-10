package wire

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// These are unit tests of the wire package on its own. The byte-parity
// contract with the SDK is pinned end to end by the goldens in
// internal/ai/ollama, internal/builtins and internal/messaging; the cases
// here restate the load-bearing parts of it locally so a change shows up in
// this package's own run.

type captured struct {
	method, path, accept, ua, ctype string
	body                            string
}

func serve(t *testing.T, status int, body string) (*Client, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = captured{r.Method, r.URL.Path, r.Header.Get("Accept"), r.Header.Get("User-Agent"), r.Header.Get("Content-Type"), string(b)}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	c, err := NewClient(u, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return c, got
}

func TestGenerateStreamsChunksAndSendsSDKRequest(t *testing.T) {
	c, got := serve(t, 200, `{"response":"a","done":false}`+"\n"+`{"response":"b","done":true,"prompt_eval_count":3,"eval_count":2}`+"\n")
	var text strings.Builder
	var last Metrics
	err := c.Generate(context.Background(), &GenerateRequest{Model: "m", Prompt: "<p>", Options: map[string]any{"seed": 42}}, func(r GenerateResponse) error {
		text.WriteString(r.Response)
		last = r.Metrics
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "ab" || last.PromptEvalCount != 3 || last.EvalCount != 2 {
		t.Errorf("text=%q metrics=%+v", text.String(), last)
	}
	want := `{"model":"m","prompt":"\u003cp\u003e","suffix":"","system":"","template":"","options":{"seed":42}}`
	if got.body != want {
		t.Errorf("body\n got %s\nwant %s", got.body, want)
	}
	if got.method != "POST" || got.path != "/api/generate" || got.accept != "application/x-ndjson" || got.ctype != "application/json" || !strings.HasPrefix(got.ua, "ollama/0.0.0 (") {
		t.Errorf("request = %+v", got)
	}
}

func TestChatDecodesToolCallsAndLowercasesRole(t *testing.T) {
	c, got := serve(t, 200, `{"message":{"role":"ASSISTANT","content":"","tool_calls":[{"function":{"name":"f","arguments":{"z":1,"a":{"y":2,"x":1}}}}]},"done":true}`+"\n")
	var msg Message
	err := c.Chat(context.Background(), &ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi", Images: []ImageData{[]byte("PNG")}}}}, func(r ChatResponse) error {
		msg = r.Message
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Role != "assistant" || len(msg.ToolCalls) != 1 {
		t.Fatalf("message = %+v", msg)
	}
	if args := msg.ToolCalls[0].Function.Arguments.String(); args != `{"z":1,"a":{"x":1,"y":2}}` {
		t.Errorf("arguments = %s (document order at the top, sorted below)", args)
	}
	if want := `{"model":"m","messages":[{"role":"user","content":"hi","images":["UE5H"]}],"options":null}`; got.body != want {
		t.Errorf("body\n got %s\nwant %s", got.body, want)
	}
}

func TestStreamErrors(t *testing.T) {
	long := `{"response":"` + strings.Repeat("a", maxLineBytes) + `"}`
	cases := []struct {
		name, body, want string
		status           int
	}{
		{"status json", `{"error":"nope"}`, "404 Not Found: nope", 404},
		{"status text", "plain", "500 Internal Server Error: plain", 500},
		{"unauthorized", `{"error":"x","signin_url":"u"}`, "401 Unauthorized", 401},
		{"error line", `{"response":"a"}` + "\n" + `{"error":"boom"}`, "boom", 200},
		{"garbage line", "not json", "not json", 200},
		{"blank line", "\n", "", 200},
		{"wrong type", `{"response":5}`, "json: cannot unmarshal number into Go struct field GenerateResponse.response of type string", 200},
		{"too long", long, "bufio.Scanner: token too long", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := serve(t, tc.status, tc.body)
			err := c.Generate(context.Background(), &GenerateRequest{}, func(GenerateResponse) error { return nil })
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
	t.Run("empty body is success", func(t *testing.T) {
		c, _ := serve(t, 500, "")
		if err := c.Chat(context.Background(), &ChatRequest{}, func(ChatResponse) error { return nil }); err != nil {
			t.Fatalf("err = %v; the SDK returned nil for a body with no lines", err)
		}
	})
	t.Run("callback error stops the stream", func(t *testing.T) {
		c, _ := serve(t, 200, `{"done":false}`+"\n"+`{"done":true}`)
		stop := errors.New("stop")
		n := 0
		err := c.Chat(context.Background(), &ChatRequest{}, func(ChatResponse) error { n++; return stop })
		if !errors.Is(err, stop) || n != 1 {
			t.Fatalf("err=%v n=%d", err, n)
		}
	})
}

func TestBufferedCalls(t *testing.T) {
	t.Run("embed", func(t *testing.T) {
		c, got := serve(t, 200, `{"model":"e","embeddings":[[0.5,1]]}`)
		r, err := c.Embed(context.Background(), &EmbedRequest{Model: "e", Input: "x"})
		if err != nil || len(r.Embeddings) != 1 || r.Embeddings[0][1] != 1 {
			t.Fatalf("r=%+v err=%v", r, err)
		}
		if got.body != `{"model":"e","input":"x","options":null}` || got.accept != "application/json" {
			t.Errorf("request = %+v", got)
		}
	})
	t.Run("list", func(t *testing.T) {
		c, got := serve(t, 200, `{"models":[{"name":"q","modified_at":"2026-01-01T00:00:00Z"}]}`)
		r, err := c.List(context.Background())
		if err != nil || len(r.Models) != 1 || r.Models[0].Name != "q" {
			t.Fatalf("r=%+v err=%v", r, err)
		}
		if got.method != "GET" || got.path != "/api/tags" || got.body != "" {
			t.Errorf("request = %+v", got)
		}
	})
	cases := []struct {
		name, body, want string
		status           int
	}{
		{"json error", `{"error":"boom"}`, "boom", 500},
		{"status key fills Status", `{"error":"e","status":"odd"}`, "odd: e", 500},
		{"text error", "plain", "plain", 500},
		{"empty error", "", "something went wrong, please see the ollama server logs for details", 400},
		{"unauthorized", `{"error":"x"}`, "something went wrong, please see the ollama server logs for details", 401},
		{"unauthorized status", `{"Status":"401 nope"}`, "401 nope", 401},
		{"bad json", "<html>", "invalid character '<' looking for beginning of value", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := serve(t, tc.status, tc.body)
			_, err := c.Embed(context.Background(), &EmbedRequest{})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
	t.Run("empty 200 is an empty reply", func(t *testing.T) {
		c, _ := serve(t, 200, "")
		r, err := c.Embed(context.Background(), &EmbedRequest{})
		if err != nil || len(r.Embeddings) != 0 {
			t.Fatalf("r=%+v err=%v", r, err)
		}
	})
}

func TestToolSchemaNormalisation(t *testing.T) {
	in := `{"$schema":"x","type":"object","properties":{"b":{"type":["string","null"],"enum":[1,{"z":1,"a":2}],"minimum":0},"a":{"type":"object","properties":{"y":{"type":"string"},"x":{"anyOf":[{"type":"number"}]}}},"b":{"type":"integer"}},"required":["b"],"additionalProperties":false}`
	var p ToolFunctionParameters
	if err := json.Unmarshal([]byte(in), &p); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(Tool{Type: "function", Function: ToolFunction{Name: "f", Parameters: p}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"function","function":{"name":"f","parameters":{"type":"object","required":["b"],"properties":{"b":{"type":"integer"},"a":{"type":"object","properties":{"y":{"type":"string"},"x":{"anyOf":[{"type":"number"}]}}}}}}}`
	if string(out) != want {
		t.Errorf("\n got %s\nwant %s", out, want)
	}
	var empty ToolFunctionParameters
	if b, _ := json.Marshal(empty); string(b) != `{"type":"","properties":null}` {
		t.Errorf("empty parameters = %s", b)
	}
	var bad ToolFunctionParameters
	err = json.Unmarshal([]byte(`{"properties":{"x":{"type":5}}}`), &bad)
	if err == nil || !strings.Contains(err.Error(), "ToolFunctionParameters.properties.type of type []string") {
		t.Errorf("err = %v", err)
	}
}

func TestOrderedMapEdges(t *testing.T) {
	var a ToolCallFunctionArguments
	err := json.Unmarshal([]byte(`"{\"a\":1}"`), &a)
	if err == nil || err.Error() != errMalformedObject.Error() {
		t.Fatalf("string arguments: err = %v", err)
	}
	if got := a.String(); got != "{}" {
		t.Errorf("after a failed decode String() = %q, want {}", got)
	}
	var keys ToolCallFunctionArguments
	if err := json.Unmarshal([]byte(`{"k":1,"j":2,"k":3,"tab\tkey":1,"bs\bkey":2,"ls ":3,"<&>":4}`), &keys); err != nil {
		t.Fatal(err)
	}
	want := `{"k":3,"j":2,"tab\tkey":1,"bs\u0008key":2,"ls\u2028":3,"\u003c\u0026\u003e":4}`
	if got := keys.String(); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
	var buf strings.Builder
	m := &orderedMap[int]{}
	m.set("bad\xffkey\r\n\"\\", 1)
	b, _ := m.MarshalJSON()
	buf.Write(b)
	if got := buf.String(); got != `{"bad�key\r\n\"\\":1}` {
		t.Errorf("escaping = %s", got)
	}
	var nested ToolPropertiesMap
	if err := json.Unmarshal([]byte(`{"x":{"type":5}}`), &nested); err == nil {
		t.Error("a property that does not decode must fail")
	}
	var pt PropertyType
	if err := json.Unmarshal([]byte(`5`), &pt); err == nil {
		t.Error("a numeric type must fail")
	}
}

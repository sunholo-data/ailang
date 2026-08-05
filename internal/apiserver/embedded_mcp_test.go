package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type embeddedTestSession struct{ name string }

type embeddedTestHost struct {
	resolve func(context.Context, *http.Request) (any, error)
	tools   func(context.Context, any) ([]ToolDescriptor, error)
	invoke  func(context.Context, any, string, json.RawMessage) (json.RawMessage, error)
}

func embeddedHandler(t *testing.T, host embeddedTestHost, timeout time.Duration, capacity int) http.Handler {
	t.Helper()
	runner, err := NewCallbackRunner(timeout, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return NewEmbeddedMCPHandler(EmbeddedMCPConfig{
		AgentName: "embedded-test", AgentVersion: "1", Runner: runner,
		Resolve: host.resolve, Tools: host.tools, Invoke: host.invoke,
	})
}

func objectTool(name string) ToolDescriptor {
	return ToolDescriptor{Name: name, Description: name, InputSchema: json.RawMessage(`{"type":"object"}`)}
}

func mcpPost(handler http.Handler, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	handler.ServeHTTP(recorder, request)
	return recorder
}

func mcpResult(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := recorder.Body.String()
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") {
			body = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("response is not JSON: %v; body=%q", err, recorder.Body.String())
	}
	return result
}

func embeddedToolNames(t *testing.T, result map[string]any) []string {
	t.Helper()
	value, ok := result["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", result)
	}
	items, ok := value["tools"].([]any)
	if !ok {
		t.Fatalf("missing tools: %#v", result)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.(map[string]any)["name"].(string))
	}
	sort.Strings(names)
	return names
}

func listRequest(id int) string {
	return `{"jsonrpc":"2.0","id":` + jsonNumber(id) + `,"method":"tools/list","params":{}}`
}

func jsonNumber(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestEmbeddedMCPExactRequestLocalSurfaces(t *testing.T) {
	sessionA, sessionB := &embeddedTestSession{"A"}, &embeddedTestSession{"B"}
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) {
			if r.Header.Get("X-Session") == "B" {
				return sessionB, nil
			}
			return sessionA, nil
		},
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			if session == sessionB {
				return []ToolDescriptor{objectTool("shared"), objectTool("beta_only")}, nil
			}
			return []ToolDescriptor{objectTool("shared"), objectTool("alpha_only")}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	}
	handler := embeddedHandler(t, host, time.Second, 8)
	request := func(session string) []string {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(listRequest(1)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("X-Session", session)
		handler.ServeHTTP(r, req)
		return embeddedToolNames(t, mcpResult(t, r))
	}
	a, b := request("A"), request("B")
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("instrument failure")
	}
	if strings.Join(a, ",") != "alpha_only,shared" || strings.Join(b, ",") != "beta_only,shared" {
		t.Fatalf("surfaces A=%v B=%v", a, b)
	}
	if strings.Contains(strings.Join(a, ","), "beta_only") || strings.Contains(strings.Join(b, ","), "alpha_only") {
		t.Fatal("foreign sentinel leaked")
	}
}

func TestEmbeddedMCPDispatchAuthorizationAndSessionIdentity(t *testing.T) {
	sessionA, sessionB := &embeddedTestSession{"A"}, &embeddedTestSession{"B"}
	var calls atomic.Int32
	var gotSession any
	var gotArgs json.RawMessage
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) {
			if r.Header.Get("X-Session") == "B" {
				return sessionB, nil
			}
			return sessionA, nil
		},
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			if session == sessionA {
				return []ToolDescriptor{objectTool("alpha_only")}, nil
			}
			return []ToolDescriptor{objectTool("beta_only")}, nil
		},
		invoke: func(_ context.Context, session any, _ string, args json.RawMessage) (json.RawMessage, error) {
			calls.Add(1)
			gotSession = session
			gotArgs = append(json.RawMessage(nil), args...)
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
	handler := embeddedHandler(t, host, time.Second, 8)
	call := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"alpha_only","arguments":{"nonce":"A-137"}}}`
	request := func(session string) map[string]any {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(call))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("X-Session", session)
		handler.ServeHTTP(r, req)
		return mcpResult(t, r)
	}
	request("A")
	if calls.Load() != 1 || gotSession != sessionA || gotSession == sessionB || !bytes.Equal(gotArgs, []byte(`{"nonce":"A-137"}`)) {
		t.Fatalf("dispatch calls=%d session=%p args=%s", calls.Load(), gotSession, gotArgs)
	}
	calls.Store(0)
	result := request("B")
	if calls.Load() != 0 || result["error"] == nil {
		t.Fatalf("unauthorized dispatch result=%#v calls=%d", result, calls.Load())
	}
}

func TestEmbeddedMCPSubmitFeedbackIsCallerSupplied(t *testing.T) {
	var enabled atomic.Bool
	var calls atomic.Int32
	host := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return "session", nil },
		tools: func(context.Context, any) ([]ToolDescriptor, error) {
			if enabled.Load() {
				return []ToolDescriptor{objectTool("submit_feedback")}, nil
			}
			return nil, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			calls.Add(1)
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
	handler := embeddedHandler(t, host, time.Second, 8)
	if names := embeddedToolNames(t, mcpResult(t, mcpPost(handler, listRequest(1)))); len(names) != 0 {
		t.Fatalf("ambient tools: %v", names)
	}
	enabled.Store(true)
	if names := embeddedToolNames(t, mcpResult(t, mcpPost(handler, listRequest(2)))); strings.Join(names, ",") != "submit_feedback" {
		t.Fatalf("supplied tools: %v", names)
	}
	mcpPost(handler, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"submit_feedback","arguments":{}}}`)
	if calls.Load() != 1 {
		t.Fatalf("invoke calls=%d", calls.Load())
	}
}

func TestEmbeddedMCPStatelessTransport(t *testing.T) {
	host := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return "s", nil },
		tools: func(context.Context, any) ([]ToolDescriptor, error) {
			return []ToolDescriptor{objectTool("sentinel")}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	}
	handler := embeddedHandler(t, host, time.Second, 8)
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/mcp/", nil))
	if get.Code != http.StatusMethodNotAllowed || get.Header().Get("Allow") != "POST" || strings.TrimSpace(get.Body.String()) != "Method Not Allowed" {
		t.Fatalf("GET status=%d allow=%q body=%q", get.Code, get.Header().Get("Allow"), get.Body.String())
	}
	post := mcpPost(handler, listRequest(4))
	if post.Code != http.StatusOK || !strings.HasPrefix(post.Header().Get("Content-Type"), "text/event-stream") || !strings.Contains(post.Body.String(), "event: message") {
		t.Fatalf("POST status=%d content-type=%q body=%q", post.Code, post.Header().Get("Content-Type"), post.Body.String())
	}
	if names := embeddedToolNames(t, mcpResult(t, post)); strings.Join(names, ",") != "sentinel" {
		t.Fatalf("tools=%v", names)
	}
}

func TestEmbeddedMCPPanicSafetyAndDescriptorValidation(t *testing.T) {
	var schema atomic.Value
	schema.Store(json.RawMessage(`{"type":"object"}`))
	var calls atomic.Int32
	host := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return "s", nil },
		tools: func(context.Context, any) ([]ToolDescriptor, error) {
			return []ToolDescriptor{{Name: "tool", InputSchema: schema.Load().(json.RawMessage)}}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			calls.Add(1)
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
	handler := embeddedHandler(t, host, time.Second, 8)
	invalid := []struct {
		schema  json.RawMessage
		wantMsg string
	}{
		{nil, "input schema is required"},
		{json.RawMessage(`"scalar"`), "invalid input schema"},
		{json.RawMessage(`{"type":"object","properties":{"bad":{"type":"array","x-mcp-header":"Bad Header"}}}`), "invalid parameter header annotations"},
	}
	for i, tc := range invalid {
		schema.Store(tc.schema)
		r := mcpPost(handler, listRequest(10+i))
		result := mcpResult(t, r)
		if r.Code != http.StatusOK || result["error"] == nil || calls.Load() != 0 {
			t.Fatalf("invalid %d: status=%d result=%#v calls=%d", i, r.Code, result, calls.Load())
		}
		// The message must be the gateway's LOUD rejection, not the recover()
		// backstop ("host tool registration failed") — the backstop alone would
		// silently accept a descriptor class the contract calls invalid.
		message, _ := result["error"].(map[string]any)["message"].(string)
		if !strings.Contains(message, tc.wantMsg) || strings.Contains(message, "host tool registration failed") {
			t.Fatalf("invalid %d: message=%q want substring %q from gateway validation", i, message, tc.wantMsg)
		}
	}
	schema.Store(json.RawMessage(`{"type":"object"}`))
	good := mcpPost(handler, `{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"tool","arguments":{}}}`)
	if good.Code != http.StatusOK || mcpResult(t, good)["result"] == nil || calls.Load() != 1 {
		t.Fatalf("good status=%d body=%q calls=%d", good.Code, good.Body.String(), calls.Load())
	}
}

type testAuthorizationError struct{ status int }

func (e testAuthorizationError) Error() string   { return "denied" }
func (e testAuthorizationError) HTTPStatus() int { return e.status }

func TestEmbeddedMCPFrozenCallbackEnvelopes(t *testing.T) {
	block := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	for _, stage := range []string{"resolve", "tools", "invoke"} {
		bodies := []string{
			`{"jsonrpc":"2.0","id":"echo-me","method":"tools/call","params":{"name":"tool","arguments":{}}}`,
		}
		if stage == "resolve" {
			bodies = append(bodies, `not-json`)
		}
		for _, body := range bodies {
			var next atomic.Int32
			host := embeddedTestHost{
				resolve: func(ctx context.Context, _ *http.Request) (any, error) {
					if stage == "resolve" {
						return nil, block(ctx)
					}
					next.Add(1)
					return "s", nil
				},
				tools: func(ctx context.Context, _ any) ([]ToolDescriptor, error) {
					if stage == "tools" {
						return nil, block(ctx)
					}
					next.Add(1)
					return []ToolDescriptor{objectTool("tool")}, nil
				},
				invoke: func(ctx context.Context, _ any, _ string, _ json.RawMessage) (json.RawMessage, error) {
					if stage == "invoke" {
						return nil, block(ctx)
					}
					next.Add(1)
					return nil, nil
				},
			}
			start := time.Now()
			r := mcpPost(embeddedHandler(t, host, 20*time.Millisecond, 4), body)
			elapsed := time.Since(start)
			result := mcpResult(t, r)
			protocolError := result["error"].(map[string]any)
			if r.Code == 500 || r.Code == 504 || r.Code != 200 || r.Header().Get("Content-Type") != "application/json" || protocolError["code"] != float64(-32603) || protocolError["message"] != "host callback timed out" || elapsed < 20*time.Millisecond || elapsed >= 250*time.Millisecond {
				t.Fatalf("stage=%s status=%d elapsed=%v result=%#v", stage, r.Code, elapsed, result)
			}
			if body == "not-json" && result["id"] != nil {
				t.Fatalf("malformed id=%#v", result["id"])
			}
			if body != "not-json" && result["id"] != "echo-me" {
				t.Fatalf("echoed id=%#v", result["id"])
			}
			wantNext := int32(0)
			if stage == "tools" {
				wantNext = 1
			}
			if stage == "invoke" {
				wantNext = 2
			}
			if next.Load() != wantNext {
				t.Fatalf("stage=%s next=%d want=%d", stage, next.Load(), wantNext)
			}
		}
	}
	canceled := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return nil, context.Canceled },
		tools:   func(context.Context, any) ([]ToolDescriptor, error) { return nil, nil },
		invoke:  func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	}
	if message := mcpResult(t, mcpPost(embeddedHandler(t, canceled, time.Second, 4), listRequest(1)))["error"].(map[string]any)["message"]; message != "host callback canceled" {
		t.Fatalf("message=%v", message)
	}
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		auth := canceled
		auth.resolve = func(context.Context, *http.Request) (any, error) { return nil, testAuthorizationError{status} }
		r := mcpPost(embeddedHandler(t, auth, time.Second, 4), listRequest(1))
		if r.Code != status || r.Code == 500 || r.Code == 504 {
			t.Fatalf("authorization status=%d want=%d", r.Code, status)
		}
	}
}

func TestEmbeddedMCPOverloadEnvelopeAndFastControl(t *testing.T) {
	release := make(chan struct{})
	blocking := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { <-release; return "s", nil },
		tools:   func(context.Context, any) ([]ToolDescriptor, error) { return nil, nil },
		invoke:  func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) { return nil, nil },
	}
	handler := embeddedHandler(t, blocking, 30*time.Millisecond, 4)
	var overload atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := mcpResult(t, mcpPost(handler, listRequest(id)))
			if e, ok := result["error"].(map[string]any); ok && e["message"] == "host callback capacity exceeded" && e["code"] == float64(-32603) {
				overload.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if overload.Load() == 0 {
		t.Fatal("no overload envelopes")
	}
	close(release)
	fast := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return "s", nil },
		tools:   func(context.Context, any) ([]ToolDescriptor, error) { return nil, nil },
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("unused")
		},
	}
	fastHandler := embeddedHandler(t, fast, time.Second, 4)
	overload.Store(0)
	wg = sync.WaitGroup{}
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := mcpResult(t, mcpPost(fastHandler, listRequest(id)))
			if e, ok := result["error"].(map[string]any); ok && e["message"] == "host callback capacity exceeded" {
				overload.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if overload.Load() != 0 {
		t.Fatalf("fast overloads=%d", overload.Load())
	}
}

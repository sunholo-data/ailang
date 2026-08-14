package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

const exitFixturePrefix = "internal/embed/testdata/"

func newExitHandlerTestServer(t *testing.T) *Server {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}
	t.Setenv("AILANG_STDLIB_PATH", root)

	effCtx := effects.NewEffContext(nil)
	effCtx.Grant(effects.NewCapability("IO"))
	srv := New(root, Config{EffCtx: effCtx})
	t.Cleanup(func() { _ = srv.Close() })

	for _, fixture := range []string{"exit_zero.ail", "exit_nonzero.ail", "no_exit.ail"} {
		if err := srv.LoadModules([]string{filepath.Join(root, exitFixturePrefix, fixture)}); err != nil {
			t.Fatalf("LoadModules(%s): %v", fixture, err)
		}
	}
	return srv
}

func callRouteFixture(t *testing.T, srv *Server, module string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/"+module+"/main", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.callFunction(w, req, module, "main")

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decoding route response %q: %v", w.Body.String(), err)
	}
	return w.Code, body
}

func callA2AFixture(t *testing.T, srv *Server, module string) map[string]any {
	t.Helper()
	skillID := strings.ReplaceAll(module, "/", ".") + ".main"
	body := `{"jsonrpc":"2.0","id":1,"method":"tasks/send","params":{"id":"exit-test","metadata":{"skill_id":"` + skillID + `"},"message":{"parts":[]}}}`
	req := httptest.NewRequest(http.MethodPost, "/a2a/", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleA2ATask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("A2A HTTP status = %d, want 200: %s", w.Code, w.Body.String())
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decoding A2A response %q: %v", w.Body.String(), err)
	}
	return response
}

func a2aTaskState(t *testing.T, response map[string]any) string {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("A2A response has no task result: %#v", response)
	}
	status, ok := result["status"].(map[string]any)
	if !ok {
		t.Fatalf("A2A task has no status: %#v", result)
	}
	state, ok := status["state"].(string)
	if !ok {
		t.Fatalf("A2A task has no string state: %#v", status)
	}
	return state
}

func TestRoutesDispatchExitZeroSucceeds(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	status, body := callRouteFixture(t, srv, exitFixturePrefix+"exit_zero")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %#v", status, body)
	}
	if _, present := body["error"]; present {
		t.Fatalf("clean exit response contains error: %#v", body)
	}
}

func TestRoutesDispatchExitNonzeroFails(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	status, body := callRouteFixture(t, srv, exitFixturePrefix+"exit_nonzero")
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %#v", status, body)
	}
	if body["error"] != "program called exit(1)" {
		t.Fatalf("error = %#v, want unchanged exit(1) message", body["error"])
	}
}

func TestA2AExitZeroSucceeds(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	response := callA2AFixture(t, srv, exitFixturePrefix+"exit_zero")
	if state := a2aTaskState(t, response); state != "completed" {
		t.Fatalf("task state = %q, want completed: %#v", state, response)
	}
}

func TestA2AExitNonzeroFails(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	response := callA2AFixture(t, srv, exitFixturePrefix+"exit_nonzero")
	if state := a2aTaskState(t, response); state != "failed" {
		t.Fatalf("task state = %q, want failed: %#v", state, response)
	}
}

func TestMCPExitZeroSucceeds(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	handler := NewMCPServer(srv).makeToolHandler(exitFixturePrefix+"exit_zero", "main", nil)
	res, err := handler(context.Background(), mcpCallReq("main", `{}`))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("clean exit returned MCP error: %s", mcpResultText(t, res))
	}
	if text := mcpResultText(t, res); text != "null" {
		t.Fatalf("clean exit result = %q, want unit-shaped null", text)
	}
}

func TestMCPExitNonzeroFails(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	handler := NewMCPServer(srv).makeToolHandler(exitFixturePrefix+"exit_nonzero", "main", nil)
	res, err := handler(context.Background(), mcpCallReq("main", `{}`))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("exit(1) returned normal MCP result: %s", mcpResultText(t, res))
	}
	if text := mcpResultText(t, res); !strings.Contains(text, "function call failed: program called exit(1)") {
		t.Fatalf("MCP error changed: %q", text)
	}
}

func TestExitHandlerPositiveControl(t *testing.T) {
	srv := newExitHandlerTestServer(t)
	status, body := callRouteFixture(t, srv, exitFixturePrefix+"no_exit")
	if status != http.StatusOK {
		t.Fatalf("no-exit control status = %d, want 200: %#v", status, body)
	}
	if body["result"] != float64(42) {
		t.Fatalf("no-exit control result = %#v, want 42", body["result"])
	}
}

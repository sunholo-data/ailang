package apiserver

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func sendA2ATestTask(t *testing.T, srv *Server, skillID string) map[string]any {
	t.Helper()
	body := `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"tasks/send",
		"params":{
			"metadata":{"skill_id":"` + skillID + `"},
			"message":{"role":"user","parts":[{"type":"data","data":{"args":[]}}]}
		}
	}`
	req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.buildRoutes().ServeHTTP(w, req)
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	return resp
}

func requireA2ANotFound(t *testing.T, resp map[string]any) {
	t.Helper()
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("hidden function reached dispatch; response: %#v", resp)
	}
	if got := errObj["code"]; got != float64(-32602) {
		t.Fatalf("error code = %v, want -32602; response: %#v", got, resp)
	}
	if msg, _ := errObj["message"].(string); !strings.Contains(msg, "not found in module") {
		t.Errorf("error message = %q, want existing indistinguishable not-found shape", msg)
	}
}

func TestA2ADispatch_RespectsRoutesOnly(t *testing.T) {
	srv := feedbackSurfaceServer(t, Config{RoutesOnly: true, A2A: true})
	defer srv.Close()

	hidden := sendA2ATestTask(t, srv, "test.api.surface.helper")
	requireA2ANotFound(t, hidden)

	visible := sendA2ATestTask(t, srv, "test.api.surface.status")
	result, ok := visible["result"].(map[string]any)
	if !ok {
		t.Fatalf("positive control route did not dispatch: %#v", visible)
	}
	status, ok := result["status"].(map[string]any)
	if !ok || status["state"] != "completed" {
		t.Fatalf("positive control route did not complete: %#v", visible)
	}
}

func TestA2ADispatch_RespectsNoExpose(t *testing.T) {
	srv := nomcpTestServer(t)
	defer srv.Close()
	srv.a2aEnabled = true

	hidden := sendA2ATestTask(t, srv, "test.api.keys.internalSecret")
	requireA2ANotFound(t, hidden)

	visible := sendA2ATestTask(t, srv, "test.api.keys.status")
	result, ok := visible["result"].(map[string]any)
	if !ok {
		t.Fatalf("positive control route did not dispatch: %#v", visible)
	}
	status, ok := result["status"].(map[string]any)
	if !ok || status["state"] != "completed" {
		t.Fatalf("positive control route did not complete: %#v", visible)
	}
}

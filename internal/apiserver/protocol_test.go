package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/apiserver/schema"
)

// --- OpenAPI Tests ---

func TestOpenAPISpec(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("GET", "/api/_meta/openapi.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatal(err)
	}

	// Check OpenAPI version
	if spec["openapi"] != "3.1.0" {
		t.Errorf("expected openapi 3.1.0, got %v", spec["openapi"])
	}

	// Check info block
	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info object")
	}
	if info["title"] != "AILANG API" {
		t.Errorf("expected title 'AILANG API', got %v", info["title"])
	}

	// Check paths exist
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths object")
	}

	// Should have paths for hello and add functions
	helloPath := paths["/api/test/api/greet/hello"]
	if helloPath == nil {
		t.Error("expected path for /api/test/api/greet/hello")
	}
	addPath := paths["/api/test/api/greet/add"]
	if addPath == nil {
		t.Error("expected path for /api/test/api/greet/add")
	}

	// Should have meta endpoints
	if paths["/api/_meta/modules"] == nil {
		t.Error("expected path for /api/_meta/modules")
	}
	if paths["/api/_health"] == nil {
		t.Error("expected path for /api/_health")
	}
}

func TestOpenAPISpecMethodNotAllowed(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("POST", "/api/_meta/openapi.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestOpenAPISpecPureAnnotation(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	spec := srv.buildOpenAPISpec()
	paths := spec["paths"].(map[string]any)

	// Both hello and add are pure in the test module
	for _, pathKey := range []string{"/api/test/api/greet/hello", "/api/test/api/greet/add"} {
		pathObj, ok := paths[pathKey].(map[string]any)
		if !ok {
			t.Fatalf("expected path object for %s", pathKey)
		}
		postObj, ok := pathObj["post"].(map[string]any)
		if !ok {
			t.Fatalf("expected post object for %s", pathKey)
		}
		if postObj["x-ailang-pure"] != true {
			t.Errorf("expected x-ailang-pure=true for %s", pathKey)
		}
	}
}

// --- A2A Tests ---

func TestA2AAgentCard(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var card map[string]any
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatal(err)
	}

	if card["name"] != "AILANG Function Server" {
		t.Errorf("expected name 'AILANG Function Server', got %v", card["name"])
	}

	// Check skills exist
	skills, ok := card["skills"].([]any)
	if !ok {
		t.Fatal("expected skills array")
	}
	if len(skills) < 2 {
		t.Errorf("expected at least 2 skills, got %d", len(skills))
	}

	// Check skill IDs are dot-separated
	for _, s := range skills {
		skill := s.(map[string]any)
		id := skill["id"].(string)
		if strings.Contains(id, "/") {
			t.Errorf("skill ID should not contain '/', got %q", id)
		}
	}
}

func TestA2AAgentCardMethodNotAllowed(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("POST", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestA2ATaskSend(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	t.Run("missing skill_id", func(t *testing.T) {
		body := `{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "tasks/send",
			"params": {
				"message": {"role": "user", "parts": []}
			}
		}`
		req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 (JSON-RPC error), got %d", w.Code)
		}

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)

		errObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatal("expected JSON-RPC error object")
		}
		msg := errObj["message"].(string)
		if !strings.Contains(msg, "skill_id") {
			t.Errorf("expected error about skill_id, got %q", msg)
		}
	})

	t.Run("invalid skill_id format", func(t *testing.T) {
		body := `{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tasks/send",
			"params": {
				"metadata": {"skill_id": "nope"},
				"message": {"role": "user", "parts": []}
			}
		}`
		req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)

		if resp["error"] == nil {
			t.Fatal("expected JSON-RPC error for invalid skill_id")
		}
	})

	t.Run("unknown module", func(t *testing.T) {
		body := `{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "tasks/send",
			"params": {
				"metadata": {"skill_id": "no.such.module.func"},
				"message": {"role": "user", "parts": []}
			}
		}`
		req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)

		if resp["error"] == nil {
			t.Fatal("expected JSON-RPC error for unknown module")
		}
	})
}

func TestA2ATaskGet(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	body := `{
		"jsonrpc": "2.0",
		"id": 10,
		"method": "tasks/get",
		"params": {"id": "task-123"}
	}`
	req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	// tasks/get returns error since all tasks are synchronous
	if resp["error"] == nil {
		t.Fatal("expected JSON-RPC error for tasks/get")
	}
}

func TestA2AUnknownMethod(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	body := `{
		"jsonrpc": "2.0",
		"id": 20,
		"method": "unknown/method",
		"params": {}
	}`
	req := httptest.NewRequest("POST", "/a2a/", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != -32601 {
		t.Errorf("expected code -32601, got %v", errObj["code"])
	}
}

func TestA2AInvalidJSON(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	req := httptest.NewRequest("POST", "/a2a/", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != -32700 {
		t.Errorf("expected code -32700 (parse error), got %v", errObj["code"])
	}
}

func TestA2AMethodNotAllowed(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	req := httptest.NewRequest("GET", "/a2a/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// --- MCP Input Schema Tests ---

func TestBuildMCPInputSchema(t *testing.T) {
	t.Run("zero arity", func(t *testing.T) {
		fs := &schema.FunctionSchema{Arity: 0}
		s := buildMCPInputSchema(fs)

		if s["type"] != "object" {
			t.Errorf("expected type 'object', got %v", s["type"])
		}
		props := s["properties"].(map[string]any)
		if len(props) != 0 {
			t.Errorf("expected empty properties, got %v", props)
		}
	})

	t.Run("with parameters", func(t *testing.T) {
		fs := schema.FromTypeString("int -> string -> bool")
		s := buildMCPInputSchema(fs)

		if s["type"] != "object" {
			t.Errorf("expected type 'object', got %v", s["type"])
		}

		required, ok := s["required"].([]string)
		if !ok || len(required) != 1 || required[0] != "args" {
			t.Errorf("expected required=[args], got %v", s["required"])
		}

		props := s["properties"].(map[string]any)
		argsSchema := props["args"].(map[string]any)
		if argsSchema["type"] != "array" {
			t.Errorf("expected args type 'array', got %v", argsSchema["type"])
		}
		if argsSchema["minItems"] != 2 {
			t.Errorf("expected minItems 2, got %v", argsSchema["minItems"])
		}
	})
}

func TestMCPError(t *testing.T) {
	result := mcpError("something went wrong")
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
}

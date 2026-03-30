package apiserver

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/eval"
)

func TestExtractRouteAnnotations(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/api",
		Exports: []ExportInfo{
			{Name: "parse", Type: "string -> string", Arity: 1},
			{Name: "health", Type: "string", Arity: 0, Pure: true},
			{Name: "helper", Type: "int -> int", Arity: 1},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "parse",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "POST"},
							&ast.Literal{Kind: ast.StringLit, Value: "/api/v1/parse"},
						},
					},
				},
			},
			{
				Name:     "health",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/health"},
						},
					},
				},
			},
			{
				Name:     "helper",
				IsExport: true,
				// No @route annotation
			},
		},
	}

	extractRouteAnnotations(modInfo, file)

	// parse should have route
	if modInfo.Exports[0].RouteMethod != "POST" {
		t.Errorf("expected parse route method 'POST', got %q", modInfo.Exports[0].RouteMethod)
	}
	if modInfo.Exports[0].RoutePath != "/api/v1/parse" {
		t.Errorf("expected parse route path '/api/v1/parse', got %q", modInfo.Exports[0].RoutePath)
	}

	// health should have route
	if modInfo.Exports[1].RouteMethod != "GET" {
		t.Errorf("expected health route method 'GET', got %q", modInfo.Exports[1].RouteMethod)
	}
	if modInfo.Exports[1].RoutePath != "/health" {
		t.Errorf("expected health route path '/health', got %q", modInfo.Exports[1].RoutePath)
	}

	// helper should NOT have route
	if modInfo.Exports[2].RouteMethod != "" || modInfo.Exports[2].RoutePath != "" {
		t.Errorf("expected helper to have no route, got %q %q", modInfo.Exports[2].RouteMethod, modInfo.Exports[2].RoutePath)
	}
}

func TestExtractRouteAnnotations_Raw(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/webhooks",
		Exports: []ExportInfo{
			{Name: "handleStripe", Type: "HttpRequest -> Result", Arity: 1},
			{Name: "handleNormal", Type: "string -> string", Arity: 1},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "handleStripe",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{Name: "raw"},
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "POST"},
							&ast.Literal{Kind: ast.StringLit, Value: "/webhooks/stripe"},
						},
					},
				},
			},
			{
				Name:     "handleNormal",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "POST"},
							&ast.Literal{Kind: ast.StringLit, Value: "/api/normal"},
						},
					},
				},
			},
		},
	}

	extractRouteAnnotations(modInfo, file)

	if !modInfo.Exports[0].IsRaw {
		t.Error("expected handleStripe to have IsRaw=true")
	}
	if modInfo.Exports[0].RouteMethod != "POST" {
		t.Errorf("expected POST, got %q", modInfo.Exports[0].RouteMethod)
	}
	if modInfo.Exports[0].RoutePath != "/webhooks/stripe" {
		t.Errorf("expected /webhooks/stripe, got %q", modInfo.Exports[0].RoutePath)
	}
	if modInfo.Exports[1].IsRaw {
		t.Error("expected handleNormal to have IsRaw=false")
	}
}

func TestExtractRouteAnnotations_Nowrap(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/api",
		Exports: []ExportInfo{
			{Name: "agentCard", Type: "record -> record", Arity: 0},
			{Name: "status", Type: "string -> string", Arity: 0},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "agentCard",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{Name: "nowrap"},
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/.well-known/agent.json"},
						},
					},
				},
			},
			{
				Name:     "status",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/status"},
						},
					},
				},
			},
		},
	}

	extractRouteAnnotations(modInfo, file)

	if !modInfo.Exports[0].IsNowrap {
		t.Error("expected agentCard to have IsNowrap=true")
	}
	if modInfo.Exports[1].IsNowrap {
		t.Error("expected status to have IsNowrap=false")
	}
}

func TestNowrapResponse(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Call the "hello" function with nowrap option — should return raw JSON
	req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call directly with nowrap
	srv.callFunction(w, req, "test/api/greet", "hello", callOpts{Nowrap: true})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response should be raw string, not wrapped in FunctionCallResponse
	body := strings.TrimSpace(w.Body.String())
	if body != `"Hello, World!"` {
		t.Errorf("expected raw JSON string, got %s", body)
	}

	// Should not contain envelope fields
	if strings.Contains(body, "module") || strings.Contains(body, "elapsed_ms") {
		t.Errorf("expected no envelope, got %s", body)
	}

	// Should have timing header
	if w.Header().Get("X-Elapsed-Ms") == "" {
		t.Error("expected X-Elapsed-Ms header")
	}
}

func TestNonNowrapResponse(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Call without nowrap — should return envelope
	req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.callFunction(w, req, "test/api/greet", "hello")

	body := w.Body.String()
	if !strings.Contains(body, `"module"`) || !strings.Contains(body, `"elapsed_ms"`) {
		t.Errorf("expected FunctionCallResponse envelope, got %s", body)
	}
}

func TestGetCustomRoutes(t *testing.T) {
	s := &Server{
		modules: map[string]*ModuleInfo{
			"mod1": {
				Path: "mod1",
				Exports: []ExportInfo{
					{Name: "f1", RouteMethod: "POST", RoutePath: "/api/v1/f1"},
					{Name: "f2"}, // no route
				},
			},
			"mod2": {
				Path: "mod2",
				Exports: []ExportInfo{
					{Name: "g1", RouteMethod: "GET", RoutePath: "/health"},
				},
			},
		},
	}

	routes := s.getCustomRoutes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 custom routes, got %d", len(routes))
	}

	// Check that both routes are present (order may vary)
	found := map[string]bool{}
	for _, r := range routes {
		found[r.Path] = true
	}
	if !found["/api/v1/f1"] {
		t.Error("expected route /api/v1/f1")
	}
	if !found["/health"] {
		t.Error("expected route /health")
	}
}

func TestParseArgs_EmptyBody(t *testing.T) {
	args, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args != nil {
		t.Errorf("expected nil args for empty body, got %v", args)
	}
}

func TestParseArgs_EmptyString(t *testing.T) {
	args, err := parseArgs([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args != nil {
		t.Errorf("expected nil args for empty string, got %v", args)
	}
}

func TestParseArgs_ArgsArray(t *testing.T) {
	args, err := parseArgs([]byte(`{"args": ["hello", 42]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "hello" {
		t.Errorf("expected first arg 'hello', got %v", args[0])
	}
}

func TestParseArgs_SingleValue(t *testing.T) {
	args, err := parseArgs([]byte(`"just a string"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "just a string" {
		t.Errorf("expected 'just a string', got %v", args[0])
	}
}

func TestParseQueryArgs_Positional(t *testing.T) {
	query := url.Values{"args": {"3", "5"}}
	args := parseQueryArgs(query)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	// Numbers should be parsed as float64
	if args[0] != float64(3) {
		t.Errorf("expected 3, got %v (%T)", args[0], args[0])
	}
	if args[1] != float64(5) {
		t.Errorf("expected 5, got %v (%T)", args[1], args[1])
	}
}

func TestParseQueryArgs_Named(t *testing.T) {
	query := url.Values{"name": {"Alice"}, "age": {"30"}}
	args := parseQueryArgs(query)
	if len(args) != 1 {
		t.Fatalf("expected 1 record arg, got %d", len(args))
	}
	record, ok := args[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", args[0])
	}
	if record["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", record["name"])
	}
	if record["age"] != float64(30) {
		t.Errorf("expected age=30, got %v (%T)", record["age"], record["age"])
	}
}

func TestParseQueryArgs_Empty(t *testing.T) {
	args := parseQueryArgs(url.Values{})
	if args != nil {
		t.Fatalf("expected nil for empty query, got %v", args)
	}
}

func TestParseQueryArgs_StringValues(t *testing.T) {
	query := url.Values{"query": {"hello world"}, "flag": {"true"}}
	args := parseQueryArgs(query)
	if len(args) != 1 {
		t.Fatalf("expected 1 record arg, got %d", len(args))
	}
	record := args[0].(map[string]interface{})
	if record["query"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", record["query"])
	}
	if record["flag"] != true {
		t.Errorf("expected true, got %v (%T)", record["flag"], record["flag"])
	}
}

func TestBuildHttpRequestRecord(t *testing.T) {
	req, _ := http.NewRequest("POST", "/webhooks/stripe?event=checkout&debug=true", nil)
	req.Header.Set("Stripe-Signature", "t=123,v1=abc")
	req.Header.Set("Content-Type", "application/json")

	body := []byte(`{"type": "checkout.session.completed"}`)
	rec := buildHttpRequestRecord(req, body)

	// Check body
	if rec["body"] != `{"type": "checkout.session.completed"}` {
		t.Errorf("unexpected body: %v", rec["body"])
	}

	// Check method
	if rec["method"] != "POST" {
		t.Errorf("unexpected method: %v", rec["method"])
	}

	// Check path
	if rec["path"] != "/webhooks/stripe" {
		t.Errorf("unexpected path: %v", rec["path"])
	}

	// Check headers are JObject
	headersJObj, ok := rec["headers"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("headers is not *eval.TaggedValue: %T", rec["headers"])
	}
	if headersJObj.CtorName != "JObject" {
		t.Fatalf("headers should be JObject, got %s", headersJObj.CtorName)
	}
	// Verify Stripe-Signature header is present (hyphenated key!)
	found := findJObjectValue(headersJObj, "Stripe-Signature")
	if found != "t=123,v1=abc" {
		t.Errorf("expected Stripe-Signature 't=123,v1=abc', got %q", found)
	}
	found = findJObjectValue(headersJObj, "Content-Type")
	if found != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", found)
	}

	// Check query is JObject
	queryJObj, ok := rec["query"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("query is not *eval.TaggedValue: %T", rec["query"])
	}
	if queryJObj.CtorName != "JObject" {
		t.Fatalf("query should be JObject, got %s", queryJObj.CtorName)
	}
	if findJObjectValue(queryJObj, "event") != "checkout" {
		t.Errorf("expected query event 'checkout', got %q", findJObjectValue(queryJObj, "event"))
	}
}

// findJObjectValue extracts a string value from a JObject by key (for testing).
func findJObjectValue(jobj *eval.TaggedValue, key string) string {
	if len(jobj.Fields) == 0 {
		return ""
	}
	list, ok := jobj.Fields[0].(*eval.ListValue)
	if !ok {
		return ""
	}
	for _, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			continue
		}
		kv, ok := rec.Fields["key"].(*eval.StringValue)
		if !ok || kv.Value != key {
			continue
		}
		valTag, ok := rec.Fields["value"].(*eval.TaggedValue)
		if !ok || valTag.CtorName != "JString" || len(valTag.Fields) == 0 {
			continue
		}
		sv, ok := valTag.Fields[0].(*eval.StringValue)
		if ok {
			return sv.Value
		}
	}
	return ""
}

// --- Route Collision Guard Tests ---

func TestRouteCollisionGuard_SkipsBuiltinPath(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Manually inject a custom route that collides with a built-in path
	srv.mu.Lock()
	for _, mod := range srv.modules {
		mod.Exports = append(mod.Exports, ExportInfo{
			Name:        "agentCard",
			RoutePath:   "/.well-known/agent.json",
			RouteMethod: "GET",
		})
	}
	srv.mu.Unlock()

	// Enable A2A so the built-in path is registered
	srv.a2aEnabled = true

	// buildRoutes should NOT panic — collision guard skips the duplicate
	mux := srv.buildRoutes()

	// The built-in A2A handler should still work
	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from built-in A2A handler, got %d", w.Code)
	}
}

func TestRouteCollisionGuard_AllowsNonCollidingRoute(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// Inject a custom route that does NOT collide with any built-in
	srv.mu.Lock()
	for _, mod := range srv.modules {
		mod.Exports = append(mod.Exports, ExportInfo{
			Name:        "custom",
			RoutePath:   "/custom/endpoint",
			RouteMethod: "GET",
		})
	}
	srv.mu.Unlock()

	// buildRoutes should register the non-colliding route
	mux := srv.buildRoutes()

	req := httptest.NewRequest("GET", "/custom/endpoint", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not 404 — the custom route was registered
	if w.Code == http.StatusNotFound {
		t.Fatal("expected custom route to be registered, got 404")
	}
}

func TestA2ADisabledByDefault(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()
	// a2aEnabled is false by default

	mux := srv.buildRoutes()

	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 when A2A is disabled, got 200")
	}
}

func TestA2AEnabledRegistersRoutes(t *testing.T) {
	srv := testServer(t)
	srv.a2aEnabled = true
	defer srv.Close()

	mux := srv.buildRoutes()

	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when A2A is enabled, got %d", w.Code)
	}
}

func TestBuildHttpRequestRecord_EmptyBody(t *testing.T) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rec := buildHttpRequestRecord(req, nil)

	if rec["body"] != "" {
		t.Errorf("expected empty body, got %q", rec["body"])
	}
	if rec["method"] != "GET" {
		t.Errorf("unexpected method: %v", rec["method"])
	}

	queryJObj, ok := rec["query"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("query should be *eval.TaggedValue, got %T", rec["query"])
	}
	if queryJObj.CtorName != "JObject" {
		t.Fatalf("query should be JObject, got %s", queryJObj.CtorName)
	}
	list := queryJObj.Fields[0].(*eval.ListValue)
	if len(list.Elements) != 0 {
		t.Errorf("expected empty query JObject, got %d elements", len(list.Elements))
	}
}

// TestPackageDependencyRouteDiscovery verifies that @route annotations in
// package dependency modules are discovered and registered by serve-api.
// This simulates the new code path in loadFile() that scans result.Modules.
func TestPackageDependencyRouteDiscovery(t *testing.T) {
	// Build a ModuleInfo as if it came from a package dependency's Iface
	depInfo := &ModuleInfo{
		Path: "pkg/sunholo/ailang_parse/services/tools",
		Exports: []ExportInfo{
			{Name: "toolDefinitions", Type: "HttpRequest -> string", Arity: 1},
			{Name: "internalHelper", Type: "int -> int", Arity: 1},
		},
	}

	// Build an AST with @route annotations (simulating what the parser produces)
	depFile := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "toolDefinitions",
				IsExport: true,
				Annotations: []*ast.Annotation{
					{
						Name: "route",
						Args: []ast.Expr{
							&ast.Literal{Kind: ast.StringLit, Value: "GET"},
							&ast.Literal{Kind: ast.StringLit, Value: "/api/v1/tools"},
						},
					},
				},
			},
			{
				Name:     "internalHelper",
				IsExport: true,
				// No @route annotation
			},
		},
	}

	// Run the same extraction pipeline as loadFile()
	extractParamInfo(depInfo, depFile)
	extractRouteAnnotations(depInfo, depFile)
	extractNoExposeAnnotations(depInfo, depFile)

	// Verify route was discovered
	found := false
	for _, exp := range depInfo.Exports {
		if exp.Name == "toolDefinitions" {
			if exp.RoutePath != "/api/v1/tools" {
				t.Errorf("expected RoutePath /api/v1/tools, got %q", exp.RoutePath)
			}
			if exp.RouteMethod != "GET" {
				t.Errorf("expected RouteMethod GET, got %q", exp.RouteMethod)
			}
			found = true
		}
		if exp.Name == "internalHelper" && exp.RoutePath != "" {
			t.Errorf("internalHelper should not have a route, got %q", exp.RoutePath)
		}
	}
	if !found {
		t.Fatal("toolDefinitions export not found")
	}

	// Verify that only modules with routes would be registered (the hasRoutes check)
	hasRoutes := false
	for _, exp := range depInfo.Exports {
		if exp.RoutePath != "" {
			hasRoutes = true
			break
		}
	}
	if !hasRoutes {
		t.Fatal("expected hasRoutes to be true for dependency with @route annotations")
	}
}

// TestPackageDependencyWithoutRoutes verifies that package dependencies WITHOUT
// @route annotations are not registered as serve-api modules.
func TestPackageDependencyWithoutRoutes(t *testing.T) {
	depInfo := &ModuleInfo{
		Path: "pkg/sunholo/utils/helpers",
		Exports: []ExportInfo{
			{Name: "formatDate", Type: "string -> string", Arity: 1},
		},
	}

	depFile := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "formatDate",
				IsExport: true,
				// No annotations
			},
		},
	}

	extractParamInfo(depInfo, depFile)
	extractRouteAnnotations(depInfo, depFile)
	extractNoExposeAnnotations(depInfo, depFile)

	hasRoutes := false
	for _, exp := range depInfo.Exports {
		if exp.RoutePath != "" {
			hasRoutes = true
			break
		}
	}
	if hasRoutes {
		t.Fatal("expected hasRoutes to be false for dependency without @route annotations")
	}
}

// TestFindRouteByPath verifies that findRouteByPath locates custom routes
// from package modules registered in s.modules.
func TestFindRouteByPath(t *testing.T) {
	srv := &Server{
		modules: map[string]*ModuleInfo{
			"pkg/sunholo/ailang_parse/services/tools": {
				Path: "pkg/sunholo/ailang_parse/services/tools",
				Exports: []ExportInfo{
					{
						Name:        "toolDefinitions",
						Type:        "HttpRequest -> string",
						Arity:       1,
						RouteMethod: "GET",
						RoutePath:   "/api/v1/tools",
						IsNowrap:    true,
					},
				},
			},
			"local/api": {
				Path: "local/api",
				Exports: []ExportInfo{
					{Name: "health", Type: "string", Arity: 0},
				},
			},
		},
	}

	// Should find the package module's route
	route := srv.findRouteByPath("/api/v1/tools")
	if route == nil {
		t.Fatal("expected to find route for /api/v1/tools")
	}
	if route.Module != "pkg/sunholo/ailang_parse/services/tools" {
		t.Errorf("expected module pkg/sunholo/ailang_parse/services/tools, got %q", route.Module)
	}
	if route.Function != "toolDefinitions" {
		t.Errorf("expected function toolDefinitions, got %q", route.Function)
	}
	if !route.IsNowrap {
		t.Error("expected IsNowrap to be true")
	}

	// Should not find non-existent route
	route = srv.findRouteByPath("/api/v1/nonexistent")
	if route != nil {
		t.Errorf("expected nil for non-existent route, got %+v", route)
	}
}

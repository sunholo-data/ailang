package apiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
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

// TestPackageDependencyRouteDiscovery verifies that @route annotations in
// package dependency modules are discovered and registered by serve-api.
func TestPackageDependencyRouteDiscovery(t *testing.T) {
	depInfo := &ModuleInfo{
		Path: "pkg/sunholo/ailang_parse/services/tools",
		Exports: []ExportInfo{
			{Name: "toolDefinitions", Type: "HttpRequest -> string", Arity: 1},
			{Name: "internalHelper", Type: "int -> int", Arity: 1},
		},
	}

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

	extractParamInfo(depInfo, depFile)
	extractRouteAnnotations(depInfo, depFile)
	extractNoExposeAnnotations(depInfo, depFile)

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

	route = srv.findRouteByPath("/api/v1/nonexistent")
	if route != nil {
		t.Errorf("expected nil for non-existent route, got %+v", route)
	}
}

// TestRegisterCustomRoutes_NoDuplicatePanic verifies that duplicate route paths
// from multiple modules don't panic Go 1.22+ ServeMux.
func TestRegisterCustomRoutes_NoDuplicatePanic(t *testing.T) {
	srv := &Server{
		modules: map[string]*ModuleInfo{
			"pkg/owner/repo/tools": {
				Path: "pkg/owner/repo/tools",
				Exports: []ExportInfo{
					{Name: "list", RouteMethod: "GET", RoutePath: "/api/v1/tools"},
				},
			},
			"local/tools": {
				Path: "local/tools",
				Exports: []ExportInfo{
					{Name: "list", RouteMethod: "GET", RoutePath: "/api/v1/tools"},
				},
			},
		},
	}

	mux := http.NewServeMux()
	builtinPaths := map[string]bool{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registerCustomRoutes panicked on duplicate route: %v", r)
		}
	}()
	srv.registerCustomRoutes(mux, builtinPaths)
}

// --- Route Collision Guard Tests ---

func TestRouteCollisionGuard_SkipsBuiltinPath(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	srv.mu.Lock()
	for _, mod := range srv.modules {
		mod.Exports = append(mod.Exports, ExportInfo{
			Name:        "agentCard",
			RoutePath:   "/.well-known/agent.json",
			RouteMethod: "GET",
		})
	}
	srv.mu.Unlock()

	srv.a2aEnabled = true

	mux := srv.buildRoutes()

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

	srv.mu.Lock()
	for _, mod := range srv.modules {
		mod.Exports = append(mod.Exports, ExportInfo{
			Name:        "custom",
			RoutePath:   "/custom/endpoint",
			RouteMethod: "GET",
		})
	}
	srv.mu.Unlock()

	mux := srv.buildRoutes()

	req := httptest.NewRequest("GET", "/custom/endpoint", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Fatal("expected custom route to be registered, got 404")
	}
}

func TestA2ADisabledByDefault(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

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

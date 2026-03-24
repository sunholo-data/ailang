package apiserver

import (
	"net/url"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
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

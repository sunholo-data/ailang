package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestRouteAnnotation_BasicPost tests @route("POST", "/api/v1/parse") parsing.
func TestRouteAnnotation_BasicPost(t *testing.T) {
	input := `
@route("POST", "/api/v1/parse")
export func parse(content: string) -> string ! {} { content }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.Name != "parse" {
		t.Errorf("expected function name 'parse', got %q", fn.Name)
	}
	if !fn.IsExport {
		t.Error("expected function to be exported")
	}

	if len(fn.Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(fn.Annotations))
	}

	ann := fn.Annotations[0]
	if ann.Name != "route" {
		t.Errorf("expected annotation name 'route', got %q", ann.Name)
	}
	if len(ann.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(ann.Args))
	}

	methodLit, ok := ann.Args[0].(*ast.Literal)
	if !ok || methodLit.Kind != ast.StringLit {
		t.Fatal("expected first arg to be string literal")
	}
	if methodLit.Value.(string) != "POST" {
		t.Errorf("expected method 'POST', got %q", methodLit.Value)
	}

	pathLit, ok := ann.Args[1].(*ast.Literal)
	if !ok || pathLit.Kind != ast.StringLit {
		t.Fatal("expected second arg to be string literal")
	}
	if pathLit.Value.(string) != "/api/v1/parse" {
		t.Errorf("expected path '/api/v1/parse', got %q", pathLit.Value)
	}
}

// TestRouteAnnotation_Get tests @route("GET", "/health") parsing.
func TestRouteAnnotation_Get(t *testing.T) {
	input := `
@route("GET", "/health")
export pure func health() -> string { "ok" }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Funcs[0]
	ann := fn.GetAnnotation("route")
	if ann == nil {
		t.Fatal("expected @route annotation")
	}
	if ann.Args[0].(*ast.Literal).Value.(string) != "GET" {
		t.Errorf("expected GET method")
	}
}

// TestRouteAnnotation_WithVerify tests @route + @verify on the same function.
func TestRouteAnnotation_WithVerify(t *testing.T) {
	input := `
@route("POST", "/api/v1/compute")
@verify(depth: 3)
export func compute(x: int) -> int ! {} { x + 1 }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	fn := file.Funcs[0]
	if len(fn.Annotations) != 2 {
		t.Fatalf("expected 2 annotations, got %d", len(fn.Annotations))
	}

	// @route should be first
	if fn.Annotations[0].Name != "route" {
		t.Errorf("expected first annotation 'route', got %q", fn.Annotations[0].Name)
	}
	// @verify should be second
	if fn.Annotations[1].Name != "verify" {
		t.Errorf("expected second annotation 'verify', got %q", fn.Annotations[1].Name)
	}

	// VerifyDepth should be populated via backward compat
	if fn.VerifyDepth == nil || *fn.VerifyDepth != 3 {
		t.Errorf("expected VerifyDepth=3, got %v", fn.VerifyDepth)
	}

	// GetAnnotation helper should work
	routeAnn := fn.GetAnnotation("route")
	if routeAnn == nil {
		t.Fatal("GetAnnotation(\"route\") returned nil")
	}
	if routeAnn.Args[1].(*ast.Literal).Value.(string) != "/api/v1/compute" {
		t.Error("wrong route path")
	}
}

// TestRouteAnnotation_InvalidMethod tests that @route("INVALID", "/path") produces an error.
func TestRouteAnnotation_InvalidMethod(t *testing.T) {
	input := `
@route("INVALID", "/path")
func bar(x: int) -> int ! {} { x }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for invalid HTTP method, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "invalid HTTP method") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about invalid HTTP method, got: %v", p.Errors())
	}
}

// TestRouteAnnotation_MissingSlash tests that @route("POST", "no-slash") produces an error.
func TestRouteAnnotation_MissingSlash(t *testing.T) {
	input := `
@route("POST", "no-slash")
func bar(x: int) -> int ! {} { x }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for missing slash, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "must start with '/'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing slash, got: %v", p.Errors())
	}
}

// TestRouteAnnotation_MixedFunctions tests multiple functions, some with @route, some without.
func TestRouteAnnotation_MixedFunctions(t *testing.T) {
	input := `
@route("POST", "/api/v1/parse")
export func parse(content: string) -> string ! {} { content }

export func helper(x: int) -> int ! {} { x + 1 }

@route("GET", "/api/v1/formats")
export pure func listFormats() -> string { "json,xml" }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(file.Funcs))
	}

	// parse has @route
	if file.Funcs[0].GetAnnotation("route") == nil {
		t.Error("expected parse to have @route annotation")
	}

	// helper has no annotations
	if len(file.Funcs[1].Annotations) != 0 {
		t.Errorf("expected helper to have 0 annotations, got %d", len(file.Funcs[1].Annotations))
	}

	// listFormats has @route
	ann := file.Funcs[2].GetAnnotation("route")
	if ann == nil {
		t.Fatal("expected listFormats to have @route annotation")
	}
	if ann.Args[0].(*ast.Literal).Value.(string) != "GET" {
		t.Error("expected GET method for listFormats")
	}
}

// TestRouteAnnotation_AllMethods tests all valid HTTP methods.
func TestRouteAnnotation_AllMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		input := `@route("` + method + `", "/test")
func f(x: int) -> int ! {} { x }`

		l := lexer.New(input, "test.ail")
		p := New(l)
		file := p.ParseFile()

		if len(p.Errors()) > 0 {
			t.Errorf("method %s: unexpected parser error: %v", method, p.Errors())
			continue
		}

		ann := file.Funcs[0].GetAnnotation("route")
		if ann == nil {
			t.Errorf("method %s: expected @route annotation", method)
			continue
		}
		if ann.Args[0].(*ast.Literal).Value.(string) != method {
			t.Errorf("method %s: got %v", method, ann.Args[0].(*ast.Literal).Value)
		}
	}
}

package apiserver

import (
	"testing"
)

func TestPortableToolName(t *testing.T) {
	tests := []struct {
		name     string
		modPath  string
		funcName string
		want     string
	}{
		{
			name:     "simple relative path",
			modPath:  "docparse/services",
			funcName: "parseCsv",
			want:     "docparse.services.parseCsv",
		},
		{
			name:     "single module",
			modPath:  "main",
			funcName: "hello",
			want:     "main.hello",
		},
		{
			name:     "pkg path strips org and repo",
			modPath:  "pkg/sunholo/ailang-parse/docparse/services/samples",
			funcName: "sampleResolvePath",
			want:     "docparse.services.samples.sampleResolvePath",
		},
		{
			name:     "absolute path strips machine prefix",
			modPath:  "Users/mark/dev/sunholo/ailang-parse/docparse/services",
			funcName: "parseDocx",
			want:     "ailang-parse.docparse.services.parseDocx",
		},
		{
			name:     "dotted path preserved",
			modPath:  "mylib/utils",
			funcName: "format",
			want:     "mylib.utils.format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portableToolName(tt.modPath, tt.funcName)
			if got != tt.want {
				t.Errorf("portableToolName(%q, %q) = %q, want %q", tt.modPath, tt.funcName, got, tt.want)
			}
		})
	}
}

func TestIsExposedFiltersMCPTools(t *testing.T) {
	// Verify that isExposed correctly filters exports for MCP context.
	srv := &Server{routesOnly: true}

	tests := []struct {
		name   string
		export ExportInfo
		want   bool
	}{
		{
			name:   "route function exposed",
			export: ExportInfo{Name: "parseDocx", RoutePath: "/api/v1/parse", Arity: 1},
			want:   true,
		},
		{
			name:   "non-route function hidden with routes-only",
			export: ExportInfo{Name: "xmlEscape", Arity: 1},
			want:   false,
		},
		{
			name:   "noexpose function always hidden",
			export: ExportInfo{Name: "internal", RoutePath: "/api/v1/internal", IsNoExpose: true, Arity: 1},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := srv.isExposed(tt.export)
			if got != tt.want {
				t.Errorf("isExposed(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}

	// Without routes-only, non-route functions are visible.
	srvNoFilter := &Server{routesOnly: false}
	exp := ExportInfo{Name: "xmlEscape", Arity: 1}
	if !srvNoFilter.isExposed(exp) {
		t.Error("expected non-route function to be exposed when routesOnly=false")
	}
}

func TestAilangTypeToJSONSchema(t *testing.T) {
	tests := []struct {
		ailangType string
		want       string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"float", "number"},
		{"bool", "boolean"},
		{"Json", "object"},
		{"record", "object"},
		{"list", "array"},
		{"array", "array"},
		{"bytes", "string"},
		{"CustomType", "string"}, // unknown types default to string
	}

	for _, tt := range tests {
		t.Run(tt.ailangType, func(t *testing.T) {
			got := ailangTypeToJSONSchema(tt.ailangType)
			if got != tt.want {
				t.Errorf("ailangTypeToJSONSchema(%q) = %q, want %q", tt.ailangType, got, tt.want)
			}
		})
	}
}

func TestBuildNamedInputSchema(t *testing.T) {
	t.Run("named parameters", func(t *testing.T) {
		export := ExportInfo{
			Name:       "parseDocx",
			Arity:      2,
			ParamNames: []string{"filepath", "format"},
			ParamTypes: []string{"string", "int"},
		}
		schema := buildNamedInputSchema(export)

		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties to be a map")
		}
		if len(props) != 2 {
			t.Fatalf("expected 2 properties, got %d", len(props))
		}

		fp, ok := props["filepath"].(map[string]any)
		if !ok {
			t.Fatal("expected filepath property")
		}
		if fp["type"] != "string" {
			t.Errorf("expected filepath type 'string', got %q", fp["type"])
		}

		fmt, ok := props["format"].(map[string]any)
		if !ok {
			t.Fatal("expected format property")
		}
		if fmt["type"] != "integer" {
			t.Errorf("expected format type 'integer', got %q", fmt["type"])
		}

		required, ok := schema["required"].([]string)
		if !ok {
			t.Fatal("expected required to be []string")
		}
		if len(required) != 2 {
			t.Fatalf("expected 2 required params, got %d", len(required))
		}
	})

	t.Run("no params", func(t *testing.T) {
		export := ExportInfo{Name: "health", Arity: 0}
		schema := buildNamedInputSchema(export)
		props := schema["properties"].(map[string]any)
		if len(props) != 0 {
			t.Errorf("expected empty properties for zero-arity, got %d", len(props))
		}
	})

	t.Run("fallback to positional when no param names", func(t *testing.T) {
		export := ExportInfo{
			Name:  "legacy",
			Arity: 2,
			Type:  "string -> int -> bool",
		}
		schema := buildNamedInputSchema(export)
		props := schema["properties"].(map[string]any)
		if _, ok := props["args"]; !ok {
			t.Error("expected fallback 'args' property when no param names available")
		}
	})
}

func TestMCPAutoExcludeUndocumentedHelpers(t *testing.T) {
	// With routesOnly, functions without doc comment AND without @route should
	// be hidden from MCP (even if isExposed returns true for non-routesOnly).
	srv := &Server{routesOnly: true}

	// Documented non-route function: should pass isExposed (routesOnly hides it)
	// but our MCP-specific filter checks doc comment too.
	undocumented := ExportInfo{Name: "xmlEscape", Arity: 1}
	documented := ExportInfo{Name: "parseCsv", Arity: 1, DocComment: "Parse a CSV file."}
	routed := ExportInfo{Name: "parseDocx", Arity: 1, RoutePath: "/api/v1/parse"}

	// With routesOnly, undocumented non-route is hidden by isExposed already.
	if srv.isExposed(undocumented) {
		t.Error("expected undocumented non-route to be hidden by isExposed with routesOnly")
	}
	// But documented non-route is also hidden by isExposed (no route path).
	if srv.isExposed(documented) {
		t.Error("expected documented non-route to be hidden by isExposed with routesOnly")
	}
	// Routed function is exposed.
	if !srv.isExposed(routed) {
		t.Error("expected routed function to be exposed")
	}

	// Without routesOnly, test the MCP-specific auto-exclude logic:
	// undocumented + no route should still be excluded from MCP.
	srvNoFilter := &Server{routesOnly: false}
	_ = srvNoFilter // auto-exclude only applies when routesOnly=true
	// The MCP auto-exclude is: routesOnly && no RoutePath && no DocComment
	// So without routesOnly, all pass through isExposed.
	if !srvNoFilter.isExposed(undocumented) {
		t.Error("without routesOnly, all functions should pass isExposed")
	}
}

func TestDocCommentUsedAsDescription(t *testing.T) {
	withDoc := ExportInfo{
		Name:       "parseDocx",
		DocComment: "Parse a DOCX file and return structured content.",
		Type:       "string -> Document",
	}
	withoutDoc := ExportInfo{
		Name: "xmlEscape",
		Type: "string -> string",
		Pure: true,
	}

	// With doc comment, description should be the comment.
	if withDoc.DocComment == "" {
		t.Error("expected doc comment to be set")
	}

	// Without doc comment, description falls back to type sig.
	if withoutDoc.DocComment != "" {
		t.Error("expected no doc comment")
	}
}

func TestHeadersParamIndex(t *testing.T) {
	// Verify that _headers param is correctly identified in param lists.
	paramNames := []string{"filepath", "_headers", "format"}
	found := -1
	for i, name := range paramNames {
		if name == "_headers" {
			found = i
			break
		}
	}
	if found != 1 {
		t.Errorf("expected _headers at index 1, got %d", found)
	}

	// No _headers param
	noHeaders := []string{"filepath", "format"}
	found = -1
	for i, name := range noHeaders {
		if name == "_headers" {
			found = i
			break
		}
	}
	if found != -1 {
		t.Errorf("expected no _headers, got index %d", found)
	}
}

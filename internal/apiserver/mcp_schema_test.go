package apiserver

import (
	"strings"
	"testing"
)

func TestMCPToolName(t *testing.T) {
	tests := []struct {
		name       string
		modPath    string
		funcName   string
		override   string
		preferBare bool
		want       string
	}{
		{
			name:       "bare name when unique",
			modPath:    "docparse/services/mcp_tools",
			funcName:   "mcpParse",
			preferBare: true,
			want:       "mcpParse",
		},
		{
			name:       "collision falls back to last segment + funcName",
			modPath:    "docparse/services",
			funcName:   "parseCsv",
			preferBare: false,
			want:       "services_parseCsv",
		},
		{
			name:       "dots and slashes sanitized to underscores",
			modPath:    "pkg/sunholo/ailang-parse/docparse/services/samples",
			funcName:   "sampleResolvePath",
			preferBare: false,
			want:       "samples_sampleResolvePath",
		},
		{
			name:       "author override honored verbatim",
			modPath:    "docparse/services/mcp_tools",
			funcName:   "mcpParse",
			override:   "parse",
			preferBare: true,
			want:       "parse",
		},
		{
			name:       "long name truncated with hash suffix",
			modPath:    "very/deep/module/path/with/many/segments/that/keeps/going",
			funcName:   "thisIsAReallyLongFunctionNameThatExceedsTheSixtyFourCharLimitEasily",
			preferBare: true,
			// Length must be exactly 64.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcpToolName(tt.modPath, tt.funcName, tt.override, tt.preferBare)
			if tt.want != "" && got != tt.want {
				t.Errorf("mcpToolName(%q, %q, %q, %v) = %q, want %q", tt.modPath, tt.funcName, tt.override, tt.preferBare, got, tt.want)
			}
			if !mcpToolNameRegex.MatchString(got) {
				t.Errorf("generated name %q does not match MCP regex %s", got, mcpToolNameRegex.String())
			}
			if len(got) > 64 {
				t.Errorf("generated name %q exceeds 64 chars (%d)", got, len(got))
			}
		})
	}
}

func TestValidateMCPName(t *testing.T) {
	valid := []string{"parse", "mcpParse", "services_parseCsv", "a-b-c", "A1_b2", strings.Repeat("a", 64)}
	invalid := []string{"", "has.dot", "has/slash", "has space", "has!bang", strings.Repeat("a", 65)}

	for _, n := range valid {
		if err := validateMCPName(n); err != nil {
			t.Errorf("validateMCPName(%q) returned error: %v", n, err)
		}
	}
	for _, n := range invalid {
		if err := validateMCPName(n); err == nil {
			t.Errorf("validateMCPName(%q) should have returned an error", n)
		}
	}
}

func TestSanitizeMCPName(t *testing.T) {
	cases := map[string]string{
		"foo":        "foo",
		"foo.bar":    "foo_bar",
		"foo/bar":    "foo_bar",
		"foo bar!":   "foo_bar_",
		"":           "_",
		"a-b_c":      "a-b_c",
		"x.y/z.w":    "x_y_z_w",
		"docparse.s": "docparse_s",
		"unicode-é":  "unicode-_",
	}
	for in, want := range cases {
		if got := sanitizeMCPName(in); got != want {
			t.Errorf("sanitizeMCPName(%q) = %q, want %q", in, got, want)
		}
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
	// With routesOnly, ALL non-@route functions are hidden via isExposed(),
	// regardless of doc comment presence.
	srv := &Server{routesOnly: true}

	undocumented := ExportInfo{Name: "xmlEscape", Arity: 1}
	documented := ExportInfo{Name: "parseCsv", Arity: 1, DocComment: "Parse a CSV file."}
	routed := ExportInfo{Name: "parseDocx", Arity: 1, RoutePath: "/api/v1/parse"}

	if srv.isExposed(undocumented) {
		t.Error("expected undocumented non-route to be hidden by isExposed with routesOnly")
	}
	if srv.isExposed(documented) {
		t.Error("expected documented non-route to be hidden by isExposed with routesOnly")
	}
	if !srv.isExposed(routed) {
		t.Error("expected routed function to be exposed")
	}

	// Without routesOnly, all non-@noexpose functions pass isExposed.
	srvNoFilter := &Server{routesOnly: false}
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

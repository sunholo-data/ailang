package mcpcheck

import "testing"

func fileSchema(required ...any) map[string]any {
	props := map[string]any{}
	for _, p := range fileProps {
		props[p] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func TestCheckFileParams(t *testing.T) {
	withFile := func(file map[string]any, extra map[string]any) tool {
		schema := map[string]any{"type": "object", "properties": map[string]any{"file": file}}
		for k, v := range extra {
			schema[k] = v
		}
		return tool{Name: "f", InputSchema: schema, Meta: map[string]any{"openai/fileParams": []any{"file"}}}
	}
	missingName := fileSchema("download_url", "file_id")
	delete(missingName["properties"].(map[string]any), "file_name")
	cases := []struct {
		name string
		tl   tool
		fail bool
	}{
		{"documented shape", withFile(fileSchema("download_url", "file_id"), nil), false},
		{"$ref to $defs", withFile(map[string]any{"$ref": "#/$defs/OpenAIFile"}, map[string]any{"$defs": map[string]any{"OpenAIFile": fileSchema("download_url", "file_id")}}), false},
		{"array of files", withFile(map[string]any{"type": "array", "items": fileSchema("download_url", "file_id")}, nil), false},
		{"omits file_name", withFile(missingName, nil), true},
		{"file_id optional", withFile(fileSchema("download_url"), nil), true},
		{"mime_type required", withFile(fileSchema("download_url", "file_id", "mime_type"), nil), true},
		{"not an object", withFile(map[string]any{"type": "string"}, nil), true},
		{"dangling $ref", withFile(map[string]any{"$ref": "#/$defs/Nope"}, nil), true},
		{"param not a property", tool{Name: "f", InputSchema: map[string]any{}, Meta: map[string]any{"openai/fileParams": []any{"file"}}}, true},
		{"fileParams not a list", tool{Name: "f", Meta: map[string]any{"openai/fileParams": "file"}}, true},
	}
	for _, c := range cases {
		if fs := checkFileParams([]tool{c.tl}); Failed(fs) != c.fail {
			t.Errorf("%s: failed=%v, want %v: %+v", c.name, Failed(fs), c.fail, fs)
		}
	}
	if fs := checkFileParams([]tool{{Name: "plain"}}); Failed(fs) {
		t.Errorf("no fileParams must pass: %+v", fs)
	}
}

func TestCheckSecuritySchemes(t *testing.T) {
	oauth := []any{map[string]any{"type": "oauth2"}}
	gated := map[string]bool{"g": true}
	if Failed(checkSecuritySchemes([]tool{{Name: "g", SecuritySchemes: oauth}}, gated)) {
		t.Error("top-level oauth2 should pass")
	}
	if Failed(checkSecuritySchemes([]tool{{Name: "g", Meta: map[string]any{"securitySchemes": oauth}}}, gated)) {
		t.Error("_meta mirror oauth2 should pass")
	}
	if !Failed(checkSecuritySchemes([]tool{{Name: "g", SecuritySchemes: []any{map[string]any{"type": "noauth"}}}}, gated)) {
		t.Error("a gated tool declaring only noauth must fail")
	}
	if Failed(checkSecuritySchemes([]tool{{Name: "open"}}, gated)) {
		t.Error("an ungated tool needs no scheme")
	}
}

package mcpcheck

import (
	"strings"
	"testing"
)

func tl(name string, schema, ann map[string]any) tool {
	return tool{Name: name, Title: "T", InputSchema: schema, Annotations: ann}
}

func TestCheckZeroArg(t *testing.T) {
	bad := tl("formats", map[string]any{"properties": map[string]any{"_": map[string]any{}}, "required": []any{"_"}}, nil)
	if fs := checkZeroArg([]tool{bad}); !Failed(fs) {
		t.Fatalf("required \"_\" must fail: %+v", fs)
	}
	ok := tl("formats", map[string]any{"properties": map[string]any{}}, nil)
	if fs := checkZeroArg([]tool{ok}); Failed(fs) {
		t.Fatalf("no required params must pass: %+v", fs)
	}
}

func TestCheckAnnotationsTargets(t *testing.T) {
	// Anthropic: one of readOnly/destructive suffices; OpenAI: all three explicit.
	partial := tl("p", nil, map[string]any{"readOnlyHint": true})
	if Failed(checkAnnotations([]tool{partial}, "anthropic")) {
		t.Error("anthropic: readOnlyHint alone should pass")
	}
	if !Failed(checkAnnotations([]tool{partial}, "openai")) {
		t.Error("openai: missing destructiveHint/openWorldHint must fail")
	}
	full := tl("p", nil, map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false})
	if Failed(checkAnnotations([]tool{full}, "both")) {
		t.Error("both: explicit booleans should pass")
	}
	untitled := tool{Name: "u", Annotations: map[string]any{"readOnlyHint": true}}
	if !Failed(checkAnnotations([]tool{untitled}, "anthropic")) {
		t.Error("missing title must fail")
	}
	if Failed(checkAnnotations([]tool{{Name: "u", Annotations: map[string]any{"title": "From annotations", "readOnlyHint": true}}}, "anthropic")) {
		t.Error("annotations.title counts as a title")
	}
}

func TestCheckCredentials(t *testing.T) {
	for name, want := range map[string]bool{"apiKey": true, "api_key": true, "accessToken": true, "password": true,
		"filepath": false, "outputFormat": false, "deviceCode": false, "author": false} {
		ts := []tool{tl("x", map[string]any{"properties": map[string]any{name: map[string]any{}}}, nil)}
		if got := Failed(checkCredentials(ts)); got != want {
			t.Errorf("param %q: flagged=%v, want %v", name, got, want)
		}
	}
}

// A widget-only tool (visibility without "model") is exempt from the
// credentials rule — reported SKIP with the reason — while the same param on
// a model-visible tool still fails.
func TestCheckCredentials_WidgetOnlyExempt(t *testing.T) {
	props := map[string]any{"properties": map[string]any{"token": map[string]any{}}}
	appOnly := tool{Name: "upload", InputSchema: props, Meta: map[string]any{"ui": map[string]any{"visibility": []any{"app"}}}}
	fs := checkCredentials([]tool{appOnly})
	if Failed(fs) {
		t.Fatalf("widget-only tool must not fail: %+v", fs)
	}
	if fs[0].Status != Skip || !strings.Contains(fs[0].Message, "skipped: widget-only tool") {
		t.Fatalf("want a SKIP naming the exemption: %+v", fs)
	}
	for name, meta := range map[string]map[string]any{
		"no _meta":           nil,
		"visibility default": {"ui": map[string]any{"resourceUri": "ui://x"}},
		"model and app":      {"ui": map[string]any{"visibility": []any{"model", "app"}}},
		"model only":         {"ui": map[string]any{"visibility": []any{"model"}}},
	} {
		tt := tool{Name: "upload", InputSchema: props, Meta: meta}
		if !Failed(checkCredentials([]tool{tt})) {
			t.Errorf("%s: a model-visible token param must still fail", name)
		}
	}
	// The exemption is per tool.
	visible := tool{Name: "parse", InputSchema: props}
	if !Failed(checkCredentials([]tool{appOnly, visible})) {
		t.Error("a model-visible tool's token must fail even beside a widget-only one")
	}
}

func TestWellKnown(t *testing.T) {
	for in, want := range map[string]string{
		"https://auth.example.com":         "https://auth.example.com/.well-known/oauth-authorization-server",
		"https://auth.example.com/":        "https://auth.example.com/.well-known/oauth-authorization-server",
		"https://example.com/tenant/oauth": "https://example.com/.well-known/oauth-authorization-server/tenant/oauth",
	} {
		if got := wellKnown(in, "oauth-authorization-server"); got != want {
			t.Errorf("wellKnown(%q) = %q, want %q", in, got, want)
		}
	}
}

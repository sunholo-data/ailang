package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSensitiveConnectionRedactsEveryPresentation(t *testing.T) {
	secretURL := "wss://connect.example/session?token=super-secret"
	secretHeader := "Bearer browser-key"
	conn := NewSensitiveConnection(
		MCPServerSpec{Name: "playwright", Command: "npx", Args: []string{"-y", "@playwright/mcp@0.0.79"}},
		map[string]string{
			"PLAYWRIGHT_MCP_CDP_ENDPOINT": secretURL,
			"BROWSER_AUTH_HEADER":         secretHeader,
		},
	)

	for name, rendered := range map[string]string{
		"String": conn.String(),
		"format": fmt.Sprintf("%v", conn),
		"error":  fmt.Errorf("connect failed: %w", conn).Error(),
	} {
		if strings.Contains(rendered, secretURL) || strings.Contains(rendered, secretHeader) {
			t.Fatalf("%s leaked a connection secret: %s", name, rendered)
		}
		if !strings.Contains(rendered, Redacted) {
			t.Fatalf("%s did not make redaction explicit: %s", name, rendered)
		}
	}

	b, err := json.Marshal(conn)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "super-secret") || strings.Contains(string(b), "browser-key") {
		t.Fatalf("JSON leaked connection material: %s", b)
	}
}

func TestSanitizeDiagnosticsRecursivelyRemovesSensitiveKeys(t *testing.T) {
	input := map[string]any{
		"status":  "failed",
		"api_key": "abc123",
		"nested": map[string]any{
			"connectUrl": "wss://example.invalid/token",
			"cookies":    []any{"sid=secret", "theme=dark"},
			"safe":       "kept",
		},
	}
	got := SanitizeDiagnostics(input).(map[string]any)
	if got["api_key"] != Redacted {
		t.Fatalf("api_key = %v, want redacted", got["api_key"])
	}
	nested := got["nested"].(map[string]any)
	if nested["connectUrl"] != Redacted || nested["cookies"] != Redacted {
		t.Fatalf("nested secrets not redacted: %#v", nested)
	}
	if nested["safe"] != "kept" {
		t.Fatalf("safe diagnostic changed: %#v", nested)
	}
}

func TestBrowserRunManifestContainsOnlySafeConnectionSummary(t *testing.T) {
	manifest := BrowserRunManifest{
		RunID:             "run-1",
		Provider:          "local-playwright",
		ProviderSessionID: "safe-session",
		ToolSurface:       "playwright-mcp",
		ViewportWidth:     1280,
		ViewportHeight:    720,
		Locale:            "en-US",
		Timezone:          "UTC",
		Headless:          true,
		Comparable:        true,
		Cost:              Cost{USD: nil, Currency: "USD", Source: "local-resource-unpriced"},
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "endpoint") || strings.Contains(string(b), "header") {
		t.Fatalf("manifest unexpectedly has a secret-bearing field: %s", b)
	}
}

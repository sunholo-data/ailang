package mcpcheck

import (
	"strings"
	"testing"
)

func TestCopilotRegistrationFinding(t *testing.T) {
	cases := []struct {
		name string
		md   authServerMetadata
		want string
		has  string
	}{
		{"cimd only fails", authServerMetadata{CIMD: true, AuthMeth: []string{"none"}}, Fail, "cannot sign in"},
		{"dcr passes", authServerMetadata{CIMD: true, AuthMeth: []string{"none"}, Register: "https://x/oauth/register"}, Pass, "dynamic registration"},
		{"client secrets pass", authServerMetadata{AuthMeth: []string{"none", "client_secret_basic"}}, Pass, "static registration"},
		{"post secret passes", authServerMetadata{AuthMeth: []string{"client_secret_post"}}, Pass, "Manual"},
		{"both named", authServerMetadata{AuthMeth: []string{"client_secret_basic"}, Register: "https://x/r"}, Pass, " and "},
	}
	for _, c := range cases {
		f := copilotRegistrationFinding(c.md)
		if f.Status != c.want || !strings.Contains(f.Message, c.has) {
			t.Errorf("%s: got %s %q, want %s containing %q", c.name, f.Status, f.Message, c.want, c.has)
		}
	}
}

func TestCopilotFileParams(t *testing.T) {
	plain := tool{Name: "a"}
	file := tool{Name: "parse", Meta: map[string]any{"openai/fileParams": []any{"file"}}}
	if f := checkCopilotFileParams([]tool{plain}); f.Status != Pass {
		t.Errorf("no fileParams: got %s", f.Status)
	}
	if f := checkCopilotFileParams([]tool{plain, file}); f.Status != Warn || f.Tool != "parse" {
		t.Errorf("fileParams: got %s tool=%q", f.Status, f.Tool)
	}
}

func TestCopilotWidgets(t *testing.T) {
	plain := tool{Name: "a"}
	mcpApp := tool{Name: "pick", Meta: map[string]any{"ui": map[string]any{"resourceUri": "ui://x/picker"}}}
	openaiApp := tool{Name: "card", Meta: map[string]any{"openai/outputTemplate": "ui://x/card"}}
	if f := checkCopilotWidgets([]tool{plain}); f.Status != Pass {
		t.Errorf("no widgets: got %s", f.Status)
	}
	f := checkCopilotWidgets([]tool{plain, mcpApp, openaiApp})
	if f.Status != Warn || !strings.Contains(f.Message, "2 tool(s)") || !strings.Contains(f.Message, copilotWidgetHostGenerator) {
		t.Errorf("widgets: got %s %q", f.Status, f.Message)
	}
}

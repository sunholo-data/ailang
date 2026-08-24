package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadedNoMCPRemainsInStandaloneA2ACard(t *testing.T) {
	srv := nomcpTestServer(t)
	defer srv.Close()
	card := srv.buildAgentCard(httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil))
	encoded, _ := json.Marshal(card["skills"])
	if !strings.Contains(string(encoded), "getKeyUsage") {
		t.Fatalf("@nomcp export missing from A2A card: %s", encoded)
	}
	if strings.Contains(string(encoded), "internalSecret") {
		t.Fatalf("@noexpose export leaked into A2A card: %s", encoded)
	}
}

func TestStandaloneMCPFeedbackCompatibilityBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name         string
		config       Config
		wantFeedback bool
	}{
		{"default", Config{}, true},
		{"suppressed", Config{NoFeedbackTool: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := feedbackSurfaceServer(t, tc.config)
			defer srv.Close()
			result, _ := listFeedbackSurfaceTools(t, srv)
			names := feedbackToolSet(result.Tools)
			if !names["status"] {
				t.Fatalf("positive control status missing; tools=%v", toolNames(result.Tools))
			}
			if names["submit_feedback"] != tc.wantFeedback {
				t.Fatalf("submit_feedback present=%v want=%v; tools=%v", names["submit_feedback"], tc.wantFeedback, toolNames(result.Tools))
			}
		})
	}
}

func TestLoadedExportMembershipAndNoMCPProjection(t *testing.T) {
	tests := []struct {
		name        string
		routesOnly  bool
		export      ExportInfo
		member, mcp bool
	}{
		{"noexpose", false, ExportInfo{Name: "hidden", IsNoExpose: true}, false, false},
		{"routes only non-route", true, ExportInfo{Name: "plain"}, false, false},
		{"nomcp projection", false, ExportInfo{Name: "http_a2a_openapi", IsNoMCP: true}, true, false},
		{"ordinary", false, ExportInfo{Name: "ordinary"}, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			member := loadedExportMember(tc.routesOnly, tc.export)
			mcp := member && !tc.export.IsNoMCP
			if member != tc.member || mcp != tc.mcp {
				t.Fatalf("member=%v mcp=%v", member, mcp)
			}
		})
	}
}

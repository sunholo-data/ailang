package mcpcheck

import (
	"context"
	"fmt"
	"strings"
)

// ---- copilot target: Microsoft 365 Copilot declarative agents and Copilot Studio ----
//
// Sources (Microsoft Learn, read 2026-11-10; ailang
// design_docs/planned/m-connector-channels-marketplaces.md V8, V9, V11):
//   C1 build-mcp-plugins: an MCP plugin signs users in with "OAuth (with
//      static registration)" (client ID + secret), "OAuth (with dynamic
//      registration)" (DCR), Entra SSO or None. No client ID metadata
//      documents (CIMD).
//   C2 plugin-mcp-apps: "_meta[\"openai/fileParams\"]" is not supported.
//   C3 plugin-mcp-apps: widgets render under a per-server host,
//      {hashed-mcp-domain}.widget-renderer.usercontent.microsoft.com, which
//      the MCP server and the identity provider must both allow.
// Copilot Studio (mcp-add-existing-server-to-agent) takes the same two OAuth
// shapes: "Dynamic discovery"/"Dynamic" (DCR) or "Manual" (a client secret).

// copilotWidgetHostGenerator is Microsoft's tool for the widget host of a
// given MCP URL. The checker does not compute the host itself: Microsoft
// documents it only as "the SHA-256 hash of your MCP server's domain".
const copilotWidgetHostGenerator = "https://aka.ms/mcpwidgeturlgenerator"

func (s *session) checkCopilot(ctx context.Context, tools []tool, issuer string) []Finding {
	fs := []Finding{s.checkCopilotRegistration(ctx, issuer)}
	fs = append(fs, checkCopilotFileParams(tools))
	return append(fs, checkCopilotWidgets(tools))
}

// checkCopilotRegistration: a server that signs users in must offer Copilot a
// client model it uses. CIMD alone, enough for Claude and ChatGPT, is not.
func (s *session) checkCopilotRegistration(ctx context.Context, issuer string) Finding {
	if issuer == "" {
		return Finding{"copilot-registration", Skip, "", `no authorization server named (no gated tools): Copilot's "None" authentication fits`, "C1"}
	}
	var md authServerMetadata
	if err := s.getJSON(ctx, wellKnown(issuer, "oauth-authorization-server"), &md); err != nil {
		return Finding{"copilot-registration", Skip, "", "authorization-server metadata unreadable (see authorization-server)", "C1"}
	}
	return copilotRegistrationFinding(md)
}

func copilotRegistrationFinding(md authServerMetadata) Finding {
	var ways []string
	if md.Register != "" {
		ways = append(ways, `dynamic registration (Copilot "OAuth (with dynamic registration)", Copilot Studio "Dynamic discovery")`)
	}
	if contains(md.AuthMeth, "client_secret_basic") || contains(md.AuthMeth, "client_secret_post") {
		ways = append(ways, `pre-registered clients with a secret (Copilot "OAuth (with static registration)", Copilot Studio "Manual")`)
	}
	if len(ways) == 0 {
		return Finding{"copilot-registration", Fail, "",
			"no registration_endpoint and no client_secret_basic/client_secret_post: Copilot does not use client ID metadata documents, so its users cannot sign in", "C1"}
	}
	return Finding{"copilot-registration", Pass, "", "offers " + strings.Join(ways, " and "), "C1"}
}

// checkCopilotFileParams: tools relying on ChatGPT's file hand-off need
// another way in under Copilot.
func checkCopilotFileParams(tools []tool) Finding {
	var names []string
	for _, t := range tools {
		if _, ok := t.Meta["openai/fileParams"]; ok {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		return Finding{"copilot-file-params", Pass, "", "no tool depends on openai/fileParams", "C2"}
	}
	return Finding{"copilot-file-params", Warn, strings.Join(names, ","),
		fmt.Sprintf("%d tool(s) declare openai/fileParams, which Copilot does not support: each must also take the file another way (an upload widget, a URL or content)", len(names)), "C2"}
}

// checkCopilotWidgets: a server with MCP Apps widgets must allow Copilot's
// per-server widget origin, in CORS and at its identity provider.
func checkCopilotWidgets(tools []tool) Finding {
	n := 0
	for _, t := range tools {
		ui, _ := t.Meta["ui"].(map[string]any)
		if _, ok := ui["resourceUri"]; ok {
			n++
		} else if _, ok := t.Meta["openai/outputTemplate"]; ok {
			n++
		}
	}
	if n == 0 {
		return Finding{"copilot-widgets", Pass, "", "no widgets", "C3"}
	}
	return Finding{"copilot-widgets", Warn, "",
		fmt.Sprintf("%d tool(s) render widgets: Copilot hosts them on its own per-server origin (find it with %s); allow that origin in CORS (serve-api --cors-origin) and at your identity provider", n, copilotWidgetHostGenerator), "C3"}
}

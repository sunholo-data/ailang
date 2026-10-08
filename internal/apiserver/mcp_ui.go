package apiserver

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// M-MCP-FILE-HANDOFF F1c: MCP Apps serving (ext-apps, specification
// 2026-01-26, github.com/modelcontextprotocol/ext-apps
// specification/2026-01-26/apps.mdx, fetched 2026-10-07).
//
//   - @mcp_ui_resource("ui://svc/name", "<connectDomain>", ...) on an exported
//     () -> string function: its HTML is the widget, listed by resources/list
//     and served by resources/read with mimeType "text/html;profile=mcp-app"
//     and _meta.ui.csp.connectDomains. The spec: resource content carries
//     "_meta?: { ui?: UIResourceMeta }", where csp.connectDomains are "Origins
//     for network requests (fetch/XHR/WebSocket) … Maps to CSP connect-src",
//     e.g. ["https://api.openweathermap.org", "wss://realtime.service.com"].
//     The literal "self" is the server's own public origin, taken from the
//     request (protocol.PublicBaseURL), so a service never hard-codes its host.
//   - @mcp_ui("ui://…") on a tool: _meta.ui.resourceUri, plus the flat
//     _meta["ui/resourceUri"] that the spec marks "@deprecated Use
//     ui.resourceUri instead. Will be removed before GA." and that hosts
//     released before GA still read.
//   - @mcp_app_only on a tool: _meta.ui.visibility = ["app"]. The spec:
//     "Who can access this tool. Default: ["model", "app"]. 'model': Tool
//     visible to and callable by the agent. 'app': Tool callable by the app
//     from this server only." and "Host MUST NOT include tools in the agent's
//     tool list when their visibility does not include 'model'". The server
//     still lists the tool: hiding it from the model is the host's job, and
//     the widget needs tools/list to call it.
//
// The resources capability is always advertised (ailang://meta/modules is
// always registered), so a UI resource needs no capability change.

// uiResourceMIME is the ext-apps HTML resource type.
const uiResourceMIME = "text/html;profile=mcp-app"

// uiSelf in @mcp_ui_resource's domains means the server's own origin.
const uiSelf = "self"

// legacyResourceURIKey is the pre-GA flat tool _meta key.
const legacyResourceURIKey = "ui/resourceUri"

// extractMCPUIAnnotations records @mcp_ui_resource, @mcp_ui and
// @mcp_app_only and checks what one module can check: the URI scheme, the
// connect domains, and the resource function's signature. That an @mcp_ui
// URI names a loaded resource is checked across modules (validateMCPUI).
func extractMCPUIAnnotations(modInfo *ModuleInfo, file *ast.File) error {
	for _, fn := range file.Funcs {
		res := fn.GetAnnotation("mcp_ui_resource")
		ui := fn.GetAnnotation("mcp_ui")
		appOnly := fn.GetAnnotation("mcp_app_only") != nil
		if res == nil && ui == nil && !appOnly {
			continue
		}
		var e *ExportInfo
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				e = &modInfo.Exports[i]
				break
			}
		}
		if e == nil {
			return fmt.Errorf("%s.%s: @mcp_ui_resource/@mcp_ui/@mcp_app_only apply to exported functions only", modInfo.Path, fn.Name)
		}
		if res != nil {
			if err := setUIResource(e, fn, stringLitArgs(res)); err != nil {
				return fmt.Errorf("%s.%s: %w", modInfo.Path, fn.Name, err)
			}
		}
		if ui != nil {
			args := stringLitArgs(ui)
			if len(args) != 1 || !strings.HasPrefix(args[0], "ui://") {
				return fmt.Errorf("%s.%s: @mcp_ui(%q): the URI must start with ui://", modInfo.Path, fn.Name, strings.Join(args, ", "))
			}
			e.MCPUI = args[0]
		}
		e.IsAppOnly = appOnly
		if e.UIResource != "" && (e.MCPUI != "" || appOnly) {
			return fmt.Errorf("%s.%s: an @mcp_ui_resource function is not a tool, so @mcp_ui/@mcp_app_only do not apply", modInfo.Path, fn.Name)
		}
	}
	return nil
}

func setUIResource(e *ExportInfo, fn *ast.FuncDecl, args []string) error {
	uri := args[0]
	if !strings.HasPrefix(uri, "ui://") || len(uri) == len("ui://") {
		return fmt.Errorf("@mcp_ui_resource(%q): the URI must be ui://<service>/<name>", uri)
	}
	for _, p := range fn.Params {
		if !isUnitParamType(paramTypeToString(p.Type)) {
			return fmt.Errorf("@mcp_ui_resource(%q): the function must take no arguments; it is called with none on resources/read", uri)
		}
	}
	if fn.ReturnType == nil || paramTypeToString(fn.ReturnType) != "string" {
		return fmt.Errorf("@mcp_ui_resource(%q): the function must return string (the widget's HTML)", uri)
	}
	for _, d := range args[1:] {
		if err := checkConnectDomain(d); err != nil {
			return fmt.Errorf("@mcp_ui_resource(%q): connect domain %q: %w", uri, d, err)
		}
	}
	e.UIResource = uri
	e.UIConnect = args[1:]
	// The HTML function is served as a resource, never as a tool, and (without
	// @route) never as an HTTP endpoint.
	e.IsNoMCP = true
	if e.RoutePath == "" {
		e.IsNoExpose = true
	}
	return nil
}

// checkConnectDomain accepts "self" or a bare origin: https/wss (or http/ws),
// a host, an optional port, and nothing else.
func checkConnectDomain(d string) error {
	if d == uiSelf {
		return nil
	}
	u, err := url.Parse(d)
	if err != nil {
		return err
	}
	switch {
	case u.Scheme != "https" && u.Scheme != "wss" && u.Scheme != "http" && u.Scheme != "ws":
		return fmt.Errorf("want an origin such as https://api.example.com, or %q", uiSelf)
	case u.Host == "":
		return fmt.Errorf("no host")
	case (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil:
		return fmt.Errorf("an origin has no path, query or credentials")
	}
	return nil
}

// validateMCPUI runs once every module is registered: a ui:// URI belongs to
// one resource function, and every @mcp_ui names one of them.
func (s *Server) validateMCPUI() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	owner := map[string]string{}
	type ref struct{ who, uri string }
	var refs []ref
	for _, info := range s.modules {
		for _, e := range info.Exports {
			who := info.Path + "." + e.Name
			if e.UIResource != "" {
				if prev, dup := owner[e.UIResource]; dup && prev != who {
					return fmt.Errorf("@mcp_ui_resource(%q) is declared twice: %s and %s", e.UIResource, prev, who)
				}
				owner[e.UIResource] = who
			}
			if e.MCPUI != "" {
				refs = append(refs, ref{who, e.MCPUI})
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].who < refs[j].who })
	for _, r := range refs {
		if _, ok := owner[r.uri]; !ok {
			known := make([]string, 0, len(owner))
			for u := range owner {
				known = append(known, u)
			}
			sort.Strings(known)
			return fmt.Errorf("%s: @mcp_ui(%q) names no @mcp_ui_resource (declared: %s)", r.who, r.uri, strings.Join(known, ", "))
		}
	}
	return nil
}

// uiToolMeta adds a tool's MCP Apps keys to meta (nil when it has none).
func uiToolMeta(meta map[string]any, e ExportInfo) map[string]any {
	if e.MCPUI == "" && !e.IsAppOnly {
		return meta
	}
	if meta == nil {
		meta = map[string]any{}
	}
	ui := map[string]any{}
	if e.MCPUI != "" {
		ui["resourceUri"] = e.MCPUI
		meta[legacyResourceURIKey] = e.MCPUI
	}
	if e.IsAppOnly {
		ui["visibility"] = []string{"app"}
	}
	meta["ui"] = ui
	return meta
}

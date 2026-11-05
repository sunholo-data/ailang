package apiserver

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// metaModulesURI is the module-inventory MCP resource.
const metaModulesURI = "ailang://meta/modules"

// registerResources registers the module-inventory resource on the agent
// surface only (M-MCP-OAUTH-CLIENT-MODELS D8):
//   - /mcp/connect/ (listed) never carries it: a directory listing has no use
//     for the server's module map, and on v0.57.1 it leaked @noexpose helpers,
//     the token verifier's name and absolute paths (design V18);
//   - --no-introspection turns it off on /mcp/ as it does /api/_meta/*;
//   - its content is metaModulesView, never the raw module map.
//
// The resources capability stays advertised either way: an empty
// resources/list is valid, and ui:// widgets need the capability.
func (ms *MCPServer) registerResources() {
	if ms.listed || ms.server.noIntrospection {
		return
	}
	ms.mcpServer.AddResource(&mcp.Resource{
		URI:         metaModulesURI,
		Name:        "AILANG Modules",
		Description: "List of all loaded AILANG modules and their exports",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		data, err := json.MarshalIndent(ms.server.metaModulesView(), "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: metaModulesURI, MIMEType: "application/json", Text: string(data)},
			},
		}, nil
	})
}

// metaModulesView is the agent surface's module inventory: the
// /api/_meta/modules projection (exposedView: no @noexpose, --routes-only
// applied), minus the @mcp_token_verifier, whose name tells a caller what to
// guess tokens against even when it has an @route. Modules are listed by
// their relative path, and modules and exports are sorted (export order in
// ModuleInfo is not stable), so the content carries no absolute path and is
// the same on every read.
func (s *Server) metaModulesView() ModulesListResponse {
	s.mu.RLock()
	modules := make([]*ModuleInfo, 0, len(s.modules))
	for _, m := range s.modules {
		view := s.exposedView(m)
		if view == nil {
			continue
		}
		exports := view.Exports[:0:0]
		for _, e := range view.Exports {
			if !e.IsTokenVerifier {
				exports = append(exports, e)
			}
		}
		if len(exports) == 0 {
			continue
		}
		sort.Slice(exports, func(i, j int) bool { return exports[i].Name < exports[j].Name })
		view.Exports = exports
		modules = append(modules, view)
	}
	s.mu.RUnlock()
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	return ModulesListResponse{Modules: modules, Count: len(modules)}
}

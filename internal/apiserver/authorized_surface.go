package apiserver

import "fmt"

func loadedExportMember(routesOnly bool, export ExportInfo) bool {
	// A WebSocket route is served only by its upgrade handler: it is not an
	// HTTP function, an OpenAPI operation, an MCP tool or an A2A skill.
	if export.IsNoExpose || export.IsWS {
		return false
	}
	return !routesOnly || export.RoutePath != ""
}

// isExposed is the HTTP projection of the loaded-export gateway: function
// dispatch, @route registration, OpenAPI, A2A and _meta. It is the gateway
// narrowed by connectOnlyHidden; the MCP surfaces use loadedExportMember.
func (s *Server) isExposed(export ExportInfo) bool {
	return loadedExportMember(s.routesOnly, export) && !s.connectOnlyHidden(export)
}

// connectOnlyHidden reports whether --mcp-connect-only keeps export off plain
// HTTP. Under that flag a tool is reachable only through the gated
// /mcp/connect/, so HTTP serves @nomcp exports alone (the OAuth and sign-in
// routes). Without it, an executor tool with an @route would be callable by
// anyone as POST /api/... on a public service.
func (s *Server) connectOnlyHidden(export ExportInfo) bool {
	return s.mcpConnectOnly && !export.IsNoMCP
}

// routesOnlyHint returns a banner line when @route exports and auto-exposed
// exports coexist without --routes-only: the operator may not know the plain
// exports are served too. It is a hint, not a default flip
// (M-SERVEAPI-OPERATOR-SURFACE D3).
func (s *Server) routesOnlyHint() string {
	if s.routesOnly {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	routed, auto := 0, 0
	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			switch {
			case exp.RoutePath != "":
				routed++
			case exp.Arity >= 0 && loadedExportMember(false, exp):
				auto++
			}
		}
	}
	if routed == 0 || auto == 0 {
		return ""
	}
	return fmt.Sprintf("  Note: %d @route endpoint(s) and %d auto-generated /api/ endpoint(s) are both served; pass --routes-only to serve only the @route ones", routed, auto)
}

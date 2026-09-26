package apiserver

func loadedExportMember(routesOnly bool, export ExportInfo) bool {
	// A WebSocket route is served only by its upgrade handler: it is not an
	// HTTP function, an OpenAPI operation, an MCP tool or an A2A skill.
	if export.IsNoExpose || export.IsWS {
		return false
	}
	return !routesOnly || export.RoutePath != ""
}

func (s *Server) isExposed(export ExportInfo) bool {
	return loadedExportMember(s.routesOnly, export)
}

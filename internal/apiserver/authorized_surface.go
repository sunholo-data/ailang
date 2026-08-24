package apiserver

func loadedExportMember(routesOnly bool, export ExportInfo) bool {
	if export.IsNoExpose {
		return false
	}
	return !routesOnly || export.RoutePath != ""
}

func (s *Server) isExposed(export ExportInfo) bool {
	return loadedExportMember(s.routesOnly, export)
}

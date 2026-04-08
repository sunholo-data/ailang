package apiserver

import (
	"net/http"
	"strings"
)

// ModulesListResponse is the response for GET /api/_meta/modules.
type ModulesListResponse struct {
	Modules []*ModuleInfo `json:"modules"`
	Count   int           `json:"count"`
}

// HealthResponse is the response for GET /api/_health.
type HealthResponse struct {
	Status       string `json:"status"`
	ModulesCount int    `json:"modules_count"`
	ExportsCount int    `json:"exports_count"`
}

// handleListModules returns all loaded modules and their exports.
// GET /api/_meta/modules
func (s *Server) handleListModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}

	s.mu.RLock()
	modules := make([]*ModuleInfo, 0, len(s.modules))
	for _, m := range s.modules {
		modules = append(modules, m)
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, ModulesListResponse{
		Modules: modules,
		Count:   len(modules),
	})
}

// handleModuleDetail returns detailed info about a specific module.
// GET /api/_meta/modules/{modulePath}
func (s *Server) handleModuleDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}

	// Extract module path from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/_meta/modules/")
	path = strings.TrimSuffix(path, "/")

	s.mu.RLock()
	modInfo, ok := s.findModuleByRelPath(path)
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "module not found: " + path,
		})
		return
	}

	writeJSON(w, http.StatusOK, modInfo)
}

// handleHealth returns server health status.
// GET /api/_health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	totalExports := 0
	for _, m := range s.modules {
		totalExports += len(m.Exports)
	}
	modCount := len(s.modules)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:       "ok",
		ModulesCount: modCount,
		ExportsCount: totalExports,
	})
}

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
//
// M-SERVEAPI-SURFACE-DROPS: when the server started with one or more
// modules dropped by the under-basePath filter, Status is "degraded"
// and DroppedModules lists the offenders. A healthy server omits
// DroppedModules entirely (omitempty) — readiness probes can check
// Status == "ok" or len(DroppedModules) == 0.
type HealthResponse struct {
	Status         string                `json:"status"`
	ModulesCount   int                   `json:"modules_count"`
	ExportsCount   int                   `json:"exports_count"`
	DroppedModules []DroppedModuleHealth `json:"dropped_modules,omitempty"`
	DroppedWarning string                `json:"dropped_warning,omitempty"`
}

// DroppedModuleHealth is the /health-endpoint projection of a
// DroppedModule. Drops fields the operator doesn't need at the health
// surface (FileBaseName, Reason) — those are in the server logs.
type DroppedModuleHealth struct {
	Declared    string   `json:"declared"`
	Resolved    string   `json:"resolved"`
	Annotations []string `json:"annotations,omitempty"`
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
	drops := make([]DroppedModule, len(s.droppedModules))
	copy(drops, s.droppedModules)
	s.mu.RUnlock()

	resp := HealthResponse{
		Status:       "ok",
		ModulesCount: modCount,
		ExportsCount: totalExports,
	}

	// M-SERVEAPI-SURFACE-DROPS: surface partial registration. ALL drops
	// are listed in DroppedModules so operators retain diagnostic
	// visibility into stdlib resolution edges and other noise-floor
	// rejections. But Status only goes "degraded" when an @route-bearing
	// drop is present — that's the case where a customer-facing endpoint
	// is missing and the readiness probe should route traffic away. A
	// stdlib resolution edge on an otherwise-functional service shouldn't
	// take the revision out of rotation.
	if len(drops) > 0 {
		resp.DroppedModules = make([]DroppedModuleHealth, 0, len(drops))
		hasRouteDrop := false
		for _, d := range drops {
			resp.DroppedModules = append(resp.DroppedModules, DroppedModuleHealth{
				Declared:    d.DeclaredPath,
				Resolved:    d.PhysicalPath,
				Annotations: d.Annotations,
			})
			for _, ann := range d.Annotations {
				if ann == "@route" {
					hasRouteDrop = true
					break
				}
			}
		}
		if hasRouteDrop {
			resp.Status = "degraded"
			resp.DroppedWarning = AllowDropsEnvVar + " is set — service is running with @route-bearing modules dropped"
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

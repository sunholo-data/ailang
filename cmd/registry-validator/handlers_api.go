package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

// API response types

// PackageDetailResponse is the response for GET /api/packages/:vendor/:name.
type PackageDetailResponse struct {
	Index      pkg.IndexEntry       `json:"index"`
	Versions   []VersionWithHistory `json:"versions"`
	Dependents []string             `json:"dependents"`
}

// VersionWithHistory pairs a version's metadata with its history.
type VersionWithHistory struct {
	Version  string               `json:"version"`
	Metadata *pkg.PackageMetadata `json:"metadata,omitempty"`
	History  *pkg.VersionHistory  `json:"history,omitempty"`
}

// EcosystemStats holds aggregated registry statistics.
type EcosystemStats struct {
	TotalPackages        int              `json:"total_packages"`
	TotalVersions        int              `json:"total_versions"`
	EffectDistribution   map[string]int   `json:"effect_distribution"`
	StabilityBreakdown   map[string]int   `json:"stability_breakdown"`
	DependencyDepthMax   int              `json:"dependency_depth_max"`
	AvgExportsPerPackage float64          `json:"avg_exports_per_package"`
	ValidationPassRate   float64          `json:"validation_pass_rate"`
	TopDependedOn        []DependentCount `json:"top_depended_on"`
	PurePackages         int              `json:"pure_packages"`
	AgentVsHuman         AgentHumanCount  `json:"agent_vs_human"`
}

// DependentCount tracks how many packages depend on a given package.
type DependentCount struct {
	Name           string `json:"name"`
	DependentCount int    `json:"dependent_count"`
}

// AgentHumanCount tracks how many packages were last updated by agents vs humans.
type AgentHumanCount struct {
	Agent int `json:"agent"`
	Human int `json:"human"`
}

// corsMiddleware adds CORS headers for the docs site and local dev.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range []string{
			"https://ailang.sunholo.com",
			"http://localhost:3000",
			"http://localhost:3001",
		} {
			if origin == o {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// handleAPIPackages serves GET /api/packages — the full registry index.
func (v *validator) handleAPIPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	index, err := v.cache.GetIndex(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to load index: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(index)
}

// handleAPIPackageDetail serves:
//   - GET /api/packages/:vendor/:name         — full package detail
//   - GET /api/packages/:vendor/:name/:version — single version detail
func (v *validator) handleAPIPackageDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/packages/vendor/name[/version]
	path := strings.TrimPrefix(r.URL.Path, "/api/packages/")
	path = strings.TrimSuffix(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 || len(parts) > 3 {
		jsonError(w, http.StatusBadRequest, "Invalid path: expected /api/packages/vendor/name[/version]")
		return
	}

	vendor := parts[0]
	name := parts[1]
	pkgName := fmt.Sprintf("%s/%s", vendor, name)

	detail, err := v.cache.GetPackageDetail(r.Context(), pkgName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, http.StatusNotFound, "Package %s not found", pkgName)
		} else {
			jsonError(w, http.StatusInternalServerError, "Failed to load package: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")

	// If a specific version was requested, return just that version
	if len(parts) == 3 {
		version := parts[2]
		for _, vwh := range detail.Versions {
			if vwh.Version == version {
				json.NewEncoder(w).Encode(vwh)
				return
			}
		}
		jsonError(w, http.StatusNotFound, "Version %s@%s not found", pkgName, version)
		return
	}

	json.NewEncoder(w).Encode(detail)
}

// handleAPIStats serves GET /api/stats — aggregated ecosystem statistics.
func (v *validator) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := v.cache.GetStats(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to compute stats: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=120")
	json.NewEncoder(w).Encode(stats)
}

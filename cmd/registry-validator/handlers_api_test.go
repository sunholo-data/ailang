package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/pkg"
)

// testCache creates a pre-populated cache for testing (no GCS).
func testCache() *registryCache {
	c := &registryCache{
		packages:  make(map[string]*PackageDetailResponse),
		packageAt: make(map[string]time.Time),
		ttl:       5 * time.Minute,
	}
	c.index = &pkg.RegistryIndex{
		Schema:    "ailang.registry/v1",
		UpdatedAt: "2026-03-24T12:00:00Z",
		Packages: []pkg.IndexEntry{
			{
				Name:         "sunholo/auth",
				Latest:       "0.2.0",
				Versions:     []string{"0.1.0", "0.2.0"},
				AISummary:    "API key validation, HMAC signing",
				Tags:         []string{"auth", "security"},
				Effects:      nil,
				Stability:    "experimental",
				Exports:      []string{"sunholo/auth/keys", "sunholo/auth/bearer"},
				Dependencies: nil,
				LastUpdated:  "2026-03-20T12:00:00Z",
				UpdatedBy:    "sunholo-voight-kampff",
			},
			{
				Name:         "sunholo/firestore",
				Latest:       "0.1.0",
				Versions:     []string{"0.1.0"},
				AISummary:    "Firestore REST client",
				Tags:         []string{"database"},
				Effects:      []string{"Net", "FS", "Env"},
				Stability:    "experimental",
				Exports:      []string{"sunholo/firestore/client"},
				Dependencies: []string{"sunholo/auth"},
				LastUpdated:  "2026-03-18T12:00:00Z",
				UpdatedBy:    "sunholo-voight-kampff",
			},
			{
				Name:         "sunholo/logging",
				Latest:       "0.1.1",
				Versions:     []string{"0.1.0", "0.1.1"},
				AISummary:    "Structured JSON logging",
				Tags:         []string{"logging"},
				Effects:      []string{"IO"},
				Stability:    "experimental",
				Exports:      []string{"sunholo/logging/core"},
				Dependencies: nil,
				LastUpdated:  "2026-03-19T12:00:00Z",
				UpdatedBy:    "MarkEdmondson1234",
			},
		},
	}
	c.indexAt = time.Now()

	// Pre-populate package detail for sunholo/auth
	c.packages["sunholo/auth"] = &PackageDetailResponse{
		Index: c.index.Packages[0],
		Versions: []VersionWithHistory{
			{
				Version: "0.1.0",
				Metadata: &pkg.PackageMetadata{
					Schema:      "ailang.package-metadata/v1",
					Name:        "sunholo/auth",
					Version:     "0.1.0",
					PublishedAt: "2026-03-15T12:00:00Z",
					ContentHash: "sha256:abc123",
					InterfHash:  "sha256:def456",
					TarballHash: "sha256:ghi789",
					TarballSize: 3800,
				},
			},
			{
				Version: "0.2.0",
				Metadata: &pkg.PackageMetadata{
					Schema:      "ailang.package-metadata/v1",
					Name:        "sunholo/auth",
					Version:     "0.2.0",
					PublishedAt: "2026-03-20T12:00:00Z",
					ContentHash: "sha256:jkl012",
					InterfHash:  "sha256:def456",
					TarballHash: "sha256:mno345",
					TarballSize: 4100,
					Provenance: &pkg.ProvenanceInfo{
						ChangeClass:     "A",
						AutoApproved:    true,
						PreviousVersion: "0.1.0",
					},
				},
			},
		},
		Dependents: []string{"sunholo/firestore"},
	}
	c.packageAt["sunholo/auth"] = time.Now()

	return c
}

func testValidator() *validator {
	return &validator{
		cache: testCache(),
	}
}

func TestHandleAPIPackages(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackages(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var index pkg.RegistryIndex
	if err := json.Unmarshal(w.Body.Bytes(), &index); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(index.Packages) != 3 {
		t.Errorf("expected 3 packages, got %d", len(index.Packages))
	}

	if index.Packages[0].Name != "sunholo/auth" {
		t.Errorf("expected first package sunholo/auth, got %s", index.Packages[0].Name)
	}
}

func TestHandleAPIPackages_MethodNotAllowed(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodPost, "/api/packages", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackages(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAPIPackageDetail(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/packages/sunholo/auth", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackageDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var detail PackageDetailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if detail.Index.Name != "sunholo/auth" {
		t.Errorf("expected sunholo/auth, got %s", detail.Index.Name)
	}

	if len(detail.Versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(detail.Versions))
	}

	if len(detail.Dependents) != 1 || detail.Dependents[0] != "sunholo/firestore" {
		t.Errorf("expected dependents [sunholo/firestore], got %v", detail.Dependents)
	}
}

func TestHandleAPIPackageDetail_Version(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/packages/sunholo/auth/0.2.0", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackageDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var vwh VersionWithHistory
	if err := json.Unmarshal(w.Body.Bytes(), &vwh); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if vwh.Version != "0.2.0" {
		t.Errorf("expected version 0.2.0, got %s", vwh.Version)
	}

	if vwh.Metadata == nil {
		t.Fatal("expected metadata to be present")
	}

	if vwh.Metadata.Provenance == nil {
		t.Fatal("expected provenance to be present")
	}

	if vwh.Metadata.Provenance.ChangeClass != "A" {
		t.Errorf("expected change class A, got %s", vwh.Metadata.Provenance.ChangeClass)
	}
}

func TestHandleAPIPackageDetail_NotFound(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/packages/acme/widgets", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackageDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAPIPackageDetail_VersionNotFound(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/packages/sunholo/auth/9.9.9", nil)
	w := httptest.NewRecorder()

	v.handleAPIPackageDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAPIStats(t *testing.T) {
	v := testValidator()

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()

	v.handleAPIStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var stats EcosystemStats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if stats.TotalPackages != 3 {
		t.Errorf("expected 3 packages, got %d", stats.TotalPackages)
	}

	if stats.TotalVersions != 5 {
		t.Errorf("expected 5 versions, got %d", stats.TotalVersions)
	}

	if stats.PurePackages != 1 {
		t.Errorf("expected 1 pure package, got %d", stats.PurePackages)
	}

	if stats.EffectDistribution["Net"] != 1 {
		t.Errorf("expected Net=1, got %d", stats.EffectDistribution["Net"])
	}

	if stats.EffectDistribution["IO"] != 1 {
		t.Errorf("expected IO=1, got %d", stats.EffectDistribution["IO"])
	}

	if stats.AgentVsHuman.Human != 1 {
		t.Errorf("expected 1 human update, got %d", stats.AgentVsHuman.Human)
	}

	if stats.AgentVsHuman.Agent != 2 {
		t.Errorf("expected 2 agent updates, got %d", stats.AgentVsHuman.Agent)
	}

	if len(stats.TopDependedOn) < 1 || stats.TopDependedOn[0].Name != "sunholo/auth" {
		t.Errorf("expected top depended-on to be sunholo/auth, got %v", stats.TopDependedOn)
	}

	if stats.ValidationPassRate != 1.0 {
		t.Errorf("expected validation pass rate 1.0, got %f", stats.ValidationPassRate)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Allowed origin
	req := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	req.Header.Set("Origin", "https://ailang.sunholo.com")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://ailang.sunholo.com" {
		t.Errorf("expected CORS origin header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Disallowed origin
	req2 := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	req2.Header.Set("Origin", "https://evil.com")
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header for disallowed origin, got %s", w2.Header().Get("Access-Control-Allow-Origin"))
	}

	// Preflight
	req3 := httptest.NewRequest(http.MethodOptions, "/api/packages", nil)
	req3.Header.Set("Origin", "https://ailang.sunholo.com")
	w3 := httptest.NewRecorder()
	handler(w3, req3)

	if w3.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w3.Code)
	}
}

func TestComputeEcosystemStats(t *testing.T) {
	index := &pkg.RegistryIndex{
		Packages: []pkg.IndexEntry{
			{Name: "a/pure", Effects: nil, Stability: "stable", Versions: []string{"1.0"}, Exports: []string{"a/pure/core"}, UpdatedBy: "agent-1"},
			{Name: "a/io", Effects: []string{"IO"}, Stability: "experimental", Versions: []string{"1.0", "1.1"}, Exports: []string{"a/io/core", "a/io/file"}, Dependencies: []string{"a/pure"}, UpdatedBy: "MarkEdmondson1234"},
		},
	}

	stats := computeEcosystemStats(index)

	if stats.TotalPackages != 2 {
		t.Errorf("total packages: want 2, got %d", stats.TotalPackages)
	}
	if stats.TotalVersions != 3 {
		t.Errorf("total versions: want 3, got %d", stats.TotalVersions)
	}
	if stats.PurePackages != 1 {
		t.Errorf("pure packages: want 1, got %d", stats.PurePackages)
	}
	if stats.StabilityBreakdown["stable"] != 1 {
		t.Errorf("stable: want 1, got %d", stats.StabilityBreakdown["stable"])
	}
	if stats.AvgExportsPerPackage != 1.5 {
		t.Errorf("avg exports: want 1.5, got %f", stats.AvgExportsPerPackage)
	}
	if stats.DependencyDepthMax != 2 {
		t.Errorf("dependency depth max: want 2, got %d", stats.DependencyDepthMax)
	}
}

func TestHandleAPIPackageDetail_BadPath(t *testing.T) {
	v := testValidator()

	// Too few segments
	req := httptest.NewRequest(http.MethodGet, "/api/packages/onlyone", nil)
	w := httptest.NewRecorder()
	v.handleAPIPackageDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad path, got %d", w.Code)
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := testCache()

	if c.index == nil {
		t.Fatal("expected cache to be populated")
	}

	c.Invalidate()

	if c.index != nil {
		t.Error("expected index to be nil after invalidation")
	}
	if c.stats != nil {
		t.Error("expected stats to be nil after invalidation")
	}
	if len(c.packages) != 0 {
		t.Error("expected packages to be empty after invalidation")
	}
}

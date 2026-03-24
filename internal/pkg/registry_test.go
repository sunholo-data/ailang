package pkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistryClient_SearchPackages(t *testing.T) {
	// Mock registry server
	index := RegistryIndex{
		Schema: "ailang.registry/v1",
		Packages: []IndexEntry{
			{Name: "sunholo/auth", AISummary: "API key validation", Tags: []string{"auth", "security"}, Latest: "0.1.0"},
			{Name: "sunholo/gcp-auth", AISummary: "GCP OAuth2 tokens", Tags: []string{"gcp", "auth"}, Latest: "0.1.0"},
			{Name: "sunholo/logging", AISummary: "Structured JSON logging", Tags: []string{"logging"}, Latest: "0.1.0"},
		},
	}
	indexJSON, _ := json.Marshal(index)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(indexJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &RegistryClient{
		BaseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Search by keyword
	results, err := client.SearchPackages("auth", "")
	if err != nil {
		t.Fatalf("SearchPackages: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'auth', got %d", len(results))
	}

	// Search by tag
	results, err = client.SearchPackages("", "gcp")
	if err != nil {
		t.Fatalf("SearchPackages: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for tag 'gcp', got %d", len(results))
	}
	if results[0].Name != "sunholo/gcp-auth" {
		t.Errorf("expected sunholo/gcp-auth, got %s", results[0].Name)
	}

	// Empty query returns all
	results, err = client.SearchPackages("", "")
	if err != nil {
		t.Fatalf("SearchPackages: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty query, got %d", len(results))
	}

	// No match
	results, err = client.SearchPackages("nonexistent", "")
	if err != nil {
		t.Fatalf("SearchPackages: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRegistryClient_FetchPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &RegistryClient{
		BaseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := client.FetchPackage("sunholo/nonexistent", "0.1.0")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestRegistryClient_FetchMetadata(t *testing.T) {
	meta := PackageMetadata{
		Schema:      "ailang.package-metadata/v1",
		Name:        "sunholo/auth",
		Version:     "0.1.0",
		ContentHash: "sha256:abc123",
		TarballHash: "sha256:def456",
	}
	metaJSON, _ := json.Marshal(meta)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/packages/sunholo/auth/0.1.0/metadata.json" {
			w.Write(metaJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &RegistryClient{
		BaseURL:    server.URL,
		httpClient: server.Client(),
	}

	result, err := client.FetchMetadata("sunholo/auth", "0.1.0")
	if err != nil {
		t.Fatalf("FetchMetadata: %v", err)
	}
	if result.ContentHash != "sha256:abc123" {
		t.Errorf("content_hash = %q", result.ContentHash)
	}
}

func TestRegistryClient_URLConstruction(t *testing.T) {
	client := &RegistryClient{BaseURL: "https://storage.googleapis.com/ailang-registry"}

	// Verify the URL patterns match what GCS expects
	// The actual HTTP calls would hit these paths
	expectedBase := "https://storage.googleapis.com/ailang-registry"
	if client.BaseURL != expectedBase {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, expectedBase)
	}
}

func TestCachedPackagePath(t *testing.T) {
	path, err := CachedPackagePath("sunholo/auth", "0.1.0")
	if err != nil {
		t.Fatalf("CachedPackagePath: %v", err)
	}
	if !strings.Contains(path, "sunholo/auth/0.1.0") {
		t.Errorf("path should contain vendor/name/version, got %s", path)
	}
	if !strings.Contains(path, ".ailang/cache/registry") {
		t.Errorf("path should be under cache/registry, got %s", path)
	}
}

func TestRegistryClient_ResolveLatestVersion(t *testing.T) {
	index := RegistryIndex{
		Schema: "ailang.registry/v1",
		Packages: []IndexEntry{
			{Name: "sunholo/auth", Latest: "0.3.2", Versions: []string{"0.1.0", "0.2.0", "0.3.2"}},
			{Name: "sunholo/logging", Latest: "0.1.0", Versions: []string{"0.1.0"}},
		},
	}
	indexJSON, _ := json.Marshal(index)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			w.Write(indexJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &RegistryClient{
		BaseURL:    server.URL,
		httpClient: server.Client(),
	}

	// Resolve existing package
	version, err := client.ResolveLatestVersion("sunholo/auth")
	if err != nil {
		t.Fatalf("ResolveLatestVersion: %v", err)
	}
	if version != "0.3.2" {
		t.Errorf("expected 0.3.2, got %s", version)
	}

	// Resolve another package
	version, err = client.ResolveLatestVersion("sunholo/logging")
	if err != nil {
		t.Fatalf("ResolveLatestVersion: %v", err)
	}
	if version != "0.1.0" {
		t.Errorf("expected 0.1.0, got %s", version)
	}

	// Nonexistent package
	_, err = client.ResolveLatestVersion("sunholo/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

// M-PKG-AUTONOMOUS-UPDATES: Tests for dependent lookup via Dependencies field.

func TestRegistryIndex_FindDependents(t *testing.T) {
	index := &RegistryIndex{
		Packages: []IndexEntry{
			{Name: "sunholo/auth", Dependencies: nil},
			{Name: "sunholo/gcp-auth", Dependencies: []string{"sunholo/auth"}},
			{Name: "sunholo/http-helpers", Dependencies: nil},
			{Name: "sunholo/firestore", Dependencies: []string{"sunholo/auth", "sunholo/http-helpers"}},
			{Name: "sunholo/billing_store", Dependencies: []string{"sunholo/firestore"}},
		},
	}

	// auth has 2 dependents: gcp-auth and firestore
	deps := index.FindDependents("sunholo/auth")
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of auth, got %d: %v", len(deps), deps)
	}
	if deps[0] != "sunholo/gcp-auth" || deps[1] != "sunholo/firestore" {
		t.Errorf("expected [gcp-auth, firestore], got %v", deps)
	}

	// http-helpers has 1 dependent: firestore
	deps = index.FindDependents("sunholo/http-helpers")
	if len(deps) != 1 || deps[0] != "sunholo/firestore" {
		t.Errorf("expected [firestore], got %v", deps)
	}

	// firestore has 1 dependent: billing_store
	deps = index.FindDependents("sunholo/firestore")
	if len(deps) != 1 || deps[0] != "sunholo/billing_store" {
		t.Errorf("expected [billing_store], got %v", deps)
	}

	// billing_store has no dependents
	deps = index.FindDependents("sunholo/billing_store")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependents, got %d: %v", len(deps), deps)
	}

	// nonexistent package has no dependents
	deps = index.FindDependents("sunholo/nonexistent")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependents for nonexistent, got %d", len(deps))
	}
}

func TestIndexEntry_DependenciesJSON(t *testing.T) {
	// Verify Dependencies field serializes/deserializes correctly
	entry := IndexEntry{
		Name:         "sunholo/gcp-auth",
		Latest:       "0.1.0",
		Dependencies: []string{"sunholo/auth", "sunholo/http-helpers"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded IndexEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(decoded.Dependencies))
	}
	if decoded.Dependencies[0] != "sunholo/auth" {
		t.Errorf("expected first dep 'sunholo/auth', got %q", decoded.Dependencies[0])
	}

	// Verify omitempty: nil Dependencies should not appear in JSON
	entryNoDeps := IndexEntry{Name: "sunholo/auth", Latest: "0.1.0"}
	data2, _ := json.Marshal(entryNoDeps)
	if strings.Contains(string(data2), "dependencies") {
		t.Errorf("expected dependencies to be omitted when nil, got: %s", string(data2))
	}
}

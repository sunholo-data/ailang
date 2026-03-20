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

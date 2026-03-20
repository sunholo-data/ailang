package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultRegistryURL is the default AILANG package registry.
const DefaultRegistryURL = "https://storage.googleapis.com/ailang-registry"

// RegistryClient fetches packages and metadata from the AILANG registry.
type RegistryClient struct {
	BaseURL    string
	httpClient *http.Client
	indexCache *RegistryIndex
	indexETag  string
}

// NewRegistryClient creates a client using AILANG_REGISTRY env var or default URL.
func NewRegistryClient() *RegistryClient {
	baseURL := os.Getenv("AILANG_REGISTRY")
	if baseURL == "" {
		baseURL = DefaultRegistryURL
	}
	return &RegistryClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchIndex downloads and parses the registry index.json.
// Uses ETag caching — returns cached version if unchanged.
func (rc *RegistryClient) FetchIndex() (*RegistryIndex, error) {
	url := rc.BaseURL + "/index.json"

	// Cache-bust: append timestamp to bypass GCS CDN stale cache
	url = url + "?t=" + fmt.Sprintf("%d", time.Now().Unix())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Cache-Control", "no-cache")

	// ETag-based caching (within same session)
	if rc.indexETag != "" {
		req.Header.Set("If-None-Match", rc.indexETag)
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		// If network fails but we have cache, return cached
		if rc.indexCache != nil {
			return rc.indexCache, nil
		}
		return nil, fmt.Errorf("failed to fetch registry index: %w", err)
	}
	defer resp.Body.Close()

	// 304 Not Modified — use cache
	if resp.StatusCode == http.StatusNotModified && rc.indexCache != nil {
		return rc.indexCache, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry index: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to parse registry index: %w", err)
	}

	// Cache for future requests
	rc.indexCache = &index
	if etag := resp.Header.Get("ETag"); etag != "" {
		rc.indexETag = etag
	}

	return &index, nil
}

// FetchPackage downloads a package tarball from the registry.
func (rc *RegistryClient) FetchPackage(name, version string) ([]byte, error) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package name: %s (must be vendor/name)", name)
	}

	url := fmt.Sprintf("%s/packages/%s/%s/%s/package.tar.gz", rc.BaseURL, parts[0], parts[1], version)

	resp, err := rc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s@%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s@%s not found in registry", name, version)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d for %s@%s", resp.StatusCode, name, version)
	}

	return io.ReadAll(resp.Body)
}

// FetchMetadata downloads the metadata.json for a specific package version.
func (rc *RegistryClient) FetchMetadata(name, version string) (*PackageMetadata, error) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package name: %s", name)
	}

	url := fmt.Sprintf("%s/packages/%s/%s/%s/metadata.json", rc.BaseURL, parts[0], parts[1], version)

	resp, err := rc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for %s@%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d for metadata of %s@%s", resp.StatusCode, name, version)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meta PackageMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata for %s@%s: %w", name, version, err)
	}

	return &meta, nil
}

// SearchPackages searches the index by keyword matching on name, ai_summary, and tags.
func (rc *RegistryClient) SearchPackages(query string, tagFilter string) ([]IndexEntry, error) {
	index, err := rc.FetchIndex()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	tagFilter = strings.ToLower(tagFilter)

	var results []IndexEntry
	for _, pkg := range index.Packages {
		if tagFilter != "" {
			if !containsTag(pkg.Tags, tagFilter) {
				continue
			}
		}

		if query == "" {
			results = append(results, pkg)
			continue
		}

		// Match against name, ai_summary, or tags
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.AISummary), query) ||
			containsTag(pkg.Tags, query) {
			results = append(results, pkg)
		}
	}

	return results, nil
}

// RegistryCacheDir returns the local cache directory for registry packages.
func RegistryCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := fmt.Sprintf("%s/.ailang/cache/registry", home)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// CachedPackagePath returns the cache path for a specific package version.
func CachedPackagePath(name, version string) (string, error) {
	cacheDir, err := RegistryCacheDir()
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid package name: %s", name)
	}
	return fmt.Sprintf("%s/%s/%s/%s", cacheDir, parts[0], parts[1], version), nil
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.ToLower(t) == target {
			return true
		}
	}
	return false
}

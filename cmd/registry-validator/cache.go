package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// registryCache provides an in-memory cache for registry data.
// It caches index.json and per-package detail responses with a TTL.
// Cache is invalidated on publish.
type registryCache struct {
	mu sync.RWMutex

	index     *pkg.RegistryIndex
	indexAt   time.Time
	stats     *EcosystemStats
	packages  map[string]*PackageDetailResponse // keyed by "vendor/name"
	packageAt map[string]time.Time
	ttl       time.Duration
	bucket    *storage.BucketHandle
}

func newRegistryCache(bucket *storage.BucketHandle, ttl time.Duration) *registryCache {
	return &registryCache{
		bucket:    bucket,
		ttl:       ttl,
		packages:  make(map[string]*PackageDetailResponse),
		packageAt: make(map[string]time.Time),
	}
}

// Invalidate clears all cached data. Called after a publish.
func (c *registryCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.index = nil
	c.stats = nil
	c.packages = make(map[string]*PackageDetailResponse)
	c.packageAt = make(map[string]time.Time)
	log.Printf("Cache invalidated")
}

// GetIndex returns the cached index, refreshing from GCS if stale.
func (c *registryCache) GetIndex(ctx context.Context) (*pkg.RegistryIndex, error) {
	c.mu.RLock()
	if c.index != nil && time.Since(c.indexAt) < c.ttl {
		idx := c.index
		c.mu.RUnlock()
		return idx, nil
	}
	c.mu.RUnlock()

	return c.refreshIndex(ctx)
}

func (c *registryCache) refreshIndex(ctx context.Context) (*pkg.RegistryIndex, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.index != nil && time.Since(c.indexAt) < c.ttl {
		return c.index, nil
	}

	if c.bucket == nil {
		return nil, fmt.Errorf("no GCS bucket configured")
	}

	reader, err := c.bucket.Object("index.json").NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read index.json: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read index.json body: %w", err)
	}

	var index pkg.RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}

	c.index = &index
	c.indexAt = time.Now()
	c.stats = nil // stats depend on index, invalidate
	return &index, nil
}

// GetPackageDetail returns the cached detail for a package, refreshing from GCS if stale.
func (c *registryCache) GetPackageDetail(ctx context.Context, name string) (*PackageDetailResponse, error) {
	c.mu.RLock()
	if detail, ok := c.packages[name]; ok && time.Since(c.packageAt[name]) < c.ttl {
		c.mu.RUnlock()
		return detail, nil
	}
	c.mu.RUnlock()

	return c.refreshPackageDetail(ctx, name)
}

func (c *registryCache) refreshPackageDetail(ctx context.Context, name string) (*PackageDetailResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check
	if detail, ok := c.packages[name]; ok && time.Since(c.packageAt[name]) < c.ttl {
		return detail, nil
	}

	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package name: %s", name)
	}

	// Get index to find versions and compute dependents
	// Use already-cached index if available, otherwise try GCS
	var index *pkg.RegistryIndex
	var err error
	if c.index != nil && time.Since(c.indexAt) < c.ttl {
		index = c.index
	} else if c.bucket != nil {
		index, err = c.getIndexLocked(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("package %s not found (no bucket configured)", name)
	}

	var indexEntry *pkg.IndexEntry
	for i := range index.Packages {
		if index.Packages[i].Name == name {
			indexEntry = &index.Packages[i]
			break
		}
	}
	if indexEntry == nil {
		return nil, fmt.Errorf("package %s not found in index", name)
	}

	// Fetch metadata and history for each version
	var versions []VersionWithHistory
	for _, ver := range indexEntry.Versions {
		vwh := VersionWithHistory{Version: ver}

		// Fetch metadata.json
		metaPath := fmt.Sprintf("packages/%s/%s/%s/metadata.json", parts[0], parts[1], ver)
		if data, err := c.readGCSObject(ctx, metaPath); err == nil {
			var meta pkg.PackageMetadata
			if json.Unmarshal(data, &meta) == nil {
				vwh.Metadata = &meta
			}
		}

		// Fetch history.json (may not exist for all versions)
		historyPath := fmt.Sprintf("packages/%s/%s/%s/history.json", parts[0], parts[1], ver)
		if data, err := c.readGCSObject(ctx, historyPath); err == nil {
			var hist pkg.VersionHistory
			if json.Unmarshal(data, &hist) == nil {
				vwh.History = &hist
			}
		}

		versions = append(versions, vwh)
	}

	dependents := index.FindDependents(name)

	detail := &PackageDetailResponse{
		Index:      *indexEntry,
		Versions:   versions,
		Dependents: dependents,
	}

	c.packages[name] = detail
	c.packageAt[name] = time.Now()
	return detail, nil
}

// GetStats returns cached ecosystem stats, computing from index if stale.
func (c *registryCache) GetStats(ctx context.Context) (*EcosystemStats, error) {
	c.mu.RLock()
	if c.stats != nil && c.index != nil && time.Since(c.indexAt) < c.ttl {
		s := c.stats
		c.mu.RUnlock()
		return s, nil
	}
	c.mu.RUnlock()

	return c.refreshStats(ctx)
}

func (c *registryCache) refreshStats(ctx context.Context) (*EcosystemStats, error) {
	index, err := c.GetIndex(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	stats := computeEcosystemStats(index)
	c.stats = stats
	return stats, nil
}

// getIndexLocked returns the cached index — caller must hold c.mu write lock.
func (c *registryCache) getIndexLocked(ctx context.Context) (*pkg.RegistryIndex, error) {
	if c.index != nil && time.Since(c.indexAt) < c.ttl {
		return c.index, nil
	}

	reader, err := c.bucket.Object("index.json").NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read index.json: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read index.json body: %w", err)
	}

	var index pkg.RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}

	c.index = &index
	c.indexAt = time.Now()
	return &index, nil
}

func (c *registryCache) readGCSObject(ctx context.Context, path string) ([]byte, error) {
	reader, err := c.bucket.Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// computeEcosystemStats aggregates statistics from the registry index.
func computeEcosystemStats(index *pkg.RegistryIndex) *EcosystemStats {
	stats := &EcosystemStats{
		EffectDistribution: make(map[string]int),
		StabilityBreakdown: make(map[string]int),
	}

	totalVersions := 0
	totalExports := 0
	pureCount := 0
	agentCount := 0
	humanCount := 0

	// Dependency depth calculation
	depMap := make(map[string][]string)
	for _, p := range index.Packages {
		depMap[p.Name] = p.Dependencies
	}

	for _, p := range index.Packages {
		stats.TotalPackages++
		totalVersions += len(p.Versions)
		totalExports += len(p.Exports)

		// Effects
		if len(p.Effects) == 0 {
			pureCount++
		}
		for _, effect := range p.Effects {
			stats.EffectDistribution[effect]++
		}

		// Stability
		stability := p.Stability
		if stability == "" {
			stability = "experimental"
		}
		stats.StabilityBreakdown[stability]++

		// Agent vs human
		switch p.UpdatedBy {
		case "human", "MarkEdmondson1234":
			humanCount++
		default:
			if p.UpdatedBy != "" {
				agentCount++
			}
		}
	}

	stats.TotalVersions = totalVersions
	stats.PurePackages = pureCount
	stats.AgentVsHuman = AgentHumanCount{Agent: agentCount, Human: humanCount}

	if stats.TotalPackages > 0 {
		stats.AvgExportsPerPackage = float64(totalExports) / float64(stats.TotalPackages)
	}

	// Top depended-on
	depCounts := make(map[string]int)
	for _, p := range index.Packages {
		for _, dep := range p.Dependencies {
			depCounts[dep]++
		}
	}
	for name, count := range depCounts {
		stats.TopDependedOn = append(stats.TopDependedOn, DependentCount{Name: name, DependentCount: count})
	}
	sort.Slice(stats.TopDependedOn, func(i, j int) bool {
		return stats.TopDependedOn[i].DependentCount > stats.TopDependedOn[j].DependentCount
	})

	// Dependency depth (simple BFS)
	maxDepth := 0
	for _, p := range index.Packages {
		depth := computeDepthBFS(p.Name, depMap)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	stats.DependencyDepthMax = maxDepth

	// Validation pass rate: all published packages passed validation
	stats.ValidationPassRate = 1.0

	return stats
}

// computeDepthBFS finds the longest dependency chain starting from a package.
func computeDepthBFS(name string, depMap map[string][]string) int {
	visited := make(map[string]bool)
	return depthDFS(name, depMap, visited)
}

func depthDFS(name string, depMap map[string][]string, visited map[string]bool) int {
	if visited[name] {
		return 0 // cycle
	}
	visited[name] = true
	maxChild := 0
	for _, dep := range depMap[name] {
		d := depthDFS(dep, depMap, visited)
		if d > maxChild {
			maxChild = d
		}
	}
	visited[name] = false
	return maxChild + 1
}

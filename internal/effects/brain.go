// Package effects provides BrainStore, a two-tier semantic cache (user + project).
// Part of M-BRAIN (Persistent Semantic Cache).
package effects

import (
	"encoding/base64"
	"sort"
)

// BrainStore wraps two SQLiteSharedCache instances: user-level (global) and
// project-level (repo-scoped). Both are queried by default with project results
// ranked higher. Follows the same two-tier pattern as Claude Code settings and
// ailang messages inboxes.
type BrainStore struct {
	User    *SQLiteSharedCache // ~/.ailang/state/brain.db (cross-project)
	Project *SQLiteSharedCache // .ailang/state/brain.db (project-specific)
}

// BrainScope controls which tier(s) to query or write to.
type BrainScope string

const (
	ScopeBoth    BrainScope = "both"    // Query both, merge results (default)
	ScopeUser    BrainScope = "user"    // User brain only
	ScopeProject BrainScope = "project" // Project brain only
)

// NewBrainStore creates a BrainStore from two database paths.
// Either path may be empty to skip that tier.
// Optional CacheOption values are applied to both caches (e.g., WithEmbedder).
func NewBrainStore(userDBPath, projectDBPath string, opts ...CacheOption) (*BrainStore, error) {
	var userCache, projectCache *SQLiteSharedCache
	var err error

	if userDBPath != "" {
		userCache, err = NewSQLiteSharedCache(userDBPath, opts...)
		if err != nil {
			return nil, err
		}
	}

	if projectDBPath != "" {
		projectCache, err = NewSQLiteSharedCache(projectDBPath, opts...)
		if err != nil {
			if userCache != nil {
				userCache.Close()
			}
			return nil, err
		}
	}

	return &BrainStore{User: userCache, Project: projectCache}, nil
}

// Close closes both database connections.
func (b *BrainStore) Close() error {
	var firstErr error
	if b.Project != nil {
		if err := b.Project.Close(); err != nil {
			firstErr = err
		}
	}
	if b.User != nil {
		if err := b.User.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Search queries both brains by SimHash and merges results.
// Project results get a boost factor to rank higher than user results at equal scores.
func (b *BrainStore) Search(namespace string, queryHash int64, limit int, scope BrainScope) []BrainSearchResult {
	const projectBoost = 0.05 // Small boost for project-local results

	var results []BrainSearchResult

	if (scope == ScopeBoth || scope == ScopeProject) && b.Project != nil {
		projectResults := b.Project.SearchBySimHash(namespace, queryHash, 0) // get all, merge later
		for i := range projectResults {
			projectResults[i].Tier = "project"
			projectResults[i].Score += projectBoost
			if projectResults[i].Score > 1.0 {
				projectResults[i].Score = 1.0
			}
		}
		results = append(results, projectResults...)
	}

	if (scope == ScopeBoth || scope == ScopeUser) && b.User != nil {
		userResults := b.User.SearchBySimHash(namespace, queryHash, 0)
		for i := range userResults {
			userResults[i].Tier = "user"
		}
		results = append(results, userResults...)
	}

	// Deterministic sort: score DESC, tier (project first), key ASC
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Tier != results[j].Tier {
			return results[i].Tier == "project"
		}
		return results[i].Frame.Key < results[j].Frame.Key
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SearchByEmbedding queries both brains by cosine similarity.
func (b *BrainStore) SearchByEmbedding(query []float32, namespace string, limit int, scope BrainScope) []BrainSearchResult {
	const projectBoost = 0.05

	var results []BrainSearchResult

	if (scope == ScopeBoth || scope == ScopeProject) && b.Project != nil {
		projectResults := b.Project.SearchByEmbedding(query, namespace, 0)
		for i := range projectResults {
			projectResults[i].Tier = "project"
			projectResults[i].Score += projectBoost
			if projectResults[i].Score > 1.0 {
				projectResults[i].Score = 1.0
			}
		}
		results = append(results, projectResults...)
	}

	if (scope == ScopeBoth || scope == ScopeUser) && b.User != nil {
		userResults := b.User.SearchByEmbedding(query, namespace, 0)
		for i := range userResults {
			userResults[i].Tier = "user"
		}
		results = append(results, userResults...)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Tier != results[j].Tier {
			return results[i].Tier == "project"
		}
		return results[i].Frame.Key < results[j].Frame.Key
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SearchThreeTier performs a three-tier search: cosine > SimHash > text.
// Cosine results get a +0.1 boost over SimHash-only results.
// Results are merged and deduplicated by key.
func (b *BrainStore) SearchThreeTier(query string, queryHash int64, queryEmbedding []float32, namespace string, limit int, scope BrainScope) []BrainSearchResult {
	const cosineBoost = 0.1

	seen := make(map[string]bool)
	var merged []BrainSearchResult

	// Tier 1: Cosine (if embedding available)
	if len(queryEmbedding) > 0 {
		cosResults := b.SearchByEmbedding(queryEmbedding, namespace, 0, scope)
		for _, r := range cosResults {
			r.Score += cosineBoost
			if r.Score > 1.0 {
				r.Score = 1.0
			}
			seen[r.Frame.Key] = true
			merged = append(merged, r)
		}
	}

	// Tier 2: SimHash
	simResults := b.Search(namespace, queryHash, 0, scope)
	for _, r := range simResults {
		if !seen[r.Frame.Key] {
			seen[r.Frame.Key] = true
			merged = append(merged, r)
		}
	}

	// Tier 3: Text
	textResults := b.SearchText(query, namespace, 0, scope)
	for _, r := range textResults {
		if !seen[r.Frame.Key] {
			seen[r.Frame.Key] = true
			merged = append(merged, r)
		}
	}

	// Sort merged: score DESC, tier (project first), key ASC
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Score != merged[j].Score {
			return merged[i].Score > merged[j].Score
		}
		if merged[i].Tier != merged[j].Tier {
			return merged[i].Tier == "project"
		}
		return merged[i].Frame.Key < merged[j].Frame.Key
	})

	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

// EmbeddingStats returns embedding coverage for both tiers.
func (b *BrainStore) EmbeddingStats() map[string]struct {
	Total, WithEmbedding int
	Models               map[string]int
} {
	result := make(map[string]struct {
		Total, WithEmbedding int
		Models               map[string]int
	})
	for name, cache := range map[string]*SQLiteSharedCache{"user": b.User, "project": b.Project} {
		if cache == nil {
			continue
		}
		total, withEmb, models := cache.EmbeddingStats()
		result[name] = struct {
			Total, WithEmbedding int
			Models               map[string]int
		}{total, withEmb, models}
	}
	return result
}

// ExportFrameRecord creates an export-ready map from a BrainFrame, including base64 embedding.
func ExportFrameRecord(f BrainFrame, tier string) map[string]interface{} {
	record := map[string]interface{}{
		"tier":       tier,
		"key":        f.Key,
		"namespace":  f.Namespace,
		"value":      string(f.Value),
		"simhash":    f.SimHash,
		"content":    f.Content,
		"version":    f.Version,
		"created_at": f.CreatedAt,
		"updated_at": f.UpdatedAt,
		"source":     f.Source,
	}
	if f.ExpiresAt != nil {
		record["expires_at"] = *f.ExpiresAt
	}
	if len(f.Embedding) > 0 {
		record["embedding"] = base64.StdEncoding.EncodeToString(encodeEmbedding(f.Embedding))
		record["embedding_dim"] = f.EmbeddingDim
		record["embed_model"] = f.EmbedModel
	}
	return record
}

// ImportFrameEmbedding decodes base64 embedding from an import record.
func ImportFrameEmbedding(record map[string]interface{}, f *BrainFrame) {
	if embStr, ok := record["embedding"].(string); ok && embStr != "" {
		if embBytes, err := base64.StdEncoding.DecodeString(embStr); err == nil {
			f.Embedding = decodeEmbedding(embBytes)
			f.EmbeddingDim = len(f.Embedding)
		}
	}
	if model, ok := record["embed_model"].(string); ok {
		f.EmbedModel = model
	}
}

// SearchText queries both brains by keyword and merges results.
func (b *BrainStore) SearchText(query string, namespace string, limit int, scope BrainScope) []BrainSearchResult {
	var results []BrainSearchResult

	if (scope == ScopeBoth || scope == ScopeProject) && b.Project != nil {
		projectResults := b.Project.SearchByText(query, namespace, 0)
		for i := range projectResults {
			projectResults[i].Tier = "project"
		}
		results = append(results, projectResults...)
	}

	if (scope == ScopeBoth || scope == ScopeUser) && b.User != nil {
		userResults := b.User.SearchByText(query, namespace, 0)
		for i := range userResults {
			userResults[i].Tier = "user"
		}
		results = append(results, userResults...)
	}

	// Project first, then by updated_at (already sorted per-tier)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Tier != results[j].Tier {
			return results[i].Tier == "project"
		}
		return results[i].Frame.UpdatedAt > results[j].Frame.UpdatedAt
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Put writes a frame to the appropriate tier (project by default).
func (b *BrainStore) Put(f BrainFrame, scope BrainScope) error {
	cache := b.cacheForScope(scope)
	if cache == nil {
		return nil
	}
	return cache.PutFrame(f)
}

// Promote copies a frame from the project brain to the user brain.
// Returns false if the frame doesn't exist in the project brain.
func (b *BrainStore) Promote(key string) bool {
	if b.Project == nil || b.User == nil {
		return false
	}

	value, ok := b.Project.Get(key)
	if !ok {
		return false
	}

	// Read the full frame for metadata (v2 schema with 13 columns)
	rows, err := b.Project.db.Query(
		`SELECT key, namespace, value, simhash, content, version, created_at, updated_at, expires_at, source, embedding, embedding_dim, embed_model
		 FROM brain_frames WHERE key = ?`, key,
	)
	if err != nil {
		b.User.Put(key, value)
		return true
	}
	defer rows.Close()

	if rows.Next() {
		f := scanBrainFrame(rows)
		if f != nil {
			f.Source = "promoted:" + f.Source
			_ = b.User.PutFrame(*f)
			return true
		}
	}

	b.User.Put(key, value)
	return true
}

// Stats returns combined stats from both brains.
func (b *BrainStore) Stats() map[string]BrainStats {
	result := make(map[string]BrainStats)
	if b.User != nil {
		result["user"] = b.User.Stats()
	}
	if b.Project != nil {
		result["project"] = b.Project.Stats()
	}
	return result
}

func (b *BrainStore) cacheForScope(scope BrainScope) *SQLiteSharedCache {
	switch scope {
	case ScopeUser:
		return b.User
	case ScopeProject:
		if b.Project != nil {
			return b.Project
		}
		return b.User // fallback
	default:
		if b.Project != nil {
			return b.Project
		}
		return b.User
	}
}

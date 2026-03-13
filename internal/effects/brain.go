// Package effects provides BrainStore, a two-tier semantic cache (user + project).
// Part of M-BRAIN (Persistent Semantic Cache).
package effects

import (
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

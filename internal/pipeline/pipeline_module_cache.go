package pipeline

import (
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/version"
)

// prepareCacheLookup computes the cache key for a module and reports whether it
// is cacheable. A disabled/uninitialized cache bypasses lookup; a nil
// SourceContent warns and bypasses both lookup and publication. Pointer-to-empty
// source stays cacheable (it produces a real non-empty key).
func (st *modulePipelineState) prepareCacheLookup(mod *loader.LoadedModule, modID string) (string, bool) {
	if st.moduleCache == nil || st.moduleCache.store == nil {
		return "", false
	}
	if mod.SourceContent == nil {
		st.moduleCache.warnSourceUnavailable(modID)
		return "", false
	}
	depDigests := make(map[string]string)
	for _, imp := range mod.Imports {
		if cu, ok := st.compiledUnits[imp]; ok && cu.Iface != nil {
			depDigests[imp] = cu.Iface.Digest
		}
	}
	return ModuleCacheKey(version.Commit, *mod.SourceContent, depDigests), true
}

// serveFromCache performs the verified lookup accounting and, on a verified hit
// with a non-nil cached payload, hydrates the CompileUnit, registers its
// interface with the linker, and stores it. It returns true only when the unit
// was actually served from cache (skipping compilation).
func (st *modulePipelineState) serveFromCache(mod *loader.LoadedModule, unit *CompileUnit, key string) bool {
	cached, entry, verified := st.moduleCache.load(unit.ID, key)
	if !verified {
		st.cacheMisses++
		st.debugCacheMiss(entry, unit.ID)
		return false
	}
	st.cacheHits++
	if cached == nil {
		// Verified but nil payload: fall through to a fresh compile. The lookup
		// is counted BEFORE the nil-payload check, preserving the original order.
		return false
	}
	if st.cfg.DebugCompile {
		fmt.Fprintf(os.Stderr, "[CACHE] %s: SKIP (cached %s ago)\n", unit.ID, time.Since(entry.Timestamp).Truncate(time.Second))
	}
	unit.Core = cached.Core
	unit.CoreTI = cached.CoreTI
	unit.Iface = cached.Iface
	unit.Constructors = cached.Constructors
	if unit.Iface != nil {
		st.modLinker.RegisterIface(unit.Iface)
	}
	st.compiledUnits[unit.ID] = unit
	return true
}

// debugCacheMiss emits the MISS or INVALID debug form for a failed verified
// lookup.
func (st *modulePipelineState) debugCacheMiss(entry *CacheEntry, modID string) {
	if !st.cfg.DebugCompile {
		return
	}
	if entry != nil {
		fmt.Fprintf(os.Stderr, "[CACHE] %s: INVALID, recompiling (compiled %s ago)\n", modID, time.Since(entry.Timestamp).Truncate(time.Second))
	} else {
		fmt.Fprintf(os.Stderr, "[CACHE] %s: MISS\n", modID)
	}
}

// publishFreshModule stores a freshly-compiled module's cache entry and
// artifacts. Publication requires a live cache/store and a nonempty computed
// key; artifacts are stored before the manifest entry is authorized.
func (st *modulePipelineState) publishFreshModule(modID, cacheKey string, unit *CompileUnit) {
	if st.moduleCache == nil || st.moduleCache.store == nil || cacheKey == "" {
		return
	}
	ifaceJSON, _ := unit.Iface.ToNormalizedJSON()
	entry := &CacheEntry{
		CacheKey:      cacheKey,
		IfaceDigest:   unit.Iface.Digest,
		IfaceJSON:     ifaceJSON,
		CompileTimeMs: 0, // TODO: per-module timing
		Timestamp:     time.Now(),
	}
	st.moduleCache.publish(modID, cacheKey, entry, &CachedModule{
		Core:         unit.Core,
		CoreTI:       unit.CoreTI,
		Iface:        unit.Iface,
		Constructors: unit.Constructors,
	})
}

// saveCacheAndReportStats persists the cache manifest and, under DebugCompile,
// reports the aggregate hit/miss summary counters.
func (st *modulePipelineState) saveCacheAndReportStats() {
	if st.moduleCache == nil || st.moduleCache.store == nil {
		return
	}
	st.moduleCache.save()
	if st.cfg.DebugCompile {
		totalEntries, _ := st.moduleCache.store.Stats()
		fmt.Fprintf(os.Stderr, "[CACHE] Summary: %d hits, %d misses (%d modules cached)\n",
			st.cacheHits, st.cacheMisses, totalEntries)
	}
}

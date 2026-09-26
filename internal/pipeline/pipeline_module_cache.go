package pipeline

import (
	"fmt"
	"os"
	"sync"
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
	// The compiled Core differs by pipeline mode — --release erases Debug
	// calls — so the mode is part of the compiler identity: without it a
	// --release compile was served to normal runs (all Debug output gone)
	// and vice versa (M-DEBUG-SINK-STRUCTURED-LINES, measured 2026-09-11).
	identity, err := currentCompilerIdentity(st.cfg)
	if err != nil {
		// No per-build identity: a commit-only key would serve another
		// build's verdicts, so this module is not cached at all.
		st.moduleCache.warnFingerprintUnavailable(err)
		return "", false
	}
	return ModuleCacheKey(identity, *mod.SourceContent, depDigests), true
}

// currentCompilerIdentity is compilerIdentity for THIS binary: the commit,
// plus the executable fingerprint when version.Dirty().
func currentCompilerIdentity(cfg Config) (string, error) {
	fingerprint := ""
	if version.Dirty() {
		fp, err := dirtyBuildFingerprint()
		if err != nil {
			return "", err
		}
		fingerprint = fp
	}
	return compilerIdentity(version.Commit, fingerprint, cfg), nil
}

// compilerIdentity is the cache-key component that must change whenever the
// same source would compile to different Core: the build commit, a per-build
// fingerprint when the commit does not identify the build (version.Dirty),
// and every Config flag that alters the emitted program.
//
// Without the fingerprint, every rebuild of a dirty tree shared one identity
// and was served its predecessor's verdicts: a fixed type checker reported
// "No errors" and `ailang run` executed programs it rejects
// (M-COMPILE-CACHE-DIRTY-BUILD-KEY, #1275).
func compilerIdentity(commit, fingerprint string, cfg Config) string {
	id := commit
	if fingerprint != "" {
		id += "+build:" + fingerprint
	}
	if cfg.ReleaseMode {
		id += "+release"
	}
	return id
}

// dirtyBuildFingerprint identifies this executable by size, nanosecond mtime
// and (on unix) inode: microseconds to read, and it changes on every go
// build/go install, including same-size rebuilds on coarse-mtime filesystems.
// A content hash would cost ~0.2s per invocation on a 100 MB binary.
func dirtyBuildFingerprint() (string, error) {
	buildFingerprint.once.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			buildFingerprint.err = err
			return
		}
		fi, err := os.Stat(exe)
		if err != nil {
			buildFingerprint.err = err
			return
		}
		buildFingerprint.value = fmt.Sprintf("%d-%d%s", fi.Size(), fi.ModTime().UnixNano(), fileIdentity(fi))
	})
	return buildFingerprint.value, buildFingerprint.err
}

var buildFingerprint struct {
	once  sync.Once
	value string
	err   error
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

package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// M-PERF6 M3: Content-addressed cache key computation for module compilation.

// cacheKeyVersion is bumped when the cache format changes, invalidating all entries.
const cacheKeyVersion = "v1"

// ModuleCacheKey computes a deterministic cache key for a module.
// The key incorporates:
//   - Compiler cache version (invalidates on format changes)
//   - Module source content hash
//   - Sorted dependency interface digests (invalidates when any dep changes)
//
// Returns hex-encoded SHA-256.
func ModuleCacheKey(compilerVersion string, sourceContent string, depDigests map[string]string) string {
	h := sha256.New()
	fmt.Fprintf(h, "ailang-cache:%s:%s\n", cacheKeyVersion, compilerVersion)
	fmt.Fprintf(h, "source:%x\n", sha256.Sum256([]byte(sourceContent)))

	// Sort dependency names for determinism
	deps := make([]string, 0, len(depDigests))
	for dep := range depDigests {
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	for _, dep := range deps {
		fmt.Fprintf(h, "dep:%s:%s\n", dep, depDigests[dep])
	}

	return hex.EncodeToString(h.Sum(nil))
}

package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// M-PERF6 M3: Content-addressed cache key computation for module compilation.

// cacheKeyVersion is bumped when the cache format changes, invalidating all entries.
//
// v1 -> v2 (M-LAMBDA-OPEN-RECORD-PATTERN): core.RecordPattern gained a gob-encoded
// Rest bool. Same-version dev/worktree builds could otherwise collide cache keys and
// round-trip pre-Rest blobs into the new decoder. Bumping guarantees old blobs never
// decode into the new Core struct.
//
// v2 -> v3 (M-XMOD-ALIAS-POLY): the on-disk Iface gained an AliasParams
// map[string][]string (params for parameterized aliases). The blob is JSON
// (tolerant of new/missing fields), so this is a defensive guard against a
// same-version dev/worktree build decoding a pre-AliasParams blob and treating a
// parameterized alias as nullary.
const cacheKeyVersion = "v3"

// ModuleCacheKey computes a deterministic cache key for a module.
// The key incorporates:
//   - Cache format version (cacheKeyVersion, bumped on format changes)
//   - Compiler identity (typically the build commit from internal/version.Commit) —
//     this invalidates cache on every rebuild, so bugfixes to elaboration,
//     type-checking, or op-lowering take effect without manual cache nukes.
//     For tests, any stable string works.
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

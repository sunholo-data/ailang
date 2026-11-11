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
//
// v3 -> v4 (M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS): executable blobs are
// authorized by a versioned artifacts.json stamp that binds their hashes to the
// exact module ID and caller-computed cache key.
//
// v4 -> v5: the on-disk Iface gained DerivedEq (types declared `deriving (Eq)`).
// A v4 blob decodes with none, and a module served from it would silently lose
// its Eq instances again, so v4 entries must miss.
// v5 -> v6 (M-ALIAS-BODY-CLOSURE): interface aliases, export schemes and
// constructor types are closed in their defining module. Open v5 bodies must miss.
// v6 -> v7 (M-CORETI-SHARED-ENCODING): coretypeinfo.gob is a hash-consed DAG
// (cache_coreti_dag.go), not a plain gob map. A v6 blob must never reach the new decoder.
// v7 -> v8 (M-IFC-CROSS-MODULE): iface v2 requires complete IFC summaries,
// including canonical forbidden arrays and reserved applied labels; pre-IFC
// verdicts must miss. Preserve the v7 CoreTI shared-DAG codec.
// v8 -> v9 (M-TRACE-LABEL-AWARE): core.App gained a gob-encoded TraceRedact
// label. A v8 blob decodes with none, so a cached module would trace its
// labelled values verbatim again; v8 entries must miss.
// Also v9 (M-PURE-ROW-AND-IFACE-PURITY, #1443, same unreleased window): export
// purity is derived from checked effect rows. v8 interfaces carrying always-true
// stub metadata must miss; same-version builds across the two changes are already
// separated by compilerIdentity (the build commit is in the key).
// v9 -> v10 (M-IFC-CROSS-MODULE M3, #1134): callers are checked against their
// imports' IFC contracts, and export summaries see the module's own imports
// (transitive intrinsic taint). A cache hit skips the IFC gate, so a v9 caller
// verdict — reached without imported contracts — must never be served.
// v10 -> v11 (M-V1-NOMINAL-TYPE-IDENTITY M2): a nominal type is its defining
// module plus its name. types.TCon gained Module (carried by the iface type
// JSON, the CoreTI DAG codec and gob), constructor factories are named
// make_<T>_<C>@<module>, and the interface digest renders qualified refs. A v10
// blob decodes every nominal type with no module, which no longer unifies with
// a qualified one, so v10 entries must miss.
const cacheKeyVersion = "v11"

// ModuleCacheKey computes a deterministic cache key for a module.
// The key incorporates:
//   - Cache format version (cacheKeyVersion, bumped on format changes)
//   - Compiler identity (compilerIdentity: the build commit, plus a per-build
//     fingerprint when the commit does not identify the build, see
//     version.Dirty). The commit ALONE only invalidates across commits: every
//     rebuild of a dirty tree used to share one identity and be served its
//     predecessor's verdicts (#1275). For tests, any stable string works.
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

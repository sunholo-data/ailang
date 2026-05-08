package ai

// Once-per-session prompt-cache hint warnings (M-AI-PROMPT-CACHING, v0.18.4).
//
// When AILANG callers set Request.CacheBreakpoints but the bound provider
// can't honor the hint (OpenAI auto-caches, Gemini lacks an explicit API,
// Ollama is local), we emit a single structured warning per (process,
// provider, reason) tuple — not per step — so long agent loops don't flood
// stderr.
//
// The warning is informational only. The provider behavior is unchanged
// (OpenAI still auto-caches; Gemini still has no caching). The warning
// exists so callers debugging "why isn't my cache hit firing?" see that
// hints WERE observed but DELIBERATELY ignored.

import (
	"fmt"
	"os"
	"sync"
)

// cacheWarningsEmitted tracks which (provider, reason) pairs have already
// emitted a warning in this process. Stored as a sync.Map for lock-free
// reads; the LoadOrStore pattern ensures exactly one emission per pair
// even under concurrent calls (the second goroutine's Store loses but its
// read of `loaded == true` short-circuits the print).
var cacheWarningsEmitted sync.Map

// WarnOnceCacheHintIgnored emits a single structured warning per process
// for a given (provider, reason) pair. Subsequent calls with the same pair
// return without I/O.
//
// The warning text is greppable: callers debugging cache miss patterns can
// search session logs for `cache_hint_ignored_<provider>_<reason>`.
//
// Provider names should be the canonical AILANG provider IDs ("openai",
// "gemini", "ollama", "openrouter_routed_to_openai", etc.). Reason should
// be a short stable token explaining WHY the hint is ignored (not a
// human-readable sentence).
func WarnOnceCacheHintIgnored(provider, reason string) {
	key := provider + ":" + reason
	if _, loaded := cacheWarningsEmitted.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	fmt.Fprintf(os.Stderr,
		"[ai] cache_hint_ignored_%s_%s: provider observed Request.CacheBreakpoints but cannot honor it (this is informational; the call proceeds normally and will not be re-warned this session)\n",
		provider, reason)
}

// resetCacheWarningsForTesting clears the dedup set. Test-only — production
// code never calls this.
func resetCacheWarningsForTesting() {
	cacheWarningsEmitted = sync.Map{}
}

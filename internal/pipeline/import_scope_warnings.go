package pipeline

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// scopeWarningOut receives the loader's import-scope warnings (a dependency
// importing a module of the root package, LDR006 as a warning — see
// loader.ScopeWarnings). A variable so tests can capture it.
var scopeWarningOut io.Writer = os.Stderr

// scopeWarnedOnce dedups those warnings at process level, as MOD010's are:
// every named-test body and every serve-api entry compiles its own pipeline.
var scopeWarnedOnce = struct {
	sync.Mutex
	seen map[string]bool
}{seen: make(map[string]bool)}

// emitScopeWarnings prints each warning once per process.
func emitScopeWarnings(warnings []string) {
	scopeWarnedOnce.Lock()
	defer scopeWarnedOnce.Unlock()
	for _, w := range warnings {
		if scopeWarnedOnce.seen[w] {
			continue
		}
		scopeWarnedOnce.seen[w] = true
		fmt.Fprintf(scopeWarningOut, "WARNING %s\n", w)
	}
}

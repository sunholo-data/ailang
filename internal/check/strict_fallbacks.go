package check

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// StrictFallbacks runs the STRICT_FALLBACK_001 detector over each compiled
// module's Core (with its surface AST for the return-type filter and the
// @allow_empty_ok suppression) and returns the findings as error strings. In
// package mode these promote to a hard error (exit 1) — the publish boundary.
// Mirrors InterFunctionRefs. Iterates modules in sorted order for
// deterministic output.
func StrictFallbacks(result pipeline.Result) []string {
	modIDs := make([]string, 0, len(result.Modules))
	for modID := range result.Modules {
		modIDs = append(modIDs, modID)
	}
	sort.Strings(modIDs)

	var errs []string
	for _, modID := range modIDs {
		mod := result.Modules[modID]
		if mod == nil || mod.Core == nil {
			continue
		}
		for _, w := range pipeline.DetectStrictFallbacks(mod.File, mod.Core) {
			errs = append(errs, w.String())
		}
	}
	return errs
}

// mod010 warning emission — extracted from pipeline_module.go to keep it under
// the 800-line AI-maintainability ceiling (CI file-size check). Pure process-level
// warning dedup; see M-DX-PI-HARNESS dogfooding findings.

package pipeline

import (
	"fmt"
	"os"
	"sync"
)

// mod010WarnedOnce dedups relaxed-MOD010 warnings at process level: each named-test
// body runs its own pipeline cfg, so per-cfg dedup still warned once per test
// (observed 2026-08-28: 6 identical warnings for one module).
var mod010WarnedOnce = struct {
	sync.Mutex
	seen map[string]bool
}{seen: make(map[string]bool)}

func mod010Reason(isTempPath bool) string {
	if isTempPath {
		return "temp-path"
	}
	return "relaxed"
}

// warnMOD010Relaxed emits a warning for module path mismatch in relaxed mode.
// Deduplicated at process level per (declaredPath, canonicalPath) pair.
func warnMOD010Relaxed(declaredPath, canonicalPath, reason string) {
	mod010WarnedOnce.Lock()
	defer mod010WarnedOnce.Unlock()
	k := declaredPath + "\x00" + canonicalPath
	if mod010WarnedOnce.seen[k] {
		return
	}
	mod010WarnedOnce.seen[k] = true
	switch reason {
	case "temp-path":
		fmt.Fprintf(os.Stderr, "WARNING MOD010 (%s): module '%s' does not match canonical path '%s'\n  Auto-relaxed for temporary directory. For strict checking, move file outside temp directory.\n",
			reason, declaredPath, canonicalPath)
	case "relaxed":
		fmt.Fprintf(os.Stderr, "WARNING MOD010 (%s): module '%s' does not match canonical path '%s'\n  Running under --relax-modules; mismatch ignored. For strict checking, omit --relax-modules flag.\n",
			reason, declaredPath, canonicalPath)
	default:
		fmt.Fprintf(os.Stderr, "WARNING MOD010: module '%s' does not match canonical path '%s'\n",
			declaredPath, canonicalPath)
	}
}

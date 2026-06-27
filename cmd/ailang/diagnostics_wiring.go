package main

import (
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/stdlibindex"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-AGENT-ERGONOMICS: wire stdlib import suggestions into diagnostics. Kept in the CLI (not
// internal/types or internal/importhint) so those packages stay free of a stdlib-scan dependency.
func init() {
	// "undefined variable" type errors -> suggest the missing import.
	types.ImportSuggester = stdlibindex.Modules
	// IMP010 "not exported by" link/loader errors -> auto-import / wrong-module hint
	// (M-AGENT-STUCK-FIXES M2). importhint is shared by both IMP010 producers.
	importhint.Locator = stdlibindex.Modules
}
